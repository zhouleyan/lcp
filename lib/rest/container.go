package rest

import (
	"errors"
	"net/http"
	"sync"

	"lcp.io/lcp/lib/logger"
)

// Container holds a collection of WebServices to dispatch HTTP requests
// The requests are further dispatched to routes of WebServices using a RouteSelector
type Container struct {
	webServicesLock        sync.RWMutex
	webServices            []*WebService
	router                 RouteSelector // default is a CurlyRouter
	serviceErrorHandleFunc ServiceErrorHandleFunction
}

// NewContainer creates a new Container using a default router (CurlyRouter)
func NewContainer() *Container {
	return &Container{
		webServices:            []*WebService{},
		router:                 CurlyRouter{},
		serviceErrorHandleFunc: writeServiceError,
	}
}

func (c *Container) Dispatch(w http.ResponseWriter, r *http.Request) {
	if w == nil {
		panic("HTTP response writer cannot be nil")
	}
	if r == nil {
		panic("HTTP request cannot be nil")
	}
	c.dispatch(w, r)
}

// dispatch the incoming HTTP Request to the appropriate WebService
func (c *Container) dispatch(w http.ResponseWriter, r *http.Request) {

	logger.Infof("dispatching request to %s", r.URL.Path)

	// Find best match Route
	var webService *WebService
	var route *Route
	var err error
	func() {
		c.webServicesLock.RLock()
		defer c.webServicesLock.RUnlock()
		webService, route, err = c.router.SelectRoute(
			c.webServices,
			r)
	}()
	if err != nil {
		var ser ServiceError
		if errors.As(err, &ser) {
			c.serviceErrorHandleFunc(ser, w, r)
		}
		return
	}
	// ExtractParameters
	pathProcessor, ok := c.router.(PathProcessor)
	if !ok {
		pathProcessor = defaultPathProcessor{}
	}
	pathParams := pathProcessor.ExtractParameters(route, webService, r.URL.Path)
	r = WithPathParams(r, pathParams)
	route.Function(w, r)
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

func (c *Container) Remove(service *WebService) error {
	c.webServicesLock.Lock()
	defer c.webServicesLock.Unlock()
	var newServices []*WebService
	for _, each := range c.webServices {
		if each.rootPath != service.rootPath {
			newServices = append(newServices, each)
		}
	}
	c.webServices = newServices
	return nil
}

// RegisteredWebServices returns the collections of added WebServices
func (c *Container) RegisteredWebServices() []*WebService {
	c.webServicesLock.RLock()
	defer c.webServicesLock.RUnlock()
	result := make([]*WebService, len(c.webServices))
	copy(result, c.webServices)
	return result
}

// ServiceErrorHandleFunction declares functions that can be used to handle a service error situation.
// The first argument is the service error, the second is the request that resulted in the error and
// the third must be used to communicate an error response.
type ServiceErrorHandleFunction func(ServiceError, http.ResponseWriter, *http.Request)

// ServiceErrorHandler changes the default function (writeServiceError) to be called
// when a ServiceError is detected.
func (c *Container) ServiceErrorHandler(handler ServiceErrorHandleFunction) {
	c.serviceErrorHandleFunc = handler
}

// writeServiceError is the default ServiceErrorHandleFunction and is called
// when a ServiceError is returned during route selection. Default implementation
// calls resp.WriteErrorString(err.Code, err.Message)
func writeServiceError(err ServiceError, w http.ResponseWriter, r *http.Request) {
	for header, values := range err.Header {
		for _, value := range values {
			w.Header().Add(header, value)
		}
	}
	w.WriteHeader(err.Code)
	_, _ = w.Write([]byte(err.Message))
}
