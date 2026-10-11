package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	mongoadapter "musicgo/internal/adapters/mongo"
	youtubeadapter "musicgo/internal/adapters/youtube"
	"musicgo/internal/config"
	"musicgo/internal/httpapi"
)

const defaultShutdownTimeout = 5 * time.Second

func Run(ctx context.Context, cfg config.Config) error {
	searchService, playlistService, cleanup, err := newServices(ctx, cfg)
	if err != nil {
		return err
	}
	defer cleanup()

	srv := &http.Server{
		Addr:              net.JoinHostPort("", cfg.Port),
		Handler:           httpapi.NewRouter(cfg, httpapi.Dependencies{SearchService: searchService, PlaylistService: playlistService}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)

	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		return nil
	case err := <-errCh:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func newServices(ctx context.Context, cfg config.Config) (*SearchService, *PlaylistService, func(), error) {
	mongoClient, err := mongoadapter.NewClient(ctx, cfg.MongoURI)
	if err != nil {
		return nil, nil, nil, err
	}

	databaseName, err := mongoadapter.DatabaseNameFromURI(cfg.MongoURI)
	if err != nil {
		_ = mongoClient.Disconnect(context.Background())
		return nil, nil, nil, err
	}

	db := mongoClient.Database(databaseName)
	cacheRepository := mongoadapter.NewSearchCacheRepository(db)
	if err := cacheRepository.EnsureIndexes(ctx); err != nil {
		_ = mongoClient.Disconnect(context.Background())
		return nil, nil, nil, err
	}

	budgetRepository, err := mongoadapter.NewSearchBudgetRepository(db)
	if err != nil {
		_ = mongoClient.Disconnect(context.Background())
		return nil, nil, nil, err
	}
	if err := budgetRepository.EnsureIndexes(ctx); err != nil {
		_ = mongoClient.Disconnect(context.Background())
		return nil, nil, nil, err
	}

	playlistRepository := mongoadapter.NewPlaylistRepository(db)
	if err := playlistRepository.EnsureIndexes(ctx); err != nil {
		_ = mongoClient.Disconnect(context.Background())
		return nil, nil, nil, err
	}

	youtubeClient, err := youtubeadapter.NewClientFromOAuth(ctx, cfg.YouTubeClientID, cfg.YouTubeClientSecret, cfg.YouTubeTokenPath, nil)
	if err != nil {
		_ = mongoClient.Disconnect(context.Background())
		return nil, nil, nil, err
	}

	searchService := NewSearchService(
		cacheRepository,
		budgetRepository,
		youtubeClient,
		cfg.YouTubeSearchDailyBudget,
		time.Now,
	)

	playlistService := NewPlaylistService(playlistRepository, time.Now)

	cleanup := func() {
		_ = mongoClient.Disconnect(context.Background())
	}

	return searchService, playlistService, cleanup, nil
}
