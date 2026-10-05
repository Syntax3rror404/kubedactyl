package eggstore

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/egg"
)

// Updater checks the eggs with source.autoUpdate every Interval and applies newer files
// (a manager runnable). The result of each check is the egg's status.
type Updater struct {
	Store    *Store
	Interval time.Duration
	Log      *slog.Logger
}

// Start implements manager.Runnable: the first check runs a minute after the start.
func (u *Updater) Start(ctx context.Context) error {
	wait := time.Minute
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(wait):
		}
		u.checkAll(ctx)
		wait = u.Interval
	}
}

func (u *Updater) checkAll(ctx context.Context) {
	var list v1alpha1.EggList
	if err := u.Store.Reader.List(ctx, &list, client.InNamespace(u.Store.Namespace)); err != nil {
		u.Log.Warn("listing eggs for auto updates", "err", err)
		return
	}
	for i := range list.Items {
		e := &list.Items[i]
		if !e.Spec.Source.AutoUpdate {
			continue
		}
		changed, err := u.Store.AutoUpdate(ctx, e)
		switch {
		case err != nil:
			u.Log.Warn("egg auto update failed", "egg", e.Name, "err", err)
		case changed:
			u.Log.Info("egg updated automatically", "egg", e.Name)
		}
	}
}

// AutoUpdate downloads the egg from its update URL and replaces it when the file differs from
// the stored egg. The check is recorded in the egg's status; changed reports an applied update.
func (s *Store) AutoUpdate(ctx context.Context, e *v1alpha1.Egg) (changed bool, err error) {
	spec, err := fetchUpdate(ctx, e)
	if err == nil && !sameEgg(&e.Spec, spec) {
		e.Spec = *spec
		if err = s.Client.Update(ctx, e); err == nil {
			changed = true
		}
	}
	now := metav1.Now()
	e.Status.UpdateCheckedAt = &now
	e.Status.UpdateError = ""
	if err != nil {
		e.Status.UpdateError = err.Error()
	}
	if changed {
		e.Status.UpdatedAt = &now
	}
	if serr := s.Client.Status().Update(ctx, e); serr != nil && err == nil {
		err = serr
	}
	return changed, err
}

// sameEgg compares two eggs without their source details (import time, URL), both normalized
// like an egg saved in the editor (trimmed values); JSON so that empty and missing lists count
// as equal.
func sameEgg(a, b *v1alpha1.EggSpec) bool {
	strip := func(s *v1alpha1.EggSpec) []byte {
		c := s.DeepCopy()
		c.Source = v1alpha1.EggSource{}
		egg.Normalize(c)
		out, _ := json.Marshal(c)
		return out
	}
	return string(strip(a)) == string(strip(b))
}
