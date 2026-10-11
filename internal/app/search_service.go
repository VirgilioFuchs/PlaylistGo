package app

import (
	"context"
	"errors"
	"time"

	"musicgo/internal/domain"
)

type SearchCacheRepository interface {
	Get(ctx context.Context, query domain.NormalizedQuery, now time.Time) (domain.SearchCacheEntry, bool, error)
	Save(ctx context.Context, entry domain.SearchCacheEntry) error
}

type SearchBudgetRepository interface {
	CurrentWindow(ctx context.Context, now time.Time, dailyLimit int) (domain.SearchBudgetWindow, error)
	Consume(ctx context.Context, now time.Time, dailyLimit int) (domain.SearchBudgetWindow, bool, error)
}

type SearchProvider interface {
	SearchTracks(ctx context.Context, query string) ([]domain.TrackSnapshot, error)
}

type SearchService struct {
	cacheRepo  SearchCacheRepository
	budgetRepo SearchBudgetRepository
	provider   SearchProvider
	dailyLimit int
	now        func() time.Time
}

func NewSearchService(cacheRepo SearchCacheRepository, budgetRepo SearchBudgetRepository, provider SearchProvider, dailyLimit int, now func() time.Time) *SearchService {
	if now == nil {
		now = time.Now
	}

	return &SearchService{
		cacheRepo:  cacheRepo,
		budgetRepo: budgetRepo,
		provider:   provider,
		dailyLimit: dailyLimit,
		now:        now,
	}
}

func (s *SearchService) Search(ctx context.Context, rawQuery string) (domain.SearchResponse, error) {
	normalizedQuery, err := domain.NormalizeSearchQuery(rawQuery)
	if err != nil {
		return domain.SearchResponse{}, err
	}

	now := s.now().UTC()
	quotaWindow, err := s.budgetRepo.CurrentWindow(ctx, now, s.dailyLimit)
	if err != nil {
		return domain.SearchResponse{}, domain.NewUpstreamUnavailableError("Servico de busca indisponivel no momento.", err)
	}

	cached, found, err := s.cacheRepo.Get(ctx, normalizedQuery, now)
	if err != nil {
		return domain.SearchResponse{}, domain.NewUpstreamUnavailableError("Servico de busca indisponivel no momento.", err)
	}
	if found && cached.ExpiresAt.After(now) {
		return domain.SearchResponse{
			Data:  cached.Results,
			Quota: quotaWindow.PublicState(),
			Cache: domain.SearchCacheStatus{Hit: true},
		}, nil
	}

	quotaAfterConsume, consumed, err := s.budgetRepo.Consume(ctx, now, s.dailyLimit)
	if err != nil {
		return domain.SearchResponse{}, domain.NewUpstreamUnavailableError("Servico de busca indisponivel no momento.", err)
	}
	if !consumed {
		return domain.SearchResponse{
			Quota: quotaAfterConsume.PublicState(),
			Cache: domain.SearchCacheStatus{Hit: false},
		}, domain.NewQuotaLimitedError(domain.QuotaLimitedMessage, nil)
	}

	tracks, err := s.provider.SearchTracks(ctx, string(normalizedQuery))
	if err != nil {
		var searchErr *domain.SearchError
		if errors.As(err, &searchErr) {
			return domain.SearchResponse{
				Quota: quotaAfterConsume.PublicState(),
				Cache: domain.SearchCacheStatus{Hit: false},
			}, searchErr
		}
		return domain.SearchResponse{
			Quota: quotaAfterConsume.PublicState(),
			Cache: domain.SearchCacheStatus{Hit: false},
		}, domain.NewUpstreamUnavailableError("Servico do YouTube indisponivel no momento.", err)
	}

	entry := domain.SearchCacheEntry{
		Query:     normalizedQuery,
		Results:   tracks,
		CachedAt:  now,
		ExpiresAt: now.Add(domain.SearchCacheTTL),
	}
	if err := s.cacheRepo.Save(ctx, entry); err != nil {
		return domain.SearchResponse{}, domain.NewUpstreamUnavailableError("Servico de busca indisponivel no momento.", err)
	}

	return domain.SearchResponse{
		Data:  tracks,
		Quota: quotaAfterConsume.PublicState(),
		Cache: domain.SearchCacheStatus{Hit: false},
	}, nil
}
