package files

import (
	"context"
	"errors"
	"net/url"
	"path"
	"strings"
)

// Errors of downloads from a URL.
var (
	ErrBadURL       = errors.New("only http and https URLs can be downloaded")
	ErrBadName      = errors.New(`the file name must not contain slashes or be "." or ".."`)
	ErrNameRequired = errors.New("give a file name, the URL does not end with one")
)

// ValidName reports whether n can be the name of a file inside a folder (no path).
func ValidName(n string) bool {
	return n != "" && n != "." && n != ".." && !strings.ContainsAny(n, "/\\\x00")
}

// PullName returns the target file name: the given one, or the last path segment of the URL.
func PullName(rawURL, name string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", ErrBadURL
	}
	if name = strings.TrimSpace(name); name != "" {
		if !ValidName(name) {
			return "", ErrBadName
		}
		return name, nil
	}
	if name = path.Base(u.Path); name == "" || name == "." || name == "/" || name == ".." {
		return "", ErrNameRequired
	}
	return name, nil
}

// Pull downloads a URL into dir inside the files pod. The download runs in the pod, so
// the network isolation of the user namespace applies and the panel's own network access
// cannot be abused. Existing files are not overwritten.
func (m *Manager) Pull(ctx context.Context, server Ref, rawURL, dir, name string, denylist []string) error {
	name, err := PullName(rawURL, name)
	if err != nil {
		return err
	}
	target, err := Resolve(path.Join(dir, name), denylist)
	if err != nil {
		return err
	}
	if _, err := m.checkLinks(ctx, server, denylist, true, target); err != nil {
		return err
	}
	return m.run(ctx, server, nil, nil, `inside "$(real "$2")"
[ -e "$2" ] && exit 5
mkdir -p "$(dirname -- "$2")" || exit 1
tmp="$(dirname -- "$2")/.$(basename -- "$2").download"
if out=$(wget -T 30 -O "$tmp" -- "$1" 2>&1); then mv "$tmp" "$2"
else rm -f "$tmp"; echo "download failed: $(echo "$out" | tail -n 1)" >&2; exit 1; fi`,
		strings.TrimSpace(rawURL), target)
}
