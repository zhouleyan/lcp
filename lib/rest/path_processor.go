package rest

import (
	"bytes"
	"strings"
)

// PathProcessor is extra behaviour that a Router can provide to extract path parameters from the path
// If a Router does not implement this interface then the default behaviour will be used
type PathProcessor interface {
	// ExtractParameters gets the path parameters defined in the route and webService from the urlPath
	ExtractParameters(route *Route, webService *WebService, urlPath string) map[string]string
}

type defaultPathProcessor struct{}

// ExtractParameters extract the parameters from the request url path
func (d defaultPathProcessor) ExtractParameters(r *Route, _ *WebService, urlPath string) map[string]string {
	urlParts := tokenizePath(urlPath)
	pathParameters := map[string]string{}
	for i, key := range r.pathParts {
		var value string
		if i >= len(urlParts) {
			value = ""
		} else {
			value = urlParts[i]
		}
		if r.hasCustomVerb && hasCustomVerb(key) {
			key = removeCustomVerb(key)
			value = removeCustomVerb(value)
		}

		if strings.Contains(key, "{") { // path-parameter
			if colon := strings.Index(key, ":"); colon != -1 {
				// extract by regex
				regPart := key[colon+1 : len(key)-1]
				keyPart := key[1:colon]
				if regPart == "*" {
					pathParameters[keyPart] = unTokenizePath(i, urlParts)
					break
				} else {
					pathParameters[keyPart] = value
				}
			} else {
				// without enclosing {}
				startIndex := strings.Index(key, "{")
				endKeyIndex := strings.Index(key, "}")

				suffixLength := len(key) - endKeyIndex - 1
				endValueIndex := len(value) - suffixLength

				pathParameters[key[startIndex+1:endKeyIndex]] = value[startIndex:endValueIndex]
			}
		}
	}
	return pathParameters
}

// unTokenizePath back into a URL path using the slash separator
func unTokenizePath(offset int, parts []string) string {
	var buffer bytes.Buffer
	for p := offset; p < len(parts); p++ {
		buffer.WriteString(parts[p])
		// do not end
		if p < len(parts)-1 {
			buffer.WriteString("/")
		}
	}
	return buffer.String()
}
