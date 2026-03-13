package infra

// Stores holds all Infra Store instances.
type Stores struct {
	Host        HostStore
	Environment EnvironmentStore
	Region         RegionStore
	Site           SiteStore
	Location       LocationStore
	Rack           RackStore
	IPBinder       IPBinder
}

// RESTStorageProvider centralizes all Infra Store instances.
// Downstream code (e.g. v1/install.go) accesses stores via exported fields.
type RESTStorageProvider struct {
	Stores
}

// NewRESTStorageProvider creates a RESTStorageProvider from a Stores struct.
func NewRESTStorageProvider(s Stores) *RESTStorageProvider {
	return &RESTStorageProvider{Stores: s}
}
