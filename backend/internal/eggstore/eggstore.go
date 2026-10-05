// Package eggstore stores eggs as Egg resources in the panel namespace: creating them in the
// editor, importing files, updating them from their update URL and deleting unused ones. Parsing,
// validation and export are in package egg.
package eggstore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/egg"
	"app/internal/tenancy"
)

// MaxSize is the largest egg file that is accepted.
const MaxSize = 4 << 20

// Errors of egg operations.
var (
	// ErrInvalid wraps parse errors of egg files.
	ErrInvalid = errors.New("invalid egg")
	// ErrInvalidURL is returned for URLs that are not http(s).
	ErrInvalidURL = errors.New("url must be an http(s) URL")
	// ErrNoUpdateURL is returned when an egg has no URL to update from.
	ErrNoUpdateURL = errors.New("the egg has no http(s) update URL")
	// ErrDownload wraps failed downloads of egg files.
	ErrDownload = errors.New("download failed")
	// ErrInUse is returned when servers still use the egg.
	ErrInUse = errors.New("egg is used by servers")
	// ErrNameTaken is returned when no free name is left for a new egg.
	ErrNameTaken = errors.New("too many eggs with this name")
)

// Store reads and writes eggs.
type Store struct {
	Client client.Client
	// Reader reads uncached: create/update decisions and just created eggs need the current state.
	Reader    client.Reader
	Namespace string
}

func (s *Store) key(name string) client.ObjectKey {
	return client.ObjectKey{Namespace: s.Namespace, Name: name}
}

// Get reads an egg from the cache and, when it is not there (yet), from the API server:
// an egg created a moment ago may not have reached the informer.
func (s *Store) Get(ctx context.Context, name string) (*v1alpha1.Egg, error) {
	e := &v1alpha1.Egg{}
	err := s.Client.Get(ctx, s.key(name), e)
	if apierrors.IsNotFound(err) {
		err = s.Reader.Get(ctx, s.key(name), e)
	}
	return e, err
}

// Create validates an egg made in the editor and stores it under a free name derived from its
// display name ("paper", "paper-2", …) with a new UUID.
func (s *Store) Create(ctx context.Context, spec *v1alpha1.EggSpec) (*v1alpha1.Egg, error) {
	if err := normalize(spec); err != nil {
		return nil, err
	}
	now := metav1.Now()
	spec.Source = v1alpha1.EggSource{
		UUID: egg.NewUUID(), UpdateURL: spec.Source.UpdateURL, AutoUpdate: autoUpdate(spec.Source), EditedAt: &now,
	}
	base := egg.Slug(spec.DisplayName)
	for i := 1; i <= 50; i++ {
		name := base
		if i > 1 {
			name = fmt.Sprintf("%s-%d", base, i)
		}
		e := &v1alpha1.Egg{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: s.Namespace}, Spec: *spec}
		err := s.Client.Create(ctx, e)
		if !apierrors.IsAlreadyExists(err) {
			return e, err
		}
	}
	return nil, fmt.Errorf("%w: %q", ErrNameTaken, spec.DisplayName)
}

// Update validates an egg edited in the panel and replaces its fields; UUID and import details
// are kept. Clearing the update URL removes every link of the egg: the URL it was imported from
// and the results of its auto updates.
func (s *Store) Update(ctx context.Context, name string, spec *v1alpha1.EggSpec) (*v1alpha1.Egg, error) {
	if err := normalize(spec); err != nil {
		return nil, err
	}
	cur := &v1alpha1.Egg{}
	if err := s.Reader.Get(ctx, s.key(name), cur); err != nil {
		return nil, err
	}
	now := metav1.Now()
	src := cur.Spec.Source
	src.UUID = egg.UUIDFor(cur.Name, &cur.Spec)
	src.UpdateURL = spec.Source.UpdateURL
	src.AutoUpdate = autoUpdate(spec.Source)
	src.EditedAt = &now
	if src.UpdateURL == "" {
		src.ImportedFrom = ""
	}
	spec.Source = src
	cur.Spec = *spec
	if err := s.Client.Update(ctx, cur); err != nil {
		return nil, err
	}
	if !src.AutoUpdate && cur.Status != (v1alpha1.EggStatus{}) {
		cur.Status = v1alpha1.EggStatus{}
		return cur, s.Client.Status().Update(ctx, cur)
	}
	return cur, nil
}

// Import parses an egg file and stores it; an egg with the same name is replaced. source is
// the URL the file came from ("" for uploads) and counts as update URL when the file has none.
// autoUpdate turns the hourly update on (an egg imported again keeps it on). created reports
// whether the egg is new.
func (s *Store) Import(
	ctx context.Context, data []byte, source string, autoUpdate bool,
) (e *v1alpha1.Egg, created bool, err error) {
	spec, err := egg.Parse(data)
	if err != nil {
		return nil, false, invalid{err}
	}
	spec.Source.ImportedFrom = source
	spec.Source.ImportedAt = metav1.Now()
	if spec.Source.UpdateURL == "" {
		spec.Source.UpdateURL = source
	}
	name := egg.Slug(spec.DisplayName)
	cur := &v1alpha1.Egg{}
	err = s.Reader.Get(ctx, s.key(name), cur)
	switch {
	case apierrors.IsNotFound(err):
		if spec.Source.UUID == "" {
			spec.Source.UUID = egg.NewUUID()
		}
		spec.Source.AutoUpdate = autoUpdate && spec.Source.UpdateURL != ""
		e := &v1alpha1.Egg{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: s.Namespace}, Spec: *spec}
		return e, true, s.Client.Create(ctx, e)
	case err != nil:
		return nil, false, err
	}
	// Re-importing keeps the UUID when the file has none (PTDL), and the panel's auto update setting.
	if spec.Source.UUID == "" {
		spec.Source.UUID = egg.UUIDFor(cur.Name, &cur.Spec)
	}
	spec.Source.AutoUpdate = (autoUpdate || cur.Spec.Source.AutoUpdate) && spec.Source.UpdateURL != ""
	cur.Spec = *spec
	return cur, false, s.Client.Update(ctx, cur)
}

// UpdateFromURL downloads the egg from its update URL and replaces all fields, including
// changes made in the panel; name and UUID are kept.
func (s *Store) UpdateFromURL(ctx context.Context, name string) (*v1alpha1.Egg, error) {
	cur := &v1alpha1.Egg{}
	if err := s.Reader.Get(ctx, s.key(name), cur); err != nil {
		return nil, err
	}
	spec, err := fetchUpdate(ctx, cur)
	if err != nil {
		return nil, err
	}
	cur.Spec = *spec
	return cur, s.Client.Update(ctx, cur)
}

// fetchUpdate downloads the egg from its update URL; the result keeps name, UUID, the configured
// update URL and the panel's auto update setting.
func fetchUpdate(ctx context.Context, cur *v1alpha1.Egg) (*v1alpha1.EggSpec, error) {
	raw := cur.Spec.Source.UpdateURL
	if _, err := httpURL(raw); err != nil {
		return nil, ErrNoUpdateURL
	}
	data, err := Download(ctx, raw)
	if err != nil {
		return nil, err
	}
	spec, err := egg.Parse(data)
	if err != nil {
		return nil, invalid{err}
	}
	spec.Source.ImportedFrom, spec.Source.ImportedAt = raw, metav1.Now()
	spec.Source.UUID = egg.UUIDFor(cur.Name, &cur.Spec)
	spec.Source.AutoUpdate = cur.Spec.Source.AutoUpdate
	// The configured update URL stays, even when the file names another one.
	spec.Source.UpdateURL = raw
	return spec, nil
}

// autoUpdate is the auto update setting of an egg saved in the panel: it needs an update URL.
func autoUpdate(src v1alpha1.EggSource) bool { return src.AutoUpdate && src.UpdateURL != "" }

// Delete removes an egg that no server of this installation uses (read uncached: a server
// created a moment ago must count).
func (s *Store) Delete(ctx context.Context, name string) error {
	var servers v1alpha1.GameServerList
	if err := s.Reader.List(ctx, &servers); err != nil {
		return err
	}
	for _, gs := range servers.Items {
		if tenancy.Owns(gs.Namespace) && gs.Spec.EggRef == name {
			return fmt.Errorf("%w: %q", ErrInUse, gs.Spec.DisplayName)
		}
	}
	return s.Client.Delete(ctx, &v1alpha1.Egg{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: s.Namespace}})
}

// Download fetches an egg file from an http(s) URL.
func Download(ctx context.Context, raw string) ([]byte, error) {
	u, err := httpURL(raw)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDownload, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s", ErrDownload, resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, MaxSize))
}

// invalid marks a parse error as ErrInvalid and keeps its message.
type invalid struct{ error }

func (e invalid) Is(target error) bool { return target == ErrInvalid }
func (e invalid) Unwrap() error        { return e.error }

func httpURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if raw == "" || err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return nil, ErrInvalidURL
	}
	return u, nil
}

// normalize cleans up an egg edited in the panel and validates it (field errors: 422 with the fields).
func normalize(spec *v1alpha1.EggSpec) error {
	egg.Normalize(spec)
	return egg.ValidateSpec(spec)
}
