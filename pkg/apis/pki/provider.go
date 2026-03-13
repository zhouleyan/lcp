package pki

// Stores holds all PKI Store instances.
type Stores struct {
	Certificate CertificateStore
}

// RESTStorageProvider centralizes all PKI Store instances.
type RESTStorageProvider struct {
	Stores
}

// NewRESTStorageProvider creates a RESTStorageProvider from a Stores struct.
func NewRESTStorageProvider(s Stores) *RESTStorageProvider {
	return &RESTStorageProvider{Stores: s}
}
