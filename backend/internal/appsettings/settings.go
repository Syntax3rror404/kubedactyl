// Package appsettings holds the panel settings of Kubedactyl beyond the base (base/settings): the storage
// classes and load balancer pools of servers (and the cluster objects behind them), the external domain, the
// server notice, the subuser limit and the egg libraries.
package appsettings

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"

	k8svalidation "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/base/validation"
)

// Settings adds the fields of Kubedactyl to the panel settings (settings.App).
type Settings struct {
	// Reader reads the storage classes and pools directly from the API server.
	Reader client.Reader
}

// Subusers every user may create when the settings name none, and the limit (the CRD validation has the same).
const (
	DefaultSubusersPerUser = 10
	maxSubusersPerUser     = 100
	maxNoticeLength        = 2000
)

// SubusersPerUser fills in the default.
func SubusersPerUser(spec v1alpha1.PanelSettingsSpec) int {
	return int(ptr.Deref(spec.SubusersPerUser, DefaultSubusersPerUser))
}

// Defaults fills in the subuser limit.
func (Settings) Defaults(spec *v1alpha1.PanelSettingsSpec) {
	spec.SubusersPerUser = cmp.Or(spec.SubusersPerUser, ptr.To[int32](DefaultSubusersPerUser))
}

// Apply does nothing: the fields of Kubedactyl are read when they are needed.
func (Settings) Apply(v1alpha1.PanelSettingsSpec) {}

// ForUsers keeps what the web UI shows users: the address and pools of their servers, the server notice and
// their subuser limit.
func (Settings) ForUsers(spec v1alpha1.PanelSettingsSpec, out *v1alpha1.PanelSettingsSpec) {
	out.ExternalDomain = spec.ExternalDomain
	out.LoadBalancerPools = spec.LoadBalancerPools
	out.DefaultLoadBalancerPool = spec.DefaultLoadBalancerPool
	out.AllowClusterIP = spec.AllowClusterIP
	out.ServerNotice = spec.ServerNotice
	out.SubusersPerUser = spec.SubusersPerUser
}

// Validate normalizes the fields and checks that every storage class and pool exists and can be used. Missing
// defaults are set to the first entry.
func (s Settings) Validate(ctx context.Context, spec v1alpha1.PanelSettingsSpec) (v1alpha1.PanelSettingsSpec, error) {
	errs := validation.Errors{}
	spec.ExternalDomain = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(spec.ExternalDomain)), ".")
	if spec.ExternalDomain != "" {
		if msgs := k8svalidation.IsDNS1123Subdomain(spec.ExternalDomain); len(msgs) > 0 {
			errs["externalDomain"] = "not a valid host name: " + msgs[0]
		}
	}
	spec.ServerNotice = strings.TrimSpace(spec.ServerNotice)
	if n := len([]rune(spec.ServerNotice)); n > maxNoticeLength {
		errs["serverNotice"] = fmt.Sprintf("too long (%d of %d characters)", n, maxNoticeLength)
	}
	if n := spec.SubusersPerUser; n != nil && (*n < 0 || *n > maxSubusersPerUser) {
		errs["subusersPerUser"] = fmt.Sprintf("must be between 0 and %d", maxSubusersPerUser)
	}
	var err error
	if spec.EggLibraries, err = normalizeEggLibraries(spec.EggLibraries); err != nil {
		errs["eggLibraries"] = err.Error()
	}
	spec.StorageClasses = unique(spec.StorageClasses)
	spec.LoadBalancerPools = unique(spec.LoadBalancerPools)
	for field, check := range map[string]func() error{
		"storageClasses":    func() error { return s.checkStorageClasses(ctx, spec.StorageClasses) },
		"loadBalancerPools": func() error { return s.checkPools(ctx, spec.LoadBalancerPools) },
	} {
		var fe fieldErr
		if err := check(); errors.As(err, &fe) {
			errs[field] = fe.Error()
		} else if err != nil {
			return spec, err
		}
	}
	spec.DefaultStorageClass = pickDefault(spec.DefaultStorageClass, spec.StorageClasses)
	spec.DefaultLoadBalancerPool = pickDefault(spec.DefaultLoadBalancerPool, spec.LoadBalancerPools)
	return spec, errs.OrNil()
}

// fieldErr is a wrong value of the checked field (other errors: the cluster could not be read).
type fieldErr struct{ error }

const maxEggLibraries = 20

// ErrNotRepository is returned for an egg library that is not the URL of a git repository.
var ErrNotRepository = errors.New("not a git repository URL (https://<host>/<owner>/<repo>)")

// NormalizeEggLibrary trims a repository URL (also a trailing slash or ".git"). It takes the
// HTTP(S) URL of a repository on any host: owner and name, or more path segments (GitLab
// subgroups). An empty URL stays empty.
func NormalizeEggLibrary(raw string) (string, error) {
	s := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSpace(raw), "/"), ".git")
	if s == "" {
		return "", nil
	}
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil ||
		strings.ContainsAny(s, "?# ") || len(strings.Split(strings.TrimPrefix(u.Path, "/"), "/")) < 2 ||
		strings.Contains(u.Path, "//") {
		return "", ErrNotRepository
	}
	return s, nil
}

// normalizeEggLibraries normalizes the repository URLs and drops empty ones and duplicates.
func normalizeEggLibraries(urls []string) ([]string, error) {
	out := make([]string, 0, len(urls))
	for _, raw := range urls {
		u, err := NormalizeEggLibrary(raw)
		if err != nil {
			return nil, fmt.Errorf("%q: %s", strings.TrimSpace(raw), err)
		}
		if u != "" {
			out = append(out, u)
		}
	}
	out = unique(out)
	if len(out) > maxEggLibraries {
		return nil, fmt.Errorf("at most %d repositories", maxEggLibraries)
	}
	return out, nil
}

// checkStorageClasses requires at least one enabled class, and every one must exist.
func (s Settings) checkStorageClasses(ctx context.Context, names []string) error {
	if len(names) == 0 {
		return fieldErr{errors.New("enable at least one storage class")}
	}
	classes, err := ListStorageClasses(ctx, s.Reader)
	if err != nil {
		return err
	}
	for _, name := range names {
		if !slices.ContainsFunc(classes, func(c StorageClass) bool { return c.Name == name }) {
			return fieldErr{fmt.Errorf("storage class %q does not exist", name)}
		}
	}
	return nil
}

// checkPools requires every enabled pool to exist and to be usable (label selector).
func (s Settings) checkPools(ctx context.Context, names []string) error {
	if len(names) == 0 {
		return nil
	}
	pools, err := ListPools(ctx, s.Reader)
	if err != nil {
		return err
	}
	for _, name := range names {
		i := slices.IndexFunc(pools, func(p Pool) bool { return p.Name == name })
		if i < 0 {
			return fieldErr{fmt.Errorf("pool %q does not exist", name)}
		}
		if !pools[i].Selectable {
			return fieldErr{fmt.Errorf("pool %q cannot be used: %s", name, pools[i].Reason)}
		}
	}
	return nil
}

// Choose returns the requested value when it is allowed, the default when nothing was
// requested, or an error.
func Choose(requested, def string, allowed []string, what string) (string, error) {
	if requested == "" {
		return def, nil
	}
	if !slices.Contains(allowed, requested) {
		return "", fmt.Errorf("%s %q %w", what, requested, ErrNotEnabled)
	}
	return requested, nil
}

// Errors when a server asks for a storage class or pool.
var (
	// ErrNotConfigured means a server needs a setting that the admin has not made.
	ErrNotConfigured = errors.New("no storage class is enabled, configure the panel settings first")
	// ErrNotEnabled means the requested value is not among the enabled ones.
	ErrNotEnabled = errors.New("is not enabled in the panel settings")
)

func pickDefault(def string, list []string) string {
	if slices.Contains(list, def) {
		return def
	}
	if len(list) > 0 {
		return list[0]
	}
	return ""
}

func unique(in []string) []string {
	out := []string{}
	for _, v := range in {
		if v = strings.TrimSpace(v); v != "" && !slices.Contains(out, v) {
			out = append(out, v)
		}
	}
	return out
}
