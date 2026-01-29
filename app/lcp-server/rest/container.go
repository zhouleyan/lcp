package rest

import (
	"net/http"
	"sync"
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

// Add a WebService to the Container. It will detect duplicate root paths and exit in that case
func (c *Container) Add(service *WebService) *Container {
	c.webServicesLock.Lock()
	defer c.webServicesLock.Unlock()

	// if rootPath was not set then lazy initialize it
	if len(service.rootPath) == 0 {
		//service.
	}

	return c
}

// RegisteredWebServices returns the collections of added WebServices
func (c *Container) RegisteredWebServices(ws *WebService) []*WebService {
	c.webServicesLock.RLock()
	defer c.webServicesLock.RUnlock()
	result := make([]*WebService, len(c.webServices))
	copy(result, c.webServices)
	return result
}
