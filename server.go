package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Server struct {
	port     int
	mux      *http.ServeMux
	appState *AppState
}

func NewServer(port int) *Server {
	s := &Server{
		port:     port,
		mux:      http.NewServeMux(),
		appState: NewAppState(),
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// API routes
	s.mux.HandleFunc("/api/health", s.handleHealth)
	s.mux.HandleFunc("/api/profiles", s.handleAPIProfiles)
	s.mux.HandleFunc("/api/settings", s.handleAPISettings)
	s.mux.HandleFunc("/api/buckets", s.handleAPIBuckets)
	s.mux.HandleFunc("/api/shutdown", s.handleAPIShutdown)

	// Serve static files and SPA routing
	s.mux.HandleFunc("/", s.handleStaticFiles)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
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
