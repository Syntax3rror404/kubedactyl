// Package cli is the panel process around the app: the flags every panel takes, the start banner and the log
// handler.
package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/go-logr/logr"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"

	"app/app"
)

// Flags are the settings every panel takes from its command line and environment; the app adds its own flags
// next to them before flag.Parse.
type Flags struct {
	// KubeContext selects a kubeconfig context ("": the current one, or the in-cluster config).
	KubeContext string
	// Listen is the HTTP listen address.
	Listen  string
	Verbose bool
	// TrustedProxies are comma separated IPs/CIDRs of reverse proxies whose X-Forwarded-For is trusted.
	TrustedProxies string
	// Namespace is the panel namespace (users, settings, session key).
	Namespace string
	// HTTP1 forces HTTP/1.1 to the API server (environment only: unless KUBE_HTTP2 is set).
	HTTP1 bool
}

// Bind adds the flags to the command line; namespaceUsage says what the app keeps in the panel namespace.
func (f *Flags) Bind(namespaceUsage string) {
	flag.StringVar(
		&f.KubeContext, "kube-context", os.Getenv("KUBE_CONTEXT"),
		"kubeconfig context (default: current context, or in-cluster config)",
	)
	flag.StringVar(&f.Listen, "listen", ":"+EnvOr("PORT", "8080"), "HTTP listen address")
	flag.BoolVar(&f.Verbose, "verbose", os.Getenv("VERBOSE") != "", "enable debug logging")
	flag.StringVar(
		&f.TrustedProxies, "trusted-proxies", os.Getenv("TRUSTED_PROXIES"),
		"comma separated IPs/CIDRs of reverse proxies whose X-Forwarded-For is trusted",
	)
	flag.StringVar(&f.Namespace, "namespace", EnvOr(app.EnvPrefix+"_NAMESPACE", app.ID), namespaceUsage)
	f.HTTP1 = os.Getenv("KUBE_HTTP2") == ""
}

// EnvOr is the environment variable key, or fallback when it is empty.
func EnvOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Fatal logs the error and ends the process.
func Fatal(log *slog.Logger, msg string, err error) {
	log.Error(msg, "err", err)
	os.Exit(1)
}

// PrintBanner writes the logo and the version. Colors only on a terminal: log collectors would show the escape codes.
func PrintBanner(w io.Writer, version string, color bool) {
	lines := strings.Split(app.Logo, "\n")
	width := 0
	for _, line := range lines {
		width = max(width, utf8.RuneCountInString(line))
	}
	for _, line := range lines {
		if color {
			line = gradient(line, width)
		}
		fmt.Fprintln(w, line)
	}
	if version != "dev" {
		version = "v" + version
	}
	fmt.Fprintf(w, "%s · %s\n\n", version, app.Tagline)
}

// gradient colors every character of line by its column with the brand gradient (24-bit terminal colors).
func gradient(line string, width int) string {
	var b strings.Builder
	for i, r := range []rune(line) {
		mix := func(k int) int { return app.BrandFrom[k] + (app.BrandTo[k]-app.BrandFrom[k])*i/max(width-1, 1) }
		fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm%c", mix(0), mix(1), mix(2), r)
	}
	return b.String() + "\x1b[0m"
}

// ColorTerminal reports whether f is a terminal that wants colors (NO_COLOR unset).
func ColorTerminal(f *os.File) bool {
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0 && os.Getenv("NO_COLOR") == ""
}

// SetupLogging routes slog, controller-runtime and klog through one handler on stderr.
func SetupLogging(verbose bool) *slog.Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	log := slog.New(quietHandler{slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})})
	slog.SetDefault(log)
	logger := logr.FromSlogHandler(log.Handler())
	ctrl.SetLogger(logger)
	klog.SetLogger(logger)
	return log
}

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
