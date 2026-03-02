package handler

import (
	"context"
	"encoding/json"

	"lcp.io/lcp/lib/api/types"
	"lcp.io/lcp/lib/runtime"
	"lcp.io/lcp/lib/service"
)

type userHandler struct {
	svc *service.Service
}

func newUserHandler(svc *service.Service) *userHandler {
	return &userHandler{svc: svc}
}

func (h *userHandler) Create(ctx context.Context, params map[string]string, body []byte) (runtime.Object, error) {
	var user types.User
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, err
	}
	return h.svc.Users().CreateUser(ctx, &user)
}

func (h *userHandler) Get(ctx context.Context, params map[string]string, body []byte) (runtime.Object, error) {
	return h.svc.Users().GetUser(ctx, params["userId"])
}
