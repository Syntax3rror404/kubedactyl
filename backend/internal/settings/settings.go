// Package settings manages the panel wide settings (PanelSettings resource) and the
// cluster objects they refer to: storage classes and Cilium load balancer IP pools.
package settings

import (
	"cmp"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8svalidation "k8s.io/apimachinery/pkg/util/validation"
	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/kube"
	"app/internal/sso"
	"app/internal/validation"
)

// Store reads and writes the PanelSettings object of the installation.
type Store struct {
	// Client writes; Reader reads directly from the API server (changes are visible at once).
	Client    client.Client
	Reader    client.Reader
	Namespace string
	// Limiter gets the Kubernetes API limits of saved settings (nil: not applied).
	Limiter *kube.RateLimiter
	// Changed is called after the settings changed (saved or edited with kubectl), for other caches built
	// from them (nil: none).
	Changed func()

	mu     sync.Mutex
	cached *v1alpha1.PanelSettingsSpec // nil: Current reads the settings again
	secret *string                     // nil: OIDCClientSecret reads the secret again
}

// Get returns the current settings with the default lifetimes filled in (only those when the
// object does not exist yet). It reads directly from the API server, so changes are visible at once.
func (s *Store) Get(ctx context.Context) (v1alpha1.PanelSettingsSpec, error) {
	obj := &v1alpha1.PanelSettings{}
	err := s.Reader.Get(ctx, client.ObjectKey{Namespace: s.Namespace, Name: v1alpha1.SettingsName}, obj)
	if err != nil && !apierrors.IsNotFound(err) {
		return obj.Spec, err
	}
	obj.Spec.SessionHours = cmp.Or(obj.Spec.SessionHours, DefaultSessionHours)
	obj.Spec.APITokenMaxDays = cmp.Or(obj.Spec.APITokenMaxDays, DefaultAPITokenMaxDays)
	obj.Spec.KubeAPIQPS = cmp.Or(obj.Spec.KubeAPIQPS, DefaultKubeAPIQPS)
	obj.Spec.KubeAPIUserQPS = cmp.Or(obj.Spec.KubeAPIUserQPS, DefaultKubeAPIUserQPS)
	obj.Spec.OIDC.UsernameClaim = cmp.Or(obj.Spec.OIDC.UsernameClaim, sso.DefaultUsernameClaim)
	obj.Spec.OIDC.GroupsClaim = cmp.Or(obj.Spec.OIDC.GroupsClaim, sso.DefaultGroupsClaim)
	return obj.Spec, nil
}

// Current returns the settings like Get, but keeps them in memory until they change: for values
// needed on every request (external domain, lifetimes, branding). A save through the store and a
// change of the object reported by the informer (Watch) make the next call read them again. The
// result is shared, do not change it.
func (s *Store) Current(ctx context.Context) (v1alpha1.PanelSettingsSpec, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cached != nil {
		return *s.cached, nil
	}
	spec, err := s.Get(ctx)
	if err == nil {
		s.cached = &spec
	}
	return spec, err
}

// Forget drops the kept settings; the next Current reads them again.
func (s *Store) Forget() {
	s.mu.Lock()
	s.cached, s.secret = nil, nil
	s.mu.Unlock()
	if s.Changed != nil {
		s.Changed()
	}
}

// Watch makes a change of the settings object made outside the panel (kubectl) reach Current: the
// informer of the shared cache reports it, no polling.
func (s *Store) Watch(ctx context.Context, c cache.Informers) error {
	inf, err := c.GetInformer(ctx, &v1alpha1.PanelSettings{})
	if err != nil {
		return err
	}
	forget := func(obj any) {
		if o, ok := obj.(client.Object); !ok || o.GetNamespace() == s.Namespace {
			s.Forget()
		}
	}
	_, err = inf.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
		AddFunc:    forget,
		UpdateFunc: func(_, obj any) { forget(obj) },
		DeleteFunc: forget,
	})
	return err
}

// Update validates the settings against the cluster and stores them.
func (s *Store) Update(ctx context.Context, spec v1alpha1.PanelSettingsSpec) (v1alpha1.PanelSettingsSpec, error) {
	spec, err := s.Validate(ctx, spec)
	if err != nil {
		return spec, err
	}
	obj := &v1alpha1.PanelSettings{}
	obj.Name, obj.Namespace = v1alpha1.SettingsName, s.Namespace
	err = kube.CreateOrPatch(ctx, s.Reader, s.Client, obj, func() error {
		obj.Spec = spec
		return nil
	})
	if err != nil {
		return spec, err
	}
	s.Forget() // Current reads the saved settings next
	if s.Limiter != nil {
		s.Limiter.SetLimits(int(cmp.Or(spec.KubeAPIQPS, DefaultKubeAPIQPS)),
			int(cmp.Or(spec.KubeAPIUserQPS, DefaultKubeAPIUserQPS)))
	}
	return spec, nil
}

// Lifetimes of sign-ins and API tokens when the settings name none, and their limits (the CRD
// validation has the same).
const (
	DefaultSessionHours    = 12
	DefaultAPITokenMaxDays = 90
	maxSessionHours        = 720
	maxAPITokenDays        = 3650
)

// Requests per second the panel sends to the Kubernetes API and one user may send to the panel
// when the settings name none, and the limits (the CRD validation has the same).
const (
	DefaultKubeAPIQPS     = 50
	minKubeAPIQPS         = 5
	maxKubeAPIQPS         = 1000
	DefaultKubeAPIUserQPS = 10
	minKubeAPIUserQPS     = 1
	maxKubeAPIUserQPS     = 200
)

// Lifetimes are the session and API token lifetimes with the defaults filled in.
type Lifetimes struct {
	Session  time.Duration
	APIToken time.Duration
	// APITokenDays is APIToken in days (the unit of the setting and of new tokens).
	APITokenDays int
}

// LifetimesOf fills in the defaults.
func LifetimesOf(spec v1alpha1.PanelSettingsSpec) Lifetimes {
	hours := cmp.Or(int(spec.SessionHours), DefaultSessionHours)
	days := cmp.Or(int(spec.APITokenMaxDays), DefaultAPITokenMaxDays)
	return Lifetimes{
		Session: time.Duration(hours) * time.Hour, APIToken: time.Duration(days) * 24 * time.Hour, APITokenDays: days,
	}
}

// Lifetimes reads the lifetimes with Current: they are checked on every request. When the
// settings cannot be read, the defaults apply.
func (s *Store) Lifetimes(ctx context.Context) Lifetimes {
	spec, _ := s.Current(ctx)
	return LifetimesOf(spec)
}

// Limits match the CRD validation.
const (
	maxNoticeLength  = 2000
	maxLegalLength   = 20000
	maxBrandName     = 40
	maxBrandTagline  = 80
	maxBrandImage    = 128 << 10
	maxBrandImageURL = 180000
)

// brandImage is a data URL of an image type browsers show as logo or favicon.
var brandImage = regexp.MustCompile(
	`^data:image/(png|jpeg|gif|webp|svg\+xml|x-icon|vnd\.microsoft\.icon);base64,([A-Za-z0-9+/]+=*)$`,
)

func invalid(field, format string, args ...any) error {
	return validation.Errors{field: fmt.Sprintf(format, args...)}
}

// Validate normalizes the settings and checks that every storage class and pool exists
// and can be used. Missing defaults are set to the first entry.
func (s *Store) Validate(ctx context.Context, spec v1alpha1.PanelSettingsSpec) (v1alpha1.PanelSettingsSpec, error) {
	spec, err := normalizeTexts(spec)
	if err != nil {
		return spec, err
	}
	if spec.SessionHours < 0 || spec.SessionHours > maxSessionHours {
		return spec, invalid("sessionHours", "must be between 1 and %d hours", maxSessionHours)
	}
	if spec.APITokenMaxDays < 0 || spec.APITokenMaxDays > maxAPITokenDays {
		return spec, invalid("apiTokenMaxDays", "must be between 1 and %d days", maxAPITokenDays)
	}
	if q := spec.KubeAPIQPS; q != 0 && (q < minKubeAPIQPS || q > maxKubeAPIQPS) {
		return spec, invalid("kubeApiQps", "must be between %d and %d per second", minKubeAPIQPS, maxKubeAPIQPS)
	}
	if q := spec.KubeAPIUserQPS; q < 0 || q > maxKubeAPIUserQPS {
		return spec, invalid("kubeApiUserQps", "must be between %d and %d per second",
			minKubeAPIUserQPS, maxKubeAPIUserQPS)
	}
	if spec.EggLibraries, err = normalizeEggLibraries(spec.EggLibraries); err != nil {
		return spec, err
	}
	if spec.OIDC, err = checkOIDC(ctx, spec.OIDC); err != nil {
		return spec, err
	}
	spec.StorageClasses = unique(spec.StorageClasses)
	spec.LoadBalancerPools = unique(spec.LoadBalancerPools)
	if err := s.checkStorageClasses(ctx, spec.StorageClasses); err != nil {
		return spec, err
	}
	if err := s.checkPools(ctx, spec.LoadBalancerPools); err != nil {
		return spec, err
	}
	spec.DefaultStorageClass = pickDefault(spec.DefaultStorageClass, spec.StorageClasses)
	spec.DefaultLoadBalancerPool = pickDefault(spec.DefaultLoadBalancerPool, spec.LoadBalancerPools)
	return spec, nil
}

// normalizeTexts trims domain, notice and legal texts and checks their length.
func normalizeTexts(spec v1alpha1.PanelSettingsSpec) (v1alpha1.PanelSettingsSpec, error) {
	spec.ExternalDomain = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(spec.ExternalDomain)), ".")
	if spec.ExternalDomain != "" {
		if errs := k8svalidation.IsDNS1123Subdomain(spec.ExternalDomain); len(errs) > 0 {
			return spec, invalid("externalDomain", "not a valid host name: %s", errs[0])
		}
	}
	spec.ServerNotice = strings.TrimSpace(spec.ServerNotice)
	if n := len([]rune(spec.ServerNotice)); n > maxNoticeLength {
		return spec, invalid("serverNotice", "the notice is too long (%d of %d characters)", n, maxNoticeLength)
	}
	spec.LegalNotice = strings.TrimSpace(spec.LegalNotice)
	spec.PrivacyPolicy = strings.TrimSpace(spec.PrivacyPolicy)
	for field, text := range map[string]string{"legalNotice": spec.LegalNotice, "privacyPolicy": spec.PrivacyPolicy} {
		if n := len([]rune(text)); n > maxLegalLength {
			return spec, invalid(field, "the text is too long (%d of %d characters)", n, maxLegalLength)
		}
	}
	return normalizeBranding(spec)
}

// normalizeBranding trims the brand texts and checks the logo and favicon images.
func normalizeBranding(spec v1alpha1.PanelSettingsSpec) (v1alpha1.PanelSettingsSpec, error) {
	spec.BrandName = strings.TrimSpace(spec.BrandName)
	spec.BrandTagline = strings.TrimSpace(spec.BrandTagline)
	for field, c := range map[string]struct {
		text string
		max  int
	}{"brandName": {spec.BrandName, maxBrandName}, "brandTagline": {spec.BrandTagline, maxBrandTagline}} {
		if n := len([]rune(c.text)); n > c.max {
			return spec, invalid(field, "too long (%d of %d characters)", n, c.max)
		}
	}
	for field, url := range map[string]string{"brandLogo": spec.BrandLogo, "favicon": spec.Favicon} {
		if err := checkBrandImage(url); err != nil {
			return spec, invalid(field, "%s", err)
		}
	}
	return spec, nil
}

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
			return nil, invalid("eggLibraries", "%q: %s", strings.TrimSpace(raw), err)
		}
		if u != "" {
			out = append(out, u)
		}
	}
	out = unique(out)
	if len(out) > maxEggLibraries {
		return nil, invalid("eggLibraries", "at most %d repositories", maxEggLibraries)
	}
	return out, nil
}

// checkBrandImage accepts an empty value or a base64 data URL of an image of at most 128 KiB.
func checkBrandImage(url string) error {
	if url == "" {
		return nil
	}
	errImage := errors.New("use a PNG, JPEG, GIF, WebP, SVG or ICO image of at most 128 KiB")
	m := brandImage.FindStringSubmatch(url)
	if m == nil || len(url) > maxBrandImageURL {
		return errImage
	}
	if data, err := base64.StdEncoding.DecodeString(m[2]); err != nil || len(data) == 0 || len(data) > maxBrandImage {
		return errImage
	}
	return nil
}

// checkStorageClasses requires at least one enabled class, and every one must exist.
func (s *Store) checkStorageClasses(ctx context.Context, names []string) error {
	if len(names) == 0 {
		return invalid("storageClasses", "enable at least one storage class")
	}
	classes, err := ListStorageClasses(ctx, s.Reader)
	if err != nil {
		return err
	}
	for _, name := range names {
		if !slices.ContainsFunc(classes, func(c StorageClass) bool { return c.Name == name }) {
			return invalid("storageClasses", "storage class %q does not exist", name)
		}
	}
	return nil
}

// checkPools requires every enabled pool to exist and to be usable (label selector).
func (s *Store) checkPools(ctx context.Context, names []string) error {
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
			return invalid("loadBalancerPools", "pool %q does not exist", name)
		}
		if !pools[i].Selectable {
			return invalid("loadBalancerPools", "pool %q cannot be used: %s", name, pools[i].Reason)
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
	// ErrNotConfigured: a server needs a setting that the admin has not made.
	ErrNotConfigured = errors.New("no storage class is enabled, configure the panel settings first")
	// ErrNotEnabled: the requested value is not among the enabled ones.
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
