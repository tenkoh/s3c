package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tenkoh/s3c/pkg/service"
)

func TestAPIHandler_HandleProfiles(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		profilesResult []string
		profilesError  error
		expectedStatus int
		expectedData   interface{}
	}{
		{
			name:           "successful profiles retrieval",
			method:         "GET",
			profilesResult: []string{"default", "work"},
			profilesError:  nil,
			expectedStatus: http.StatusOK,
			expectedData:   map[string]interface{}{"profiles": []interface{}{"default", "work"}},
		},
		{
			name:           "profiles error",
			method:         "GET",
			profilesResult: nil,
			profilesError:  errors.New("file not found"),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "method not allowed",
			method:         "POST",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			mockProfileRepo := &MockProfileRepository{
				GetProfilesFunc: func() ([]string, error) {
					return tt.profilesResult, tt.profilesError
				},
			}

			handler := NewAPIHandler(mockProfileRepo, nil)

			// Create request
			req := httptest.NewRequest(tt.method, "/api/profiles", nil)
			w := httptest.NewRecorder()

			// Execute
			handler.HandleProfiles(w, req)

			// Assert
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var response APIResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if !response.Success {
					t.Error("Expected success to be true")
				}

				// Compare profiles data using go-cmp
				actualData, ok := response.Data.(map[string]interface{})
				if !ok {
					t.Fatal("Response data is not a map")
				}

				if diff := cmp.Diff(tt.expectedData, actualData); diff != "" {
					t.Errorf("Response data mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestAPIHandler_HandleSettings(t *testing.T) {
	tests := []struct {
		name                string
		method              string
		requestBody         interface{}
		createServiceError  error
		testConnectionError error
		expectedStatus      int
	}{
		{
			name:   "successful settings configuration",
			method: "POST",
			requestBody: service.S3Config{
				Profile: "default",
				Region:  "us-east-1",
			},
			createServiceError:  nil,
			testConnectionError: nil,
			expectedStatus:      http.StatusOK,
		},
		{
			name:   "missing profile",
			method: "POST",
			requestBody: service.S3Config{
				Region: "us-east-1",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "missing region",
			method: "POST",
			requestBody: service.S3Config{
				Profile: "default",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "method not allowed",
			method:         "GET",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "service creation error",
			method: "POST",
			requestBody: service.S3Config{
				Profile: "default",
				Region:  "us-east-1",
			},
			createServiceError: errors.New("AWS config error"),
			expectedStatus:     http.StatusInternalServerError,
		},
		{
			name:   "connection test error",
			method: "POST",
			requestBody: service.S3Config{
				Profile: "default",
				Region:  "us-east-1",
			},
			createServiceError:  nil,
			testConnectionError: errors.New("connection failed"),
			expectedStatus:      http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			mockS3Service := service.NewMockS3Service()
			mockS3Service.TestConnectionFunc = func(ctx context.Context) error {
				return tt.testConnectionError
			}

			mockCreator := func(ctx context.Context, cfg service.S3Config) (service.S3Operations, error) {
				if tt.createServiceError != nil {
					return nil, tt.createServiceError
				}
				return mockS3Service, nil
			}

			handler := NewAPIHandler(nil, mockCreator)

			// Create request
			var body bytes.Buffer
			if tt.requestBody != nil {
				json.NewEncoder(&body).Encode(tt.requestBody)
			}

			req := httptest.NewRequest(tt.method, "/api/settings", &body)
			w := httptest.NewRecorder()

			// Execute
			handler.HandleSettings(w, req)

			// Assert
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestAPIHandler_HandleBuckets(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		hasS3Service   bool
		bucketsResult  []string
		bucketsError   error
		expectedStatus int
	}{
		{
			name:           "successful buckets listing",
			method:         "GET",
			hasS3Service:   true,
			bucketsResult:  []string{"bucket1", "bucket2"},
			bucketsError:   nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "no S3 service configured",
			method:         "GET",
			hasS3Service:   false,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "S3 error",
			method:         "GET",
			hasS3Service:   true,
			bucketsResult:  nil,
			bucketsError:   errors.New("S3 error"),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "method not allowed",
			method:         "POST",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAPIHandler(nil, nil)

			// Setup S3 service if needed
			if tt.hasS3Service {
				mockS3Service := service.NewMockS3Service()
				mockS3Service.ListBucketsFunc = func(ctx context.Context) ([]string, error) {
					return tt.bucketsResult, tt.bucketsError
				}
				handler.s3Service = mockS3Service
			}

			// Create request
			req := httptest.NewRequest(tt.method, "/api/buckets", nil)
			w := httptest.NewRecorder()

			// Execute
			handler.HandleBuckets(w, req)

			// Assert
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// Mock implementations for testing

type MockProfileRepository struct {
	GetProfilesFunc func() ([]string, error)
}

func (m *MockProfileRepository) GetProfiles() ([]string, error) {
	if m.GetProfilesFunc != nil {
		return m.GetProfilesFunc()
	}
	return []string{}, nil
}

func TestAPIHandler_HandleObjects(t *testing.T) {
	tests := []struct {
		name            string
		method          string
		queryParams     string
		hasS3Service    bool
		listObjectsFunc func(ctx context.Context, input service.ListObjectsInput) (*service.ListObjectsOutput, error)
		expectedStatus  int
	}{
		{
			name:         "successful objects listing",
			method:       "GET",
			queryParams:  "bucket=test-bucket&prefix=folder/&delimiter=/",
			hasS3Service: true,
			listObjectsFunc: func(ctx context.Context, input service.ListObjectsInput) (*service.ListObjectsOutput, error) {
				return &service.ListObjectsOutput{
					Objects: []service.S3Object{
						{Key: "folder/file1.txt", Size: 1024, IsFolder: false},
						{Key: "folder/subfolder", Size: 0, IsFolder: true},
					},
					CommonPrefixes: []string{"folder/subfolder"},
					IsTruncated:    false,
				}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing bucket parameter",
			method:         "GET",
			queryParams:    "prefix=folder/",
			hasS3Service:   true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "no S3 service configured",
			method:         "GET",
			queryParams:    "bucket=test-bucket",
			hasS3Service:   false,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "S3 error",
			method:       "GET",
			queryParams:  "bucket=test-bucket",
			hasS3Service: true,
			listObjectsFunc: func(ctx context.Context, input service.ListObjectsInput) (*service.ListObjectsOutput, error) {
				return nil, errors.New("S3 error")
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "method not allowed",
			method:         "POST",
			queryParams:    "bucket=test-bucket",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAPIHandler(nil, nil)

			// Setup S3 service if needed
			if tt.hasS3Service {
				mockS3Service := service.NewMockS3Service()
				if tt.listObjectsFunc != nil {
					mockS3Service.ListObjectsFunc = tt.listObjectsFunc
				}
				handler.s3Service = mockS3Service
			}

			// Create request
			url := "/api/objects"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}
			req := httptest.NewRequest(tt.method, url, nil)
			w := httptest.NewRecorder()

			// Execute
			handler.HandleObjects(w, req)

			// Assert
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var response APIResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if !response.Success {
					t.Error("Expected success to be true")
				}

				// Verify the response contains expected object data
				output, ok := response.Data.(*service.ListObjectsOutput)
				if !ok {
					// Handle case where Data is unmarshaled as map[string]interface{}
					dataMap, mapOk := response.Data.(map[string]interface{})
					if !mapOk {
						t.Fatal("Response data is not of expected type")
					}

					objects, exists := dataMap["objects"]
					if !exists {
						t.Error("Response should contain objects field")
					}

					objectsList, isList := objects.([]interface{})
					if !isList || len(objectsList) == 0 {
						t.Error("Expected objects list to contain items")
					}
				} else {
					if len(output.Objects) == 0 {
						t.Error("Expected objects list to contain items")
					}
				}
			}
		})
	}
}

func TestAPIHandler_HandleDeleteObjects(t *testing.T) {
	tests := []struct {
		name              string
		method            string
		requestBody       interface{}
		hasS3Service      bool
		deleteObjectFunc  func(ctx context.Context, bucket, key string) error
		deleteObjectsFunc func(ctx context.Context, bucket string, keys []string) error
		expectedStatus    int
	}{
		{
			name:   "successful single object deletion",
			method: "DELETE",
			requestBody: DeleteObjectRequest{
				Bucket: "test-bucket",
				Keys:   []string{"test-key"},
			},
			hasS3Service: true,
			deleteObjectFunc: func(ctx context.Context, bucket, key string) error {
				return nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "successful multiple objects deletion",
			method: "DELETE",
			requestBody: DeleteObjectRequest{
				Bucket: "test-bucket",
				Keys:   []string{"key1", "key2", "key3"},
			},
			hasS3Service: true,
			deleteObjectsFunc: func(ctx context.Context, bucket string, keys []string) error {
				return nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "missing bucket",
			method: "DELETE",
			requestBody: DeleteObjectRequest{
				Keys: []string{"test-key"},
			},
			hasS3Service:   true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "missing keys",
			method: "DELETE",
			requestBody: DeleteObjectRequest{
				Bucket: "test-bucket",
				Keys:   []string{},
			},
			hasS3Service:   true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "no S3 service configured",
			method:         "DELETE",
			requestBody:    DeleteObjectRequest{Bucket: "test-bucket", Keys: []string{"key"}},
			hasS3Service:   false,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "S3 delete error",
			method: "DELETE",
			requestBody: DeleteObjectRequest{
				Bucket: "test-bucket",
				Keys:   []string{"test-key"},
			},
			hasS3Service: true,
			deleteObjectFunc: func(ctx context.Context, bucket, key string) error {
				return errors.New("delete failed")
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "method not allowed",
			method:         "GET",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "invalid JSON",
			method:         "DELETE",
			requestBody:    "invalid json",
			hasS3Service:   true,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAPIHandler(nil, nil)

			// Setup S3 service if needed
			if tt.hasS3Service {
				mockS3Service := service.NewMockS3Service()
				if tt.deleteObjectFunc != nil {
					mockS3Service.DeleteObjectFunc = tt.deleteObjectFunc
				}
				if tt.deleteObjectsFunc != nil {
					mockS3Service.DeleteObjectsFunc = tt.deleteObjectsFunc
				}
				handler.s3Service = mockS3Service
			}

			// Create request
			var body bytes.Buffer
			if tt.requestBody != nil {
				if str, ok := tt.requestBody.(string); ok {
					body.WriteString(str)
				} else {
					json.NewEncoder(&body).Encode(tt.requestBody)
				}
			}

			req := httptest.NewRequest(tt.method, "/api/objects/delete", &body)
			w := httptest.NewRecorder()

			// Execute
			handler.HandleDeleteObjects(w, req)

			// Assert
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var response APIResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if !response.Success {
					t.Error("Expected success to be true")
				}
			}
		})
	}
}

func TestAPIHandler_HandleUpload(t *testing.T) {
	tests := []struct {
		name             string
		hasS3Service     bool
		uploadObjectFunc func(ctx context.Context, input service.UploadObjectInput) (*service.UploadObjectOutput, error)
		expectedStatus   int
	}{
		{
			name:         "successful file upload",
			hasS3Service: true,
			uploadObjectFunc: func(ctx context.Context, input service.UploadObjectInput) (*service.UploadObjectOutput, error) {
				return &service.UploadObjectOutput{
					Key:  "test-key",
					ETag: "\"etag-12345\"",
				}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "no S3 service configured",
			hasS3Service:   false,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "S3 upload error",
			hasS3Service: true,
			uploadObjectFunc: func(ctx context.Context, input service.UploadObjectInput) (*service.UploadObjectOutput, error) {
				return nil, errors.New("upload failed")
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAPIHandler(nil, nil)

			// Setup S3 service if needed
			if tt.hasS3Service {
				mockS3Service := service.NewMockS3Service()
				if tt.uploadObjectFunc != nil {
					mockS3Service.UploadObjectFunc = tt.uploadObjectFunc
				}
				handler.s3Service = mockS3Service
			}

			// Create multipart form request
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)

			// Add form fields
			writer.WriteField("bucket", "test-bucket")
			writer.WriteField("key", "test-key")

			// Add file
			fileWriter, err := writer.CreateFormFile("file", "test.txt")
			if err != nil {
				t.Fatalf("Failed to create form file: %v", err)
			}
			fileWriter.Write([]byte("test file content"))
			writer.Close()

			req := httptest.NewRequest("POST", "/api/upload", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			w := httptest.NewRecorder()

			// Execute
			handler.HandleUpload(w, req)

			// Assert
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var response APIResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if !response.Success {
					t.Error("Expected success to be true")
				}
			}
		})
	}
}

func TestAPIHandler_HandleDownload(t *testing.T) {
	tests := []struct {
		name               string
		queryParams        string
		hasS3Service       bool
		downloadObjectFunc func(ctx context.Context, input service.DownloadObjectInput) (*service.DownloadObjectOutput, error)
		expectedStatus     int
	}{
		{
			name:         "successful file download",
			queryParams:  "bucket=test-bucket&key=test-key",
			hasS3Service: true,
			downloadObjectFunc: func(ctx context.Context, input service.DownloadObjectInput) (*service.DownloadObjectOutput, error) {
				return &service.DownloadObjectOutput{
					Body:          []byte("test file content"),
					ContentType:   "text/plain",
					ContentLength: 17,
					LastModified:  "2023-01-01T00:00:00Z",
					Metadata:      map[string]string{"original-filename": "test.txt"},
				}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing bucket parameter",
			queryParams:    "key=test-key",
			hasS3Service:   true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing key parameter",
			queryParams:    "bucket=test-bucket",
			hasS3Service:   true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "no S3 service configured",
			queryParams:    "bucket=test-bucket&key=test-key",
			hasS3Service:   false,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "S3 download error",
			queryParams:  "bucket=test-bucket&key=test-key",
			hasS3Service: true,
			downloadObjectFunc: func(ctx context.Context, input service.DownloadObjectInput) (*service.DownloadObjectOutput, error) {
				return nil, errors.New("download failed")
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAPIHandler(nil, nil)

			// Setup S3 service if needed
			if tt.hasS3Service {
				mockS3Service := service.NewMockS3Service()
				if tt.downloadObjectFunc != nil {
					mockS3Service.DownloadObjectFunc = tt.downloadObjectFunc
				}
				handler.s3Service = mockS3Service
			}

			// Create request
			url := "/api/download"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}
			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			// Execute
			handler.HandleDownload(w, req)

			// Assert
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				// Check content headers
				if w.Header().Get("Content-Type") != "text/plain" {
					t.Errorf("Expected Content-Type text/plain, got %s", w.Header().Get("Content-Type"))
				}
				if w.Header().Get("Content-Length") != "17" {
					t.Errorf("Expected Content-Length 17, got %s", w.Header().Get("Content-Length"))
				}

				// Check file content
				if w.Body.String() != "test file content" {
					t.Errorf("Expected file content 'test file content', got '%s'", w.Body.String())
				}
			}
		})
	}
}
