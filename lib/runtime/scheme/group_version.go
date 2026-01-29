package scheme

// GroupVersionKind unambiguously identifies a kind of resource
type GroupVersionKind struct {
	Group   string
	Version string
	Kind    string
}

// Empty returns true if group, version, and kind are empty
func (gvk GroupVersionKind) Empty() bool {
	return len(gvk.Group) == 0 && len(gvk.Version) == 0 && len(gvk.Kind) == 0
}

func (gvk GroupVersionKind) String() string {
	return gvk.Group + "/" + gvk.Version + ", Kind=" + gvk.Kind
}
