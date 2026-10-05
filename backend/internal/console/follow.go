package console

import (
	"bufio"
	"context"
	"io"
	"strings"
	"time"

	"app/internal/gameserver"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"k8s.io/apimachinery/pkg/types"
)

// follow streams the output of a pod's main container into the hub until the container has
// exited. The Kubernetes API cuts log streams now and then; every reconnect continues at the
// last timestamp and drops lines that were already delivered.
func (h *Hub) follow(ctx context.Context, namespace, server, pod string, uid types.UID, kind Kind, m *Matcher) {
	f := &logFollower{hub: h, namespace: namespace, server: server, pod: pod, uid: uid, kind: kind, matcher: m}
	for ctx.Err() == nil {
		rc, err := f.open(ctx)
		if err != nil {
			// The container may still be starting; retry.
			sleep(ctx, 2*time.Second)
			continue
		}
		f.read(ctx, rc)
		rc.Close()
		if h.containerTerminated(ctx, namespace, pod) {
			// All output of an exited container has been delivered.
			return
		}
		sleep(ctx, time.Second)
	}
}

// logFollower is the state of one followed pod.
type logFollower struct {
	hub                    *Hub
	namespace, server, pod string
	uid                    types.UID
	kind                   Kind
	matcher                *Matcher
	dedup                  deduper
}

// open starts a log stream: the last historySize lines at first, later from the last timestamp.
func (f *logFollower) open(ctx context.Context) (io.ReadCloser, error) {
	opts := &corev1.PodLogOptions{Container: gameserver.ContainerName, Follow: true, Timestamps: true}
	if f.dedup.last.IsZero() {
		opts.TailLines = ptr.To(int64(historySize))
	} else {
		// sinceTime has second precision; duplicates are filtered by the deduper.
		opts.SinceTime = ptr.To(metav1.NewTime(f.dedup.last))
	}
	return f.hub.kube.Clientset.CoreV1().Pods(f.namespace).GetLogs(f.pod, opts).Stream(ctx)
}

// maxLineLength cuts longer output lines (timestamp included): the history keeps historySize lines per
// server in memory, and a game can print without ever ending a line.
const maxLineLength = 4096

// read delivers the lines of one stream until it ends.
func (f *logFollower) read(ctx context.Context, rc io.Reader) {
	reader := bufio.NewReaderSize(rc, maxLineLength)
	f.dedup.reconnect()
	for {
		raw, err := readLine(reader)
		if raw != "" {
			if ts, text := splitTimestamp(raw); f.dedup.accept(ts) {
				f.deliver(text)
			}
		}
		if err != nil {
			if err != io.EOF && ctx.Err() == nil {
				f.hub.log.Debug("log stream interrupted", "server", f.server, "err", err)
			}
			return
		}
	}
}

// readLine returns the next line; the part of a line beyond the reader's buffer is skipped.
func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadSlice('\n')
	if err != bufio.ErrBufferFull {
		return string(line), err
	}
	cut := strings.ToValidUTF8(string(line), "") + " …\n"
	for err == bufio.ErrBufferFull {
		_, err = r.ReadSlice('\n')
	}
	return cut, err
}

// deliver publishes a line and marks the game pod as running at the egg's "done" line.
func (f *logFollower) deliver(text string) {
	event := EventConsoleOutput
	if f.kind == KindInstall {
		event = EventInstallOutput
	}
	f.hub.Publish(f.server, Event{Event: event, Args: []string{text}})
	if f.kind != KindGame || !f.matcher.Match(text) {
		return
	}
	f.hub.mu.Lock()
	first := !f.hub.done[f.uid]
	f.hub.done[f.uid] = true
	f.hub.mu.Unlock()
	if first && f.hub.OnDone != nil {
		f.hub.OnDone(f.namespace, f.server)
	}
}

// containerTerminated reports whether the main container of a pod has exited (or the pod is gone).
func (h *Hub) containerTerminated(ctx context.Context, namespace, pod string) bool {
	p, err := h.kube.Clientset.CoreV1().Pods(namespace).Get(ctx, pod, metav1.GetOptions{})
	if err != nil {
		return apierrors.IsNotFound(err)
	}
	for _, cs := range p.Status.ContainerStatuses {
		if cs.Name == gameserver.ContainerName {
			return cs.State.Terminated != nil
		}
	}
	return p.Status.Phase == corev1.PodSucceeded || p.Status.Phase == corev1.PodFailed
}

// deduper drops lines that were already delivered before a log stream reconnect.
// Several lines can share one timestamp, so it counts the lines at the newest one.
type deduper struct {
	last      time.Time
	lastCount int
	skip      int
}

// reconnect is called before reading a new stream that starts at d.last.
func (d *deduper) reconnect() { d.skip = d.lastCount }

// accept reports whether a line with timestamp ts is new.
func (d *deduper) accept(ts time.Time) bool {
	switch {
	case ts.IsZero():
		return true
	case ts.Before(d.last):
		return false
	case ts.Equal(d.last) && d.skip > 0:
		d.skip--
		return false
	case ts.Equal(d.last):
		d.lastCount++
		return true
	default:
		d.last, d.lastCount, d.skip = ts, 1, 0
		return true
	}
}

// splitTimestamp separates the RFC3339Nano timestamp kubelet prefixes to each line.
// Carriage returns are normalized: only the text after the last \r of a line is kept.
func splitTimestamp(raw string) (time.Time, string) {
	raw = strings.TrimRight(raw, "\r\n")
	var ts time.Time
	if i := strings.IndexByte(raw, ' '); i > 0 {
		if t, err := time.Parse(time.RFC3339Nano, raw[:i]); err == nil {
			ts, raw = t, raw[i+1:]
		}
	}
	if i := strings.LastIndexByte(raw, '\r'); i >= 0 {
		raw = raw[i+1:]
	}
	return ts, raw
}

func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
