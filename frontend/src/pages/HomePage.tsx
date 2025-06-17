import React, { useState, useEffect } from 'react';
import { api, APIError } from '../services/api';
import { useErrorHandler } from '../hooks/useErrorHandler';
import { useToast } from '../contexts/ToastContext';

type Bucket = {
  name: string;
};

type HomePageProps = {
  onNavigate: (path: string) => void;
};

export function HomePage({ onNavigate }: HomePageProps) {
  const [buckets, setBuckets] = useState<Bucket[]>([]);
  const [loading, setLoading] = useState(false);
  const [isConnected, setIsConnected] = useState(false);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [bucketName, setBucketName] = useState('');
  const [creating, setCreating] = useState(false);
  const { handleAPIError } = useErrorHandler();
  const { showSuccess } = useToast();

  useEffect(() => {
    checkConnection();
  }, []);

  async function checkConnection() {
    try {
      await api.health();
      loadBuckets();
    } catch (err) {
      setIsConnected(false);
      if (err instanceof APIError) {
        handleAPIError(err, checkConnection, 'Health Check Failed');
      } else {
        handleAPIError(new APIError('Failed to connect to server'), checkConnection, 'Connection Error');
      }
    }
  }

  async function loadBuckets() {
    setLoading(true);

    try {
      const result = await api.listBuckets();
      setBuckets(result.buckets.map((name: string) => ({ name })));
      setIsConnected(true);
    } catch (err) {
      if (err instanceof APIError) {
        if (err.message.includes('not configured')) {
          setIsConnected(false);
        }
        handleAPIError(err, loadBuckets, 'Failed to Load Buckets');
      } else {
        setIsConnected(false);
        handleAPIError(new APIError('Failed to connect to server'), loadBuckets, 'Connection Error');
      }
    } finally {
      setLoading(false);
    }
  }

  async function handleCreateBucket() {
    if (!bucketName.trim()) {
      return;
    }

    setCreating(true);
    try {
      await api.createBucket(bucketName.trim());
      
      // Success - close modal and refresh bucket list
      setShowCreateModal(false);
      setBucketName('');
      await loadBuckets(); // Refresh the bucket list
      
      // Show success message
      showSuccess('Bucket Created', `Bucket "${bucketName.trim()}" has been created successfully`)
    } catch (err) {
      if (err instanceof APIError) {
        handleAPIError(err, handleCreateBucket, 'Failed to Create Bucket');
      } else {
        handleAPIError(new APIError('Failed to create bucket'), handleCreateBucket, 'Bucket Creation Error');
      }
    } finally {
      setCreating(false);
    }
  }

  function handleCloseModal() {
    setShowCreateModal(false);
    setBucketName('');
  }

  if (!isConnected) {
    return (
      <div className="text-center py-12">
        <div className="max-w-md mx-auto bg-white rounded-lg shadow-md p-6">
          <div className="mb-4">
            <svg className="mx-auto h-12 w-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
            </svg>
          </div>
          <h3 className="text-lg font-medium text-gray-900 mb-2">S3 Not Connected</h3>
          <p className="text-gray-600 mb-4">
            You need to configure your S3 connection before you can access buckets and objects.
          </p>
          <button
            onClick={() => onNavigate('/settings')}
            className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 transition-colors"
          >
            Configure S3 Connection
          </button>
        </div>
      </div>
    );
  }

  return (
    <div>
      <div className="mb-6">
        <div className="flex justify-between items-center">
          <div>
            <h2 className="text-2xl font-bold text-gray-900">Buckets</h2>
            <p className="text-gray-600">Select a bucket to browse its contents</p>
          </div>
          <button
            onClick={() => setShowCreateModal(true)}
            className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 transition-colors flex items-center"
          >
            <svg className="h-4 w-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
            </svg>
            Create Bucket
          </button>
        </div>
      </div>


      {loading ? (
        <div className="text-center py-8">
          <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          <p className="mt-2 text-gray-600">Loading buckets...</p>
        </div>
      ) : buckets.length === 0 ? (
        <div className="text-center py-8">
          <svg className="mx-auto h-12 w-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2 2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
          </svg>
          <h3 className="mt-2 text-lg font-medium text-gray-900">No buckets found</h3>
          <p className="text-gray-600 mb-4">You don't have any buckets in this account.</p>
          <button
            onClick={() => setShowCreateModal(true)}
            className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 transition-colors"
          >
            Create Your First Bucket
          </button>
        </div>
      ) : (
        <div className="bg-white shadow rounded-lg overflow-hidden">
          <ul className="divide-y divide-gray-200">
            {buckets.map((bucket) => (
              <li key={bucket.name}>
                <button
                  onClick={() => onNavigate(`/buckets/${encodeURIComponent(bucket.name)}`)}
                  className="w-full px-6 py-4 text-left hover:bg-gray-50 focus:outline-none focus:bg-gray-50 transition-colors"
                >
                  <div className="flex items-center">
                    <svg className="h-5 w-5 text-gray-400 mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2 2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
                    </svg>
                    <span className="text-lg font-medium text-gray-900">{bucket.name}</span>
                  </div>
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* Create Bucket Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-md mx-4">
            <h3 className="text-lg font-medium text-gray-900 mb-4">Create New Bucket</h3>
            
            <div className="mb-4">
              <label htmlFor="bucketName" className="block text-sm font-medium text-gray-700 mb-2">
                Bucket Name
              </label>
              <input
                type="text"
                id="bucketName"
                value={bucketName}
                onChange={(e) => setBucketName(e.target.value)}
                placeholder="my-bucket-name"
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                disabled={creating}
              />
              <p className="mt-1 text-xs text-gray-500">
                Bucket names must be 3-63 characters, lowercase letters, numbers, dots, and hyphens only
              </p>
            </div>

            <div className="flex justify-end space-x-3">
              <button
                onClick={handleCloseModal}
                className="px-4 py-2 text-gray-700 bg-gray-200 rounded-md hover:bg-gray-300 transition-colors"
                disabled={creating}
              >
                Cancel
              </button>
              <button
                onClick={handleCreateBucket}
                disabled={creating || !bucketName.trim()}
                className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors flex items-center"
              >
                {creating && (
                  <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                )}
                {creating ? 'Creating...' : 'Create Bucket'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}