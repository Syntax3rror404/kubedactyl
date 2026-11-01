package users

import (
	"context"
	"errors"
	"testing"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"app/api/v1alpha1"
	"app/internal/testutil"
	"app/internal/validation"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	c := testutil.Builder(t).Build()
	return &Store{Client: c, Reader: c, Namespace: testutil.Namespace}
}

func TestCreateValidates(t *testing.T) {
	s := newStore(t)
	_, err := s.Create(context.Background(), CreateInput{Username: "Bad Name!", Password: "short", Role: "root"})
	var ve validation.Error
	if !errors.As(err, &ve) {
		t.Fatalf("want validation errors, got %v", err)
	}
	for _, field := range []string{"username", "password", "role"} {
		if ve.Fields()[field] == "" {
			t.Errorf("no message for %s: %v", field, ve.Fields())
		}
	}
}

func TestLastAdminIsKept(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	admin, err := s.Create(ctx, CreateInput{Username: " Admin ", Password: "a-long-password", Role: v1alpha1.RoleAdmin})
	if err != nil {
		t.Fatal(err)
	}
	if admin.Name != "admin" {
		t.Errorf("name = %q, want it lower-cased and trimmed", admin.Name)
	}
	user := v1alpha1.RoleUser
	if err := s.Update(ctx, admin, UpdateInput{Role: &user}); !errors.Is(err, ErrLastAdmin) {
		t.Errorf("demoting the last admin: %v", err)
	}
	if err := s.Delete(ctx, admin, "someone"); !errors.Is(err, ErrLastAdmin) {
		t.Errorf("deleting the last admin: %v", err)
	}
	if err := s.Delete(ctx, admin, "admin"); !errors.Is(err, ErrOwnAccount) {
		t.Errorf("deleting the own account: %v", err)
	}
	second, err := s.Create(ctx, CreateInput{Username: "bob", Password: "a-long-password", Role: v1alpha1.RoleAdmin})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Update(ctx, second, UpdateInput{Role: &user}); err != nil {
		t.Errorf("demoting one of two admins: %v", err)
	}
}

// Two administrators removed at the same time: counting and changing must not interleave, or both
// requests see "another admin exists" and the panel ends up without one. The slow list widens the
// window between the count and the change.
func TestLastAdminIsKeptUnderConcurrency(t *testing.T) {
	for range 20 {
		c := testutil.Builder(t).WithInterceptorFuncs(interceptor.Funcs{
			List: func(
				ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption,
			) error {
				time.Sleep(20 * time.Millisecond)
				return c.List(ctx, list, opts...)
			},
		}).Build()
		s := &Store{Client: c, Reader: c, Namespace: testutil.Namespace}
		ctx := context.Background()
		a, _ := s.Create(ctx, CreateInput{Username: "alice", Password: "a-long-password", Role: v1alpha1.RoleAdmin})
		b, _ := s.Create(ctx, CreateInput{Username: "bob", Password: "a-long-password", Role: v1alpha1.RoleAdmin})
		user := v1alpha1.RoleUser
		errs := make(chan error, 2)
		go func() { errs <- s.Update(ctx, a, UpdateInput{Role: &user}) }()
		go func() { errs <- s.Delete(ctx, b, "someone") }()
		refused := 0
		for range 2 {
			if err := <-errs; errors.Is(err, ErrLastAdmin) {
				refused++
			} else if err != nil {
				t.Fatal(err)
			}
		}
		if n, _ := s.ActiveAdmins(ctx); refused != 1 || n != 1 {
			t.Fatalf("refused %d of 2 changes, %d active admins left, want 1 and 1", refused, n)
		}
	}
}
