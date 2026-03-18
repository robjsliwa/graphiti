package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"

	"graphiti/internal/domain"
	"graphiti/internal/ports/driven"
	"graphiti/web/templates"
	"graphiti/web/templates/pages"
)

func handleLogin(auth driven.AuthProvider, sessions *SessionStore, branding templates.Branding) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If already authenticated, redirect to dashboard
		if sess, err := sessions.Get(r); err == nil && sess != nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		// For fake auth, auto-redirect to callback
		url := auth.GetAuthURL(generateState())

		// If the auth URL is local (fake auth), just redirect directly
		if len(url) > 0 && url[0] == '/' {
			http.Redirect(w, r, url, http.StatusFound)
			return
		}

		// For real OAuth, show login page with GitHub button
		pages.LoginWithProvider(pages.LoginData{Provider: "github", Branding: branding}).Render(r.Context(), w)
	}
}

func handleCallback(auth driven.AuthProvider, userRepo driven.UserRepository, sessions *SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing code parameter", http.StatusBadRequest)
			return
		}

		token, err := auth.ExchangeCode(r.Context(), code)
		if err != nil {
			slog.Error("auth exchange failed", "error", err)
			http.Error(w, "authentication failed", http.StatusInternalServerError)
			return
		}

		userInfo, err := auth.GetUserInfo(r.Context(), token.AccessToken)
		if err != nil {
			slog.Error("get user info failed", "error", err)
			http.Error(w, "failed to get user info", http.StatusInternalServerError)
			return
		}

		user := &domain.User{
			ID:             userInfo.ID,
			Username:       userInfo.Username,
			Email:          userInfo.Email,
			AvatarURL:      userInfo.Avatar,
			AuthProvider:   userInfo.Provider,
			AuthProviderID: userInfo.ID,
			LastLoginAt:    time.Now(),
		}
		if err := userRepo.Upsert(context.Background(), user); err != nil {
			slog.Error("user upsert failed", "error", err)
		}

		session := &Session{
			UserID:    user.ID,
			Username:  user.Username,
			Email:     user.Email,
			ExpiresAt: time.Now().Add(sessions.maxAge),
		}
		if err := sessions.Set(w, session); err != nil {
			slog.Error("session set failed", "error", err)
			http.Error(w, "failed to create session", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func handleLogout(sessions *SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessions.Clear(w)
		http.Redirect(w, r, "/auth/login", http.StatusFound)
	}
}

func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
