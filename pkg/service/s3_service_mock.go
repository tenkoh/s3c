package service

import "context"

// MockS3Service implements S3Operations for testing
type MockS3Service struct {
	ListBucketsFunc    func(ctx context.Context) ([]string, error)
	TestConnectionFunc func(ctx context.Context) error
	ListObjectsFunc    func(ctx context.Context, input ListObjectsInput) (*ListObjectsOutput, error)
	DeleteObjectFunc   func(ctx context.Context, bucket, key string) error
	DeleteObjectsFunc  func(ctx context.Context, bucket string, keys []string) error
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

// MockS3ServiceFactory implements S3ServiceFactory for testing
type MockS3ServiceFactory struct {
	CreateS3ServiceFunc func(ctx context.Context, cfg S3Config) (*MockS3Service, error)
	mockService         *MockS3Service
}

// NewMockS3ServiceFactory creates a new mock S3 service factory
func NewMockS3ServiceFactory(mockService *MockS3Service) *MockS3ServiceFactory {
	return &MockS3ServiceFactory{
		mockService: mockService,
	}
}

// CreateS3Service creates a mock S3 service
func (f *MockS3ServiceFactory) CreateS3Service(ctx context.Context, cfg S3Config) (*MockS3Service, error) {
	if f.CreateS3ServiceFunc != nil {
		return f.CreateS3ServiceFunc(ctx, cfg)
	}
	return f.mockService, nil
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