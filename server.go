package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/tenkoh/s3c/pkg/handler"
	"github.com/tenkoh/s3c/pkg/repository"
	"github.com/tenkoh/s3c/pkg/service"
)

// Server represents the HTTP server with dependency injection
type Server struct {
	port       int
	mux        *http.ServeMux
	apiHandler *handler.APIHandler
}

// NewServer creates a new server with dependency injection
func NewServer(port int) *Server {
	// Initialize dependencies
	profileRepo := repository.NewFileSystemProfileRepository()

	apiHandler := handler.NewAPIHandler(profileRepo, service.NewS3Service)

	s := &Server{
		port:       port,
		mux:        http.NewServeMux(),
		apiHandler: apiHandler,
	}

	s.setupRoutes()
	return s
}

// NewTestServer creates a server with mock dependencies for testing
func NewTestServer(port int, profileProvider handler.ProfileProvider, s3ServiceCreator handler.S3ServiceCreator) *Server {
	apiHandler := handler.NewAPIHandler(profileProvider, s3ServiceCreator)

	s := &Server{
		port:       port,
		mux:        http.NewServeMux(),
		apiHandler: apiHandler,
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// API routes with POST-unified design
	s.mux.HandleFunc("POST /api/health", s.apiHandler.HandleHealth)
	s.mux.HandleFunc("POST /api/profiles", s.apiHandler.HandleProfiles)
	s.mux.HandleFunc("POST /api/settings", s.apiHandler.HandleSettings)
	s.mux.HandleFunc("POST /api/buckets", s.apiHandler.HandleBuckets)
	s.mux.HandleFunc("POST /api/objects/list", s.apiHandler.HandleObjectsList)
	s.mux.HandleFunc("POST /api/objects/delete", s.apiHandler.HandleObjectsDelete)
	s.mux.HandleFunc("POST /api/objects/upload", s.apiHandler.HandleObjectsUpload)
	s.mux.HandleFunc("POST /api/objects/download", s.apiHandler.HandleObjectsDownload)
	s.mux.HandleFunc("POST /api/shutdown", s.apiHandler.HandleShutdown)

	// Serve static files and SPA routing
	s.mux.HandleFunc("/", s.handleStaticFiles)
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
