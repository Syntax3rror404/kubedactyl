package files

import (
	"context"
	"crypto/rand"
	"path"
	"time"

	"app/internal/gameserver"
)

// NewArchiveName returns the name of a new archive of the file manager, e.g. "archive-2026-09-28T120000.tar.gz".
func NewArchiveName(now time.Time) string {
	return "archive-" + now.UTC().Format("2006-01-02T150405") + ".tar.gz"
}

// compressScript packs the entries "$3"… (each "./name") of folder "$1" into the archive "$2".
const compressScript = jobFuncs + `inside "$(real "$1")"
cd "$1" || exit 3
sweep .
out="$2"; shift 2
pack "$out" "" "$@"`

// Compress creates the tar.gz archive of the given entries inside dir. tar stores links as links.
// Nothing denied may be packed, also not inside a folder.
func (m *Manager) Compress(
	ctx context.Context, ref Ref, dir, archive string, names []string, denylist []string, report func(Progress),
) error {
	absDir, real, err := m.resolve(ctx, ref, dir, denylist)
	if err != nil {
		return err
	}
	entries := make([]string, 0, len(names))
	for _, n := range names {
		if !ValidName(n) {
			return ErrBadName
		}
		if denied(path.Join(real, n), denylist) {
			return ErrDenied
		}
		entries = append(entries, "./"+n)
	}
	err = m.checkTree(ctx, ref, denylist, path.Join(gameserver.ServerRoot, real), []string{real}, entries...)
	if err != nil {
		return err
	}
	args := append([]string{absDir, archive}, entries...)
	return m.run(ctx, ref, nil, &progressWriter{report: report}, compressScript, args...)
}

// unpackScript extracts the archive "$1" into the new temporary folder "$2" next to it; the
// folder is removed when the extraction fails.
const unpackScript = jobFuncs + `[ -f "$1" ] || exit 3
inside "$(real "$1")"
inside "$(entry "$2")"
cd "$(dirname -- "$1")" || exit 3
sweep .
mkdir "$2" || exit 1
trap 'rm -rf "$2"' EXIT
unpack "$1" "$2" || exit 1
trap - EXIT`

// mergeScript moves the extracted entries of the temporary folder "$1" into the folder around it
// and removes it: entries cannot write through links or "../" outside of it, existing folders are
// merged and existing files overwritten, links in the way replaced.
const mergeScript = jobFuncs + `merge() {
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
inside "$(entry "$1")"
cd "$(dirname -- "$1")" || exit 3
trap 'rm -rf "$1"' EXIT
merge "$1" .`

// Decompress extracts a tar(.gz/.xz/.bz2) or zip archive into its directory.
//  1. Unpack: the archive is extracted into a temporary folder next to it.
//  2. Check: nothing extracted may be denied where it would land, else nothing is merged.
//  3. Merge: the entries move into the archive's folder.
func (m *Manager) Decompress(
	ctx context.Context, ref Ref, file string, denylist []string, report func(Progress),
) error {
	abs, real, err := m.resolve(ctx, ref, file, denylist)
	if err != nil {
		return err
	}
	// 1. Unpack
	tmp := path.Join(path.Dir(abs), ".extract-"+rand.Text())
	if err := m.run(ctx, ref, nil, &progressWriter{report: report}, unpackScript, abs, tmp); err != nil {
		return err
	}
	// 2. Check
	err = m.checkTree(ctx, ref, denylist, tmp, []string{path.Dir(real)}, ".")
	// 3. Merge
	if err == nil {
		err = m.run(ctx, ref, nil, nil, mergeScript, tmp)
	}
	if err != nil {
		// Also after a cancel, so the extracted files do not stay behind.
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), cancelGrace)
		defer cancel()
		_ = m.run(cleanup, ref, nil, nil, `inside "$(entry "$1")"
rm -rf -- "$1"`, tmp)
	}
	return err
}

// StartCompress packs the entries names of the folder dir into a new archive in the background;
// the job's label is the archive's path.
func (s *Service) StartCompress(ref Ref, dir string, names []string, denylist []string) (Job, error) {
	if _, err := Resolve(dir, denylist); err != nil {
		return Job{}, err
	}
	archive := NewArchiveName(time.Now())
	return s.Start(ref, JobCompress, path.Join(dir, archive), func(ctx context.Context, report func(Progress)) error {
		return s.Manager.Compress(ctx, ref, dir, archive, names, denylist, report)
	})
}

// StartDecompress extracts the archive file in the background; the job's label is its path.
func (s *Service) StartDecompress(ref Ref, file string, denylist []string) (Job, error) {
	if _, err := Resolve(file, denylist); err != nil {
		return Job{}, err
	}
	return s.Start(ref, JobDecompress, file, func(ctx context.Context, report func(Progress)) error {
		return s.Manager.Decompress(ctx, ref, file, denylist, report)
	})
}
