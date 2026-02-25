package VirtualRouterServer

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static/*
var monitorStaticFiles embed.FS

func monitorStaticHandler() http.Handler {
	staticFS, err := fs.Sub(monitorStaticFiles, "static")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "static assets not found", http.StatusInternalServerError)
		})
	}
	return http.FileServer(http.FS(staticFS))
}
