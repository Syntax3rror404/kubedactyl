package files

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"app/internal/gameserver"
)

// BackupDir holds the backups inside the server volume. Backups pack everything else and a
// restore replaces everything else, so the folder itself is never part of a backup.
const BackupDir = ".backups"

var backupName = regexp.MustCompile(`^[A-Za-z0-9._-]+\.tar\.gz$`)

// labelChars are the characters a backup label must not contain.
var labelChars = regexp.MustCompile(`[^A-Za-z0-9_-]+`)

// NewBackupName returns "backup-2026-09-29_0400[-label].tar.gz".
func NewBackupName(label string, now time.Time) string {
	name := "backup-" + now.Format("2006-01-02_150405")
	if l := strings.Trim(labelChars.ReplaceAllString(label, "-"), "-"); l != "" {
		if len(l) > 40 {
			l = l[:40]
		}
		name += "-" + l
	}
	return name + ".tar.gz"
}

// ValidBackupName rejects paths and anything that is not a backup archive name.
func ValidBackupName(name string) bool {
	return backupName.MatchString(name) && !strings.HasPrefix(name, ".")
}

// Backups lists the backup archives (newest first).
func (m *Manager) Backups(ctx context.Context, server Ref) ([]Entry, error) {
	if err := m.run(ctx, server, nil, nil, `inside "$(real "$1")"
mkdir -p "$1"`, backupDirPath()); err != nil {
		return nil, err
	}
	entries, err := m.List(ctx, server, "/"+BackupDir, nil)
	if err != nil {
		return nil, err
	}
	out := entries[:0]
	for _, e := range entries {
		if !e.IsDir && ValidBackupName(e.Name) {
			out = append(out, e)
		}
	}
	// Names start with the timestamp, so the reversed alphabetical order is newest first.
	slices.Reverse(out)
	return out, nil
}

// CreateBackup packs everything except the backup folder into a new archive. It is written
// under a temporary name first, so an aborted backup never looks complete.
func (m *Manager) CreateBackup(ctx context.Context, server Ref, name string) error {
	if !ValidBackupName(name) {
		return ErrDenied
	}
	return m.run(ctx, server, nil, nil, `inside "$(real "$1/$2")"
cd "$1" || exit 3
mkdir -p "$2" || exit 1
tmp="$2/.$3.partial"
if tar -czf "$tmp" --exclude="./$2" . ; then mv "$tmp" "$2/$3"; else rm -f "$tmp"; exit 1; fi`,
		gameserver.ServerRoot, BackupDir, name)
}

// RestoreBackup deletes everything except the backup folder and extracts the archive.
// The server must be stopped.
func (m *Manager) RestoreBackup(ctx context.Context, server Ref, name string) error {
	if !ValidBackupName(name) {
		return ErrNotFound
	}
	return m.run(ctx, server, nil, nil, `inside "$(real "$1/$2/$3")"
cd "$1" || exit 3
[ -f "$2/$3" ] || exit 3
find "$1" -mindepth 1 -maxdepth 1 ! -name "$2" -exec rm -rf {} +
tar -xzf "$2/$3" -C "$1"`,
		gameserver.ServerRoot, BackupDir, name)
}

// DeleteBackup removes one archive.
func (m *Manager) DeleteBackup(ctx context.Context, server Ref, name string) error {
	if !ValidBackupName(name) {
		return ErrNotFound
	}
	return m.run(ctx, server, nil, nil, `inside "$(entry "$1")"
[ -f "$1" ] || exit 3
rm -f -- "$1"`, backupDirPath()+"/"+name)
}

// StartBackup creates a backup in the background; label and now give its name.
func (s *Service) StartBackup(ref Ref, label string, now time.Time) (Job, error) {
	name := NewBackupName(label, now)
	return s.Start(ref, JobBackup, name, func(ctx context.Context) error {
		return s.Manager.CreateBackup(ctx, ref, name)
	})
}

// RunBackup is StartBackup that waits for the backup (used by schedules).
func (s *Service) RunBackup(ctx context.Context, ref Ref, label string, now time.Time) error {
	name := NewBackupName(label, now)
	return s.Run(ctx, ref, JobBackup, name, func(ctx context.Context) error {
		return s.Manager.CreateBackup(ctx, ref, name)
	})
}

// StartRestore restores a backup in the background. The server must be stopped.
func (s *Service) StartRestore(ref Ref, name string) (Job, error) {
	if !ValidBackupName(name) {
		return Job{}, ErrNotFound
	}
	return s.Start(ref, JobRestore, name, func(ctx context.Context) error {
		return s.Manager.RestoreBackup(ctx, ref, name)
	})
}

// DeleteBackup removes a backup unless a restore is running.
func (s *Service) DeleteBackup(ctx context.Context, ref Ref, name string) error {
	if s.Busy(ref, JobRestore) {
		return ErrRestoring
	}
	return s.Manager.DeleteBackup(ctx, ref, name)
}

func backupDirPath() string { return gameserver.ServerRoot + "/" + BackupDir }

// BackupPath is the file manager path of a backup (for downloads).
func BackupPath(name string) string { return fmt.Sprintf("/%s/%s", BackupDir, name) }
