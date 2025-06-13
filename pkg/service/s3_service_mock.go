package service

import "context"

// MockS3Service implements S3Operations for testing
type MockS3Service struct {
	ListBucketsFunc    func(ctx context.Context) ([]string, error)
	TestConnectionFunc func(ctx context.Context) error
	ListObjectsFunc    func(ctx context.Context, input ListObjectsInput) (*ListObjectsOutput, error)
	DeleteObjectFunc   func(ctx context.Context, bucket, key string) error
	DeleteObjectsFunc  func(ctx context.Context, bucket string, keys []string) error
	UploadObjectFunc   func(ctx context.Context, input UploadObjectInput) (*UploadObjectOutput, error)
	DownloadObjectFunc func(ctx context.Context, input DownloadObjectInput) (*DownloadObjectOutput, error)
}

// NewMockS3Service creates a new mock S3 service
func NewMockS3Service() *MockS3Service {
	return &MockS3Service{}
}

// ListBuckets calls the mock function if set, otherwise returns empty slice
func (m *MockS3Service) ListBuckets(ctx context.Context) ([]string, error) {
	if m.ListBucketsFunc != nil {
		return m.ListBucketsFunc(ctx)
	}
	return []string{}, nil
}

// TestConnection calls the mock function if set, otherwise returns nil
func (m *MockS3Service) TestConnection(ctx context.Context) error {
	if m.TestConnectionFunc != nil {
		return m.TestConnectionFunc(ctx)
	}
	return nil
}

// ListObjects calls the mock function if set, otherwise returns empty result
func (m *MockS3Service) ListObjects(ctx context.Context, input ListObjectsInput) (*ListObjectsOutput, error) {
	if m.ListObjectsFunc != nil {
		return m.ListObjectsFunc(ctx, input)
	}
	return &ListObjectsOutput{
		Objects:        []S3Object{},
		CommonPrefixes: []string{},
		IsTruncated:    false,
	}, nil
}

// DeleteObject calls the mock function if set, otherwise returns nil
func (m *MockS3Service) DeleteObject(ctx context.Context, bucket, key string) error {
	if m.DeleteObjectFunc != nil {
		return m.DeleteObjectFunc(ctx, bucket, key)
	}
	return nil
}

// DeleteObjects calls the mock function if set, otherwise returns nil
func (m *MockS3Service) DeleteObjects(ctx context.Context, bucket string, keys []string) error {
	if m.DeleteObjectsFunc != nil {
		return m.DeleteObjectsFunc(ctx, bucket, keys)
	}
	return nil
}

// UploadObject calls the mock function if set, otherwise returns default result
func (m *MockS3Service) UploadObject(ctx context.Context, input UploadObjectInput) (*UploadObjectOutput, error) {
	if m.UploadObjectFunc != nil {
		return m.UploadObjectFunc(ctx, input)
	}
	return &UploadObjectOutput{
		Key:  input.Key,
		ETag: "\"mock-etag-12345\"",
	}, nil
}

// DownloadObject calls the mock function if set, otherwise returns default result
func (m *MockS3Service) DownloadObject(ctx context.Context, input DownloadObjectInput) (*DownloadObjectOutput, error) {
	if m.DownloadObjectFunc != nil {
		return m.DownloadObjectFunc(ctx, input)
	}
	return &DownloadObjectOutput{
		Body:          []byte("mock file content"),
		ContentType:   "text/plain",
		ContentLength: 17,
		LastModified:  "2023-01-01T00:00:00Z",
		Metadata:      map[string]string{},
	}, nil
}

// NewMockS3Service creates a mock S3 service creator function
func NewMockS3ServiceCreator(mockService *MockS3Service) func(ctx context.Context, cfg S3Config) (S3Operations, error) {
	return func(ctx context.Context, cfg S3Config) (S3Operations, error) {
		return mockService, nil
	}
}

// MockProfileRepository implements ProfileProvider for testing
type MockProfileRepository struct {
	GetProfilesFunc func() ([]string, error)
}

// NewMockProfileRepository creates a new mock profile repository
func NewMockProfileRepository() *MockProfileRepository {
	return &MockProfileRepository{}
}

// GetProfiles calls the mock function if set, otherwise returns empty slice
func (m *MockProfileRepository) GetProfiles() ([]string, error) {
	if m.GetProfilesFunc != nil {
		return m.GetProfilesFunc()
	}
	return []string{}, nil
}
