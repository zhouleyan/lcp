package rest

import (
	"regexp"
	"sync"
)

// getCachedRegexp retrieves a compiled regex from the cache if found and valid.
// Returns the regex and true if found and valid, nil and false otherwise.
func getCachedRegexp(cache *sync.Map, pattern string) (*regexp.Regexp, bool) {
	if cached, found := cache.Load(pattern); found {
		if regex, ok := cached.(*regexp.Regexp); ok {
			return regex, true
		}
	}
	return nil, false
}
