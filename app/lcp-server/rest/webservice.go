package rest

// WebService holds a collection of Route values that bind a HTTP Method + URL Path to a function
type WebService struct {
	rootPath   string
	routes     []Route
	apiVersion string
}
