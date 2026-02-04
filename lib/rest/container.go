package rest

import (
	"net/http"
	"sync"

	"lcp.io/lcp/lib/logger"
)

// Container holds a collection of WebServices and an http.ServeMux to dispatch HTTP requests
// The requests are further dispatched to routes of WebServices using a RouteSelector
type Container struct {
	webServicesLock sync.RWMutex
	webServices     []*WebService
}

// NewContainer creates a new Container using a new ServeMux and default router (CurlyRouter)
func NewContainer() *Container {
	return &Container{
		webServices: []*WebService{},
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
