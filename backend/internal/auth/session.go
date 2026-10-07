package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// HTTPS reports whether the request came over HTTPS, directly or through a TLS terminating proxy.
func HTTPS(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

// ErrInvalidToken is returned for malformed, tampered or expired tokens.
var ErrInvalidToken = errors.New("invalid or expired token")

// Session is the payload of a signed session token.
type Session struct {
	User      string `json:"u"`
	Epoch     int64  `json:"e"`
	ExpiresAt int64  `json:"x"`
	// IssuedAt lets a shorter session lifetime end sessions that were signed before (unix time).
	IssuedAt int64 `json:"i,omitempty"`
	// Nonce makes every token unique, so logging out one session never ends another.
	Nonce string `json:"n,omitempty"`
}

// Signer creates and verifies HMAC-SHA256 signed session tokens.
type Signer struct {
	key []byte

	mu sync.Mutex
	// revoked holds the signatures of logged out tokens until they expire (unix time). It lives
	// in memory: after a panel restart a logged out token works again until it expires, while
	// "log out everywhere" (the user's session epoch) also survives restarts.
	revoked map[string]int64
}

// NewSigner returns a signer for a secret key (at least 32 bytes).
func NewSigner(key []byte) (*Signer, error) {
	if len(key) < 32 {
		return nil, errors.New("session key must be at least 32 bytes")
	}
	return &Signer{key: key}, nil
}

func (s *Signer) mac(payload string) string {
	m := hmac.New(sha256.New, s.key)
	m.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

// Sign returns a token "<payload>.<signature>".
func (s *Signer) Sign(sess Session) (string, error) {
	if sess.Nonce == "" {
		b := make([]byte, 8)
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		sess.Nonce = base64.RawURLEncoding.EncodeToString(b)
	}
	raw, err := json.Marshal(sess)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(raw)
	return payload + "." + s.mac(payload), nil
}

// Revoke invalidates a valid session token until it expires (logout).
func (s *Signer) Revoke(token string, now time.Time) {
	sess, err := s.Verify(token, now)
	if err != nil {
		return
	}
	_, sig, _ := strings.Cut(token, ".")
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.revoked == nil {
		s.revoked = map[string]int64{}
	}
	for k, expires := range s.revoked {
		if now.Unix() >= expires {
			delete(s.revoked, k)
		}
	}
	s.revoked[sig] = sess.ExpiresAt
}

func (s *Signer) isRevoked(sig string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.revoked[sig]
	return ok
}

// Verify checks signature, expiry and revocation and returns the session.
func (s *Signer) Verify(token string, now time.Time) (Session, error) {
	payload, sig, ok := strings.Cut(token, ".")
	if !ok || !hmac.Equal([]byte(sig), []byte(s.mac(payload))) || s.isRevoked(sig) {
		return Session{}, ErrInvalidToken
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return Session{}, ErrInvalidToken
	}
	var sess Session
	if err := json.Unmarshal(raw, &sess); err != nil || sess.User == "" {
		return Session{}, ErrInvalidToken
	}
	if now.Unix() >= sess.ExpiresAt {
		return Session{}, ErrInvalidToken
	}
	return sess, nil
}

// DeviceToken marks a browser in which user signed in, until expires: "<unix expiry>.<signature>". It is no
// session token (Verify rejects it) and grants nothing on its own.
func (s *Signer) DeviceToken(user string, expires time.Time) string {
	exp := strconv.FormatInt(expires.Unix(), 10)
	return exp + "." + s.mac("device|"+user+"|"+exp)
}

// KnownDevice reports whether token is an unexpired device token of user.
func (s *Signer) KnownDevice(token, user string, now time.Time) bool {
	exp, sig, ok := strings.Cut(token, ".")
	unix, err := strconv.ParseInt(exp, 10, 64)
	return ok && err == nil && now.Unix() < unix && hmac.Equal([]byte(sig), []byte(s.mac("device|"+user+"|"+exp)))
}

// SignValue signs a JSON value for one purpose until expires: "<payload>.<signature>". The purpose is part of the
// signature, so a value signed for one use is never accepted for another (and no session token as either).
func (s *Signer) SignValue(purpose string, v any, expires time.Time) (string, error) {
	raw, err := json.Marshal(signedValue[any]{ExpiresAt: expires.Unix(), Value: v})
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(raw)
	return payload + "." + s.mac(purpose+"|"+payload), nil
}

// VerifyValue checks the signature and expiry of a value signed by SignValue and decodes it into v (left
// unchanged when the check fails).
func (s *Signer) VerifyValue(purpose, token string, v any, now time.Time) error {
	payload, sig, ok := strings.Cut(token, ".")
	if !ok || !hmac.Equal([]byte(sig), []byte(s.mac(purpose+"|"+payload))) {
		return ErrInvalidToken
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return ErrInvalidToken
	}
	var sv signedValue[json.RawMessage]
	if err := json.Unmarshal(raw, &sv); err != nil || now.Unix() >= sv.ExpiresAt {
		return ErrInvalidToken
	}
	if err := json.Unmarshal(sv.Value, v); err != nil {
		return ErrInvalidToken
	}
	return nil
}

type signedValue[T any] struct {
	ExpiresAt int64 `json:"x"`
	Value     T     `json:"v"`
}
