package iam

// RESTStorageProvider centralizes all IAM Store instances.
// Downstream code (e.g. v1/install.go) only sees Store interfaces.
type RESTStorageProvider struct {
	userStore UserStore
	nsStore   NamespaceStore
	unStore   UserNamespaceStore
}

// NewRESTStorageProvider creates a RESTStorageProvider from pre-built Store instances.
func NewRESTStorageProvider(userStore UserStore, nsStore NamespaceStore, unStore UserNamespaceStore) *RESTStorageProvider {
	return &RESTStorageProvider{
		userStore: userStore,
		nsStore:   nsStore,
		unStore:   unStore,
	}
}

func (p *RESTStorageProvider) UserStore() UserStore                   { return p.userStore }
func (p *RESTStorageProvider) NamespaceStore() NamespaceStore         { return p.nsStore }
func (p *RESTStorageProvider) UserNamespaceStore() UserNamespaceStore { return p.unStore }
