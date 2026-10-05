package kube

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// TestAttachIntegration needs a cluster: KUBE_IT_CONTEXT=<context> go test ./internal/kube -run Attach -v
func TestAttachIntegration(t *testing.T) {
	kctx := os.Getenv("KUBE_IT_CONTEXT")
	if kctx == "" {
		t.Skip("KUBE_IT_CONTEXT not set")
	}
	cfg, err := config.GetConfigWithContext(kctx)
	if err != nil {
		t.Fatal(err)
	}
	k, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	ns := "kubedactyl-attachtest"
	pods := k.Clientset.CoreV1().Pods(ns)
	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}
	_, _ = k.Clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	defer func() { _ = k.Clientset.CoreV1().Namespaces().Delete(ctx, ns, metav1.DeleteOptions{}) }()

	script := `trap 'echo GOT-SIGINT; exit 3' INT; echo READY; while read -r line; do echo "GOT:$line"; done`
	_, err = pods.Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "t"},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{{
				Name: "server", Image: "alpine:3.24.2", Command: []string{"sh", "-c", script},
				Stdin: true, TTY: true,
			}},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool {
		p, err := pods.Get(ctx, "t", metav1.GetOptions{})
		return err == nil && p.Status.Phase == corev1.PodRunning
	})
	logs := func() string {
		b, _ := pods.GetLogs("t", &corev1.PodLogOptions{Container: "server"}).DoRaw(ctx)
		return string(b)
	}
	if err := k.AttachWrite(ctx, ns, "t", "server", []byte("stop\n")); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return strings.Contains(logs(), "GOT:stop") })
	if err := k.AttachWrite(ctx, ns, "t", "server", []byte("second command\n")); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return strings.Contains(logs(), "GOT:second command") })

	out, err := k.ExecOutput(
		ctx, ns, "t", "server", []string{"sh", "-c", `cat > /tmp/x && echo "len=$(wc -c < /tmp/x)"`},
		strings.NewReader("hello"),
	)
	if err != nil || !strings.Contains(string(out), "len=5") {
		t.Fatalf("exec with stdin: %q %v", out, err)
	}
	if err := k.Exec(
		ctx, ns, "t", "server", []string{"sh", "-c", "echo boom >&2; exit 7"}, nil, nil,
	); err == nil || !strings.Contains(err.Error(), "code 7") ||
		!strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected exit error, got %v", err)
	}

	// ^C through the TTY must deliver SIGINT to the process.
	if err := k.AttachWrite(ctx, ns, "t", "server", []byte{0x03}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool {
		p, err := pods.Get(ctx, "t", metav1.GetOptions{})
		if err != nil || len(p.Status.ContainerStatuses) == 0 || p.Status.ContainerStatuses[0].State.Terminated == nil {
			return false
		}
		return p.Status.ContainerStatuses[0].State.Terminated.ExitCode == 3
	})
	t.Logf("logs:\n%s", logs())

	// Writing to an exited container or a missing pod must fail fast instead of hanging.
	for _, pod := range []string{"t", "does-not-exist"} {
		start := time.Now()
		err := k.AttachWrite(ctx, ns, pod, "server", []byte("stop\n"))
		if err == nil || time.Since(start) > 12*time.Second {
			t.Fatalf("AttachWrite(%s) = %v after %v, want a fast error", pod, err, time.Since(start))
		}
		t.Logf("AttachWrite(%s) failed as expected after %v: %v", pod, time.Since(start).Round(time.Millisecond), err)
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatal("timeout waiting for condition")
}
