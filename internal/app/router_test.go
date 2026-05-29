package app_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"review-view/internal/app"
	"review-view/internal/config"
)

func TestRouterRegistersCoreRoutes(t *testing.T) {
	server, err := app.NewServer(config.Config{
		Addr:        ":0",
		DatabaseDSN: "file::memory:?cache=shared",
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	router := server.Handler()
	cases := []string{"/", "/projects", "/models", "/tasks", "/settings"}
	for _, path := range cases {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code == http.StatusNotFound {
			t.Fatalf("expected route %s to exist", path)
		}
	}
}
