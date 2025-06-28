import type React from "react";
import { useEffect, useState } from "react";
import { api } from "../services/api";

type LayoutProps = {
  children: React.ReactNode;
  onNavigate: (path: string) => void;
};

export function Layout({ children, onNavigate }: LayoutProps) {
  const [connectionStatus, setConnectionStatus] = useState<{
    connected: boolean;
    message: string;
    profile?: string;
    region?: string;
    endpoint?: string;
  }>({
    connected: false,
    message: "Not connected",
  });

  useEffect(() => {
    loadConnectionStatus();
    // Poll connection status every 5 seconds
    const interval = setInterval(loadConnectionStatus, 5000);
    return () => clearInterval(interval);
  }, []);

  async function loadConnectionStatus() {
    try {
      const status = await api.getStatus();
      setConnectionStatus(status);
    } catch (error) {
      setConnectionStatus({
        connected: false,
        message: "Connection check failed",
      });
    }
  }

  function getConnectionDisplay() {
    if (!connectionStatus.connected) {
      return <span className="text-red-600">{connectionStatus.message}</span>;
    }

    const parts = [];
    if (connectionStatus.endpoint) {
      parts.push(connectionStatus.endpoint);
    } else if (connectionStatus.region) {
      parts.push(`AWS ${connectionStatus.region}`);
    }

    if (connectionStatus.profile) {
      parts.push(`(${connectionStatus.profile})`);
    }

    return (
      <span className="text-green-600">
        {parts.length > 0 ? parts.join(" ") : "Connected to S3"}
      </span>
    );
  }

  async function handleShutdown() {
    if (!confirm("Are you sure you want to exit the application?")) {
      return;
    }

    try {
      // Call shutdown API
      await api.shutdown();

      // Close the browser tab/window
      // Note: This may not work in all browsers due to security restrictions
      // but will work when the page was opened programmatically or from local file
      window.close();

      // Fallback: Show a message if window.close() doesn't work
      setTimeout(() => {
        alert("Server has been shut down. You can safely close this tab.");
      }, 500);
    } catch (error) {
      // If shutdown API fails, still allow manual close
      console.error("Shutdown API failed:", error);
      if (confirm("Server shutdown failed. Close tab manually?")) {
        window.close();
      }
    }
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Navigation Bar */}
      <nav className="bg-white shadow-sm border-b">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            {/* Left side - App name and connection info */}
            <div className="flex items-center">
              <h1 className="text-xl font-semibold text-gray-900">s3c</h1>
              <div className="ml-4 text-sm">{getConnectionDisplay()}</div>
            </div>

            {/* Right side - Navigation icons */}
            <div className="flex items-center space-x-4">
              <button
                onClick={() => onNavigate("/")}
                className="p-2 text-gray-400 hover:text-gray-600 transition-colors"
                title="Home"
              >
                <HomeIcon />
              </button>
              <button
                onClick={() => onNavigate("/settings")}
                className="p-2 text-gray-400 hover:text-gray-600 transition-colors"
                title="Settings"
              >
                <SettingsIcon />
              </button>
              <button
                onClick={handleShutdown}
                className="p-2 text-gray-400 hover:text-red-600 transition-colors"
                title="Exit"
              >
                <ExitIcon />
              </button>
            </div>
          </div>
        </div>
      </nav>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">{children}</main>
    </div>
  );
}

// Simple SVG icons
function HomeIcon() {
  return (
    <svg
      className="w-5 h-5"
      fill="none"
      stroke="currentColor"
      viewBox="0 0 24 24"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"
      />
    </svg>
  );
}

function SettingsIcon() {
  return (
    <svg
      className="w-5 h-5"
      fill="none"
      stroke="currentColor"
      viewBox="0 0 24 24"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
      />
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
      />
    </svg>
  );
}

function ExitIcon() {
  return (
    <svg
      className="w-5 h-5"
      fill="none"
      stroke="currentColor"
      viewBox="0 0 24 24"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"
      />
    </svg>
  );
}
