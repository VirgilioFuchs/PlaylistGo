package domain

import "testing"

func TestThumbnailVariantsBestURLUsesQualityFallback(t *testing.T) {
	variants := &ThumbnailVariants{
		Default:  &Thumbnail{URL: "https://default"},
		High:     &Thumbnail{URL: "https://high"},
		Standard: &Thumbnail{URL: "https://standard"},
	}

	if got := variants.BestURL(); got != "https://standard" {
		t.Fatalf("expected standard thumbnail, got %q", got)
	}
}

func TestTrackSnapshotNormalizesThumbnailFallback(t *testing.T) {
	track := TrackSnapshot{
		VideoID:         "video",
		Title:           "Song",
		Channel:         "Artist",
		DurationSeconds: 60,
		Thumbnails: &ThumbnailVariants{
			High: &Thumbnail{URL: " https://high ", Width: 480, Height: 360},
		},
	}

	normalized := track.NormalizeForPlaylist()
	if normalized.ThumbnailURL != "https://high" {
		t.Fatalf("expected variant fallback URL, got %q", normalized.ThumbnailURL)
	}
	if normalized.Thumbnails == nil || normalized.Thumbnails.High == nil || normalized.Thumbnails.High.URL != "https://high" {
		t.Fatalf("expected normalized variants, got %+v", normalized.Thumbnails)
	}
}
