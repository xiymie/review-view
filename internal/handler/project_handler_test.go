package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSPARoutesServeVueApp(t *testing.T) {
	router := NewTestRouter(t)

	cases := []string{"/", "/projects", "/models", "/tasks", "/settings", "/login"}
	for _, path := range cases {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("path %s: expected 200, got %d", path, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "<html") {
			t.Fatalf("path %s: expected HTML response", path)
		}
	}
}
