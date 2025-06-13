package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

			deps := &Dependencies{
				ProfileProvider: mockProfileRepo,
			}
			handler := NewAPIHandler(deps)

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

			mockFactory := &MockS3ServiceFactory{
				CreateS3ServiceFunc: func(ctx context.Context, cfg service.S3Config) (service.S3Operations, error) {
					if tt.createServiceError != nil {
						return nil, tt.createServiceError
					}
					return mockS3Service, nil
				},
			}

			deps := &Dependencies{
				S3ServiceFactory: mockFactory,
			}
			handler := NewAPIHandler(deps)

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
			deps := &Dependencies{}
			handler := NewAPIHandler(deps)

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

type MockS3ServiceFactory struct {
	CreateS3ServiceFunc func(ctx context.Context, cfg service.S3Config) (service.S3Operations, error)
}

func (f *MockS3ServiceFactory) CreateS3Service(ctx context.Context, cfg service.S3Config) (service.S3Operations, error) {
	if f.CreateS3ServiceFunc != nil {
		return f.CreateS3ServiceFunc(ctx, cfg)
	}
	return service.NewMockS3Service(), nil
}