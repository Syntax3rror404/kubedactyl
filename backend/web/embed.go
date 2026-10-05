//go:build !dev

package web

import (
	"embed"
	"io/fs"
)

// The Vite build (npm run build) writes to web/dist, which is embedded
// into the binary at compile time.
//
//go:embed all:dist
var dist embed.FS

// FS returns the embedded frontend, or nil in development mode.
func FS() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
