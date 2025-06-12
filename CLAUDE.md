# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

s3c is a single-binary, cross-platform S3 client written in Go that serves a web-based GUI on localhost. The application allows users to interact with S3-compatible object storage through a browser interface.

## Architecture

- **Backend**: Go 1.24 with standard `net/http`, AWS SDK for Go v2, `urfave/cli/v2`
- **Frontend**: React.js with Vite, Tailwind CSS (light theme)
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
- Root `/` serves the React SPA
- API endpoints under `/api/`
- Frontend assets embedded in Go binary using `embed`

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