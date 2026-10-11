package mongo

import (
	"testing"
	"time"

	"musicgo/internal/domain"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestPlaylistSortSpec(t *testing.T) {
	cases := []struct {
		name string
		sort domain.PlaylistSort
		want string
	}{
		{name: "recent", sort: domain.PlaylistSortRecent, want: "created_at"},
		{name: "name", sort: domain.PlaylistSortName, want: "name_normalized"},
		{name: "updated", sort: domain.PlaylistSortUpdated, want: "updated_at"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spec := playlistSortSpec(tc.sort)
			if len(spec) == 0 || spec[0].Key != tc.want {
				t.Fatalf("unexpected sort spec %+v", spec)
			}
		})
	}
}

func TestPlaylistDocumentRoundTrip(t *testing.T) {
	now := time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC)
	record := domain.PlaylistRecord{
		ID:             "p1",
		Name:           domain.PlaylistName{Display: "Road Trip", Normalized: "road trip"},
		NameNormalized: "road trip",
		Revision:       3,
		Tracks: []domain.TrackSnapshot{{
			VideoID:         "abc123",
			Title:           "Song",
			Channel:         "Band",
			DurationSeconds: 180,
			ThumbnailURL:    "https://example.com/thumb.jpg",
			AddedAt:         now.Format(time.RFC3339),
		}},
		CreatedAt: now,
		UpdatedAt: now,
	}

	doc := newPlaylistDocument(record)
	roundTrip := doc.toRecord()

	if roundTrip.ID != record.ID {
		t.Fatalf("unexpected id %q", roundTrip.ID)
	}
	if roundTrip.Name.Display != record.Name.Display || roundTrip.Name.Normalized != record.Name.Normalized {
		t.Fatalf("unexpected name %+v", roundTrip.Name)
	}
	if len(roundTrip.Tracks) != 1 || roundTrip.Tracks[0].VideoID != "abc123" {
		t.Fatalf("unexpected tracks %+v", roundTrip.Tracks)
	}
	if roundTrip.Revision != record.Revision {
		t.Fatalf("unexpected revision %d", roundTrip.Revision)
	}
	if !roundTrip.CreatedAt.Equal(now) || !roundTrip.UpdatedAt.Equal(now) {
		t.Fatalf("unexpected timestamps %+v", roundTrip)
	}
}

func TestPlaylistTracksUpdateUsesRevisionFilterAndAtomicOperators(t *testing.T) {
	now := time.Date(2026, 6, 23, 10, 5, 0, 0, time.UTC)
	filter := playlistTracksUpdateFilter("p1", 3)
	if len(filter) != 2 || filter[0].Key != "id" || filter[0].Value != "p1" || filter[1].Key != "revision" || filter[1].Value != int64(3) {
		t.Fatalf("expected id and revision filter, got %+v", filter)
	}

	tracks := []domain.TrackSnapshot{{VideoID: "abc123", Title: "Song", Channel: "Band", DurationSeconds: 180, ThumbnailURL: "https://example.com/thumb.jpg"}}
	update := playlistTracksUpdate(tracks, now)
	if len(update) != 2 || update[0].Key != "$set" || update[1].Key != "$inc" {
		t.Fatalf("expected atomic set and increment update, got %+v", update)
	}
	set, ok := update[0].Value.(bson.D)
	if !ok || len(set) != 2 || set[0].Key != "tracks" || set[1].Key != "updated_at" || !set[1].Value.(time.Time).Equal(now) {
		t.Fatalf("expected tracks and timestamp set, got %+v", update[0].Value)
	}
	inc, ok := update[1].Value.(bson.D)
	if !ok || len(inc) != 1 || inc[0].Key != "revision" || inc[0].Value != 1 {
		t.Fatalf("expected revision increment, got %+v", update[1].Value)
	}

	storedTracks := set[0].Value.([]domain.TrackSnapshot)
	tracks[0].Title = "Mutated outside repository"
	if storedTracks[0].Title != "Song" {
		t.Fatalf("expected defensive track clone, got %+v", storedTracks)
	}
}
