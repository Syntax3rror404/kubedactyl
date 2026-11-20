// Package httpserver is the HTTP server of the panel: the app's API, the API document, the embedded web UI and the
// security middleware.
package httpserver

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/swaggo/swag"

	"app/app"
	"app/base/api"
	"app/base/auth"
	"app/base/settings"
)

// Routes is the app's API (the base routes and its own), mounted at /api.
type Routes interface {
	Register(m *api.Mux)
}

// Config configures the HTTP server.
type Config struct {
	API Routes
	// Settings can turn the API documentation off.
	Settings *settings.Store
	// Frontend is the embedded web UI (nil in development mode, where Vite serves it).
	Frontend fs.FS
	Version  string
	Log      *slog.Logger
}

// NewRouter builds the HTTP handler: the API, the API document and the embedded frontend.
func NewRouter(cfg Config) (http.Handler, error) {
	var static *staticFiles
	if cfg.Frontend != nil {
		var err error
		if static, err = newStaticFiles(cfg.Frontend); err != nil {
			return nil, err
		}
	}
	mux := NewAPI(cfg)
	// Only the API goes through the mux: it redirects paths like "a//b", which are client routes of the web UI.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		// "/api" and below it; /api-docs is a page of the web UI.
		case strings.HasPrefix(r.URL.Path+"/", "/api/"):
			mux.ServeHTTP(w, r)
		case static == nil:
			http.Error(w, "Development mode: the frontend is served by Vite (http://localhost:5173)",
				http.StatusNotFound)
		default:
			static.serve(w, r)
		}
	})
	// The log sees every answer, also the 500 of a panic.
	return api.Chain(handler,
		logRequests(cfg.Log, cfg.Frontend == nil, mux),
		recoverPanics(cfg.Log, mux),
		securityHeaders,
		prepareRequest(mux),
	), nil
}

// NewAPI registers the API routes (below /api) on a new mux; unknown API paths are answered with 404.
func NewAPI(cfg Config) *http.ServeMux {
	mux := http.NewServeMux()
	m := api.NewMux(mux, "/api")
	m.Handle("GET /health", getHealth)
	m.Handle("GET /info", getInfo(cfg.Version, cfg.Frontend == nil))
	m.Handle("GET /openapi.json", getAPIDocument(cfg.Settings))
	cfg.API.Register(m)
	mux.HandleFunc("/api", notFound) // the mux would redirect it to "/api/"
	mux.HandleFunc("/api/", notFound)
	return mux
}

func notFound(w http.ResponseWriter, _ *http.Request) {
	api.JSON(w, http.StatusNotFound, api.ErrorResponse{Error: "not found"})
}

// route is the pattern of the API route a request goes to ("/api/servers/{server}/ws"), empty for none. Path
// values may hold an encoded slash ("%2F"), so decisions by route are made with it, never with the path.
func route(routes *http.ServeMux, r *http.Request) string {
	_, pattern := routes.Handler(r)
	if _, path, ok := strings.Cut(pattern, " "); ok {
		return path
	}
	return pattern
}

// statusRecorder remembers the status of an answer for the log.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.status == 0 {
		s.status = code
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	return s.ResponseWriter.Write(b)
}

// Unwrap gives http.ResponseController and websockets the connection (flush, hijack).
func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// logRequests logs failed requests, and with all every request. Websockets are left out (their request ends when
// they close).
func logRequests(log *slog.Logger, all bool, routes *http.ServeMux) api.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)
			if rec.status < 400 && !all {
				return
			}
			rt := route(routes, r)
			if strings.HasSuffix(rt, "/ws") {
				return
			}
			log.Info("request", "method", r.Method, "path", loggedPath(rt, r),
				"status", rec.status, "duration", time.Since(start).Round(time.Microsecond))
		})
	}
}

// loggedPath is the path of a request for the log: a route with a credential in its path ({token}) is logged by
// its pattern instead.
func loggedPath(route string, r *http.Request) string {
	if strings.Contains(route, "{token}") {
		return route
	}
	return r.URL.Path
}

// recoverPanics answers a request whose handler panicked with 500 and logs the panic, instead of closing the
// connection without an answer.
func recoverPanics(log *slog.Logger, routes *http.ServeMux) api.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				v := recover()
				if v == nil {
					return
				}
				if v == http.ErrAbortHandler {
					panic(v)
				}
				log.Error("handler panicked", "path", loggedPath(route(routes, r), r), "panic", v,
					"stack", string(debug.Stack()))
				w.WriteHeader(http.StatusInternalServerError)
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// Uploads stream to the file container and are not limited here; everything else is small.
const maxRequestBody = 8 << 20

// prepareRequest gives the handlers their request, copied at most once (the request a middleware is given stays
// as it is). A request that changes something runs to its end when the client goes away: a change cut off half
// way would leave objects behind; reads stop with the client. A body is limited to maxRequestBody, but on the
// upload routes (decided by the route, not the path).
func prepareRequest(routes *http.ServeMux) api.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			change := r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions
			limit := r.Body != nil && r.Body != http.NoBody && !strings.HasSuffix(route(routes, r), "/files/upload")
			if !change && !limit {
				next.ServeHTTP(w, r)
				return
			}
			ctx := r.Context()
			if change {
				ctx = context.WithoutCancel(ctx)
			}
			r = r.WithContext(ctx)
			if limit {
				r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Serve runs the HTTP server until ctx ends and then shuts it down gracefully.
func Serve(ctx context.Context, addr string, handler http.Handler) error {
	// No ReadTimeout: uploads stream for as long as they take.
	srv := &http.Server{
		Addr: addr, Handler: handler, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 2 * time.Minute,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// LocalURL is the address of the web interface on this machine (for the log), built from the listen address.
func LocalURL(addr string) string {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "http://localhost"
	}
	return "http://" + net.JoinHostPort("localhost", port)
}

// HealthResponse is returned by GET /api/health.
type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}

// InfoResponse is returned by GET /api/info.
type InfoResponse struct {
	Name    string `json:"name"    example:"My Panel"`
	Version string `json:"version" example:"0.1.0"`
	Mode    string `json:"mode"    example:"production" enums:"development,production"`
}

// getHealth godoc
//
//	@Summary		Health check
//	@Description	Reports whether the panel process is running.
//	@Tags			System
//	@Produce		json
//	@Success		200	{object}	HealthResponse
//	@Router			/health [get]
func getHealth(w http.ResponseWriter, _ *http.Request) {
	api.JSON(w, http.StatusOK, HealthResponse{Status: "ok"})
}

// getInfo godoc
//
//	@Summary		Panel information
//	@Description	Name, version and run mode (public, so nothing else is revealed).
//	@Tags			System
//	@Produce		json
//	@Success		200	{object}	InfoResponse
//	@Router			/info [get]
func getInfo(version string, devMode bool) http.HandlerFunc {
	mode := "production"
	if devMode {
		mode = "development"
	}
	return func(w http.ResponseWriter, _ *http.Request) {
		api.JSON(w, http.StatusOK, InfoResponse{Name: app.Name, Version: version, Mode: mode})
	}
}

// getAPIDocument godoc
//
//	@Summary		API document
//	@Description	This documentation as OpenAPI 2.0 document, read by the API reference page (public, unless an
//	@Description	administrator turned the API documentation off).
//	@Tags			System
//	@Produce		json
//	@Success		200
//	@Failure		404	{object}	api.ErrorResponse
//	@Router			/openapi.json [get]
func getAPIDocument(store *settings.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if set, err := store.Current(r.Context()); err != nil || set.DisableAPIDocs {
			api.JSON(w, http.StatusNotFound, api.ErrorResponse{Error: "the API documentation is turned off"})
			return
		}
		doc, err := swag.ReadDoc()
		if err != nil {
			api.JSON(w, http.StatusInternalServerError, api.ErrorResponse{Error: err.Error()})
			return
		}
		api.Data(w, http.StatusOK, "application/json; charset=utf-8", []byte(doc))
	}
}

// contentSecurityPolicy allows only resources of the panel itself. Inline styles are
// needed by xterm.js, CodeMirror and Base UI positioning; scripts are never inline.
const contentSecurityPolicy = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self'; worker-src 'self' blob:; " +
	"object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'"

// securityHeaders sets browser protections on every response.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
		h.Set("Content-Security-Policy", contentSecurityPolicy)
		// API answers hold new tokens, settings and file contents: neither the browser nor a proxy
		// may keep them. A handler that passes on another server's answer removes the header.
		if strings.HasPrefix(r.URL.Path, "/api/") {
			h.Set("Cache-Control", "no-store")
		}
		// Only meaningful over HTTPS (directly or behind a TLS terminating proxy).
		if auth.HTTPS(r) {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}
