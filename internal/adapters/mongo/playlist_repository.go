package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"musicgo/internal/domain"

	"go.mongodb.org/mongo-driver/v2/bson"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type PlaylistRepository struct {
	collection *mongodriver.Collection
}

func NewPlaylistRepository(db *mongodriver.Database) *PlaylistRepository {
	return &PlaylistRepository{collection: db.Collection("playlists")}
}

func (r *PlaylistRepository) EnsureIndexes(ctx context.Context) error {
	models := []mongodriver.IndexModel{
		{
			Keys:    bson.D{{Key: "name_normalized", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "created_at", Value: -1}},
		},
	}

	if _, err := r.collection.Indexes().CreateMany(ctx, models); err != nil {
		return fmt.Errorf("create playlist indexes: %w", err)
	}
	return nil
}

func (r *PlaylistRepository) Create(ctx context.Context, playlist domain.PlaylistRecord) (domain.PlaylistRecord, error) {
	doc := newPlaylistDocument(playlist)
	if _, err := r.collection.InsertOne(ctx, doc); err != nil {
		if mongodriver.IsDuplicateKeyError(err) {
			return domain.PlaylistRecord{}, domain.NewPlaylistConflictError("Ja existe uma playlist com esse nome.", err)
		}
		return domain.PlaylistRecord{}, fmt.Errorf("insert playlist: %w", err)
	}

	return playlist, nil
}

func (r *PlaylistRepository) List(ctx context.Context, sort domain.PlaylistSort) ([]domain.PlaylistRecord, error) {
	opts := options.Find().SetSort(playlistSortSpec(sort))
	cursor, err := r.collection.Find(ctx, bson.D{}, opts)
	if err != nil {
		return nil, fmt.Errorf("list playlists: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []playlistDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode playlists: %w", err)
	}

	records := make([]domain.PlaylistRecord, 0, len(docs))
	for _, doc := range docs {
		records = append(records, doc.toRecord())
	}
	return records, nil
}

func (r *PlaylistRepository) GetByID(ctx context.Context, id string) (domain.PlaylistRecord, bool, error) {
	doc, found, err := r.findOne(ctx, bson.D{{Key: "id", Value: id}})
	if err != nil || !found {
		return domain.PlaylistRecord{}, found, err
	}
	return doc.toRecord(), true, nil
}

func (r *PlaylistRepository) GetByNormalizedName(ctx context.Context, normalized string) (domain.PlaylistRecord, bool, error) {
	doc, found, err := r.findOne(ctx, bson.D{{Key: "name_normalized", Value: normalized}})
	if err != nil || !found {
		return domain.PlaylistRecord{}, found, err
	}
	return doc.toRecord(), true, nil
}

func (r *PlaylistRepository) UpdateTracks(ctx context.Context, playlistID string, tracks []domain.TrackSnapshot, expectedRevision int64, updatedAt time.Time) (domain.PlaylistRecord, error) {
	var doc playlistDocument
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	err := r.collection.FindOneAndUpdate(
		ctx,
		playlistTracksUpdateFilter(playlistID, expectedRevision),
		playlistTracksUpdate(tracks, updatedAt),
		opts,
	).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongodriver.ErrNoDocuments) {
			return domain.PlaylistRecord{}, domain.NewPlaylistConflictError("A playlist foi alterada. Atualize e tente novamente.", nil)
		}
		return domain.PlaylistRecord{}, fmt.Errorf("update playlist tracks: %w", err)
	}

	return doc.toRecord(), nil
}

func playlistTracksUpdateFilter(playlistID string, expectedRevision int64) bson.D {
	return bson.D{{Key: "id", Value: playlistID}, {Key: "revision", Value: expectedRevision}}
}

func playlistTracksUpdate(tracks []domain.TrackSnapshot, updatedAt time.Time) bson.D {
	return bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "tracks", Value: clonePlaylistTrackSnapshots(tracks)},
			{Key: "updated_at", Value: updatedAt},
		}},
		{Key: "$inc", Value: bson.D{{Key: "revision", Value: 1}}},
	}
}

func (r *PlaylistRepository) findOne(ctx context.Context, filter interface{}) (playlistDocument, bool, error) {
	var doc playlistDocument
	if err := r.collection.FindOne(ctx, filter).Decode(&doc); err != nil {
		if errors.Is(err, mongodriver.ErrNoDocuments) {
			return playlistDocument{}, false, nil
		}
		return playlistDocument{}, false, fmt.Errorf("find playlist: %w", err)
	}
	return doc, true, nil
}

func playlistSortSpec(sort domain.PlaylistSort) bson.D {
	switch sort {
	case domain.PlaylistSortName:
		return bson.D{{Key: "name_normalized", Value: 1}, {Key: "created_at", Value: -1}}
	case domain.PlaylistSortUpdated:
		return bson.D{{Key: "updated_at", Value: -1}}
	case domain.PlaylistSortRecent:
		fallthrough
	default:
		return bson.D{{Key: "created_at", Value: -1}}
	}
}

type playlistDocument struct {
	ID             string                 `bson:"id"`
	Name           string                 `bson:"name"`
	NameNormalized string                 `bson:"name_normalized"`
	Tracks         []domain.TrackSnapshot `bson:"tracks"`
	Revision       int64                  `bson:"revision"`
	CreatedAt      time.Time              `bson:"created_at"`
	UpdatedAt      time.Time              `bson:"updated_at"`
}

func newPlaylistDocument(record domain.PlaylistRecord) playlistDocument {
	return playlistDocument{
		ID:             record.ID,
		Name:           record.Name.Display,
		NameNormalized: record.NameNormalized,
		Tracks:         clonePlaylistTrackSnapshots(record.Tracks),
		Revision:       record.Revision,
		CreatedAt:      record.CreatedAt,
		UpdatedAt:      record.UpdatedAt,
	}
}

func (d playlistDocument) toRecord() domain.PlaylistRecord {
	return domain.PlaylistRecord{
		ID:             d.ID,
		Name:           domain.PlaylistName{Display: d.Name, Normalized: d.NameNormalized},
		NameNormalized: d.NameNormalized,
		Tracks:         clonePlaylistTrackSnapshots(d.Tracks),
		Revision:       d.Revision,
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      d.UpdatedAt,
	}
}

func clonePlaylistTrackSnapshots(tracks []domain.TrackSnapshot) []domain.TrackSnapshot {
	if len(tracks) == 0 {
		return []domain.TrackSnapshot{}
	}
	cloned := make([]domain.TrackSnapshot, len(tracks))
	copy(cloned, tracks)
	return cloned
}
