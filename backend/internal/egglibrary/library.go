// Package egglibrary lists the eggs of git repositories for the egg library on the eggs page.
// Every repository is downloaded as one archive in the layout of its host (hosts.go) and its egg
// files are parsed; the summaries are kept in memory for CacheTTL. Nothing is stored in the
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
	"net/url"
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
	// Repository is the repository URL as configured (https://<host>/<owner>/<repo>).
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
	mu    sync.Mutex
	cache map[string]Repository
}

// New returns an empty library.
func New() *Library {
	return &Library{cache: map[string]Repository{}}
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

// GetRepository returns the eggs of one repository, configured or not (the settings check a URL
// with it before it is saved). A readable one is kept like a listed one, so the library shows it
// at once after it was added; the next List keeps only the configured repositories. Failures are
// not kept: a repository made public is found by the next check.
func (l *Library) GetRepository(ctx context.Context, url string) Repository {
	l.mu.Lock()
	r, ok := l.cache[url]
	l.mu.Unlock()
	if ok && time.Since(r.FetchedAt) < CacheTTL {
		return r
	}
	if r = l.read(ctx, url); r.Error == "" {
		l.mu.Lock()
		l.cache[url] = r
		l.mu.Unlock()
	}
	return r
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
func (l *Library) read(ctx context.Context, repoURL string) Repository {
	r := Repository{URL: repoURL, Eggs: []Egg{}, FetchedAt: time.Now()}
	u, err := url.Parse(repoURL)
	if err != nil {
		r.Error = err.Error()
		return r
	}
	files, h, err := download(ctx, u)
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
			Repository: repoURL, Path: file, URL: h.raw(u, file),
			Name: spec.DisplayName, Description: spec.Description, Author: spec.Author,
			Format: spec.Source.Format, UUID: spec.Source.UUID, Tags: append([]string{}, spec.Tags...), Icon: spec.Icon,
		})
	}
	r.Eggs = dedupe(r.Eggs)
	return r
}

var (
	// errNoArchive is returned when a URL answers without a repository archive.
	errNoArchive = errors.New("repository not found or not public")
	// errUnknownHost is returned when no layout of a self-hosted server answers with an archive.
	errUnknownHost = errors.New("no repository archive found (GitHub, GitLab, Gitea, Forgejo and Bitbucket work)")
)

// download returns the egg files of a repository and the layout of its host. On a host it does
// not know it tries every layout until one answers with an archive.
func download(ctx context.Context, repo *url.URL) (map[string][]byte, host, error) {
	hosts := hostsOf(repo)
	var err error
	for _, h := range hosts {
		var files map[string][]byte
		if files, err = readArchive(ctx, h.archive(repo)); err == nil {
			return files, h, nil
		}
		if !errors.Is(err, errNoArchive) {
			return nil, host{}, err
		}
	}
	if len(hosts) > 1 {
		err = errUnknownHost
	}
	return nil, host{}, err
}

// readArchive downloads a repository archive and returns its egg files by their path in the
// repository.
func readArchive(ctx context.Context, archive string) (map[string][]byte, error) {
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
		return nil, fmt.Errorf("%w (%s)", errNoArchive, res.Status)
	}
	gz, err := gzip.NewReader(io.LimitReader(res.Body, maxArchive))
	if err != nil {
		return nil, fmt.Errorf("%w (the answer is no archive)", errNoArchive)
	}
	return eggFiles(tar.NewReader(gz))
}

// eggFiles reads the files named like eggs out of an archive. Every host puts the repository
// into one top folder ("<repo>-<ref>/").
func eggFiles(tr *tar.Reader) (map[string][]byte, error) {
	files := map[string][]byte{}
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return files, nil
		}
		if err != nil {
			return nil, fmt.Errorf("reading the repository: %w", err)
		}
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
