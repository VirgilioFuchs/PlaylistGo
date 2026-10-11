package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"musicgo/internal/domain"
)

type PlaylistService interface {
	Create(ctx context.Context, rawName string) (domain.Playlist, error)
	List(ctx context.Context, sort domain.PlaylistSort) ([]domain.Playlist, error)
	AddTrack(ctx context.Context, playlistID string, snapshot domain.TrackSnapshot) (domain.Playlist, error)
	RemoveTrack(ctx context.Context, playlistID, videoID string) (domain.Playlist, error)
	ReorderTracks(ctx context.Context, playlistID string, trackIDs []string, expectedRevision int64) (domain.Playlist, error)
}

type PlaylistHandler struct {
	service PlaylistService
}

type playlistListResponse struct {
	Data []domain.Playlist `json:"data"`
}

type createPlaylistRequest struct {
	Name string `json:"name"`
}

type reorderTracksRequest struct {
	TrackIDs         *[]string `json:"trackIds"`
	ExpectedRevision *int64    `json:"expectedRevision"`
}

func NewPlaylistHandler(service PlaylistService) http.Handler {
	return &PlaylistHandler{service: service}
}

func (h *PlaylistHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/playlists" || r.URL.Path == "/api/playlists/":
		h.handleCollection(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/playlists/"):
		h.handleItem(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *PlaylistHandler) handleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleList(w, r)
	case http.MethodPost:
		h.handleCreate(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
	}
}

func (h *PlaylistHandler) handleItem(w http.ResponseWriter, r *http.Request) {
	parts := playlistPathParts(r.URL.Path)
	if len(parts) == 3 && parts[1] == "tracks" && parts[2] == "order" {
		if r.Method != http.MethodPut {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		h.handleReorderTracks(w, r, parts[0])
		return
	}
	if len(parts) == 3 && parts[1] == "tracks" && r.Method == http.MethodDelete {
		h.handleRemove(w, r, parts[0], parts[2])
		return
	}
	if len(parts) != 2 || parts[1] != "tracks" {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodPost:
		h.handleAddTrack(w, r, parts[0])
	case http.MethodDelete:
		http.NotFound(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
	}
}

func (h *PlaylistHandler) handleList(w http.ResponseWriter, r *http.Request) {
	sort, err := domain.NormalizePlaylistSort(r.URL.Query().Get("sort"))
	if err != nil {
		writePlaylistError(w, http.StatusBadRequest, err)
		return
	}

	playlists, err := h.service.List(r.Context(), sort)
	if err != nil {
		writePlaylistError(w, playlistStatusFromError(err), err)
		return
	}

	writeJSON(w, http.StatusOK, playlistListResponse{Data: playlists})
}

func (h *PlaylistHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req createPlaylistRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Payload invalido.")
		return
	}

	playlist, err := h.service.Create(r.Context(), req.Name)
	if err != nil {
		writePlaylistError(w, playlistStatusFromError(err), err)
		return
	}

	writeJSON(w, http.StatusCreated, playlist)
}

func (h *PlaylistHandler) handleAddTrack(w http.ResponseWriter, r *http.Request, playlistID string) {
	var req domain.TrackSnapshot
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Payload invalido.")
		return
	}

	playlist, err := h.service.AddTrack(r.Context(), playlistID, req)
	if err != nil {
		writePlaylistError(w, playlistStatusFromError(err), err)
		return
	}

	writeJSON(w, http.StatusOK, playlist)
}

func (h *PlaylistHandler) handleRemove(w http.ResponseWriter, r *http.Request, playlistID, videoID string) {
	playlist, err := h.service.RemoveTrack(r.Context(), playlistID, videoID)
	if err != nil {
		writePlaylistError(w, playlistStatusFromError(err), err)
		return
	}

	writeJSON(w, http.StatusOK, playlist)
}

func (h *PlaylistHandler) handleReorderTracks(w http.ResponseWriter, r *http.Request, playlistID string) {
	var req reorderTracksRequest
	if err := decodeJSONBody(r, &req); err != nil || req.TrackIDs == nil || req.ExpectedRevision == nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Payload invalido.")
		return
	}

	playlist, err := h.service.ReorderTracks(r.Context(), playlistID, *req.TrackIDs, *req.ExpectedRevision)
	if err != nil {
		writePlaylistError(w, playlistStatusFromError(err), err)
		return
	}

	writeJSON(w, http.StatusOK, playlist)
}

func decodeJSONBody(r *http.Request, v any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("payload must contain a single JSON value")
	}
	return nil
}

func playlistPathParts(path string) []string {
	if !strings.HasPrefix(path, "/api/playlists/") {
		return nil
	}

	trimmed := strings.TrimPrefix(path, "/api/playlists/")
	trimmed = strings.Trim(trimmed, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func playlistStatusFromError(err error) int {
	var playlistErr *domain.PlaylistError
	if !errors.As(err, &playlistErr) {
		return http.StatusServiceUnavailable
	}

	switch playlistErr.Kind {
	case domain.PlaylistErrorValidation:
		return http.StatusBadRequest
	case domain.PlaylistErrorConflict:
		return http.StatusConflict
	case domain.PlaylistErrorNotFound:
		return http.StatusNotFound
	default:
		return http.StatusServiceUnavailable
	}
}

func writePlaylistError(w http.ResponseWriter, status int, err error) {
	var playlistErr *domain.PlaylistError
	if errors.As(err, &playlistErr) {
		writeError(w, status, playlistErr.PublicCode(), playlistErr.Message)
		return
	}
	writeError(w, status, "upstream_unavailable", "Servico indisponivel no momento.")
}
