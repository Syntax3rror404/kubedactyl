package httpserver

import (
	"net/http"
	"strings"

	"app/internal/auth"

	"github.com/gin-gonic/gin"
)

// contentSecurityPolicy allows only resources of the panel itself. Inline styles are
// needed by xterm.js, CodeMirror and Radix positioning; scripts are never inline.
const contentSecurityPolicy = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self'; worker-src 'self' blob:; " +
	"object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'"

// securityHeaders sets browser protections on every response. The Swagger UI uses inline
// scripts and therefore gets no CSP (it only documents the API, all calls need a token).
func securityHeaders(c *gin.Context) {
	h := c.Writer.Header()
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("X-Frame-Options", "DENY")
	h.Set("Referrer-Policy", "same-origin")
	h.Set("Cross-Origin-Opener-Policy", "same-origin")
	h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
	if !strings.HasPrefix(c.Request.URL.Path, "/swagger/") {
		h.Set("Content-Security-Policy", contentSecurityPolicy)
	}
	// API answers hold new tokens, settings and file contents: neither the browser nor a proxy
	// may keep them.
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

// ParseTrustedProxies turns "10.0.0.0/8, 192.168.1.10" into a list for gin. Without trusted
// proxies the client IP is the peer address, so X-Forwarded-For cannot be spoofed.
func ParseTrustedProxies(v string) []string {
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
