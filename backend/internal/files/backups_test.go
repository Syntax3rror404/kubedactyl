package files

import (
	"testing"
	"time"
)

func TestBackupNames(t *testing.T) {
	now := time.Date(2026, 9, 29, 4, 0, 5, 0, time.UTC)
	if got := NewBackupName("", now); got != "backup-2026-09-29_040005.tar.gz" {
		t.Errorf("name = %q", got)
	}
	if got := NewBackupName(
		" before update / 1.21 ",
		now,
	); got != "backup-2026-09-29_040005-before-update-1-21.tar.gz" {
		t.Errorf("label = %q", got)
	}
	for name, want := range map[string]bool{
		"backup-2026-09-29_040005.tar.gz": true,
		"../etc/passwd.tar.gz":            false,
		"a/b.tar.gz":                      false,
		".hidden.tar.gz":                  false,
		"backup.zip":                      false,
		"":                                false,
	} {
		if ValidBackupName(name) != want {
			t.Errorf("ValidBackupName(%q) = %v", name, !want)
		}
	}
}

func TestFetchName(t *testing.T) {
	cases := []struct{ url, name, want string }{
		{"https://example.com/mods/plugin.jar", "", "plugin.jar"},
		{"https://example.com/download?id=1", "custom.zip", "custom.zip"},
		{"http://example.com/a/b/", "", "b"},
	}
	for _, c := range cases {
		if got, err := PullName(c.url, c.name); err != nil || got != c.want {
			t.Errorf("FetchName(%q, %q) = %q, %v", c.url, c.name, got, err)
		}
	}
	for _, bad := range [][2]string{
		{"ftp://example.com/x", ""}, {"file:///etc/passwd", ""}, {"javascript:alert(1)", ""},
		{"https://example.com", ""}, {"https://example.com/x", "../evil"}, {"https://example.com/x", "a/b"},
	} {
		if _, err := PullName(bad[0], bad[1]); err == nil {
			t.Errorf("FetchName(%q, %q) accepted", bad[0], bad[1])
		}
	}
}
