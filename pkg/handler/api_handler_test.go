package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tenkoh/s3c/pkg/service"
)

// Test doubles
type mockProfileProvider struct {
	profiles []string
	err      error
}

func (m *mockProfileProvider) GetProfiles() ([]string, error) {
	return m.profiles, m.err
}

type mockS3Service struct {
	testConnectionErr error
	listBucketsResult []string
	listBucketsErr    error
	createBucketErr   error
	listObjectsResult *service.ListObjectsOutput
	listObjectsErr    error
	deleteObjectErr   error
	deleteObjectsErr  error
	uploadResult      *service.UploadObjectOutput
	uploadErr         error
	downloadResult    *service.DownloadObjectOutput
	downloadErr       error
}

func (m *mockS3Service) TestConnection(ctx context.Context) error {
	return m.testConnectionErr
}

func (m *mockS3Service) ListBuckets(ctx context.Context) ([]string, error) {
	return m.listBucketsResult, m.listBucketsErr
}

func (m *mockS3Service) CreateBucket(ctx context.Context, bucketName string) error {
	return m.createBucketErr
}

func (m *mockS3Service) ListObjects(ctx context.Context, input service.ListObjectsInput) (*service.ListObjectsOutput, error) {
	return m.listObjectsResult, m.listObjectsErr
}

func (m *mockS3Service) DeleteObject(ctx context.Context, bucket, key string) error {
	return m.deleteObjectErr
}

func (m *mockS3Service) DeleteObjects(ctx context.Context, bucket string, keys []string) error {
	return m.deleteObjectsErr
}

func (m *mockS3Service) UploadObject(ctx context.Context, input service.UploadObjectInput) (*service.UploadObjectOutput, error) {
	return m.uploadResult, m.uploadErr
}

func (m *mockS3Service) DownloadObject(ctx context.Context, input service.DownloadObjectInput) (*service.DownloadObjectOutput, error) {
	return m.downloadResult, m.downloadErr
}

func mockS3ServiceCreator(mockService *mockS3Service) S3ServiceCreator {
	return func(ctx context.Context, cfg service.S3Config) (service.S3Operations, error) {
		if mockService.testConnectionErr != nil && mockService.testConnectionErr.Error() == "creation_error" {
			return nil, mockService.testConnectionErr
		}
		return mockService, nil
	}
}

// Integration tests using real ServeMux to test POST-unified API
func TestAPIHandler_Integration(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		url            string
		body           interface{}
		expectedStatus int
		setupHandler   func(*APIHandler)
	}{
		{
			name:           "POST /api/profiles success",
			method:         "POST",
			url:            "/api/profiles",
			body:           map[string]interface{}{},
			expectedStatus: http.StatusOK,
			setupHandler: func(h *APIHandler) {
				h.profileProvider = &mockProfileProvider{
					profiles: []string{"default", "work"},
				}
			},
		},
		{
			name:   "POST /api/objects/list success",
			method: "POST",
			url:    "/api/objects/list",
			body: ListObjectsRequest{
				Bucket: "test-bucket",
				Prefix: "folder/",
			},
			expectedStatus: http.StatusOK,
			setupHandler: func(h *APIHandler) {
				h.s3Service = &mockS3Service{
					listObjectsResult: &service.ListObjectsOutput{
						Objects: []service.S3Object{
							{Key: "folder/file1.txt", Size: 1024},
						},
					},
				}
			},
		},
		{
			name:   "POST /api/objects/delete success",
			method: "POST",
			url:    "/api/objects/delete",
			body: DeleteObjectsRequest{
				Bucket: "test-bucket",
				Keys:   []string{"file1.txt", "file2.txt"},
			},
			expectedStatus: http.StatusOK,
			setupHandler: func(h *APIHandler) {
				h.s3Service = &mockS3Service{}
			},
		},
		{
			name:   "POST /api/objects/download single file",
			method: "POST",
			url:    "/api/objects/download",
			body: DownloadObjectRequest{
				Bucket: "test-bucket",
				Type:   "files",
				Keys:   []string{"file.txt"},
			},
			expectedStatus: http.StatusOK,
			setupHandler: func(h *APIHandler) {
				h.s3Service = &mockS3Service{
					downloadResult: &service.DownloadObjectOutput{
						Body:          []byte("content"),
						ContentType:   "text/plain",
						ContentLength: 7,
					},
				}
			},
		},
		{
			name:   "POST /api/buckets/create success",
			method: "POST",
			url:    "/api/buckets/create",
			body: CreateBucketRequest{
				Name: "test-bucket-123",
			},
			expectedStatus: http.StatusOK,
			setupHandler: func(h *APIHandler) {
				h.s3Service = &mockS3Service{}
			},
		},
		{
			name:   "POST /api/buckets/create invalid name",
			method: "POST",
			url:    "/api/buckets/create",
			body: CreateBucketRequest{
				Name: "InvalidBucketName", // Invalid: contains uppercase
			},
			expectedStatus: http.StatusBadRequest,
			setupHandler: func(h *APIHandler) {
				h.s3Service = &mockS3Service{}
			},
		},
		{
			name:   "POST /api/buckets/create missing name",
			method: "POST",
			url:    "/api/buckets/create",
			body: CreateBucketRequest{
				Name: "",
			},
			expectedStatus: http.StatusBadRequest,
			setupHandler: func(h *APIHandler) {
				h.s3Service = &mockS3Service{}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange: Setup real ServeMux with POST routing
			mux := http.NewServeMux()
			handler := NewAPIHandler(nil, nil, slog.Default())
			tt.setupHandler(handler)

			// Setup POST-unified routes
			mux.HandleFunc("POST /api/health", handler.HandleHealth)
			mux.HandleFunc("POST /api/profiles", handler.HandleProfiles)
			mux.HandleFunc("POST /api/settings", handler.HandleSettings)
			mux.HandleFunc("POST /api/buckets", handler.HandleBuckets)
			mux.HandleFunc("POST /api/buckets/create", handler.HandleBucketCreate)
			mux.HandleFunc("POST /api/objects/list", handler.HandleObjectsList)
			mux.HandleFunc("POST /api/objects/delete", handler.HandleObjectsDelete)
			mux.HandleFunc("POST /api/objects/upload", handler.HandleObjectsUpload)
			mux.HandleFunc("POST /api/objects/download", handler.HandleObjectsDownload)

			var body *bytes.Buffer
			if tt.body != nil {
				bodyBytes, _ := json.Marshal(tt.body)
				body = bytes.NewBuffer(bodyBytes)
			} else {
				body = &bytes.Buffer{}
			}

			req := httptest.NewRequest(tt.method, tt.url, body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Act: Execute request through ServeMux
			mux.ServeHTTP(w, req)

			// Assert: Verify response
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
				t.Logf("Response body: %s", w.Body.String())
			}
		})
	}
}

// Unit tests for individual handler responsibilities
func TestAPIHandler_HandleProfiles(t *testing.T) {
	tests := []struct {
		name           string
		profilesResult []string
		profilesError  error
		expectedStatus int
	}{
		{
			name:           "successful profiles retrieval",
			profilesResult: []string{"default", "work"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "profiles provider error",
			profilesError:  errors.New("file not found"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockProvider := &mockProfileProvider{
				profiles: tt.profilesResult,
				err:      tt.profilesError,
			}
			handler := NewAPIHandler(mockProvider, nil, slog.Default())
			req := httptest.NewRequest("POST", "/api/profiles", bytes.NewBuffer([]byte("{}")))
			w := httptest.NewRecorder()

			// Act
			handler.HandleProfiles(w, req)

			// Assert
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestAPIHandler_HandleObjectsList(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    ListObjectsRequest
		hasS3Service   bool
		listResult     *service.ListObjectsOutput
		listError      error
		expectedStatus int
	}{
		{
			name: "successful objects listing",
			requestBody: ListObjectsRequest{
				Bucket: "test-bucket",
				Prefix: "folder/",
			},
			hasS3Service: true,
			listResult: &service.ListObjectsOutput{
				Objects: []service.S3Object{
					{Key: "folder/file1.txt", Size: 1024},
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing bucket parameter",
			requestBody: ListObjectsRequest{
				Prefix: "folder/",
			},
			hasS3Service:   true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "no S3 service configured",
			requestBody: ListObjectsRequest{
				Bucket: "test-bucket",
			},
			hasS3Service:   false,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "S3 service error",
			requestBody: ListObjectsRequest{
				Bucket: "test-bucket",
			},
			hasS3Service:   true,
			listError:      errors.New("S3 error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			handler := NewAPIHandler(nil, nil, slog.Default())
			if tt.hasS3Service {
				handler.s3Service = &mockS3Service{
					listObjectsResult: tt.listResult,
					listObjectsErr:    tt.listError,
				}
			}

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/objects/list", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			// Act
			handler.HandleObjectsList(w, req)

			// Assert
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestAPIHandler_HandleObjectsDelete(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    DeleteObjectsRequest
		hasS3Service   bool
		deleteError    error
		expectedStatus int
	}{
		{
			name: "successful objects deletion",
			requestBody: DeleteObjectsRequest{
				Bucket: "test-bucket",
				Keys:   []string{"file1.txt", "file2.txt"},
			},
			hasS3Service:   true,
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing bucket",
			requestBody: DeleteObjectsRequest{
				Keys: []string{"file1.txt"},
			},
			hasS3Service:   true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing keys",
			requestBody: DeleteObjectsRequest{
				Bucket: "test-bucket",
				Keys:   []string{},
			},
			hasS3Service:   true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "S3 delete error",
			requestBody: DeleteObjectsRequest{
				Bucket: "test-bucket",
				Keys:   []string{"file1.txt"},
			},
			hasS3Service:   true,
			deleteError:    errors.New("delete failed"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			handler := NewAPIHandler(nil, nil, slog.Default())
			if tt.hasS3Service {
				handler.s3Service = &mockS3Service{
					deleteObjectErr:  tt.deleteError,
					deleteObjectsErr: tt.deleteError,
				}
			}

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/objects/delete", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			// Act
			handler.HandleObjectsDelete(w, req)

			// Assert
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestAPIHandler_HandleObjectsUpload(t *testing.T) {
	t.Run("successful multiple file upload", func(t *testing.T) {
		// Arrange
		mockService := &mockS3Service{
			uploadResult: &service.UploadObjectOutput{
				Key:  "test-key",
				ETag: "etag-123",
			},
		}

		handler := NewAPIHandler(nil, nil, slog.Default())
		handler.s3Service = mockService

		// Create multipart form
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)

		// Add form fields
		writer.WriteField("bucket", "test-bucket")
		writer.WriteField("uploads", `[{"key": "file1.txt", "file": "file1"}, {"key": "file2.txt", "file": "file2"}]`)

		// Add files
		fileWriter1, _ := writer.CreateFormFile("file1", "test1.txt")
		fileWriter1.Write([]byte("content1"))
		fileWriter2, _ := writer.CreateFormFile("file2", "test2.txt")
		fileWriter2.Write([]byte("content2"))
		writer.Close()

		req := httptest.NewRequest("POST", "/api/objects/upload", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		// Act
		handler.HandleObjectsUpload(w, req)

		// Assert
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
			t.Logf("Response: %s", w.Body.String())
		}
	})

	t.Run("missing bucket parameter", func(t *testing.T) {
		// Arrange
		handler := NewAPIHandler(nil, nil, slog.Default())
		handler.s3Service = &mockS3Service{}

		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		writer.WriteField("uploads", `[{"key": "file1.txt", "file": "file1"}]`)
		writer.Close()

		req := httptest.NewRequest("POST", "/api/objects/upload", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		// Act
		handler.HandleObjectsUpload(w, req)

		// Assert
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestAPIHandler_HandleObjectsDownload(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    DownloadObjectRequest
		hasS3Service   bool
		downloadResult *service.DownloadObjectOutput
		downloadError  error
		expectedStatus int
	}{
		{
			name: "successful single file download",
			requestBody: DownloadObjectRequest{
				Bucket: "test-bucket",
				Type:   "files",
				Keys:   []string{"file.txt"},
			},
			hasS3Service: true,
			downloadResult: &service.DownloadObjectOutput{
				Body:          []byte("content"),
				ContentType:   "text/plain",
				ContentLength: 7,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing bucket",
			requestBody: DownloadObjectRequest{
				Type: "files",
				Keys: []string{"file.txt"},
			},
			hasS3Service:   true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid type",
			requestBody: DownloadObjectRequest{
				Bucket: "test-bucket",
				Type:   "invalid",
			},
			hasS3Service:   true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing keys for files download",
			requestBody: DownloadObjectRequest{
				Bucket: "test-bucket",
				Type:   "files",
				Keys:   []string{},
			},
			hasS3Service:   true,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			handler := NewAPIHandler(nil, nil, slog.Default())
			if tt.hasS3Service {
				handler.s3Service = &mockS3Service{
					downloadResult: tt.downloadResult,
					downloadErr:    tt.downloadError,
				}
			}

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/objects/download", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			// Act
			handler.HandleObjectsDownload(w, req)

			// Assert
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
				t.Logf("Response: %s", w.Body.String())
			}
		})
	}
}

func TestAPIHandler_ErrorHandling(t *testing.T) {
	t.Run("invalid JSON in request body", func(t *testing.T) {
		// Arrange
		handler := NewAPIHandler(nil, nil, slog.Default())
		req := httptest.NewRequest("POST", "/api/objects/list", strings.NewReader("invalid json"))
		w := httptest.NewRecorder()

		// Act
		handler.HandleObjectsList(w, req)

		// Assert
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}

		var response APIResponse
		json.NewDecoder(w.Body).Decode(&response)
		if response.Success {
			t.Error("Expected success to be false for invalid JSON")
		}
	})
}

func TestValidateBucketName(t *testing.T) {
	tests := []struct {
		name        string
		bucketName  string
		expectError bool
	}{
		{
			name:        "valid bucket name",
			bucketName:  "my-test-bucket",
			expectError: false,
		},
		{
			name:        "valid bucket name with numbers",
			bucketName:  "bucket123",
			expectError: false,
		},
		{
			name:        "valid bucket name with dots",
			bucketName:  "my.test.bucket",
			expectError: false,
		},
		{
			name:        "too short",
			bucketName:  "ab",
			expectError: true,
		},
		{
			name:        "too long",
			bucketName:  "a-very-long-bucket-name-that-exceeds-the-maximum-allowed-length-of-sixty-three-characters",
			expectError: true,
		},
		{
			name:        "contains uppercase",
			bucketName:  "MyBucket",
			expectError: true,
		},
		{
			name:        "starts with hyphen",
			bucketName:  "-mybucket",
			expectError: true,
		},
		{
			name:        "ends with hyphen",
			bucketName:  "mybucket-",
			expectError: true,
		},
		{
			name:        "contains adjacent periods",
			bucketName:  "my..bucket",
			expectError: true,
		},
		{
			name:        "starts with xn--",
			bucketName:  "xn--mybucket",
			expectError: true,
		},
		{
			name:        "ends with -s3alias",
			bucketName:  "mybucket-s3alias",
			expectError: true,
		},
		{
			name:        "ip address format",
			bucketName:  "192.168.1.1",
			expectError: true,
		},
		{
			name:        "contains invalid characters",
			bucketName:  "my_bucket",
			expectError: true,
		},
		{
			name:        "empty string",
			bucketName:  "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBucketName(tt.bucketName)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for bucket name '%s', but got none", tt.bucketName)
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for bucket name '%s', but got: %v", tt.bucketName, err)
			}
		})
	}
}

func TestAPIHandler_HandleBucketCreate(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    CreateBucketRequest
		hasS3Service   bool
		createError    error
		expectedStatus int
	}{
		{
			name: "successful bucket creation",
			requestBody: CreateBucketRequest{
				Name: "test-bucket-123",
			},
			hasS3Service:   true,
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing bucket name",
			requestBody: CreateBucketRequest{
				Name: "",
			},
			hasS3Service:   true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid bucket name",
			requestBody: CreateBucketRequest{
				Name: "InvalidBucketName",
			},
			hasS3Service:   true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "no S3 service configured",
			requestBody: CreateBucketRequest{
				Name: "test-bucket",
			},
			hasS3Service:   false,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "S3 service error",
			requestBody: CreateBucketRequest{
				Name: "test-bucket",
			},
			hasS3Service:   true,
			createError:    errors.New("bucket already exists"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			handler := NewAPIHandler(nil, nil, slog.Default())

			if tt.hasS3Service {
				handler.s3Service = &mockS3Service{
					createBucketErr: tt.createError,
				}
			}

			bodyBytes, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/buckets/create", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Act
			handler.HandleBucketCreate(w, req)

			// Assert
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
				t.Logf("Response body: %s", w.Body.String())
			}

			// For successful creation, verify response structure
			if tt.expectedStatus == http.StatusOK {
				var response APIResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if !response.Success {
					t.Error("Expected success to be true")
				}

				if response.Data == nil {
					t.Error("Expected data to be present")
				}
			}
		})
	}
}
