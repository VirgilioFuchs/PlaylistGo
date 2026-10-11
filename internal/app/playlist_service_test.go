package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"musicgo/internal/domain"
)

func TestPlaylistServiceCreateSanitizesNameAndSetsTimestamps(t *testing.T) {
	now := time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC)
	repo := newFakePlaylistRepository()
	service := NewPlaylistService(repo, func() time.Time { return now })

	playlist, err := service.Create(context.Background(), "  Best   of  90s  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if playlist.Name != "Best of 90s" {
		t.Fatalf("unexpected cleaned name %q", playlist.Name)
	}
	if playlist.TrackCount != 0 {
		t.Fatalf("expected empty playlist, got %d tracks", playlist.TrackCount)
	}
	if playlist.DurationSeconds != 0 {
		t.Fatalf("expected empty playlist duration, got %d", playlist.DurationSeconds)
	}
	if playlist.Revision != 1 {
		t.Fatalf("expected initial revision 1, got %d", playlist.Revision)
	}
	if repo.created.ID == "" {
		t.Fatal("expected repository create call")
	}
	if repo.created.Name.Display != "Best of 90s" || repo.created.Name.Normalized != "best of 90s" {
		t.Fatalf("unexpected stored name %+v", repo.created.Name)
	}
	if !repo.created.CreatedAt.Equal(now) || !repo.created.UpdatedAt.Equal(now) {
		t.Fatalf("unexpected timestamps %+v", repo.created)
	}
}

func TestPlaylistServiceCreateRejectsCanonicalDuplicateName(t *testing.T) {
	now := time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC)
	repo := newFakePlaylistRepository()
	repo.existingByName["best of 90s"] = playlistRecord("p1", "Best of 90s", now)
	service := NewPlaylistService(repo, func() time.Time { return now })

	_, err := service.Create(context.Background(), "  BEST   OF 90S ")
	if err == nil || !domain.IsPlaylistConflictError(err) {
		t.Fatalf("expected conflict error, got %v", err)
	}
}

func TestPlaylistServiceListPassesSortMode(t *testing.T) {
	now := time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC)
	repo := newFakePlaylistRepository()
	repo.listResult = []domain.PlaylistRecord{
		playlistRecordWithTracks("p1", "A", now, []domain.TrackSnapshot{
			{VideoID: "a", Title: "A", Channel: "Band", DurationSeconds: 120, ThumbnailURL: "https://example.com/a.jpg"},
			{VideoID: "b", Title: "B", Channel: "Band", DurationSeconds: -5, ThumbnailURL: "https://example.com/b.jpg"},
		}),
		playlistRecord("p2", "B", now),
	}
	service := NewPlaylistService(repo, time.Now)

	playlists, err := service.List(context.Background(), domain.PlaylistSortName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastSort != domain.PlaylistSortName {
		t.Fatalf("expected sort to be forwarded, got %q", repo.lastSort)
	}
	if len(playlists) != 2 {
		t.Fatalf("unexpected playlist count %d", len(playlists))
	}
	if playlists[0].DurationSeconds != 120 {
		t.Fatalf("expected aggregate duration 120, got %d", playlists[0].DurationSeconds)
	}
	if playlists[0].Tracks != nil {
		t.Fatalf("expected list response to omit tracks, got %+v", playlists[0].Tracks)
	}
}

func TestPlaylistServiceAddTrackReturnsUpdatedDuration(t *testing.T) {
	now := time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC)
	repo := newFakePlaylistRepository()
	repo.records["p1"] = playlistRecordWithTracks("p1", "Road Trip", now, []domain.TrackSnapshot{{
		VideoID:         "existing",
		Title:           "Existing",
		Channel:         "Band",
		DurationSeconds: 180,
		ThumbnailURL:    "https://example.com/existing.jpg",
	}})
	service := NewPlaylistService(repo, func() time.Time { return now })

	playlist, err := service.AddTrack(context.Background(), "p1", domain.TrackSnapshot{
		VideoID:         "new",
		Title:           "New",
		Channel:         "Band",
		DurationSeconds: 75,
		ThumbnailURL:    "https://example.com/new.jpg",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if playlist.DurationSeconds != 255 {
		t.Fatalf("expected updated duration 255, got %d", playlist.DurationSeconds)
	}
	if len(playlist.Tracks) != 2 {
		t.Fatalf("expected mutation response tracks, got %+v", playlist.Tracks)
	}
	if playlist.Revision != 2 {
		t.Fatalf("expected revision 2 after add, got %d", playlist.Revision)
	}
	if repo.lastExpectedRevision != 1 {
		t.Fatalf("expected conditional update from revision 1, got %d", repo.lastExpectedRevision)
	}
}

func TestPlaylistServiceAddTrackRejectsDuplicateVideoID(t *testing.T) {
	now := time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC)
	repo := newFakePlaylistRepository()
	repo.records["p1"] = playlistRecordWithTracks("p1", "Road Trip", now, []domain.TrackSnapshot{{
		VideoID:         "abc123",
		Title:           "Song",
		Channel:         "Band",
		DurationSeconds: 180,
		ThumbnailURL:    "https://example.com/thumb.jpg",
		AddedAt:         now.Format(time.RFC3339),
	}})
	service := NewPlaylistService(repo, func() time.Time { return now })

	_, err := service.AddTrack(context.Background(), "p1", domain.TrackSnapshot{
		VideoID:         " abc123 ",
		Title:           "Song 2",
		Channel:         "Band",
		DurationSeconds: 181,
		ThumbnailURL:    "https://example.com/thumb2.jpg",
	})
	if err == nil || !domain.IsPlaylistConflictError(err) {
		t.Fatalf("expected duplicate-track conflict, got %v", err)
	}
}

func TestPlaylistServiceAddTrackRejectsInvalidSnapshot(t *testing.T) {
	service := NewPlaylistService(newFakePlaylistRepository(), time.Now)

	_, err := service.AddTrack(context.Background(), "p1", domain.TrackSnapshot{VideoID: "abc"})
	if err == nil || !domain.IsPlaylistValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestPlaylistServiceRemoveTrackReturnsNotFoundForMissingVideoID(t *testing.T) {
	now := time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC)
	repo := newFakePlaylistRepository()
	repo.records["p1"] = playlistRecordWithTracks("p1", "Road Trip", now, []domain.TrackSnapshot{{
		VideoID:         "abc123",
		Title:           "Song",
		Channel:         "Band",
		DurationSeconds: 180,
		ThumbnailURL:    "https://example.com/thumb.jpg",
		AddedAt:         now.Format(time.RFC3339),
	}})
	service := NewPlaylistService(repo, func() time.Time { return now })

	_, err := service.RemoveTrack(context.Background(), "p1", "missing")
	if err == nil || !domain.IsPlaylistNotFoundError(err) {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestPlaylistServiceRemoveTrackUpdatesPlaylist(t *testing.T) {
	now := time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC)
	repo := newFakePlaylistRepository()
	repo.records["p1"] = playlistRecordWithTracks("p1", "Road Trip", now, []domain.TrackSnapshot{{
		VideoID:         "abc123",
		Title:           "Song",
		Channel:         "Band",
		DurationSeconds: 180,
		ThumbnailURL:    "https://example.com/thumb.jpg",
		AddedAt:         now.Format(time.RFC3339),
	}})
	service := NewPlaylistService(repo, func() time.Time { return now.Add(time.Minute) })

	playlist, err := service.RemoveTrack(context.Background(), "p1", "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if playlist.TrackCount != 0 {
		t.Fatalf("expected empty playlist, got %d", playlist.TrackCount)
	}
	if playlist.DurationSeconds != 0 {
		t.Fatalf("expected empty playlist duration, got %d", playlist.DurationSeconds)
	}
	if len(repo.updated.Tracks) != 0 {
		t.Fatalf("expected track removal to persist, got %+v", repo.updated.Tracks)
	}
	if !repo.updated.UpdatedAt.Equal(now.Add(time.Minute).UTC()) {
		t.Fatalf("unexpected updated timestamp %+v", repo.updated.UpdatedAt)
	}
	if playlist.Revision != 2 {
		t.Fatalf("expected revision 2 after removal, got %d", playlist.Revision)
	}
	if repo.lastExpectedRevision != 1 {
		t.Fatalf("expected conditional update from revision 1, got %d", repo.lastExpectedRevision)
	}
}

func TestPlaylistServiceReorderTracksRebuildsCanonicalTrackOrder(t *testing.T) {
	now := time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC)
	repo := newFakePlaylistRepository()
	repo.records["p1"] = playlistRecordWithTracks("p1", "Road Trip", now, []domain.TrackSnapshot{
		{VideoID: "first", Title: "First", Channel: "Band A", DurationSeconds: 120, ThumbnailURL: "https://example.com/first.jpg", AddedAt: now.Format(time.RFC3339)},
		{VideoID: "second", Title: "Second", Channel: "Band B", DurationSeconds: 180, ThumbnailURL: "https://example.com/second.jpg", AddedAt: now.Format(time.RFC3339)},
	})
	service := NewPlaylistService(repo, func() time.Time { return now.Add(time.Minute) })

	playlist, err := service.ReorderTracks(context.Background(), "p1", []string{" second ", "first"}, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if playlist.Revision != 2 {
		t.Fatalf("expected revision 2 after reorder, got %d", playlist.Revision)
	}
	if got := []string{playlist.Tracks[0].VideoID, playlist.Tracks[1].VideoID}; got[0] != "second" || got[1] != "first" {
		t.Fatalf("unexpected reordered ids %v", got)
	}
	if playlist.Tracks[0].Title != "Second" || playlist.Tracks[0].Channel != "Band B" {
		t.Fatalf("expected server snapshot metadata, got %+v", playlist.Tracks[0])
	}
	if repo.lastExpectedRevision != 1 {
		t.Fatalf("expected conditional update from revision 1, got %d", repo.lastExpectedRevision)
	}
}

func TestPlaylistServiceReorderTracksRejectsInvalidExactSetWithoutPersisting(t *testing.T) {
	now := time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC)
	testCases := []struct {
		name     string
		trackIDs []string
	}{
		{name: "empty id", trackIDs: []string{""}},
		{name: "duplicate id", trackIDs: []string{"first", "first"}},
		{name: "unknown id", trackIDs: []string{"first", "unknown"}},
		{name: "missing id", trackIDs: []string{"first"}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			repo := newFakePlaylistRepository()
			repo.records["p1"] = playlistRecordWithTracks("p1", "Road Trip", now, []domain.TrackSnapshot{
				{VideoID: "first", Title: "First", Channel: "Band A", DurationSeconds: 120, ThumbnailURL: "https://example.com/first.jpg"},
				{VideoID: "second", Title: "Second", Channel: "Band B", DurationSeconds: 180, ThumbnailURL: "https://example.com/second.jpg"},
			})
			service := NewPlaylistService(repo, func() time.Time { return now.Add(time.Minute) })

			_, err := service.ReorderTracks(context.Background(), "p1", testCase.trackIDs, 1)
			if err == nil || !domain.IsPlaylistValidationError(err) {
				t.Fatalf("expected validation error, got %v", err)
			}
			if repo.updateCalled {
				t.Fatal("expected invalid order not to persist")
			}
		})
	}
}

func TestPlaylistServiceReorderTracksRejectsStaleRevisionWithoutPersisting(t *testing.T) {
	now := time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC)
	repo := newFakePlaylistRepository()
	repo.records["p1"] = playlistRecordWithTracks("p1", "Road Trip", now, []domain.TrackSnapshot{{
		VideoID: "first", Title: "First", Channel: "Band A", DurationSeconds: 120, ThumbnailURL: "https://example.com/first.jpg",
	}})
	repo.records["p1"] = withRevision(repo.records["p1"], 2)
	service := NewPlaylistService(repo, time.Now)

	_, err := service.ReorderTracks(context.Background(), "p1", []string{"first"}, 1)
	if err == nil || !domain.IsPlaylistConflictError(err) {
		t.Fatalf("expected stale revision conflict, got %v", err)
	}
	if repo.updateCalled {
		t.Fatal("expected stale order not to persist")
	}
}

func TestPlaylistServiceReorderTracksRejectsNegativeRevisionWithoutReadingOrPersisting(t *testing.T) {
	repo := newFakePlaylistRepository()
	service := NewPlaylistService(repo, time.Now)

	_, err := service.ReorderTracks(context.Background(), "p1", []string{"first"}, -1)
	if err == nil || !domain.IsPlaylistValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if repo.updateCalled {
		t.Fatal("expected negative revision not to persist")
	}
}

func TestPlaylistServiceTrackMutationsPreserveConditionalConflict(t *testing.T) {
	now := time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC)
	testCases := []struct {
		name   string
		mutate func(*PlaylistService) error
	}{
		{
			name: "add",
			mutate: func(service *PlaylistService) error {
				_, err := service.AddTrack(context.Background(), "p1", domain.TrackSnapshot{VideoID: "second", Title: "Second", Channel: "Band B", DurationSeconds: 180, ThumbnailURL: "https://example.com/second.jpg"})
				return err
			},
		},
		{
			name: "remove",
			mutate: func(service *PlaylistService) error {
				_, err := service.RemoveTrack(context.Background(), "p1", "first")
				return err
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			repo := newFakePlaylistRepository()
			repo.records["p1"] = playlistRecordWithTracks("p1", "Road Trip", now, []domain.TrackSnapshot{{
				VideoID: "first", Title: "First", Channel: "Band A", DurationSeconds: 120, ThumbnailURL: "https://example.com/first.jpg",
			}})
			repo.updateTracksErr = domain.NewPlaylistConflictError("playlist changed", nil)
			service := NewPlaylistService(repo, time.Now)

			err := testCase.mutate(service)
			if err == nil || !domain.IsPlaylistConflictError(err) {
				t.Fatalf("expected conditional conflict, got %v", err)
			}
			if repo.lastExpectedRevision != 1 {
				t.Fatalf("expected conditional update from revision 1, got %d", repo.lastExpectedRevision)
			}
		})
	}
}

func TestPlaylistServiceWrapsUnexpectedRepositoryErrors(t *testing.T) {
	repo := newFakePlaylistRepository()
	repo.getByNameErr = errors.New("db down")
	service := NewPlaylistService(repo, time.Now)

	_, err := service.Create(context.Background(), "best")
	if err == nil || !domain.IsPlaylistUpstreamUnavailableError(err) {
		t.Fatalf("expected upstream error, got %v", err)
	}
}

type fakePlaylistRepository struct {
	records              map[string]domain.PlaylistRecord
	existingByName       map[string]domain.PlaylistRecord
	listResult           []domain.PlaylistRecord
	lastSort             domain.PlaylistSort
	created              domain.PlaylistRecord
	updated              domain.PlaylistRecord
	lastExpectedRevision int64
	updateCalled         bool
	createErr            error
	listErr              error
	getByIDErr           error
	getByNameErr         error
	updateTracksErr      error
}

func newFakePlaylistRepository() *fakePlaylistRepository {
	return &fakePlaylistRepository{
		records:        make(map[string]domain.PlaylistRecord),
		existingByName: make(map[string]domain.PlaylistRecord),
	}
}

func (f *fakePlaylistRepository) Create(_ context.Context, playlist domain.PlaylistRecord) (domain.PlaylistRecord, error) {
	if f.createErr != nil {
		return domain.PlaylistRecord{}, f.createErr
	}
	f.created = playlist
	f.records[playlist.ID] = playlist
	f.existingByName[playlist.NameNormalized] = playlist
	return playlist, nil
}

func (f *fakePlaylistRepository) List(_ context.Context, sort domain.PlaylistSort) ([]domain.PlaylistRecord, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	f.lastSort = sort
	return append([]domain.PlaylistRecord(nil), f.listResult...), nil
}

func (f *fakePlaylistRepository) GetByID(_ context.Context, id string) (domain.PlaylistRecord, bool, error) {
	if f.getByIDErr != nil {
		return domain.PlaylistRecord{}, false, f.getByIDErr
	}
	playlist, ok := f.records[id]
	return playlist, ok, nil
}

func (f *fakePlaylistRepository) GetByNormalizedName(_ context.Context, normalized string) (domain.PlaylistRecord, bool, error) {
	if f.getByNameErr != nil {
		return domain.PlaylistRecord{}, false, f.getByNameErr
	}
	playlist, ok := f.existingByName[normalized]
	return playlist, ok, nil
}

func (f *fakePlaylistRepository) UpdateTracks(_ context.Context, playlistID string, tracks []domain.TrackSnapshot, expectedRevision int64, updatedAt time.Time) (domain.PlaylistRecord, error) {
	f.updateCalled = true
	f.lastExpectedRevision = expectedRevision
	if f.updateTracksErr != nil {
		return domain.PlaylistRecord{}, f.updateTracksErr
	}
	playlist, found := f.records[playlistID]
	if !found {
		return domain.PlaylistRecord{}, domain.NewPlaylistNotFoundError("playlist not found", nil)
	}
	if playlist.Revision != expectedRevision {
		return domain.PlaylistRecord{}, domain.NewPlaylistConflictError("playlist changed", nil)
	}
	playlist.Tracks = append([]domain.TrackSnapshot(nil), tracks...)
	playlist.Revision++
	playlist.UpdatedAt = updatedAt
	f.updated = playlist
	f.records[playlist.ID] = playlist
	return playlist, nil
}

func playlistRecord(id, name string, now time.Time) domain.PlaylistRecord {
	nameInfo, err := domain.NormalizePlaylistName(name)
	if err != nil {
		panic(err)
	}
	return domain.PlaylistRecord{
		ID:             id,
		Name:           nameInfo,
		NameNormalized: nameInfo.Normalized,
		Revision:       1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func withRevision(record domain.PlaylistRecord, revision int64) domain.PlaylistRecord {
	record.Revision = revision
	return record
}

func playlistRecordWithTracks(id, name string, now time.Time, tracks []domain.TrackSnapshot) domain.PlaylistRecord {
	record := playlistRecord(id, name, now)
	record.Tracks = append([]domain.TrackSnapshot(nil), tracks...)
	return record
}

var _ PlaylistRepository = (*fakePlaylistRepository)(nil)
