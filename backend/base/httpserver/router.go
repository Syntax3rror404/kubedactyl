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
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/swaggo/swag"

	"app/app"
	"app/base/api"
	"app/base/auth"
	"app/base/settings"
)

// Routes is the app's API (the base routes and its own), mounted at /api.
type Routes interface {
	Register(r *gin.RouterGroup)
}

// Config configures the HTTP server.
type Config struct {
	API Routes
	// Settings can turn the API documentation off.
	Settings *settings.Store
	// Frontend is the embedded web UI (nil in development mode, where Vite serves it).
	Frontend fs.FS
	// TrustedProxies are comma separated IPs/CIDRs that may set X-Forwarded-For.
	TrustedProxies string
	Version        string
	Log            *slog.Logger
}

// NewRouter builds the HTTP routes: the API, the API document and the embedded frontend.
func NewRouter(cfg Config) (*gin.Engine, error) {
	var static *staticFiles
	if cfg.Frontend != nil {
		var err error
		if static, err = newStaticFiles(cfg.Frontend); err != nil {
			return nil, err
		}
	}
	r := gin.New()
	// nil trusts no proxy: gin trusts every proxy by default, which would let clients
	// fake their IP (login throttling) with X-Forwarded-For.
	if err := r.SetTrustedProxies(trustedProxies(cfg.TrustedProxies)); err != nil {
		return nil, err
	}
	r.Use(gin.Recovery(), requestLogger(cfg.Log), securityHeaders, limitBody)
	apiGroup := r.Group("/api")
	apiGroup.GET("/health", getHealth)
	apiGroup.GET("/info", getInfo(cfg.Version, cfg.Frontend == nil))
	apiGroup.GET("/openapi.json", getAPIDocument(cfg.Settings))
	cfg.API.Register(apiGroup)

	r.NoRoute(func(c *gin.Context) {
		// "/api" and below it; /api-docs is a page of the web UI.
		if strings.HasPrefix(c.Request.URL.Path+"/", "/api/") {
			c.JSON(http.StatusNotFound, api.ErrorResponse{Error: "not found"})
			return
		}
		if static == nil {
			c.String(http.StatusNotFound, "Development mode: the frontend is served by Vite (http://localhost:5173)")
			return
		}
		static.serve(c)
	})
	return r, nil
}

// requestLogger logs failed requests, and every request outside release mode. Websockets are left out (their
// request ends when they close).
func requestLogger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if strings.HasSuffix(c.FullPath(), "/ws") {
			return
		}
		if c.Writer.Status() >= 400 || gin.Mode() != gin.ReleaseMode {
			path := c.Request.URL.Path
			if strings.Contains(c.FullPath(), ":token") {
				path = c.FullPath() // the path carries a credential
			}
			log.Info("request", "method", c.Request.Method, "path", path,
				"status", c.Writer.Status(), "duration", time.Since(start).Round(time.Microsecond))
		}
	}
}

// trustedProxies turns "10.0.0.0/8, 192.168.1.10" into a list for gin. Without trusted proxies the client IP is
// the peer address, so X-Forwarded-For cannot be spoofed.
func trustedProxies(v string) []string {
	var out []string
	for p := range strings.SplitSeq(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
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
func getHealth(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{Status: "ok"})
}

// getInfo godoc
//
//	@Summary		Panel information
//	@Description	Name, version and run mode (public, so nothing else is revealed).
//	@Tags			System
//	@Produce		json
//	@Success		200	{object}	InfoResponse
//	@Router			/info [get]
func getInfo(version string, devMode bool) gin.HandlerFunc {
	mode := "production"
	if devMode {
		mode = "development"
	}
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, InfoResponse{Name: app.Name, Version: version, Mode: mode})
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
func getAPIDocument(store *settings.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if set, err := store.Current(c); err != nil || set.DisableAPIDocs {
			c.JSON(http.StatusNotFound, api.ErrorResponse{Error: "the API documentation is turned off"})
			return
		}
		doc, err := swag.ReadDoc()
		if err != nil {
			c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: err.Error()})
			return
		}
		c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(doc))
	}
}

// contentSecurityPolicy allows only resources of the panel itself. Inline styles are
// needed by xterm.js, CodeMirror and Base UI positioning; scripts are never inline.
const contentSecurityPolicy = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self'; worker-src 'self' blob:; " +
	"object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'"

// securityHeaders sets browser protections on every response.
func securityHeaders(c *gin.Context) {
	h := c.Writer.Header()
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("X-Frame-Options", "DENY")
	h.Set("Referrer-Policy", "same-origin")
	h.Set("Cross-Origin-Opener-Policy", "same-origin")
	h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
	h.Set("Content-Security-Policy", contentSecurityPolicy)
	// API answers hold new tokens, settings and file contents: neither the browser nor a proxy
	// may keep them. A handler that passes on another server's answer removes the header.
	if strings.HasPrefix(c.Request.URL.Path, "/api/") {
		h.Set("Cache-Control", "no-store")
	}
	// Only meaningful over HTTPS (directly or behind a TLS terminating proxy).
	if auth.HTTPS(c.Request) {
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	}
	c.Next()
}

// Uploads stream to the file container and are not limited here; everything else is small.
const maxRequestBody = 8 << 20

func limitBody(c *gin.Context) {
	if c.Request.Body != nil && !strings.HasSuffix(c.Request.URL.Path, "/files/upload") {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBody)
	}
	c.Next()
}
