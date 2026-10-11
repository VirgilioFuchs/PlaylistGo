package domain

import (
	"errors"
	"strings"
	"time"
)

type PlaylistSort string

const (
	PlaylistSortRecent  PlaylistSort = "recent"
	PlaylistSortName    PlaylistSort = "name"
	PlaylistSortUpdated PlaylistSort = "updated"
)

type PlaylistName struct {
	Display    string
	Normalized string
}

type Playlist struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	TrackCount      int             `json:"trackCount"`
	DurationSeconds int             `json:"durationSeconds"`
	Tracks          []TrackSnapshot `json:"tracks,omitempty"`
	Revision        int64           `json:"revision"`
	CreatedAt       string          `json:"createdAt"`
	UpdatedAt       string          `json:"updatedAt"`
}

type PlaylistRecord struct {
	ID             string
	Name           PlaylistName
	NameNormalized string
	Tracks         []TrackSnapshot
	Revision       int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type PlaylistErrorKind string

const (
	PlaylistErrorValidation          PlaylistErrorKind = "validation"
	PlaylistErrorConflict            PlaylistErrorKind = "conflict"
	PlaylistErrorNotFound            PlaylistErrorKind = "not_found"
	PlaylistErrorUpstreamUnavailable PlaylistErrorKind = "upstream_unavailable"
)

type PlaylistError struct {
	Kind    PlaylistErrorKind
	Message string
	Err     error
}

func (e *PlaylistError) Error() string {
	return e.Message
}

func (e *PlaylistError) Unwrap() error {
	return e.Err
}

func (e *PlaylistError) PublicCode() string {
	switch e.Kind {
	case PlaylistErrorValidation:
		return "validation_error"
	case PlaylistErrorConflict:
		return "conflict"
	case PlaylistErrorNotFound:
		return "not_found"
	default:
		return "upstream_unavailable"
	}
}

func NewPlaylistValidationError(message string) *PlaylistError {
	return &PlaylistError{Kind: PlaylistErrorValidation, Message: message}
}

func NewPlaylistConflictError(message string, err error) *PlaylistError {
	return &PlaylistError{Kind: PlaylistErrorConflict, Message: message, Err: err}
}

func NewPlaylistNotFoundError(message string, err error) *PlaylistError {
	return &PlaylistError{Kind: PlaylistErrorNotFound, Message: message, Err: err}
}

func NewPlaylistUpstreamUnavailableError(message string, err error) *PlaylistError {
	return &PlaylistError{Kind: PlaylistErrorUpstreamUnavailable, Message: message, Err: err}
}

func NormalizePlaylistName(raw string) (PlaylistName, error) {
	display := strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
	if display == "" {
		return PlaylistName{}, NewPlaylistValidationError("Informe um nome para a playlist.")
	}

	return PlaylistName{
		Display:    display,
		Normalized: strings.ToLower(display),
	}, nil
}

func NormalizePlaylistSort(raw string) (PlaylistSort, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", string(PlaylistSortRecent):
		return PlaylistSortRecent, nil
	case string(PlaylistSortName):
		return PlaylistSortName, nil
	case string(PlaylistSortUpdated):
		return PlaylistSortUpdated, nil
	default:
		return "", NewPlaylistValidationError("Ordem da playlist invalida.")
	}
}

func (r PlaylistRecord) TrackCount() int {
	return len(r.Tracks)
}

func (r PlaylistRecord) DurationSeconds() int {
	total := 0
	for _, track := range r.Tracks {
		if track.DurationSeconds > 0 {
			total += track.DurationSeconds
		}
	}
	return total
}

func (r PlaylistRecord) ToPlaylist(includeTracks bool) Playlist {
	playlist := Playlist{
		ID:              r.ID,
		Name:            r.Name.Display,
		TrackCount:      r.TrackCount(),
		DurationSeconds: r.DurationSeconds(),
		Revision:        r.Revision,
		CreatedAt:       r.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:       r.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if includeTracks {
		playlist.Tracks = cloneTrackSnapshots(r.Tracks)
	}
	return playlist
}

func cloneTrackSnapshots(tracks []TrackSnapshot) []TrackSnapshot {
	if len(tracks) == 0 {
		return nil
	}
	cloned := make([]TrackSnapshot, len(tracks))
	copy(cloned, tracks)
	return cloned
}

func IsPlaylistValidationError(err error) bool {
	var playlistErr *PlaylistError
	return errors.As(err, &playlistErr) && playlistErr.Kind == PlaylistErrorValidation
}

func IsPlaylistConflictError(err error) bool {
	var playlistErr *PlaylistError
	return errors.As(err, &playlistErr) && playlistErr.Kind == PlaylistErrorConflict
}

func IsPlaylistNotFoundError(err error) bool {
	var playlistErr *PlaylistError
	return errors.As(err, &playlistErr) && playlistErr.Kind == PlaylistErrorNotFound
}

func IsPlaylistUpstreamUnavailableError(err error) bool {
	var playlistErr *PlaylistError
	return errors.As(err, &playlistErr) && playlistErr.Kind == PlaylistErrorUpstreamUnavailable
}
