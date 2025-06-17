//go:build integration

package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/localstack"
)

const (
	testBucket = "test-bucket"
)

// TestS3ServiceIntegration tests S3Service against localstack
func TestS3ServiceIntegration(t *testing.T) {
	// Start localstack container using dedicated module
	ctx := context.Background()
	localstackContainer, endpoint := startLocalStack(t, ctx)
	defer func() {
		if err := localstackContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate LocalStack container: %v", err)
		}
	}()

	// Create S3 service with localstack endpoint
	s3Service := createTestS3Service(t, ctx, endpoint)

	// Run comprehensive tests
	t.Run("CreateBucketAndTestConnection", func(t *testing.T) {
		testCreateBucketAndConnection(t, ctx, s3Service, endpoint)
	})

	t.Run("UploadFilesAndFolders", func(t *testing.T) {
		testUploadFilesAndFolders(t, ctx, s3Service, endpoint)
	})

	t.Run("ListObjectsWithFolders", func(t *testing.T) {
		testListObjectsWithFolders(t, ctx, s3Service)
	})

	t.Run("NestedFolderStructure", func(t *testing.T) {
		testNestedFolderStructure(t, ctx, s3Service, endpoint)
	})

	t.Run("DeleteOperations", func(t *testing.T) {
		testDeleteOperations(t, ctx, s3Service, endpoint)
	})
}

func startLocalStack(t *testing.T, ctx context.Context) (*localstack.LocalStackContainer, string) {
	t.Helper()

	// Use LocalStack S3-specific lightweight image
	localstackContainer, err := localstack.RunContainer(ctx,
		testcontainers.WithImage("localstack/localstack:s3-latest"),
	)
	if err != nil {
		t.Fatalf("Failed to start LocalStack container: %v", err)
	}

	// Get the endpoint URL
	mappedPort, err := localstackContainer.MappedPort(ctx, "4566/tcp")
	if err != nil {
		t.Fatalf("Failed to get mapped port: %v", err)
	}

	hostIP, err := localstackContainer.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get container host: %v", err)
	}

	endpoint := fmt.Sprintf("http://%s:%s", hostIP, mappedPort.Port())
	t.Logf("LocalStack S3 endpoint: %s", endpoint)

	return localstackContainer, endpoint
}

func createTestS3Service(t *testing.T, ctx context.Context, endpoint string) S3Operations {
	t.Helper()

	// Create a custom S3 service for testing with dummy credentials
	awsConfig, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			"test-access-key-id",
			"test-secret-access-key",
			"test-session-token",
		)),
	)
	if err != nil {
		t.Fatalf("Failed to load AWS config: %v", err)
	}

	// Create S3 client with LocalStack endpoint
	client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	// Create AWSS3Service directly for testing with logger
	return &AWSS3Service{
		client: client,
		config: S3Config{
			Profile:     "",
			Region:      "us-east-1",
			EndpointURL: endpoint,
		},
		logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})),
	}
}

func createDirectS3Client(t *testing.T, ctx context.Context, endpoint string) *s3.Client {
	t.Helper()

	// Use dummy credentials for LocalStack
	awsConfig, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			"test-access-key-id",
			"test-secret-access-key",
			"test-session-token",
		)),
	)
	if err != nil {
		t.Fatalf("Failed to load AWS config: %v", err)
	}

	client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		// Enable path-style addressing for localstack
		o.UsePathStyle = true
	})

	return client
}

func testCreateBucketAndConnection(t *testing.T, ctx context.Context, s3Service S3Operations, endpoint string) {
	// Create bucket using direct S3 client
	client := createDirectS3Client(t, ctx, endpoint)

	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(testBucket),
	})
	if err != nil {
		t.Fatalf("Failed to create bucket: %v", err)
	}

	// Test connection
	err = s3Service.TestConnection(ctx)
	if err != nil {
		t.Errorf("Connection test failed: %v", err)
	}

	// List buckets
	buckets, err := s3Service.ListBuckets(ctx)
	if err != nil {
		t.Errorf("Failed to list buckets: %v", err)
	}

	found := false
	for _, bucket := range buckets {
		if bucket == testBucket {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Test bucket not found in bucket list: %v", buckets)
	}
}

func testUploadFilesAndFolders(t *testing.T, ctx context.Context, s3Service S3Operations, endpoint string) {
	client := createDirectS3Client(t, ctx, endpoint)

	testCases := []struct {
		key     string
		content string
		isFile  bool
	}{
		{"file1.txt", "content of file1", true},
		{"file2.txt", "content of file2", true},
		{"folder1/", "", false}, // Folder marker
		{"folder1/file3.txt", "content of file3", true},
		{"folder1/subfolder/", "", false}, // Nested folder marker
		{"folder1/subfolder/file4.txt", "content of file4", true},
		{"folder2/", "", false}, // Another folder marker
		{"folder2/file5.txt", "content of file5", true},
	}

	for _, tc := range testCases {
		if tc.isFile {
			// Upload file using our service
			_, err := s3Service.UploadObject(ctx, UploadObjectInput{
				Bucket:      testBucket,
				Key:         tc.key,
				Body:        []byte(tc.content),
				ContentType: "text/plain",
			})
			if err != nil {
				t.Errorf("Failed to upload file %s: %v", tc.key, err)
			}
		} else {
			// Create folder marker using direct client
			_, err := client.PutObject(ctx, &s3.PutObjectInput{
				Bucket: aws.String(testBucket),
				Key:    aws.String(tc.key),
				Body:   strings.NewReader(""),
			})
			if err != nil {
				t.Errorf("Failed to create folder marker %s: %v", tc.key, err)
			}
		}
	}

	t.Log("✅ All test files and folders uploaded successfully")
}

func testListObjectsWithFolders(t *testing.T, ctx context.Context, s3Service S3Operations) {
	tests := []struct {
		name      string
		prefix    string
		delimiter string
		expected  map[string]bool // key -> isFolder
	}{
		{
			name:      "Root level with delimiter",
			prefix:    "",
			delimiter: "/",
			expected: map[string]bool{
				"file1.txt": false,
				"file2.txt": false,
				"folder1":   true,
				"folder2":   true,
			},
		},
		{
			name:      "Folder1 contents",
			prefix:    "folder1/",
			delimiter: "/",
			expected: map[string]bool{
				"folder1/file3.txt": false,
				"folder1/subfolder": true,
			},
		},
		{
			name:      "Nested subfolder contents",
			prefix:    "folder1/subfolder/",
			delimiter: "/",
			expected: map[string]bool{
				"folder1/subfolder/file4.txt": false,
			},
		},
		{
			name:      "All objects without delimiter",
			prefix:    "",
			delimiter: "",
			expected: map[string]bool{
				"file1.txt":                   false,
				"file2.txt":                   false,
				"folder1/":                    true, // Folder marker: zero-size object ending with "/"
				"folder1/file3.txt":           false,
				"folder1/subfolder/":          true, // Folder marker: zero-size object ending with "/"
				"folder1/subfolder/file4.txt": false,
				"folder2/":                    true, // Folder marker: zero-size object ending with "/"
				"folder2/file5.txt":           false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := s3Service.ListObjects(ctx, ListObjectsInput{
				Bucket:    testBucket,
				Prefix:    tt.prefix,
				Delimiter: tt.delimiter,
				MaxKeys:   100,
			})
			if err != nil {
				t.Fatalf("Failed to list objects: %v", err)
			}

			t.Logf("📋 Listed %d objects for prefix '%s' with delimiter '%s'",
				len(result.Objects), tt.prefix, tt.delimiter)

			// Check each expected object
			resultMap := make(map[string]bool)
			for _, obj := range result.Objects {
				resultMap[obj.Key] = obj.IsFolder
				t.Logf("  📄 %s (folder: %v, size: %d)", obj.Key, obj.IsFolder, obj.Size)
			}

			for expectedKey, expectedIsFolder := range tt.expected {
				actualIsFolder, found := resultMap[expectedKey]
				if !found {
					t.Errorf("Expected object %s not found in results", expectedKey)
					continue
				}
				if actualIsFolder != expectedIsFolder {
					t.Errorf("Object %s: expected isFolder=%v, got isFolder=%v",
						expectedKey, expectedIsFolder, actualIsFolder)
				}
			}

			// Check for unexpected objects
			for actualKey := range resultMap {
				if _, expected := tt.expected[actualKey]; !expected {
					t.Logf("⚠️  Unexpected object found: %s", actualKey)
				}
			}
		})
	}
}

func testNestedFolderStructure(t *testing.T, ctx context.Context, s3Service S3Operations, endpoint string) {
	client := createDirectS3Client(t, ctx, endpoint)

	// Create a deeper folder structure
	deepStructure := []struct {
		key     string
		content string
		isFile  bool
	}{
		{"deep/", "", false},
		{"deep/level1/", "", false},
		{"deep/level1/level2/", "", false},
		{"deep/level1/level2/level3/", "", false},
		{"deep/level1/level2/level3/deep_file.txt", "very deep content", true},
		{"deep/level1/mid_file.txt", "middle level content", true},
		{"deep/top_file.txt", "top level content", true},
	}

	for _, item := range deepStructure {
		if item.isFile {
			_, err := s3Service.UploadObject(ctx, UploadObjectInput{
				Bucket:      testBucket,
				Key:         item.key,
				Body:        []byte(item.content),
				ContentType: "text/plain",
			})
			if err != nil {
				t.Errorf("Failed to upload deep file %s: %v", item.key, err)
			}
		} else {
			_, err := client.PutObject(ctx, &s3.PutObjectInput{
				Bucket: aws.String(testBucket),
				Key:    aws.String(item.key),
				Body:   strings.NewReader(""),
			})
			if err != nil {
				t.Errorf("Failed to create deep folder %s: %v", item.key, err)
			}
		}
	}

	// Test navigation through deep structure
	testLevels := []struct {
		prefix   string
		expected []string
	}{
		{
			prefix:   "deep/",
			expected: []string{"deep/top_file.txt", "deep/level1"},
		},
		{
			prefix:   "deep/level1/",
			expected: []string{"deep/level1/mid_file.txt", "deep/level1/level2"},
		},
		{
			prefix:   "deep/level1/level2/",
			expected: []string{"deep/level1/level2/level3"},
		},
		{
			prefix:   "deep/level1/level2/level3/",
			expected: []string{"deep/level1/level2/level3/deep_file.txt"},
		},
	}

	for _, level := range testLevels {
		t.Run(fmt.Sprintf("Level_%s", level.prefix), func(t *testing.T) {
			result, err := s3Service.ListObjects(ctx, ListObjectsInput{
				Bucket:    testBucket,
				Prefix:    level.prefix,
				Delimiter: "/",
				MaxKeys:   100,
			})
			if err != nil {
				t.Fatalf("Failed to list objects for prefix %s: %v", level.prefix, err)
			}

			found := make(map[string]bool)
			for _, obj := range result.Objects {
				found[obj.Key] = true
				t.Logf("📁 Found: %s (folder: %v)", obj.Key, obj.IsFolder)
			}

			for _, expectedKey := range level.expected {
				if !found[expectedKey] {
					t.Errorf("Expected to find %s in prefix %s", expectedKey, level.prefix)
				}
			}
		})
	}
}

func testDeleteOperations(t *testing.T, ctx context.Context, s3Service S3Operations, endpoint string) {
	// Upload test files for deletion
	testFiles := []string{"delete_me1.txt", "delete_me2.txt", "keep_me.txt"}

	for _, filename := range testFiles {
		_, err := s3Service.UploadObject(ctx, UploadObjectInput{
			Bucket:      testBucket,
			Key:         filename,
			Body:        []byte("test content"),
			ContentType: "text/plain",
		})
		if err != nil {
			t.Fatalf("Failed to upload test file %s: %v", filename, err)
		}
	}

	// Test single delete
	err := s3Service.DeleteObject(ctx, testBucket, "delete_me1.txt")
	if err != nil {
		t.Errorf("Failed to delete single object: %v", err)
	}

	// Test batch delete
	err = s3Service.DeleteObjects(ctx, testBucket, []string{"delete_me2.txt"})
	if err != nil {
		t.Errorf("Failed to delete objects in batch: %v", err)
	}

	// Verify deletions
	result, err := s3Service.ListObjects(ctx, ListObjectsInput{
		Bucket:    testBucket,
		Prefix:    "delete_me",
		Delimiter: "",
		MaxKeys:   100,
	})
	if err != nil {
		t.Fatalf("Failed to list objects after deletion: %v", err)
	}

	if len(result.Objects) > 0 {
		t.Errorf("Expected no objects with prefix 'delete_me', but found %d", len(result.Objects))
		for _, obj := range result.Objects {
			t.Logf("  Remaining: %s", obj.Key)
		}
	}

	// Verify keep_me.txt still exists
	result, err = s3Service.ListObjects(ctx, ListObjectsInput{
		Bucket:    testBucket,
		Prefix:    "keep_me",
		Delimiter: "",
		MaxKeys:   100,
	})
	if err != nil {
		t.Fatalf("Failed to list kept objects: %v", err)
	}

	if len(result.Objects) != 1 || result.Objects[0].Key != "keep_me.txt" {
		t.Errorf("Expected to find keep_me.txt, but got %v", result.Objects)
	}
}
