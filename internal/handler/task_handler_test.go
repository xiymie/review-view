package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebhookTriggersTask(t *testing.T) {
	router := NewTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/webhook/1", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rec.Code)
	}
}
