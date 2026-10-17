package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"strings"
)

// APITokenPrefix marks personal access tokens ("kdt_<id>_<secret>").
const APITokenPrefix = "kdt_"

// NewAPIToken returns a new token, its public id and the hash that is stored.
func NewAPIToken() (token, id, hash string) {
	id = hex.EncodeToString(randomBytes(6))
	token = APITokenPrefix + id + "_" + base64.RawURLEncoding.EncodeToString(randomBytes(32))
	return token, id, HashToken(token)
}

// randomBytes returns n random bytes (crypto/rand never fails).
func randomBytes(n int) []byte {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return b
}

// HashToken hashes a token for storage. Tokens are 256-bit random, so plain SHA-256 is sufficient.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// APITokenID extracts the id of a token, or "" when it is not an API token.
func APITokenID(token string) string {
	rest, ok := strings.CutPrefix(token, APITokenPrefix)
	if !ok {
		return ""
	}
	id, _, ok := strings.Cut(rest, "_")
	if !ok {
		return ""
	}
	return id
}

// MatchToken compares a token with a stored hash in constant time.
func MatchToken(token, hash string) bool {
	return subtle.ConstantTimeCompare([]byte(HashToken(token)), []byte(hash)) == 1
}

// NewInviteID returns a random public id for an invite (the name of its Invite object).
func NewInviteID() string {
	return hex.EncodeToString(randomBytes(6))
}

// NewInviteToken returns a new token for the link of an invite ("<id>.<secret>", URL safe) and the hash that is
// stored; a new token replaces the old link.
func NewInviteToken(id string) (token, hash string) {
	token = id + "." + base64.RawURLEncoding.EncodeToString(randomBytes(32))
	return token, HashToken(token)
}

// InviteID extracts the id of an invite token, or "" when it is none.
func InviteID(token string) string {
	id, secret, ok := strings.Cut(token, ".")
	if !ok || secret == "" {
		return ""
	}
	return id
}

// NewSetupToken returns a random one-time token for the setup page (URL safe).
func NewSetupToken() string {
	return base64.RawURLEncoding.EncodeToString(randomBytes(18))
}

// EqualTokens compares two secrets in constant time.
func EqualTokens(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
