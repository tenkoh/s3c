# s3c

> **⚠️ DEVELOPMENT WARNING**: This project is under active development. While there is no malicious intent towards your data, bugs may exist. **Use with production or important S3 data at your own risk. We accept no responsibility for any data loss or corruption.**

> **Experimental Project**: This application was primarily developed using [Claude Code](https://claude.ai/code) as an experiment in AI-assisted software development.

S3 and S3-compatible object storage Client (s3c) - A cross-platform, single-binary GUI application for managing S3 and S3-compatible object storage services through a modern web interface.

Built with Go and React, s3c provides a localhost-served web GUI that makes it easy to interact with AWS S3, MinIO, LocalStack, and other S3-compatible storage systems.

## Motivation

This project was created to provide a user-friendly GUI for interacting with S3-compatible emulators like LocalStack during development and testing workflows, while also serving as a general-purpose S3 management tool.

## Features

- 🚀 **Single Binary**: Cross-platform executable with embedded frontend
- 🌐 **Web-based GUI**: Modern React interface served on localhost
- 📁 **S3 Operations**: Bucket listing, object upload/download/delete, folder navigation
- 🔄 **Batch Operations**: Multiple file upload, bulk download with ZIP compression
- 👀 **File Preview**: Text files (30+ formats) and images with zoom/pan controls
- ⚙️ **Flexible Configuration**: AWS profiles, custom endpoints, region selection

## Supported Operations

### ✅ Currently Supported
- **Bucket Listing**: View all available S3 buckets
- **Object Listing**: Browse bucket contents with folder navigation
- **File Download**: Single file download with original filename preservation
- **Bulk Download**: Multiple files download with automatic ZIP compression
- **Folder Download**: Recursive folder download as ZIP archive
- **File Upload**: Multiple file upload with drag & drop support
- **File Preview**: Text files (30+ formats, <100KB) and images (JPEG/PNG/GIF/SVG/WebP, <5MB)
- **File Deletion**: Single file and batch deletion operations

### 🚧 Planned Features
- **Bucket Creation**: Create new S3 buckets
- **Folder Creation**: Create new folders within buckets

## Installation

### Using Go (Recommended)

```bash
go install github.com/tenkoh/s3c@latest
```

### Pre-built Binaries

Pre-built binaries will be available through [GitHub Releases](https://github.com/tenkoh/s3c/releases) (coming soon).

## Prerequisites

### AWS Credentials Configuration

s3c requires AWS credentials to be configured in your home directory:

```bash
# Ensure you have AWS profiles configured
cat ~/.aws/credentials
```

Example configuration:
```ini
[default]
aws_access_key_id = YOUR_ACCESS_KEY
aws_secret_access_key = YOUR_SECRET_KEY

[localstack]
aws_access_key_id = test
aws_secret_access_key = test
```

### Required IAM Permissions

Your AWS profile must have the following S3 permissions:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "s3:ListAllMyBuckets",
                "s3:ListBucket", 
                "s3:GetObject",
                "s3:PutObject",
                "s3:DeleteObject"
            ],
            "Resource": [
                "arn:aws:s3:::*",
                "arn:aws:s3:::*/*"
            ]
        }
    ]
}
```

## Usage

### Basic Usage

```bash
# Start the application (default port 8080)
s3c

# Start on custom port
s3c --port 3000

# Enable debug logging
s3c --log-level debug --log-format text
```

### Command Options

- `--port, -p`: Port to serve the web interface (default: 8080)
- `--log-level`: Log level - debug, info, warn, error (default: info)
- `--log-format`: Log format - text, json (default: json)
- `--help, -h`: Show help

### Accessing the Interface

After starting s3c, open your browser and navigate to:
```
http://localhost:8080
```

## GUI Interface

### Home Page (`/`)
- **Bucket Listing**: Displays all available S3 buckets
- **Connection Status**: Shows current S3 connection status
- **Quick Navigation**: Click any bucket to browse its contents

### Settings Page (`/settings`)
- **Profile Selection**: Choose from available AWS profiles in `~/.aws/credentials`
- **Region Configuration**: Enter AWS region (required for AWS S3)
- **Endpoint URL**: Specify custom endpoint for S3-compatible services (leave empty for AWS S3)

Configuration is stored in memory only and must be set each time the application starts.

### Objects Browser (`/buckets/:bucket`)
- **File Management**: Upload, download, delete files and folders
- **Folder Navigation**: Navigate through nested folder structures
- **Batch Operations**: Select multiple items for bulk operations
- **File Preview**: View text files and images directly in the browser
- **Drag & Drop Upload**: Drop files directly into the browser

### Upload Interface (`/upload`)
- **Multi-file Upload**: Select or drag multiple files for batch upload
- **S3 Key Customization**: Edit object keys before uploading
- **Progress Tracking**: Real-time upload progress for each file
- **Context-aware**: Automatically targets current bucket/folder when accessed from object browser

### File Preview
- **Text Files**: Syntax highlighting for 30+ file formats (up to 100KB)
- **Images**: JPEG, PNG, GIF, SVG, WebP support with zoom/pan controls (up to 5MB)
- **Modal Interface**: ESC key or click-to-close preview overlay

## Screenshots

TODO: Add screenshots of the main interface

## Development

### Building and Running

```bash
# Run the application in development mode
make run

# Build the complete application
make build

# Run Go tests
go test ./...

# Run integration tests (requires Docker)
go test -tags=integration ./...
```

### Project Structure

- **Backend**: Go with standard `net/http` and AWS SDK v2
- **Frontend**: React.js with Vite and Tailwind CSS
- **Distribution**: Single binary with embedded frontend assets

### Contributing

We welcome contributions! Areas where help is particularly appreciated:

- **Frontend Testing**: Jest/Vitest test suite implementation
- **End-to-End Testing**: Browser automation test coverage
- **Frontend Tooling**: ESLint, Prettier, and formatting improvements

Please report issues and submit pull requests through [GitHub Issues](https://github.com/tenkoh/s3c/issues).

## License

MIT License

## Author

[tenkoh](https://github.com/tenkoh)
