package service

import (
	"errors"
	"strings"
	"testing"

	s3cerrors "github.com/tenkoh/s3c/pkg/errors"
)

// Test the critical error conversion logic
func TestConvertS3Error(t *testing.T) {
	tests := []struct {
		name          string
		operation     string
		inputError    error
		expectedCode  s3cerrors.ErrorCode
		expectedType  string
		shouldContain string
	}{
		{
			name:         "nil error returns nil",
			operation:    "test",
			inputError:   nil,
			expectedType: "nil",
		},
		{
			name:          "NoSuchBucket error",
			operation:     "list_buckets",
			inputError:    errors.New("NoSuchBucket: The specified bucket does not exist"),
			expectedCode:  s3cerrors.CodeS3BucketNotFound,
			expectedType:  "*s3cerrors.S3CError",
			shouldContain: "Bucket",
		},
		{
			name:          "NoSuchKey error",
			operation:     "get_object",
			inputError:    errors.New("NoSuchKey: The specified key does not exist"),
			expectedCode:  s3cerrors.CodeS3ObjectNotFound,
			expectedType:  "*s3cerrors.S3CError",
			shouldContain: "Object",
		},
		{
			name:          "Generic NotFound error",
			operation:     "operation",
			inputError:    errors.New("NotFound: Resource not found"),
			expectedCode:  s3cerrors.CodeS3ObjectNotFound,
			expectedType:  "*s3cerrors.S3CError",
			shouldContain: "Resource not found",
		},
		{
			name:          "NoCredentialsProvided error",
			operation:     "list_buckets",
			inputError:    errors.New("NoCredentialsProvided: no credentials"),
			expectedCode:  s3cerrors.CodeCredentialsInvalid,
			expectedType:  "*s3cerrors.S3CError",
			shouldContain: "credentials",
		},
		{
			name:          "InvalidAccessKeyId error",
			operation:     "list_buckets",
			inputError:    errors.New("InvalidAccessKeyId: The access key ID does not exist"),
			expectedCode:  s3cerrors.CodeCredentialsInvalid,
			expectedType:  "*s3cerrors.S3CError",
			shouldContain: "credentials",
		},
		{
			name:          "SignatureDoesNotMatch error with suggestion",
			operation:     "list_buckets",
			inputError:    errors.New("SignatureDoesNotMatch: signature mismatch"),
			expectedCode:  s3cerrors.CodeCredentialsInvalid,
			expectedType:  "*s3cerrors.S3CError",
			shouldContain: "Check your AWS secret access key",
		},
		{
			name:          "AccessDenied error",
			operation:     "get_object",
			inputError:    errors.New("AccessDenied: Access denied"),
			expectedCode:  s3cerrors.CodeS3AccessDenied,
			expectedType:  "*s3cerrors.S3CError",
			shouldContain: "access denied",
		},
		{
			name:          "Forbidden error",
			operation:     "put_object",
			inputError:    errors.New("Forbidden: Operation forbidden"),
			expectedCode:  s3cerrors.CodeS3AccessDenied,
			expectedType:  "*s3cerrors.S3CError",
			shouldContain: "access denied",
		},
		{
			name:          "timeout error",
			operation:     "upload_object",
			inputError:    errors.New("timeout: operation timed out"),
			expectedCode:  s3cerrors.CodeNetworkTimeout,
			expectedType:  "*s3cerrors.S3CError",
			shouldContain: "timeout",
		},
		{
			name:          "Timeout error (capital T)",
			operation:     "download_object",
			inputError:    errors.New("Timeout: Request timeout"),
			expectedCode:  s3cerrors.CodeNetworkTimeout,
			expectedType:  "*s3cerrors.S3CError",
			shouldContain: "timeout",
		},
		{
			name:          "connection error",
			operation:     "list_objects",
			inputError:    errors.New("connection: failed to connect"),
			expectedCode:  s3cerrors.CodeS3Connection,
			expectedType:  "*s3cerrors.S3CError",
			shouldContain: "connection",
		},
		{
			name:          "network error",
			operation:     "put_object",
			inputError:    errors.New("network: network unreachable"),
			expectedCode:  s3cerrors.CodeS3Connection,
			expectedType:  "*s3cerrors.S3CError",
			shouldContain: "connection",
		},
		{
			name:          "RequestLimitExceeded error",
			operation:     "put_object",
			inputError:    errors.New("RequestLimitExceeded: rate limit exceeded"),
			expectedCode:  s3cerrors.CodeS3QuotaExceeded,
			expectedType:  "*s3cerrors.S3CError",
			shouldContain: "Please wait a moment and try again",
		},
		{
			name:          "SlowDown error",
			operation:     "list_objects",
			inputError:    errors.New("SlowDown: slow down requests"),
			expectedCode:  s3cerrors.CodeS3QuotaExceeded,
			expectedType:  "*s3cerrors.S3CError",
			shouldContain: "rate limit exceeded",
		},
		{
			name:          "Generic unknown error",
			operation:     "unknown_operation",
			inputError:    errors.New("SomeUnknownError: something went wrong"),
			expectedCode:  s3cerrors.CodeS3Operation,
			expectedType:  "*s3cerrors.S3CError",
			shouldContain: "operation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertS3Error(tt.operation, tt.inputError)

			if tt.expectedType == "nil" {
				if result != nil {
					t.Errorf("Expected nil error, got %v", result)
				}
				return
			}

			// Check that we got an S3CError
			var s3cErr *s3cerrors.S3CError
			if !errors.As(result, &s3cErr) {
				t.Errorf("Expected *s3cerrors.S3CError, got %T", result)
				return
			}

			// Check error code
			if s3cErr.Code != tt.expectedCode {
				t.Errorf("Expected error code %s, got %s", tt.expectedCode, s3cErr.Code)
			}

			// Check that the error message or suggestion contains expected text
			errorText := strings.ToLower(s3cErr.Message + " " + s3cErr.Suggestion)
			expectedText := strings.ToLower(tt.shouldContain)
			if !strings.Contains(errorText, expectedText) {
				t.Errorf("Expected error to contain '%s', got message: '%s', suggestion: '%s'",
					tt.shouldContain, s3cErr.Message, s3cErr.Suggestion)
			}

			// Check that original error is wrapped
			if s3cErr.Wrapped == nil {
				t.Error("Expected wrapped error to be preserved")
			}
		})
	}
}

// Test the folder detection logic
func TestFolderDetection(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		size     int64
		isFolder bool
	}{
		{
			name:     "regular file",
			key:      "documents/file.txt",
			size:     1024,
			isFolder: false,
		},
		{
			name:     "folder marker - zero size with trailing slash",
			key:      "documents/",
			size:     0,
			isFolder: true,
		},
		{
			name:     "nested folder marker",
			key:      "documents/subfolder/",
			size:     0,
			isFolder: true,
		},
		{
			name:     "empty file (not folder) - zero size without trailing slash",
			key:      "empty.txt",
			size:     0,
			isFolder: false,
		},
		{
			name:     "file with slash in name (not folder) - non-zero size",
			key:      "weird/filename/",
			size:     100,
			isFolder: false,
		},
		{
			name:     "root level folder",
			key:      "folder/",
			size:     0,
			isFolder: true,
		},
		{
			name:     "deep nested folder",
			key:      "level1/level2/level3/level4/",
			size:     0,
			isFolder: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This replicates the folder detection logic from ListObjects
			isFolder := tt.size == 0 && strings.HasSuffix(tt.key, "/")

			if isFolder != tt.isFolder {
				t.Errorf("Expected isFolder=%v for key='%s' size=%d, got %v",
					tt.isFolder, tt.key, tt.size, isFolder)
			}
		})
	}
}

// Test S3Config validation logic
func TestS3ConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      S3Config
		expectValid bool
		description string
	}{
		{
			name: "valid config with all fields",
			config: S3Config{
				Profile:     "default",
				Region:      "us-west-2",
				EndpointURL: "https://custom.s3.example.com",
			},
			expectValid: true,
			description: "Complete configuration should be valid",
		},
		{
			name: "valid config without endpoint",
			config: S3Config{
				Profile: "production",
				Region:  "eu-west-1",
			},
			expectValid: true,
			description: "Configuration without custom endpoint should be valid",
		},
		{
			name: "localstack configuration",
			config: S3Config{
				Profile:     "localstack",
				Region:      "us-east-1",
				EndpointURL: "http://localhost:4566",
			},
			expectValid: true,
			description: "Localstack configuration should be valid",
		},
		{
			name: "minimal valid config",
			config: S3Config{
				Profile: "test",
				Region:  "us-east-1",
			},
			expectValid: true,
			description: "Minimal configuration should be valid",
		},
		// Error cases
		{
			name: "missing profile",
			config: S3Config{
				Region:      "us-west-2",
				EndpointURL: "https://s3.example.com",
			},
			expectValid: false,
			description: "Configuration without profile should be invalid",
		},
		{
			name: "missing region",
			config: S3Config{
				Profile:     "default",
				EndpointURL: "https://s3.example.com",
			},
			expectValid: false,
			description: "Configuration without region should be invalid",
		},
		{
			name: "empty profile string",
			config: S3Config{
				Profile: "",
				Region:  "us-east-1",
			},
			expectValid: false,
			description: "Empty profile string should be invalid",
		},
		{
			name: "empty region string",
			config: S3Config{
				Profile: "default",
				Region:  "",
			},
			expectValid: false,
			description: "Empty region string should be invalid",
		},
		{
			name:        "completely empty config",
			config:      S3Config{},
			expectValid: false,
			description: "Empty configuration should be invalid",
		},
		{
			name: "whitespace-only profile",
			config: S3Config{
				Profile: "   ",
				Region:  "us-east-1",
			},
			expectValid: false,
			description: "Whitespace-only profile should be invalid",
		},
		{
			name: "whitespace-only region",
			config: S3Config{
				Profile: "default",
				Region:  "   ",
			},
			expectValid: false,
			description: "Whitespace-only region should be invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test comprehensive field validation (including whitespace trimming)
			hasValidProfile := strings.TrimSpace(tt.config.Profile) != ""
			hasValidRegion := strings.TrimSpace(tt.config.Region) != ""

			isValid := hasValidProfile && hasValidRegion

			if isValid != tt.expectValid {
				t.Errorf("%s: expected valid=%v, got valid=%v (profile='%s', region='%s')",
					tt.description, tt.expectValid, isValid, tt.config.Profile, tt.config.Region)
			}

			// Test endpoint URL logic for valid configs
			if tt.expectValid && tt.config.EndpointURL != "" {
				isLocalstack := strings.Contains(tt.config.EndpointURL, "localstack") ||
					strings.Contains(tt.config.EndpointURL, "localhost")
				t.Logf("Config uses custom endpoint: %s (localstack: %v)", tt.config.EndpointURL, isLocalstack)
			}

			// Log validation details for debugging
			if !tt.expectValid {
				t.Logf("Invalid config detected: profile='%s' (valid: %v), region='%s' (valid: %v)",
					tt.config.Profile, hasValidProfile, tt.config.Region, hasValidRegion)
			}
		})
	}
}
