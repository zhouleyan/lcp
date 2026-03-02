package rest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
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

func TestHandle_Get(t *testing.T) {
	scope := &RequestScope{Serializer: runtime.NewCodecFactory()}
	fn := func(ctx context.Context, params map[string]string, body []byte) (runtime.Object, error) {
		return &testObj{TypeMeta: runtime.TypeMeta{Kind: "Test"}, Name: "hello"}, nil
	}
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	Handle(scope, http.StatusOK, fn)(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandle_GetWithPathParams(t *testing.T) {
	scope := &RequestScope{Serializer: runtime.NewCodecFactory()}
	fn := func(ctx context.Context, params map[string]string, body []byte) (runtime.Object, error) {
		name := params["userId"]
		return &testObj{TypeMeta: runtime.TypeMeta{Kind: "User"}, Name: name}, nil
	}
	req := httptest.NewRequest("GET", "/users/alice", nil)
	req = WithPathParams(req, map[string]string{"userId": "alice"})
	w := httptest.NewRecorder()
	Handle(scope, http.StatusOK, fn)(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	respBody, _ := io.ReadAll(w.Body)
	var obj testObj
	json.Unmarshal(respBody, &obj)
	if obj.Name != "alice" {
		t.Errorf("expected name alice, got %s", obj.Name)
	}
}

func TestHandle_List(t *testing.T) {
	scope := &RequestScope{Serializer: runtime.NewCodecFactory()}
	fn := func(ctx context.Context, params map[string]string, body []byte) (runtime.Object, error) {
		return &testObj{TypeMeta: runtime.TypeMeta{Kind: "TestList"}, Name: "list"}, nil
	}
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	Handle(scope, http.StatusOK, fn)(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandle_Create(t *testing.T) {
	scope := &RequestScope{Serializer: runtime.NewCodecFactory()}
	fn := func(ctx context.Context, params map[string]string, body []byte) (runtime.Object, error) {
		return &testObj{TypeMeta: runtime.TypeMeta{Kind: "Test"}, Name: "created"}, nil
	}
	body := strings.NewReader(`{"name":"test"}`)
	req := httptest.NewRequest("POST", "/test", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	Handle(scope, http.StatusCreated, fn)(w, req)
	if w.Code != 201 {
		t.Errorf("expected 201, got %d", w.Code)
	}
}

func TestHandle_CreateValidationError(t *testing.T) {
	scope := &RequestScope{Serializer: runtime.NewCodecFactory()}
	fn := func(ctx context.Context, params map[string]string, body []byte) (runtime.Object, error) {
		return nil, apierrors.NewBadRequest("validation failed", nil)
	}
	body := strings.NewReader(`{"name":"test"}`)
	req := httptest.NewRequest("POST", "/test", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	Handle(scope, http.StatusCreated, fn)(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandle_Put(t *testing.T) {
	scope := &RequestScope{Serializer: runtime.NewCodecFactory()}
	fn := func(ctx context.Context, params map[string]string, body []byte) (runtime.Object, error) {
		return &testObj{TypeMeta: runtime.TypeMeta{Kind: "Test"}, Name: "updated"}, nil
	}
	body := strings.NewReader(`{"name":"updated"}`)
	req := httptest.NewRequest("PUT", "/test/1", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	Handle(scope, http.StatusOK, fn)(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandle_DeleteWithBody(t *testing.T) {
	scope := &RequestScope{Serializer: runtime.NewCodecFactory()}
	fn := func(ctx context.Context, params map[string]string, body []byte) (runtime.Object, error) {
		return &testObj{TypeMeta: runtime.TypeMeta{Kind: "Test"}, Name: "deleted"}, nil
	}
	req := httptest.NewRequest("DELETE", "/test/1", nil)
	w := httptest.NewRecorder()
	Handle(scope, http.StatusOK, fn)(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandle_DeleteNoContent(t *testing.T) {
	scope := &RequestScope{Serializer: runtime.NewCodecFactory()}
	fn := func(ctx context.Context, params map[string]string, body []byte) (runtime.Object, error) {
		return nil, nil
	}
	req := httptest.NewRequest("DELETE", "/test/1", nil)
	w := httptest.NewRecorder()
	Handle(scope, http.StatusOK, fn)(w, req)
	if w.Code != 204 {
		t.Errorf("expected 204, got %d", w.Code)
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
