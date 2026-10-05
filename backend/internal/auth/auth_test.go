package auth

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/argon2"
)

func TestPasswordHash(t *testing.T) {
	h, err := HashPassword("correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$m=19456,t=2,p=1$") || strings.Contains(h, "correct horse") {
		t.Fatalf("unexpected hash format: %s", h)
	}
	if !VerifyPassword("correct horse battery", h) {
		t.Error("correct password rejected")
	}
	if VerifyPassword("wrong password!", h) {
		t.Error("wrong password accepted")
	}
	h2, _ := HashPassword("correct horse battery")
	if h == h2 {
		t.Error("hashes must be salted")
	}
	if _, err := HashPassword("short"); err == nil {
		t.Error("short password accepted")
	}
	for _, bad := range []string{"", "plain", "$argon2id$v=19$m=1,t=1,p=1$$", "$bcrypt$x"} {
		if VerifyPassword("x", bad) {
			t.Errorf("malformed hash %q accepted", bad)
		}
	}
	if Outdated(h) {
		t.Error("a new hash is outdated")
	}
}

// Hashes made with the former parameters (64 MiB, 3 iterations, 2 threads) stay valid.
func TestOutdatedHash(t *testing.T) {
	salt := []byte("0123456789abcdef")
	key := argon2.IDKey([]byte("correct horse battery"), salt, 3, 64*1024, 2, 32)
	b64 := base64.RawStdEncoding
	old := "$argon2id$v=19$m=65536,t=3,p=2$" + b64.EncodeToString(salt) + "$" + b64.EncodeToString(key)
	if !VerifyPassword("correct horse battery", old) {
		t.Error("old hash rejected")
	}
	if !Outdated(old) {
		t.Error("old hash not outdated")
	}
}

func TestSessionToken(t *testing.T) {
	s, _ := NewSigner([]byte(strings.Repeat("k", 32)))
	now := time.Unix(1_800_000_000, 0)
	tok, _ := s.Sign(Session{User: "alice", Epoch: 2, ExpiresAt: now.Add(time.Hour).Unix()})
	got, err := s.Verify(tok, now)
	if err != nil || got.User != "alice" || got.Epoch != 2 {
		t.Fatalf("verify: %+v %v", got, err)
	}
	if _, err := s.Verify(tok, now.Add(2*time.Hour)); err == nil {
		t.Error("expired token accepted")
	}
	payload, sig, _ := strings.Cut(tok, ".")
	forged, _ := s.Sign(Session{User: "admin", ExpiresAt: now.Add(time.Hour).Unix()})
	fp, _, _ := strings.Cut(forged, ".")
	if _, err := s.Verify(fp+"."+sig, now); err == nil {
		t.Error("tampered payload accepted")
	}
	other, _ := NewSigner([]byte(strings.Repeat("x", 32)))
	if _, err := other.Verify(tok, now); err == nil {
		t.Error("token of another key accepted")
	}
	if _, err := s.Verify(payload, now); err == nil {
		t.Error("token without signature accepted")
	}
}

func TestAPIToken(t *testing.T) {
	tok, id, hash := NewAPIToken()
	if !strings.HasPrefix(tok, "kdt_"+id+"_") || APITokenID(tok) != id || len(id) != 12 {
		t.Fatalf("token %s id %s", tok, id)
	}
	if !MatchToken(tok, hash) || MatchToken(tok+"x", hash) {
		t.Error("token match wrong")
	}
	if APITokenID("eyJ1IjoiYSJ9.sig") != "" {
		t.Error("session token parsed as API token")
	}
}

func TestInviteToken(t *testing.T) {
	id := NewInviteID()
	tok, hash := NewInviteToken(id)
	if !strings.HasPrefix(tok, id+".") || InviteID(tok) != id || len(id) != 12 || !MatchToken(tok, hash) {
		t.Fatalf("token %s id %s", tok, id)
	}
	if InviteID("abc") != "" || InviteID("abc.") != "" {
		t.Error("malformed invite token accepted")
	}
}

func TestLimiter(t *testing.T) {
	l := NewLimiter(3, time.Minute)
	now := time.Unix(0, 0)
	for i := 0; i < 3; i++ {
		if ok, _ := l.Take("k", now); !ok {
			t.Fatalf("attempt %d blocked", i)
		}
	}
	// Attempts count before they are checked: a 4th one is refused while the first three still run.
	if ok, wait := l.Take("k", now); ok || wait <= 0 {
		t.Error("4th attempt must be blocked")
	}
	l.Forgive("k", now)
	if ok, _ := l.Take("k", now); !ok {
		t.Error("a forgiven attempt must free its slot")
	}
	if ok, _ := l.Take("k", now.Add(61*time.Second)); !ok {
		t.Error("window did not expire")
	}
	l.Reset("k")
	if ok, _ := l.Take("k", now); !ok {
		t.Error("reset did not clear failures")
	}
}

func TestArgonRunsAtMostTwiceAtOnce(t *testing.T) {
	h, _ := HashPassword("correct horse battery")
	// Occupy both slots: a password check must wait until one is free.
	argonSlots <- struct{}{}
	argonSlots <- struct{}{}
	done := make(chan bool)
	go func() { done <- VerifyPassword("correct horse battery", h) }()
	select {
	case <-done:
		t.Fatal("the check ran although both slots were taken")
	case <-time.After(100 * time.Millisecond):
	}
	<-argonSlots
	if !<-done {
		t.Error("the check must succeed once a slot is free")
	}
	<-argonSlots
}

func TestExpensiveHashesAreRejected(t *testing.T) {
	h, _ := HashPassword("correct horse battery")
	expensive := strings.Replace(h, "m=19456", "m=4194304", 1)
	if VerifyPassword("correct horse battery", expensive) {
		t.Error("a hash that needs 4 GiB must not be checked")
	}
}

func TestRevokedSessions(t *testing.T) {
	s, _ := NewSigner([]byte(strings.Repeat("k", 32)))
	now := time.Now()
	a, _ := s.Sign(Session{User: "alice", ExpiresAt: now.Add(time.Hour).Unix()})
	b, _ := s.Sign(Session{User: "alice", ExpiresAt: now.Add(time.Hour).Unix() + 1})
	s.Revoke(a, now)
	if _, err := s.Verify(a, now); err == nil {
		t.Error("a revoked token must be rejected")
	}
	if _, err := s.Verify(b, now); err != nil {
		t.Error("other tokens of the user stay valid")
	}
	c, _ := s.Sign(Session{User: "alice", ExpiresAt: now.Add(3 * time.Hour).Unix()})
	s.Revoke(c, now.Add(2*time.Hour)) // a's entry has expired by then and is pruned
	if len(s.revoked) != 1 {
		t.Errorf("expired entries must be pruned: %d", len(s.revoked))
	}
	x, _ := s.Sign(Session{User: "alice", ExpiresAt: now.Add(time.Hour).Unix()})
	y, _ := s.Sign(Session{User: "alice", ExpiresAt: now.Add(time.Hour).Unix()})
	if x == y {
		t.Error("two logins in the same second must get different tokens")
	}
}

func TestLimiterForgetsOldKeys(t *testing.T) {
	l := NewLimiter(5, time.Minute)
	start := time.Now()
	for i := range 100 {
		l.Take(fmt.Sprintf("10.0.0.%d|guess", i), start)
	}
	// A window later, one more attempt sweeps every key whose failures expired.
	l.Take("10.0.1.1|admin", start.Add(time.Minute))
	if len(l.failures) != 1 {
		t.Errorf("expired keys must be swept: %d keys left", len(l.failures))
	}
}

func TestLimiterRemembersAtMostMaxKeys(t *testing.T) {
	l := NewLimiter(5, time.Hour)
	l.maxKeys = 3
	now := time.Now()
	for i := range 10 {
		l.Take(fmt.Sprintf("key-%d", i), now.Add(time.Duration(i)*time.Second))
	}
	if len(l.failures) != 3 {
		t.Fatalf("keys: %d, want at most 3", len(l.failures))
	}
	if _, ok := l.failures["key-9"]; !ok {
		t.Error("the newest key must be kept")
	}
	if _, ok := l.failures["key-0"]; ok {
		t.Error("the oldest key must be dropped first")
	}
	// A known key keeps counting although the limiter is full.
	for range 4 {
		l.Take("key-9", now.Add(10*time.Second))
	}
	if ok, _ := l.Take("key-9", now.Add(11*time.Second)); ok {
		t.Error("a key at its limit must stay blocked")
	}
}
