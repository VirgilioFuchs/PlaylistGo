package domain

import (
	"errors"
	"strings"
)

type TrackSnapshot struct {
	VideoID         string             `json:"videoId"`
	Title           string             `json:"title"`
	Channel         string             `json:"channel"`
	DurationSeconds int                `json:"durationSeconds"`
	ThumbnailURL    string             `json:"thumbnailUrl"`
	Thumbnails      *ThumbnailVariants `json:"thumbnails,omitempty"`
	AddedAt         string             `json:"addedAt,omitempty"`
}

// Thumbnail contains one image variant supplied by the upstream video service.
// Width and Height are optional because the YouTube API may omit them.
type Thumbnail struct {
	URL    string `json:"url"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

// ThumbnailVariants keeps the source-provided variants so each client surface
// can select a suitable image without inventing URLs or requesting maxres by
// default. ThumbnailURL remains the compatibility fallback for older clients
// and previously persisted playlist entries.
type ThumbnailVariants struct {
	Default  *Thumbnail `json:"default,omitempty"`
	Medium   *Thumbnail `json:"medium,omitempty"`
	High     *Thumbnail `json:"high,omitempty"`
	Standard *Thumbnail `json:"standard,omitempty"`
	MaxRes   *Thumbnail `json:"maxres,omitempty"`
}

func (t TrackSnapshot) NormalizeForPlaylist() TrackSnapshot {
	thumbnails := t.Thumbnails.Normalize()
	thumbnailURL := strings.TrimSpace(t.ThumbnailURL)
	if thumbnailURL == "" {
		thumbnailURL = thumbnails.BestURL()
	}

	return TrackSnapshot{
		VideoID:         strings.TrimSpace(t.VideoID),
		Title:           strings.TrimSpace(t.Title),
		Channel:         strings.TrimSpace(t.Channel),
		DurationSeconds: t.DurationSeconds,
		ThumbnailURL:    thumbnailURL,
		Thumbnails:      thumbnails,
		AddedAt:         strings.TrimSpace(t.AddedAt),
	}
}

func (t *ThumbnailVariants) Normalize() *ThumbnailVariants {
	if t == nil {
		return nil
	}

	normalized := &ThumbnailVariants{
		Default:  normalizeThumbnail(t.Default),
		Medium:   normalizeThumbnail(t.Medium),
		High:     normalizeThumbnail(t.High),
		Standard: normalizeThumbnail(t.Standard),
		MaxRes:   normalizeThumbnail(t.MaxRes),
	}
	if normalized.Default == nil && normalized.Medium == nil && normalized.High == nil && normalized.Standard == nil && normalized.MaxRes == nil {
		return nil
	}
	return normalized
}

func (t *ThumbnailVariants) BestURL() string {
	if t == nil {
		return ""
	}

	for _, thumbnail := range []*Thumbnail{t.MaxRes, t.Standard, t.High, t.Medium, t.Default} {
		if thumbnail != nil && strings.TrimSpace(thumbnail.URL) != "" {
			return strings.TrimSpace(thumbnail.URL)
		}
	}
	return ""
}

func normalizeThumbnail(thumbnail *Thumbnail) *Thumbnail {
	if thumbnail == nil {
		return nil
	}

	normalized := &Thumbnail{
		URL:    strings.TrimSpace(thumbnail.URL),
		Width:  thumbnail.Width,
		Height: thumbnail.Height,
	}
	if normalized.URL == "" {
		return nil
	}
	return normalized
}

func (t TrackSnapshot) ValidateForPlaylist() error {
	normalized := t.NormalizeForPlaylist()
	if normalized.VideoID == "" {
		return errors.New("videoId is required")
	}
	if normalized.Title == "" {
		return errors.New("title is required")
	}
	if normalized.Channel == "" {
		return errors.New("channel is required")
	}
	if normalized.DurationSeconds <= 0 {
		return errors.New("durationSeconds must be greater than zero")
	}
	if normalized.ThumbnailURL == "" {
		return errors.New("thumbnailUrl is required")
	}
	return nil
}
