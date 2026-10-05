package main

import (
	"context"
	"log/slog"
	"strings"
)

// quietHandler drops noisy client-go messages that are expected when the panel
// closes attach streams on purpose.
type quietHandler struct{ slog.Handler }

var ignoredMessages = []string{
	"Waiting for server to close stdin failed",
	"Copying stdout failed",
	"Copying stdin failed",
	// Heartbeat of an attach connection the panel closed right after writing a command.
	"Websocket Ping failed",
}

func (h quietHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, m := range ignoredMessages {
		if strings.Contains(r.Message, m) {
			return nil
		}
	}
	return h.Handler.Handle(ctx, r)
}

func (h quietHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return quietHandler{h.Handler.WithAttrs(attrs)}
}

func (h quietHandler) WithGroup(name string) slog.Handler {
	return quietHandler{h.Handler.WithGroup(name)}
}
