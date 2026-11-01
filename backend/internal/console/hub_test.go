package console

import (
	"bufio"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"app/internal/gameserver"
	"app/internal/kube"
	"app/internal/testutil"
)

func TestDeduperReconnect(t *testing.T) {
	t0 := time.Date(2026, 9, 28, 12, 0, 0, 100, time.UTC)
	t1 := t0.Add(time.Millisecond)
	var d deduper
	d.reconnect()
	// First stream: two lines share t1.
	for i, ts := range []time.Time{t0, t1, t1} {
		if !d.accept(ts) {
			t.Fatalf("line %d dropped on first stream", i)
		}
	}
	// Reconnect replays from the start of the second: t0, t1, t1 are old, then a new t1 line and t2.
	d.reconnect()
	want := []bool{false, false, false, true, true}
	for i, ts := range []time.Time{t0, t1, t1, t1, t1.Add(time.Second)} {
		if got := d.accept(ts); got != want[i] {
			t.Errorf("replayed line %d: accept=%v, want %v", i, got, want[i])
		}
	}
}

func TestSplitTimestamp(t *testing.T) {
	ts, text := splitTimestamp(
		"2026-09-28T02:44:15.123456789Z [12:00:00 INFO]: Done (3.2s)! For help, type \"help\"\r\n",
	)
	if ts.IsZero() || text != `[12:00:00 INFO]: Done (3.2s)! For help, type "help"` {
		t.Fatalf("got %v %q", ts, text)
	}
	if _, text := splitTimestamp("2026-09-28T02:44:15Z 10%\r50%\r100%\n"); text != "100%" {
		t.Errorf("carriage return handling: %q", text)
	}
}

func TestMatcher(t *testing.T) {
	m := NewMatcher([]string{")! For help, type ", "regex:^Server started on port \\d+$"}, true)
	if !m.Match("\x1b[32m[12:00 INFO]: Done (1s)! For help, type \"help\"\x1b[0m") {
		t.Error("plain match failed")
	}
	if !m.Match("Server started on port 25565") || m.Match("Server started on port x") {
		t.Error("regex match wrong")
	}
}

// TestReadLine: a line without end is cut at the buffer size, the rest of it is skipped.
func TestReadLine(t *testing.T) {
	long := strings.Repeat("x", 3*maxLineLength)
	r := bufio.NewReaderSize(strings.NewReader("short\n"+long+"\nnext\n"+long), maxLineLength)
	var got []string
	for {
		line, err := readLine(r)
		got = append(got, line)
		if err == io.EOF {
			break
		}
	}
	want := []string{"short\n", long[:maxLineLength] + " …\n", "next\n", long[:maxLineLength] + " …\n"}
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d: got %d bytes, want %d", i, len(got[i]), len(want[i]))
		}
	}
}

// The done line counts for the followed game pod only; a new pod and the removal of the server forget it.
func TestDone(t *testing.T) {
	h := NewHub(&kube.Client{Clientset: k8sfake.NewClientset()}, testutil.Logger())
	h.stream("mc").session = &session{uid: "pod-1", cancel: func() {}}
	for _, tc := range []struct {
		name      string
		uid       types.UID
		wantFirst bool
	}{
		{"followed pod", "pod-1", true},
		{"again", "pod-1", false},
		{"pod no longer followed", "pod-0", false},
	} {
		if first := h.markDone("mc", tc.uid); first != tc.wantFirst {
			t.Errorf("%s: first = %v, want %v", tc.name, first, tc.wantFirst)
		}
	}
	if !h.IsDone("mc", "pod-1") || h.IsDone("mc", "pod-0") || h.IsDone("other", "pod-1") {
		t.Error("done state of the wrong pod or server")
	}
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "mc-game", Namespace: "ns", UID: "pod-2"}}
	h.Follow("mc", pod, KindGame, NewMatcher(nil, false))
	if h.IsDone("mc", "pod-1") {
		t.Error("a new pod keeps the done state of the previous one")
	}
	h.markDone("mc", "pod-2")
	h.Remove("mc")
	if h.IsDone("mc", "pod-2") {
		t.Error("a removed server keeps its done state")
	}
}

// A pod with the same name but another UID is a new pod: the old one is gone.
func TestContainerTerminated(t *testing.T) {
	pod := func(uid types.UID, state corev1.ContainerState) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "mc-game", Namespace: "ns", UID: uid},
			Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{
				{Name: gameserver.ContainerName, State: state},
			}},
		}
	}
	running := corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}
	exited := corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}}
	for _, tc := range []struct {
		name string
		pods []runtime.Object
		want bool
	}{
		{"running", []runtime.Object{pod("pod-1", running)}, false},
		{"exited", []runtime.Object{pod("pod-1", exited)}, true},
		{"gone", nil, true},
		{"replaced by a new pod", []runtime.Object{pod("pod-2", running)}, true},
	} {
		h := NewHub(&kube.Client{Clientset: k8sfake.NewClientset(tc.pods...)}, testutil.Logger())
		if got := h.containerTerminated(context.Background(), "ns", "mc-game", "pod-1"); got != tc.want {
			t.Errorf("%s: terminated = %v, want %v", tc.name, got, tc.want)
		}
	}
}
