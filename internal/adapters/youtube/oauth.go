package youtube

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type tokenStore struct {
	path string
}

func newOAuthTokenSource(ctx context.Context, clientID, clientSecret, tokenPath string) (oauth2.TokenSource, error) {
	config := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{youtubeScope},
		Endpoint:     google.Endpoint,
	}

	store := tokenStore{path: tokenPath}
	token, err := store.Load()
	if err != nil {
		return nil, err
	}
	if token == nil {
		token, err = authorizeInteractive(ctx, config, store)
		if err != nil {
			return nil, err
		}
	}

	return config.TokenSource(ctx, token), nil
}

func authorizeInteractive(ctx context.Context, config oauth2.Config, store tokenStore) (*oauth2.Token, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("start oauth listener: %w", err)
	}
	defer ln.Close()

	callbackURL := (&url.URL{
		Scheme: "http",
		Host:   ln.Addr().String(),
		Path:   "/oauth2/callback",
	}).String()
	config.RedirectURL = callbackURL

	state := fmt.Sprintf("%d", time.Now().UnixNano())
	result := make(chan oauthResult, 1)

	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth2/callback" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("state") != state {
			http.Error(w, "invalid state", http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			return
		}
		token, err := config.Exchange(r.Context(), code)
		if err != nil {
			result <- oauthResult{err: fmt.Errorf("exchange oauth code: %w", err)}
			return
		}
		result <- oauthResult{token: token}
		_, _ = w.Write([]byte("Autorizacao concluida. Voce pode fechar esta janela."))
	})}

	go func() {
		_ = server.Serve(ln)
	}()

	authURL := config.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
	)
	_ = openBrowser(authURL)
	fmt.Fprintf(os.Stderr, "Authorize YouTube access by opening: %s\n", authURL)

	select {
	case <-ctx.Done():
		_ = server.Shutdown(context.Background())
		return nil, ctx.Err()
	case res := <-result:
		_ = server.Shutdown(context.Background())
		if res.err != nil {
			return nil, res.err
		}
		if err := store.Save(res.token); err != nil {
			return nil, err
		}
		return res.token, nil
	}
}

type oauthResult struct {
	token *oauth2.Token
	err   error
}

func openBrowser(rawURL string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL).Start()
	case "darwin":
		return exec.Command("open", rawURL).Start()
	default:
		return exec.Command("xdg-open", rawURL).Start()
	}
}

func (s tokenStore) Load() (*oauth2.Token, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read oauth token: %w", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("decode oauth token: %w", err)
	}
	return &token, nil
}

func (s tokenStore) Save(token *oauth2.Token) error {
	if token == nil {
		return errors.New("oauth token is required")
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil && filepath.Dir(s.path) != "." {
		return fmt.Errorf("create oauth token directory: %w", err)
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("encode oauth token: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0o600); err != nil {
		return fmt.Errorf("write oauth token: %w", err)
	}
	return nil
}
