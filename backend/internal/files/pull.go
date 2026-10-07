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

// pullScript downloads "$1" to "$2" under a temporary name. Every second it prints "pos N SIZE": the
// bytes written and the Content-Length of the last response (0 when the server sends none). The log
// of wget lies next to the file (the files pod has no writable /tmp) and goes into the error.
const pullScript = jobFuncs + `inside "$(real "$2")"
[ -e "$2" ] && exit 5
mkdir -p "$(dirname -- "$2")" || exit 1
tmp="$(dirname -- "$2")/.$(basename -- "$2").download"
trap 'rm -f "$tmp" "$tmp.log"' EXIT
wget -S -T 30 -O "$tmp" -- "$1" 2>"$tmp.log" &
job=$!
while kill -0 "$job" 2>/dev/null; do
  size=$(sed -n 's/^ *[Cc]ontent-[Ll]ength: *\([0-9]*\).*/\1/p' "$tmp.log" | tail -n 1)
  echo "pos $(stat -c %s "$tmp" 2>/dev/null || echo 0) ${size:-0}"
  sleep 1
done
wait "$job" || { echo "download failed: $(tail -n 1 "$tmp.log")" >&2; exit 1; }
mv "$tmp" "$2"`

// Pull downloads a URL into dir inside the files pod. The download runs in the pod, so
// the network isolation of the user namespace applies and the panel's own network access
// cannot be abused. Existing files are not overwritten.
func (m *Manager) Pull(
	ctx context.Context, server Ref, rawURL, dir, name string, denylist []string, report func(Progress),
) error {
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
	out := &progressWriter{report: report}
	return m.run(ctx, server, nil, out, pullScript, strings.TrimSpace(rawURL), target)
}
