// Package httpserver builds the HTTP server of the panel: API routes, Swagger UI, the
// embedded web UI and the security middleware.
package httpserver

import (
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"app/internal/httpapi"
)

func requestLogger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if strings.HasSuffix(c.FullPath(), "/ws") {
			return
		}
		if c.Writer.Status() >= 400 || gin.Mode() != gin.ReleaseMode {
			log.Info("request", "method", c.Request.Method, "path", c.Request.URL.Path,
				"status", c.Writer.Status(), "duration", time.Since(start).Round(time.Microsecond))
		}
	}
}

// Config configures the HTTP server.
type Config struct {
	API *httpapi.API
	// Frontend is the embedded web UI (nil in development mode, where Vite serves it).
	Frontend fs.FS
	// TrustedProxies may set X-Forwarded-For (see ParseTrustedProxies).
	TrustedProxies []string
	Name, Version  string
	Log            *slog.Logger
}

// NewRouter builds the HTTP routes: the API, the Swagger UI and the embedded frontend.
func NewRouter(cfg Config) (*gin.Engine, error) {
	api, frontend, log := cfg.API, cfg.Frontend, cfg.Log
	r := gin.New()
	// nil trusts no proxy: gin trusts every proxy by default, which would let clients
	// fake their IP (login throttling) with X-Forwarded-For.
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, err
	}
	r.Use(gin.Recovery(), requestLogger(log), securityHeaders, limitBody)
	apiGroup := r.Group("/api")
	apiGroup.GET("/health", getHealth)
	apiGroup.GET("/info", getInfo(cfg.Name, cfg.Version, frontend == nil))
	api.Register(apiGroup)

	// Swagger UI (fully embedded, works offline), unless an administrator turned it off.
	swaggerUI := ginSwagger.WrapHandler(swaggerFiles.Handler)
	r.GET("/swagger/*any", func(c *gin.Context) {
		if set, err := api.Settings.Current(c); err != nil || set.DisableAPIDocs {
			c.String(http.StatusNotFound, "The API documentation is turned off.")
			return
		}
		if c.Param("any") == "/" {
			c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
			return
		}
		swaggerUI(c)
	})

	var static *staticFiles
	if frontend != nil {
		var err error
		if static, err = newStaticFiles(frontend); err != nil {
			return nil, err
		}
	}
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, httpapi.ErrorResponse{Error: "not found"})
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
