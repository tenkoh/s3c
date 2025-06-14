package errors

import (
	"errors"
	"testing"
)

func TestS3CError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *S3CError
		expected string
	}{
		{
			name:     "simple error",
			err:      NewS3CError(CodeInvalidInput, CategoryValidation, SeverityError, "test message"),
			expected: "test message",
		},
		{
			name: "error with wrapped",
			err: NewS3CError(CodeS3Connection, CategoryS3, SeverityError, "connection failed").
				WithWrapped(errors.New("network error")),
			expected: "connection failed: network error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("S3CError.Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestS3CError_Is(t *testing.T) {
	err1 := NewInvalidInputError("test", "value")
	err2 := NewInvalidInputError("other", "value")
	err3 := NewMissingFieldError("field")

	tests := []struct {
		name     string
		err      error
		target   error
		expected bool
	}{
		{
			name:     "same error code",
			err:      err1,
			target:   err2,
			expected: true,
		},
		{
			name:     "different error code",
			err:      err1,
			target:   err3,
			expected: false,
		},
		{
			name:     "non-S3CError target",
			err:      err1,
			target:   errors.New("other"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(tt.err, tt.target); got != tt.expected {
				t.Errorf("errors.Is() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestS3CError_As(t *testing.T) {
	originalErr := NewS3ConnectionError(errors.New("network error"))
	wrappedErr := errors.Join(originalErr, errors.New("other error"))

	var s3cErr *S3CError
	if !errors.As(wrappedErr, &s3cErr) {
		t.Error("errors.As() should find S3CError in wrapped error")
	}

	if s3cErr.Code != CodeS3Connection {
		t.Errorf("Found error code = %v, want %v", s3cErr.Code, CodeS3Connection)
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "network timeout is retryable",
			err:      NewNetworkTimeoutError("test"),
			expected: true,
		},
		{
			name:     "invalid input is not retryable",
			err:      NewInvalidInputError("field", "value"),
			expected: false,
		},
		{
			name:     "S3 connection error is retryable",
			err:      NewS3ConnectionError(errors.New("connection failed")),
			expected: true,
		},
		{
			name:     "access denied is not retryable",
			err:      NewS3AccessDeniedError("read", "bucket"),
			expected: false,
		},
		{
			name:     "non-S3CError is not retryable",
			err:      errors.New("unknown error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.expected {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsUserError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "validation error is user error",
			err:      NewInvalidInputError("field", "value"),
			expected: true,
		},
		{
			name:     "config error is user error",
			err:      NewProfileNotFoundError("profile"),
			expected: true,
		},
		{
			name:     "S3 error is not user error",
			err:      NewS3ConnectionError(errors.New("network")),
			expected: false,
		},
		{
			name:     "internal error is not user error",
			err:      NewFileOperationError("read", "/path", errors.New("io error")),
			expected: false,
		},
		{
			name:     "non-S3CError is not user error",
			err:      errors.New("unknown error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUserError(tt.err); got != tt.expected {
				t.Errorf("IsUserError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetSeverity(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected Severity
	}{
		{
			name:     "validation error has error severity",
			err:      NewInvalidInputError("field", "value"),
			expected: SeverityError,
		},
		{
			name:     "network error has warning severity",
			err:      NewNetworkTimeoutError("operation"),
			expected: SeverityWarning,
		},
		{
			name:     "internal error has critical severity",
			err:      NewFileOperationError("read", "/path", errors.New("io error")),
			expected: SeverityCritical,
		},
		{
			name:     "non-S3CError has default error severity",
			err:      errors.New("unknown error"),
			expected: SeverityError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSeverity(tt.err); got != tt.expected {
				t.Errorf("GetSeverity() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewNotImplementedError(t *testing.T) {
	err := NewNotImplementedError("test feature")

	if !errors.Is(err, errors.ErrUnsupported) {
		t.Error("NewNotImplementedError should wrap errors.ErrUnsupported")
	}

	if err.Code != CodeNotImplemented {
		t.Errorf("Error code = %v, want %v", err.Code, CodeNotImplemented)
	}
}

func TestJoinErrors(t *testing.T) {
	err1 := NewInvalidInputError("field1", "value1")
	err2 := NewMissingFieldError("field2")
	err3 := errors.New("standard error")

	joined := JoinErrors(err1, err2, err3)

	if !errors.Is(joined, err1) {
		t.Error("Joined error should contain err1")
	}
	if !errors.Is(joined, err2) {
		t.Error("Joined error should contain err2")
	}
	if !errors.Is(joined, err3) {
		t.Error("Joined error should contain err3")
	}
}
