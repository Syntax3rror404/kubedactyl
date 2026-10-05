package egg

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestScanDirectory parses every egg below $EGG_SCAN_DIR (skipped when unset).
func TestScanDirectory(t *testing.T) {
	dir := os.Getenv("EGG_SCAN_DIR")
	if dir == "" {
		t.Skip("EGG_SCAN_DIR not set")
	}
	var ok, failed int
	formats := map[string]int{}
	parsers := map[string]int{}
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".github" {
				return filepath.SkipDir
			}
			return nil
		}
		base := d.Name()
		egg := strings.HasPrefix(base, "egg-") || strings.HasPrefix(base, "pterodactyl-egg-")
		if ext := filepath.Ext(base); !egg || (ext != ".json" && ext != ".yaml" && ext != ".yml") {
			return nil
		}
		data, _ := os.ReadFile(path)
		spec, err := Parse(data)
		if err != nil {
			failed++
			t.Errorf("%s: %v", path, err)
			return nil
		}
		ok++
		formats[spec.Source.Format]++
		for _, f := range spec.ConfigFiles {
			parsers[f.Parser]++
		}
		return nil
	})
	t.Logf("parsed=%d failed=%d formats=%v parsers=%v", ok, failed, formats, parsers)
}
