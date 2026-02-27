package service

import "lcp.io/lcp/lib/store"

// Service is the top-level business service aggregator.
type Service struct {
	store store.Store
}

func New(s store.Store) *Service {
	return &Service{store: s}
}

func (s *Service) Users() *UserService {
	return &UserService{s: s}
}

func (s *Service) Namespaces() *NamespaceService {
	return &NamespaceService{s: s}
}
