package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/tenkoh/s3c/pkg/service"
)

// ProfileProvider interface for dependency injection
type ProfileProvider interface {
	GetProfiles() ([]string, error)
}

// S3ServiceCreator is a function type for creating S3 services
type S3ServiceCreator func(ctx context.Context, cfg service.S3Config) (service.S3Operations, error)

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Dependencies holds all the dependencies for the API handler
type Dependencies struct {
	ProfileProvider  ProfileProvider
	S3ServiceCreator S3ServiceCreator
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

	s3Service, err := h.deps.S3ServiceCreator(ctx, config)
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

// HandleObjects handles GET /api/objects?bucket=<bucket>&prefix=<prefix>&delimiter=<delimiter>&maxKeys=<maxKeys>&continuationToken=<token>
func (h *APIHandler) HandleObjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.s3Service == nil {
		h.writeError(w, "S3 service not configured", http.StatusBadRequest)
		return
	}

	// Parse query parameters
	bucket := r.URL.Query().Get("bucket")
	if bucket == "" {
		h.writeError(w, "Bucket parameter is required", http.StatusBadRequest)
		return
	}

	prefix := r.URL.Query().Get("prefix")
	delimiter := r.URL.Query().Get("delimiter")
	continuationToken := r.URL.Query().Get("continuationToken")

	// Parse maxKeys parameter
	var maxKeys int32 = 100 // Default
	if maxKeysStr := r.URL.Query().Get("maxKeys"); maxKeysStr != "" {
		if parsedMaxKeys, err := json.Number(maxKeysStr).Int64(); err == nil && parsedMaxKeys > 0 && parsedMaxKeys <= 1000 {
			maxKeys = int32(parsedMaxKeys)
		}
	}

	// Create input
	input := service.ListObjectsInput{
		Bucket:            bucket,
		Prefix:            prefix,
		Delimiter:         delimiter,
		MaxKeys:           maxKeys,
		ContinuationToken: continuationToken,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	output, err := h.s3Service.ListObjects(ctx, input)
	if err != nil {
		h.writeError(w, fmt.Sprintf("Failed to list objects: %v", err), http.StatusInternalServerError)
		return
	}

	response := APIResponse{
		Success: true,
		Data:    output,
	}

	h.writeResponse(w, response)
}

// DeleteObjectRequest represents the request payload for deleting objects
type DeleteObjectRequest struct {
	Bucket string   `json:"bucket"`
	Keys   []string `json:"keys"`
}

// HandleDeleteObjects handles DELETE /api/objects
func (h *APIHandler) HandleDeleteObjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.s3Service == nil {
		h.writeError(w, "S3 service not configured", http.StatusBadRequest)
		return
	}

	var req DeleteObjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Bucket == "" {
		h.writeError(w, "Bucket is required", http.StatusBadRequest)
		return
	}
	if len(req.Keys) == 0 {
		h.writeError(w, "At least one key is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var err error
	if len(req.Keys) == 1 {
		// Single delete for efficiency
		err = h.s3Service.DeleteObject(ctx, req.Bucket, req.Keys[0])
	} else {
		// Batch delete
		err = h.s3Service.DeleteObjects(ctx, req.Bucket, req.Keys)
	}

	if err != nil {
		h.writeError(w, fmt.Sprintf("Failed to delete objects: %v", err), http.StatusInternalServerError)
		return
	}

	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"message":     "Objects deleted successfully",
			"bucket":      req.Bucket,
			"deletedKeys": req.Keys,
		},
	}

	h.writeResponse(w, response)
}

// HandleUpload handles POST /api/upload
func (h *APIHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.s3Service == nil {
		h.writeError(w, "S3 service not configured", http.StatusBadRequest)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(32 << 20) // 32 MB max memory
	if err != nil {
		h.writeError(w, "Failed to parse multipart form", http.StatusBadRequest)
		return
	}

	// Get bucket and key parameters
	bucket := r.FormValue("bucket")
	key := r.FormValue("key")

	if bucket == "" {
		h.writeError(w, "Bucket parameter is required", http.StatusBadRequest)
		return
	}
	if key == "" {
		h.writeError(w, "Key parameter is required", http.StatusBadRequest)
		return
	}

	// Get file from form
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		h.writeError(w, "Failed to get file from form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read file content
	fileContent, err := io.ReadAll(file)
	if err != nil {
		h.writeError(w, "Failed to read file content", http.StatusInternalServerError)
		return
	}

	// Determine content type
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		// Try to detect from file extension
		ext := filepath.Ext(fileHeader.Filename)
		contentType = mime.TypeByExtension(ext)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}

	// Create upload input
	uploadInput := service.UploadObjectInput{
		Bucket:      bucket,
		Key:         key,
		Body:        fileContent,
		ContentType: contentType,
		Metadata: map[string]string{
			"original-filename": fileHeader.Filename,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	output, err := h.s3Service.UploadObject(ctx, uploadInput)
	if err != nil {
		h.writeError(w, fmt.Sprintf("Failed to upload object: %v", err), http.StatusInternalServerError)
		return
	}

	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"message": "File uploaded successfully",
			"bucket":  bucket,
			"key":     output.Key,
			"etag":    output.ETag,
			"size":    len(fileContent),
		},
	}

	h.writeResponse(w, response)
}

// HandleDownload handles GET /api/download?bucket=<bucket>&key=<key>
func (h *APIHandler) HandleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.s3Service == nil {
		h.writeError(w, "S3 service not configured", http.StatusBadRequest)
		return
	}

	// Parse query parameters
	bucket := r.URL.Query().Get("bucket")
	key := r.URL.Query().Get("key")

	if bucket == "" {
		h.writeError(w, "Bucket parameter is required", http.StatusBadRequest)
		return
	}
	if key == "" {
		h.writeError(w, "Key parameter is required", http.StatusBadRequest)
		return
	}

	// Create download input
	downloadInput := service.DownloadObjectInput{
		Bucket: bucket,
		Key:    key,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	output, err := h.s3Service.DownloadObject(ctx, downloadInput)
	if err != nil {
		h.writeError(w, fmt.Sprintf("Failed to download object: %v", err), http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", output.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(output.ContentLength, 10))
	w.Header().Set("Last-Modified", output.LastModified)

	// Set filename from metadata or key
	filename := filepath.Base(key)
	if originalFilename, exists := output.Metadata["original-filename"]; exists {
		filename = originalFilename
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	// Write file content
	w.Write(output.Body)
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
