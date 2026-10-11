package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	defaultPort              = "8080"
	defaultSearchDailyBudget = 100
	defaultYouTubeTokenPath  = "youtube_token.json"
)

type Config struct {
	Environment              string
	Port                     string
	MongoURI                 string
	YouTubeClientID          string
	YouTubeClientSecret      string
	YouTubeTokenPath         string
	YouTubeSearchDailyBudget int
	FrontendOrigin           string
}

func Load() (Config, error) {
	if err := loadEnvFile(".env"); err != nil {
		return Config{}, err
	}

	dailyBudget, err := parseSearchDailyBudget(getEnv("YOUTUBE_SEARCH_DAILY_BUDGET", strconv.Itoa(defaultSearchDailyBudget)))
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Environment:              strings.TrimSpace(getEnv("APP_ENV", "development")),
		Port:                     strings.TrimSpace(getEnv("PORT", defaultPort)),
		MongoURI:                 strings.TrimSpace(os.Getenv("MONGODB_URI")),
		YouTubeClientID:          strings.TrimSpace(os.Getenv("CLIENT_ID")),
		YouTubeClientSecret:      strings.TrimSpace(os.Getenv("CLIENT_SECRET")),
		YouTubeTokenPath:         getEnvOrDefault("YOUTUBE_TOKEN_PATH", defaultYouTubeTokenPath),
		YouTubeSearchDailyBudget: dailyBudget,
		FrontendOrigin:           strings.TrimSpace(os.Getenv("FRONTEND_ORIGIN")),
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	if c.Port == "" {
		return errors.New("PORT is required")
	}
	if _, err := strconv.Atoi(c.Port); err != nil {
		return fmt.Errorf("PORT must be numeric: %w", err)
	}
	if c.MongoURI == "" {
		return errors.New("MONGODB_URI is required")
	}
	if c.YouTubeClientID == "" {
		return errors.New("CLIENT_ID is required")
	}
	if c.YouTubeClientSecret == "" {
		return errors.New("CLIENT_SECRET is required")
	}
	if c.Environment == "" {
		return errors.New("APP_ENV is required")
	}
	if c.YouTubeTokenPath == "" {
		return errors.New("YOUTUBE_TOKEN_PATH is required")
	}
	if c.YouTubeSearchDailyBudget <= 0 {
		return errors.New("YOUTUBE_SEARCH_DAILY_BUDGET must be greater than zero")
	}
	return nil
}

func parseSearchDailyBudget(value string) (int, error) {
	budget, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("YOUTUBE_SEARCH_DAILY_BUDGET must be numeric: %w", err)
	}
	if budget <= 0 {
		return 0, errors.New("YOUTUBE_SEARCH_DAILY_BUDGET must be greater than zero")
	}
	return budget, nil
}

func getEnv(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	return value
}

func getEnvOrDefault(key, fallback string) string {
	value := strings.TrimSpace(getEnv(key, ""))
	if value == "" {
		return fallback
	}
	return value
}

func loadEnvFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("%s:%d: invalid environment assignment", path, lineNumber)
		}

		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("%s:%d: empty environment key", path, lineNumber)
		}

		if _, exists := os.LookupEnv(key); exists {
			continue
		}

		value = strings.TrimSpace(value)
		if unquoted, ok := trimMatchingQuotes(value); ok {
			value = unquoted
		}

		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("%s:%d: set %s: %w", path, lineNumber, key, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("%s: read error: %w", path, err)
	}

	return nil
}

func trimMatchingQuotes(value string) (string, bool) {
	if len(value) < 2 {
		return value, false
	}

	first := value[0]
	last := value[len(value)-1]
	if first != last {
		return value, false
	}
	if first != '"' && first != '\'' {
		return value, false
	}

	return value[1 : len(value)-1], true
}
