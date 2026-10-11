//go:build integration
// +build integration

package youtube

import (
	"bufio"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"musicgo/internal/domain"
)

func TestLiveSearchTracksAgainstYouTube(t *testing.T) {
	dotenvPath := loadDotEnv(t)
	dotenvDir := filepath.Dir(dotenvPath)

	clientID := strings.TrimSpace(os.Getenv("CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("CLIENT_SECRET"))
	tokenPath := resolvePathRelativeTo(dotenvDir, strings.TrimSpace(os.Getenv("YOUTUBE_TOKEN_PATH")))
	query := strings.TrimSpace(os.Getenv("YOUTUBE_LIVE_QUERY"))
	if query == "" {
		query = "the cure"
	}

	if clientID == "" || clientSecret == "" || tokenPath == "" {
		t.Skip("missing CLIENT_ID, CLIENT_SECRET, or YOUTUBE_TOKEN_PATH")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client, err := NewClientFromOAuth(ctx, clientID, clientSecret, tokenPath, nil)
	if err != nil {
		t.Fatalf("create youtube client: %v", err)
	}

	tracks, err := client.SearchTracks(ctx, query)
	if err != nil {
		var searchErr *domain.SearchError
		if errors.As(err, &searchErr) {
			t.Logf("search error kind=%s cause=%v", searchErr.Kind, searchErr.Err)
		}
		t.Fatalf("live youtube search failed: %v", err)
	}
	if len(tracks) == 0 {
		t.Fatalf("expected search results for %q, got none", query)
	}

	limit := len(tracks)
	if limit > 5 {
		limit = 5
	}
	for i := 0; i < limit; i++ {
		track := tracks[i]
		if track.VideoID == "" || track.Title == "" || track.Channel == "" || track.DurationSeconds <= 0 || track.ThumbnailURL == "" {
			t.Fatalf("result %d missing required fields: %+v", i, track)
		}
		t.Logf("%d. %s | %s | %ds | %s", i+1, track.Title, track.Channel, track.DurationSeconds, track.VideoID)
	}
}

func loadDotEnv(t *testing.T) string {
	t.Helper()

	if envHasRequiredKeys() {
		dotenvPath, err := findDotEnv()
		if err != nil {
			t.Skip("no .env found and required YouTube env vars are missing")
		}
		return dotenvPath
	}

	dotenvPath, err := findDotEnv()
	if err != nil {
		t.Skip("no .env found and required YouTube env vars are missing")
	}

	file, err := os.Open(dotenvPath)
	if err != nil {
		t.Fatalf("open .env: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read .env: %v", err)
	}

	return dotenvPath
}

func resolvePathRelativeTo(baseDir, value string) string {
	if value == "" {
		return value
	}
	if filepath.IsAbs(value) {
		return value
	}
	return filepath.Clean(filepath.Join(baseDir, value))
}

func envHasRequiredKeys() bool {
	return strings.TrimSpace(os.Getenv("CLIENT_ID")) != "" &&
		strings.TrimSpace(os.Getenv("CLIENT_SECRET")) != "" &&
		strings.TrimSpace(os.Getenv("YOUTUBE_TOKEN_PATH")) != ""
}

func findDotEnv() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(dir, ".env")
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", os.ErrNotExist
}
