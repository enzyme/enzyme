package web

import "embed"

//go:embed all:dist
var dist embed.FS

// HasContent reports whether the embedded filesystem contains the built web client.
func HasContent() bool {
	_, err := dist.Open("dist/index.html")
	return err == nil
}
