package rest

// APIGroupInfo describes a group of related API resources.
type APIGroupInfo struct {
	GroupName string // "" for core group
	Version   string // "v1"
	Resources []ResourceInfo
}

// BasePath returns the URL prefix for this API group.
// Core group (GroupName=="") uses /api/{version}, named groups use /apis/{group}/{version}.
func (g *APIGroupInfo) BasePath() string {
	if g.GroupName == "" {
		return "/api/" + g.Version
	}
	return "/apis/" + g.GroupName + "/" + g.Version
}

// ActionInfo describes a custom action on a resource.
type ActionInfo struct {
	Name       string      // action name, e.g. "start", "restart"
	Method     string      // HTTP method, e.g. "POST"
	StatusCode int         // 0 defaults to 200
	Handler    HandlerFunc // the handler function
}

// ResourceInfo describes a single resource and its sub-resources.
type ResourceInfo struct {
	Name         string         // plural resource name, e.g. "users"
	Storage      Storage        // implements Getter/Lister/Creator etc.
	SubResources []ResourceInfo // optional nested sub-resources
	Actions      []ActionInfo   // optional custom actions
}
