package service

import (
	"testing"
)

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
