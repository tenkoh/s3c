// File preview types and utilities

export type PreviewableFileType = 'text' | 'image' | 'none';

export type PreviewFile = {
  key: string;
  size: number;
  type: PreviewableFileType;
  mimeType?: string;
};

// File size limits for preview (to prevent loading large files)
export const PREVIEW_LIMITS = {
  TEXT_MAX_SIZE: 100 * 1024, // 100KB for text files
  IMAGE_MAX_SIZE: 5 * 1024 * 1024, // 5MB for images
} as const;

// Supported file extensions for preview
export const PREVIEWABLE_EXTENSIONS = {
  text: [
    '.txt', '.md', '.json', '.js', '.jsx', '.ts', '.tsx',
    '.css', '.scss', '.html', '.xml', '.yaml', '.yml',
    '.py', '.go', '.java', '.c', '.cpp', '.h', '.hpp',
    '.sh', '.bash', '.zsh', '.ps1', '.sql', '.log',
    '.csv', '.ini', '.conf', '.config', '.env'
  ],
  image: [
    '.jpg', '.jpeg', '.png', '.gif', '.bmp', '.webp', '.svg'
  ]
} as const;

/**
 * Determines if a file can be previewed based on its extension and size
 */
export function getPreviewableType(filename: string, size: number): PreviewableFileType {
  const extension = getFileExtension(filename);
  
  if (PREVIEWABLE_EXTENSIONS.text.includes(extension)) {
    return size <= PREVIEW_LIMITS.TEXT_MAX_SIZE ? 'text' : 'none';
  }
  
  if (PREVIEWABLE_EXTENSIONS.image.includes(extension)) {
    return size <= PREVIEW_LIMITS.IMAGE_MAX_SIZE ? 'image' : 'none';
  }
  
  return 'none';
}

/**
 * Gets file extension from filename (lowercase)
 */
export function getFileExtension(filename: string): string {
  const lastDotIndex = filename.lastIndexOf('.');
  return lastDotIndex !== -1 ? filename.slice(lastDotIndex).toLowerCase() : '';
}

/**
 * Gets MIME type for a file based on its extension
 */
export function getMimeTypeFromExtension(filename: string): string {
  const extension = getFileExtension(filename);
  
  const mimeMap: Record<string, string> = {
    // Text files
    '.txt': 'text/plain',
    '.md': 'text/markdown',
    '.json': 'application/json',
    '.js': 'text/javascript',
    '.jsx': 'text/javascript',
    '.ts': 'text/typescript',
    '.tsx': 'text/typescript',
    '.css': 'text/css',
    '.scss': 'text/scss',
    '.html': 'text/html',
    '.xml': 'text/xml',
    '.yaml': 'text/yaml',
    '.yml': 'text/yaml',
    '.py': 'text/x-python',
    '.go': 'text/x-go',
    '.java': 'text/x-java',
    '.c': 'text/x-c',
    '.cpp': 'text/x-c++',
    '.h': 'text/x-c',
    '.hpp': 'text/x-c++',
    '.sh': 'text/x-shellscript',
    '.bash': 'text/x-shellscript',
    '.zsh': 'text/x-shellscript',
    '.ps1': 'text/x-powershell',
    '.sql': 'text/x-sql',
    '.log': 'text/plain',
    '.csv': 'text/csv',
    '.ini': 'text/plain',
    '.conf': 'text/plain',
    '.config': 'text/plain',
    '.env': 'text/plain',
    
    // Images
    '.jpg': 'image/jpeg',
    '.jpeg': 'image/jpeg',
    '.png': 'image/png',
    '.gif': 'image/gif',
    '.bmp': 'image/bmp',
    '.webp': 'image/webp',
    '.svg': 'image/svg+xml',
  };
  
  return mimeMap[extension] || 'application/octet-stream';
}

/**
 * Formats file size for display
 */
export function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  
  const units = ['B', 'KB', 'MB', 'GB'];
  const k = 1024;
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${units[i]}`;
}