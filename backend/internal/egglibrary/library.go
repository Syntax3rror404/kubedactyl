// Package egglibrary lists the eggs of GitHub repositories for the egg library on the eggs page.
// Every repository is downloaded as one archive (codeload.github.com, no API rate limit) and its
// egg files are parsed; the summaries are kept in memory for CacheTTL. Nothing is stored in the
// cluster: an egg is stored only when it is installed (imported from its raw file URL).
package egglibrary

import (
	"archive/tar"
	"cmp"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"app/api/v1alpha1"
	"app/internal/egg"
	"app/internal/eggstore"
)

// CacheTTL is how long a downloaded repository is listed without downloading it again.
const CacheTTL = 15 * time.Minute

// maxArchive limits the download of one repository.
const maxArchive = 64 << 20

// ErrNotFound is returned for an egg that is not in a configured repository.
var ErrNotFound = errors.New("egg not found in the library")

// eggFile matches the file names egg repositories use (egg-paper.yaml, pterodactyl-egg-paper.json).
var eggFile = regexp.MustCompile(`(?i)(^|-)egg-.*\.(json|ya?ml)$`)

// Egg is the summary of an egg file in a repository.
type Egg struct {
	// Repository is the GitHub repository (https://github.com/<owner>/<repo>).
	Repository string `json:"repository"`
	// Path of the file in the repository.
	Path string `json:"path"`
	// URL of the raw file: installing imports it from there and keeps it as update URL.
	URL         string   `json:"url"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Format      string   `json:"format"`
	UUID        string   `json:"uuid"`
	Tags        []string `json:"tags"`
	Icon        string   `json:"icon,omitempty"`
}

// Repository is the result of reading one repository.
type Repository struct {
	URL       string    `json:"url"`
	Eggs      []Egg     `json:"eggs"`
	FetchedAt time.Time `json:"fetchedAt"`
	// Error tells why the repository could not be read (its eggs are then missing).
	Error string `json:"error,omitempty"`
}

// Library downloads repositories and keeps their eggs for CacheTTL.
type Library struct {
	// Archive returns the URL of a repository's archive (tests serve their own).
	Archive func(owner, repo string) string
	// Raw returns the URL of a file in a repository.
	Raw func(owner, repo, file string) string

	mu    sync.Mutex
	cache map[string]Repository
}

// New returns a library that reads from GitHub.
func New() *Library {
	return &Library{
		Archive: func(owner, repo string) string {
			return "https://codeload.github.com/" + owner + "/" + repo + "/tar.gz/HEAD"
		},
		Raw: func(owner, repo, file string) string {
			return "https://raw.githubusercontent.com/" + owner + "/" + repo + "/HEAD/" + file
		},
		cache: map[string]Repository{},
	}
}

// List returns the eggs of the repositories, downloading those not cached (or all with refresh).
func (l *Library) List(ctx context.Context, repos []string, refresh bool) []Repository {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]Repository, len(repos))
	var wg sync.WaitGroup
	for i, url := range repos {
		if r, ok := l.cache[url]; ok && !refresh && time.Since(r.FetchedAt) < CacheTTL {
			out[i] = r
			continue
		}
		wg.Go(func() { out[i] = l.read(ctx, url) })
	}
	wg.Wait()
	// Only the configured repositories stay in memory.
	l.cache = make(map[string]Repository, len(out))
	for _, r := range out {
		l.cache[r.URL] = r
	}
	return out
}

// Get downloads one egg of a configured repository and parses it.
func (l *Library) Get(ctx context.Context, repos []string, repo, file string) (*v1alpha1.EggSpec, Egg, error) {
	for _, r := range l.List(ctx, repos, false) {
		for _, e := range r.Eggs {
			if r.URL != repo || e.Path != file {
				continue
			}
			data, err := eggstore.Download(ctx, e.URL)
			if err != nil {
				return nil, e, err
			}
			spec, err := egg.Parse(data)
			if err != nil {
				return nil, e, fmt.Errorf("%w: %w", eggstore.ErrInvalid, err)
			}
			return spec, e, nil
		}
	}
	return nil, Egg{}, ErrNotFound
}

// read downloads a repository's archive and summarizes its egg files.
func (l *Library) read(ctx context.Context, url string) Repository {
	r := Repository{URL: url, Eggs: []Egg{}, FetchedAt: time.Now()}
	owner, repo, _ := strings.Cut(strings.TrimPrefix(url, "https://github.com/"), "/")
	files, err := l.download(ctx, l.Archive(owner, repo))
	if err != nil {
		r.Error = err.Error()
		return r
	}
	for file, data := range files {
		spec, err := egg.Parse(data)
		if err != nil {
			continue // not every matching file is an egg (e.g. GitHub issue templates)
		}
		r.Eggs = append(r.Eggs, Egg{
			Repository: url, Path: file, URL: l.Raw(owner, repo, file),
			Name: spec.DisplayName, Description: spec.Description, Author: spec.Author,
			Format: spec.Source.Format, UUID: spec.Source.UUID, Tags: append([]string{}, spec.Tags...), Icon: spec.Icon,
		})
	}
	r.Eggs = dedupe(r.Eggs)
	return r
}

// download returns the egg files of a repository archive by their path in the repository.
func (l *Library) download(ctx context.Context, archive string) (map[string][]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, archive, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading the repository: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading the repository: %s", res.Status)
	}
	gz, err := gzip.NewReader(io.LimitReader(res.Body, maxArchive))
	if err != nil {
		return nil, fmt.Errorf("reading the repository: %w", err)
	}
	files := map[string][]byte{}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return files, nil
		}
		if err != nil {
			return nil, fmt.Errorf("reading the repository: %w", err)
		}
		// Paths start with "<repo>-<ref>/".
		_, file, _ := strings.Cut(h.Name, "/")
		if h.Typeflag != tar.TypeReg || h.Size > eggstore.MaxSize || !eggFile.MatchString(path.Base(file)) {
			continue
		}
		if files[file], err = io.ReadAll(tr); err != nil {
			return nil, fmt.Errorf("reading the repository: %w", err)
		}
	}
}

// dedupe keeps one file per egg: Pelican repositories also carry each egg in the Pterodactyl
// format next to it (pterodactyl-egg-paper.json); the Pelican file wins. The result is sorted by
// name.
func dedupe(eggs []Egg) []Egg {
	best := map[string]Egg{}
	for _, e := range eggs {
		key := path.Dir(e.Path) + "\x00" + strings.ToLower(e.Name)
		if cur, ok := best[key]; !ok || rank(e) > rank(cur) || (rank(e) == rank(cur) && e.Path < cur.Path) {
			best[key] = e
		}
	}
	out := make([]Egg, 0, len(best))
	for _, e := range best {
		out = append(out, e)
	}
	slices.SortFunc(out, func(a, b Egg) int {
		byName := strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		return cmp.Or(byName, strings.Compare(a.Path, b.Path))
	})
	return out
}

func rank(e Egg) int {
	if strings.HasPrefix(e.Format, "PLCN") {
		return 1
	}
	return 0
}
