//go:build dev

package web

import "io/fs"

// In development mode (go run -tags dev .) the Vite dev server serves the
// frontend and the backend only provides the API.
func FS() fs.FS {
	return nil
}
