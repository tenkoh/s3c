package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

// Integration tests using real ServeMux to test routing and PathValue
func TestAPIHandler_Integration(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		url            string
		expectedStatus int
		setupHandler   func(*APIHandler)
	}{
		{
			name:           "GET /api/profiles success",
			method:         "GET",
			url:            "/api/profiles",
			expectedStatus: http.StatusOK,
			setupHandler: func(h *APIHandler) {
				h.profileProvider = &mockProfileProvider{
					profiles: []string{"default", "work"},
				}
			},
		},
		{
			name:           "GET /api/buckets/{bucket}/objects extracts bucket correctly",
			method:         "GET",
			url:            "/api/buckets/test-bucket/objects",
			expectedStatus: http.StatusOK,
			setupHandler: func(h *APIHandler) {
				h.s3Service = &mockS3Service{
					listObjectsResult: &service.ListObjectsOutput{
						Objects: []service.S3Object{
							{Key: "file1.txt", Size: 1024},
						},
					},
				}
			},
		},
		{
			name:           "GET /api/buckets/{bucket}/objects/{key...} extracts parameters correctly",
			method:         "GET",
			url:            "/api/buckets/test-bucket/objects/folder/file.txt",
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
			name:           "DELETE /api/buckets/{bucket}/objects extracts bucket correctly",
			method:         "DELETE",
			url:            "/api/buckets/test-bucket/objects",
			expectedStatus: http.StatusOK,
			setupHandler: func(h *APIHandler) {
				h.s3Service = &mockS3Service{}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange: Setup real ServeMux with routing
			mux := http.NewServeMux()
			handler := NewAPIHandler(nil, nil)
			tt.setupHandler(handler)

			// Setup routes (similar to server.go)
			mux.HandleFunc("GET /api/profiles", handler.HandleProfiles)
			mux.HandleFunc("GET /api/buckets/{bucket}/objects", handler.HandleObjects)
			mux.HandleFunc("DELETE /api/buckets/{bucket}/objects", handler.HandleDeleteObjects)
			mux.HandleFunc("GET /api/buckets/{bucket}/objects/{key...}", handler.HandleDownload)

			var body *bytes.Buffer
			if tt.method == "DELETE" {
				// Provide valid JSON for DELETE requests
				body = bytes.NewBuffer([]byte(`{"keys":["test.txt"]}`))
			} else {
				body = &bytes.Buffer{}
			}

			req := httptest.NewRequest(tt.method, tt.url, body)
			w := httptest.NewRecorder()

			// Act: Execute request through ServeMux
			mux.ServeHTTP(w, req)

			// Assert: Verify response
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
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
		expectedData   map[string]interface{}
	}{
		{
			name:           "successful profiles retrieval",
			profilesResult: []string{"default", "work"},
			expectedStatus: http.StatusOK,
			expectedData:   map[string]interface{}{"profiles": []interface{}{"default", "work"}},
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
			handler := NewAPIHandler(mockProvider, nil)
			req := httptest.NewRequest("GET", "/api/profiles", nil)
			w := httptest.NewRecorder()

			// Act
			handler.HandleProfiles(w, req)

			// Assert
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var response APIResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if !response.Success {
					t.Error("Expected success to be true")
				}

				// Compare profiles data
				actualData := response.Data.(map[string]interface{})
				expectedProfiles := tt.expectedData["profiles"].([]interface{})
				actualProfiles := actualData["profiles"].([]interface{})

				if len(actualProfiles) != len(expectedProfiles) {
					t.Errorf("Expected %d profiles, got %d", len(expectedProfiles), len(actualProfiles))
				}
			}
		})
	}
}

func TestAPIHandler_HandleSettings(t *testing.T) {
	tests := []struct {
		name                string
		requestBody         service.S3Config
		createServiceError  error
		testConnectionError error
		expectedStatus      int
	}{
		{
			name: "successful settings configuration",
			requestBody: service.S3Config{
				Profile: "default",
				Region:  "us-east-1",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing profile",
			requestBody: service.S3Config{
				Region: "us-east-1",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing region",
			requestBody: service.S3Config{
				Profile: "default",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "service creation error",
			requestBody: service.S3Config{
				Profile: "default",
				Region:  "us-east-1",
			},
			createServiceError: errors.New("creation_error"),
			expectedStatus:     http.StatusInternalServerError,
		},
		{
			name: "connection test error",
			requestBody: service.S3Config{
				Profile: "default",
				Region:  "us-east-1",
			},
			testConnectionError: errors.New("connection failed"),
			expectedStatus:      http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockService := &mockS3Service{
				testConnectionErr: tt.testConnectionError,
			}
			if tt.createServiceError != nil {
				mockService.testConnectionErr = tt.createServiceError
			}

			handler := NewAPIHandler(nil, mockS3ServiceCreator(mockService))

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/settings", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			// Act
			handler.HandleSettings(w, req)

			// Assert
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
				t.Logf("Response body: %s", w.Body.String())
			}
		})
	}
}

func TestAPIHandler_HandleBuckets(t *testing.T) {
	tests := []struct {
		name           string
		hasS3Service   bool
		bucketsResult  []string
		bucketsError   error
		expectedStatus int
	}{
		{
			name:           "successful buckets listing",
			hasS3Service:   true,
			bucketsResult:  []string{"bucket1", "bucket2"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "no S3 service configured",
			hasS3Service:   false,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "S3 service error",
			hasS3Service:   true,
			bucketsError:   errors.New("S3 error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			handler := NewAPIHandler(nil, nil)
			if tt.hasS3Service {
				handler.s3Service = &mockS3Service{
					listBucketsResult: tt.bucketsResult,
					listBucketsErr:    tt.bucketsError,
				}
			}

			req := httptest.NewRequest("GET", "/api/buckets", nil)
			w := httptest.NewRecorder()

			// Act
			handler.HandleBuckets(w, req)

			// Assert
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestAPIHandler_HandleUpload(t *testing.T) {
	t.Run("successful file upload", func(t *testing.T) {
		// Arrange
		mockService := &mockS3Service{
			uploadResult: &service.UploadObjectOutput{
				Key:  "test-key",
				ETag: "etag-123",
			},
		}

		mux := http.NewServeMux()
		handler := NewAPIHandler(nil, nil)
		handler.s3Service = mockService
		mux.HandleFunc("POST /api/buckets/{bucket}/objects", handler.HandleUpload)

		// Create multipart form
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		writer.WriteField("key", "test-key")
		fileWriter, _ := writer.CreateFormFile("file", "test.txt")
		fileWriter.Write([]byte("test content"))
		writer.Close()

		req := httptest.NewRequest("POST", "/api/buckets/test-bucket/objects", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		// Act
		mux.ServeHTTP(w, req)

		// Assert
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
			t.Logf("Response: %s", w.Body.String())
		}
	})
}

func TestAPIHandler_HandleDeleteObjects(t *testing.T) {
	t.Run("successful objects deletion", func(t *testing.T) {
		// Arrange
		mockService := &mockS3Service{}

		mux := http.NewServeMux()
		handler := NewAPIHandler(nil, nil)
		handler.s3Service = mockService
		mux.HandleFunc("DELETE /api/buckets/{bucket}/objects", handler.HandleDeleteObjects)

		reqBody := DeleteObjectRequest{
			Keys: []string{"file1.txt", "file2.txt"},
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("DELETE", "/api/buckets/test-bucket/objects", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		// Act
		mux.ServeHTTP(w, req)

		// Assert
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
			t.Logf("Response: %s", w.Body.String())
		}
	})

	t.Run("missing keys validation", func(t *testing.T) {
		// Arrange
		handler := NewAPIHandler(nil, nil)
		handler.s3Service = &mockS3Service{}

		reqBody := DeleteObjectRequest{Keys: []string{}}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("DELETE", "/api/delete", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		// Act
		handler.HandleDeleteObjects(w, req)

		// Assert
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestAPIHandler_ErrorHandling(t *testing.T) {
	t.Run("invalid JSON in request body", func(t *testing.T) {
		// Arrange
		handler := NewAPIHandler(nil, nil)
		req := httptest.NewRequest("POST", "/api/settings", strings.NewReader("invalid json"))
		w := httptest.NewRecorder()

		// Act
		handler.HandleSettings(w, req)

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
