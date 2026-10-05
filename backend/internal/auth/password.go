// Package auth implements password hashing, session tokens, API tokens and login throttling.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters: the first configuration recommended by OWASP (19 MiB, 2 iterations,
// 1 thread). Every check allocates argonMemory, so a larger value shows up as memory peaks of
// the panel at every sign-in. Hashes with other parameters stay valid and are replaced at the
// next sign-in (Outdated).
const (
	argonMemory  = 19 * 1024 // KiB
	argonTime    = 2
	argonThreads = 1
	argonKeyLen  = 32
	saltLen      = 16
	// MinPasswordLength and MaxPasswordLength are enforced when passwords are set (the sign-in request
	// accepts the same maximum).
	MinPasswordLength = 8
	MaxPasswordLength = 1024
)

// argonSlots limits how many Argon2 computations run at once: each one needs the memory of
// its hash (19 MiB, older hashes up to 64 MiB), so a burst of login requests must queue
// instead of exhausting the panel's memory.
var argonSlots = make(chan struct{}, 2)

// maxArgonMemory rejects stored hashes whose memory parameter would be too expensive to check.
const maxArgonMemory = 64 * 1024

// params is the parameter field of a hash made with the current parameters.
var params = fmt.Sprintf("m=%d,t=%d,p=%d", argonMemory, argonTime, argonThreads)

func idKey(password, salt []byte, time, memory uint32, threads uint8, keyLen uint32) []byte {
	argonSlots <- struct{}{}
	defer func() { <-argonSlots }()
	return argon2.IDKey(password, salt, time, memory, threads, keyLen)
}

// HashPassword returns an Argon2id hash in PHC string format.
func HashPassword(password string) (string, error) {
	if len(password) < MinPasswordLength {
		return "", fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	}
	if len(password) > MaxPasswordLength {
		return "", fmt.Errorf("password must be at most %d characters", MaxPasswordLength)
	}
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := idKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	b64 := base64.RawStdEncoding
	return fmt.Sprintf(
		"$argon2id$v=%d$%s$%s$%s", argon2.Version, params, b64.EncodeToString(salt), b64.EncodeToString(key),
	), nil
}

// Outdated reports whether a hash was made with other parameters than HashPassword uses now.
func Outdated(encoded string) bool {
	parts := strings.Split(encoded, "$")
	return len(parts) != 6 || parts[3] != params
}

// VerifyPassword checks a password against a PHC Argon2id hash in constant time.
func VerifyPassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	// "", "argon2id", "v=19", "m=..,t=..,p=..", salt, key
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false
	}
	var memory, iterations uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &threads); err != nil ||
		memory > maxArgonMemory || iterations > 10 || threads == 0 {
		return false
	}
	b64 := base64.RawStdEncoding
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return false
	}
	want, err := b64.DecodeString(parts[5])
	if err != nil || len(want) == 0 {
		return false
	}
	got := idKey([]byte(password), salt, iterations, memory, threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}

// dummyHash is verified for unknown users so that login timing does not reveal usernames. It is
// made at the first use, not at start-up.
var dummyHash = sync.OnceValue(func() string {
	h, _ := HashPassword("dummy-password-for-timing")
	return h
})

// VerifyDummy spends the same time as a real password check.
func VerifyDummy(password string) { VerifyPassword(password, dummyHash()) }
