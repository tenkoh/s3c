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

func TestNewMockS3ServiceCreator(t *testing.T) {
	mockService := NewMockS3Service()
	creator := NewMockS3ServiceCreator(mockService)

	config := S3Config{
		Profile: "default",
		Region:  "us-east-1",
	}

	service, err := creator(context.Background(), config)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if service == nil {
		t.Error("Expected service to be non-nil")
	}
	if service != mockService {
		t.Error("Expected to return the same mock service instance")
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
		IsTruncated:           true,
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

func TestMockS3Service_UploadObject(t *testing.T) {
	tests := []struct {
		name           string
		input          UploadObjectInput
		mockFunc       func(ctx context.Context, input UploadObjectInput) (*UploadObjectOutput, error)
		expectedResult *UploadObjectOutput
		expectError    bool
	}{
		{
			name: "successful upload",
			input: UploadObjectInput{
				Bucket:      "test-bucket",
				Key:         "test-key",
				Body:        []byte("test content"),
				ContentType: "text/plain",
			},
			mockFunc: func(ctx context.Context, input UploadObjectInput) (*UploadObjectOutput, error) {
				return &UploadObjectOutput{
					Key:  "test-key",
					ETag: "\"etag-12345\"",
				}, nil
			},
			expectedResult: &UploadObjectOutput{
				Key:  "test-key",
				ETag: "\"etag-12345\"",
			},
			expectError: false,
		},
		{
			name: "default behavior",
			input: UploadObjectInput{
				Bucket: "test-bucket",
				Key:    "test-key",
				Body:   []byte("test content"),
			},
			mockFunc: nil,
			expectedResult: &UploadObjectOutput{
				Key:  "test-key",
				ETag: "\"mock-etag-12345\"",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockS3Service()
			mock.UploadObjectFunc = tt.mockFunc

			result, err := mock.UploadObject(context.Background(), tt.input)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if diff := cmp.Diff(tt.expectedResult, result); diff != "" {
				t.Errorf("UploadObject() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMockS3Service_DownloadObject(t *testing.T) {
	tests := []struct {
		name           string
		input          DownloadObjectInput
		mockFunc       func(ctx context.Context, input DownloadObjectInput) (*DownloadObjectOutput, error)
		expectedResult *DownloadObjectOutput
		expectError    bool
	}{
		{
			name: "successful download",
			input: DownloadObjectInput{
				Bucket: "test-bucket",
				Key:    "test-key",
			},
			mockFunc: func(ctx context.Context, input DownloadObjectInput) (*DownloadObjectOutput, error) {
				return &DownloadObjectOutput{
					Body:          []byte("test file content"),
					ContentType:   "text/plain",
					ContentLength: 17,
					LastModified:  "2023-01-01T00:00:00Z",
					Metadata:      map[string]string{"test": "value"},
				}, nil
			},
			expectedResult: &DownloadObjectOutput{
				Body:          []byte("test file content"),
				ContentType:   "text/plain",
				ContentLength: 17,
				LastModified:  "2023-01-01T00:00:00Z",
				Metadata:      map[string]string{"test": "value"},
			},
			expectError: false,
		},
		{
			name: "default behavior",
			input: DownloadObjectInput{
				Bucket: "test-bucket",
				Key:    "test-key",
			},
			mockFunc: nil,
			expectedResult: &DownloadObjectOutput{
				Body:          []byte("mock file content"),
				ContentType:   "text/plain",
				ContentLength: 17,
				LastModified:  "2023-01-01T00:00:00Z",
				Metadata:      map[string]string{},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockS3Service()
			mock.DownloadObjectFunc = tt.mockFunc

			result, err := mock.DownloadObject(context.Background(), tt.input)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if diff := cmp.Diff(tt.expectedResult, result); diff != "" {
				t.Errorf("DownloadObject() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestUploadObjectInput(t *testing.T) {
	input := UploadObjectInput{
		Bucket:      "test-bucket",
		Key:         "test-key",
		Body:        []byte("test content"),
		ContentType: "text/plain",
		Metadata:    map[string]string{"test": "value"},
	}

	if input.Bucket != "test-bucket" {
		t.Errorf("Expected bucket 'test-bucket', got '%s'", input.Bucket)
	}
	if input.Key != "test-key" {
		t.Errorf("Expected key 'test-key', got '%s'", input.Key)
	}
	if string(input.Body) != "test content" {
		t.Errorf("Expected body 'test content', got '%s'", string(input.Body))
	}
	if input.ContentType != "text/plain" {
		t.Errorf("Expected content type 'text/plain', got '%s'", input.ContentType)
	}
}

func TestUploadObjectOutput(t *testing.T) {
	output := UploadObjectOutput{
		Key:  "test-key",
		ETag: "\"etag-12345\"",
	}

	if output.Key != "test-key" {
		t.Errorf("Expected key 'test-key', got '%s'", output.Key)
	}
	if output.ETag != "\"etag-12345\"" {
		t.Errorf("Expected ETag '\"etag-12345\"', got '%s'", output.ETag)
	}
}

func TestDownloadObjectInput(t *testing.T) {
	input := DownloadObjectInput{
		Bucket: "test-bucket",
		Key:    "test-key",
	}

	if input.Bucket != "test-bucket" {
		t.Errorf("Expected bucket 'test-bucket', got '%s'", input.Bucket)
	}
	if input.Key != "test-key" {
		t.Errorf("Expected key 'test-key', got '%s'", input.Key)
	}
}

func TestDownloadObjectOutput(t *testing.T) {
	output := DownloadObjectOutput{
		Body:          []byte("test content"),
		ContentType:   "text/plain",
		ContentLength: 12,
		LastModified:  "2023-01-01T00:00:00Z",
		Metadata:      map[string]string{"test": "value"},
	}

	if string(output.Body) != "test content" {
		t.Errorf("Expected body 'test content', got '%s'", string(output.Body))
	}
	if output.ContentType != "text/plain" {
		t.Errorf("Expected content type 'text/plain', got '%s'", output.ContentType)
	}
	if output.ContentLength != 12 {
		t.Errorf("Expected content length 12, got %d", output.ContentLength)
	}
}
