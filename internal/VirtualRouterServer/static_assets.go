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
	fileServer := http.FileServer(http.FS(staticFS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			token := extractToken(r)
			if token == "" || !ValidateToken(token) {
				http.Redirect(w, r, "/login.html", http.StatusFound)
				return
			}
		}
		fileServer.ServeHTTP(w, r)
	})
}
