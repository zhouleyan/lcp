package rest

import (
	"fmt"
	"regexp"
	"sync"
)

var (
	customVerbReg          = regexp.MustCompile(":([A-Za-z]+)$")
	customVerbCache        sync.Map // Cache
	customVerbCacheEnabled = true   // Enable/Disable custom verb regex caching
)

func hasCustomVerb(routeToken string) bool {
	return customVerbReg.MatchString(routeToken)
}

// getCachedRegexp retrieves a compiled regex from the cache if found and valid.
// Returns the regex and true if found and valid, nil and false otherwise
func getCachedRegexp(cache *sync.Map, pattern string) (*regexp.Regexp, bool) {
	if cached, found := cache.Load(pattern); found {
		if regex, ok := cached.(*regexp.Regexp); ok {
			return regex, true
		}
	}
	return nil, false
}

func isMatchCustomVerb(routeToken, pathToken string) bool {
	rs := customVerbReg.FindStringSubmatch(routeToken)
	if len(rs) < 2 {
		return false
	}

	customVerb := rs[1]
	regexPattern := fmt.Sprintf(":%s$", customVerb)

	if specificVerbReg, found := getCachedRegexp(&customVerbCache, regexPattern); found {
		return specificVerbReg.MatchString(pathToken)
	}

	// Compile the regex
	specificVerbReg := regexp.MustCompile(regexPattern)

	// Cache the regex
	customVerbCache.Store(regexPattern, specificVerbReg)

	return specificVerbReg.MatchString(pathToken)
}

func removeCustomVerb(str string) string {
	return customVerbReg.ReplaceAllString(str, "")
}
