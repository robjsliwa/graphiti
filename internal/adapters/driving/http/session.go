package http

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const sessionCookieName = "graphiti_session"

// Session holds the authenticated user's session data.
type Session struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SessionStore manages secure cookie-based sessions signed with HMAC-SHA256.
type SessionStore struct {
	secret []byte
	maxAge time.Duration
	secure bool
}

// NewSessionStore creates a new SessionStore with the given HMAC secret, max session age, and secure flag.
func NewSessionStore(secret string, maxAge time.Duration, secure bool) *SessionStore {
	return &SessionStore{
		secret: []byte(secret),
		maxAge: maxAge,
		secure: secure,
	}
}

// Get reads and validates the session from the request cookie.
func (s *SessionStore) Get(r *http.Request) (*Session, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, fmt.Errorf("no session cookie: %w", err)
	}

	session, err := s.decode(cookie.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid session cookie: %w", err)
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, errors.New("session expired")
	}

	return session, nil
}

// Set writes a signed session cookie to the response.
func (s *SessionStore) Set(w http.ResponseWriter, session *Session) error {
	value, err := s.encode(session)
	if err != nil {
		return fmt.Errorf("encoding session: %w", err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   int(s.maxAge.Seconds()),
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteStrictMode,
	})

	return nil
}

// Clear removes the session cookie from the response.
func (s *SessionStore) Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteStrictMode,
	})
}

// encode serializes the session to JSON, signs it with HMAC-SHA256, and returns
// a base64-encoded string in the format "signature.payload".
func (s *SessionStore) encode(session *Session) (string, error) {
	payload, err := json.Marshal(session)
	if err != nil {
		return "", fmt.Errorf("marshaling session: %w", err)
	}

	payloadB64 := base64.URLEncoding.EncodeToString(payload)
	sig := s.sign(payloadB64)
	sigB64 := base64.URLEncoding.EncodeToString(sig)

	return sigB64 + "." + payloadB64, nil
}

// decode verifies the HMAC signature and deserializes the session from a cookie value.
func (s *SessionStore) decode(value string) (*Session, error) {
	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 {
		return nil, errors.New("malformed session cookie")
	}

	sigB64, payloadB64 := parts[0], parts[1]

	sig, err := base64.URLEncoding.DecodeString(sigB64)
	if err != nil {
		return nil, fmt.Errorf("decoding signature: %w", err)
	}

	expectedSig := s.sign(payloadB64)
	if !hmac.Equal(sig, expectedSig) {
		return nil, errors.New("invalid session signature")
	}

	payload, err := base64.URLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil, fmt.Errorf("decoding payload: %w", err)
	}

	var session Session
	if err := json.Unmarshal(payload, &session); err != nil {
		return nil, fmt.Errorf("unmarshaling session: %w", err)
	}

	return &session, nil
}

// sign computes the HMAC-SHA256 signature of the given data.
func (s *SessionStore) sign(data string) []byte {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}
