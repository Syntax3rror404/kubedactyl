package controller

import (
	"context"
	"strings"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"app/api/v1alpha1"
	"app/internal/tenancy"
	"app/internal/testutil"
)

// A deleted subuser loses its shares; a share changed meanwhile is never overwritten (retried).
func TestReconcileSubuserRemovesShares(t *testing.T) {
	for _, tc := range []struct {
		name     string
		conflict bool
	}{
		{"shares removed", false},
		{"server changed meanwhile", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			b := testutil.Builder(t)
			sub := &v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "carol", Namespace: testutil.Namespace, Finalizers: []string{UserFinalizer},
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Spec: v1alpha1.UserSpec{Owner: "alice"},
			}
			gs := &v1alpha1.GameServer{
				ObjectMeta: metav1.ObjectMeta{Name: "mc", Namespace: tenancy.Namespace("alice")},
				Spec: v1alpha1.GameServerSpec{SharedWith: []v1alpha1.ServerShare{
					{User: "carol", Access: v1alpha1.ShareView}, {User: "dave", Access: v1alpha1.ShareControl},
				}},
			}
			c := b.WithObjects(sub, gs).WithInterceptorFuncs(interceptor.Funcs{Patch: func(
				ctx context.Context, c client.WithWatch, obj client.Object, patch client.Patch,
				opts ...client.PatchOption,
			) error {
				if data, _ := patch.Data(obj); !strings.Contains(string(data), `"resourceVersion"`) {
					t.Errorf("the patch has no optimistic lock: %s", data)
				}
				if tc.conflict {
					return apierrors.NewConflict(schema.GroupResource{Resource: "gameservers"}, obj.GetName(), nil)
				}
				return c.Patch(ctx, obj, patch, opts...)
			}}).Build()
			r := &UserReconciler{Client: c, Reader: c, Namespace: testutil.Namespace, Log: testutil.Logger()}
			res, err := r.reconcileSubuser(ctx, sub)
			if err != nil || (res.RequeueAfter > 0) != tc.conflict {
				t.Fatalf("result %+v, %v", res, err)
			}
			stored := &v1alpha1.GameServer{}
			_ = c.Get(ctx, client.ObjectKeyFromObject(gs), stored)
			if want := map[bool]int{false: 1, true: 2}[tc.conflict]; len(stored.Spec.SharedWith) != want {
				t.Errorf("shares %+v, want %d", stored.Spec.SharedWith, want)
			}
			err = c.Get(ctx, client.ObjectKeyFromObject(sub), &v1alpha1.User{})
			if apierrors.IsNotFound(err) == tc.conflict {
				t.Errorf("finalizer removed = %v, want %v", apierrors.IsNotFound(err), !tc.conflict)
			}
		})
	}
}

// Only the settings of this installation and only a change of the private network switch reconcile
// the users.
func TestPrivateNetworksChanged(t *testing.T) {
	r := &UserReconciler{Namespace: testutil.Namespace}
	settings := func(ns string, allow bool) *v1alpha1.PanelSettings {
		return &v1alpha1.PanelSettings{
			ObjectMeta: metav1.ObjectMeta{Name: v1alpha1.SettingsName, Namespace: ns},
			Spec:       v1alpha1.PanelSettingsSpec{AllowPrivateNetworks: allow, BrandName: "x"},
		}
	}
	other := settings("other", false)
	branded := settings(testutil.Namespace, false)
	branded.Spec.BrandName = "y"
	p := r.privateNetworksChanged()
	for _, tc := range []struct {
		name string
		got  bool
		want bool
	}{
		{"created", p.Create(event.CreateEvent{Object: settings(testutil.Namespace, false)}), true},
		{"created elsewhere", p.Create(event.CreateEvent{Object: other}), false},
		{"deleted", p.Delete(event.DeleteEvent{Object: settings(testutil.Namespace, true)}), true},
		{"switch turned", p.Update(event.UpdateEvent{
			ObjectOld: settings(testutil.Namespace, false), ObjectNew: settings(testutil.Namespace, true),
		}), true},
		{"other setting changed", p.Update(event.UpdateEvent{
			ObjectOld: settings(testutil.Namespace, false), ObjectNew: branded,
		}), false},
		{"switch turned elsewhere", p.Update(event.UpdateEvent{
			ObjectOld: other, ObjectNew: settings("other", true),
		}), false},
	} {
		if tc.got != tc.want {
			t.Errorf("%s: %v, want %v", tc.name, tc.got, tc.want)
		}
	}
}
