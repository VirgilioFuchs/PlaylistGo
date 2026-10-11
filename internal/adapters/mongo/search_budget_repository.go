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

type SearchBudgetRepository struct {
	collection *mongodriver.Collection
	location   *time.Location
}

func NewSearchBudgetRepository(db *mongodriver.Database) (*SearchBudgetRepository, error) {
	location, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		return nil, fmt.Errorf("load pacific timezone: %w", err)
	}

	return &SearchBudgetRepository{
		collection: db.Collection("search_budget"),
		location:   location,
	}, nil
}

func (r *SearchBudgetRepository) EnsureIndexes(ctx context.Context) error {
	model := mongodriver.IndexModel{
		Keys:    bson.D{{Key: "window_key", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	if _, err := r.collection.Indexes().CreateOne(ctx, model); err != nil {
		return fmt.Errorf("create search budget index: %w", err)
	}
	return nil
}

func (r *SearchBudgetRepository) CurrentWindow(ctx context.Context, now time.Time, dailyLimit int) (domain.SearchBudgetWindow, error) {
	windowKey, resetAt := r.windowFor(now)

	var doc searchBudgetDocument
	err := r.collection.FindOne(ctx, bson.D{{Key: "window_key", Value: windowKey}}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongodriver.ErrNoDocuments) {
			return domain.SearchBudgetWindow{WindowKey: windowKey, DailyLimit: dailyLimit, ResetAt: resetAt}, nil
		}
		return domain.SearchBudgetWindow{}, fmt.Errorf("find search budget: %w", err)
	}

	return domain.SearchBudgetWindow{
		WindowKey:  doc.WindowKey,
		DailyLimit: dailyLimit,
		Used:       doc.Used,
		ResetAt:    doc.ResetAt,
	}, nil
}

func (r *SearchBudgetRepository) Consume(ctx context.Context, now time.Time, dailyLimit int) (domain.SearchBudgetWindow, bool, error) {
	windowKey, resetAt := r.windowFor(now)
	filter := bson.D{
		{Key: "window_key", Value: windowKey},
		{Key: "used", Value: bson.D{{Key: "$lt", Value: dailyLimit}}},
	}
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "daily_limit", Value: dailyLimit},
			{Key: "reset_at", Value: resetAt},
			{Key: "updated_at", Value: now},
		}},
		{Key: "$setOnInsert", Value: bson.D{
			{Key: "window_key", Value: windowKey},
			{Key: "created_at", Value: now},
		}},
		{Key: "$inc", Value: bson.D{{Key: "used", Value: 1}}},
	}

	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var doc searchBudgetDocument
	err := r.collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&doc)
	if err == nil {
		return domain.SearchBudgetWindow{
			WindowKey:  doc.WindowKey,
			DailyLimit: dailyLimit,
			Used:       doc.Used,
			ResetAt:    doc.ResetAt,
		}, true, nil
	}

	if errors.Is(err, mongodriver.ErrNoDocuments) {
		window, currentErr := r.CurrentWindow(ctx, now, dailyLimit)
		if currentErr != nil {
			return domain.SearchBudgetWindow{}, false, currentErr
		}
		return window, false, nil
	}

	return domain.SearchBudgetWindow{}, false, fmt.Errorf("consume search budget: %w", err)
}

func (r *SearchBudgetRepository) windowFor(now time.Time) (string, time.Time) {
	localNow := now.In(r.location)
	resetAt := time.Date(localNow.Year(), localNow.Month(), localNow.Day()+1, 0, 0, 0, 0, r.location)
	windowStart := resetAt.Add(-24 * time.Hour)
	return windowStart.Format("2006-01-02"), resetAt.UTC()
}

type searchBudgetDocument struct {
	WindowKey string    `bson:"window_key"`
	Used      int       `bson:"used"`
	ResetAt   time.Time `bson:"reset_at"`
}
