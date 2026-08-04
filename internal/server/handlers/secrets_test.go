package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Omotolani98/foostash/internal/server/middleware"
	"github.com/Omotolani98/foostash/internal/service"
)

func doBulkSet(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	h := &SecretsHandler{Secrets: service.NewSecrets(nil)}
	req := httptest.NewRequest(http.MethodPut, "/v1/projects/demo/envs/dev/secrets", strings.NewReader(body))
	rec := httptest.NewRecorder()
	middleware.BufferBody(http.HandlerFunc(h.BulkSet)).ServeHTTP(rec, req)
	return rec
}

func TestBulkSetRejectsInvalidJSON(t *testing.T) {
	rec := doBulkSet(t, "{not json")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestBulkSetRejectsEmptySecretList(t *testing.T) {
	rec := doBulkSet(t, `{"secrets":[]}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
