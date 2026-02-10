package rest

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
)

type CurlyRouter struct{}

var (
	regexCache sync.Map // Cache for compiled regex patterns
)

func (c CurlyRouter) SelectRoute(
	webServices []*WebService,
	httpRequest *http.Request) (selectedService *WebService, selected *Route, err error) {

	requestTokens := tokenizePath(httpRequest.URL.Path)

	detectedService := c.detectWebService(requestTokens, webServices)
	if detectedService == nil {
		return nil, nil, NewError(http.StatusNotFound, "404: page not found")
	}
	candidateRoutes := c.selectRoutes(detectedService, requestTokens)
	if len(candidateRoutes) == 0 {
		return detectedService, nil, NewError(http.StatusNotFound, "404: page not found")
	}
	selectedRoute, err := c.detectRoute(candidateRoutes, httpRequest)
	if selectedRoute == nil {
		return detectedService, nil, err
	}
	return detectedService, selectedRoute, err
}

// detectWebService returns the best matching WebService given the list of path tokens
func (c CurlyRouter) detectWebService(requestTokens []string, webServices []*WebService) *WebService {
	var selected *WebService
	score := -1
	for _, service := range webServices {
		matches, serviceScore := c.computeWebServiceScore(requestTokens, service.pathExpr.tokens)
		if matches && (serviceScore > score) {
			selected = service
			score = serviceScore
		}
	}
	return selected
}

// computeWebServiceScore returns whether tokens match and
// the weighted score of the longest matching consecutive tokens from the beginning
func (c CurlyRouter) computeWebServiceScore(requestTokens []string, routeTokens []string) (bool, int) {
	if len(routeTokens) > len(requestTokens) {
		return false, 0
	}
	score := 0
	for i := 0; i < len(routeTokens); i++ {
		eachRequestToken := requestTokens[i]
		eachRouteToken := routeTokens[i]
		if len(eachRequestToken) == 0 && len(eachRouteToken) == 0 {
			score++
			continue
		}
		if len(eachRouteToken) > 0 && strings.HasPrefix(eachRouteToken, "{") {
			// no empty match
			if len(eachRequestToken) == 0 {
				return false, score
			}
			score++
			if colon := strings.Index(eachRouteToken, ":"); colon != -1 {
				// {zipcode:[\d][\d][\d][\d][A-Z][A-Z]}
				// match by regex
				matchesToken, _ := c.regularMatchesPathToken(eachRouteToken, colon, eachRequestToken)
				if matchesToken {
					score++
				}
			}
		} else {
			// not a parameter
			if eachRequestToken != eachRouteToken {
				return false, score
			}
			score += (len(routeTokens) - i) * 10
		}
	}
	return true, score
}

func (c CurlyRouter) selectRoutes(ws *WebService, requestTokens []string) sortableCurlyRoutes {
	candidates := make(sortableCurlyRoutes, 0, 8)
	for _, eachRoute := range ws.routes {
		//match
		matches, paramCount, staticCount := c.matchesRouteByPathTokens(eachRoute.pathParts, requestTokens, eachRoute.hasCustomVerb)
		eachRoute.paramCount = paramCount
		eachRoute.staticCount = staticCount
		if matches {
			candidates = append(candidates, &eachRoute)
		}
	}
	sort.Sort(&candidates)
	return candidates
}

// matchesRouteByPathTokens computes whether is matches, how many parameters do match and what the number of static path elements are
func (c CurlyRouter) matchesRouteByPathTokens(routeTokens, requestTokens []string, routeHasCustomVerb bool) (matches bool, paramCount, staticCount int) {
	if len(routeTokens) < len(requestTokens) {
		// proceed in matching only if last routeToken is wildcard
		count := len(routeTokens)
		if count == 0 || !strings.HasSuffix(routeTokens[count-1], "*") {
			return false, 0, 0
		}
		// proceed
	}
	for i, routeToken := range routeTokens {
		if i == len(requestTokens) {
			// reached the end of request path
			return false, 0, 0
		}
		requestToken := requestTokens[i]
		if routeHasCustomVerb && hasCustomVerb(routeToken) {
			if !isMatchCustomVerb(routeToken, requestToken) {
				return false, 0, 0
			}
			staticCount++
			requestToken = removeCustomVerb(requestToken)
			routeToken = removeCustomVerb(routeToken)
		}

		if strings.HasPrefix(routeToken, "{") {
			paramCount++
			if colon := strings.Index(requestToken, ":"); colon != -1 {
				// match by regex
				matchesToken, matchesRemainder := c.regularMatchesPathToken(requestToken, colon, requestToken)
				if !matchesToken {
					return false, 0, 0
				}
				if matchesRemainder {
					break
				}
			}
		} else {
			// no "{" prefix
			if requestToken != routeToken {
				return false, 0, 0
			}
			staticCount++
		}
	}
	return true, paramCount, staticCount
}

// regularMatchesPathToken tests whether the regular expression part of routeToken matches the requestToken of all remaining tokens
// format routeToken is {someVar:someExpression}, e.g. {zipcode:[\d][\d][\d][\d][A-Z][A-Z]}
func (c CurlyRouter) regularMatchesPathToken(routeToken string, colon int, requestToken string) (matchesToken bool, matchesRemainder bool) {
	regPart := routeToken[colon+1 : len(routeToken)-1]
	if regPart == "*" {
		return true, true
	}
	// Check cache first (if enabled)
	if regex, found := getCachedRegexp(&regexCache, regPart); found {
		matched := regex.MatchString(requestToken)
		return matched, false
	}

	// Compile the regex
	regex, err := regexp.Compile(requestToken)
	if err != nil {
		return false, false
	}

	// Cache the regex
	regexCache.Store(regPart, regex)
	matched := regex.MatchString(requestToken)

	return matched, false
}

func (c CurlyRouter) detectRoute(candidateRoutes sortableCurlyRoutes, httpRequest *http.Request) (*Route, error) {
	candidates := make([]*Route, 0, 8)
	for _, each := range candidateRoutes {
		candidates = append(candidates, each)
	}
	if len(candidates) == 0 {
		return nil, NewError(http.StatusNotFound, "404: Route Not Found")
	}

	// HTTP method
	previous := candidates
	candidates = candidates[:0]
	for _, each := range previous {
		if httpRequest.Method == each.Method {
			candidates = append(candidates, each)
		}
	}
	if len(candidates) == 0 {
		var allowedMethods []string
	allowedLoop:
		for _, candidate := range previous {
			for _, method := range allowedMethods {
				if method == candidate.Method {
					continue allowedLoop
				}
			}
			allowedMethods = append(allowedMethods, candidate.Method)
		}
		header := http.Header{"Allow": []string{strings.Join(allowedMethods, ", ")}}
		return nil, NewErrorWithHeader(http.StatusMethodNotAllowed, "405: Method Not Allowed", header)
	}

	// Content-Type
	contentType := httpRequest.Header.Get(HEADER_ContentType)
	previous = candidates
	candidates = candidates[:0]
	for _, each := range previous {
		if each.matchesContentType(contentType) {
			candidates = append(candidates, each)
		}
	}
	if len(candidates) == 0 {
		return nil, NewError(http.StatusUnsupportedMediaType, "415: Unsupported Media Type")
	}

	// Accept
	previous = candidates
	candidates = candidates[:0]
	accept := httpRequest.Header.Get(HEADER_Accept)
	if len(accept) == 0 {
		accept = "*/*"
	}
	for _, each := range previous {
		if each.matchesAccept(accept) {
			candidates = append(candidates, each)
		}
	}
	if len(candidates) == 0 {
		var available []string
		for _, candidate := range previous {
			available = append(available, candidate.Produces...)
		}
		return nil, NewError(
			http.StatusNotAcceptable,
			fmt.Sprintf("406: Not Acceptable\n\nAvailable representations: %s", strings.Join(available, ", ")))
	}
	return candidates[0], nil
}
