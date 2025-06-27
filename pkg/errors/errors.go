package errors

import (
	"errors"
	"fmt"
)

// ErrorCode represents a specific error code for categorization
type ErrorCode string

// Error codes for different categories
const (
	// Validation errors
	CodeInvalidInput  ErrorCode = "INVALID_INPUT"
	CodeMissingField  ErrorCode = "MISSING_FIELD"
	CodeInvalidFormat ErrorCode = "INVALID_FORMAT"
	CodeOutOfRange    ErrorCode = "OUT_OF_RANGE"

	// S3 operation errors
	CodeS3Connection     ErrorCode = "S3_CONNECTION"
	CodeS3BucketNotFound ErrorCode = "S3_BUCKET_NOT_FOUND"
	CodeS3ObjectNotFound ErrorCode = "S3_OBJECT_NOT_FOUND"
	CodeS3AccessDenied   ErrorCode = "S3_ACCESS_DENIED"
	CodeS3QuotaExceeded  ErrorCode = "S3_QUOTA_EXCEEDED"
	CodeS3Operation      ErrorCode = "S3_OPERATION"

	// Configuration errors
	CodeConfigMissing      ErrorCode = "CONFIG_MISSING"
	CodeConfigInvalid      ErrorCode = "CONFIG_INVALID"
	CodeProfileNotFound    ErrorCode = "PROFILE_NOT_FOUND"
	CodeCredentialsInvalid ErrorCode = "CREDENTIALS_INVALID"

	// Network errors
	CodeNetworkTimeout     ErrorCode = "NETWORK_TIMEOUT"
	CodeNetworkUnavailable ErrorCode = "NETWORK_UNAVAILABLE"
	CodeNetworkUnknown     ErrorCode = "NETWORK_UNKNOWN"

	// Internal errors
	CodeInternalError  ErrorCode = "INTERNAL_ERROR"
	CodeNotImplemented ErrorCode = "NOT_IMPLEMENTED"
	CodeFileOperation  ErrorCode = "FILE_OPERATION"
)

// ErrorCategory represents the category of an error
type ErrorCategory string

const (
	CategoryValidation ErrorCategory = "validation"
	CategoryS3         ErrorCategory = "s3"
	CategoryConfig     ErrorCategory = "config"
	CategoryNetwork    ErrorCategory = "network"
	CategoryInternal   ErrorCategory = "internal"
)

// Severity represents the severity level of an error
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// S3CError represents a structured error in s3c application
type S3CError struct {
	Code       ErrorCode     `json:"code"`
	Category   ErrorCategory `json:"category"`
	Severity   Severity      `json:"severity"`
	Message    string        `json:"message"`
	Details    any           `json:"details,omitempty"`
	Suggestion string        `json:"suggestion,omitempty"`
	Wrapped    error         `json:"-"` // Original error, not serialized
}

// Error implements the error interface
func (e *S3CError) Error() string {
	if e.Wrapped != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Wrapped)
	}
	return e.Message
}

// Unwrap implements the error unwrapping interface for Go 1.13+
func (e *S3CError) Unwrap() error {
	return e.Wrapped
}

// Is implements error comparison for Go 1.13+ errors.Is
func (e *S3CError) Is(target error) bool {
	t, ok := target.(*S3CError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// NewS3CError creates a new structured error
func NewS3CError(code ErrorCode, category ErrorCategory, severity Severity, message string) *S3CError {
	return &S3CError{
		Code:     code,
		Category: category,
		Severity: severity,
		Message:  message,
	}
}

// WithDetails adds details to the error
func (e *S3CError) WithDetails(details any) *S3CError {
	e.Details = details
	return e
}

// WithSuggestion adds a suggestion to the error
func (e *S3CError) WithSuggestion(suggestion string) *S3CError {
	e.Suggestion = suggestion
	return e
}

// WithWrapped adds a wrapped error
func (e *S3CError) WithWrapped(err error) *S3CError {
	e.Wrapped = err
	return e
}

// Validation error constructors
func NewValidationError(code ErrorCode, message string) *S3CError {
	return NewS3CError(code, CategoryValidation, SeverityError, message)
}

func NewInvalidInputError(field string, value any) *S3CError {
	return NewValidationError(CodeInvalidInput, fmt.Sprintf("Invalid input for field '%s'", field)).
		WithDetails(map[string]any{
			"field": field,
			"value": value,
		}).
		WithSuggestion("Please check the input format and try again")
}

func NewMissingFieldError(field string) *S3CError {
	return NewValidationError(CodeMissingField, fmt.Sprintf("Required field '%s' is missing", field)).
		WithDetails(map[string]any{
			"field": field,
		}).
		WithSuggestion(fmt.Sprintf("Please provide a value for '%s'", field))
}

// S3 error constructors
func NewS3Error(code ErrorCode, message string) *S3CError {
	return NewS3CError(code, CategoryS3, SeverityError, message)
}

func NewS3ConnectionError(err error) *S3CError {
	return NewS3Error(CodeS3Connection, "Failed to connect to S3").
		WithWrapped(err).
		WithSuggestion("Check your AWS credentials and network connection")
}

func NewS3BucketNotFoundError(bucket string) *S3CError {
	return NewS3Error(CodeS3BucketNotFound, fmt.Sprintf("Bucket '%s' not found", bucket)).
		WithDetails(map[string]any{
			"bucket": bucket,
		}).
		WithSuggestion("Verify the bucket name and your access permissions")
}

func NewS3ObjectNotFoundError(bucket, key string) *S3CError {
	return NewS3Error(CodeS3ObjectNotFound, fmt.Sprintf("Object '%s' not found in bucket '%s'", key, bucket)).
		WithDetails(map[string]any{
			"bucket": bucket,
			"key":    key,
		}).
		WithSuggestion("Check the object key and ensure the object exists")
}

func NewS3AccessDeniedError(operation, resource string) *S3CError {
	return NewS3Error(CodeS3AccessDenied, fmt.Sprintf("Access denied for %s on %s", operation, resource)).
		WithDetails(map[string]any{
			"operation": operation,
			"resource":  resource,
		}).
		WithSuggestion("Check your AWS permissions and IAM policies")
}

func NewS3OperationError(operation string, err error) *S3CError {
	return NewS3Error(CodeS3Operation, fmt.Sprintf("S3 %s operation failed", operation)).
		WithWrapped(err).
		WithDetails(map[string]any{
			"operation": operation,
		})
}

// Config error constructors
func NewConfigError(code ErrorCode, message string) *S3CError {
	return NewS3CError(code, CategoryConfig, SeverityError, message)
}

func NewProfileNotFoundError(profile string) *S3CError {
	return NewConfigError(CodeProfileNotFound, fmt.Sprintf("AWS profile '%s' not found", profile)).
		WithDetails(map[string]any{
			"profile": profile,
		}).
		WithSuggestion("Check your ~/.aws/credentials file or AWS profile configuration")
}

func NewCredentialsInvalidError(err error) *S3CError {
	return NewConfigError(CodeCredentialsInvalid, "Invalid AWS credentials").
		WithWrapped(err).
		WithSuggestion("Verify your AWS access key, secret key, and session token")
}

// Network error constructors
func NewNetworkError(code ErrorCode, message string) *S3CError {
	return NewS3CError(code, CategoryNetwork, SeverityWarning, message)
}

func NewNetworkTimeoutError(operation string) *S3CError {
	return NewNetworkError(CodeNetworkTimeout, fmt.Sprintf("Network timeout during %s", operation)).
		WithDetails(map[string]any{
			"operation": operation,
		}).
		WithSuggestion("Check your network connection and try again")
}

// Internal error constructors
func NewInternalError(code ErrorCode, message string) *S3CError {
	return NewS3CError(code, CategoryInternal, SeverityCritical, message)
}

func NewFileOperationError(operation, path string, err error) *S3CError {
	return NewInternalError(CodeFileOperation, fmt.Sprintf("File %s failed for %s", operation, path)).
		WithWrapped(err).
		WithDetails(map[string]any{
			"operation": operation,
			"path":      path,
		})
}

func NewNotImplementedError(feature string) *S3CError {
	return NewInternalError(CodeNotImplemented, fmt.Sprintf("Feature '%s' is not implemented", feature)).
		WithWrapped(errors.ErrUnsupported). // Go 1.21+ sentinel error
		WithDetails(map[string]any{
			"feature": feature,
		}).
		WithSuggestion("This feature is planned for a future release")
}

// JoinErrors combines multiple errors using Go 1.20+ errors.Join
func JoinErrors(errs ...error) error {
	return errors.Join(errs...)
}

// IsRetryable determines if an error is retryable
func IsRetryable(err error) bool {
	var s3cErr *S3CError
	if errors.As(err, &s3cErr) {
		switch s3cErr.Code {
		case CodeNetworkTimeout, CodeNetworkUnavailable, CodeS3Connection:
			return true
		case CodeS3QuotaExceeded:
			return true // Can retry after some delay
		default:
			return false
		}
	}
	return false
}

// IsUserError determines if an error is caused by user input
func IsUserError(err error) bool {
	var s3cErr *S3CError
	if errors.As(err, &s3cErr) {
		return s3cErr.Category == CategoryValidation || s3cErr.Category == CategoryConfig
	}
	return false
}

// GetSeverity returns the severity of an error
func GetSeverity(err error) Severity {
	var s3cErr *S3CError
	if errors.As(err, &s3cErr) {
		return s3cErr.Severity
	}
	return SeverityError // Default severity for unknown errors
}
