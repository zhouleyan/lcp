package handler

import (
	"context"
	"encoding/json"

	"lcp.io/lcp/lib/api/types"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/lib/runtime"
	"lcp.io/lcp/lib/service"
)

type namespaceHandler struct {
	svc *service.Service
}

func newNamespaceHandler(svc *service.Service) *namespaceHandler {
	return &namespaceHandler{svc: svc}
}

func (h *namespaceHandler) Create(ctx context.Context, body []byte) (runtime.Object, error) {
	var ns types.Namespace
	if err := json.Unmarshal(body, &ns); err != nil {
		return nil, err
	}
	return h.svc.Namespaces().CreateNamespace(ctx, &ns)
}

func (h *namespaceHandler) Get(ctx context.Context, name string) (runtime.Object, error) {
	return h.svc.Namespaces().GetNamespace(ctx, name)
}

func (h *namespaceHandler) AddMemberCreator(namespaceID string) rest.CreatorFunc {
	return func(ctx context.Context, body []byte) (runtime.Object, error) {
		var member types.NamespaceMember
		if err := json.Unmarshal(body, &member); err != nil {
			return nil, err
		}
		return h.svc.Namespaces().AddMember(ctx, namespaceID, &member)
	}
}
