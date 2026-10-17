package controller

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"app/api/v1alpha1"
	"app/internal/tenancy"
	"app/internal/users"
)

// UserFinalizer makes sure servers, volumes and the namespace are removed with a user, and the shares of a subuser.
const UserFinalizer = "kubedactyl.io/user-cleanup"

// UserReconciler provisions the namespace of every user and cleans it up on deletion.
type UserReconciler struct {
	client.Client
	Reader client.Reader
	Log    *slog.Logger
}

// SetupWithManager registers the controller.
func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.User{}).Named("user").
		Watches(&v1alpha1.PanelSettings{}, handler.EnqueueRequestsFromMapFunc(r.usersForSettings)).
		Complete(r)
}

// Reconcile gives a user a namespace with the panel's role binding and the network policy, and
// removes the user's servers and namespace when the user is deleted.
func (r *UserReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	if req.Namespace != tenancy.SystemNamespace {
		return reconcile.Result{}, nil // user of another installation
	}
	user := &v1alpha1.User{}
	if err := r.Reader.Get(ctx, req.NamespacedName, user); err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}
	if users.Subuser(user) {
		return reconcile.Result{}, r.reconcileSubuser(ctx, user)
	}
	ns := tenancy.Namespace(user.Name)

	if !user.DeletionTimestamp.IsZero() {
		return r.cleanup(ctx, user, ns)
	}
	if controllerutil.AddFinalizer(user, UserFinalizer) {
		if err := r.Update(ctx, user); err != nil {
			return reconcile.Result{}, err
		}
	}
	if _, err := tenancy.EnsureNamespace(ctx, r.Client, user.Name); err != nil {
		return reconcile.Result{}, err
	}
	if err := r.ensureNetworkPolicy(ctx, ns); err != nil {
		return reconcile.Result{}, fmt.Errorf("network policy: %w", err)
	}
	if user.Status.Namespace != ns {
		patch := client.MergeFrom(user.DeepCopy())
		user.Status.Namespace = ns
		if err := r.Status().Patch(ctx, user, patch); err != nil {
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}

// cleanup deletes the game servers first (their finalizer releases the volumes), then the namespace.
func (r *UserReconciler) cleanup(ctx context.Context, user *v1alpha1.User, ns string) (reconcile.Result, error) {
	if !controllerutil.ContainsFinalizer(user, UserFinalizer) {
		return reconcile.Result{}, nil
	}
	var servers v1alpha1.GameServerList
	if err := r.Reader.List(ctx, &servers, client.InNamespace(ns)); err != nil {
		return reconcile.Result{}, err
	}
	if len(servers.Items) > 0 {
		for i := range servers.Items {
			if servers.Items[i].DeletionTimestamp.IsZero() {
				if err := r.Delete(ctx, &servers.Items[i]); client.IgnoreNotFound(err) != nil {
					return reconcile.Result{}, err
				}
			}
		}
		r.Log.Info("waiting for game servers of deleted user", "user", user.Name, "remaining", len(servers.Items))
		return reconcile.Result{RequeueAfter: 3 * time.Second}, nil
	}
	namespace := &corev1.Namespace{}
	err := r.Get(ctx, client.ObjectKey{Name: ns}, namespace)
	switch {
	case apierrors.IsNotFound(err):
	case err != nil:
		return reconcile.Result{}, err
	case namespace.Labels[tenancy.LabelUser] != user.Name:
		// Never delete a namespace the panel did not create for this user.
		r.Log.Warn("namespace not owned by user, keeping it", "namespace", ns, "user", user.Name)
	case namespace.DeletionTimestamp.IsZero():
		if err := r.Delete(ctx, namespace); client.IgnoreNotFound(err) != nil {
			return reconcile.Result{}, err
		}
	}
	controllerutil.RemoveFinalizer(user, UserFinalizer)
	return reconcile.Result{}, r.Update(ctx, user)
}

// reconcileSubuser gives a subuser no namespace. When it is deleted, its shares on the owner's servers go with
// it, so a later account with the same name gets none of them.
func (r *UserReconciler) reconcileSubuser(ctx context.Context, user *v1alpha1.User) error {
	if user.DeletionTimestamp.IsZero() {
		if controllerutil.AddFinalizer(user, UserFinalizer) {
			return r.Update(ctx, user)
		}
		return nil
	}
	if !controllerutil.ContainsFinalizer(user, UserFinalizer) {
		return nil
	}
	var servers v1alpha1.GameServerList
	if err := r.Reader.List(ctx, &servers, client.InNamespace(users.Home(user))); err != nil {
		return err
	}
	for i := range servers.Items {
		gs := &servers.Items[i]
		shares := slices.DeleteFunc(slices.Clone(gs.Spec.SharedWith), func(s v1alpha1.ServerShare) bool {
			return s.User == user.Name
		})
		if len(shares) == len(gs.Spec.SharedWith) {
			continue
		}
		patch := client.MergeFrom(gs.DeepCopy())
		gs.Spec.SharedWith = shares
		if err := r.Patch(ctx, gs, patch); client.IgnoreNotFound(err) != nil {
			return err
		}
	}
	controllerutil.RemoveFinalizer(user, UserFinalizer)
	return r.Update(ctx, user)
}
