package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Config holds the configuration for S3 connection
type S3Config struct {
	Profile     string `json:"profile"`
	EndpointURL string `json:"endpoint_url"`
	Region      string `json:"region"`
}

// AWSS3Service implements S3Operations using AWS SDK
type AWSS3Service struct {
	client *s3.Client
	config S3Config
}

// S3Object represents an S3 object with metadata
type S3Object struct {
	Key          string `json:"key"`
	Size         int64  `json:"size"`
	LastModified string `json:"last_modified"`
	IsFolder     bool   `json:"is_folder"`
}

// ListObjectsInput represents input for listing objects
type ListObjectsInput struct {
	Bucket            string `json:"bucket"`
	Prefix            string `json:"prefix,omitempty"`
	Delimiter         string `json:"delimiter,omitempty"`
	MaxKeys           int32  `json:"max_keys,omitempty"`
	ContinuationToken string `json:"continuation_token,omitempty"`
}

// ListObjectsOutput represents output from listing objects
type ListObjectsOutput struct {
	Objects               []S3Object `json:"objects"`
	CommonPrefixes        []string   `json:"common_prefixes"`
	IsTruncated           bool       `json:"is_truncated"`
	NextContinuationToken string     `json:"next_continuation_token,omitempty"`
}

// S3ConnectionTester interface for testing S3 connectivity
type S3ConnectionTester interface {
	TestConnection(ctx context.Context) error
}

// S3BucketLister interface for bucket operations
type S3BucketLister interface {
	ListBuckets(ctx context.Context) ([]string, error)
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
	ContentType string            `json:"content_type,omitempty"`
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
	ContentType   string            `json:"content_type"`
	ContentLength int64             `json:"content_length"`
	LastModified  string            `json:"last_modified"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// S3ObjectUploader interface for object upload operations
type S3ObjectUploader interface {
	UploadObject(ctx context.Context, input UploadObjectInput) (*UploadObjectOutput, error)
}

// S3ObjectDownloader interface for object download operations
type S3ObjectDownloader interface {
	DownloadObject(ctx context.Context, input DownloadObjectInput) (*DownloadObjectOutput, error)
}

// S3Operations combines all S3 operation interfaces
type S3Operations interface {
	S3ConnectionTester
	S3BucketLister
	S3ObjectReader
	S3ObjectDeleter
	S3ObjectUploader
	S3ObjectDownloader
}

// NewS3Service creates a new S3Service with the given configuration
func NewS3Service(ctx context.Context, cfg S3Config) (S3Operations, error) {
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
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client options
	var s3Options []func(*s3.Options)

	// Set custom endpoint if specified (for S3-compatible services)
	if cfg.EndpointURL != "" {
		s3Options = append(s3Options, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.EndpointURL)
		})
	}

	// Create S3 client
	client := s3.NewFromConfig(awsConfig, s3Options...)

	return &AWSS3Service{
		client: client,
		config: cfg,
	}, nil
}

// ListBuckets returns a list of all buckets
func (s *AWSS3Service) ListBuckets(ctx context.Context) ([]string, error) {
	result, err := s.client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	buckets := make([]string, len(result.Buckets))
	for i, bucket := range result.Buckets {
		if bucket.Name != nil {
			buckets[i] = *bucket.Name
		}
	}

	return buckets, nil
}

// TestConnection verifies that the S3 service is accessible
func (s *AWSS3Service) TestConnection(ctx context.Context) error {
	_, err := s.client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return fmt.Errorf("failed to connect to S3: %w", err)
	}
	return nil
}

// ListObjects lists objects in a bucket with optional prefix and pagination
func (s *AWSS3Service) ListObjects(ctx context.Context, input ListObjectsInput) (*ListObjectsOutput, error) {
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
		return nil, fmt.Errorf("failed to list objects in bucket %s: %w", input.Bucket, err)
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

	// Convert objects
	for _, obj := range result.Contents {
		if obj.Key == nil {
			continue
		}

		s3Obj := S3Object{
			Key:      *obj.Key,
			Size:     aws.ToInt64(obj.Size),
			IsFolder: false,
		}

		if obj.LastModified != nil {
			s3Obj.LastModified = obj.LastModified.Format(time.RFC3339)
		}

		output.Objects = append(output.Objects, s3Obj)
	}

	// Convert common prefixes (folders)
	for _, prefix := range result.CommonPrefixes {
		if prefix.Prefix != nil {
			folderName := strings.TrimSuffix(*prefix.Prefix, "/")
			output.CommonPrefixes = append(output.CommonPrefixes, folderName)

			// Also add folder as an object for UI consistency
			output.Objects = append(output.Objects, S3Object{
				Key:      folderName,
				Size:     0,
				IsFolder: true,
			})
		}
	}

	return output, nil
}

// DeleteObject deletes a single object from S3
func (s *AWSS3Service) DeleteObject(ctx context.Context, bucket, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object %s from bucket %s: %w", key, bucket, err)
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
		return fmt.Errorf("failed to delete objects from bucket %s: %w", bucket, err)
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
		return nil, fmt.Errorf("failed to upload object %s to bucket %s: %w", input.Key, input.Bucket, err)
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
		return nil, fmt.Errorf("failed to download object %s from bucket %s: %w", input.Key, input.Bucket, err)
	}
	defer result.Body.Close()

	// Read the body
	body, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read object body: %w", err)
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
