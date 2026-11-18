package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
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

// Signer creates and verifies HMAC-SHA256 signed session tokens. It keeps no state: logged out sessions are kept
// in the status of their user (users.Store.LogOut), so they stay logged out after a restart and on every replica.
type Signer struct {
	key []byte
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

// seal encodes v as JSON and signs it: "<payload>.<signature>". The prefix goes into the signature only.
func (s *Signer) seal(prefix string, v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(raw)
	return payload + "." + s.mac(prefix+payload), nil
}

// open checks the signature of a sealed token and returns its JSON and signature.
func (s *Signer) open(prefix, token string) (raw []byte, sig string, err error) {
	payload, sig, ok := strings.Cut(token, ".")
	if !ok || !hmac.Equal([]byte(sig), []byte(s.mac(prefix+payload))) {
		return nil, "", ErrInvalidToken
	}
	raw, err = base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, "", ErrInvalidToken
	}
	return raw, sig, nil
}

// Sign returns a token "<payload>.<signature>".
func (s *Signer) Sign(sess Session) (string, error) {
	if sess.Nonce == "" {
		sess.Nonce = base64.RawURLEncoding.EncodeToString(randomBytes(8))
	}
	return s.seal("", sess)
}

// SessionID names a session token without being one: a hash of its signature. A logged out session is kept by it,
// so the token itself is never stored.
func SessionID(token string) string {
	_, sig, _ := strings.Cut(token, ".")
	sum := sha256.Sum256([]byte(sig))
	return base64.RawURLEncoding.EncodeToString(sum[:16])
}

// Verify checks signature and expiry and returns the session; whether it was logged out is the user's status.
func (s *Signer) Verify(token string, now time.Time) (Session, error) {
	raw, _, err := s.open("", token)
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
	return s.seal(purpose+"|", signedValue[any]{ExpiresAt: expires.Unix(), Value: v})
}

// VerifyValue checks the signature and expiry of a value signed by SignValue and decodes it into v (left
// unchanged when the check fails).
func (s *Signer) VerifyValue(purpose, token string, v any, now time.Time) error {
	raw, _, err := s.open(purpose+"|", token)
	if err != nil {
		return err
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
