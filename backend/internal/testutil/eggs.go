package testutil

import (
	"context"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"
)

// The Paper egg of the Pelican egg repository in both formats the egg tests parse. The files are
// downloaded when a test needs them instead of being kept in the repository: the tests always
// see the current upstream eggs, and no third-party file is stored here.
const (
	PaperPLCN = "https://raw.githubusercontent.com/pelican-eggs/minecraft/refs/heads/main/java/paper/egg-paper.yaml"
	PaperPTDL = "https://raw.githubusercontent.com/pelican-eggs/minecraft/refs/heads/main/java/paper/" +
		"pterodactyl-egg-paper.json"
)

var (
	downloadsMu sync.Mutex
	downloads   = map[string][]byte{}
)

// Download returns the file at url, kept in memory for the rest of the test run.
func Download(t testing.TB, url string) []byte {
	t.Helper()
	downloadsMu.Lock()
	defer downloadsMu.Unlock()
	if data, ok := downloads[url]; ok {
		return data
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("downloading the test egg (the egg tests need internet access): %v", err)
	}
	defer func() { _ = res.Body.Close() }()
	data, err := io.ReadAll(res.Body)
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("downloading %s: status %d, %v", url, res.StatusCode, err)
	}
	downloads[url] = data
	return data
}
