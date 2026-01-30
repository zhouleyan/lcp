package rest

import "net/http"

// RouteSelector finds the best matching Route given the input HTTP Request
// RouteSelectors can optionally also implement the PathProcessor interface to also calculate the
// path parameters after the route has been selected
type RouteSelector interface {
	// SelectRoute finds a Route given the input HTTP Request and a list if WebServices
	// It returns a selected Route and its containing WebService or an error indicating a problem
	SelectRoute(webServices []*WebService, httpRequest *http.Request) (selectedService *WebService, selected *Route, err error)
}
