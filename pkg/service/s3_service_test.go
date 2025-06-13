package service

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMockS3Service_ListBuckets(t *testing.T) {
	tests := []struct {
		name           string
		mockFunc       func(ctx context.Context) ([]string, error)
		expectedResult []string
		expectError    bool
	}{
		{
			name: "successful bucket listing",
			mockFunc: func(ctx context.Context) ([]string, error) {
				return []string{"bucket1", "bucket2"}, nil
			},
			expectedResult: []string{"bucket1", "bucket2"},
			expectError:    false,
		},
		{
			name: "empty bucket list",
			mockFunc: func(ctx context.Context) ([]string, error) {
				return []string{}, nil
			},
			expectedResult: []string{},
			expectError:    false,
		},
		{
			name:           "default behavior",
			mockFunc:       nil,
			expectedResult: []string{},
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockS3Service()
			mock.ListBucketsFunc = tt.mockFunc

			result, err := mock.ListBuckets(context.Background())

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if diff := cmp.Diff(tt.expectedResult, result); diff != "" {
				t.Errorf("ListBuckets() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMockS3Service_ListObjects(t *testing.T) {
	tests := []struct {
		name           string
		input          ListObjectsInput
		mockFunc       func(ctx context.Context, input ListObjectsInput) (*ListObjectsOutput, error)
		expectedResult *ListObjectsOutput
		expectError    bool
	}{
		{
			name: "successful object listing",
			input: ListObjectsInput{
				Bucket:  "test-bucket",
				Prefix:  "folder/",
				MaxKeys: 10,
			},
			mockFunc: func(ctx context.Context, input ListObjectsInput) (*ListObjectsOutput, error) {
				return &ListObjectsOutput{
					Objects: []S3Object{
						{Key: "folder/file1.txt", Size: 1024, IsFolder: false},
						{Key: "folder/file2.txt", Size: 2048, IsFolder: false},
					},
					CommonPrefixes: []string{},
					IsTruncated:    false,
				}, nil
			},
			expectedResult: &ListObjectsOutput{
				Objects: []S3Object{
					{Key: "folder/file1.txt", Size: 1024, IsFolder: false},
					{Key: "folder/file2.txt", Size: 2048, IsFolder: false},
				},
				CommonPrefixes: []string{},
				IsTruncated:    false,
			},
			expectError: false,
		},
		{
			name: "default behavior",
			input: ListObjectsInput{
				Bucket: "test-bucket",
			},
			mockFunc: nil,
			expectedResult: &ListObjectsOutput{
				Objects:        []S3Object{},
				CommonPrefixes: []string{},
				IsTruncated:    false,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockS3Service()
			mock.ListObjectsFunc = tt.mockFunc

			result, err := mock.ListObjects(context.Background(), tt.input)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if diff := cmp.Diff(tt.expectedResult, result); diff != "" {
				t.Errorf("ListObjects() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMockS3Service_DeleteObject(t *testing.T) {
	tests := []struct {
		name        string
		bucket      string
		key         string
		mockFunc    func(ctx context.Context, bucket, key string) error
		expectError bool
	}{
		{
			name:   "successful deletion",
			bucket: "test-bucket",
			key:    "test-key",
			mockFunc: func(ctx context.Context, bucket, key string) error {
				return nil
			},
			expectError: false,
		},
		{
			name:        "default behavior",
			bucket:      "test-bucket",
			key:         "test-key",
			mockFunc:    nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockS3Service()
			mock.DeleteObjectFunc = tt.mockFunc

			err := mock.DeleteObject(context.Background(), tt.bucket, tt.key)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestMockS3Service_DeleteObjects(t *testing.T) {
	tests := []struct {
		name        string
		bucket      string
		keys        []string
		mockFunc    func(ctx context.Context, bucket string, keys []string) error
		expectError bool
	}{
		{
			name:   "successful batch deletion",
			bucket: "test-bucket",
			keys:   []string{"key1", "key2", "key3"},
			mockFunc: func(ctx context.Context, bucket string, keys []string) error {
				return nil
			},
			expectError: false,
		},
		{
			name:        "default behavior",
			bucket:      "test-bucket",
			keys:        []string{"key1"},
			mockFunc:    nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockS3Service()
			mock.DeleteObjectsFunc = tt.mockFunc

			err := mock.DeleteObjects(context.Background(), tt.bucket, tt.keys)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestMockS3Service_TestConnection(t *testing.T) {
	tests := []struct {
		name        string
		mockFunc    func(ctx context.Context) error
		expectError bool
	}{
		{
			name: "successful connection test",
			mockFunc: func(ctx context.Context) error {
				return nil
			},
			expectError: false,
		},
		{
			name:        "default behavior",
			mockFunc:    nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockS3Service()
			mock.TestConnectionFunc = tt.mockFunc

			err := mock.TestConnection(context.Background())

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestMockS3ServiceFactory_CreateS3Service(t *testing.T) {
	tests := []struct {
		name        string
		config      S3Config
		mockService *MockS3Service
		expectError bool
	}{
		{
			name: "successful service creation",
			config: S3Config{
				Profile: "default",
				Region:  "us-east-1",
			},
			mockService: NewMockS3Service(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewMockS3ServiceFactory(tt.mockService)

			service, err := factory.CreateS3Service(context.Background(), tt.config)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if service == nil {
				t.Error("Expected service to be non-nil")
			}
		})
	}
}

func TestS3Config(t *testing.T) {
	config := S3Config{
		Profile:     "test-profile",
		EndpointURL: "https://s3.example.com",
		Region:      "us-west-2",
	}

	if config.Profile != "test-profile" {
		t.Errorf("Expected profile 'test-profile', got '%s'", config.Profile)
	}
	if config.EndpointURL != "https://s3.example.com" {
		t.Errorf("Expected endpoint URL 'https://s3.example.com', got '%s'", config.EndpointURL)
	}
	if config.Region != "us-west-2" {
		t.Errorf("Expected region 'us-west-2', got '%s'", config.Region)
	}
}

func TestS3Object(t *testing.T) {
	obj := S3Object{
		Key:          "test/file.txt",
		Size:         1024,
		LastModified: "2023-01-01T00:00:00Z",
		IsFolder:     false,
	}

	if obj.Key != "test/file.txt" {
		t.Errorf("Expected key 'test/file.txt', got '%s'", obj.Key)
	}
	if obj.Size != 1024 {
		t.Errorf("Expected size 1024, got %d", obj.Size)
	}
	if obj.IsFolder {
		t.Error("Expected IsFolder to be false")
	}
}

func TestListObjectsInput(t *testing.T) {
	input := ListObjectsInput{
		Bucket:            "test-bucket",
		Prefix:            "folder/",
		Delimiter:         "/",
		MaxKeys:           100,
		ContinuationToken: "token123",
	}

	if input.Bucket != "test-bucket" {
		t.Errorf("Expected bucket 'test-bucket', got '%s'", input.Bucket)
	}
	if input.Prefix != "folder/" {
		t.Errorf("Expected prefix 'folder/', got '%s'", input.Prefix)
	}
	if input.MaxKeys != 100 {
		t.Errorf("Expected max keys 100, got %d", input.MaxKeys)
	}
}

func TestListObjectsOutput(t *testing.T) {
	output := ListObjectsOutput{
		Objects: []S3Object{
			{Key: "file1.txt", Size: 1024, IsFolder: false},
			{Key: "folder", Size: 0, IsFolder: true},
		},
		CommonPrefixes:        []string{"folder/"},
		IsTruncated:          true,
		NextContinuationToken: "next-token",
	}

	if len(output.Objects) != 2 {
		t.Errorf("Expected 2 objects, got %d", len(output.Objects))
	}
	if len(output.CommonPrefixes) != 1 {
		t.Errorf("Expected 1 common prefix, got %d", len(output.CommonPrefixes))
	}
	if !output.IsTruncated {
		t.Error("Expected IsTruncated to be true")
	}
}