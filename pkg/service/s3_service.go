package service

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

// S3ServiceFactory creates AWS S3 service instances
type AWSS3ServiceFactory struct{}

// NewAWSS3ServiceFactory creates a new AWS S3 service factory
func NewAWSS3ServiceFactory() *AWSS3ServiceFactory {
	return &AWSS3ServiceFactory{}
}

// S3Operations interface for dependency injection
type S3Operations interface {
	ListBuckets(ctx context.Context) ([]string, error)
	TestConnection(ctx context.Context) error
}

// CreateS3Service creates a new S3Service with the given configuration
func (f *AWSS3ServiceFactory) CreateS3Service(ctx context.Context, cfg S3Config) (S3Operations, error) {
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