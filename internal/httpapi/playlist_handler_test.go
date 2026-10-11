package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"musicgo/internal/config"
	"musicgo/internal/domain"
)

func TestPlaylistHandlerCreatesPlaylist(t *testing.T) {
	handler := NewRouter(config.Config{Environment: "development"}, Dependencies{
		PlaylistService: fakeHTTPPlaylistService{
			createFn: func(_ context.Context, rawName string) (domain.Playlist, error) {
				if rawName != "  Road   Trip  " {
					t.Fatalf("unexpected raw name %q", rawName)
				}
				return domain.Playlist{
					ID:         "p1",
					Name:       "Road Trip",
					TrackCount: 0,
					CreatedAt:  "2026-06-23T10:00:00Z",
					UpdatedAt:  "2026-06-23T10:00:00Z",
				}, nil
			},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/playlists", strings.NewReader(`{"name":"  Road   Trip  "}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{`"id":"p1"`, `"name":"Road Trip"`, `"trackCount":0`, `"durationSeconds":0`} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected body to contain %q, got %s", want, body)
		}
	}
}

func TestPlaylistHandlerRejectsInvalidSort(t *testing.T) {
	handler := NewRouter(config.Config{Environment: "development"}, Dependencies{
		PlaylistService: fakeHTTPPlaylistService{},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/playlists?sort=fastest", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"code":"validation_error"`) {
		t.Fatalf("expected validation error payload, got %s", rec.Body.String())
	}
}

func TestPlaylistHandlerForwardSortAllowlist(t *testing.T) {
	var gotSort domain.PlaylistSort
	handler := NewRouter(config.Config{Environment: "development"}, Dependencies{
		PlaylistService: fakeHTTPPlaylistService{
			listFn: func(_ context.Context, sort domain.PlaylistSort) ([]domain.Playlist, error) {
				gotSort = sort
				return []domain.Playlist{{
					ID:              "p1",
					Name:            "Road Trip",
					TrackCount:      2,
					DurationSeconds: 360,
					CreatedAt:       "2026-06-23T10:00:00Z",
					UpdatedAt:       "2026-06-23T10:05:00Z",
				}}, nil
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/playlists?sort=name", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if gotSort != domain.PlaylistSortName {
		t.Fatalf("expected sort allowlist to forward name, got %q", gotSort)
	}
	for _, want := range []string{`"trackCount":2`, `"durationSeconds":360`} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("expected playlist summary to contain %q, got %s", want, rec.Body.String())
		}
	}
}

func TestPlaylistHandlerRejectsDuplicateTrackConflict(t *testing.T) {
	handler := NewRouter(config.Config{Environment: "development"}, Dependencies{
		PlaylistService: fakeHTTPPlaylistService{
			addFn: func(context.Context, string, domain.TrackSnapshot) (domain.Playlist, error) {
				return domain.Playlist{}, domain.NewPlaylistConflictError("Essa faixa ja existe nesta playlist.", nil)
			},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/playlists/p1/tracks", strings.NewReader(`{
		"videoId":"abc123",
		"title":"Song",
		"channel":"Band",
		"durationSeconds":180,
		"thumbnailUrl":"https://example.com/thumb.jpg"
	}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"code":"conflict"`) {
		t.Fatalf("expected conflict payload, got %s", rec.Body.String())
	}
}

func TestPlaylistHandlerReturnsNotFoundForMissingTrackRemoval(t *testing.T) {
	handler := NewRouter(config.Config{Environment: "development"}, Dependencies{
		PlaylistService: fakeHTTPPlaylistService{
			removeFn: func(context.Context, string, string) (domain.Playlist, error) {
				return domain.Playlist{}, domain.NewPlaylistNotFoundError("Faixa nao encontrada na playlist.", nil)
			},
		},
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/playlists/p1/tracks/missing", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
}

func TestPlaylistHandlerRejectsInvalidPlaylistName(t *testing.T) {
	handler := NewRouter(config.Config{Environment: "development"}, Dependencies{
		PlaylistService: fakeHTTPPlaylistService{
			createFn: func(context.Context, string) (domain.Playlist, error) {
				return domain.Playlist{}, domain.NewPlaylistValidationError("Informe um nome para a playlist.")
			},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/playlists", strings.NewReader(`{"name":"   "}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestPlaylistHandlerReordersTracksAndReturnsCanonicalPlaylist(t *testing.T) {
	var gotTrackIDs []string
	var gotRevision int64
	handler := NewRouter(config.Config{Environment: "development"}, Dependencies{
		PlaylistService: fakeHTTPPlaylistService{
			reorderFn: func(_ context.Context, playlistID string, trackIDs []string, expectedRevision int64) (domain.Playlist, error) {
				if playlistID != "p1" {
					t.Fatalf("unexpected playlist id %q", playlistID)
				}
				gotTrackIDs = append([]string(nil), trackIDs...)
				gotRevision = expectedRevision
				return domain.Playlist{
					ID:       "p1",
					Name:     "Road Trip",
					Revision: 2,
					Tracks: []domain.TrackSnapshot{
						{VideoID: "second", Title: "Second", Channel: "Band B", DurationSeconds: 180, ThumbnailURL: "https://example.com/second.jpg"},
						{VideoID: "first", Title: "First", Channel: "Band A", DurationSeconds: 120, ThumbnailURL: "https://example.com/first.jpg"},
					},
					UpdatedAt: "2026-06-23T10:05:00Z",
				}, nil
			},
		},
	})

	req := httptest.NewRequest(http.MethodPut, "/api/playlists/p1/tracks/order", strings.NewReader(`{"trackIds":["second","first"],"expectedRevision":1}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if gotRevision != 1 || len(gotTrackIDs) != 2 || gotTrackIDs[0] != "second" || gotTrackIDs[1] != "first" {
		t.Fatalf("unexpected service request: ids=%v revision=%d", gotTrackIDs, gotRevision)
	}
	for _, want := range []string{`"revision":2`, `"updatedAt":"2026-06-23T10:05:00Z"`, `"videoId":"second"`} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("expected body to contain %q, got %s", want, rec.Body.String())
		}
	}
}

func TestPlaylistHandlerRejectsMalformedReorderPayload(t *testing.T) {
	testCases := []struct {
		name string
		body string
	}{
		{name: "unknown field", body: `{"trackIds":[],"expectedRevision":1,"extra":true}`},
		{name: "missing track ids", body: `{"expectedRevision":1}`},
		{name: "missing revision", body: `{"trackIds":[]}`},
		{name: "multiple json values", body: `{"trackIds":[],"expectedRevision":1} {"ignored":true}`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			called := false
			handler := NewRouter(config.Config{Environment: "development"}, Dependencies{
				PlaylistService: fakeHTTPPlaylistService{
					reorderFn: func(context.Context, string, []string, int64) (domain.Playlist, error) {
						called = true
						return domain.Playlist{}, nil
					},
				},
			})

			req := httptest.NewRequest(http.MethodPut, "/api/playlists/p1/tracks/order", strings.NewReader(testCase.body))
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d", rec.Code)
			}
			if !strings.Contains(rec.Body.String(), `"code":"validation_error"`) {
				t.Fatalf("expected validation payload, got %s", rec.Body.String())
			}
			if called {
				t.Fatal("expected malformed payload not to call the service")
			}
		})
	}
}

func TestPlaylistHandlerMapsReorderValidationAndConflictErrors(t *testing.T) {
	testCases := []struct {
		name       string
		serviceErr error
		wantStatus int
		wantCode   string
	}{
		{name: "invalid exact set", serviceErr: domain.NewPlaylistValidationError("invalid order"), wantStatus: http.StatusBadRequest, wantCode: "validation_error"},
		{name: "stale revision", serviceErr: domain.NewPlaylistConflictError("playlist changed", nil), wantStatus: http.StatusConflict, wantCode: "conflict"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			handler := NewRouter(config.Config{Environment: "development"}, Dependencies{
				PlaylistService: fakeHTTPPlaylistService{
					reorderFn: func(context.Context, string, []string, int64) (domain.Playlist, error) {
						return domain.Playlist{}, testCase.serviceErr
					},
				},
			})

			req := httptest.NewRequest(http.MethodPut, "/api/playlists/p1/tracks/order", strings.NewReader(`{"trackIds":["first"],"expectedRevision":1}`))
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != testCase.wantStatus {
				t.Fatalf("expected status %d, got %d", testCase.wantStatus, rec.Code)
			}
			if !strings.Contains(rec.Body.String(), `"code":"`+testCase.wantCode+`"`) {
				t.Fatalf("expected %s payload, got %s", testCase.wantCode, rec.Body.String())
			}
		})
	}
}

func TestPlaylistHandlerReservesOrderPathBeforeTrackDelete(t *testing.T) {
	removeCalled := false
	handler := NewRouter(config.Config{Environment: "development"}, Dependencies{
		PlaylistService: fakeHTTPPlaylistService{
			removeFn: func(context.Context, string, string) (domain.Playlist, error) {
				removeCalled = true
				return domain.Playlist{}, nil
			},
		},
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/playlists/p1/tracks/order", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", rec.Code)
	}
	if removeCalled {
		t.Fatal("expected order path not to route to track deletion")
	}
}

type fakeHTTPPlaylistService struct {
	createFn  func(context.Context, string) (domain.Playlist, error)
	listFn    func(context.Context, domain.PlaylistSort) ([]domain.Playlist, error)
	addFn     func(context.Context, string, domain.TrackSnapshot) (domain.Playlist, error)
	removeFn  func(context.Context, string, string) (domain.Playlist, error)
	reorderFn func(context.Context, string, []string, int64) (domain.Playlist, error)
}

func (f fakeHTTPPlaylistService) Create(ctx context.Context, rawName string) (domain.Playlist, error) {
	if f.createFn != nil {
		return f.createFn(ctx, rawName)
	}
	return domain.Playlist{}, domain.NewPlaylistValidationError("not implemented")
}

func (f fakeHTTPPlaylistService) List(ctx context.Context, sort domain.PlaylistSort) ([]domain.Playlist, error) {
	if f.listFn != nil {
		return f.listFn(ctx, sort)
	}
	return nil, nil
}

func (f fakeHTTPPlaylistService) AddTrack(ctx context.Context, playlistID string, snapshot domain.TrackSnapshot) (domain.Playlist, error) {
	if f.addFn != nil {
		return f.addFn(ctx, playlistID, snapshot)
	}
	return domain.Playlist{}, domain.NewPlaylistValidationError("not implemented")
}

func (f fakeHTTPPlaylistService) RemoveTrack(ctx context.Context, playlistID, videoID string) (domain.Playlist, error) {
	if f.removeFn != nil {
		return f.removeFn(ctx, playlistID, videoID)
	}
	return domain.Playlist{}, domain.NewPlaylistValidationError("not implemented")
}

func (f fakeHTTPPlaylistService) ReorderTracks(ctx context.Context, playlistID string, trackIDs []string, expectedRevision int64) (domain.Playlist, error) {
	if f.reorderFn != nil {
		return f.reorderFn(ctx, playlistID, trackIDs, expectedRevision)
	}
	return domain.Playlist{}, domain.NewPlaylistValidationError("not implemented")
}

var _ PlaylistService = fakeHTTPPlaylistService{}

func TestPlaylistHandlerMapsUnexpectedErrorsTo503(t *testing.T) {
	handler := NewRouter(config.Config{Environment: "development"}, Dependencies{
		PlaylistService: fakeHTTPPlaylistService{
			createFn: func(context.Context, string) (domain.Playlist, error) {
				return domain.Playlist{}, errors.New("db down")
			},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/playlists", strings.NewReader(`{"name":"Road Trip"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rec.Code)
	}
}
