package o11y

// RESTStorageProvider centralizes all o11y store instances.
type RESTStorageProvider struct {
	Stores
}

// NewRESTStorageProvider creates a RESTStorageProvider from a Stores struct.
func NewRESTStorageProvider(s Stores) *RESTStorageProvider {
	return &RESTStorageProvider{Stores: s}
}
