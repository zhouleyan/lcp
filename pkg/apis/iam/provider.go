package iam

// RESTStorageProvider centralizes all IAM Store instances.
// Downstream code (e.g. v1/install.go) only sees Store interfaces.
type RESTStorageProvider struct {
	userStore         UserStore
	wsStore           WorkspaceStore
	nsStore           NamespaceStore
	uwStore           UserWorkspaceStore
	unStore           UserNamespaceStore
	refreshTokenStore RefreshTokenStore
}

// NewRESTStorageProvider creates a RESTStorageProvider from pre-built Store instances.
func NewRESTStorageProvider(
	userStore UserStore,
	wsStore WorkspaceStore,
	nsStore NamespaceStore,
	uwStore UserWorkspaceStore,
	unStore UserNamespaceStore,
	refreshTokenStore RefreshTokenStore,
) *RESTStorageProvider {
	return &RESTStorageProvider{
		userStore:         userStore,
		wsStore:           wsStore,
		nsStore:           nsStore,
		uwStore:           uwStore,
		unStore:           unStore,
		refreshTokenStore: refreshTokenStore,
	}
}

func (p *RESTStorageProvider) UserStore() UserStore                   { return p.userStore }
func (p *RESTStorageProvider) WorkspaceStore() WorkspaceStore         { return p.wsStore }
func (p *RESTStorageProvider) NamespaceStore() NamespaceStore         { return p.nsStore }
func (p *RESTStorageProvider) UserWorkspaceStore() UserWorkspaceStore { return p.uwStore }
func (p *RESTStorageProvider) UserNamespaceStore() UserNamespaceStore { return p.unStore }
func (p *RESTStorageProvider) RefreshTokenStore() RefreshTokenStore   { return p.refreshTokenStore }
