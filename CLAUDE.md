# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

s3c is a single-binary, cross-platform S3 client written in Go that serves a web-based GUI on localhost. The application allows users to interact with S3-compatible object storage through a browser interface.

## Architecture

- **Backend**: Go 1.24 with standard `net/http`, AWS SDK for Go v2, `urfave/cli/v2`
- **Frontend**: React.js SPA with Vite, Tailwind CSS (light theme) using `@tailwindcss/vite` plugin
- **Routing**: Hash-based routing without React Router, using `hashchange` events
- **Distribution**: Single binary with embedded frontend assets using Go's `embed` package
- **Testing**: Standard Go testing, `testcontainers-go` with `localstack` for S3 integration tests
- **API Design**: POST-unified endpoints for consistency and simplified frontend implementation

## Development Commands

### Go Backend
```bash
# Run the application (defaults to localhost:8080)
go run .

# Run with custom port
go run . -p 3000

# Build binary
go build -o s3c .

# Run tests
go test ./...

# Run integration tests (requires Docker)
go test -tags=integration ./...
```

### Frontend (in frontend/ directory)
```bash
# Install dependencies
npm install

# Development server
npm run dev

# Build for production
npm run build
```

### Makefile Targets
```bash
# Run the application
make run

# Build complete application (frontend + backend)
make build

# Run all tests
make test

# Frontend operations
make frontend/install
make frontend/dev
make frontend/build
```

## Key Implementation Details

### POST-Unified API Design
All API endpoints use POST method for consistency and simplified frontend implementation:

```
POST /api/health           - Health check
POST /api/profiles         - AWS profile listing  
POST /api/settings         - S3 connection configuration
POST /api/buckets          - Bucket listing
POST /api/objects/list     - Object listing with JSON parameters
POST /api/objects/delete   - Object deletion (single/multiple)
POST /api/objects/upload   - File upload (multiple files supported)
POST /api/objects/download - Download (single file/multiple files/folder as ZIP)
POST /api/shutdown         - Server shutdown
```

**Benefits of POST-unified design:**
- Consistent request pattern across all endpoints
- Complex parameters easily expressed in JSON body
- No URL path construction needed in frontend
- Unified error handling and request interceptors

### Enhanced File Operations
- **Multiple file upload**: Structured uploads via JSON configuration + multipart form
- **Unified download**: Single endpoint handles files/folders via Keys array approach
- **ZIP generation**: Automatic ZIP creation for multiple files and folder downloads
- **Partial failure handling**: Graceful error handling for batch operations

### AWS Configuration
- Reads AWS profiles from `~/.aws/credentials`
- Settings (profile, endpoint URL, region) are stored in memory only
- PathStyle access is disabled in production (enabled only for localstack tests)

### Web Server Structure
- Root `/` serves the React SPA `index.html` (all routes go to same HTML)
- API endpoints under `/api/` with POST-unified design
- Frontend assets embedded in Go binary using `embed`
- Complete SPA architecture: all routing handled client-side via hash navigation

### Frontend Architecture
- Hash-based routing using `window.location.hash` and `hashchange` events
- URLs format: `#/buckets/my-bucket`, `#/settings`, `#/upload`
- Application state synchronized with hash fragments
- No external routing library dependencies for minimal bundle size and complexity

### S3 Operations
- Bucket and object listing with pagination (100 items per page)
- Multiple file upload with structured configuration
- Download (individual files, multiple files as ZIP, or folders as ZIP)
- Delete with recursive folder support and batch operations
- File preview for text files (<100KB) and images (<5MB)

### S3 Folder Handling Philosophy
**Important**: S3 has no native concept of "folders" - everything is an object with a key. Folder detection is heuristic-based and follows common S3 client conventions:

- **Folder Marker Detection**: `size == 0 && strings.HasSuffix(key, "/")`
- **CommonPrefixes**: Treated as folders when delimiter is used
- **Consistency**: Follows AWS CLI, boto3, and other popular S3 tools
- **Limitation**: 100% accurate folder detection is impossible in S3

This approach prioritizes usability and consistency with established S3 tooling patterns.

### Testing Strategy
- Unit tests with high coverage for Go backend
- Integration tests using testcontainers-go and localstack
- Frontend logic testing with Jest/Vitest
- POST-unified API testing with proper request/response validation
- Use PathStyle access only in localstack integration tests

## Architecture Achievements

### ✅ Implemented Clean Architecture

The current implementation successfully addresses many common design issues:

#### 1. Well-Defined Interfaces
- **S3Operations**: Clean interface for S3 operations with proper mocking support
- **ProfileProvider**: Abstracted AWS profile reading for testability
- **S3ServiceCreator**: Factory pattern for dependency injection

#### 2. Proper Separation of Concerns  
- **API Handlers**: Focus purely on HTTP concerns (request parsing, response formatting)
- **Service Layer**: Business logic isolated in service package
- **Repository Layer**: Data access abstracted through interfaces

#### 3. Comprehensive Testing Strategy
- **Unit Tests**: 80%+ coverage with proper mocks and test doubles
- **Integration Tests**: Using testcontainers-go and localstack for S3 testing
- **HTTP Tests**: Complete request/response testing with httptest

#### 4. POST-Unified API Benefits
- **Simplified Testing**: Consistent request patterns across all endpoints
- **Better Error Handling**: Unified error response structure
- **Enhanced Functionality**: Support for complex operations (multiple file upload, ZIP downloads)
- **Frontend Consistency**: Single HTTP client pattern for all API calls

### 🎯 Current Best Practices

#### Interface-Based Design
```go
type S3Operations interface {
    ListBuckets(ctx context.Context) ([]string, error)
    ListObjects(ctx context.Context, input ListObjectsInput) (*ListObjectsOutput, error)
    UploadObject(ctx context.Context, input UploadObjectInput) (*UploadObjectOutput, error)
    DownloadObject(ctx context.Context, input DownloadObjectInput) (*DownloadObjectOutput, error)
    DeleteObject(ctx context.Context, bucket, key string) error
    DeleteObjects(ctx context.Context, bucket string, keys []string) error
    TestConnection(ctx context.Context) error
}
```

#### Clean Handler Implementation
- Request validation and parameter extraction
- Delegate business logic to service layer
- Consistent error response formatting
- Proper HTTP status code usage

#### Robust Testing Infrastructure
- Mock implementations for all external dependencies
- Table-driven tests with comprehensive edge cases
- Integration tests with real S3-compatible storage
- HTTP handler tests with complete request/response validation

## S3 Folder Handling Philosophy

### Fundamental AWS S3 Constraints

AWS S3 has **no native concept of "folders"** - everything is an object with a key. What users perceive as "folders" are actually:

1. **Folder Markers**: Zero-size objects with keys ending in "/" (e.g., `folder1/`)
2. **CommonPrefixes**: S3 API groups objects by common path segments when using delimiter

### Heuristic-Based Folder Detection

Our implementation uses the standard S3 client convention:
```go
isFolder := size == 0 && strings.HasSuffix(key, "/")
```

This heuristic is used by:
- AWS CLI
- AWS Console
- Most S3 client libraries
- Other S3-compatible storage tools

### Test Case Reality

When listing objects **without delimiter**, folder markers appear as regular objects but are correctly identified as folders by size and suffix heuristics:

```
Objects without delimiter:
- file1.txt (size: 16, isFolder: false)
- folder1/ (size: 0, isFolder: true)    ← Folder marker detected
- folder1/file3.txt (size: 16, isFolder: false)
- folder1/subfolder/ (size: 0, isFolder: true) ← Nested folder marker
```

This behavior is **correct** and matches AWS S3's fundamental architecture. The folder detection logic properly identifies zero-size objects ending with "/" as folders, regardless of delimiter usage.