package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"musicgo/internal/domain"
)

type PlaylistRepository interface {
	Create(ctx context.Context, playlist domain.PlaylistRecord) (domain.PlaylistRecord, error)
	List(ctx context.Context, sort domain.PlaylistSort) ([]domain.PlaylistRecord, error)
	GetByID(ctx context.Context, id string) (domain.PlaylistRecord, bool, error)
	GetByNormalizedName(ctx context.Context, normalized string) (domain.PlaylistRecord, bool, error)
	UpdateTracks(ctx context.Context, playlistID string, tracks []domain.TrackSnapshot, expectedRevision int64, updatedAt time.Time) (domain.PlaylistRecord, error)
}

type PlaylistService struct {
	repo  PlaylistRepository
	now   func() time.Time
	newID func() string
}

func NewPlaylistService(repo PlaylistRepository, now func() time.Time) *PlaylistService {
	if now == nil {
		now = time.Now
	}

	return &PlaylistService{
		repo:  repo,
		now:   now,
		newID: newPlaylistID,
	}
}

func (s *PlaylistService) Create(ctx context.Context, rawName string) (domain.Playlist, error) {
	name, err := domain.NormalizePlaylistName(rawName)
	if err != nil {
		return domain.Playlist{}, err
	}

	if _, found, err := s.repo.GetByNormalizedName(ctx, name.Normalized); err != nil {
		return domain.Playlist{}, s.asUnavailableError("Servico de playlists indisponivel no momento.", err)
	} else if found {
		return domain.Playlist{}, domain.NewPlaylistConflictError("Ja existe uma playlist com esse nome.", nil)
	}

	now := s.now().UTC()
	record := domain.PlaylistRecord{
		ID:             s.newID(),
		Name:           name,
		NameNormalized: name.Normalized,
		Revision:       1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	created, err := s.repo.Create(ctx, record)
	if err != nil {
		return domain.Playlist{}, s.wrapPlaylistError("Servico de playlists indisponivel no momento.", err)
	}

	return created.ToPlaylist(false), nil
}

func (s *PlaylistService) List(ctx context.Context, sort domain.PlaylistSort) ([]domain.Playlist, error) {
	records, err := s.repo.List(ctx, sort)
	if err != nil {
		return nil, s.asUnavailableError("Servico de playlists indisponivel no momento.", err)
	}

	playlists := make([]domain.Playlist, 0, len(records))
	for _, record := range records {
		playlists = append(playlists, record.ToPlaylist(false))
	}
	return playlists, nil
}

func (s *PlaylistService) AddTrack(ctx context.Context, playlistID string, snapshot domain.TrackSnapshot) (domain.Playlist, error) {
	playlistID = strings.TrimSpace(playlistID)
	if playlistID == "" {
		return domain.Playlist{}, domain.NewPlaylistValidationError("Informe uma playlist valida.")
	}

	normalizedTrack := snapshot.NormalizeForPlaylist()
	if err := normalizedTrack.ValidateForPlaylist(); err != nil {
		return domain.Playlist{}, domain.NewPlaylistValidationError(err.Error())
	}

	record, found, err := s.repo.GetByID(ctx, playlistID)
	if err != nil {
		return domain.Playlist{}, s.asUnavailableError("Servico de playlists indisponivel no momento.", err)
	}
	if !found {
		return domain.Playlist{}, domain.NewPlaylistNotFoundError("Playlist nao encontrada.", nil)
	}

	now := s.now().UTC()
	for _, existing := range record.Tracks {
		if existing.VideoID == normalizedTrack.VideoID {
			return domain.Playlist{}, domain.NewPlaylistConflictError("Essa faixa ja existe nesta playlist.", nil)
		}
	}

	normalizedTrack.AddedAt = now.Format(time.RFC3339)
	tracks := append([]domain.TrackSnapshot(nil), record.Tracks...)
	tracks = append(tracks, normalizedTrack)

	updated, err := s.repo.UpdateTracks(ctx, record.ID, tracks, record.Revision, now)
	if err != nil {
		return domain.Playlist{}, s.wrapPlaylistError("Servico de playlists indisponivel no momento.", err)
	}

	return updated.ToPlaylist(true), nil
}

func (s *PlaylistService) RemoveTrack(ctx context.Context, playlistID, videoID string) (domain.Playlist, error) {
	playlistID = strings.TrimSpace(playlistID)
	videoID = strings.TrimSpace(videoID)
	if playlistID == "" || videoID == "" {
		return domain.Playlist{}, domain.NewPlaylistValidationError("Informe uma playlist e uma faixa validas.")
	}

	record, found, err := s.repo.GetByID(ctx, playlistID)
	if err != nil {
		return domain.Playlist{}, s.asUnavailableError("Servico de playlists indisponivel no momento.", err)
	}
	if !found {
		return domain.Playlist{}, domain.NewPlaylistNotFoundError("Playlist nao encontrada.", nil)
	}

	index := -1
	for i, track := range record.Tracks {
		if track.VideoID == videoID {
			index = i
			break
		}
	}
	if index < 0 {
		return domain.Playlist{}, domain.NewPlaylistNotFoundError("Faixa nao encontrada na playlist.", nil)
	}

	now := s.now().UTC()
	tracks := make([]domain.TrackSnapshot, 0, len(record.Tracks)-1)
	tracks = append(tracks, record.Tracks[:index]...)
	tracks = append(tracks, record.Tracks[index+1:]...)

	updated, err := s.repo.UpdateTracks(ctx, record.ID, tracks, record.Revision, now)
	if err != nil {
		return domain.Playlist{}, s.wrapPlaylistError("Servico de playlists indisponivel no momento.", err)
	}

	return updated.ToPlaylist(true), nil
}

func (s *PlaylistService) ReorderTracks(ctx context.Context, playlistID string, trackIDs []string, expectedRevision int64) (domain.Playlist, error) {
	playlistID = strings.TrimSpace(playlistID)
	if playlistID == "" {
		return domain.Playlist{}, domain.NewPlaylistValidationError("Informe uma playlist valida.")
	}
	if expectedRevision < 0 {
		return domain.Playlist{}, domain.NewPlaylistValidationError("Informe uma revisao valida da playlist.")
	}

	record, found, err := s.repo.GetByID(ctx, playlistID)
	if err != nil {
		return domain.Playlist{}, s.asUnavailableError("Servico de playlists indisponivel no momento.", err)
	}
	if !found {
		return domain.Playlist{}, domain.NewPlaylistNotFoundError("Playlist nao encontrada.", nil)
	}
	if record.Revision != expectedRevision {
		return domain.Playlist{}, domain.NewPlaylistConflictError("A playlist foi alterada. Atualize e tente novamente.", nil)
	}
	if len(trackIDs) != len(record.Tracks) {
		return domain.Playlist{}, domain.NewPlaylistValidationError("A ordem deve conter exatamente as faixas atuais da playlist.")
	}

	tracksByVideoID := make(map[string]domain.TrackSnapshot, len(record.Tracks))
	for _, track := range record.Tracks {
		videoID := strings.TrimSpace(track.VideoID)
		if videoID == "" || tracksByVideoID[videoID].VideoID != "" {
			return domain.Playlist{}, domain.NewPlaylistConflictError("A playlist contem faixas invalidas para reordenacao.", nil)
		}
		tracksByVideoID[videoID] = track
	}

	reordered := make([]domain.TrackSnapshot, 0, len(trackIDs))
	seen := make(map[string]struct{}, len(trackIDs))
	for _, rawVideoID := range trackIDs {
		videoID := strings.TrimSpace(rawVideoID)
		if videoID == "" {
			return domain.Playlist{}, domain.NewPlaylistValidationError("A ordem nao pode conter faixas vazias.")
		}
		if _, duplicate := seen[videoID]; duplicate {
			return domain.Playlist{}, domain.NewPlaylistValidationError("A ordem nao pode repetir faixas.")
		}
		track, exists := tracksByVideoID[videoID]
		if !exists {
			return domain.Playlist{}, domain.NewPlaylistValidationError("A ordem deve conter exatamente as faixas atuais da playlist.")
		}
		seen[videoID] = struct{}{}
		reordered = append(reordered, track)
	}
	if len(seen) != len(tracksByVideoID) {
		return domain.Playlist{}, domain.NewPlaylistValidationError("A ordem deve conter exatamente as faixas atuais da playlist.")
	}

	updated, err := s.repo.UpdateTracks(ctx, record.ID, reordered, record.Revision, s.now().UTC())
	if err != nil {
		return domain.Playlist{}, s.wrapPlaylistError("Servico de playlists indisponivel no momento.", err)
	}

	return updated.ToPlaylist(true), nil
}

func (s *PlaylistService) asUnavailableError(message string, err error) error {
	var playlistErr *domain.PlaylistError
	if errors.As(err, &playlistErr) {
		return err
	}
	return domain.NewPlaylistUpstreamUnavailableError(message, err)
}

func (s *PlaylistService) wrapPlaylistError(message string, err error) error {
	var playlistErr *domain.PlaylistError
	if errors.As(err, &playlistErr) {
		return err
	}
	return domain.NewPlaylistUpstreamUnavailableError(message, err)
}

func newPlaylistID() string {
	var seed [12]byte
	if _, err := rand.Read(seed[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(seed[:])
}
