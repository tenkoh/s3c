package handler

import (
	"archive/zip"
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

// APIHandler handles API requests with dependency injection
type APIHandler struct {
	profileProvider  ProfileProvider
	s3ServiceCreator S3ServiceCreator
	s3Service        service.S3Operations // Current S3 service instance
}

// NewAPIHandler creates a new API handler with dependencies
func NewAPIHandler(profileProvider ProfileProvider, s3ServiceCreator S3ServiceCreator) *APIHandler {
	return &APIHandler{
		profileProvider:  profileProvider,
		s3ServiceCreator: s3ServiceCreator,
	}
}

// HandleProfiles handles GET /api/profiles
func (h *APIHandler) HandleProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.profileProvider.GetProfiles()
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

	s3Service, err := h.s3ServiceCreator(ctx, config)
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

// HandleObjectsList handles POST /api/objects/list
func (h *APIHandler) HandleObjectsList(w http.ResponseWriter, r *http.Request) {
	if h.s3Service == nil {
		h.writeError(w, "S3 service not configured", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req ListObjectsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Bucket == "" {
		h.writeError(w, "Bucket is required", http.StatusBadRequest)
		return
	}

	// Set default maxKeys if not provided
	maxKeys := req.MaxKeys
	if maxKeys == 0 {
		maxKeys = 100
	}
	if maxKeys > 1000 {
		maxKeys = 1000
	}

	// Create input
	input := service.ListObjectsInput{
		Bucket:            req.Bucket,
		Prefix:            req.Prefix,
		Delimiter:         req.Delimiter,
		MaxKeys:           maxKeys,
		ContinuationToken: req.ContinuationToken,
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

// Request structures for new POST-unified API

// ListObjectsRequest represents the request for listing objects
type ListObjectsRequest struct {
	Bucket            string `json:"bucket"`
	Prefix            string `json:"prefix,omitempty"`
	Delimiter         string `json:"delimiter,omitempty"`
	MaxKeys           int32  `json:"maxKeys,omitempty"`
	ContinuationToken string `json:"continuationToken,omitempty"`
}

// DeleteObjectsRequest represents the request payload for deleting objects
type DeleteObjectsRequest struct {
	Bucket string   `json:"bucket"`
	Keys   []string `json:"keys"`
}

// UploadObjectsRequest represents the request for uploading multiple objects
type UploadObjectsRequest struct {
	Bucket  string           `json:"bucket"`
	Uploads []UploadFileInfo `json:"uploads"`
}

// UploadFileInfo represents information for a single file upload
type UploadFileInfo struct {
	Key  string `json:"key"`  // S3 object key
	File string `json:"file"` // multipart form field name
}

// DownloadObjectRequest represents the request for downloading objects
type DownloadObjectRequest struct {
	Bucket string   `json:"bucket"`
	Type   string   `json:"type"`             // "files" or "folder"
	Keys   []string `json:"keys,omitempty"`   // for files (single or multiple)
	Prefix string   `json:"prefix,omitempty"` // for folder
}

// HandleObjectsDelete handles POST /api/objects/delete
func (h *APIHandler) HandleObjectsDelete(w http.ResponseWriter, r *http.Request) {
	if h.s3Service == nil {
		h.writeError(w, "S3 service not configured", http.StatusBadRequest)
		return
	}

	var req DeleteObjectsRequest
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

// HandleObjectsUpload handles POST /api/objects/upload
func (h *APIHandler) HandleObjectsUpload(w http.ResponseWriter, r *http.Request) {
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

	// Get bucket parameter
	bucket := r.FormValue("bucket")
	if bucket == "" {
		h.writeError(w, "Bucket parameter is required", http.StatusBadRequest)
		return
	}

	// Get uploads configuration from form
	uploadsJSON := r.FormValue("uploads")
	if uploadsJSON == "" {
		h.writeError(w, "uploads parameter is required", http.StatusBadRequest)
		return
	}

	// Parse uploads configuration
	var uploads []UploadFileInfo
	if err := json.Unmarshal([]byte(uploadsJSON), &uploads); err != nil {
		h.writeError(w, "Invalid uploads JSON format", http.StatusBadRequest)
		return
	}

	if len(uploads) == 0 {
		h.writeError(w, "At least one upload is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var results []map[string]interface{}
	var errors []string

	// Process each file upload
	for _, upload := range uploads {
		file, fileHeader, err := r.FormFile(upload.File)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to get file %s: %v", upload.File, err))
			continue
		}

		// Read file content
		fileContent, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to read file %s: %v", upload.File, err))
			continue
		}

		// Determine content type
		contentType := fileHeader.Header.Get("Content-Type")
		if contentType == "" {
			ext := filepath.Ext(fileHeader.Filename)
			contentType = mime.TypeByExtension(ext)
			if contentType == "" {
				contentType = "application/octet-stream"
			}
		}

		// Create upload input
		uploadInput := service.UploadObjectInput{
			Bucket:      bucket,
			Key:         upload.Key,
			Body:        fileContent,
			ContentType: contentType,
			Metadata: map[string]string{
				"original-filename": fileHeader.Filename,
			},
		}

		// Upload to S3
		output, err := h.s3Service.UploadObject(ctx, uploadInput)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to upload %s: %v", upload.Key, err))
			continue
		}

		// Add successful result
		results = append(results, map[string]interface{}{
			"key":      output.Key,
			"etag":     output.ETag,
			"size":     len(fileContent),
			"filename": fileHeader.Filename,
		})
	}

	// Prepare response
	responseData := map[string]interface{}{
		"bucket":   bucket,
		"uploaded": results,
		"success":  len(results),
		"total":    len(uploads),
	}

	if len(errors) > 0 {
		responseData["errors"] = errors
		responseData["failed"] = len(errors)
	}

	// Determine overall success
	success := len(results) > 0
	message := fmt.Sprintf("Uploaded %d of %d files successfully", len(results), len(uploads))

	response := APIResponse{
		Success: success,
		Data:    responseData,
	}

	// Set appropriate status code
	statusCode := http.StatusOK
	if len(results) == 0 {
		statusCode = http.StatusInternalServerError
		response.Error = "All uploads failed"
	} else if len(errors) > 0 {
		statusCode = http.StatusPartialContent
		response.Error = message
	}

	w.WriteHeader(statusCode)
	h.writeResponse(w, response)
}

// HandleObjectsDownload handles POST /api/objects/download
func (h *APIHandler) HandleObjectsDownload(w http.ResponseWriter, r *http.Request) {
	if h.s3Service == nil {
		h.writeError(w, "S3 service not configured", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req DownloadObjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Bucket == "" {
		h.writeError(w, "Bucket is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	switch req.Type {
	case "files":
		if len(req.Keys) == 0 {
			h.writeError(w, "At least one key is required for files download", http.StatusBadRequest)
			return
		}
		if len(req.Keys) == 1 {
			h.downloadSingleFile(w, ctx, req.Bucket, req.Keys[0])
		} else {
			h.downloadMultipleFiles(w, ctx, req.Bucket, req.Keys)
		}
	case "folder":
		if req.Prefix == "" {
			h.writeError(w, "Prefix is required for folder download", http.StatusBadRequest)
			return
		}
		h.downloadFolder(w, ctx, req.Bucket, req.Prefix)
	default:
		h.writeError(w, "Type must be 'files' or 'folder'", http.StatusBadRequest)
	}
}

// downloadSingleFile downloads a single file directly
func (h *APIHandler) downloadSingleFile(w http.ResponseWriter, ctx context.Context, bucket, key string) {
	downloadInput := service.DownloadObjectInput{
		Bucket: bucket,
		Key:    key,
	}

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

// downloadMultipleFiles downloads multiple files as a ZIP
func (h *APIHandler) downloadMultipleFiles(w http.ResponseWriter, ctx context.Context, bucket string, keys []string) {
	// Set response headers for ZIP
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\"files.zip\"")

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	for _, key := range keys {
		downloadInput := service.DownloadObjectInput{
			Bucket: bucket,
			Key:    key,
		}

		output, err := h.s3Service.DownloadObject(ctx, downloadInput)
		if err != nil {
			// Skip failed downloads and continue with others
			continue
		}

		// Create file in ZIP
		fileWriter, err := zipWriter.Create(key)
		if err != nil {
			continue
		}

		// Write file content to ZIP
		fileWriter.Write(output.Body)
	}
}

// downloadFolder downloads all objects in a folder as a ZIP
func (h *APIHandler) downloadFolder(w http.ResponseWriter, ctx context.Context, bucket, prefix string) {
	// List all objects in the folder
	listInput := service.ListObjectsInput{
		Bucket:    bucket,
		Prefix:    prefix,
		MaxKeys:   1000, // Get up to 1000 objects
		Delimiter: "",   // No delimiter to get all nested objects
	}

	listOutput, err := h.s3Service.ListObjects(ctx, listInput)
	if err != nil {
		h.writeError(w, fmt.Sprintf("Failed to list folder objects: %v", err), http.StatusInternalServerError)
		return
	}

	if len(listOutput.Objects) == 0 {
		h.writeError(w, "No objects found in folder", http.StatusNotFound)
		return
	}

	// Extract keys from objects (exclude folders)
	var keys []string
	for _, obj := range listOutput.Objects {
		if !obj.IsFolder {
			keys = append(keys, obj.Key)
		}
	}

	if len(keys) == 0 {
		h.writeError(w, "No files found in folder", http.StatusNotFound)
		return
	}

	// Set response headers for ZIP
	folderName := filepath.Base(prefix)
	if folderName == "" || folderName == "." {
		folderName = "folder"
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.zip\"", folderName))

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	for _, key := range keys {
		downloadInput := service.DownloadObjectInput{
			Bucket: bucket,
			Key:    key,
		}

		output, err := h.s3Service.DownloadObject(ctx, downloadInput)
		if err != nil {
			// Skip failed downloads and continue with others
			continue
		}

		// Create file in ZIP with relative path
		relativePath := key
		if len(prefix) > 0 && len(key) > len(prefix) {
			relativePath = key[len(prefix):]
		}

		fileWriter, err := zipWriter.Create(relativePath)
		if err != nil {
			continue
		}

		// Write file content to ZIP
		fileWriter.Write(output.Body)
	}
}

// HandleHealth handles POST /api/health
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
