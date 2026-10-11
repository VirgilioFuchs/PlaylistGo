package mongo

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	connectTimeout = 5 * time.Second
	databaseName   = "musicgo"
)

func NewClient(ctx context.Context, mongoURI string) (*mongodriver.Client, error) {
	connectCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	client, err := mongodriver.Connect(options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, fmt.Errorf("connect mongo: %w", err)
	}

	if err := client.Ping(connectCtx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("ping mongo: %w", err)
	}

	return client, nil
}

func DatabaseNameFromURI(mongoURI string) (string, error) {
	if strings.TrimSpace(mongoURI) == "" {
		return "", errors.New("mongo uri is required")
	}

	parsed, err := url.Parse(mongoURI)
	if err != nil {
		return "", fmt.Errorf("parse mongo uri: %w", err)
	}

	name := strings.Trim(parsed.Path, "/")
	if name == "" {
		return databaseName, nil
	}
	return name, nil
}
