package rest

// Route binds a HTTP Method, Path, Consumes combination to a RouteFunction
type Route struct {
	Method string
	Path   string // webservice root path + described path
}
