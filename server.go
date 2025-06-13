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
	s3Factory := service.NewAWSS3ServiceFactory()

	deps := &handler.Dependencies{
		ProfileProvider:  profileRepo,
		S3ServiceFactory: s3Factory,
	}

	apiHandler := handler.NewAPIHandler(deps)

	s := &Server{
		port:       port,
		mux:        http.NewServeMux(),
		apiHandler: apiHandler,
	}
	
	s.setupRoutes()
	return s
}

// NewTestServer creates a server with mock dependencies for testing
func NewTestServer(port int, deps *handler.Dependencies) *Server {
	apiHandler := handler.NewAPIHandler(deps)

	s := &Server{
		port:       port,
		mux:        http.NewServeMux(),
		apiHandler: apiHandler,
	}
	
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// API routes with proper dependency injection
	s.mux.HandleFunc("/api/health", s.apiHandler.HandleHealth)
	s.mux.HandleFunc("/api/profiles", s.apiHandler.HandleProfiles)
	s.mux.HandleFunc("/api/settings", s.apiHandler.HandleSettings)
	s.mux.HandleFunc("/api/buckets", s.apiHandler.HandleBuckets)
	s.mux.HandleFunc("/api/objects", s.apiHandler.HandleObjects)
	s.mux.HandleFunc("/api/objects/delete", s.apiHandler.HandleDeleteObjects)
	s.mux.HandleFunc("/api/shutdown", s.apiHandler.HandleShutdown)

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