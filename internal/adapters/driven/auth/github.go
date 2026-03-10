package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"graphiti/internal/domain"
	"graphiti/internal/ports/driven"
)

// GitHubConfig holds GitHub OAuth2 configuration.
type GitHubConfig struct {
	ClientID     string
	ClientSecret string
	Scopes       []string
	AllowedOrgs  []string
}

// GitHubAuthAdapter implements the AuthProvider port using GitHub OAuth2.
type GitHubAuthAdapter struct {
	clientID     string
	clientSecret string
	scopes       []string
	allowedOrgs  []string
	httpClient   *http.Client

	// URLs overridable for testing
	authorizeURL string
	tokenURL     string
	apiURL       string
}

// NewGitHubAuth creates a new GitHubAuthAdapter.
func NewGitHubAuth(cfg GitHubConfig) *GitHubAuthAdapter {
	return &GitHubAuthAdapter{
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		scopes:       cfg.Scopes,
		allowedOrgs:  cfg.AllowedOrgs,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		authorizeURL: "https://github.com/login/oauth/authorize",
		tokenURL:     "https://github.com/login/oauth/access_token",
		apiURL:       "https://api.github.com",
	}
}

// compile-time interface check
var _ driven.AuthProvider = (*GitHubAuthAdapter)(nil)

// GetAuthURL returns a GitHub OAuth2 authorization URL.
func (g *GitHubAuthAdapter) GetAuthURL(state string) string {
	params := url.Values{
		"client_id": {g.clientID},
		"state":     {state},
	}
	if len(g.scopes) > 0 {
		params.Set("scope", strings.Join(g.scopes, " "))
	}
	return g.authorizeURL + "?" + params.Encode()
}

// ExchangeCode exchanges an authorization code for an access token.
func (g *GitHubAuthAdapter) ExchangeCode(ctx context.Context, code string) (*driven.TokenPair, error) {
	form := url.Values{
		"client_id":     {g.clientID},
		"client_secret": {g.clientSecret},
		"code":          {code},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d", resp.StatusCode)
	}

	var result struct {
		AccessToken      string `json:"access_token"`
		RefreshToken     string `json:"refresh_token"`
		TokenType        string `json:"token_type"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding token response: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("github oauth error: %s - %s", result.Error, result.ErrorDescription)
	}

	return &driven.TokenPair{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    time.Now().Add(8 * time.Hour),
	}, nil
}

// GetUserInfo fetches the authenticated user's profile from GitHub.
func (g *GitHubAuthAdapter) GetUserInfo(ctx context.Context, accessToken string) (*domain.UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.apiURL+"/user", nil)
	if err != nil {
		return nil, fmt.Errorf("creating user request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("user info request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github user API returned status %d", resp.StatusCode)
	}

	var ghUser struct {
		ID        int    `json:"id"`
		Login     string `json:"login"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ghUser); err != nil {
		return nil, fmt.Errorf("decoding user response: %w", err)
	}

	// Check org membership if configured
	if len(g.allowedOrgs) > 0 {
		if err := g.checkOrgMembership(ctx, accessToken); err != nil {
			return nil, err
		}
	}

	return &domain.UserInfo{
		ID:       fmt.Sprintf("%d", ghUser.ID),
		Username: ghUser.Login,
		Email:    ghUser.Email,
		Avatar:   ghUser.AvatarURL,
		Provider: "github",
	}, nil
}

// RefreshToken exchanges a refresh token for a new token pair.
func (g *GitHubAuthAdapter) RefreshToken(ctx context.Context, refreshToken string) (*driven.TokenPair, error) {
	form := url.Values{
		"client_id":     {g.clientID},
		"client_secret": {g.clientSecret},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh token failed with status %d", resp.StatusCode)
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		Error        string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding refresh response: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("github refresh error: %s", result.Error)
	}

	return &driven.TokenPair{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    time.Now().Add(8 * time.Hour),
	}, nil
}

// checkOrgMembership verifies the user belongs to at least one allowed org.
func (g *GitHubAuthAdapter) checkOrgMembership(ctx context.Context, accessToken string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.apiURL+"/user/orgs", nil)
	if err != nil {
		return fmt.Errorf("creating orgs request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("orgs request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github orgs API returned status %d", resp.StatusCode)
	}

	var orgs []struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&orgs); err != nil {
		return fmt.Errorf("decoding orgs response: %w", err)
	}

	for _, org := range orgs {
		for _, allowed := range g.allowedOrgs {
			if org.Login == allowed {
				return nil
			}
		}
	}

	return fmt.Errorf("user is not a member of any allowed organization")
}
