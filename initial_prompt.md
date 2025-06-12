### Role: Expert Go and Frontend Developer

You are an expert software developer specializing in creating robust, single-binary, cross-platform tools with Go and modern web frontends. Your task is to develop a local S3 client application named `s3c`.

### Project Goal

Create a single-binary, cross-platform S3 client (`s3c`) written in Go. The application will serve a web-based GUI on a local port, allowing users to interact with S3-compatible object storage.

### Core Technologies

*   **Backend**: Go 1.24, `net/http` (standard library), AWS SDK for Go v2, `urfave/cli/v2`.
*   **Frontend**: React.js with Vite, styled with Tailwind CSS (Light Theme).
*   **Testing**: Go's standard testing package, `testcontainers-go` with `localstack` for S3 integration tests. Jest/Vitest for frontend logic.
*   **CI/CD**: GitHub Actions with GoReleaser.
*   **Task Runner**: Makefile.

### Step-by-Step Implementation Plan

Follow these steps to build the application. For each step, generate the necessary code, including source files, tests, and configuration.

#### Step 1: Project Scaffolding and Backend Setup

1.  **Initialize Go Module**: Create a new Go module.
2.  **CLI with `urfave/cli/v2`**:
    *   Implement the main application entry point.
    *   Add a `-p` or `--port` flag to specify the serving port, defaulting to `8080`.
    *   Set up a basic "start server" command.
3.  **Basic HTTP Server**:
    *   Using the `net/http` package, create a simple HTTP server.
    *   Create a handler that will eventually serve the frontend static files.
    *   Create a placeholder `/api` endpoint that returns a JSON message like `{"status": "ok"}`.
4.  **Makefile**: Create a `Makefile` with initial targets:
    *   `run`: Runs the Go application (`go run .`).
    *   `build`: Builds the Go binary.
    *   `test`: Runs the Go tests (`go test ./...`).

#### Step 2: Frontend Scaffolding

1.  **Initialize Frontend Project**: Inside a `frontend` directory, set up a new React.js project using Vite.
2.  **Install Dependencies**: Add `tailwindcss` and its dependencies. Configure `tailwind.config.js` and `postcss.config.js`.
3.  **Basic Layout**: Create the main application layout with a left sidebar (for navigation) and a main content area, using a light theme.
4.  **Makefile Integration**: Add targets to the root `Makefile` to manage the frontend:
    *   `frontend/install`: Runs `npm install` in the `frontend` directory.
    *   `frontend/dev`: Runs the Vite development server (`npm run dev`).
    *   `frontend/build`: Builds the static frontend assets (`npm run build`).

#### Step 3: Integrating Frontend and Backend

1.  **Embed Frontend Assets**:
    *   Use Go's `embed` package to embed the built frontend static files (from `frontend/dist`) into the Go binary.
    *   Modify the Go HTTP server to serve these embedded files. The root path `/` should serve `index.html`, and other paths (`/assets/*`) should serve the corresponding static files.
2.  **Update Build Process**: Modify the `build` target in the `Makefile` to first build the frontend and then build the Go binary.

#### Step 4: Settings Page and S3 Configuration

1.  **Backend API for Settings**:
    *   Create an API endpoint (`POST /api/settings`) that accepts a JSON payload with `profile`, `endpoint_url`, and `region`.
    *   This handler will initialize an in-memory AWS `Config` object using the provided details. For now, just store it in a global variable. Subsequent API calls will use this config.
    *   Create an endpoint (`GET /api/profiles`) that reads `~/.aws/credentials` and returns a list of profile names.
2.  **Frontend Settings Page**:
    *   Create a React component for the settings page (`/settings`).
    *   On load, it should fetch profiles from `/api/profiles` and populate a dropdown.
    *   Include input fields for Endpoint URL and Region.
    *   On "Connect" button click, it should POST the selected configuration to `/api/settings`. On success, redirect to the main bucket list view (`/`).
    *   The application should redirect to `/settings` if no configuration is set.

#### Step 5: Bucket and Object Listing

1.  **Backend API**:
    *   `GET /api/buckets`: List all buckets.
    *   `GET /api/buckets/{bucketName}`: List objects in a bucket. Support a `prefix` query parameter for folders and a `marker` or `continuation-token` for pagination. Return objects and a `next_marker`.
2.  **Frontend Display**:
    *   Create components to display bucket and object lists in a table format.
    *   The table should show "Name", "Last Modified", and "Size".
    *   Implement routing with `react-router-dom` to handle URLs like `/buckets/{bucketName}` and `/buckets/{bucketName}/{prefix...}`.
    *   Implement pagination logic (e.g., a "Load More" button or page numbers).

#### Step 6: Core S3 Operations (Download, Delete, Upload)

1.  **Backend API Endpoints**:
    *   `POST /api/download`: Accepts a list of files or a single folder to download. For a folder, zip it server-side and stream the response.
    *   `POST /api/delete`: Accepts a list of files or a single folder to delete. Handle recursive deletion for folders.
    *   `POST /api/upload`: Handle `multipart/form-data` uploads to a specified bucket/prefix. Implement progress tracking using a websocket or SSE connection.
2.  **Frontend UI**:
    *   Add checkboxes to the list for selection.
    *   Implement UI logic to enable/disable "Download" and "Delete" buttons based on selection.
    *   For downloads, trigger the browser's file download.
    *   For uploads, create a dedicated page (`/upload`) with a drag-and-drop area and a file input button.
    *   Implement a blocking modal with a progress bar for long-running operations. Use toast notifications for success/error messages.

#### Step 7: File Preview

1.  **Backend API**:
    *   `GET /api/preview?file_key=...`: An endpoint to get file content for preview.
    *   Apply security (sanitize text) and size limits (100KB for text, 5MB for images) on the server-side.
2.  **Frontend UI**:
    *   When a file name is clicked, open a modal.
    *   In the modal, fetch content from the `/api/preview` endpoint and render it appropriately (e.g., in a `<pre>` tag for text, `<img>` tag for images).

#### Step 8: Final Touches and CI/CD

1.  **Application Shutdown**:
    *   Create a `/api/shutdown` endpoint.
    *   On the frontend, make the "Exit" icon call this endpoint after a confirmation dialog. The backend handler for this endpoint should call `os.Exit(0)`.
2.  **Integration Testing**:
    *   Write integration tests for the Go API handlers using `testcontainers-go` and `localstack`. Set up a test S3 instance, create buckets/objects, and verify API responses. Remember to use PathStyle access for `localstack` in tests.
3.  **GitHub Actions & GoReleaser**:
    *   Create a `.goreleaser.yml` configuration file.
    *   Set up a GitHub Actions workflow (`.github/workflows/release.yml`) that:
        *   Triggers on new tags.
        *   Checks out the code.
        *   Sets up Go and Node.js.
        *   Installs frontend dependencies.
        *   Builds the frontend assets.
        *   Runs GoReleaser to build cross-platform binaries (embedding the frontend) and create a GitHub Release.

Please proceed step by step, providing the complete code for each part. I will review and guide you as you progress. Let's start with **Step 1: Project Scaffolding and Backend Setup**.
