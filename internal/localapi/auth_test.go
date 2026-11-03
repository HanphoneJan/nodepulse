package localapi

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestTokenRequired(t *testing.T) {
	s := New(Options{Token: "test-token", ResultsPath: filepath.Join(t.TempDir(), "results.json")})
	handler := s.Handler()

	req := httptest.NewRequest(http.MethodGet, "/api/version", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("without token status = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/version", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("with token status = %d body=%s", rec.Code, rec.Body.String())
	}
}
