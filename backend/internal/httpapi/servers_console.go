package httpapi

import (
	"context"
	"errors"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/gin-gonic/gin"

	"app/internal/console"
	"app/internal/serverctl"
)

// openConsole godoc
//
//	@Summary		Open the console
//	@Description	Streams events {"event": "...", "args": [...]}: "console output", "install output", "daemon
//	@Description	message", "status". Send {"event":"send command","args":["say hi"]} or {"event":"set
//	@Description	state","args":["start"]}.
//	@Tags			Servers
//	@Param			server	path	string	true	"Server name"
//	@Success		101
//	@Security		BearerAuth
//	@Router			/servers/{server}/ws [get]
func (a *API) openConsole(c *gin.Context) {
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	// Only same-origin connections; in development the Vite dev server proxies from another port.
	opts := &websocket.AcceptOptions{}
	if a.DevMode {
		opts.OriginPatterns = []string{"localhost:*", "127.0.0.1:*"}
	}
	conn, err := websocket.Accept(c.Writer, c.Request, opts)
	if err != nil {
		return
	}
	defer func() { _ = conn.CloseNow() }()
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	history, events, unsubscribe := a.Hub.Subscribe(gs.Name)
	defer unsubscribe()
	s := &consoleSession{
		api: a, conn: conn, token: credentials(c), caller: principal(c), namespace: gs.Namespace, server: gs.Name,
	}
	if err := s.sendStart(ctx, string(gs.Status.Phase), history); err != nil {
		return
	}
	go func() {
		defer cancel()
		s.readCommands(ctx)
	}()
	s.stream(ctx, events)
}

// consoleSession is one open console websocket of a server.
type consoleSession struct {
	api  *API
	conn *websocket.Conn
	// token is the credential the connection was opened with; it is checked again regularly.
	token string
	// caller opened the connection (for the audit log).
	caller    *Principal
	namespace string
	server    string
}

// authorized checks the credentials of the connection again: a logout, "log out everywhere",
// a password change, a disabled account or a lost admin role ends an open console.
func (s *consoleSession) authorized(ctx context.Context) bool {
	p, err := s.api.principalFor(ctx, s.token)
	return err == nil && (p.Admin() || p.Namespace == s.namespace)
}

// end closes a connection whose credentials are no longer valid.
func (s *consoleSession) end() {
	s.conn.Close(websocket.StatusPolicyViolation, "session ended")
}

// write sends one event (the websocket library allows concurrent writers).
func (s *consoleSession) write(ctx context.Context, ev console.Event) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return wsjson.Write(ctx, s.conn, ev)
}

func (s *consoleSession) daemonMessage(ctx context.Context, text string) {
	_ = s.write(ctx, console.Event{Event: console.EventDaemonMessage, Args: []string{text}})
}

// sendStart sends the current phase and the output history.
func (s *consoleSession) sendStart(ctx context.Context, phase string, history []console.Event) error {
	if err := s.write(ctx, console.Event{Event: console.EventStatus, Args: []string{phase}}); err != nil {
		return err
	}
	for _, ev := range history {
		if err := s.write(ctx, ev); err != nil {
			return err
		}
	}
	return nil
}

// readCommands handles client messages until the connection closes:
// {"event":"send command","args":["say hi"]} and {"event":"set state","args":["start"]}.
func (s *consoleSession) readCommands(ctx context.Context) {
	for {
		var msg console.Event
		if err := wsjson.Read(ctx, s.conn, &msg); err != nil {
			return
		}
		if len(msg.Args) == 0 {
			continue
		}
		if !s.authorized(ctx) {
			s.end()
			return
		}
		if _, err := s.api.admit(s.caller); err != nil {
			s.daemonMessage(ctx, err.Error())
			continue
		}
		s.handle(ctx, msg)
	}
}

// handle runs one client message on the server.
func (s *consoleSession) handle(ctx context.Context, msg console.Event) {
	// Reload for every message: permissions and state (suspended, running) may have changed.
	gs, ok := s.api.reloadServer(ctx, s.namespace, s.server)
	if !ok {
		return
	}
	switch msg.Event {
	case "send command":
		if err := s.api.Ops.Command(ctx, gs, msg.Args[0]); err != nil {
			text := err.Error()
			if errors.Is(err, serverctl.ErrOffline) {
				text = "Server is not running, command was not sent."
			}
			s.daemonMessage(ctx, text)
			return
		}
		s.api.auditAs(s.caller, "console command sent", "server", gs.Name, "command", msg.Args[0])
	case "set state":
		if err := s.api.Ops.Power(ctx, gs, msg.Args[0]); err != nil {
			s.daemonMessage(ctx, err.Error())
		} else {
			s.api.auditAs(s.caller, "power action sent", "server", gs.Name, "signal", msg.Args[0])
		}
		s.api.Trigger(gs.Namespace, gs.Name)
	}
}

// stream forwards hub events and pings the client every 30 seconds.
func (s *consoleSession) stream(ctx context.Context, events <-chan console.Event) {
	ping := time.NewTicker(30 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-ctx.Done():
			s.conn.Close(websocket.StatusNormalClosure, "")
			return
		case ev := <-events:
			if err := s.write(ctx, ev); err != nil {
				return
			}
		case <-ping.C:
			if !s.authorized(ctx) {
				s.end()
				return
			}
			pctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := s.conn.Ping(pctx)
			cancel()
			if err != nil {
				return
			}
		}
	}
}
