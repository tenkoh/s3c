// API client for POST-unified endpoints

type APIResponse<T = any> = {
  success: boolean;
  data?: T;
  error?: string;
};

class APIError extends Error {
  constructor(message: string, public status?: number) {
    super(message);
    this.name = 'APIError';
  }
}

async function apiCall<T>(endpoint: string, data: any = {}): Promise<T> {
  try {
    const response = await fetch(`/api/${endpoint}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(data),
    });

    const result: APIResponse<T> = await response.json();

    if (!result.success) {
      throw new APIError(result.error || 'API call failed', response.status);
    }

    return result.data as T;
  } catch (error) {
    if (error instanceof APIError) {
      throw error;
    }
    throw new APIError('Network error or server unavailable');
  }
}

// API endpoints
export const api = {
  // Health check
  health: () => apiCall('health'),

  // Profile management
  getProfiles: (): Promise<{ profiles: string[] }> => 
    apiCall('profiles'),

  // S3 configuration
  configureS3: (config: { profile: string; region: string; endpoint?: string }) =>
    apiCall('settings', config),

  // Bucket operations
  listBuckets: (): Promise<{ buckets: string[] }> =>
    apiCall('buckets'),

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
  }> => apiCall('objects/list', params),

  deleteObjects: (params: { bucket: string; keys: string[] }) =>
    apiCall('objects/delete', params),

  // Upload operations
  uploadObjects: (params: {
    bucket: string;
    files: { file: File; key: string }[];
    onProgress?: (progress: number) => void;
  }) => {
    const formData = new FormData();
    
    // Add bucket parameter
    formData.append('bucket', params.bucket);
    
    // Create uploads configuration
    const uploads = params.files.map((item, index) => ({
      key: item.key,
      file: `file_${index}`, // form field name
    }));
    
    formData.append('uploads', JSON.stringify(uploads));
    
    // Add files to form data
    params.files.forEach((item, index) => {
      formData.append(`file_${index}`, item.file);
    });

    return fetch('/api/objects/upload', {
      method: 'POST',
      body: formData,
    });
  },

  // Download operations
  downloadObjects: (params: {
    bucket: string;
    type: 'files' | 'folder';
    keys?: string[];
    prefix?: string;
  }) => {
    // For downloads, we need to handle binary responses differently
    return fetch('/api/objects/download', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(params),
    });
  },

  // Server shutdown
  shutdown: () => apiCall('shutdown'),
};

export { APIError };