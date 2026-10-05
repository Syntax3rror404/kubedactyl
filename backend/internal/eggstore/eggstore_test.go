package eggstore

import (
	"context"
	"errors"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"app/api/v1alpha1"
	"app/internal/egg"
	"app/internal/tenancy"
	"app/internal/testutil"
)

func newStore(t *testing.T, objs ...runtime.Object) *Store {
	t.Helper()
	c := testutil.Builder(t).WithRuntimeObjects(objs...).Build()
	return &Store{Client: c, Reader: c, Namespace: testutil.Namespace}
}

// paperSpec returns the spec of the Paper egg from the test data.
func paperSpec(t *testing.T) *v1alpha1.EggSpec {
	t.Helper()
	spec, err := egg.Parse(testutil.Download(t, testutil.PaperPLCN))
	if err != nil {
		t.Fatal(err)
	}
	return spec
}

func TestCreatePicksFreeName(t *testing.T) {
	s := newStore(t)
	for _, want := range []string{"paper", "paper-2"} {
		e, err := s.Create(context.Background(), paperSpec(t))
		if err != nil {
			t.Fatal(err)
		}
		if e.Name != want || e.Spec.Source.UUID == "" {
			t.Errorf("name = %q uuid = %q, want %q and a UUID", e.Name, e.Spec.Source.UUID, want)
		}
	}
}

func TestImportReplacesAndKeepsUUID(t *testing.T) {
	data := testutil.Download(t, testutil.PaperPLCN)
	s := newStore(t)
	ctx := context.Background()
	first, created, err := s.Import(ctx, data, "https://example.com/paper.yaml", false)
	if err != nil || !created {
		t.Fatalf("first import: created=%v err=%v", created, err)
	}
	if first.Spec.Source.UpdateURL == "" {
		t.Error("the update URL must be kept")
	}
	again, created, err := s.Import(ctx, data, "", false)
	if err != nil || created || again.Spec.Source.UUID != first.Spec.Source.UUID {
		t.Errorf(
			"re-import: created=%v err=%v uuid %q → %q", created, err, first.Spec.Source.UUID, again.Spec.Source.UUID,
		)
	}
	if _, _, err := s.Import(ctx, []byte("not an egg"), "", false); !errors.Is(err, ErrInvalid) {
		t.Errorf("invalid file: %v", err)
	}
}

func TestDeleteRefusesUsedEgg(t *testing.T) {
	used := &v1alpha1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "srv", Namespace: tenancy.Namespace("alice")},
		Spec:       v1alpha1.GameServerSpec{EggRef: "paper", DisplayName: "Survival"},
	}
	egg := &v1alpha1.Egg{ObjectMeta: metav1.ObjectMeta{Name: "paper", Namespace: testutil.Namespace}}
	s := newStore(t, used, egg)
	if err := s.Delete(context.Background(), "paper"); !errors.Is(err, ErrInUse) {
		t.Errorf("delete used egg: %v", err)
	}
}

func TestCreateValidates(t *testing.T) {
	var fields egg.FieldErrors
	_, err := newStore(t).Create(context.Background(), &v1alpha1.EggSpec{DisplayName: "Paper"})
	if !errors.As(err, &fields) {
		t.Errorf("invalid egg: %v", err)
	}
}

func TestUpdateFromURLNeedsURL(t *testing.T) {
	egg := &v1alpha1.Egg{ObjectMeta: metav1.ObjectMeta{Name: "paper", Namespace: testutil.Namespace}}
	s := newStore(t, egg)
	if _, err := s.UpdateFromURL(context.Background(), "paper"); !errors.Is(err, ErrNoUpdateURL) {
		t.Errorf("without update URL: %v", err)
	}
}
