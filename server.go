package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Server struct {
	port int
	mux  *http.ServeMux
}

func NewServer(port int) *Server {
	s := &Server{
		port: port,
		mux:  http.NewServeMux(),
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// API routes
	s.mux.HandleFunc("/api/health", s.handleHealth)

	// Serve static files (placeholder for now)
	s.mux.HandleFunc("/", s.handleIndex)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	// For now, serve a simple HTML page
	html := `<!DOCTYPE html>
<html>
<head>
    <title>s3c - S3 Client</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
</head>
<body>
    <h1>s3c - S3 Client</h1>
    <p>S3 and S3 compatible object storage Client working locally.</p>
    <p>Frontend will be integrated here soon.</p>
    <p><a href="/api/health">API Health Check</a></p>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func (s *Server) Start() error {
	fmt.Printf("Starting s3c server on port %d\n", s.port)
	fmt.Printf("Open http://localhost:%d in your browser\n", s.port)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      s.mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	return server.ListenAndServe()
}
