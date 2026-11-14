// Package settings manages the panel wide settings (the PanelSettings resource): sign-in lifetimes, request
// limits, branding, legal texts and single sign-on. The app adds its own fields through App.
package settings

import (
	"cmp"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/app"
	"app/base/auth"
	"app/base/kube"
	"app/base/validation"
)

// App is what the app adds to the settings: its defaults, the checks of its fields, what it does with saved
// settings and the fields users who are no administrators may read.
type App interface {
	// Defaults fills in the defaults of the app's fields.
	Defaults(spec *v1alpha1.PanelSettingsSpec)
	// Validate normalizes and checks the app's fields; wrong values are validation.Errors.
	Validate(ctx context.Context, spec v1alpha1.PanelSettingsSpec) (v1alpha1.PanelSettingsSpec, error)
	// Apply gives the app saved settings (also the stored ones at start).
	Apply(spec v1alpha1.PanelSettingsSpec)
	// ForUsers copies the app's fields users may read from spec to out.
	ForUsers(spec v1alpha1.PanelSettingsSpec, out *v1alpha1.PanelSettingsSpec)
}

// Store reads and writes the PanelSettings object of the installation.
type Store struct {
	// Client writes; Reader reads directly from the API server (changes are visible at once).
	Client    client.Client
	Reader    client.Reader
	Namespace string
	// Limiter gets the Kubernetes API limits of saved settings (nil: not applied).
	Limiter *kube.RateLimiter
	// App adds the app's fields (nil: none).
	App App
	// Changed is called after the settings changed (saved or edited with kubectl), for other caches built
	// from them (nil: none).
	Changed func()

	spec   memo[v1alpha1.PanelSettingsSpec] // kept by Current
	secret memo[string]                     // kept by OIDCClientSecret
}

// Get returns the current settings with every default filled in (also when the object does not exist yet). It
// reads directly from the API server, so changes are visible at once.
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
	obj.Spec.OIDC.UsernameClaim = cmp.Or(obj.Spec.OIDC.UsernameClaim, auth.DefaultUsernameClaim)
	obj.Spec.OIDC.GroupsClaim = cmp.Or(obj.Spec.OIDC.GroupsClaim, auth.DefaultGroupsClaim)
	if s.App != nil {
		s.App.Defaults(&obj.Spec)
	}
	return obj.Spec, nil
}

// Current returns the settings like Get, but keeps them in memory until they change: for values
// needed on every request (lifetimes, branding). A save through the store and a change of the object
// reported by the informer (Watch) make the next call read them again; calls during that read share it.
// The result is shared, do not change it.
func (s *Store) Current(ctx context.Context) (v1alpha1.PanelSettingsSpec, error) {
	return s.spec.get(ctx, s.Get)
}

// Forget drops the kept settings; the next Current reads them again.
func (s *Store) Forget() {
	s.spec.forget()
	s.secret.forget()
	if s.Changed != nil {
		s.Changed()
	}
}

// Start applies the stored settings (Apply) and keeps Current up to date with changes made outside the panel
// (Watch).
func (s *Store) Start(ctx context.Context, c cache.Informers) error {
	spec, err := s.Get(ctx)
	if err != nil {
		return fmt.Errorf("reading panel settings: %w", err)
	}
	s.Apply(spec)
	if err := s.Watch(ctx, c); err != nil {
		return fmt.Errorf("watching panel settings: %w", err)
	}
	return nil
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

// Update validates the settings and stores them.
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
	s.Apply(spec)
	return spec, nil
}

// Apply gives the Limiter the Kubernetes API limits of the settings (the defaults where they name none) and the
// app the settings.
func (s *Store) Apply(spec v1alpha1.PanelSettingsSpec) {
	if s.Limiter != nil {
		s.Limiter.SetLimits(int(cmp.Or(spec.KubeAPIQPS, DefaultKubeAPIQPS)),
			int(cmp.Or(spec.KubeAPIUserQPS, DefaultKubeAPIUserQPS)))
	}
	if s.App != nil {
		s.App.Apply(spec)
	}
}

// ForUsers keeps the settings users who are no administrators may read: the API docs switch, their limit for API
// tokens and the app's fields for them.
func (s *Store) ForUsers(spec v1alpha1.PanelSettingsSpec) v1alpha1.PanelSettingsSpec {
	out := v1alpha1.PanelSettingsSpec{DisableAPIDocs: spec.DisableAPIDocs, APITokenMaxDays: spec.APITokenMaxDays}
	if s.App != nil {
		s.App.ForUsers(spec, &out)
	}
	return out
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
	DefaultKubeAPIUserQPS = app.KubeAPIUserQPS
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

// Validate trims the texts and checks the fields of the base and of the app, then the single sign-on settings
// (while it is on, the identity provider must answer). Every wrong field is reported at once.
func (s *Store) Validate(ctx context.Context, spec v1alpha1.PanelSettingsSpec) (v1alpha1.PanelSettingsSpec, error) {
	spec.LegalNotice = strings.TrimSpace(spec.LegalNotice)
	spec.PrivacyPolicy = strings.TrimSpace(spec.PrivacyPolicy)
	spec.BrandName = strings.TrimSpace(spec.BrandName)
	spec.BrandTagline = strings.TrimSpace(spec.BrandTagline)
	errs := checkFields(spec)
	if s.App != nil {
		var err error
		spec, err = s.App.Validate(ctx, spec)
		var appErrs validation.Errors
		if err != nil && !errors.As(err, &appErrs) {
			return spec, err
		}
		for field, msg := range appErrs {
			errs[field] = msg
		}
	}
	if err := errs.OrNil(); err != nil {
		return spec, err
	}
	var err error
	spec.OIDC, err = normalizeOIDC(ctx, spec.OIDC)
	return spec, err
}

// checkFields checks the ranges, text lengths and images of the base fields.
func checkFields(spec v1alpha1.PanelSettingsSpec) validation.Errors {
	errs := validation.Errors{}
	if spec.SessionHours < 0 || spec.SessionHours > maxSessionHours {
		errs["sessionHours"] = fmt.Sprintf("must be between 1 and %d hours", maxSessionHours)
	}
	if spec.APITokenMaxDays < 0 || spec.APITokenMaxDays > maxAPITokenDays {
		errs["apiTokenMaxDays"] = fmt.Sprintf("must be between 1 and %d days", maxAPITokenDays)
	}
	if q := spec.KubeAPIQPS; q != 0 && (q < minKubeAPIQPS || q > maxKubeAPIQPS) {
		errs["kubeApiQps"] = fmt.Sprintf("must be between %d and %d per second", minKubeAPIQPS, maxKubeAPIQPS)
	}
	if q := spec.KubeAPIUserQPS; q != 0 && (q < minKubeAPIUserQPS || q > maxKubeAPIUserQPS) {
		errs["kubeApiUserQps"] = fmt.Sprintf("must be between %d and %d per second",
			minKubeAPIUserQPS, maxKubeAPIUserQPS)
	}
	for _, f := range []struct {
		field, text string
		max         int
	}{
		{"legalNotice", spec.LegalNotice, maxLegalLength},
		{"privacyPolicy", spec.PrivacyPolicy, maxLegalLength},
		{"brandName", spec.BrandName, maxBrandName},
		{"brandTagline", spec.BrandTagline, maxBrandTagline},
	} {
		if n := len([]rune(f.text)); n > f.max {
			errs[f.field] = fmt.Sprintf("too long (%d of %d characters)", n, f.max)
		}
	}
	for field, url := range map[string]string{"brandLogo": spec.BrandLogo, "favicon": spec.Favicon} {
		if err := checkBrandImage(url); err != nil {
			errs[field] = err.Error()
		}
	}
	return errs
}

// checkBrandImage accepts an empty value or a base64 data URL of an image of at most 128 KiB.
func checkBrandImage(url string) error {
	if url == "" {
		return nil
	}
	if len(url) > maxBrandImageURL {
		return errBrandImage
	}
	m := brandImage.FindStringSubmatch(url)
	if m == nil {
		return errBrandImage
	}
	if data, err := base64.StdEncoding.DecodeString(m[2]); err != nil || len(data) == 0 || len(data) > maxBrandImage {
		return errBrandImage
	}
	return nil
}

var errBrandImage = errors.New("use a PNG, JPEG, GIF, WebP, SVG or ICO image of at most 128 KiB")
