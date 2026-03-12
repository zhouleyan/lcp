package network

// Stores holds all Network Store instances.
type Stores struct {
	Network      NetworkStore
	Subnet       SubnetStore
	IPAllocation IPAllocationStore
}

// RESTStorageProvider centralizes all Network Store instances.
type RESTStorageProvider struct {
	Stores
}

// NewRESTStorageProvider creates a RESTStorageProvider from a Stores struct.
func NewRESTStorageProvider(s Stores) *RESTStorageProvider {
	return &RESTStorageProvider{Stores: s}
}
