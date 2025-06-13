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

### AWS Configuration
- Reads AWS profiles from `~/.aws/credentials`
- Settings (profile, endpoint URL, region) are stored in memory only
- PathStyle access is disabled in production (enabled only for localstack tests)

### Web Server Structure
- Root `/` serves the React SPA `index.html` (all routes go to same HTML)
- API endpoints under `/api/`
- Frontend assets embedded in Go binary using `embed`
- Complete SPA architecture: all routing handled client-side via hash navigation

### Frontend Architecture
- Hash-based routing using `window.location.hash` and `hashchange` events
- URLs format: `#/buckets/my-bucket`, `#/settings`, `#/upload`
- Application state synchronized with hash fragments
- No external routing library dependencies for minimal bundle size and complexity

### S3 Operations
- Bucket and object listing with pagination (100 items per page)
- File upload with progress tracking
- Download (individual files or folders as ZIP)
- Delete with recursive folder support
- File preview for text files (<100KB) and images (<5MB)

### Testing Strategy
- Unit tests with high coverage for Go backend
- Integration tests using testcontainers-go and localstack
- Frontend logic testing with Jest/Vitest
- Use PathStyle access only in localstack integration tests

## Current Design Issues & Refactoring Priorities

### ⚠️ Critical Design Problems Identified

#### 1. Testability Issues (HIGH PRIORITY)
- **S3Service**: Direct AWS SDK dependency - cannot be mocked for unit tests
- **ProfileReader**: Direct filesystem dependency - requires real files for testing
- **API Handlers**: HTTP handling mixed with business logic - difficult to unit test
- **AppState**: Global state management - concurrency issues and test isolation problems

#### 2. SOLID Principle Violations
- **SRP Violation**: API handlers have multiple responsibilities (HTTP + business logic + error formatting)
- **OCP Violation**: Adding new S3 providers requires modifying existing code
- **LSP Violation**: No interfaces defined - cannot substitute implementations
- **ISP Violation**: Large structs with many responsibilities
- **DIP Violation**: Depends on concrete types instead of abstractions

#### 3. 12 Factor App Violations
- **Config**: Settings stored in memory instead of environment variables
- **Dependencies**: Poor dependency injection - hard-coded dependencies
- **Backing Services**: S3 service tightly coupled instead of treated as attached resource

### 🔧 Required Refactoring Strategy

#### Phase 1: Interface Extraction (Immediate)
```go
type S3Operations interface {
    ListBuckets(ctx context.Context) ([]string, error)
    TestConnection(ctx context.Context) error
}

type ProfileProvider interface {
    GetProfiles() ([]string, error)
}

type ConfigProvider interface {
    LoadS3Config(profile, region, endpoint string) (*S3Config, error)
}
```

#### Phase 2: Layer Separation
1. **Handler Layer**: Pure HTTP concerns only
2. **Service Layer**: Business logic and validation
3. **Repository Layer**: Data access (S3, filesystem)

#### Phase 3: Dependency Injection
```go
type Services struct {
    S3Ops     S3Operations
    Profiles  ProfileProvider
    Config    ConfigProvider
}

type Server struct {
    services *Services
    // ... other fields
}
```

#### Phase 4: Test Infrastructure
- Mock implementations for all interfaces
- Test helpers and fixtures
- Table-driven tests for all business logic
- HTTP test helpers for handler testing

### 🧪 Testing Implementation Guidelines

#### Unit Test Requirements
- **Coverage Target**: 80%+ for business logic
- **Mock Strategy**: Interface-based mocks, avoid AWS SDK calls
- **Test Structure**: Arrange-Act-Assert pattern
- **Error Testing**: Test all error paths explicitly

#### Integration Test Strategy
- **TestContainers**: Use for S3 integration tests only
- **Filesystem Tests**: Use temporary directories, not real AWS credentials
- **HTTP Tests**: Use httptest.Server for full request/response testing

#### Test File Structure
```
pkg/
├── service/
│   ├── s3_service.go
│   ├── s3_service_test.go
│   └── s3_service_integration_test.go
├── handler/
│   ├── api_handler.go
│   └── api_handler_test.go
└── repository/
    ├── profile_reader.go
    └── profile_reader_test.go
```

### 📋 Immediate Action Items

1. **Extract Interfaces**: Define S3Operations, ProfileProvider, ConfigProvider
2. **Separate Concerns**: Move business logic out of HTTP handlers
3. **Add Unit Tests**: Start with pure business logic functions
4. **Implement DI**: Use constructor injection for dependencies
5. **Environment Config**: Move to environment variables for configuration
6. **Mock Infrastructure**: Create test doubles for all external dependencies

### 🚫 Anti-Patterns to Avoid

- Direct AWS SDK calls in business logic
- Filesystem operations without abstraction
- HTTP request/response handling in business logic
- Global state mutation
- Hard-coded configuration values
- Missing error handling tests