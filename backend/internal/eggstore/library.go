// Egg library: the eggs of git repositories, listed on the eggs page. Every repository is downloaded as
// one archive in the layout of its host (library_hosts.go) and its egg files are parsed; the summaries
// are kept in memory for LibraryCacheTTL. Nothing is stored in the cluster: an egg is stored only when
// it is installed (imported from its raw file URL).

package eggstore

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
)

// Errors of the egg library.
var (
	// ErrNotInLibrary is returned for an egg that is not in a configured repository.
	ErrNotInLibrary = errors.New("egg not found in the library")
	// errNoArchive is returned when a URL answers without a repository archive.
	errNoArchive = errors.New("repository not found or not public")
	// errUnknownHost is returned when no layout of a self-hosted server answers with an archive.
	errUnknownHost = errors.New("no repository archive found (GitHub, GitLab, Gitea, Forgejo and Bitbucket work)")
	// errTooLarge is returned for repositories beyond maxUnpacked or maxEggFiles.
	errTooLarge = fmt.Errorf("the repository has more than %d MiB or %d egg files", maxUnpacked>>20, maxEggFiles)
)

// LibraryCacheTTL is how long a downloaded repository is listed without downloading it again.
const LibraryCacheTTL = 15 * time.Minute

// Limits of one repository: the download, its unpacked files and the files named like eggs.
const (
	maxArchive  = 64 << 20
	maxUnpacked = 256 << 20
	maxEggFiles = 1000
)

// eggFile matches the file names egg repositories use (egg-paper.yaml, pterodactyl-egg-paper.json).
var eggFile = regexp.MustCompile(`(?i)(^|-)egg-.*\.(json|ya?ml)$`)

// LibraryEgg is the summary of an egg file in a repository.
type LibraryEgg struct {
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

// LibraryRepository is the result of reading one repository.
type LibraryRepository struct {
	URL       string       `json:"url"`
	Eggs      []LibraryEgg `json:"eggs"`
	FetchedAt time.Time    `json:"fetchedAt"`
	// Error tells why the repository could not be read (its eggs are then missing).
	Error string `json:"error,omitempty"`
}

// Library downloads repositories and keeps their eggs for LibraryCacheTTL.
type Library struct {
	mu    sync.Mutex
	cache map[string]LibraryRepository
}

// NewLibrary returns an empty library.
func NewLibrary() *Library {
	return &Library{cache: map[string]LibraryRepository{}}
}

// List returns the eggs of the repositories, downloading those not cached (or all with refresh).
// Failures are not kept, the next List tries again.
func (l *Library) List(ctx context.Context, repos []string, refresh bool) []LibraryRepository {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]LibraryRepository, len(repos))
	var wg sync.WaitGroup
	for i, url := range repos {
		if r, ok := l.cache[url]; ok && !refresh && time.Since(r.FetchedAt) < LibraryCacheTTL {
			out[i] = r
			continue
		}
		wg.Go(func() { out[i] = l.read(ctx, url) })
	}
	wg.Wait()
	// Only the configured repositories stay in memory.
	l.cache = make(map[string]LibraryRepository, len(out))
	for _, r := range out {
		if r.Error == "" {
			l.cache[r.URL] = r
		}
	}
	return out
}

// GetRepository returns the eggs of one repository, configured or not (the settings check a URL
// with it before it is saved). A readable one is kept like a listed one, so the library shows it
// at once after it was added; the next List keeps only the configured repositories. Failures are
// not kept: a repository made public is found by the next check.
func (l *Library) GetRepository(ctx context.Context, url string) LibraryRepository {
	l.mu.Lock()
	r, ok := l.cache[url]
	l.mu.Unlock()
	if ok && time.Since(r.FetchedAt) < LibraryCacheTTL {
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
func (l *Library) Get(ctx context.Context, repos []string, repo, file string) (*v1alpha1.EggSpec, LibraryEgg, error) {
	if !slices.Contains(repos, repo) {
		return nil, LibraryEgg{}, ErrNotInLibrary
	}
	eggs := l.GetRepository(ctx, repo).Eggs
	i := slices.IndexFunc(eggs, func(e LibraryEgg) bool { return e.Path == file })
	if i < 0 {
		return nil, LibraryEgg{}, ErrNotInLibrary
	}
	e := eggs[i]
	data, err := Download(ctx, e.URL)
	if err != nil {
		return nil, e, err
	}
	spec, err := Parse(data)
	return spec, e, err
}

// read downloads a repository's archive and summarizes its egg files. The download is shared by
// everyone waiting for the library, so it does not end with the request that started it.
func (l *Library) read(ctx context.Context, repoURL string) LibraryRepository {
	r := LibraryRepository{URL: repoURL, Eggs: []LibraryEgg{}, FetchedAt: time.Now()}
	u, err := url.Parse(repoURL)
	if err != nil {
		r.Error = err.Error()
		return r
	}
	eggs, h, err := download(context.WithoutCancel(ctx), u)
	if err != nil {
		r.Error = err.Error()
		return r
	}
	for i := range eggs {
		eggs[i].Repository, eggs[i].URL = repoURL, h.raw(u, eggs[i].Path)
	}
	r.Eggs = dedupe(eggs)
	return r
}

// download returns the egg summaries of a repository and the layout of its host. On a host it
// does not know it tries every layout until one answers with an archive.
func download(ctx context.Context, repo *url.URL) ([]LibraryEgg, host, error) {
	hosts := hostsOf(repo)
	var err error
	for _, h := range hosts {
		var eggs []LibraryEgg
		if eggs, err = readArchive(ctx, h.archive(repo)); err == nil {
			return eggs, h, nil
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

// readArchive downloads a repository archive and summarizes its egg files.
func readArchive(ctx context.Context, archive string) ([]LibraryEgg, error) {
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
	return summarize(tar.NewReader(gz))
}

// summarize parses the files named like eggs of an archive one by one and keeps only their
// summaries (Path without the top folder "<repo>-<ref>/" every host adds). Files that do not parse
// are skipped: not every matching file is an egg (e.g. GitHub issue templates).
func summarize(tr *tar.Reader) ([]LibraryEgg, error) {
	var eggs []LibraryEgg
	var unpacked int64
	files := 0
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return eggs, nil
		}
		if err != nil {
			return nil, fmt.Errorf("reading the repository: %w", err)
		}
		if unpacked += h.Size; unpacked > maxUnpacked {
			return nil, errTooLarge
		}
		_, file, _ := strings.Cut(h.Name, "/")
		if h.Typeflag != tar.TypeReg || h.Size > MaxSize || !eggFile.MatchString(path.Base(file)) {
			continue
		}
		if files++; files > maxEggFiles {
			return nil, errTooLarge
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("reading the repository: %w", err)
		}
		if spec, err := egg.Parse(data); err == nil {
			eggs = append(eggs, LibraryEgg{
				Path: file, Name: spec.DisplayName, Description: spec.Description, Author: spec.Author,
				Format: spec.Source.Format, UUID: spec.Source.UUID, Tags: append([]string{}, spec.Tags...),
				Icon: spec.Icon,
			})
		}
	}
}

// dedupe keeps one file per egg: Pelican repositories also carry each egg in the Pterodactyl
// format next to it (pterodactyl-egg-paper.json); the Pelican file wins. The result is sorted by
// name.
func dedupe(eggs []LibraryEgg) []LibraryEgg {
	best := map[string]LibraryEgg{}
	for _, e := range eggs {
		key := path.Dir(e.Path) + "\x00" + strings.ToLower(e.Name)
		if cur, ok := best[key]; !ok || rank(e) > rank(cur) || (rank(e) == rank(cur) && e.Path < cur.Path) {
			best[key] = e
		}
	}
	out := make([]LibraryEgg, 0, len(best))
	for _, e := range best {
		out = append(out, e)
	}
	slices.SortFunc(out, func(a, b LibraryEgg) int {
		byName := strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		return cmp.Or(byName, strings.Compare(a.Path, b.Path))
	})
	return out
}

func rank(e LibraryEgg) int {
	if strings.HasPrefix(e.Format, "PLCN") {
		return 1
	}
	return 0
}
