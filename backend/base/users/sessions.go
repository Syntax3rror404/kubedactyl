package users

import (
	"context"
	"slices"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
)

// MaxLoggedOut is how many logged out sessions a user keeps; one more ends every session of the user instead.
const MaxLoggedOut = 100

// LogOut keeps a session the user logged out of in the user's status until it would have expired, in three steps:
//  1. Read the user from the API server: the cache may lag behind another logout a moment ago.
//  2. Drop the sessions that have expired by now and add this one.
//  3. With MaxLoggedOut kept already, end every session of the user instead (a new session epoch); the list starts
//     empty again, as every session it holds ends with the epoch.
//
// A conflict with another write reads the user again.
func (s *Store) LogOut(ctx context.Context, name, id string, expires time.Time) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		u := &v1alpha1.User{}
		if err := s.Reader.Get(ctx, client.ObjectKey{Namespace: s.Namespace, Name: name}, u); err != nil {
			return err
		}
		now := time.Now()
		kept := slices.DeleteFunc(slices.Clone(u.Status.LoggedOut), func(l v1alpha1.LoggedOutSession) bool {
			return !now.Before(l.ExpiresAt.Time)
		})
		if len(kept) >= MaxLoggedOut {
			if err := s.EndSessions(ctx, u); err != nil {
				return err
			}
			kept = nil
		} else {
			kept = append(kept, v1alpha1.LoggedOutSession{ID: id, ExpiresAt: metav1.NewTime(expires)})
		}
		before := u.DeepCopy()
		u.Status.LoggedOut = kept
		patch := client.MergeFromWithOptions(before, client.MergeFromWithOptimisticLock{})
		return s.Client.Status().Patch(ctx, u, patch)
	})
}

// LoggedOut reports whether the user logged out of the session id (auth.SessionID).
func LoggedOut(u *v1alpha1.User, id string) bool {
	return slices.ContainsFunc(u.Status.LoggedOut, func(l v1alpha1.LoggedOutSession) bool { return l.ID == id })
}
