package youtube

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"musicgo/internal/domain"

	"golang.org/x/oauth2"
)

func TestClientSearchTracksBuildsRequestsAndMapsResponse(t *testing.T) {
	var requestedPaths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPaths = append(requestedPaths, r.URL.Path+"?"+r.URL.RawQuery)
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("expected bearer token, got %q", got)
		}
		switch r.URL.Path {
		case "/search":
			if got := r.URL.Query().Get("maxResults"); got != "15" {
				t.Fatalf("expected maxResults=15, got %q", got)
			}
			_, _ = w.Write([]byte(`{"items":[{"id":{"videoId":"abc"},"snippet":{"title":"Song","channelTitle":"Artist","thumbnails":{"default":{"url":"https://search-default","width":120,"height":90},"high":{"url":"https://search-high","width":480,"height":360}}}}]}`))
		case "/videos":
			if got := r.URL.Query().Get("id"); got != "abc" {
				t.Fatalf("expected videos.list id=abc, got %q", got)
			}
			if got := r.URL.Query().Get("part"); got != "snippet,contentDetails" {
				t.Fatalf("expected videos.list snippet and contentDetails, got %q", got)
			}
			_, _ = w.Write([]byte(`{"items":[{"id":"abc","snippet":{"thumbnails":{"maxres":{"url":"https://video-maxres","width":1280,"height":720}}},"contentDetails":{"duration":"PT3M2S"}}]}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewAuthenticatedClient(staticTokenSource{token: &oauth2.Token{AccessToken: "test-token"}}, &http.Client{Timeout: 2 * time.Second})
	client.baseURL = server.URL

	tracks, err := client.SearchTracks(context.Background(), "The Cure")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tracks) != 1 {
		t.Fatalf("expected 1 track, got %d", len(tracks))
	}
	if tracks[0].DurationSeconds != 182 {
		t.Fatalf("expected duration 182, got %d", tracks[0].DurationSeconds)
	}
	if tracks[0].ThumbnailURL != "https://video-maxres" {
		t.Fatalf("expected video maxres fallback URL, got %q", tracks[0].ThumbnailURL)
	}
	if tracks[0].Thumbnails == nil || tracks[0].Thumbnails.High == nil || tracks[0].Thumbnails.MaxRes == nil {
		t.Fatalf("expected merged thumbnail variants to be retained, got %+v", tracks[0].Thumbnails)
	}
	if len(requestedPaths) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(requestedPaths))
	}
}

func TestClientReturnsQuotaLimitedError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"errors":[{"reason":"quotaExceeded"}],"message":"quota"}}`))
	}))
	defer server.Close()

	client := NewAuthenticatedClient(staticTokenSource{token: &oauth2.Token{AccessToken: "test-token"}}, &http.Client{Timeout: 2 * time.Second})
	client.baseURL = server.URL

	_, err := client.SearchTracks(context.Background(), "bjork")
	if err == nil {
		t.Fatal("expected error")
	}
	if !domain.IsQuotaLimited(err) {
		t.Fatalf("expected quota error, got %v", err)
	}
}

func TestClientSanitizesInvalidPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search":
			_, _ = w.Write([]byte(`{"items":[{"id":{"videoId":"abc"},"snippet":{"title":"Song","channelTitle":"Artist","thumbnails":{"default":{"url":"https://img"}}}}]}`))
		case "/videos":
			_, _ = w.Write([]byte(`{"items":[{"id":"abc","contentDetails":{"duration":"invalid"}}]}`))
		}
	}))
	defer server.Close()

	client := NewAuthenticatedClient(staticTokenSource{token: &oauth2.Token{AccessToken: "test-token"}}, &http.Client{Timeout: 2 * time.Second})
	client.baseURL = server.URL

	_, err := client.SearchTracks(context.Background(), "bjork")
	if err == nil {
		t.Fatal("expected error")
	}
	var searchErr *domain.SearchError
	if !errors.As(err, &searchErr) || searchErr.Kind != domain.SearchErrorUpstreamInvalid {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(err.Error(), "invalid ISO") {
		t.Fatalf("expected sanitized error, got %v", err)
	}
}

func TestParseISODuration(t *testing.T) {
	seconds, err := parseISODuration("PT1H2M3S")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if seconds != 3723 {
		t.Fatalf("expected 3723, got %d", seconds)
	}
}

type staticTokenSource struct {
	token *oauth2.Token
	err   error
}

func (s staticTokenSource) Token() (*oauth2.Token, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.token, nil
}
