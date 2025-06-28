import React, { useEffect } from 'react';
import { useHashRouter, matchRoute } from './hooks/useHashRouter';
import { Layout } from './components/Layout';
import { HomePage } from './pages/HomePage';
import { SettingsPage } from './pages/SettingsPage';
import { ObjectsPage } from './pages/ObjectsPage';
import { UploadPage } from './pages/UploadPage';
import { ToastProvider } from './contexts/ToastContext';
import ToastContainer from './components/ToastContainer';

const App: React.FC = () => {
  const [route, navigate] = useHashRouter();

  // Redirect to settings on first visit if no S3 connection
  useEffect(() => {
    if (route.path === '/') {
      // TODO: Check if S3 is configured and redirect to settings if not
      // For now, always show home page
    }
  }, [route.path]);

  function renderPage() {
    console.log('🧭 Route rendering:', { path: route.path, query: route.query });

    // Match routes and render appropriate page
    if (route.path === '/' || route.path === '') {
      console.log('🏠 Rendering HomePage');
      return <HomePage onNavigate={navigate} />;
    }
    
    if (route.path === '/settings') {
      console.log('⚙️ Rendering SettingsPage');
      return <SettingsPage onNavigate={navigate} />;
    }

    // Bucket listing route: /buckets/:bucket with optional prefix
    const bucketMatch = matchRoute('/buckets/:bucket/*', route.path);
    console.log('🔍 Bucket wildcard match:', { pattern: '/buckets/:bucket/*', result: bucketMatch });
    
    if (bucketMatch) {
      const prefix = bucketMatch['*'] || '';
      console.log('📁 Rendering ObjectsPage with prefix:', { bucket: bucketMatch.bucket, prefix });
      return (
        <ObjectsPage 
          bucket={bucketMatch.bucket!} 
          prefix={prefix}
          onNavigate={navigate} 
        />
      );
    }
    
    // Exact bucket match without prefix
    const exactBucketMatch = matchRoute('/buckets/:bucket', route.path);
    console.log('🔍 Bucket exact match:', { pattern: '/buckets/:bucket', result: exactBucketMatch });
    
    if (exactBucketMatch) {
      console.log('📁 Rendering ObjectsPage without prefix:', { bucket: exactBucketMatch.bucket });
      return (
        <ObjectsPage 
          bucket={exactBucketMatch.bucket!} 
          onNavigate={navigate} 
        />
      );
    }

    // Upload page
    if (route.path === '/upload') {
      return <UploadPage onNavigate={navigate} />;
    }

    // Upload to specific bucket/folder: /upload/:bucket with optional prefix
    const uploadMatch = matchRoute('/upload/:bucket/*', route.path);
    if (uploadMatch) {
      const prefix = uploadMatch['*'] || '';
      return (
        <UploadPage 
          bucket={uploadMatch.bucket}
          prefix={prefix}
          onNavigate={navigate} 
        />
      );
    }

    // Exact upload to bucket match: /upload/:bucket
    const exactUploadMatch = matchRoute('/upload/:bucket', route.path);
    if (exactUploadMatch) {
      return (
        <UploadPage 
          bucket={exactUploadMatch.bucket}
          onNavigate={navigate} 
        />
      );
    }

    // 404 Not Found
    return (
      <div className="text-center py-12">
        <h2 className="text-2xl font-bold text-gray-900 mb-4">Page Not Found</h2>
        <p className="text-gray-600 mb-4">The page you're looking for doesn't exist.</p>
        <button
          onClick={() => navigate('/')}
          className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700"
        >
          Go Home
        </button>
      </div>
    );
  }

  return (
    <ToastProvider>
      <Layout onNavigate={navigate}>
        {renderPage()}
      </Layout>
      <ToastContainer />
    </ToastProvider>
  );
};

export default App;