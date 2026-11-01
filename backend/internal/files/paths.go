package files

import (
	"context"
	"path"
	"strings"

	"app/internal/gameserver"
	"app/internal/kube"
)

// ValidName reports whether n can be the name of a file inside a folder (no path).
func ValidName(n string) bool {
	return n != "" && n != "." && n != ".." && !strings.ContainsAny(n, "/\\\x00")
}

// Resolve converts a user supplied path into an absolute path below the server root.
func Resolve(rel string, denylist []string) (string, error) {
	clean := path.Clean("/" + strings.ReplaceAll(rel, "\\", "/"))
	if denied(clean, denylist) {
		return "", ErrDenied
	}
	return path.Join(gameserver.ServerRoot, clean), nil
}

// denied checks a root relative path ("/foo/bar") against the egg's file_denylist, read like a
// .gitignore file: a pattern without a slash inside matches a name at any depth, one with a
// slash matches from the root, "**" matches any number of folders, a trailing slash denies what
// is inside the folder, and a denied folder denies everything inside it. Negations ("!") are not
// supported and skipped.
func denied(clean string, denylist []string) bool {
	rel := strings.TrimPrefix(clean, "/")
	if rel == "" {
		return false
	}
	segs := strings.Split(rel, "/")
	for _, pattern := range denylist {
		pat, ok := denyPattern(pattern)
		if !ok {
			return true // Fail-safe: a malformed pattern denies everything.
		}
		// A match of a parent folder denies everything inside it.
		for i := range segs {
			if pat != nil && matchSegments(pat, segs[:i+1]) {
				return true
			}
		}
	}
	return false
}

// denyPattern splits a denylist pattern into its segments; nil for empty patterns and
// negations. ok is false for a malformed pattern.
func denyPattern(pattern string) (segs []string, ok bool) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" || strings.HasPrefix(pattern, "!") {
		return nil, true
	}
	if _, err := path.Match(pattern, ""); err != nil {
		return nil, false
	}
	inside := strings.HasSuffix(pattern, "/")
	if pattern = strings.Trim(pattern, "/"); pattern == "" {
		return nil, true
	}
	segs = strings.Split(pattern, "/")
	if len(segs) == 1 {
		segs = []string{"**", pattern}
	}
	if inside {
		segs = append(segs, "**")
	}
	return segs, true
}

// matchSegments matches path segments against pattern segments; "**" matches any number of
// segments, at the end at least one (everything inside a folder).
func matchSegments(pat, segs []string) bool {
	if len(pat) == 0 {
		return len(segs) == 0
	}
	if pat[0] == "**" {
		if len(pat) == 1 {
			return len(segs) > 0
		}
		for i := range len(segs) + 1 {
			if matchSegments(pat[1:], segs[i:]) {
				return true
			}
		}
		return false
	}
	if len(segs) == 0 {
		return false
	}
	ok, _ := path.Match(pat[0], segs[0])
	return ok && matchSegments(pat[1:], segs[1:])
}

// checkLinks applies the denylist to where the paths really lead, so a symbolic link cannot get
// around it. With follow every link is resolved (a file that is read or written), otherwise only
// the folder, because deleting or moving an entry touches the link itself. It returns the resolved
// paths relative to the root ("/plugins"), or nil without a denylist (the scripts keep every path
// inside the volume anyway).
func (m *Manager) checkLinks(
	ctx context.Context, ref Ref, denylist []string, follow bool, abs ...string,
) ([]string, error) {
	if len(denylist) == 0 {
		return nil, nil
	}
	resolve := "entry"
	if follow {
		resolve = "real"
	}
	out := kube.LimitedBuffer{Max: maxListing}
	err := m.run(ctx, ref, nil, &out, `for p; do printf '%s\0' "$(`+resolve+` "$p")"; done`, abs...)
	if err != nil {
		return nil, err
	}
	return checkReal(splitNul(out.String()), denylist)
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
		if denied(rel, denylist) {
			return nil, ErrDenied
		}
		rels = append(rels, rel)
	}
	return rels, nil
}

// treeScript prints every path below the entries "$2"… ("./name") of the folder "$1", NUL
// separated and without following links. A link or file as folder has nothing below it.
const treeScript = `[ -d "$1" ] && [ ! -L "$1" ] || exit 0
inside "$(real "$1")"
cd "$1" || exit 3
shift
for p; do [ -e "$p" ] || [ -L "$p" ] || exit 3; done
find "$@" -exec printf '%s\0' {} +`

// checkTree applies the denylist to everything below the entries names of the folder dir: each
// path ("name/sub/file") must not be denied below any of the folders in roots (relative to the
// root), the places it is now or will be. Without a denylist it does nothing.
func (m *Manager) checkTree(
	ctx context.Context, ref Ref, denylist []string, dir string, roots []string, names ...string,
) error {
	if len(denylist) == 0 {
		return nil
	}
	out := kube.LimitedBuffer{Max: maxListing}
	err := m.run(ctx, ref, nil, &out, treeScript, append([]string{dir}, names...)...)
	if out.Full {
		return ErrTooManyEntries
	}
	if err != nil {
		return err
	}
	return checkBelow(splitNul(out.String()), roots, denylist)
}

// checkBelow returns ErrDenied when one of the paths is denied below one of the roots.
func checkBelow(paths, roots, denylist []string) error {
	for _, p := range paths {
		for _, root := range roots {
			if denied(path.Join(root, p), denylist) {
				return ErrDenied
			}
		}
	}
	return nil
}

// splitNul splits NUL terminated script output.
func splitNul(out string) []string {
	if out == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(out, "\x00"), "\x00")
}

// resolve is Resolve plus checkLinks for one path that is followed. It also returns where the path
// really leads (realDir).
func (m *Manager) resolve(ctx context.Context, ref Ref, rel string, denylist []string) (string, string, error) {
	abs, err := Resolve(rel, denylist)
	if err != nil {
		return "", "", err
	}
	reals, err := m.checkLinks(ctx, ref, denylist, true, abs)
	if err != nil {
		return "", "", err
	}
	return abs, realDir(rel, reals), nil
}

// realDir is where a checked path really leads (relative to the root): the entries of a folder
// behind a link are checked against the denylist where they really are.
func realDir(rel string, reals []string) string {
	if len(reals) > 0 {
		return reals[0]
	}
	return path.Clean("/" + rel)
}
