package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"musicgo/internal/domain"
)

func TestSearchServiceReturnsCacheHitWithoutBudgetChange(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	cache := &fakeSearchCacheRepository{
		entry: domain.SearchCacheEntry{
			Query:     "the cure",
			Results:   []domain.TrackSnapshot{{VideoID: "abc", Title: "Lovesong"}},
			CachedAt:  now.Add(-time.Hour),
			ExpiresAt: now.Add(time.Hour),
		},
		found: true,
	}
	budget := &fakeSearchBudgetRepository{
		current: domain.SearchBudgetWindow{DailyLimit: 100, Used: 20},
	}
	provider := &fakeSearchProvider{}
	service := NewSearchService(cache, budget, provider, 100, func() time.Time { return now })

	response, err := service.Search(context.Background(), "  The   Cure  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !response.Cache.Hit {
		t.Fatal("expected cache hit")
	}
	if budget.consumeCalls != 0 {
		t.Fatalf("expected no budget consume, got %d", budget.consumeCalls)
	}
	if provider.calls != 0 {
		t.Fatalf("expected no youtube call, got %d", provider.calls)
	}
	if got := response.Data[0].Title; got != "Lovesong" {
		t.Fatalf("unexpected cached title %q", got)
	}
}

func TestSearchServiceCachesMissAndConsumesBudget(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	cache := &fakeSearchCacheRepository{}
	budget := &fakeSearchBudgetRepository{
		current:   domain.SearchBudgetWindow{DailyLimit: 100, Used: 10},
		consumed:  domain.SearchBudgetWindow{DailyLimit: 100, Used: 11},
		consumeOK: true,
	}
	provider := &fakeSearchProvider{
		results: []domain.TrackSnapshot{{VideoID: "abc", Title: "Song"}},
	}
	service := NewSearchService(cache, budget, provider, 100, func() time.Time { return now })

	response, err := service.Search(context.Background(), "  Song  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Cache.Hit {
		t.Fatal("expected cache miss")
	}
	if provider.lastQuery != "song" {
		t.Fatalf("expected normalized query, got %q", provider.lastQuery)
	}
	if cache.saved.Query != "song" {
		t.Fatalf("expected saved normalized query, got %q", cache.saved.Query)
	}
	if cache.saved.ExpiresAt.Sub(cache.saved.CachedAt) != domain.SearchCacheTTL {
		t.Fatalf("unexpected cache ttl %s", cache.saved.ExpiresAt.Sub(cache.saved.CachedAt))
	}
	if response.Quota.Remaining != 89 || response.Quota.UsedPercent != 11 {
		t.Fatalf("unexpected quota state %+v", response.Quota)
	}
}

func TestSearchServiceRejectsEmptyQuery(t *testing.T) {
	service := NewSearchService(&fakeSearchCacheRepository{}, &fakeSearchBudgetRepository{}, &fakeSearchProvider{}, 100, time.Now)

	if _, err := service.Search(context.Background(), "   "); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestSearchServiceBlocksWhenBudgetIsExhausted(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	cache := &fakeSearchCacheRepository{}
	budget := &fakeSearchBudgetRepository{
		current:  domain.SearchBudgetWindow{DailyLimit: 100, Used: 100},
		consumed: domain.SearchBudgetWindow{DailyLimit: 100, Used: 100},
	}
	provider := &fakeSearchProvider{}
	service := NewSearchService(cache, budget, provider, 100, func() time.Time { return now })

	response, err := service.Search(context.Background(), "radiohead")
	if err == nil {
		t.Fatal("expected quota error")
	}
	if !domain.IsQuotaLimited(err) {
		t.Fatalf("expected quota-limited error, got %v", err)
	}
	if response.Quota.AlertLevel != "0" || response.Quota.Remaining != 0 {
		t.Fatalf("unexpected quota response %+v", response.Quota)
	}
	if provider.calls != 0 {
		t.Fatalf("expected no youtube call, got %d", provider.calls)
	}
}

func TestSearchServiceMapsProviderErrors(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	cache := &fakeSearchCacheRepository{}
	budget := &fakeSearchBudgetRepository{
		current:   domain.SearchBudgetWindow{DailyLimit: 100, Used: 50},
		consumed:  domain.SearchBudgetWindow{DailyLimit: 100, Used: 51},
		consumeOK: true,
	}
	provider := &fakeSearchProvider{
		err: domain.NewUpstreamUnavailableError("Servico do YouTube indisponivel no momento.", errors.New("timeout")),
	}
	service := NewSearchService(cache, budget, provider, 100, func() time.Time { return now })

	response, err := service.Search(context.Background(), "bjork")
	if err == nil {
		t.Fatal("expected provider error")
	}
	var searchErr *domain.SearchError
	if !errors.As(err, &searchErr) || searchErr.Kind != domain.SearchErrorUpstreamUnavailable {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Quota.Remaining != 49 {
		t.Fatalf("expected decremented quota, got %+v", response.Quota)
	}
}

func TestSearchQuotaAlertLevels(t *testing.T) {
	cases := []struct {
		name      string
		window    domain.SearchBudgetWindow
		alert     string
		remaining int
	}{
		{name: "normal", window: domain.SearchBudgetWindow{DailyLimit: 100, Used: 60}, alert: "", remaining: 40},
		{name: "25", window: domain.SearchBudgetWindow{DailyLimit: 100, Used: 75}, alert: "25", remaining: 25},
		{name: "15", window: domain.SearchBudgetWindow{DailyLimit: 100, Used: 85}, alert: "15", remaining: 15},
		{name: "10", window: domain.SearchBudgetWindow{DailyLimit: 100, Used: 90}, alert: "10", remaining: 10},
		{name: "5", window: domain.SearchBudgetWindow{DailyLimit: 100, Used: 95}, alert: "5", remaining: 5},
		{name: "0", window: domain.SearchBudgetWindow{DailyLimit: 100, Used: 100}, alert: "0", remaining: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			quota := tc.window.PublicState()
			if quota.AlertLevel != tc.alert || quota.Remaining != tc.remaining {
				t.Fatalf("unexpected quota state %+v", quota)
			}
		})
	}
}

type fakeSearchCacheRepository struct {
	entry domain.SearchCacheEntry
	found bool
	saved domain.SearchCacheEntry
	err   error
}

func (f *fakeSearchCacheRepository) Get(context.Context, domain.NormalizedQuery, time.Time) (domain.SearchCacheEntry, bool, error) {
	return f.entry, f.found, f.err
}

func (f *fakeSearchCacheRepository) Save(_ context.Context, entry domain.SearchCacheEntry) error {
	f.saved = entry
	return f.err
}

type fakeSearchBudgetRepository struct {
	current      domain.SearchBudgetWindow
	consumed     domain.SearchBudgetWindow
	currentErr   error
	consumeErr   error
	consumeOK    bool
	consumeCalls int
}

func (f *fakeSearchBudgetRepository) CurrentWindow(context.Context, time.Time, int) (domain.SearchBudgetWindow, error) {
	return f.current, f.currentErr
}

func (f *fakeSearchBudgetRepository) Consume(context.Context, time.Time, int) (domain.SearchBudgetWindow, bool, error) {
	f.consumeCalls++
	if f.consumeErr != nil {
		return domain.SearchBudgetWindow{}, false, f.consumeErr
	}
	if !f.consumeOK && f.consumed == (domain.SearchBudgetWindow{}) {
		return f.current, false, nil
	}
	if !f.consumeOK {
		return f.consumed, false, nil
	}
	return f.consumed, true, nil
}

type fakeSearchProvider struct {
	results   []domain.TrackSnapshot
	err       error
	calls     int
	lastQuery string
}

func (f *fakeSearchProvider) SearchTracks(_ context.Context, query string) ([]domain.TrackSnapshot, error) {
	f.calls++
	f.lastQuery = query
	if f.err != nil {
		return nil, f.err
	}
	return f.results, nil
}
