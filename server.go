package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	sloghttp "github.com/samber/slog-http"
	"github.com/tenkoh/s3c/pkg/handler"
	"github.com/tenkoh/s3c/pkg/logger"
	"github.com/tenkoh/s3c/pkg/repository"
	"github.com/tenkoh/s3c/pkg/service"
)

// Server represents the HTTP server with dependency injection
type Server struct {
	port       int
	mux        *http.ServeMux
	apiHandler *handler.APIHandler
	httpServer *http.Server
	logger     *slog.Logger
	mu         sync.RWMutex
	shutdownCh chan struct{}
}

// NewServer creates a new server with dependency injection
func NewServer(port int, appLogger *slog.Logger) *Server {
	// Initialize dependencies
	profileRepo := repository.NewFileSystemProfileRepository()

	serverLogger := logger.WithComponent(appLogger, "server")

	s := &Server{
		port:       port,
		mux:        http.NewServeMux(),
		logger:     serverLogger,
		shutdownCh: make(chan struct{}),
	}

	// Create API handler with shutdown channel reference and logger
	apiLogger := logger.WithComponent(appLogger, "api")

	// Create S3 service factory with integrated logger
	s3Logger := logger.WithComponent(appLogger, "s3service")
	s3ServiceCreator := func(ctx context.Context, cfg service.S3Config) (service.S3Operations, error) {
		return service.NewS3ServiceWithLogger(ctx, cfg, s3Logger)
	}

	apiHandler := handler.NewAPIHandlerWithShutdown(profileRepo, s3ServiceCreator, s.shutdownCh, apiLogger)
	s.apiHandler = apiHandler

	s.setupRoutes()

	serverLogger.Debug("Server initialized",
		"port", port,
		"routes", "API and static routes configured",
	)

	return s
}

// NewTestServer creates a server with mock dependencies for testing
func NewTestServer(port int, profileProvider handler.ProfileProvider, s3ServiceCreator handler.S3ServiceCreator) *Server {
	// Use default logger for tests
	testLogger := logger.NewDefaultLogger()

	s := &Server{
		port:       port,
		mux:        http.NewServeMux(),
		logger:     logger.WithComponent(testLogger, "test-server"),
		shutdownCh: make(chan struct{}),
	}

	testAPILogger := logger.WithComponent(testLogger, "test-api")
	apiHandler := handler.NewAPIHandlerWithShutdown(profileProvider, s3ServiceCreator, s.shutdownCh, testAPILogger)
	s.apiHandler = apiHandler

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// API routes with POST-unified design
	s.mux.HandleFunc("POST /api/health", s.apiHandler.HandleHealth)
	s.mux.HandleFunc("POST /api/status", s.apiHandler.HandleStatus)
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
	s.logger.Info("Starting s3c HTTP server",
		"port", s.port,
		"url", fmt.Sprintf("http://localhost:%d", s.port),
		"readTimeout", "15s",
		"writeTimeout", "15s",
	)

	// Wrap mux with HTTP logging middleware
	httpLogger := logger.WithComponent(s.logger, "http")

	// Configure slog-http middleware
	config := sloghttp.Config{
		DefaultLevel:     slog.LevelInfo,
		ClientErrorLevel: slog.LevelWarn,
		ServerErrorLevel: slog.LevelError,

		WithRequestBody:    false, // Disable for performance and security
		WithResponseBody:   false, // Disable for performance and security
		WithRequestHeader:  false, // Disable for security
		WithResponseHeader: false, // Disable for performance

		Filters: []sloghttp.Filter{
			// Exclude frequent status check endpoint to reduce log noise
			sloghttp.IgnorePath("/api/status"),
		},
	}

	middleware := sloghttp.NewWithConfig(httpLogger, config)

	s.mu.Lock()
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      middleware(s.mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
	s.mu.Unlock()

	s.logger.Info("HTTP server listening", "address", s.httpServer.Addr)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.logger.Error("HTTP server failed", "error", err)
		return err
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.RLock()
	server := s.httpServer
	s.mu.RUnlock()

	if server == nil {
		s.logger.Debug("No HTTP server to shutdown")
		return nil
	}

	s.logger.Info("Shutting down HTTP server")

	if err := server.Shutdown(ctx); err != nil {
		s.logger.Error("Error during HTTP server shutdown", "error", err)
		return err
	}

	s.logger.Info("HTTP server shutdown completed")
	return nil
}

// ShutdownChannel returns the channel used for API shutdown requests
func (s *Server) ShutdownChannel() <-chan struct{} {
	return s.shutdownCh
}
