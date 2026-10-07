// Package files implements the file manager on top of exec in the helper pod.
package files

import (
	"context"
	"errors"
	"io"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"app/api/v1alpha1"
	"app/internal/gameserver"
	"app/internal/kube"
)

// ErrNotFound is returned for missing files or directories.
var ErrNotFound = errors.New("file not found")

// ErrDenied is returned for paths outside the server root or on the egg denylist.
var ErrDenied = errors.New("access to this file is denied")

// ErrExists is returned when the target of a move or rename already exists.
var ErrExists = errors.New("the target already exists")

// ErrTooLarge is returned when a file is too big to be opened in the editor.
var ErrTooLarge = errors.New("file is too large to be edited")

// Entry is a file or directory in a listing.
type Entry struct {
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	Mode       string    `json:"mode"`
	ModifiedAt time.Time `json:"modifiedAt"`
	IsDir      bool      `json:"isDirectory"`
	IsSymlink  bool      `json:"isSymlink"`
}

// Manager runs file operations for game servers.
type Manager struct {
	Kube *kube.Client
}

// ErrIntoItself is returned when a folder would be moved into itself.
var ErrIntoItself = errors.New("cannot move a folder into itself")

// ErrTooManyEntries is returned for folders whose listing exceeds maxListing.
var ErrTooManyEntries = errors.New("the folder has too many entries to be listed")

// maxListing limits the output of a folder listing (about 100,000 entries).
const maxListing = 16 << 20

// Ref identifies a game server.
type Ref struct {
	Namespace string
	Name      string
}

// RefOf returns the reference of a game server.
func RefOf(gs *v1alpha1.GameServer) Ref {
	return Ref{Namespace: gs.Namespace, Name: gs.Name}
}

// Resolve converts a user supplied path into an absolute path below the server root.
func Resolve(rel string, denylist []string) (string, error) {
	clean := path.Clean("/" + strings.ReplaceAll(rel, "\\", "/"))
	if Denied(clean, denylist) {
		return "", ErrDenied
	}
	return path.Join(gameserver.ServerRoot, clean), nil
}

// Denied checks a root relative path ("/foo/bar") against egg file_denylist globs.
func Denied(clean string, denylist []string) bool {
	rel := strings.TrimPrefix(clean, "/")
	if rel == "" {
		return false
	}
	for _, pattern := range denylist {
		pattern = strings.TrimPrefix(strings.TrimSpace(pattern), "/")
		if pattern == "" || strings.HasPrefix(pattern, "!") {
			continue
		}
		if ok, _ := path.Match(pattern, rel); ok {
			return true
		}
		if ok, _ := path.Match(pattern, path.Base(rel)); ok {
			return true
		}
		// A denied directory denies everything inside it.
		if strings.HasPrefix(rel, strings.TrimSuffix(pattern, "/")+"/") {
			return true
		}
	}
	return false
}

// run executes script in the files pod of the server (with gameserver.PathScript); exit codes
// 3, 4 and 5 are ErrNotFound, ErrTooLarge and ErrExists.
func (m *Manager) run(
	ctx context.Context, server Ref, stdin io.Reader, stdout io.Writer, script string, args ...string,
) error {
	cmd := append([]string{"sh", "-c", gameserver.PathScript + script, scriptName(ctx)}, args...)
	pod := gameserver.FilesPodName(server.Name)
	err := m.Kube.Exec(ctx, server.Namespace, pod, gameserver.FilesContainerName, cmd, stdin, stdout)
	var exitErr *kube.ExitError
	if errors.As(err, &exitErr) {
		switch exitErr.Code {
		case 3:
			return ErrNotFound
		case 4:
			return ErrTooLarge
		case 5:
			return ErrExists
		}
		return errors.New(strings.TrimSpace(exitErr.Stderr))
	}
	return err
}

// checkLinks applies the denylist to where the paths really lead, so a symbolic link cannot get
// around it: follow resolves every link (files that are read or written), otherwise only the
// folder (an entry that is deleted or moved: a link itself). It returns the resolved paths
// relative to the root ("/plugins"). Without a denylist nothing is checked here (nil); the
// scripts keep every path inside the volume (inside).
func (m *Manager) checkLinks(
	ctx context.Context, server Ref, denylist []string, follow bool, abs ...string,
) ([]string, error) {
	if len(denylist) == 0 {
		return nil, nil
	}
	resolve := "entry"
	if follow {
		resolve = "real"
	}
	out := kube.LimitedBuffer{Max: maxListing}
	err := m.run(ctx, server, nil, &out, `for p; do printf '%s\0' "$(`+resolve+` "$p")"; done`, abs...)
	if err != nil {
		return nil, err
	}
	return checkReal(strings.Split(strings.TrimSuffix(out.String(), "\x00"), "\x00"), denylist)
}

// checkReal checks resolved paths: ErrNotFound outside the server root, ErrDenied on the
// denylist. It returns them relative to the root.
func checkReal(reals []string, denylist []string) ([]string, error) {
	rels := make([]string, 0, len(reals))
	for _, real := range reals {
		rel, ok := strings.CutPrefix(real, gameserver.ServerRoot)
		if !ok || (rel != "" && rel[0] != '/') {
			return nil, ErrNotFound
		}
		rel = path.Clean("/" + rel)
		if Denied(rel, denylist) {
			return nil, ErrDenied
		}
		rels = append(rels, rel)
	}
	return rels, nil
}

// realDir is the folder a checked directory really is (relative to the root): the entries of a
// folder behind a link are checked against the denylist where they really are.
func realDir(dir string, reals []string) string {
	if len(reals) > 0 {
		return reals[0]
	}
	return path.Clean("/" + dir)
}

// List returns the entries of a directory.
func (m *Manager) List(ctx context.Context, server Ref, dir string, denylist []string) ([]Entry, error) {
	abs, err := Resolve(dir, denylist)
	if err != nil {
		return nil, err
	}
	reals, err := m.checkLinks(ctx, server, denylist, true, abs)
	if err != nil {
		return nil, err
	}
	out := kube.LimitedBuffer{Max: maxListing}
	err = m.run(ctx, server, nil, &out, `[ -d "$1" ] || exit 3
inside "$(real "$1")"
cd "$1" || exit 3
for f in * .[!.]* ..?*; do
  [ -e "$f" ] || [ -L "$f" ] || continue
  stat -c '%F|%s|%Y|%a|%n' -- "$f"
done`, abs)
	if out.Full {
		return nil, ErrTooManyEntries
	}
	if err != nil {
		return nil, err
	}
	entries := parseListing(out.String(), func(name string) bool {
		return Denied(path.Join(realDir(dir, reals), name), denylist)
	})
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	return entries, nil
}

// parseListing reads the stat lines of List, without the entries skip reports.
func parseListing(out string, skip func(name string) bool) []Entry {
	var entries []Entry
	for line := range strings.SplitSeq(out, "\n") {
		parts := strings.SplitN(line, "|", 5)
		if len(parts) != 5 || skip(parts[4]) {
			continue
		}
		size, _ := strconv.ParseInt(parts[1], 10, 64)
		mtime, _ := strconv.ParseInt(parts[2], 10, 64)
		entries = append(entries, Entry{
			Name:       parts[4],
			Size:       size,
			Mode:       parts[3],
			ModifiedAt: time.Unix(mtime, 0).UTC(),
			IsDir:      parts[0] == "directory",
			IsSymlink:  parts[0] == "symbolic link",
		})
	}
	return entries
}

// Read returns the content of a file up to maxSize bytes. It reads one byte more to tell a file
// that is too large (also one that grows or is not a regular file behind a link).
func (m *Manager) Read(ctx context.Context, server Ref, file string, maxSize int, denylist []string) ([]byte, error) {
	abs, err := Resolve(file, denylist)
	if err != nil {
		return nil, err
	}
	if _, err := m.checkLinks(ctx, server, denylist, true, abs); err != nil {
		return nil, err
	}
	out := kube.LimitedBuffer{Max: maxSize + 1}
	err = m.run(ctx, server, nil, &out, `[ -f "$1" ] || exit 3
inside "$(real "$1")"
head -c "$2" "$1"`, abs, strconv.Itoa(out.Max))
	if out.Len() > maxSize {
		return nil, ErrTooLarge
	}
	return out.Bytes(), err
}

// Download streams a file to w.
func (m *Manager) Download(ctx context.Context, server Ref, file string, w io.Writer, denylist []string) error {
	abs, err := Resolve(file, denylist)
	if err != nil {
		return err
	}
	if _, err := m.checkLinks(ctx, server, denylist, true, abs); err != nil {
		return err
	}
	return m.run(ctx, server, nil, w, `[ -f "$1" ] || exit 3
inside "$(real "$1")"
cat -- "$1"`, abs)
}

// Write replaces a file with the content of r, creating parent directories.
func (m *Manager) Write(ctx context.Context, server Ref, file string, r io.Reader, denylist []string) error {
	abs, err := Resolve(file, denylist)
	if err != nil {
		return err
	}
	if abs == gameserver.ServerRoot {
		return ErrDenied
	}
	if _, err := m.checkLinks(ctx, server, denylist, true, abs); err != nil {
		return err
	}
	return m.run(ctx, server, r, nil, gameserver.WriteFileScript, abs)
}

// CreateFolder creates a directory.
func (m *Manager) CreateFolder(ctx context.Context, server Ref, dir string, denylist []string) error {
	abs, err := Resolve(dir, denylist)
	if err != nil {
		return err
	}
	if _, err := m.checkLinks(ctx, server, denylist, true, abs); err != nil {
		return err
	}
	return m.run(ctx, server, nil, nil, `inside "$(real "$1")"
mkdir -p -- "$1"`, abs)
}

// Delete removes files and directories (a link, not what it points to).
func (m *Manager) Delete(ctx context.Context, server Ref, paths []string, denylist []string) error {
	var abs []string
	for _, p := range paths {
		a, err := Resolve(p, denylist)
		if err != nil {
			return err
		}
		if a == gameserver.ServerRoot {
			return ErrDenied
		}
		abs = append(abs, a)
	}
	if len(abs) == 0 {
		return nil
	}
	if _, err := m.checkLinks(ctx, server, denylist, false, abs...); err != nil {
		return err
	}
	return m.run(ctx, server, nil, nil, `for p; do inside "$(entry "$p")"; done
rm -rf -- "$@"`, abs...)
}

// Rename moves a file or directory.
func (m *Manager) Rename(ctx context.Context, server Ref, from, to string, denylist []string) error {
	src, err := Resolve(from, denylist)
	if err != nil {
		return err
	}
	dst, err := Resolve(to, denylist)
	if err != nil {
		return err
	}
	if src == gameserver.ServerRoot || dst == gameserver.ServerRoot {
		return ErrDenied
	}
	if dst == src || strings.HasPrefix(dst, src+"/") {
		return ErrIntoItself
	}
	if _, err := m.checkLinks(ctx, server, denylist, false, src, dst); err != nil {
		return err
	}
	// Never overwrite: mv would replace a file or move into an existing folder.
	return m.run(ctx, server, nil, nil, `inside "$(entry "$1")"
inside "$(entry "$2")"
[ -e "$1" ] || [ -L "$1" ] || exit 3
[ -e "$2" ] || [ -L "$2" ] && exit 5
mkdir -p "$(dirname -- "$2")" && mv -- "$1" "$2"`, src, dst)
}

// NewArchiveName returns the name of a new archive of the file manager, e.g. "archive-2026-09-28T120000.tar.gz".
func NewArchiveName(now time.Time) string {
	return "archive-" + now.UTC().Format("2006-01-02T150405") + ".tar.gz"
}

// compressScript packs the entries "$3"… (each "./name") of folder "$1" into the archive "$2".
const compressScript = jobFuncs + `inside "$(real "$1")"
cd "$1" || exit 3
out="$2"; shift 2
pack "$out" "" "$@"`

// Compress creates the tar.gz archive of the given entries inside dir. tar stores links as links.
func (m *Manager) Compress(
	ctx context.Context, server Ref, dir, archive string, names []string, denylist []string, report func(Progress),
) error {
	absDir, err := Resolve(dir, denylist)
	if err != nil {
		return err
	}
	reals, err := m.checkLinks(ctx, server, denylist, true, absDir)
	if err != nil {
		return err
	}
	args := []string{absDir, archive}
	for _, n := range names {
		if strings.Contains(n, "/") || n == ".." || n == "." || n == "" {
			return ErrDenied
		}
		if Denied(path.Join(realDir(dir, reals), n), denylist) {
			return ErrDenied
		}
		args = append(args, "./"+n)
	}
	return m.run(ctx, server, nil, &progressWriter{report: report}, compressScript, args...)
}

// decompressScript extracts the archive "$1" into a temporary folder next to it and merges it
// from there into the archive's folder: entries cannot write through links or "../" outside of
// it, existing folders are merged and existing files overwritten, links in the way replaced.
const decompressScript = jobFuncs + `merge() {
  local f t
  for f in "$1"/* "$1"/.[!.]* "$1"/..?*; do
    [ -e "$f" ] || [ -L "$f" ] || continue
    t="$2/${f##*/}"
    if [ -d "$f" ] && [ ! -L "$f" ] && [ -d "$t" ] && [ ! -L "$t" ]; then
      merge "$f" "$t" || return 1
    else
      rm -rf "$t" && mv "$f" "$t" || return 1
    fi
  done
}
[ -f "$1" ] || exit 3
inside "$(real "$1")"
cd "$(dirname -- "$1")" || exit 3
tmp=$(mktemp -d .extract-XXXXXX) || exit 1
trap 'rm -rf "$tmp"' EXIT
unpack "$1" "$tmp" || exit 1
merge "$tmp" .`

// Decompress extracts a tar(.gz/.xz/.bz2) or zip archive into its directory.
func (m *Manager) Decompress(
	ctx context.Context, server Ref, file string, denylist []string, report func(Progress),
) error {
	abs, err := Resolve(file, denylist)
	if err != nil {
		return err
	}
	if _, err := m.checkLinks(ctx, server, denylist, true, abs); err != nil {
		return err
	}
	return m.run(ctx, server, nil, &progressWriter{report: report}, decompressScript, abs)
}
