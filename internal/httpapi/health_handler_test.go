package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"musicgo/internal/config"
)

func TestHealthHandler(t *testing.T) {
	handler := NewRouter(config.Config{Environment: "development", Port: "8080", MongoURI: "mongodb://localhost:27017/musicgo"}, Dependencies{})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{`"status":"ok"`, `"service":"music-go"`, `"environment":"development"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected response body to contain %q, got %s", want, body)
		}
	}
}

func TestHealthHandlerRejectsNonGet(t *testing.T) {
	handler := NewRouter(config.Config{Environment: "development", Port: "8080", MongoURI: "mongodb://localhost:27017/musicgo"}, Dependencies{})

	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", rec.Code)
	}
}
