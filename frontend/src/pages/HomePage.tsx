import React, { useState, useEffect } from 'react';
import { api, APIError } from '../services/api';

type Bucket = {
  name: string;
};

type HomePageProps = {
  onNavigate: (path: string) => void;
};

export function HomePage({ onNavigate }: HomePageProps) {
  const [buckets, setBuckets] = useState<Bucket[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isConnected, setIsConnected] = useState(false);

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
        setError(err.message);
      } else {
        setError('Failed to connect to server');
      }
    }
  }

  async function loadBuckets() {
    setLoading(true);
    setError(null);

    try {
      const result = await api.listBuckets();
      setBuckets(result.buckets.map((name: string) => ({ name })));
      setIsConnected(true);
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
        if (err.message.includes('not configured')) {
          setIsConnected(false);
        }
      } else {
        setError('Failed to connect to server');
        setIsConnected(false);
      }
    } finally {
      setLoading(false);
    }
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
        <h2 className="text-2xl font-bold text-gray-900">Buckets</h2>
        <p className="text-gray-600">Select a bucket to browse its contents</p>
      </div>

      {error && (
        <div className="mb-4 p-3 bg-red-100 border border-red-400 text-red-700 rounded">
          {error}
          <button
            onClick={loadBuckets}
            className="ml-2 text-red-800 underline hover:no-underline"
          >
            Retry
          </button>
        </div>
      )}

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
          <p className="text-gray-600">You don't have any buckets in this account.</p>
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
    </div>
  );
}