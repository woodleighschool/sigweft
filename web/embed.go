// Package web contains the built frontend served by the Go binary.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var files embed.FS

// Files returns the embedded frontend without its build directory prefix.
func Files() fs.FS {
	root, err := fs.Sub(files, "dist")
	if err != nil {
		panic(err)
	}
	return root
}
