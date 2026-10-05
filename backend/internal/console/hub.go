// Package console streams server output to websocket clients and detects the
// "server started" line of an egg (config.startup.done).
package console

import (
	"context"
	"log/slog"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"app/internal/kube"
)

// Event is a websocket message: an event name and its arguments.
type Event struct {
	Event string   `json:"event"`
	Args  []string `json:"args,omitempty"`
}

// Event names.
const (
	EventConsoleOutput = "console output"
	EventInstallOutput = "install output"
	EventDaemonMessage = "daemon message"
	EventStatus        = "status"
)

// Kind of pod whose output is followed.
type Kind int

const (
	KindGame Kind = iota
	KindInstall
)

const historySize = 1000

type session struct {
	uid    types.UID
	cancel context.CancelFunc
}

type stream struct {
	mu      sync.Mutex
	history []Event
	subs    map[chan Event]struct{}
	session *session
}

// Hub keeps one output stream per game server.
type Hub struct {
	kube *kube.Client
	log  *slog.Logger
	// OnDone is called when a game pod printed its "done" line.
	OnDone func(namespace, server string)

	mu      sync.Mutex
	streams map[string]*stream
	done    map[types.UID]bool
}

// NewHub creates the hub of the panel (one output stream per server).
func NewHub(k *kube.Client, log *slog.Logger) *Hub {
	return &Hub{kube: k, log: log, streams: map[string]*stream{}, done: map[types.UID]bool{}}
}

func (h *Hub) stream(server string) *stream {
	h.mu.Lock()
	defer h.mu.Unlock()
	s, ok := h.streams[server]
	if !ok {
		s = &stream{subs: map[chan Event]struct{}{}}
		h.streams[server] = s
	}
	return s
}

// Subscribe returns the output history and a channel for new events.
func (h *Hub) Subscribe(server string) ([]Event, <-chan Event, func()) {
	s := h.stream(server)
	ch := make(chan Event, 256)
	s.mu.Lock()
	history := append([]Event(nil), s.history...)
	s.subs[ch] = struct{}{}
	s.mu.Unlock()
	return history, ch, func() {
		s.mu.Lock()
		delete(s.subs, ch)
		s.mu.Unlock()
	}
}

// Publish sends an event to all subscribers; output events are kept in the history.
func (h *Hub) Publish(server string, ev Event) {
	s := h.stream(server)
	s.mu.Lock()
	defer s.mu.Unlock()
	if ev.Event != EventStatus {
		s.history = append(s.history, ev)
		if len(s.history) > historySize {
			s.history = s.history[len(s.history)-historySize:]
		}
	}
	for ch := range s.subs {
		select {
		case ch <- ev:
		default: // slow client: drop the event rather than blocking the stream
		}
	}
}

// Daemon publishes a message of the panel itself (shown as "[Kubedactyl]" in the console).
func (h *Hub) Daemon(server, msg string) {
	h.Publish(server, Event{Event: EventDaemonMessage, Args: []string{msg}})
}

// Status publishes a phase change.
func (h *Hub) Status(server, phase string) {
	h.Publish(server, Event{Event: EventStatus, Args: []string{phase}})
}

// IsDone reports whether the pod printed its done line.
func (h *Hub) IsDone(uid types.UID) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.done[uid]
}

// Follow makes sure the output of pod is streamed. It is a no-op when the pod is
// already followed.
func (h *Hub) Follow(server string, pod *corev1.Pod, kind Kind, m *Matcher) {
	s := h.stream(server)
	s.mu.Lock()
	if s.session != nil && s.session.uid == pod.UID {
		s.mu.Unlock()
		return
	}
	if s.session != nil {
		s.session.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.session = &session{uid: pod.UID, cancel: cancel}
	s.mu.Unlock()
	go h.follow(ctx, pod.Namespace, server, pod.Name, pod.UID, kind, m)
}

// StopFollow ends the current output stream of a server.
func (h *Hub) StopFollow(server string) {
	s := h.stream(server)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session != nil {
		s.session.cancel()
		s.session = nil
	}
}

// Remove forgets a deleted server.
func (h *Hub) Remove(server string) {
	h.StopFollow(server)
	h.mu.Lock()
	delete(h.streams, server)
	h.mu.Unlock()
}

// ForgetPod releases the done state of a pod that no longer exists.
func (h *Hub) ForgetPod(uid types.UID) {
	h.mu.Lock()
	delete(h.done, uid)
	h.mu.Unlock()
}
