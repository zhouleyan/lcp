package rest

// APIGroupInfo describes a group of related API resources.
type APIGroupInfo struct {
	GroupName string // "" for core group
	Version   string // "v1"
	Resources []ResourceInfo
}

// ResourceInfo describes a single resource and its sub-resources.
type ResourceInfo struct {
	Name         string            // plural resource name, e.g. "users"
	IDParam      string            // path parameter name, e.g. "userId"; auto-derived if empty
	Storage      interface{}       // implements Getter/Lister/Creator etc.
	SubResources []SubResourceInfo // optional sub-resources
}

// SubResourceInfo describes a sub-resource under a parent resource.
type SubResourceInfo struct {
	Name    string      // plural sub-resource name, e.g. "members"
	IDParam string      // path parameter name; auto-derived if empty
	Storage interface{} // implements Getter/Lister/Creator etc.
}
