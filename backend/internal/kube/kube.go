// Package kube wraps the Kubernetes streaming APIs (exec, attach, logs) used by the panel and
// CreateOrPatch, which decides create or update with an uncached read.
package kube

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	utilexec "k8s.io/client-go/util/exec"
	"k8s.io/streaming/pkg/httpstream"
)

// Client bundles the REST config and a typed clientset.
type Client struct {
	Config    *rest.Config
	Clientset kubernetes.Interface
}

// New creates a Client from a REST config.
func New(cfg *rest.Config) (*Client, error) {
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &Client{Config: cfg, Clientset: cs}, nil
}

// executor prefers the WebSocket protocol and falls back to SPDY.
func (k *Client) executor(req *rest.Request) (remotecommand.Executor, error) {
	ws, err := remotecommand.NewWebSocketExecutor(k.Config, http.MethodGet, req.URL().String())
	if err != nil {
		return nil, err
	}
	spdy, err := remotecommand.NewSPDYExecutor(k.Config, http.MethodPost, req.URL())
	if err != nil {
		return nil, err
	}
	return remotecommand.NewFallbackExecutor(ws, spdy, func(err error) bool {
		return httpstream.IsUpgradeFailure(err) || httpstream.IsHTTPSProxyError(err)
	})
}

// ExitError is returned by Exec when the command exits with a non-zero code.
type ExitError struct {
	Code   int
	Stderr string
}

func (e *ExitError) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf("command exited with code %d: %s", e.Code, e.Stderr)
	}
	return fmt.Sprintf("command exited with code %d", e.Code)
}

// ErrOutputTooLarge is returned by a LimitedBuffer that would grow beyond its limit.
var ErrOutputTooLarge = errors.New("the output of the command is too large")

// LimitedBuffer is a buffer for command output that holds at most Max bytes: a write beyond
// it fails (and so ends the command's stream) and sets Full.
type LimitedBuffer struct {
	bytes.Buffer
	Max  int
	Full bool
}

func (b *LimitedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > b.Max {
		b.Full = true
		return 0, ErrOutputTooLarge
	}
	return b.Buffer.Write(p)
}

// maxStderr limits the stderr an ExitError keeps; maxOutput the stdout ExecOutput returns.
const (
	maxStderr = 64 << 10
	maxOutput = 1 << 20
)

// Exec runs cmd in a container; its stderr (up to 64 KiB) ends up in the ExitError.
func (k *Client) Exec(
	ctx context.Context, ns, pod, container string, cmd []string, stdin io.Reader, stdout io.Writer,
) error {
	req := k.Clientset.CoreV1().RESTClient().Post().
		Resource("pods").Namespace(ns).Name(pod).SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   cmd,
			Stdin:     stdin != nil,
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)
	exec, err := k.executor(req)
	if err != nil {
		return err
	}
	if stdout == nil {
		stdout = io.Discard
	}
	stderr := LimitedBuffer{Max: maxStderr}
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{Stdin: stdin, Stdout: stdout, Stderr: &stderr})
	var codeErr utilexec.CodeExitError
	if errors.As(err, &codeErr) {
		return &ExitError{Code: codeErr.Code, Stderr: stderr.String()}
	}
	return err
}

// ExecOutput runs cmd and returns its stdout (at most 1 MiB).
func (k *Client) ExecOutput(
	ctx context.Context, ns, pod, container string, cmd []string, stdin io.Reader,
) ([]byte, error) {
	out := LimitedBuffer{Max: maxOutput}
	err := k.Exec(ctx, ns, pod, container, cmd, stdin, &out)
	return out.Bytes(), err
}

// AttachWrite writes data to the stdin (TTY) of a running container.
func (k *Client) AttachWrite(ctx context.Context, ns, pod, container string, data []byte) error {
	req := k.Clientset.CoreV1().RESTClient().Post().
		Resource("pods").Namespace(ns).Name(pod).SubResource("attach").
		VersionedParams(&corev1.PodAttachOptions{
			Container: container,
			Stdin:     true,
			Stdout:    true,
			TTY:       true,
		}, scheme.ParameterCodec)
	exec, err := k.executor(req)
	if err != nil {
		return err
	}
	// The attach stream only ends when the container exits, so we close it ourselves
	// shortly after stdin has been delivered.
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	in := &onceReader{data: data, sent: make(chan struct{}), ctx: ctx}
	done := make(chan error, 1)
	go func() {
		done <- exec.StreamWithContext(ctx, remotecommand.StreamOptions{Stdin: in, Stdout: io.Discard, Tty: true})
	}()
	select {
	case err := <-done:
		if err == nil {
			err = errors.New("attach stream closed before the input was sent")
		}
		return err
	case <-in.sent:
		// Give the stream a moment to flush the frame before closing it.
		select {
		case <-done:
			// The container exited (e.g. after ^C); the input was delivered.
		case <-time.After(300 * time.Millisecond):
			cancel()
			<-done
		}
		return nil
	case <-ctx.Done():
		<-done
		return fmt.Errorf("attach: %w", ctx.Err())
	}
}

// onceReader returns data once and then blocks until ctx is done, so the stdin of
// the attach session is not closed while the data is still in flight.
type onceReader struct {
	data []byte
	sent chan struct{}
	ctx  context.Context
	done bool
}

func (r *onceReader) Read(p []byte) (int, error) {
	if len(r.data) > 0 {
		n := copy(p, r.data)
		r.data = r.data[n:]
		return n, nil
	}
	if !r.done {
		r.done = true
		close(r.sent)
	}
	<-r.ctx.Done()
	return 0, io.EOF
}
