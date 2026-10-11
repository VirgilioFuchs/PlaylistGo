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

type SearchCacheRepository struct {
	collection *mongodriver.Collection
}

func NewSearchCacheRepository(db *mongodriver.Database) *SearchCacheRepository {
	return &SearchCacheRepository{collection: db.Collection("search_cache")}
}

func (r *SearchCacheRepository) EnsureIndexes(ctx context.Context) error {
	models := []mongodriver.IndexModel{
		{
			Keys:    bson.D{{Key: "query_normalized", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0),
		},
	}

	if _, err := r.collection.Indexes().CreateMany(ctx, models); err != nil {
		return fmt.Errorf("create search cache indexes: %w", err)
	}
	return nil
}

func (r *SearchCacheRepository) Get(ctx context.Context, query domain.NormalizedQuery, now time.Time) (domain.SearchCacheEntry, bool, error) {
	filter := bson.D{
		{Key: "query_normalized", Value: string(query)},
		{Key: "expires_at", Value: bson.D{{Key: "$gt", Value: now}}},
	}

	var doc searchCacheDocument
	if err := r.collection.FindOne(ctx, filter).Decode(&doc); err != nil {
		if errors.Is(err, mongodriver.ErrNoDocuments) {
			return domain.SearchCacheEntry{}, false, nil
		}
		return domain.SearchCacheEntry{}, false, fmt.Errorf("find search cache: %w", err)
	}

	return doc.toDomain(), true, nil
}

func (r *SearchCacheRepository) Save(ctx context.Context, entry domain.SearchCacheEntry) error {
	doc := newSearchCacheDocument(entry)
	filter := bson.D{{Key: "query_normalized", Value: doc.QueryNormalized}}
	update := bson.D{{Key: "$set", Value: doc}}

	_, err := r.collection.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("upsert search cache: %w", err)
	}
	return nil
}

type searchCacheDocument struct {
	QueryNormalized string                 `bson:"query_normalized"`
	Results         []domain.TrackSnapshot `bson:"results"`
	CachedAt        time.Time              `bson:"cached_at"`
	ExpiresAt       time.Time              `bson:"expires_at"`
}

func newSearchCacheDocument(entry domain.SearchCacheEntry) searchCacheDocument {
	return searchCacheDocument{
		QueryNormalized: string(entry.Query),
		Results:         entry.Results,
		CachedAt:        entry.CachedAt,
		ExpiresAt:       entry.ExpiresAt,
	}
}

func (d searchCacheDocument) toDomain() domain.SearchCacheEntry {
	return domain.SearchCacheEntry{
		Query:     domain.NormalizedQuery(d.QueryNormalized),
		Results:   d.Results,
		CachedAt:  d.CachedAt,
		ExpiresAt: d.ExpiresAt,
	}
}
