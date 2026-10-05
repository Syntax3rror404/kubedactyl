package eggstore

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"app/api/v1alpha1"
	"app/internal/egg"
	"app/internal/testutil"
)

func TestAutoUpdate(t *testing.T) {
	file := testutil.Download(t, testutil.PaperPLCN)
	var mu sync.Mutex
	serve := file
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if serve == nil {
			http.Error(w, "gone", http.StatusNotFound)
			return
		}
		_, _ = w.Write(serve)
	}))
	defer srv.Close()

	c := testutil.Builder(t).WithStatusSubresource(&v1alpha1.Egg{}).Build()
	s := &Store{Client: c, Reader: c, Namespace: testutil.Namespace}
	ctx := context.Background()
	e, _, err := s.Import(ctx, file, srv.URL+"/paper.yaml", false)
	if err != nil {
		t.Fatal(err)
	}
	// Saved in the editor like the API does: normalized (the file's done line ends with a space).
	spec := e.Spec.DeepCopy()
	spec.Source.AutoUpdate, spec.Source.UpdateURL = true, srv.URL+"/paper.yaml"
	egg.Normalize(spec)
	if _, err := s.Update(ctx, e.Name, spec); err != nil {
		t.Fatal(err)
	}
	reload := func() *v1alpha1.Egg {
		cur := &v1alpha1.Egg{}
		_ = c.Get(ctx, s.key(e.Name), cur)
		return cur
	}

	if changed, err := s.AutoUpdate(ctx, reload()); err != nil || changed {
		t.Fatalf("same file: changed=%v err=%v", changed, err)
	}
	if cur := reload(); cur.Status.UpdateCheckedAt == nil || cur.Status.UpdatedAt != nil {
		t.Errorf("same file must record the check only: %+v", cur.Status)
	}

	newer := e.Spec.DeepCopy()
	newer.Description = "A newer description"
	changedFile, _, err := egg.Export("paper", newer, egg.FormatYAML, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	serve = changedFile
	mu.Unlock()
	if changed, err := s.AutoUpdate(ctx, reload()); err != nil || !changed {
		t.Fatalf("changed file: changed=%v err=%v", changed, err)
	}
	cur := reload()
	if !bytes.Contains([]byte(cur.Spec.Description), []byte("A newer description")) || !cur.Spec.Source.AutoUpdate ||
		cur.Status.UpdatedAt == nil {
		t.Errorf(
			"update not applied or auto update lost: desc=%q auto=%v status=%+v", cur.Spec.Description,
			cur.Spec.Source.AutoUpdate, cur.Status,
		)
	}

	mu.Lock()
	serve = nil
	mu.Unlock()
	if _, err := s.AutoUpdate(ctx, reload()); err == nil {
		t.Fatal("a failed download must be an error")
	}
	if cur := reload(); cur.Status.UpdateError == "" {
		t.Errorf("the error must be in the status: %+v", cur.Status)
	}
}

// Editing, updating by hand and re-importing keep the auto update setting.
func TestAutoUpdateSettingIsKept(t *testing.T) {
	file := testutil.Download(t, testutil.PaperPLCN)
	c := testutil.Builder(t).WithStatusSubresource(&v1alpha1.Egg{}).Build()
	s := &Store{Client: c, Reader: c, Namespace: testutil.Namespace}
	ctx := context.Background()
	e, _, _ := s.Import(ctx, file, "", false)
	spec := e.Spec.DeepCopy()
	spec.Source.AutoUpdate = true
	if _, err := s.Update(ctx, e.Name, spec); err != nil {
		t.Fatal(err)
	}
	again, _, err := s.Import(ctx, file, "", false)
	if err != nil || !again.Spec.Source.AutoUpdate {
		t.Errorf("re-import lost the setting: %v %v", again.Spec.Source.AutoUpdate, err)
	}
}

func TestClearedUpdateURLTurnsAutoUpdateOff(t *testing.T) {
	file := testutil.Download(t, testutil.PaperPLCN)
	c := testutil.Builder(t).WithStatusSubresource(&v1alpha1.Egg{}).Build()
	s := &Store{Client: c, Reader: c, Namespace: testutil.Namespace}
	ctx := context.Background()
	e, _, _ := s.Import(ctx, file, "https://example.com/paper.yaml", false)
	e.Status.UpdateError = "download failed"
	if err := c.Status().Update(ctx, e); err != nil {
		t.Fatal(err)
	}
	spec := e.Spec.DeepCopy()
	spec.Source.UpdateURL, spec.Source.AutoUpdate = "", true
	if _, err := s.Update(ctx, e.Name, spec); err != nil {
		t.Fatal(err)
	}
	saved, _ := s.Get(ctx, e.Name)
	if src := saved.Spec.Source; src.AutoUpdate || src.UpdateURL != "" || src.ImportedFrom != "" {
		t.Errorf("clearing the update URL removes every link and the auto update: %+v", src)
	}
	if saved.Status != (v1alpha1.EggStatus{}) {
		t.Errorf("auto update results are removed: %+v", saved.Status)
	}
	// The URL it was imported from does not come back as update URL.
	if _, err := s.UpdateFromURL(ctx, e.Name); !errors.Is(err, ErrNoUpdateURL) {
		t.Errorf("update from URL after clearing it: %v", err)
	}
}

// Installing from the egg library can turn the auto update on with the import.
func TestImportWithAutoUpdate(t *testing.T) {
	file := testutil.Download(t, testutil.PaperPLCN)
	c := testutil.Builder(t).WithStatusSubresource(&v1alpha1.Egg{}).Build()
	s := &Store{Client: c, Reader: c, Namespace: testutil.Namespace}
	e, _, err := s.Import(context.Background(), file, "https://example.com/paper.yaml", true)
	if err != nil || !e.Spec.Source.AutoUpdate {
		t.Fatalf("auto update not on: %v %+v", err, e.Spec.Source)
	}
}
