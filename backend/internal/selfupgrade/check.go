// Package selfupgrade lets the panel upgrade its own Helm release: it looks for newer chart
// versions in the OCI registry and runs `helm upgrade` in a Job, with the release's values
// and only a new chart version.
package selfupgrade

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"
	"helm.sh/helm/v4/pkg/registry"
)

// CheckInterval is how often the registry is asked for new chart versions.
const CheckInterval = 10 * time.Minute

// Config describes the Helm release of the panel. Self-upgrades are off without a chart.
type Config struct {
	// Chart is the OCI reference of the chart without tag, e.g. oci://ghcr.io/x/charts/kubedactyl.
	Chart string
	// Release is the Helm release name; Namespace is the release (= panel) namespace.
	Release   string
	Namespace string
	// ServiceAccount runs the upgrade job (it may change the release's resources).
	ServiceAccount string
	// Image is the panel image; the job runs `kubedactyl upgrade` with it.
	Image string
	// Current is the version of this panel (chart version and app version are the same).
	Current string
}

// Enabled reports whether everything needed for self-upgrades is configured.
func (c Config) Enabled() bool {
	return c.Chart != "" && c.Release != "" && c.Namespace != "" && c.ServiceAccount != "" && c.Image != ""
}

// registryRef turns "oci://host/path" into "host/path" (what the registry client expects).
func (c Config) registryRef() string { return strings.TrimPrefix(c.Chart, "oci://") }

// Check is the result of the last registry check.
type Check struct {
	CheckedAt time.Time
	// Latest is the highest stable chart version in the registry.
	Latest string
	// Newer lists the stable versions above the current one, highest first.
	Newer []string
	Error string
}

// Checker asks the registry for chart versions every CheckInterval. It runs as a manager
// runnable; without internet the check fails quietly and the panel keeps working.
type Checker struct {
	Config Config
	Log    *slog.Logger

	mu   sync.Mutex
	last Check
	// tags is replaced in tests.
	tags func(ref string) ([]string, error)
}

// Start implements manager.Runnable.
func (c *Checker) Start(ctx context.Context) error {
	c.Refresh()
	t := time.NewTicker(CheckInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			c.Refresh()
		}
	}
}

// Last returns the result of the last check.
func (c *Checker) Last() Check {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.last
}

// Refresh asks the registry now.
func (c *Checker) Refresh() Check {
	res := Check{CheckedAt: time.Now()}
	tags, err := c.listTags()
	if err != nil {
		res.Error = err.Error()
		c.Log.Debug("checking for panel updates", "chart", c.Config.Chart, "err", err)
	} else {
		res.Latest, res.Newer = newerVersions(tags, c.Config.Current)
	}
	c.mu.Lock()
	c.last = res
	c.mu.Unlock()
	return res
}

func (c *Checker) listTags() ([]string, error) {
	if c.tags != nil {
		return c.tags(c.Config.registryRef())
	}
	rc, err := newRegistryClient()
	if err != nil {
		return nil, err
	}
	return rc.Tags(c.Config.registryRef())
}

// newRegistryClient creates an anonymous registry client. The credentials file only has to
// be a path: the root file system of the panel is read-only and nothing is stored.
func newRegistryClient() (*registry.Client, error) {
	return registry.NewClient(
		registry.ClientOptCredentialsFile(filepath.Join(os.TempDir(), "kubedactyl-registry.json")),
	)
}

// newerVersions returns the highest stable version and the stable versions above current
// (highest first). Pre-releases are left out; a current version that is no semantic version
// (a dev build) has no newer versions.
func newerVersions(tags []string, current string) (string, []string) {
	cur, curErr := semver.StrictNewVersion(current)
	var latest *semver.Version
	var newer []*semver.Version
	for _, t := range tags {
		v, err := semver.StrictNewVersion(t)
		if err != nil || v.Prerelease() != "" {
			continue
		}
		if latest == nil || v.GreaterThan(latest) {
			latest = v
		}
		if curErr == nil && v.GreaterThan(cur) {
			newer = append(newer, v)
		}
	}
	out := make([]string, 0, len(newer))
	sort.Sort(sort.Reverse(semver.Collection(newer)))
	for _, v := range newer {
		out = append(out, v.Original())
	}
	if latest == nil {
		return "", out
	}
	return latest.Original(), out
}
