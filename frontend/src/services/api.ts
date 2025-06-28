// API client for POST-unified endpoints

type StructuredAPIError = {
  code: string;
  message: string;
  details?: any;
  suggestion?: string;
  category?: string;
  severity?: string;
  retryable?: boolean;
};

class APIError extends Error {
  public code?: string;
  public details?: any;
  public suggestion?: string;
  public category?: string;
  public severity?: string;
  public retryable?: boolean;
  public requestId?: string;

  constructor(
    message: string,
    public status?: number,
    structured?: StructuredAPIError & { requestId?: string },
  ) {
    super(message);
    this.name = "APIError";

    if (structured) {
      this.code = structured.code;
      this.details = structured.details;
      this.suggestion = structured.suggestion;
      this.category = structured.category;
      this.severity = structured.severity;
      this.retryable = structured.retryable;
      this.requestId = structured.requestId;
    }
  }
}

async function apiCall<T>(endpoint: string, data: any = {}): Promise<T> {
  try {
    const response = await fetch(`/api/${endpoint}`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(data),
    });

    const result = await response.json();

    if (!result.success) {
      // Check if this is a structured error response
      if (
        result.error &&
        typeof result.error === "object" &&
        result.error.code
      ) {
        const structuredError = result.error as StructuredAPIError;
        throw new APIError(structuredError.message, response.status, {
          ...structuredError,
          requestId: result.requestId,
        });
      } else {
        // Legacy error format
        throw new APIError(result.error || "API call failed", response.status);
      }
    }

    return result.data as T;
  } catch (error) {
    if (error instanceof APIError) {
      throw error;
    }

    // Network or parsing errors
    throw new APIError("Network error or server unavailable");
  }
}

// API endpoints
export const api = {
  // Health check
  health: () => apiCall("health"),

  // Connection status
  getStatus: (): Promise<{
    connected: boolean;
    message: string;
    profile?: string;
    region?: string;
    endpoint?: string;
    error?: string;
  }> => apiCall("status"),

  // Profile management
  getProfiles: (): Promise<{ profiles: string[] }> => apiCall("profiles"),

  // S3 configuration
  configureS3: (config: {
    profile: string;
    region: string;
    endpoint?: string;
  }) =>
    apiCall("settings", {
      profile: config.profile,
      region: config.region,
      endpointUrl: config.endpoint || undefined,
    }),

  // Bucket operations
  listBuckets: (): Promise<{ buckets: string[] }> => apiCall("buckets"),

  createBucket: (
    bucketName: string,
  ): Promise<{ message: string; bucket: string }> =>
    apiCall("buckets/create", { name: bucketName }),

  // Folder operations
  createFolder: (
    bucket: string,
    prefix: string,
  ): Promise<{ message: string; bucket: string; prefix: string }> =>
    apiCall("objects/folder/create", { bucket, prefix }),

  // Object operations
  listObjects: (params: {
    bucket: string;
    prefix?: string;
    delimiter?: string;
    maxKeys?: number;
    continuationToken?: string;
  }): Promise<{
    objects: Array<{
      key: string;
      size: number;
      lastModified: string;
      isFolder: boolean;
    }>;
    nextContinuationToken?: string;
    isTruncated: boolean;
  }> => apiCall("objects/list", params),

  deleteObjects: (params: { bucket: string; keys: string[] }) =>
    apiCall("objects/delete", params),

  // Upload operations
  uploadObjects: (params: {
    bucket: string;
    files: { file: File; key: string }[];
    onProgress?: (progress: number) => void;
  }) => {
    const formData = new FormData();

    // Add bucket parameter
    formData.append("bucket", params.bucket);

    // Create uploads configuration
    const uploads = params.files.map((item, index) => ({
      key: item.key,
      file: `file_${index}`, // form field name
    }));

    formData.append("uploads", JSON.stringify(uploads));

    // Add files to form data
    params.files.forEach((item, index) => {
      formData.append(`file_${index}`, item.file);
    });

    return fetch("/api/objects/upload", {
      method: "POST",
      body: formData,
    });
  },

  // Download operations
  downloadObjects: (params: {
    bucket: string;
    type: "files" | "folder";
    keys?: string[];
    prefix?: string;
  }) => {
    // For downloads, we need to handle binary responses differently
    return fetch("/api/objects/download", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(params),
    });
  },

  // Server shutdown
  shutdown: () => apiCall("shutdown"),
};

export { APIError };
