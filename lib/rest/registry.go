package rest

import (
	"lcp.io/lcp/lib/logger"
	"lcp.io/lcp/lib/runtime"
)

// InstallAPIGroups registers multiple API groups on a container.
// For each group, it creates (or reuses) a WebService at the group's BasePath,
// and uses an APIInstaller to register all routes.
func InstallAPIGroups(container *Container, ns runtime.NegotiatedSerializer, groups ...*APIGroupInfo) error {
	for _, group := range groups {
		if err := InstallAPIGroup(container, ns, group); err != nil {
			return err
		}
	}
	return nil
}

// InstallAPIGroup registers a single API group on the container.
func InstallAPIGroup(container *Container, ns runtime.NegotiatedSerializer, group *APIGroupInfo) error {
	basePath := group.BasePath()
	logger.Infof("installing API group %q at %s", group.GroupName, basePath)

	ws := findOrCreateWebService(container, basePath)

	installer := &APIInstaller{
		group:      group,
		ws:         ws,
		serializer: ns,
	}
	installer.Install()
	return nil
}

// findOrCreateWebService returns the existing WebService for the given root path,
// or creates and registers a new one on the container.
func findOrCreateWebService(container *Container, rootPath string) *WebService {
	for _, ws := range container.RegisteredWebServices() {
		if ws.RootPath() == rootPath {
			return ws
		}
	}
	ws := new(WebService)
	ws.Path(rootPath).
		Produces("application/json", "application/yaml").
		Consumes("application/json", "application/yaml")
	container.Add(ws)
	return ws
}
