package rest

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/lib/runtime"
)

type testObj struct {
	runtime.TypeMeta `json:",inline"`
	Name             string `json:"name"`
}

func (t *testObj) GetTypeMeta() *runtime.TypeMeta { return &t.TypeMeta }

func TestGetResource(t *testing.T) {
	scope := &RequestScope{Serializer: runtime.NewCodecFactory()}
	getter := func(ctx context.Context, name string) (runtime.Object, error) {
		return &testObj{TypeMeta: runtime.TypeMeta{Kind: "Test"}, Name: "hello"}, nil
	}
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	GetResource(scope, getter)(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestCreateResource(t *testing.T) {
	scope := &RequestScope{Serializer: runtime.NewCodecFactory()}
	creator := func(ctx context.Context, body []byte) (runtime.Object, error) {
		return &testObj{TypeMeta: runtime.TypeMeta{Kind: "Test"}, Name: "created"}, nil
	}
	body := strings.NewReader(`{"name":"test"}`)
	req := httptest.NewRequest("POST", "/test", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	CreateResource(scope, creator)(w, req)
	if w.Code != 201 {
		t.Errorf("expected 201, got %d", w.Code)
	}
}

func TestCreateResource_ValidationError(t *testing.T) {
	scope := &RequestScope{Serializer: runtime.NewCodecFactory()}
	creator := func(ctx context.Context, body []byte) (runtime.Object, error) {
		return nil, apierrors.NewBadRequest("validation failed", nil)
	}
	body := strings.NewReader(`{"name":"test"}`)
	req := httptest.NewRequest("POST", "/test", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	CreateResource(scope, creator)(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestErrorNegotiated_StatusError(t *testing.T) {
	ns := runtime.NewCodecFactory()
	err := apierrors.NewNotFound("User", "alice")
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	ErrorNegotiated(w, req, ns, err)
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
	respBody, _ := io.ReadAll(w.Body)
	var resp map[string]any
	json.Unmarshal(respBody, &resp)
	if resp["reason"] != "NotFound" {
		t.Errorf("expected NotFound reason, got %v", resp["reason"])
	}
}
