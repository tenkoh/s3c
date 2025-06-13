package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/tenkoh/s3c/pkg/service"
)

// ProfileProvider interface for dependency injection
type ProfileProvider interface {
	GetProfiles() ([]string, error)
}

// S3ServiceFactory interface for creating S3 services
type S3ServiceFactory interface {
	CreateS3Service(ctx context.Context, cfg service.S3Config) (service.S3Operations, error)
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Dependencies holds all the dependencies for the API handler
type Dependencies struct {
	ProfileProvider  ProfileProvider
	S3ServiceFactory S3ServiceFactory
}

// APIHandler handles API requests with dependency injection
type APIHandler struct {
	deps      *Dependencies
	s3Service service.S3Operations // Current S3 service instance
}

// NewAPIHandler creates a new API handler with dependencies
func NewAPIHandler(deps *Dependencies) *APIHandler {
	return &APIHandler{
		deps: deps,
	}
}

// HandleProfiles handles GET /api/profiles
func (h *APIHandler) HandleProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	profiles, err := h.deps.ProfileProvider.GetProfiles()
	if err != nil {
		h.writeError(w, fmt.Sprintf("Failed to read AWS profiles: %v", err), http.StatusInternalServerError)
		return
	}

	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"profiles": profiles,
		},
	}

	h.writeResponse(w, response)
}

// HandleSettings handles POST /api/settings
func (h *APIHandler) HandleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var config service.S3Config
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		h.writeError(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if config.Profile == "" {
		h.writeError(w, "Profile is required", http.StatusBadRequest)
		return
	}
	if config.Region == "" {
		h.writeError(w, "Region is required", http.StatusBadRequest)
		return
	}

	// Create S3 service with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s3Service, err := h.deps.S3ServiceFactory.CreateS3Service(ctx, config)
	if err != nil {
		h.writeError(w, fmt.Sprintf("Failed to create S3 service: %v", err), http.StatusInternalServerError)
		return
	}

	// Test connection
	if err := s3Service.TestConnection(ctx); err != nil {
		h.writeError(w, fmt.Sprintf("Failed to connect to S3: %v", err), http.StatusBadRequest)
		return
	}

	// Store the service
	h.s3Service = s3Service

	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"message": "S3 connection configured successfully",
		},
	}

	h.writeResponse(w, response)
}

// HandleBuckets handles GET /api/buckets
func (h *APIHandler) HandleBuckets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.s3Service == nil {
		h.writeError(w, "S3 service not configured", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	buckets, err := h.s3Service.ListBuckets(ctx)
	if err != nil {
		h.writeError(w, fmt.Sprintf("Failed to list buckets: %v", err), http.StatusInternalServerError)
		return
	}

	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"buckets": buckets,
		},
	}

	h.writeResponse(w, response)
}

// HandleShutdown handles POST /api/shutdown
func (h *APIHandler) HandleShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"message": "Server shutting down",
		},
	}

	h.writeResponse(w, response)

	// Shutdown the server gracefully
	go func() {
		time.Sleep(100 * time.Millisecond) // Give time for response to be sent
		os.Exit(0)
	}()
}

// HandleHealth handles GET /api/health
func (h *APIHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		},
	}
	h.writeResponse(w, response)
}

// Helper methods

func (h *APIHandler) writeResponse(w http.ResponseWriter, response APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (h *APIHandler) writeError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	response := APIResponse{
		Success: false,
		Error:   message,
	}
	
	json.NewEncoder(w).Encode(response)
}