// Package files is the file manager of the servers: file operations, backups and background jobs
// run as scripts in the files pod of a server.
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

// Errors of file operations.
var (
	// ErrNotFound is returned for missing files or directories.
	ErrNotFound = errors.New("file not found")
	// ErrDenied is returned for paths outside the server root or on the egg denylist.
	ErrDenied = errors.New("access to this file is denied")
	// ErrExists is returned when the target of a move, rename or new archive already exists.
	ErrExists = errors.New("the target already exists")
	// ErrTooLarge is returned when a file is too big to be opened in the editor.
	ErrTooLarge = errors.New("file is too large to be edited")
	// ErrIntoItself is returned when a folder would be moved into itself.
	ErrIntoItself = errors.New("cannot move a folder into itself")
	// ErrTooManyEntries is returned for folders whose listing exceeds maxListing.
	ErrTooManyEntries = errors.New("the folder has too many entries to be listed")
	// ErrBadName is returned for entry names that are a path, "." or "..".
	ErrBadName = errors.New(`the file name must not contain slashes or be "." or ".."`)
)

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

// run executes script in the files pod of the server (with gameserver.PathScript); exit codes
// 3, 4 and 5 are ErrNotFound, ErrTooLarge and ErrExists.
func (m *Manager) run(
	ctx context.Context, ref Ref, stdin io.Reader, stdout io.Writer, script string, args ...string,
) error {
	cmd := append([]string{"sh", "-c", gameserver.PathScript + script, scriptName(ctx)}, args...)
	pod := gameserver.FilesPodName(ref.Name)
	err := m.Kube.Exec(ctx, ref.Namespace, pod, gameserver.FilesContainerName, cmd, stdin, stdout)
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

// listScript prints one stat line per entry of the folder "$1" (parseListing reads them).
const listScript = `[ -d "$1" ] || exit 3
inside "$(real "$1")"
cd "$1" || exit 3
for f in * .[!.]* ..?*; do
  [ -e "$f" ] || [ -L "$f" ] || continue
  stat -c '%F|%s|%Y|%a|%n' -- "$f"
done`

// listing runs a script that ends with listScript and returns its output.
func (m *Manager) listing(ctx context.Context, ref Ref, script, abs string) (string, error) {
	out := kube.LimitedBuffer{Max: maxListing}
	err := m.run(ctx, ref, nil, &out, script, abs)
	if out.Full {
		return "", ErrTooManyEntries
	}
	return out.String(), err
}

// List returns the entries of a directory.
func (m *Manager) List(ctx context.Context, ref Ref, dir string, denylist []string) ([]Entry, error) {
	abs, real, err := m.resolve(ctx, ref, dir, denylist)
	if err != nil {
		return nil, err
	}
	out, err := m.listing(ctx, ref, listScript, abs)
	if err != nil {
		return nil, err
	}
	entries := parseListing(out, func(name string) bool {
		return denied(path.Join(real, name), denylist)
	})
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	return entries, nil
}

// parseListing reads the stat lines of listScript, without the entries skip reports.
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
func (m *Manager) Read(ctx context.Context, ref Ref, file string, maxSize int, denylist []string) ([]byte, error) {
	abs, _, err := m.resolve(ctx, ref, file, denylist)
	if err != nil {
		return nil, err
	}
	out := kube.LimitedBuffer{Max: maxSize + 1}
	err = m.run(ctx, ref, nil, &out, `[ -f "$1" ] || exit 3
inside "$(real "$1")"
head -c "$2" "$1"`, abs, strconv.Itoa(out.Max))
	if out.Len() > maxSize {
		return nil, ErrTooLarge
	}
	return out.Bytes(), err
}

// Download streams a file to w.
func (m *Manager) Download(ctx context.Context, ref Ref, file string, w io.Writer, denylist []string) error {
	abs, _, err := m.resolve(ctx, ref, file, denylist)
	if err != nil {
		return err
	}
	return m.run(ctx, ref, nil, w, `[ -f "$1" ] || exit 3
inside "$(real "$1")"
cat -- "$1"`, abs)
}

// Write replaces a file with the content of r, creating parent directories.
func (m *Manager) Write(ctx context.Context, ref Ref, file string, r io.Reader, denylist []string) error {
	abs, err := Resolve(file, denylist)
	if err != nil {
		return err
	}
	if abs == gameserver.ServerRoot {
		return ErrDenied
	}
	if _, err := m.checkLinks(ctx, ref, denylist, true, abs); err != nil {
		return err
	}
	return m.run(ctx, ref, r, nil, gameserver.WriteFileScript, abs)
}

// CreateFolder creates a directory.
func (m *Manager) CreateFolder(ctx context.Context, ref Ref, dir string, denylist []string) error {
	abs, _, err := m.resolve(ctx, ref, dir, denylist)
	if err != nil {
		return err
	}
	return m.run(ctx, ref, nil, nil, `inside "$(real "$1")"
mkdir -p -- "$1"`, abs)
}

// Delete removes the entries names of the folder dir (a link, not what it points to).
func (m *Manager) Delete(ctx context.Context, ref Ref, dir string, names []string, denylist []string) error {
	abs := make([]string, 0, len(names))
	for _, n := range names {
		if !ValidName(n) {
			return ErrBadName
		}
		a, err := Resolve(path.Join(dir, n), denylist)
		if err != nil {
			return err
		}
		abs = append(abs, a)
	}
	if len(abs) == 0 {
		return nil
	}
	if _, err := m.checkLinks(ctx, ref, denylist, false, abs...); err != nil {
		return err
	}
	return m.run(ctx, ref, nil, nil, `for p; do inside "$(entry "$p")"; done
rm -rf -- "$@"`, abs...)
}

// Rename moves a file or directory. A folder moves only when nothing in it is denied where it is
// or would be afterwards.
func (m *Manager) Rename(ctx context.Context, ref Ref, from, to string, denylist []string) error {
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
	reals, err := m.checkLinks(ctx, ref, denylist, false, src, dst)
	if err != nil {
		return err
	}
	if err := m.checkTree(ctx, ref, denylist, src, reals, "."); err != nil {
		return err
	}
	// Never overwrite: mv would replace a file or move into an existing folder.
	return m.run(ctx, ref, nil, nil, `inside "$(entry "$1")"
inside "$(entry "$2")"
[ -e "$1" ] || [ -L "$1" ] || exit 3
[ -e "$2" ] || [ -L "$2" ] && exit 5
mkdir -p "$(dirname -- "$2")" && mv -- "$1" "$2"`, src, dst)
}
