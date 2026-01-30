package rest

import (
	"net/http"
	"strings"
	"sync"

	"lcp.io/lcp/lib/logger"
)

// Container holds a collection of WebServices and an http.ServeMux to dispatch HTTP requests
// The requests are further dispatched to routes of WebServices using a RouteSelector
type Container struct {
	webServicesLock sync.RWMutex
	webServices     []*WebService
	ServeMux        *http.ServeMux
}

// NewContainer creates a new Container using a new ServeMux and default router (CurlyRouter)
func NewContainer() *Container {
	return &Container{
		webServices: []*WebService{},
		ServeMux:    http.NewServeMux(),
	}
}

// dispatch the incoming HTTP Request to the appropriate WebService
func (c *Container) dispatch(w http.ResponseWriter, r *http.Request) {
	//writer := w // TODO: wrap the w

	// Find best match Route
	//var webService *WebService
	//var err error
	func() {
		c.webServicesLock.RLock()
		defer c.webServicesLock.RUnlock()

	}()
}

// Add a WebService to the Container. It will detect duplicate root paths and exit in that case
func (c *Container) Add(service *WebService) *Container {
	c.webServicesLock.Lock()
	defer c.webServicesLock.Unlock()

	// if rootPath was not set then lazy initialize it
	if len(service.rootPath) == 0 {
		service.Path("/")
	}

	// get rid of duplicate root paths
	for _, each := range c.webServices {
		if each.RootPath() == service.RootPath() {
			logger.Fatalf("duplicate root path: " + service.RootPath())
		}
	}

	c.webServices = append(c.webServices, service)
	return c
}

// addHandler may set a new HandlerFunc for the serveMux
// this function must run inside the critical region protected by the webServicesLock
// returns true if the function was registered on root ("/")
func (c *Container) addHandler(service *WebService, serveMux *http.ServeMux) bool {
	pattern := fixedPrefixPath(service.RootPath())
	// check if root path registration is needed
	if "/" == pattern || "" == pattern {
		serveMux.HandleFunc("/", c.dispatch)
		return true
	}
	return false
}

// fixedPrefixPath returns the fixed part of the pathSpec ; it may include template vars {}
func fixedPrefixPath(pathSpec string) string {
	varBegin := strings.Index(pathSpec, "{")
	if -1 == varBegin {
		return pathSpec
	}
	return pathSpec[:varBegin]
}

// RegisteredWebServices returns the collections of added WebServices
func (c *Container) RegisteredWebServices(ws *WebService) []*WebService {
	c.webServicesLock.RLock()
	defer c.webServicesLock.RUnlock()
	result := make([]*WebService, len(c.webServices))
	copy(result, c.webServices)
	return result
}
