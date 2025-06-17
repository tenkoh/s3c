package handler

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/netip"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	s3cerrors "github.com/tenkoh/s3c/pkg/errors"
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
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	RequestID string      `json:"requestId,omitempty"`
}

// APIErrorResponse represents a structured error response
type APIErrorResponse struct {
	Success   bool     `json:"success"`
	Error     APIError `json:"error"`
	RequestID string   `json:"requestId,omitempty"`
}

// APIError represents detailed error information
type APIError struct {
	Code       string      `json:"code"`
	Message    string      `json:"message"`
	Details    interface{} `json:"details,omitempty"`
	Suggestion string      `json:"suggestion,omitempty"`
	Category   string      `json:"category,omitempty"`
	Severity   string      `json:"severity,omitempty"`
	Retryable  bool        `json:"retryable,omitempty"`
}

// APIHandler handles API requests with dependency injection
type APIHandler struct {
	profileProvider  ProfileProvider
	s3ServiceCreator S3ServiceCreator
	s3Service        service.S3Operations // Current S3 service instance
	currentConfig    *service.S3Config    // Current S3 configuration
	shutdownCh       chan<- struct{}      // Channel for graceful shutdown
	logger           *slog.Logger         // Logger for operation tracking
}

// NewAPIHandler creates a new API handler with dependencies
func NewAPIHandler(profileProvider ProfileProvider, s3ServiceCreator S3ServiceCreator, logger *slog.Logger) *APIHandler {
	return &APIHandler{
		profileProvider:  profileProvider,
		s3ServiceCreator: s3ServiceCreator,
		logger:           logger,
	}
}

// NewAPIHandlerWithShutdown creates a new API handler with shutdown channel
func NewAPIHandlerWithShutdown(profileProvider ProfileProvider, s3ServiceCreator S3ServiceCreator, shutdownCh chan<- struct{}, logger *slog.Logger) *APIHandler {
	return &APIHandler{
		profileProvider:  profileProvider,
		s3ServiceCreator: s3ServiceCreator,
		shutdownCh:       shutdownCh,
		logger:           logger,
	}
}

// HandleProfiles handles GET /api/profiles
func (h *APIHandler) HandleProfiles(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	opLogger := h.logger.With("operation", "list_profiles", "requestId", requestID)

	opLogger.Debug("Starting profile listing operation")

	profiles, err := h.profileProvider.GetProfiles()
	if err != nil {
		opLogger.Error("Failed to get AWS profiles", "error", err)
		// Convert to S3C error if needed
		var s3cErr error
		if _, ok := err.(*s3cerrors.S3CError); ok {
			s3cErr = err
		} else {
			s3cErr = s3cerrors.NewFileOperationError("read", "AWS profiles", err)
		}
		h.writeStructuredError(w, s3cErr, requestID)
		return
	}

	opLogger.Info("Successfully retrieved AWS profiles", "profileCount", len(profiles))

	response := APIResponse{
		Success:   true,
		Data:      map[string]interface{}{"profiles": profiles},
		RequestID: requestID,
	}

	h.writeResponse(w, response)
}

// HandleSettings handles POST /api/settings
func (h *APIHandler) HandleSettings(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	opLogger := h.logger.With("operation", "configure_s3", "requestId", requestID)

	var config service.S3Config
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		opLogger.Error("Failed to decode S3 configuration", "error", err)
		s3cErr := s3cerrors.NewInvalidInputError("request body", "invalid JSON")
		h.writeStructuredError(w, s3cErr, requestID)
		return
	}

	opLogger.Debug("Received S3 configuration request",
		"profile", config.Profile,
		"region", config.Region,
		"hasEndpoint", config.EndpointURL != "",
	)

	// Validate required fields
	if config.Profile == "" {
		opLogger.Warn("Missing required field: profile")
		s3cErr := s3cerrors.NewMissingFieldError("profile")
		h.writeStructuredError(w, s3cErr, requestID)
		return
	}
	if config.Region == "" {
		opLogger.Warn("Missing required field: region")
		s3cErr := s3cerrors.NewMissingFieldError("region")
		h.writeStructuredError(w, s3cErr, requestID)
		return
	}

	// Create S3 service with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opLogger.Debug("Creating S3 service", "timeout", "10s")
	s3Service, err := h.s3ServiceCreator(ctx, config)
	if err != nil {
		opLogger.Error("Failed to create S3 service", "error", err)
		// The service creator should already return structured errors
		h.writeStructuredError(w, err, requestID)
		return
	}

	// Test connection
	opLogger.Debug("Testing S3 connection")
	if err := s3Service.TestConnection(ctx); err != nil {
		opLogger.Error("S3 connection test failed", "error", err)
		// Service should return structured errors
		h.writeStructuredError(w, err, requestID)
		return
	}

	// Store the service and configuration
	h.s3Service = s3Service
	h.currentConfig = &config

	opLogger.Info("S3 connection configured successfully",
		"profile", config.Profile,
		"region", config.Region,
	)

	response := APIResponse{
		Success:   true,
		Data:      map[string]interface{}{"message": "S3 connection configured successfully"},
		RequestID: requestID,
	}

	h.writeResponse(w, response)
}

// HandleBuckets handles GET /api/buckets
func (h *APIHandler) HandleBuckets(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	opLogger := h.logger.With("operation", "list_buckets", "requestId", requestID)

	opLogger.Debug("Starting bucket listing operation")

	if h.s3Service == nil {
		opLogger.Warn("S3 service not configured")
		s3cErr := s3cerrors.NewConfigError(s3cerrors.CodeConfigMissing, "S3 service not configured")
		h.writeStructuredError(w, s3cErr, requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	buckets, err := h.s3Service.ListBuckets(ctx)
	if err != nil {
		opLogger.Error("Failed to list S3 buckets", "error", err)
		// Service should return structured errors
		h.writeStructuredError(w, err, requestID)
		return
	}

	opLogger.Info("Successfully listed S3 buckets", "bucketCount", len(buckets))

	response := APIResponse{
		Success:   true,
		Data:      map[string]interface{}{"buckets": buckets},
		RequestID: requestID,
	}

	h.writeResponse(w, response)
}

// HandleBucketCreate handles POST /api/buckets/create
func (h *APIHandler) HandleBucketCreate(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	opLogger := h.logger.With("operation", "create_bucket", "requestId", requestID)

	opLogger.Debug("Starting bucket creation operation")

	if h.s3Service == nil {
		opLogger.Warn("S3 service not configured")
		s3cErr := s3cerrors.NewConfigError(s3cerrors.CodeConfigMissing, "S3 service not configured")
		h.writeStructuredError(w, s3cErr, requestID)
		return
	}

	// Parse request body
	var req CreateBucketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		opLogger.Error("Failed to decode create bucket request", "error", err)
		s3cErr := s3cerrors.NewInvalidInputError("request body", "invalid JSON")
		h.writeStructuredError(w, s3cErr, requestID)
		return
	}

	// Validate required fields
	if req.Name == "" {
		opLogger.Warn("Missing required field: name")
		s3cErr := s3cerrors.NewMissingFieldError("name")
		h.writeStructuredError(w, s3cErr, requestID)
		return
	}

	// Validate bucket name format
	if err := validateBucketName(req.Name); err != nil {
		opLogger.Warn("Invalid bucket name", "bucketName", req.Name, "error", err)
		s3cErr := s3cerrors.NewValidationError(s3cerrors.CodeInvalidInput, err.Error())
		h.writeStructuredError(w, s3cErr, requestID)
		return
	}

	opLogger.Debug("Creating S3 bucket", "bucketName", req.Name)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := h.s3Service.CreateBucket(ctx, req.Name)
	if err != nil {
		opLogger.Error("Failed to create S3 bucket", "error", err, "bucketName", req.Name)
		h.writeStructuredError(w, err, requestID)
		return
	}

	opLogger.Info("Successfully created S3 bucket", "bucketName", req.Name)

	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"message": "Bucket created successfully",
			"bucket":  req.Name,
		},
		RequestID: requestID,
	}

	h.writeResponse(w, response)
}

// HandleShutdown handles POST /api/shutdown
func (h *APIHandler) HandleShutdown(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	opLogger := h.logger.With("operation", "shutdown", "requestId", requestID)

	opLogger.Info("Received shutdown request from API")

	response := APIResponse{
		Success:   true,
		Data:      map[string]interface{}{"message": "Server shutting down"},
		RequestID: requestID,
	}

	h.writeResponse(w, response)

	// Trigger graceful shutdown via channel
	if h.shutdownCh != nil {
		go func() {
			time.Sleep(100 * time.Millisecond) // Give time for response to be sent
			select {
			case h.shutdownCh <- struct{}{}:
				opLogger.Info("Shutdown signal sent successfully")
			default:
				opLogger.Warn("Failed to send shutdown signal - channel full or closed")
			}
		}()
	} else {
		opLogger.Warn("No shutdown channel configured")
	}
}

// HandleFolderCreate handles POST /api/objects/folder/create
func (h *APIHandler) HandleFolderCreate(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	opLogger := h.logger.With("operation", "create_folder", "requestId", requestID)

	opLogger.Debug("Starting folder creation operation")

	if h.s3Service == nil {
		opLogger.Warn("S3 service not configured")
		s3cErr := s3cerrors.NewConfigError(s3cerrors.CodeConfigMissing, "S3 service not configured")
		h.writeStructuredError(w, s3cErr, requestID)
		return
	}

	// Parse request body
	var req CreateFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		opLogger.Error("Failed to decode create folder request", "error", err)
		s3cErr := s3cerrors.NewInvalidInputError("request body", "invalid JSON")
		h.writeStructuredError(w, s3cErr, requestID)
		return
	}

	// Validate required fields
	if req.Bucket == "" {
		opLogger.Warn("Missing required field: bucket")
		s3cErr := s3cerrors.NewMissingFieldError("bucket")
		h.writeStructuredError(w, s3cErr, requestID)
		return
	}
	if req.Prefix == "" {
		opLogger.Warn("Missing required field: prefix")
		s3cErr := s3cerrors.NewMissingFieldError("prefix")
		h.writeStructuredError(w, s3cErr, requestID)
		return
	}

	// Validate folder name format
	if err := validateFolderName(req.Prefix); err != nil {
		opLogger.Warn("Invalid folder name", "folderName", req.Prefix, "error", err)
		s3cErr := s3cerrors.NewValidationError(s3cerrors.CodeInvalidInput, err.Error())
		h.writeStructuredError(w, s3cErr, requestID)
		return
	}

	opLogger.Debug("Creating S3 folder", "bucket", req.Bucket, "prefix", req.Prefix)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := h.s3Service.CreateFolder(ctx, req.Bucket, req.Prefix)
	if err != nil {
		opLogger.Error("Failed to create S3 folder", "error", err, "bucket", req.Bucket, "prefix", req.Prefix)
		h.writeStructuredError(w, err, requestID)
		return
	}

	opLogger.Info("Successfully created S3 folder", "bucket", req.Bucket, "prefix", req.Prefix)

	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"message": "Folder created successfully",
			"bucket":  req.Bucket,
			"prefix":  req.Prefix,
		},
		RequestID: requestID,
	}

	h.writeResponse(w, response)
}

// HandleObjectsList handles POST /api/objects/list
func (h *APIHandler) HandleObjectsList(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	opLogger := h.logger.With("operation", "list_objects", "requestId", requestID)

	if h.s3Service == nil {
		opLogger.Warn("S3 service not configured")
		s3cErr := s3cerrors.NewConfigError(s3cerrors.CodeConfigMissing, "S3 service not configured")
		h.writeStructuredError(w, s3cErr, requestID)
		return
	}

	// Parse request body
	var req ListObjectsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		opLogger.Error("Failed to decode list objects request", "error", err)
		s3cErr := s3cerrors.NewInvalidInputError("request body", "invalid JSON")
		h.writeStructuredError(w, s3cErr, requestID)
		return
	}

	// Validate required fields
	if req.Bucket == "" {
		opLogger.Warn("Missing required field: bucket")
		s3cErr := s3cerrors.NewMissingFieldError("bucket")
		h.writeStructuredError(w, s3cErr, requestID)
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

	opLogger.Debug("Starting S3 object listing",
		"bucket", req.Bucket,
		"prefix", req.Prefix,
		"delimiter", req.Delimiter,
		"maxKeys", maxKeys,
		"hasContinuationToken", req.ContinuationToken != "",
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	output, err := h.s3Service.ListObjects(ctx, input)
	if err != nil {
		opLogger.Error("Failed to list S3 objects", "error", err, "bucket", req.Bucket)
		// Service should return structured errors
		h.writeStructuredError(w, err, requestID)
		return
	}

	opLogger.Info("Successfully listed S3 objects",
		"bucket", req.Bucket,
		"objectCount", len(output.Objects),
		"commonPrefixCount", len(output.CommonPrefixes),
		"isTruncated", output.IsTruncated,
	)

	response := APIResponse{
		Success:   true,
		Data:      output,
		RequestID: requestID,
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

// CreateBucketRequest represents the request for creating a bucket
type CreateBucketRequest struct {
	Name string `json:"name"`
}

// CreateFolderRequest represents the request for creating a folder
type CreateFolderRequest struct {
	Bucket string `json:"bucket"`
	Prefix string `json:"prefix"`
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
	requestID := generateRequestID()

	if h.s3Service == nil {
		s3cErr := s3cerrors.NewConfigError(s3cerrors.CodeConfigMissing, "S3 service not configured")
		h.writeStructuredError(w, s3cErr, requestID)
		return
	}

	var req DeleteObjectsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s3cErr := s3cerrors.NewInvalidInputError("request body", "invalid JSON")
		h.writeStructuredError(w, s3cErr, requestID)
		return
	}

	// Validate request
	if req.Bucket == "" {
		s3cErr := s3cerrors.NewMissingFieldError("bucket")
		h.writeStructuredError(w, s3cErr, requestID)
		return
	}
	if len(req.Keys) == 0 {
		s3cErr := s3cerrors.NewValidationError(s3cerrors.CodeInvalidInput, "At least one key is required")
		h.writeStructuredError(w, s3cErr, requestID)
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
		// Service should return structured errors
		h.writeStructuredError(w, err, requestID)
		return
	}

	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"message":     "Objects deleted successfully",
			"bucket":      req.Bucket,
			"deletedKeys": req.Keys,
		},
		RequestID: requestID,
	}

	h.writeResponse(w, response)
}

// HandleObjectsUpload handles POST /api/objects/upload
func (h *APIHandler) HandleObjectsUpload(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()

	if h.s3Service == nil {
		s3cErr := s3cerrors.NewConfigError(s3cerrors.CodeConfigMissing, "S3 service not configured")
		h.writeStructuredError(w, s3cErr, requestID)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(32 << 20) // 32 MB max memory
	if err != nil {
		s3cErr := s3cerrors.NewInvalidInputError("multipart form", "failed to parse")
		h.writeStructuredError(w, s3cErr, requestID)
		return
	}

	// Get bucket parameter
	bucket := r.FormValue("bucket")
	if bucket == "" {
		s3cErr := s3cerrors.NewMissingFieldError("bucket")
		h.writeStructuredError(w, s3cErr, requestID)
		return
	}

	// Get uploads configuration from form
	uploadsJSON := r.FormValue("uploads")
	if uploadsJSON == "" {
		s3cErr := s3cerrors.NewMissingFieldError("uploads")
		h.writeStructuredError(w, s3cErr, requestID)
		return
	}

	// Parse uploads configuration
	var uploads []UploadFileInfo
	if err := json.Unmarshal([]byte(uploadsJSON), &uploads); err != nil {
		s3cErr := s3cerrors.NewInvalidInputError("uploads", "invalid JSON format")
		h.writeStructuredError(w, s3cErr, requestID)
		return
	}

	if len(uploads) == 0 {
		s3cErr := s3cerrors.NewValidationError(s3cerrors.CodeInvalidInput, "At least one upload is required")
		h.writeStructuredError(w, s3cErr, requestID)
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
		Success:   success,
		Data:      responseData,
		RequestID: requestID,
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
	requestID := generateRequestID()

	if h.s3Service == nil {
		s3cErr := s3cerrors.NewConfigError(s3cerrors.CodeConfigMissing, "S3 service not configured")
		h.writeStructuredError(w, s3cErr, requestID)
		return
	}

	// Parse request body
	var req DownloadObjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s3cErr := s3cerrors.NewInvalidInputError("request body", "invalid JSON")
		h.writeStructuredError(w, s3cErr, requestID)
		return
	}

	// Validate request
	if req.Bucket == "" {
		s3cErr := s3cerrors.NewMissingFieldError("bucket")
		h.writeStructuredError(w, s3cErr, requestID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	switch req.Type {
	case "files":
		if len(req.Keys) == 0 {
			s3cErr := s3cerrors.NewValidationError(s3cerrors.CodeInvalidInput, "At least one key is required for files download")
			h.writeStructuredError(w, s3cErr, requestID)
			return
		}
		if len(req.Keys) == 1 {
			h.downloadSingleFile(w, ctx, req.Bucket, req.Keys[0], requestID)
		} else {
			h.downloadMultipleFiles(w, ctx, req.Bucket, req.Keys, requestID)
		}
	case "folder":
		if req.Prefix == "" {
			s3cErr := s3cerrors.NewMissingFieldError("prefix")
			h.writeStructuredError(w, s3cErr, requestID)
			return
		}
		h.downloadFolder(w, ctx, req.Bucket, req.Prefix, requestID)
	default:
		s3cErr := s3cerrors.NewInvalidInputError("type", "must be 'files' or 'folder'")
		h.writeStructuredError(w, s3cErr, requestID)
	}
}

// downloadSingleFile downloads a single file directly
func (h *APIHandler) downloadSingleFile(w http.ResponseWriter, ctx context.Context, bucket, key, requestID string) {
	downloadInput := service.DownloadObjectInput{
		Bucket: bucket,
		Key:    key,
	}

	output, err := h.s3Service.DownloadObject(ctx, downloadInput)
	if err != nil {
		// Service should return structured errors
		h.writeStructuredError(w, err, requestID)
		return
	}

	// Extract filename from S3 key (ignore potentially corrupted metadata)
	filename := filepath.Base(key)

	// Generate proper Content-Disposition header with UTF-8 support
	contentDisposition := setContentDisposition(filename)

	// Set response headers
	w.Header().Set("Content-Disposition", contentDisposition)
	w.Header().Set("Content-Type", output.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(output.ContentLength, 10))
	w.Header().Set("Last-Modified", output.LastModified)

	// Write file content
	w.Write(output.Body)
}

// downloadMultipleFiles downloads multiple files as a ZIP
func (h *APIHandler) downloadMultipleFiles(w http.ResponseWriter, ctx context.Context, bucket string, keys []string, requestID string) {
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
func (h *APIHandler) downloadFolder(w http.ResponseWriter, ctx context.Context, bucket, prefix, requestID string) {
	// List all objects in the folder
	listInput := service.ListObjectsInput{
		Bucket:    bucket,
		Prefix:    prefix,
		MaxKeys:   1000, // Get up to 1000 objects
		Delimiter: "",   // No delimiter to get all nested objects
	}

	listOutput, err := h.s3Service.ListObjects(ctx, listInput)
	if err != nil {
		// Service should return structured errors
		h.writeStructuredError(w, err, requestID)
		return
	}

	if len(listOutput.Objects) == 0 {
		s3cErr := s3cerrors.NewS3ObjectNotFoundError(bucket, prefix)
		h.writeStructuredError(w, s3cErr, requestID)
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
		s3cErr := s3cerrors.NewS3ObjectNotFoundError(bucket, prefix).WithSuggestion("Folder contains no files to download")
		h.writeStructuredError(w, s3cErr, requestID)
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

		// Create file in ZIP with folder structure preserved
		// For prefix "sandbox/" and key "sandbox/subdir/file.txt"
		// we want zipPath to be "sandbox/subdir/file.txt" (keep full path)
		zipPath := key

		fileWriter, err := zipWriter.Create(zipPath)
		if err != nil {
			continue
		}

		// Write file content to ZIP
		fileWriter.Write(output.Body)
	}
}

// HandleHealth handles POST /api/health
func (h *APIHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()

	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		},
		RequestID: requestID,
	}
	h.writeResponse(w, response)
}

// HandleStatus handles POST /api/status
func (h *APIHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()

	// Check if S3 service is configured
	if h.s3Service == nil {
		response := APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"connected": false,
				"message":   "Not connected",
			},
			RequestID: requestID,
		}
		h.writeResponse(w, response)
		return
	}

	// Test connection to get more detailed status
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := h.s3Service.TestConnection(ctx)
	if err != nil {
		response := APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"connected": false,
				"message":   "Connection failed",
				"error":     err.Error(),
			},
			RequestID: requestID,
		}
		h.writeResponse(w, response)
		return
	}

	// Return connection status with configuration details
	responseData := map[string]interface{}{
		"connected": true,
		"message":   "Connected to S3",
	}

	// Add configuration details if available
	if h.currentConfig != nil {
		responseData["profile"] = h.currentConfig.Profile
		responseData["region"] = h.currentConfig.Region
		if h.currentConfig.EndpointURL != "" {
			responseData["endpoint"] = h.currentConfig.EndpointURL
		}
	}

	response := APIResponse{
		Success:   true,
		Data:      responseData,
		RequestID: requestID,
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

// writeStructuredError writes a structured error response based on s3c errors
func (h *APIHandler) writeStructuredError(w http.ResponseWriter, err error, requestID string) {
	var s3cErr *s3cerrors.S3CError
	statusCode := http.StatusInternalServerError
	apiError := APIError{
		Code:    string(s3cerrors.CodeInternalError),
		Message: "Internal server error",
	}

	if errors.As(err, &s3cErr) {
		// Map S3C error to API error
		apiError = APIError{
			Code:       string(s3cErr.Code),
			Message:    s3cErr.Message,
			Details:    s3cErr.Details,
			Suggestion: s3cErr.Suggestion,
			Category:   string(s3cErr.Category),
			Severity:   string(s3cErr.Severity),
			Retryable:  s3cerrors.IsRetryable(err),
		}

		// Map error code to HTTP status code
		statusCode = mapErrorCodeToHTTPStatus(s3cErr.Code)
	} else {
		// Fallback for non-S3C errors
		apiError.Message = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := APIErrorResponse{
		Success:   false,
		Error:     apiError,
		RequestID: requestID,
	}

	json.NewEncoder(w).Encode(response)
}

// mapErrorCodeToHTTPStatus maps S3C error codes to appropriate HTTP status codes
func mapErrorCodeToHTTPStatus(code s3cerrors.ErrorCode) int {
	switch code {
	// Validation errors -> 400 Bad Request
	case s3cerrors.CodeInvalidInput, s3cerrors.CodeMissingField,
		s3cerrors.CodeInvalidFormat, s3cerrors.CodeOutOfRange:
		return http.StatusBadRequest

	// Authentication/Authorization errors -> 401/403
	case s3cerrors.CodeCredentialsInvalid:
		return http.StatusUnauthorized
	case s3cerrors.CodeS3AccessDenied:
		return http.StatusForbidden

	// Not found errors -> 404
	case s3cerrors.CodeS3BucketNotFound, s3cerrors.CodeS3ObjectNotFound:
		return http.StatusNotFound

	// Rate limiting -> 429
	case s3cerrors.CodeS3QuotaExceeded:
		return http.StatusTooManyRequests

	// Network/timeout errors -> 503 Service Unavailable
	case s3cerrors.CodeNetworkTimeout, s3cerrors.CodeNetworkUnavailable, s3cerrors.CodeS3Connection:
		return http.StatusServiceUnavailable

	// Configuration errors -> 400 Bad Request
	case s3cerrors.CodeConfigMissing, s3cerrors.CodeConfigInvalid, s3cerrors.CodeProfileNotFound:
		return http.StatusBadRequest

	// Not implemented -> 501
	case s3cerrors.CodeNotImplemented:
		return http.StatusNotImplemented

	// Default: Internal server error
	default:
		return http.StatusInternalServerError
	}
}

// generateRequestID generates a simple request ID for tracking
func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

// setContentDisposition creates a Content-Disposition header value that properly handles
// non-ASCII filenames using RFC 5987 encoding
func setContentDisposition(filename string) string {
	// Check if filename contains non-ASCII characters
	hasNonASCII := false
	for _, r := range filename {
		if r > unicode.MaxASCII {
			hasNonASCII = true
			break
		}
	}

	if !hasNonASCII {
		// ASCII filename - use simple format
		return fmt.Sprintf("attachment; filename=\"%s\"", filename)
	}

	// Non-ASCII filename - use RFC 5987 encoding
	// URL encode the filename for UTF-8, then replace + with %20 for spaces
	encodedFilename := strings.ReplaceAll(url.QueryEscape(filename), "+", "%20")

	// Create both formats for better browser compatibility:
	// 1. Simple format with ASCII fallback (replace non-ASCII with underscore)
	asciiFallback := strings.Map(func(r rune) rune {
		if r > unicode.MaxASCII {
			return '_'
		}
		return r
	}, filename)

	// 2. RFC 5987 format with UTF-8 encoding
	return fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s",
		asciiFallback, encodedFilename)
}

// validateBucketName validates S3 bucket naming rules
func validateBucketName(name string) error {
	// AWS S3 bucket naming rules:
	// - Must be between 3 and 63 characters long
	// - Can consist only of lowercase letters, numbers, dots (.), and hyphens (-)
	// - Must begin and end with a letter or number
	// - Must not contain two adjacent periods
	// - Must not be formatted as an IP address (e.g., 192.168.5.4)
	// - Must not start with 'xn--' (reserved)
	// - Must not end with '-s3alias' (reserved)

	if len(name) < 3 || len(name) > 63 {
		return errors.New("bucket name must be between 3 and 63 characters long")
	}

	// Check for valid characters
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '.' || char == '-') {
			return errors.New("bucket name can only contain lowercase letters, numbers, dots, and hyphens")
		}
	}

	// Must begin and end with a letter or number
	if !((name[0] >= 'a' && name[0] <= 'z') || (name[0] >= '0' && name[0] <= '9')) {
		return errors.New("bucket name must begin with a letter or number")
	}
	lastChar := name[len(name)-1]
	if !((lastChar >= 'a' && lastChar <= 'z') || (lastChar >= '0' && lastChar <= '9')) {
		return errors.New("bucket name must end with a letter or number")
	}

	// Must not contain two adjacent periods
	if strings.Contains(name, "..") {
		return errors.New("bucket name must not contain two adjacent periods")
	}

	// Must not start with 'xn--'
	if strings.HasPrefix(name, "xn--") {
		return errors.New("bucket name must not start with 'xn--'")
	}

	// Must not end with '-s3alias'
	if strings.HasSuffix(name, "-s3alias") {
		return errors.New("bucket name must not end with '-s3alias'")
	}

	// Must not be formatted as an IP address (IPv4 or IPv6)
	if _, err := netip.ParseAddr(name); err == nil {
		return errors.New("bucket name must not be formatted as an IP address")
	}

	return nil
}

// validateFolderName validates S3 folder naming rules
func validateFolderName(name string) error {
	// S3 folder naming rules (similar to object key rules):
	// - Can contain any UTF-8 characters
	// - Must not be empty
	// - Should not start or end with forward slash (we'll handle trailing slash internally)
	// - Should not contain double slashes

	if name == "" {
		return errors.New("folder name cannot be empty")
	}

	// Remove leading and trailing slashes for validation
	trimmedName := strings.Trim(name, "/")
	if trimmedName == "" {
		return errors.New("folder name cannot be only slashes")
	}

	// Check for double slashes
	if strings.Contains(name, "//") {
		return errors.New("folder name cannot contain double slashes")
	}

	// Check for invalid characters that might cause issues
	invalidChars := []string{"\x00", "\r", "\n"}
	for _, char := range invalidChars {
		if strings.Contains(name, char) {
			return errors.New("folder name contains invalid characters")
		}
	}

	return nil
}
