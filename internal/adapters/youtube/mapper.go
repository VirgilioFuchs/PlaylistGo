package youtube

import (
	"fmt"
	"regexp"
	"strconv"

	"musicgo/internal/domain"
)

var durationPattern = regexp.MustCompile(`^PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?$`)

func mapTracks(searchItems []searchItem, videoItems []videoItem) ([]domain.TrackSnapshot, error) {
	videoByID := make(map[string]videoItem, len(videoItems))
	for _, item := range videoItems {
		videoByID[item.ID] = item
	}

	tracks := make([]domain.TrackSnapshot, 0, len(searchItems))
	for _, item := range searchItems {
		videoID := item.ID.VideoID
		if videoID == "" {
			return nil, fmt.Errorf("missing video id in search response")
		}
		videoData, ok := videoByID[videoID]
		if !ok {
			return nil, fmt.Errorf("missing videos.list data for %s", videoID)
		}
		durationSeconds, err := parseISODuration(videoData.ContentDetails.Duration)
		if err != nil {
			return nil, fmt.Errorf("parse duration for %s: %w", videoID, err)
		}
		thumbnails := mergeThumbnailVariants(videoData.Snippet.Thumbnails, item.Snippet.Thumbnails)

		tracks = append(tracks, domain.TrackSnapshot{
			VideoID:         videoID,
			Title:           item.Snippet.Title,
			Channel:         item.Snippet.ChannelTitle,
			DurationSeconds: durationSeconds,
			ThumbnailURL:    thumbnails.BestURL(),
			Thumbnails:      thumbnails,
		})
	}

	return tracks, nil
}

// videos.list has the canonical thumbnail variants for a video. search.list
// remains a fallback because it is already part of the search flow and keeps
// incomplete upstream payloads renderable.
func mergeThumbnailVariants(preferred, fallback youtubeThumbnailVariants) *domain.ThumbnailVariants {
	preferredVariants := preferred.toDomain()
	fallbackVariants := fallback.toDomain()
	if preferredVariants == nil {
		return fallbackVariants
	}
	if fallbackVariants == nil {
		return preferredVariants
	}

	return (&domain.ThumbnailVariants{
		Default:  preferredThumbnail(preferredVariants.Default, fallbackVariants.Default),
		Medium:   preferredThumbnail(preferredVariants.Medium, fallbackVariants.Medium),
		High:     preferredThumbnail(preferredVariants.High, fallbackVariants.High),
		Standard: preferredThumbnail(preferredVariants.Standard, fallbackVariants.Standard),
		MaxRes:   preferredThumbnail(preferredVariants.MaxRes, fallbackVariants.MaxRes),
	}).Normalize()
}

func preferredThumbnail(primary, fallback *domain.Thumbnail) *domain.Thumbnail {
	if primary != nil {
		return primary
	}
	return fallback
}

func (t youtubeThumbnailVariants) toDomain() *domain.ThumbnailVariants {
	return (&domain.ThumbnailVariants{
		Default:  t.Default.toDomain(),
		Medium:   t.Medium.toDomain(),
		High:     t.High.toDomain(),
		Standard: t.Standard.toDomain(),
		MaxRes:   t.MaxRes.toDomain(),
	}).Normalize()
}

func (t *youtubeThumbnail) toDomain() *domain.Thumbnail {
	if t == nil {
		return nil
	}
	return &domain.Thumbnail{URL: t.URL, Width: t.Width, Height: t.Height}
}

func parseISODuration(value string) (int, error) {
	matches := durationPattern.FindStringSubmatch(value)
	if matches == nil {
		return 0, fmt.Errorf("invalid ISO 8601 duration: %s", value)
	}

	hours, err := parseDurationPart(matches[1])
	if err != nil {
		return 0, err
	}
	minutes, err := parseDurationPart(matches[2])
	if err != nil {
		return 0, err
	}
	seconds, err := parseDurationPart(matches[3])
	if err != nil {
		return 0, err
	}

	return hours*3600 + minutes*60 + seconds, nil
}

func parseDurationPart(value string) (int, error) {
	if value == "" {
		return 0, nil
	}
	return strconv.Atoi(value)
}
