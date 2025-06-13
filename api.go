package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"
)

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// AppState holds the application state
type AppState struct {
	s3Service     *S3Service
	profileReader *ProfileReader
}

// NewAppState creates a new application state
func NewAppState() *AppState {
	return &AppState{
		profileReader: NewProfileReader(),
	}
}

// API handlers

func (s *Server) handleAPIProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	profiles, err := s.appState.profileReader.GetProfiles()
	if err != nil {
		s.writeAPIError(w, "Failed to read AWS profiles", http.StatusInternalServerError)
		return
	}

	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"profiles": profiles,
		},
	}

	s.writeAPIResponse(w, response)
}

func (s *Server) handleAPISettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handlePostSettings(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handlePostSettings(w http.ResponseWriter, r *http.Request) {
	var config S3Config
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		s.writeAPIError(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if config.Profile == "" {
		s.writeAPIError(w, "Profile is required", http.StatusBadRequest)
		return
	}
	if config.Region == "" {
		s.writeAPIError(w, "Region is required", http.StatusBadRequest)
		return
	}

	// Create S3 service with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s3Service, err := NewS3Service(ctx, config)
	if err != nil {
		s.writeAPIError(w, "Failed to create S3 service: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Test connection
	if err := s3Service.TestConnection(ctx); err != nil {
		s.writeAPIError(w, "Failed to connect to S3: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Store the service in app state
	s.appState.s3Service = s3Service

	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"message": "S3 connection configured successfully",
		},
	}

	s.writeAPIResponse(w, response)
}

func (s *Server) handleAPIBuckets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.appState.s3Service == nil {
		s.writeAPIError(w, "S3 service not configured", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	buckets, err := s.appState.s3Service.ListBuckets(ctx)
	if err != nil {
		s.writeAPIError(w, "Failed to list buckets: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"buckets": buckets,
		},
	}

	s.writeAPIResponse(w, response)
}

func (s *Server) handleAPIShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"message": "Server shutting down",
		},
	}

	s.writeAPIResponse(w, response)

	// Shutdown the server gracefully
	go func() {
		time.Sleep(100 * time.Millisecond) // Give time for response to be sent
		os.Exit(0)
	}()
}

// Helper methods

func (s *Server) writeAPIResponse(w http.ResponseWriter, response APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (s *Server) writeAPIError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := APIResponse{
		Success: false,
		Error:   message,
	}

	json.NewEncoder(w).Encode(response)
}
