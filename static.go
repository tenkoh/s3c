package main

import (
	"embed"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

//go:embed frontend/dist/*
var staticFiles embed.FS

// getStaticFileSystem returns the embedded filesystem for serving static files
func getStaticFileSystem() http.FileSystem {
	fsys, err := fs.Sub(staticFiles, "frontend/dist")
	if err != nil {
		panic(err)
	}
	return http.FS(fsys)
}

// serveStaticFile serves a specific file from the embedded filesystem
func (s *Server) serveStaticFile(w http.ResponseWriter, r *http.Request, filename string) {
	fsys := getStaticFileSystem()

	file, err := fsys.Open(filename)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()

	// Set content type based on file extension
	ext := filepath.Ext(filename)
	switch ext {
	case ".html":
		w.Header().Set("Content-Type", "text/html")
	case ".css":
		w.Header().Set("Content-Type", "text/css")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript")
	case ".json":
		w.Header().Set("Content-Type", "application/json")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	}

	http.ServeContent(w, r, filename, time.Time{}, file.(http.File))
}

// handleStaticFiles serves static files and SPA routing
func (s *Server) handleStaticFiles(w http.ResponseWriter, r *http.Request) {
	// Skip API routes
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}

	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}

	// Try to serve the requested file
	fsys := getStaticFileSystem()
	if _, err := fsys.Open(strings.TrimPrefix(path, "/")); err == nil {
		s.serveStaticFile(w, r, strings.TrimPrefix(path, "/"))
		return
	}

	// For SPA: serve index.html for all non-API, non-asset routes
	s.serveStaticFile(w, r, "index.html")
}
