package httpapi

import (
	"context"
	"net/http"

	"musicgo/internal/config"
	"musicgo/internal/domain"
)

type SearchService interface {
	Search(rctx context.Context, rawQuery string) (domain.SearchResponse, error)
}

type Dependencies struct {
	SearchService   SearchService
	PlaylistService PlaylistService
}

func NewRouter(cfg config.Config, deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/healthz", NewHealthHandler("music-go", cfg.Environment))
	mux.Handle("/api/search", NewSearchHandler(deps.SearchService))
	mux.Handle("/api/playlists", NewPlaylistHandler(deps.PlaylistService))
	mux.Handle("/api/playlists/", NewPlaylistHandler(deps.PlaylistService))
	return mux
}
