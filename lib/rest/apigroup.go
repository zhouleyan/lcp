package rest

// APIGroupInfo describes a group of related API resources.
type APIGroupInfo struct {
	GroupName string // "" for core group
	Version   string // "v1"
	Resources []ResourceInfo
}

// BasePath returns the URL prefix for this API group.
// Core group (GroupName=="") uses /api/{version}, named groups use /api/{group}/{version}.
func (g *APIGroupInfo) BasePath() string {
	if g.GroupName == "" {
		return "/api/" + g.Version
	}
	return "/api/" + g.GroupName + "/" + g.Version
}

// APIVersion returns the wire-format apiVersion string.
// Core group returns just the version (e.g. "v1"), named groups return "group/version" (e.g. "iam/v1").
func (g *APIGroupInfo) APIVersion() string {
	if g.GroupName == "" {
		return g.Version
	}
	return g.GroupName + "/" + g.Version
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
	IDParam      string         // path parameter name for this resource's primary key, e.g. "namespaceId"; if empty, derived from Name via defaultIDParam()
	Storage      Storage        // implements Getter/Lister/Creator etc.
	SubResources []ResourceInfo // optional nested sub-resources
	Actions      []ActionInfo   // optional custom actions
}
