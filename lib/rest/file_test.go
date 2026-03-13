package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"lcp.io/lcp/lib/runtime"
)

func TestHandle_FileResponse(t *testing.T) {
	fn := func(ctx context.Context, params map[string]string, body []byte) (runtime.Object, error) {
		return &FileResponse{
			FileName:    "test.pem",
			ContentType: "application/x-pem-file",
			Data:        []byte("-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----\n"),
		}, nil
	}

	handler := Handle(nil, http.StatusOK, fn)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/x-pem-file" {
		t.Fatalf("Content-Type=%q, want application/x-pem-file", ct)
	}
	if cd := rec.Header().Get("Content-Disposition"); cd != `attachment; filename="test.pem"` {
		t.Fatalf("Content-Disposition=%q", cd)
	}
	if rec.Body.String() != "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----\n" {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}
