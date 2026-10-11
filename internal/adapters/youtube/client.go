package youtube

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"musicgo/internal/domain"

	"golang.org/x/oauth2"
)

const defaultBaseURL = "https://www.googleapis.com/youtube/v3"
const youtubeScope = "https://www.googleapis.com/auth/youtube"

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewAuthenticatedClient(tokenSource oauth2.TokenSource, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}

	return &Client{
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{Timeout: 5 * time.Second, Transport: &oauthTransport{base: baseOrDefaultTransport(httpClient), source: tokenSource}},
	}
}

func NewClientFromOAuth(ctx context.Context, clientID, clientSecret, tokenPath string, httpClient *http.Client) (*Client, error) {
	tokenSource, err := newOAuthTokenSource(ctx, clientID, clientSecret, tokenPath)
	if err != nil {
		return nil, err
	}
	return NewAuthenticatedClient(tokenSource, httpClient), nil
}

func (c *Client) SearchTracks(ctx context.Context, query string) ([]domain.TrackSnapshot, error) {
	searchResponse, err := c.runSearch(ctx, query)
	if err != nil {
		return nil, err
	}

	videoIDs := make([]string, 0, len(searchResponse.Items))
	for _, item := range searchResponse.Items {
		if item.ID.VideoID != "" {
			videoIDs = append(videoIDs, item.ID.VideoID)
		}
	}
	if len(videoIDs) == 0 {
		return []domain.TrackSnapshot{}, nil
	}

	videoResponse, err := c.runVideos(ctx, videoIDs)
	if err != nil {
		return nil, err
	}

	tracks, err := mapTracks(searchResponse.Items, videoResponse.Items)
	if err != nil {
		return nil, domain.NewUpstreamInvalidError("Servico do YouTube indisponivel no momento.", err)
	}
	return tracks, nil
}

func (c *Client) runSearch(ctx context.Context, query string) (searchListResponse, error) {
	values := url.Values{}
	values.Set("part", "snippet")
	values.Set("type", "video")
	values.Set("maxResults", fmt.Sprintf("%d", domain.SearchMaxResults))
	values.Set("q", query)

	var response searchListResponse
	if err := c.doJSON(ctx, "/search", values, &response); err != nil {
		return searchListResponse{}, err
	}
	return response, nil
}

func (c *Client) runVideos(ctx context.Context, videoIDs []string) (videosListResponse, error) {
	values := url.Values{}
	values.Set("part", "snippet,contentDetails")
	values.Set("id", strings.Join(videoIDs, ","))

	var response videosListResponse
	if err := c.doJSON(ctx, "/videos", values, &response); err != nil {
		return videosListResponse{}, err
	}
	return response, nil
}

func (c *Client) doJSON(ctx context.Context, path string, values url.Values, target any) error {
	endpoint := c.baseURL + path + "?" + values.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return domain.NewUpstreamUnavailableError("Servico do YouTube indisponivel no momento.", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if isNetworkError(err) {
			return domain.NewUpstreamUnavailableError("Servico do YouTube indisponivel no momento.", err)
		}
		return domain.NewUpstreamUnavailableError("Servico do YouTube indisponivel no momento.", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		apiErr := decodeAPIError(resp)
		if apiErr.isQuotaLimited() {
			return domain.NewQuotaLimitedError(domain.QuotaLimitedMessage, apiErr.err)
		}
		return domain.NewUpstreamUnavailableError("Servico do YouTube indisponivel no momento.", apiErr.err)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return domain.NewUpstreamInvalidError("Servico do YouTube indisponivel no momento.", err)
	}
	return nil
}

type oauthTransport struct {
	base   http.RoundTripper
	source oauth2.TokenSource
}

func (t *oauthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := t.source.Token()
	if err != nil {
		return nil, err
	}

	cloned := req.Clone(req.Context())
	cloned.Header = cloned.Header.Clone()
	token.SetAuthHeader(cloned)
	return t.base.RoundTrip(cloned)
}

func baseOrDefaultTransport(client *http.Client) http.RoundTripper {
	if client != nil && client.Transport != nil {
		return client.Transport
	}
	return http.DefaultTransport
}

func isNetworkError(err error) bool {
	var netErr net.Error
	return errors.Is(err, context.DeadlineExceeded) || errors.As(err, &netErr)
}

type searchListResponse struct {
	Items []searchItem `json:"items"`
}

type searchItem struct {
	ID struct {
		VideoID string `json:"videoId"`
	} `json:"id"`
	Snippet struct {
		Title        string                   `json:"title"`
		ChannelTitle string                   `json:"channelTitle"`
		Thumbnails   youtubeThumbnailVariants `json:"thumbnails"`
	} `json:"snippet"`
}

type youtubeThumbnail struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type youtubeThumbnailVariants struct {
	Default  *youtubeThumbnail `json:"default"`
	Medium   *youtubeThumbnail `json:"medium"`
	High     *youtubeThumbnail `json:"high"`
	Standard *youtubeThumbnail `json:"standard"`
	MaxRes   *youtubeThumbnail `json:"maxres"`
}

type videosListResponse struct {
	Items []videoItem `json:"items"`
}

type videoItem struct {
	ID      string `json:"id"`
	Snippet struct {
		Thumbnails youtubeThumbnailVariants `json:"thumbnails"`
	} `json:"snippet"`
	ContentDetails struct {
		Duration string `json:"duration"`
	} `json:"contentDetails"`
}

type apiErrorBody struct {
	Error struct {
		Errors []struct {
			Reason string `json:"reason"`
		} `json:"errors"`
		Message string `json:"message"`
	} `json:"error"`
}

type decodedAPIError struct {
	err     error
	reasons []string
}

func (e decodedAPIError) isQuotaLimited() bool {
	for _, reason := range e.reasons {
		switch reason {
		case "quotaExceeded", "dailyLimitExceeded", "rateLimitExceeded":
			return true
		}
	}
	return false
}

func decodeAPIError(resp *http.Response) decodedAPIError {
	var body apiErrorBody
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return decodedAPIError{err: fmt.Errorf("youtube status %d", resp.StatusCode)}
	}

	reasons := make([]string, 0, len(body.Error.Errors))
	for _, item := range body.Error.Errors {
		reasons = append(reasons, item.Reason)
	}

	return decodedAPIError{
		err:     fmt.Errorf("youtube status %d", resp.StatusCode),
		reasons: reasons,
	}
}
