package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateRequiresMongoURI(t *testing.T) {
	cfg := Config{Environment: "development", Port: "8080", YouTubeClientID: "id", YouTubeClientSecret: "secret", YouTubeTokenPath: "token.json", YouTubeSearchDailyBudget: 100}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestValidateRequiresYouTubeCredentials(t *testing.T) {
	cfg := Config{
		Environment:              "development",
		Port:                     "8080",
		MongoURI:                 "mongodb://localhost:27017/musicgo",
		YouTubeTokenPath:         "token.json",
		YouTubeSearchDailyBudget: 100,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestValidateAcceptsValidConfig(t *testing.T) {
	cfg := Config{
		Environment:              "development",
		Port:                     "8080",
		MongoURI:                 "mongodb://localhost:27017/musicgo",
		YouTubeClientID:          "id",
		YouTubeClientSecret:      "secret",
		YouTubeTokenPath:         "token.json",
		YouTubeSearchDailyBudget: 100,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadUsesDefaultSearchBudget(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("PORT", "8080")
	t.Setenv("MONGODB_URI", "mongodb://localhost:27017/musicgo")
	t.Setenv("CLIENT_ID", "id")
	t.Setenv("CLIENT_SECRET", "secret")
	t.Setenv("FRONTEND_ORIGIN", "http://localhost:5173")
	os.Unsetenv("YOUTUBE_SEARCH_DAILY_BUDGET")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.YouTubeSearchDailyBudget != defaultSearchDailyBudget {
		t.Fatalf("expected default budget %d, got %d", defaultSearchDailyBudget, cfg.YouTubeSearchDailyBudget)
	}
	if cfg.YouTubeTokenPath != defaultYouTubeTokenPath {
		t.Fatalf("expected default token path %q, got %q", defaultYouTubeTokenPath, cfg.YouTubeTokenPath)
	}
}

func TestLoadUsesCustomSearchBudget(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("PORT", "8080")
	t.Setenv("MONGODB_URI", "mongodb://localhost:27017/musicgo")
	t.Setenv("CLIENT_ID", "id")
	t.Setenv("CLIENT_SECRET", "secret")
	t.Setenv("YOUTUBE_SEARCH_DAILY_BUDGET", "42")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.YouTubeSearchDailyBudget != 42 {
		t.Fatalf("expected custom budget 42, got %d", cfg.YouTubeSearchDailyBudget)
	}
}

func TestLoadRejectsInvalidSearchBudget(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("PORT", "8080")
	t.Setenv("MONGODB_URI", "mongodb://localhost:27017/musicgo")
	t.Setenv("CLIENT_ID", "id")
	t.Setenv("CLIENT_SECRET", "secret")
	t.Setenv("YOUTUBE_SEARCH_DAILY_BUDGET", "invalid")

	if _, err := Load(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestLoadReadsDotEnvFile(t *testing.T) {
	dir := t.TempDir()
	dotenv := []byte(
		"APP_ENV=development\n" +
			"PORT=8080\n" +
			"MONGODB_URI=mongodb://localhost:27017/musicgo\n" +
			"CLIENT_ID=id\n" +
			"CLIENT_SECRET=secret\n" +
			"YOUTUBE_TOKEN_PATH=token.json\n",
	)

	if err := os.WriteFile(filepath.Join(dir, ".env"), dotenv, 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	withWorkingDir(t, dir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.MongoURI != "mongodb://localhost:27017/musicgo" {
		t.Fatalf("expected mongo uri from .env, got %q", cfg.MongoURI)
	}
	if cfg.YouTubeClientID != "id" {
		t.Fatalf("expected client id from .env, got %q", cfg.YouTubeClientID)
	}
	if cfg.YouTubeClientSecret != "secret" {
		t.Fatalf("expected client secret from .env, got %q", cfg.YouTubeClientSecret)
	}
	if cfg.YouTubeTokenPath != "token.json" {
		t.Fatalf("expected token path from .env, got %q", cfg.YouTubeTokenPath)
	}
}

func withWorkingDir(t *testing.T, dir string) {
	t.Helper()

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working dir: %v", err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("change working dir: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatalf("restore working dir: %v", err)
		}
	})
}
