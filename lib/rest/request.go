package rest

import (
	"context"
	"net/http"
)

type key int

const (
	PathParamsKey key = iota
)

func WithPathParams(r *http.Request, pathParams map[string]string) *http.Request {
	ctx := context.WithValue(r.Context(), PathParamsKey, pathParams)
	return r.WithContext(ctx)
}
