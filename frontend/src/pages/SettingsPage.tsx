import React, { useState, useEffect } from 'react';
import { api, APIError } from '../services/api';
import { useToast } from '../contexts/ToastContext';
import { useErrorHandler } from '../hooks/useErrorHandler';

type Profile = {
  name: string;
};

type SettingsFormData = {
  profile: string;
  region: string;
  endpoint: string;
};

type SettingsPageProps = {
  onNavigate: (path: string) => void;
};

export function SettingsPage({ onNavigate }: SettingsPageProps) {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [formData, setFormData] = useState<SettingsFormData>({
    profile: '',
    region: '',
    endpoint: ''
  });
  const [loading, setLoading] = useState(false);
  const { showSuccess } = useToast();
  const { handleAPIError } = useErrorHandler();

  // Load AWS profiles on component mount
  useEffect(() => {
    loadProfiles();
  }, []);

  async function loadProfiles() {
    try {
      const result = await api.getProfiles();
      setProfiles(result.profiles.map((name: string) => ({ name })));
    } catch (err) {
      if (err instanceof APIError) {
        handleAPIError(err, loadProfiles, 'Failed to Load Profiles');
      } else {
        handleAPIError(new APIError('Failed to connect to server'), loadProfiles, 'Connection Error');
      }
    }
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    
    if (!formData.profile || !formData.region) {
      handleAPIError(new APIError('Profile and region are required'), undefined, 'Validation Error');
      return;
    }

    setLoading(true);

    try {
      await api.configureS3({
        profile: formData.profile,
        region: formData.region,
        endpoint: formData.endpoint || undefined
      });
      
      showSuccess('S3 Connected Successfully', 'Configuration saved and connection established');
      
      // Redirect to home page after successful configuration
      setTimeout(() => onNavigate('/'), 1500);
    } catch (err) {
      if (err instanceof APIError) {
        handleAPIError(err, () => handleSubmit(e), 'S3 Connection Failed');
      } else {
        handleAPIError(new APIError('Failed to connect to server'), () => handleSubmit(e), 'Connection Error');
      }
    } finally {
      setLoading(false);
    }
  }

  function handleInputChange(field: keyof SettingsFormData, value: string) {
    setFormData(prev => ({ ...prev, [field]: value }));
  }

  return (
    <div className="max-w-md mx-auto">
      <div className="bg-white rounded-lg shadow-md p-6">
        <h2 className="text-2xl font-bold text-gray-900 mb-6">S3 Configuration</h2>

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Profile Selection */}
          <div>
            <label htmlFor="profile" className="block text-sm font-medium text-gray-700 mb-1">
              AWS Profile
            </label>
            <select
              id="profile"
              value={formData.profile}
              onChange={(e) => handleInputChange('profile', e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              disabled={loading}
            >
              <option value="">Select a profile...</option>
              {profiles.map((profile) => (
                <option key={profile.name} value={profile.name}>
                  {profile.name}
                </option>
              ))}
            </select>
          </div>

          {/* Region Input */}
          <div>
            <label htmlFor="region" className="block text-sm font-medium text-gray-700 mb-1">
              Region
            </label>
            <input
              type="text"
              id="region"
              value={formData.region}
              onChange={(e) => handleInputChange('region', e.target.value)}
              placeholder="e.g., us-east-1, ap-northeast-1"
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              disabled={loading}
              list="regions"
            />
            <datalist id="regions">
              <option value="us-east-1" />
              <option value="us-west-2" />
              <option value="eu-west-1" />
              <option value="ap-northeast-1" />
              <option value="ap-southeast-1" />
            </datalist>
          </div>

          {/* Endpoint URL (Optional) */}
          <div>
            <label htmlFor="endpoint" className="block text-sm font-medium text-gray-700 mb-1">
              Endpoint URL (Optional)
            </label>
            <input
              type="url"
              id="endpoint"
              value={formData.endpoint}
              onChange={(e) => handleInputChange('endpoint', e.target.value)}
              placeholder="https://s3.example.com (leave empty for AWS)"
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              disabled={loading}
            />
          </div>

          {/* Submit Button */}
          <button
            type="submit"
            disabled={loading || !formData.profile || !formData.region}
            className="w-full bg-blue-600 text-white py-2 px-4 rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {loading ? 'Connecting...' : 'Connect to S3'}
          </button>
        </form>

        <div className="mt-6 text-sm text-gray-600">
          <p>
            <strong>Note:</strong> Configuration is stored in memory only and will be lost when the application restarts.
          </p>
        </div>
      </div>
    </div>
  );
}