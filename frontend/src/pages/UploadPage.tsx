import React, { useState, useRef } from 'react';
import { api, APIError } from '../services/api';
import { useToast } from '../contexts/ToastContext';
import { useErrorHandler } from '../hooks/useErrorHandler';

type UploadFile = {
  file: File;
  key: string;
  status: 'pending' | 'uploading' | 'success' | 'error';
  error?: string;
};

type UploadPageProps = {
  bucket?: string;
  prefix?: string;
  onNavigate: (path: string) => void;
};

export function UploadPage({ bucket, prefix = '', onNavigate }: UploadPageProps) {
  const [selectedFiles, setSelectedFiles] = useState<UploadFile[]>([]);
  const [isUploading, setIsUploading] = useState(false);
  const [isDragOver, setIsDragOver] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // If no bucket is provided via props, we need to collect it
  const [targetBucket, setTargetBucket] = useState(bucket || '');
  const [targetPrefix, setTargetPrefix] = useState(prefix);
  
  const { showSuccess, showWarning } = useToast();
  const { handleAPIError } = useErrorHandler();

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragOver(true);
  };

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragOver(false);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragOver(false);
    
    const files = Array.from(e.dataTransfer.files);
    addFiles(files);
  };

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files) {
      const files = Array.from(e.target.files);
      addFiles(files);
    }
  };

  const addFiles = (files: File[]) => {
    const newFiles: UploadFile[] = files.map(file => ({
      file,
      key: targetPrefix ? `${targetPrefix}${file.name}` : file.name,
      status: 'pending'
    }));
    
    setSelectedFiles(prev => [...prev, ...newFiles]);
  };

  const removeFile = (index: number) => {
    setSelectedFiles(prev => prev.filter((_, i) => i !== index));
  };

  const updateFileKey = (index: number, newKey: string) => {
    setSelectedFiles(prev => prev.map((file, i) => 
      i === index ? { ...file, key: newKey } : file
    ));
  };

  const handleUpload = async () => {
    if (!targetBucket) {
      handleAPIError(new APIError('Please specify a bucket'), undefined, 'Validation Error');
      return;
    }

    if (selectedFiles.length === 0) {
      handleAPIError(new APIError('Please select files to upload'), undefined, 'Validation Error');
      return;
    }

    setIsUploading(true);

    // Reset all file statuses
    setSelectedFiles(prev => prev.map(file => ({ ...file, status: 'uploading', error: undefined })));

    try {
      const response = await api.uploadObjects({
        bucket: targetBucket,
        files: selectedFiles.map(f => ({ file: f.file, key: f.key }))
      });

      if (response.ok) {
        const result = await response.json();
        
        if (result.success) {
          // Mark uploaded files as success
          const uploadedKeys = new Set(result.data.uploaded?.map((u: any) => u.key) || []);
          
          setSelectedFiles(prev => prev.map(file => ({
            ...file,
            status: uploadedKeys.has(file.key) ? 'success' : 'error',
            error: uploadedKeys.has(file.key) ? undefined : 'Upload failed'
          })));

          // Show success message
          if (result.data.success === result.data.total) {
            showSuccess('Upload Complete', `Successfully uploaded ${result.data.success} file(s)`);
          } else {
            showWarning('Partial Upload', `Uploaded ${result.data.success} of ${result.data.total} files`);
          }
          
          // Navigate back to bucket/folder after successful upload
          if (result.data.success > 0) {
            const backPath = targetPrefix 
              ? `/buckets/${encodeURIComponent(targetBucket)}/${encodeURIComponent(targetPrefix)}`
              : `/buckets/${encodeURIComponent(targetBucket)}`;
            setTimeout(() => onNavigate(backPath), 1500);
          }
        } else {
          throw new Error(result.error || 'Upload failed');
        }
      } else {
        throw new Error(`Upload failed: ${response.status}`);
      }
    } catch (err) {
      console.error('Upload error:', err);
      
      // Mark all files as error
      setSelectedFiles(prev => prev.map(file => ({
        ...file,
        status: 'error',
        error: err instanceof Error ? err.message : 'Upload failed'
      })));
      
      if (err instanceof APIError) {
        handleAPIError(err, handleUpload, 'Upload Failed');
      } else {
        handleAPIError(new APIError('Upload failed. Please try again.'), handleUpload, 'Upload Error');
      }
    } finally {
      setIsUploading(false);
    }
  };

  const formatFileSize = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'pending':
        return <div className="w-4 h-4 border-2 border-gray-300 rounded-full"></div>;
      case 'uploading':
        return <div className="w-4 h-4 border-2 border-blue-600 border-t-transparent rounded-full animate-spin"></div>;
      case 'success':
        return (
          <svg className="w-4 h-4 text-green-600" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
          </svg>
        );
      case 'error':
        return (
          <svg className="w-4 h-4 text-red-600" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
          </svg>
        );
      default:
        return null;
    }
  };

  return (
    <div className="max-w-4xl mx-auto p-6">
      <div className="mb-6">
        <h1 className="text-3xl font-bold text-gray-900 mb-2">Upload Files</h1>
        <p className="text-gray-600">Upload files to your S3 bucket</p>
      </div>

      {/* Bucket and Prefix Configuration */}
      <div className="bg-white rounded-lg shadow p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Destination</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Bucket *
            </label>
            <input
              type="text"
              value={targetBucket}
              onChange={(e) => setTargetBucket(e.target.value)}
              placeholder="my-bucket"
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              disabled={!!bucket} // Disable if bucket is provided via props
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Folder Prefix (optional)
            </label>
            <input
              type="text"
              value={targetPrefix}
              onChange={(e) => setTargetPrefix(e.target.value)}
              placeholder="folder/"
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
        </div>
      </div>

      {/* File Drop Zone */}
      <div
        className={`border-2 border-dashed rounded-lg p-8 text-center mb-6 transition-colors ${
          isDragOver
            ? 'border-blue-500 bg-blue-50'
            : 'border-gray-300 hover:border-gray-400'
        }`}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
      >
        <div className="mb-4">
          <svg
            className="mx-auto h-12 w-12 text-gray-400"
            stroke="currentColor"
            fill="none"
            viewBox="0 0 48 48"
          >
            <path
              d="M28 8H12a4 4 0 00-4 4v20m32-12v8m0 0v8a4 4 0 01-4 4H12a4 4 0 01-4-4v-4m32-4l-3.172-3.172a4 4 0 00-5.656 0L28 28M8 32l9.172-9.172a4 4 0 015.656 0L28 28m0 0l4 4m4-24h8m-4-4v8m-12 4h.02"
              strokeWidth={2}
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        </div>
        <p className="text-lg font-medium text-gray-900 mb-2">
          Drop files here to upload
        </p>
        <p className="text-gray-600 mb-4">or</p>
        <button
          onClick={() => fileInputRef.current?.click()}
          className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 transition-colors"
        >
          Select Files
        </button>
        <input
          ref={fileInputRef}
          type="file"
          multiple
          onChange={handleFileSelect}
          className="hidden"
        />
      </div>

      {/* Selected Files List */}
      {selectedFiles.length > 0 && (
        <div className="bg-white rounded-lg shadow mb-6">
          <div className="px-6 py-4 border-b border-gray-200">
            <h2 className="text-lg font-semibold">Selected Files ({selectedFiles.length})</h2>
          </div>
          <div className="divide-y divide-gray-200">
            {selectedFiles.map((uploadFile, index) => (
              <div key={index} className="px-6 py-4 flex items-center justify-between">
                <div className="flex items-center space-x-3 flex-1">
                  {getStatusIcon(uploadFile.status)}
                  <div className="flex-1">
                    <div className="flex items-center space-x-2">
                      <span className="font-medium text-gray-900">
                        {uploadFile.file.name}
                      </span>
                      <span className="text-sm text-gray-500">
                        ({formatFileSize(uploadFile.file.size)})
                      </span>
                    </div>
                    <div className="mt-1">
                      <label className="block text-xs text-gray-500 mb-1">S3 Key:</label>
                      <input
                        type="text"
                        value={uploadFile.key}
                        onChange={(e) => updateFileKey(index, e.target.value)}
                        className="text-sm text-gray-700 border border-gray-200 rounded px-2 py-1 w-full max-w-md"
                        disabled={uploadFile.status === 'uploading'}
                      />
                    </div>
                    {uploadFile.error && (
                      <p className="text-sm text-red-600 mt-1">{uploadFile.error}</p>
                    )}
                  </div>
                </div>
                {uploadFile.status === 'pending' && (
                  <button
                    onClick={() => removeFile(index)}
                    className="text-red-600 hover:text-red-800 transition-colors"
                  >
                    <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
                    </svg>
                  </button>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Action Buttons */}
      <div className="flex items-center justify-between">
        <button
          onClick={() => onNavigate('/')}
          className="px-4 py-2 text-gray-600 hover:text-gray-800 transition-colors"
        >
          ← Back to Home
        </button>
        
        <div className="flex space-x-3">
          <button
            onClick={() => setSelectedFiles([])}
            disabled={isUploading || selectedFiles.length === 0}
            className="px-4 py-2 border border-gray-300 text-gray-700 rounded-md hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            Clear All
          </button>
          <button
            onClick={handleUpload}
            disabled={isUploading || selectedFiles.length === 0 || !targetBucket}
            className="px-6 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {isUploading ? 'Uploading...' : `Upload ${selectedFiles.length} File${selectedFiles.length !== 1 ? 's' : ''}`}
          </button>
        </div>
      </div>
    </div>
  );
}