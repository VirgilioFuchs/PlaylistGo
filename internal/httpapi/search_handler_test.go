package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"musicgo/internal/config"
	"musicgo/internal/domain"
)

func TestSearchHandlerReturnsSuccess(t *testing.T) {
	handler := NewRouter(config.Config{Environment: "development"}, Dependencies{
		SearchService: fakeHTTPSearchService{
			response: domain.SearchResponse{
				Data:  []domain.TrackSnapshot{{VideoID: "abc", Title: "Song"}},
				Quota: domain.SearchQuota{DailyLimit: 100, Remaining: 99, UsedPercent: 1},
				Cache: domain.SearchCacheStatus{Hit: false},
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/search?q=song", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{`"videoId":"abc"`, `"dailyLimit":100`, `"hit":false`} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected body to contain %q, got %s", want, body)
		}
	}
}

func TestSearchHandlerRejectsEmptyQuery(t *testing.T) {
	handler := NewRouter(config.Config{Environment: "development"}, Dependencies{
		SearchService: fakeHTTPSearchService{err: domain.NewValidationError("Informe um termo de busca.")},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/search?q=", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestSearchHandlerReturnsQuotaLimited(t *testing.T) {
	handler := NewRouter(config.Config{Environment: "development"}, Dependencies{
		SearchService: fakeHTTPSearchService{
			response: domain.SearchResponse{
				Quota: domain.SearchQuota{DailyLimit: 100, Remaining: 0, UsedPercent: 100, AlertLevel: "0"},
			},
			err: domain.NewQuotaLimitedError(domain.QuotaLimitedMessage, nil),
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/search?q=song", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"quota":{"dailyLimit":100`) {
		t.Fatalf("expected quota payload, got %s", rec.Body.String())
	}
}

func TestSearchHandlerReturnsUpstreamUnavailable(t *testing.T) {
	handler := NewRouter(config.Config{Environment: "development"}, Dependencies{
		SearchService: fakeHTTPSearchService{
			err: domain.NewUpstreamUnavailableError("Servico do YouTube indisponivel no momento.", nil),
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/search?q=song", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rec.Code)
	}
}

type fakeHTTPSearchService struct {
	response domain.SearchResponse
	err      error
}

func (f fakeHTTPSearchService) Search(context.Context, string) (domain.SearchResponse, error) {
	return f.response, f.err
}
