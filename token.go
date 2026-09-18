package connectivity

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// TokenSource supplies the bearer credential attached to every request.
type TokenSource interface {
	Token(ctx context.Context) (string, error)
}

// StaticToken is a TokenSource wrapping a fixed token string. Suitable for
// tests and short-lived scripts; long-running services should use
// NewClientCredentialsTokenSource so the credential refreshes itself.
type StaticToken string

func (s StaticToken) Token(ctx context.Context) (string, error) { return string(s), nil }

// NewClientCredentialsTokenSource returns a TokenSource that obtains and
// refreshes tokens from an OpenID Connect token endpoint using the
// client-credentials grant. tokenURL is the full token endpoint, e.g.
// "https://auth.example.com/realms/network/protocol/openid-connect/token".
// Tokens are cached and renewed 30 seconds before expiry.
func NewClientCredentialsTokenSource(tokenURL, clientID, clientSecret string) TokenSource {
	return &ccTokenSource{
		tokenURL:     tokenURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: 15 * time.Second},
	}
}

type ccTokenSource struct {
	tokenURL     string
	clientID     string
	clientSecret string
	httpClient   *http.Client

	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

func (s *ccTokenSource) Token(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token != "" && time.Now().Before(s.expiresAt) {
		return s.token, nil
	}

	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {s.clientID},
		"client_secret": {s.clientSecret},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("connectivity: token request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("connectivity: token endpoint returned %d", res.StatusCode)
	}

	var body struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("connectivity: decoding token response: %w", err)
	}
	if body.AccessToken == "" {
		return "", fmt.Errorf("connectivity: token endpoint returned no token")
	}

	s.token = body.AccessToken
	ttl := time.Duration(body.ExpiresIn) * time.Second
	if ttl > 30*time.Second {
		ttl -= 30 * time.Second
	}
	s.expiresAt = time.Now().Add(ttl)
	return s.token, nil
}
