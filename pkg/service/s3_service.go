package service

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	s3cerrors "github.com/tenkoh/s3c/pkg/errors"
)

// S3Config holds the configuration for S3 connection
type S3Config struct {
	Profile     string `json:"profile"`
	EndpointURL string `json:"endpointUrl"`
	Region      string `json:"region"`
}

// AWSS3Service implements S3Operations using AWS SDK
type AWSS3Service struct {
	client *s3.Client
	config S3Config
	logger *slog.Logger
}

// S3Object represents an S3 object with metadata
type S3Object struct {
	Key          string `json:"key"`
	Size         int64  `json:"size"`
	LastModified string `json:"lastModified"`
	IsFolder     bool   `json:"isFolder"`
}

// ListObjectsInput represents input for listing objects
type ListObjectsInput struct {
	Bucket            string `json:"bucket"`
	Prefix            string `json:"prefix,omitempty"`
	Delimiter         string `json:"delimiter,omitempty"`
	MaxKeys           int32  `json:"maxKeys,omitempty"`
	ContinuationToken string `json:"continuationToken,omitempty"`
}

// ListObjectsOutput represents output from listing objects
type ListObjectsOutput struct {
	Objects               []S3Object `json:"objects"`
	CommonPrefixes        []string   `json:"commonPrefixes"`
	IsTruncated           bool       `json:"isTruncated"`
	NextContinuationToken string     `json:"nextContinuationToken,omitempty"`
}

// S3ConnectionTester interface for testing S3 connectivity
type S3ConnectionTester interface {
	TestConnection(ctx context.Context) error
}

// S3BucketLister interface for bucket operations
type S3BucketLister interface {
	ListBuckets(ctx context.Context) ([]string, error)
}

// S3BucketCreator interface for bucket creation operations
type S3BucketCreator interface {
	CreateBucket(ctx context.Context, bucketName string) error
}

// S3ObjectReader interface for read-only object operations
type S3ObjectReader interface {
	ListObjects(ctx context.Context, input ListObjectsInput) (*ListObjectsOutput, error)
}

// S3ObjectDeleter interface for object deletion operations
type S3ObjectDeleter interface {
	DeleteObject(ctx context.Context, bucket, key string) error
	DeleteObjects(ctx context.Context, bucket string, keys []string) error
}

// UploadObjectInput represents input for uploading objects
type UploadObjectInput struct {
	Bucket      string            `json:"bucket"`
	Key         string            `json:"key"`
	Body        []byte            `json:"-"` // Don't serialize body in JSON
	ContentType string            `json:"contentType,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// UploadObjectOutput represents output from uploading objects
type UploadObjectOutput struct {
	Key  string `json:"key"`
	ETag string `json:"etag"`
}

// DownloadObjectInput represents input for downloading objects
type DownloadObjectInput struct {
	Bucket string `json:"bucket"`
	Key    string `json:"key"`
}

// DownloadObjectOutput represents output from downloading objects
type DownloadObjectOutput struct {
	Body          []byte            `json:"-"` // Don't serialize body in JSON
	ContentType   string            `json:"contentType"`
	ContentLength int64             `json:"contentLength"`
	LastModified  string            `json:"lastModified"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// S3ObjectUploader interface for object upload operations
type S3ObjectUploader interface {
	UploadObject(ctx context.Context, input UploadObjectInput) (*UploadObjectOutput, error)
}

// S3FolderCreator interface for folder creation operations
type S3FolderCreator interface {
	CreateFolder(ctx context.Context, bucket, prefix string) error
}

// S3ObjectDownloader interface for object download operations
type S3ObjectDownloader interface {
	DownloadObject(ctx context.Context, input DownloadObjectInput) (*DownloadObjectOutput, error)
}

// S3Operations combines all S3 operation interfaces
type S3Operations interface {
	S3ConnectionTester
	S3BucketLister
	S3BucketCreator
	S3ObjectReader
	S3ObjectDeleter
	S3ObjectUploader
	S3ObjectDownloader
	S3FolderCreator
}

// NewS3Service creates a new S3Service with the given configuration
func NewS3Service(ctx context.Context, cfg S3Config) (S3Operations, error) {
	return NewS3ServiceWithLogger(ctx, cfg, slog.Default())
}

// NewS3ServiceWithLogger creates a new S3Service with the given configuration and logger
func NewS3ServiceWithLogger(ctx context.Context, cfg S3Config, logger *slog.Logger) (S3Operations, error) {
	serviceLogger := logger.With("component", "s3service")

	serviceLogger.Debug("Creating S3 service",
		"profile", cfg.Profile,
		"region", cfg.Region,
		"hasEndpoint", cfg.EndpointURL != "",
	)

	// Build AWS config options
	var options []func(*config.LoadOptions) error

	// Set profile if specified
	if cfg.Profile != "" {
		options = append(options, config.WithSharedConfigProfile(cfg.Profile))
	}

	// Set region if specified
	if cfg.Region != "" {
		options = append(options, config.WithRegion(cfg.Region))
	}

	// Load AWS configuration
	awsConfig, err := config.LoadDefaultConfig(ctx, options...)
	if err != nil {
		serviceLogger.Error("Failed to load AWS configuration", "error", err)
		return nil, s3cerrors.NewCredentialsInvalidError(err)
	}

	// Create S3 client options
	var s3Options []func(*s3.Options)

	// Set custom endpoint if specified (for S3-compatible services)
	if cfg.EndpointURL != "" {
		serviceLogger.Debug("Using custom S3 endpoint", "endpoint", cfg.EndpointURL)
		s3Options = append(s3Options, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.EndpointURL)
			// Enable path-style addressing for localstack and other S3-compatible services
			if strings.Contains(cfg.EndpointURL, "localstack") || strings.Contains(cfg.EndpointURL, "localhost") {
				o.UsePathStyle = true
			}
		})
	}

	// Add option to disable checksum validation warning logs
	s3Options = append(s3Options, func(o *s3.Options) {
		o.DisableLogOutputChecksumValidationSkipped = true
	})

	// Create S3 client
	client := s3.NewFromConfig(awsConfig, s3Options...)

	serviceLogger.Info("S3 service created successfully",
		"profile", cfg.Profile,
		"region", cfg.Region,
	)

	return &AWSS3Service{
		client: client,
		config: cfg,
		logger: serviceLogger,
	}, nil
}

// ListBuckets returns a list of all buckets
func (s *AWSS3Service) ListBuckets(ctx context.Context) ([]string, error) {
	s.logger.Debug("Listing S3 buckets")

	result, err := s.client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		s.logger.Error("Failed to list S3 buckets", "error", err)
		return nil, convertS3Error("list buckets", err)
	}

	buckets := make([]string, len(result.Buckets))
	for i, bucket := range result.Buckets {
		if bucket.Name != nil {
			buckets[i] = *bucket.Name
		}
	}

	s.logger.Debug("Successfully listed S3 buckets", "bucketCount", len(buckets))
	return buckets, nil
}

// CreateBucket creates a new S3 bucket
func (s *AWSS3Service) CreateBucket(ctx context.Context, bucketName string) error {
	s.logger.Debug("Creating S3 bucket", "bucketName", bucketName, "region", s.config.Region)

	// Prepare CreateBucket input
	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}

	// AWS S3 requires LocationConstraint for all regions except us-east-1
	// Reference: https://docs.aws.amazon.com/AmazonS3/latest/API/API_CreateBucket.html
	if s.config.Region != "us-east-1" && s.config.Region != "" {
		input.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(s.config.Region),
		}
		s.logger.Debug("Setting LocationConstraint for bucket creation",
			"bucketName", bucketName,
			"locationConstraint", s.config.Region)
	}

	_, err := s.client.CreateBucket(ctx, input)
	if err != nil {
		s.logger.Error("Failed to create S3 bucket", "error", err, "bucketName", bucketName, "region", s.config.Region)
		return convertS3Error("create bucket", err).(*s3cerrors.S3CError).
			WithDetails(map[string]interface{}{
				"bucket": bucketName,
				"region": s.config.Region,
			})
	}

	s.logger.Info("Successfully created S3 bucket", "bucketName", bucketName, "region", s.config.Region)
	return nil
}

// TestConnection verifies that the S3 service is accessible
func (s *AWSS3Service) TestConnection(ctx context.Context) error {
	s.logger.Debug("Testing S3 connection")

	_, err := s.client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		s.logger.Error("S3 connection test failed", "error", err)
		return convertS3Error("test connection", err)
	}

	s.logger.Debug("S3 connection test successful")
	return nil
}

// ListObjects lists objects in a bucket with optional prefix and pagination
func (s *AWSS3Service) ListObjects(ctx context.Context, input ListObjectsInput) (*ListObjectsOutput, error) {
	s.logger.Debug("Listing S3 objects",
		"bucket", input.Bucket,
		"prefix", input.Prefix,
		"delimiter", input.Delimiter,
		"maxKeys", input.MaxKeys,
		"hasContinuationToken", input.ContinuationToken != "",
	)

	// Set default values
	maxKeys := input.MaxKeys
	if maxKeys == 0 {
		maxKeys = 100 // Default page size as per design doc
	}

	delimiter := input.Delimiter
	if delimiter == "" && input.Prefix != "" {
		delimiter = "/" // Default delimiter for folder-like browsing
	}

	// Prepare S3 input
	s3Input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(input.Bucket),
		MaxKeys:   aws.Int32(maxKeys),
		Delimiter: aws.String(delimiter),
	}

	if input.Prefix != "" {
		s3Input.Prefix = aws.String(input.Prefix)
	}

	if input.ContinuationToken != "" {
		s3Input.ContinuationToken = aws.String(input.ContinuationToken)
	}

	// Call S3
	result, err := s.client.ListObjectsV2(ctx, s3Input)
	if err != nil {
		s.logger.Error("Failed to list S3 objects",
			"error", err,
			"bucket", input.Bucket,
			"prefix", input.Prefix,
		)
		return nil, convertS3Error("list objects", err).(*s3cerrors.S3CError).
			WithDetails(map[string]interface{}{
				"bucket": input.Bucket,
				"prefix": input.Prefix,
			})
	}

	// Process results
	output := &ListObjectsOutput{
		Objects:        make([]S3Object, 0, len(result.Contents)),
		CommonPrefixes: make([]string, 0, len(result.CommonPrefixes)),
		IsTruncated:    aws.ToBool(result.IsTruncated),
	}

	if result.NextContinuationToken != nil {
		output.NextContinuationToken = *result.NextContinuationToken
	}

	// Track common prefixes to avoid duplication
	commonPrefixMap := make(map[string]bool)

	// Convert common prefixes (folders) first
	for _, prefix := range result.CommonPrefixes {
		if prefix.Prefix != nil {
			// Keep the full prefix including trailing slash for internal use
			fullPrefix := *prefix.Prefix
			output.CommonPrefixes = append(output.CommonPrefixes, fullPrefix)

			// Track this prefix to avoid duplication
			commonPrefixMap[fullPrefix] = true

			// For UI display, use the folder name without trailing slash
			// but keep the full path structure
			folderKey := strings.TrimSuffix(fullPrefix, "/")

			// Add folder as an object for UI consistency
			folderObj := S3Object{
				Key:      folderKey,
				Size:     0,
				IsFolder: true,
			}
			output.Objects = append(output.Objects, folderObj)
		}
	}

	// Convert objects, skipping folder markers that are already in common prefixes
	for _, obj := range result.Contents {
		if obj.Key == nil {
			continue
		}

		key := *obj.Key
		size := aws.ToInt64(obj.Size)

		// Check if this is a folder marker (empty object ending with /)
		isFolder := size == 0 && strings.HasSuffix(key, "/")

		// Skip folder markers that are already represented in common prefixes
		if isFolder && commonPrefixMap[key] {
			continue
		}

		// When using delimiter, skip folder markers for intermediate levels
		// (e.g., skip "folder1/" when listing "folder1/" contents)
		if isFolder && delimiter != "" && input.Prefix != "" && key == input.Prefix {
			continue
		}

		s3Obj := S3Object{
			Key:      key,
			Size:     size,
			IsFolder: isFolder,
		}

		if obj.LastModified != nil {
			s3Obj.LastModified = obj.LastModified.Format(time.RFC3339)
		}

		output.Objects = append(output.Objects, s3Obj)
	}

	s.logger.Debug("Successfully listed S3 objects",
		"bucket", input.Bucket,
		"objectCount", len(output.Objects),
		"commonPrefixCount", len(output.CommonPrefixes),
		"isTruncated", output.IsTruncated,
	)

	return output, nil
}

// DeleteObject deletes a single object from S3
func (s *AWSS3Service) DeleteObject(ctx context.Context, bucket, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return convertS3Error("delete object", err).(*s3cerrors.S3CError).
			WithDetails(map[string]interface{}{
				"bucket": bucket,
				"key":    key,
			})
	}
	return nil
}

// DeleteObjects deletes multiple objects from S3 in a batch operation
func (s *AWSS3Service) DeleteObjects(ctx context.Context, bucket string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	// Convert keys to ObjectIdentifier slice
	objects := make([]types.ObjectIdentifier, len(keys))
	for i, key := range keys {
		objects[i] = types.ObjectIdentifier{
			Key: aws.String(key),
		}
	}

	// Perform batch delete
	_, err := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(bucket),
		Delete: &types.Delete{
			Objects: objects,
			Quiet:   aws.Bool(false), // Return results for error handling
		},
	})
	if err != nil {
		return convertS3Error("delete objects", err).(*s3cerrors.S3CError).
			WithDetails(map[string]interface{}{
				"bucket": bucket,
				"keys":   keys,
				"count":  len(keys),
			})
	}

	return nil
}

// UploadObject uploads an object to S3
func (s *AWSS3Service) UploadObject(ctx context.Context, input UploadObjectInput) (*UploadObjectOutput, error) {
	// Prepare S3 input
	s3Input := &s3.PutObjectInput{
		Bucket: aws.String(input.Bucket),
		Key:    aws.String(input.Key),
		Body:   bytes.NewReader(input.Body),
	}

	if input.ContentType != "" {
		s3Input.ContentType = aws.String(input.ContentType)
	}

	if len(input.Metadata) > 0 {
		s3Input.Metadata = input.Metadata
	}

	// Upload to S3
	result, err := s.client.PutObject(ctx, s3Input)
	if err != nil {
		return nil, convertS3Error("upload object", err).(*s3cerrors.S3CError).
			WithDetails(map[string]interface{}{
				"bucket": input.Bucket,
				"key":    input.Key,
				"size":   len(input.Body),
			})
	}

	output := &UploadObjectOutput{
		Key: input.Key,
	}
	if result.ETag != nil {
		output.ETag = *result.ETag
	}

	return output, nil
}

// DownloadObject downloads an object from S3
func (s *AWSS3Service) DownloadObject(ctx context.Context, input DownloadObjectInput) (*DownloadObjectOutput, error) {
	// Get object from S3
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(input.Bucket),
		Key:    aws.String(input.Key),
	})
	if err != nil {
		return nil, convertS3Error("download object", err).(*s3cerrors.S3CError).
			WithDetails(map[string]interface{}{
				"bucket": input.Bucket,
				"key":    input.Key,
			})
	}
	defer result.Body.Close()

	// Read the body
	body, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, s3cerrors.NewFileOperationError("read", "S3 object body", err).
			WithDetails(map[string]interface{}{
				"bucket": input.Bucket,
				"key":    input.Key,
			})
	}

	output := &DownloadObjectOutput{
		Body:          body,
		ContentLength: aws.ToInt64(result.ContentLength),
		Metadata:      result.Metadata,
	}

	if result.ContentType != nil {
		output.ContentType = *result.ContentType
	}

	if result.LastModified != nil {
		output.LastModified = result.LastModified.Format(time.RFC3339)
	}

	return output, nil
}

// CreateFolder creates a folder marker in S3 by uploading a zero-byte object with a trailing slash
func (s *AWSS3Service) CreateFolder(ctx context.Context, bucket, prefix string) error {
	s.logger.Debug("Creating S3 folder",
		"bucket", bucket,
		"prefix", prefix,
	)

	// Ensure the prefix ends with a slash to create a proper folder marker
	folderKey := prefix
	if !strings.HasSuffix(folderKey, "/") {
		folderKey += "/"
	}

	// Create a zero-byte object with folder marker
	uploadInput := UploadObjectInput{
		Bucket:      bucket,
		Key:         folderKey,
		Body:        []byte{}, // Empty content for folder marker
		ContentType: "application/x-directory",
		Metadata: map[string]string{
			"folder-marker": "true",
		},
	}

	_, err := s.UploadObject(ctx, uploadInput)
	if err != nil {
		s.logger.Error("Failed to create S3 folder",
			"error", err,
			"bucket", bucket,
			"prefix", prefix,
			"folderKey", folderKey,
		)
		// UploadObject already returns structured errors
		return err
	}

	s.logger.Info("Successfully created S3 folder",
		"bucket", bucket,
		"folderKey", folderKey,
	)
	return nil
}

// convertS3Error converts AWS S3 SDK errors to structured s3c errors
func convertS3Error(operation string, err error) error {
	if err == nil {
		return nil
	}

	// Check for specific AWS SDK error types by examining error messages
	// AWS SDK v2 doesn't expose the structured error types in the same way as v1
	errMsg := err.Error()

	switch {
	case strings.Contains(errMsg, "NoSuchBucket"):
		// Extract bucket name from error message if possible
		return s3cerrors.NewS3BucketNotFoundError("").WithWrapped(err)

	case strings.Contains(errMsg, "NoSuchKey"):
		// Extract key name from error message if possible
		return s3cerrors.NewS3ObjectNotFoundError("", "").WithWrapped(err)

	case strings.Contains(errMsg, "NotFound"):
		return s3cerrors.NewS3Error(s3cerrors.CodeS3ObjectNotFound, "Resource not found").WithWrapped(err)

	case strings.Contains(errMsg, "NoCredentialsProvided") || strings.Contains(errMsg, "InvalidAccessKeyId"):
		return s3cerrors.NewCredentialsInvalidError(err)

	case strings.Contains(errMsg, "SignatureDoesNotMatch"):
		return s3cerrors.NewCredentialsInvalidError(err).
			WithSuggestion("Check your AWS secret access key")

	case strings.Contains(errMsg, "AccessDenied") || strings.Contains(errMsg, "Forbidden"):
		return s3cerrors.NewS3AccessDeniedError(operation, "S3 resource").WithWrapped(err)

	case strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "Timeout"):
		return s3cerrors.NewNetworkTimeoutError(operation).WithWrapped(err)

	case strings.Contains(errMsg, "connection") || strings.Contains(errMsg, "network"):
		return s3cerrors.NewS3ConnectionError(err)

	case strings.Contains(errMsg, "RequestLimitExceeded") || strings.Contains(errMsg, "SlowDown"):
		return s3cerrors.NewS3Error(s3cerrors.CodeS3QuotaExceeded, "S3 request rate limit exceeded").
			WithWrapped(err).
			WithSuggestion("Please wait a moment and try again")

	default:
		// Generic S3 operation error
		return s3cerrors.NewS3OperationError(operation, err)
	}
}
