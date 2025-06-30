import type React from "react";
import { useCallback, useEffect, useState } from "react";
import { useErrorHandler } from "../hooks/useErrorHandler";
import { api } from "../services/api";
import { formatFileSize, type PreviewFile } from "../types/preview";

type PreviewModalProps = {
  isOpen: boolean;
  onClose: () => void;
  file: PreviewFile;
  bucket: string;
};

const PreviewModal: React.FC<PreviewModalProps> = ({
  isOpen,
  onClose,
  file,
  bucket,
}) => {
  const [content, setContent] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [imageError, setImageError] = useState(false);
  const { handleAPIError } = useErrorHandler();

  const loadFileContent = useCallback(async () => {
    setLoading(true);
    setContent(null);
    setImageError(false);

    try {
      const response = await api.downloadObjects({
        bucket,
        type: "files",
        keys: [file.key],
      });

      if (!response.ok) {
        throw new Error(`Failed to download file: ${response.status}`);
      }

      if (file.type === "text") {
        const text = await response.text();
        setContent(text);
      } else if (file.type === "image") {
        const blob = await response.blob();
        const objectURL = URL.createObjectURL(blob);
        setContent(objectURL);
      }
    } catch (err) {
      handleAPIError(
        err instanceof Error
          ? new Error(err.message)
          : new Error("Failed to load file content"),
        loadFileContent,
        "Preview Error",
      );
    } finally {
      setLoading(false);
    }
  }, [file.key, file.type, bucket, handleAPIError]);

  useEffect(() => {
    if (isOpen && file.type !== "none") {
      loadFileContent();
    }

    return () => {
      setContent(null);
      setImageError(false);
    };
  }, [isOpen, file.type, loadFileContent]);

  // Handle ESC key to close modal
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
      }
    };

    if (isOpen) {
      document.addEventListener("keydown", handleEscape);
      document.body.style.overflow = "hidden"; // Prevent background scrolling
    }

    return () => {
      document.removeEventListener("keydown", handleEscape);
      document.body.style.overflow = "unset";
    };
  }, [isOpen, onClose]);

  const handleImageError = () => {
    setImageError(true);
    setLoading(false);
  };

  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4"
      onClick={handleBackdropClick}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          handleBackdropClick(e as unknown as React.MouseEvent);
        }
      }}
      role="dialog"
      aria-modal="true"
    >
      <div className="bg-white rounded-lg shadow-xl max-w-6xl w-full max-h-[90vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b">
          <div className="flex-1 min-w-0">
            <h2 className="text-lg font-semibold text-gray-900 truncate">
              {file.key.split("/").pop()}
            </h2>
            <p className="text-sm text-gray-600">
              {formatFileSize(file.size)} • {file.type} file
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="ml-4 text-gray-400 hover:text-gray-600 focus:outline-none"
          >
            <svg
              className="w-6 h-6"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              aria-label="Close"
            >
              <title>Close</title>
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-auto">
          {loading ? (
            <div className="flex items-center justify-center h-64">
              <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
              <span className="ml-2 text-gray-600">Loading preview...</span>
            </div>
          ) : file.type === "text" && content ? (
            <TextPreview content={content} filename={file.key} />
          ) : file.type === "image" && content && !imageError ? (
            <ImagePreview
              src={content}
              filename={file.key}
              onError={handleImageError}
            />
          ) : file.type === "image" && imageError ? (
            <div className="flex items-center justify-center h-64 text-gray-500">
              <div className="text-center">
                <svg
                  className="w-16 h-16 mx-auto mb-4 text-gray-300"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                  aria-label="Image error"
                >
                  <title>Image error</title>
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
                  />
                </svg>
                <p>Failed to load image</p>
                <button
                  type="button"
                  onClick={loadFileContent}
                  className="mt-2 text-blue-600 hover:text-blue-800 underline"
                >
                  Try again
                </button>
              </div>
            </div>
          ) : (
            <div className="flex items-center justify-center h-64 text-gray-500">
              <div className="text-center">
                <svg
                  className="w-16 h-16 mx-auto mb-4 text-gray-300"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                  aria-label="File preview unavailable"
                >
                  <title>File preview unavailable</title>
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
                  />
                </svg>
                <p>Preview not available for this file type</p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

// Text preview component with syntax highlighting
const TextPreview: React.FC<{ content: string; filename: string }> = ({
  content,
  filename,
}) => {
  const getLanguageFromFilename = (filename: string): string => {
    const extension = filename.split(".").pop()?.toLowerCase();
    const languageMap: Record<string, string> = {
      js: "javascript",
      jsx: "javascript",
      ts: "typescript",
      tsx: "typescript",
      py: "python",
      go: "go",
      java: "java",
      c: "c",
      cpp: "cpp",
      h: "c",
      hpp: "cpp",
      css: "css",
      scss: "scss",
      html: "html",
      xml: "xml",
      json: "json",
      yaml: "yaml",
      yml: "yaml",
      sh: "bash",
      bash: "bash",
      zsh: "bash",
      ps1: "powershell",
      sql: "sql",
      md: "markdown",
    };

    return languageMap[extension || ""] || "text";
  };

  const language = getLanguageFromFilename(filename);

  return (
    <div className="p-4">
      <pre className="bg-gray-50 rounded-lg p-4 overflow-auto text-sm font-mono whitespace-pre-wrap border">
        <code className={`language-${language}`}>{content}</code>
      </pre>
    </div>
  );
};

// Image preview component with zoom capabilities
const ImagePreview: React.FC<{
  src: string;
  filename: string;
  onError: () => void;
}> = ({ src, filename, onError }) => {
  const [zoom, setZoom] = useState(1);
  const [position, setPosition] = useState({ x: 0, y: 0 });
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });

  const handleZoomIn = () => setZoom((prev) => Math.min(prev * 1.5, 5));
  const handleZoomOut = () => setZoom((prev) => Math.max(prev / 1.5, 0.1));
  const handleResetZoom = () => {
    setZoom(1);
    setPosition({ x: 0, y: 0 });
  };

  const handleMouseDown = (e: React.MouseEvent) => {
    if (zoom > 1) {
      setIsDragging(true);
      setDragStart({ x: e.clientX - position.x, y: e.clientY - position.y });
    }
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    if (isDragging) {
      setPosition({
        x: e.clientX - dragStart.x,
        y: e.clientY - dragStart.y,
      });
    }
  };

  const handleMouseUp = () => {
    setIsDragging(false);
  };

  return (
    <div className="relative">
      {/* Zoom controls */}
      <div className="absolute top-4 right-4 z-10 bg-white rounded-lg shadow-md p-2 flex space-x-2">
        <button
          type="button"
          onClick={handleZoomOut}
          className="p-1 hover:bg-gray-100 rounded"
          title="Zoom out"
        >
          <svg
            className="w-5 h-5"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            aria-label="Zoom out"
          >
            <title>Zoom out</title>
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0zM13 10H7"
            />
          </svg>
        </button>
        <button
          type="button"
          onClick={handleResetZoom}
          className="px-2 py-1 text-sm hover:bg-gray-100 rounded"
          title="Reset zoom"
        >
          {Math.round(zoom * 100)}%
        </button>
        <button
          type="button"
          onClick={handleZoomIn}
          className="p-1 hover:bg-gray-100 rounded"
          title="Zoom in"
        >
          <svg
            className="w-5 h-5"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            aria-label="Zoom in"
          >
            <title>Zoom in</title>
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0zM10 7v3m0 0v3m0-3h3m-3 0H7"
            />
          </svg>
        </button>
      </div>

      {/* Image container */}
      <div
        className="flex items-center justify-center min-h-[60vh] overflow-hidden cursor-move"
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={handleMouseUp}
        onMouseLeave={handleMouseUp}
        role="img"
        aria-label="Image preview with zoom and pan controls"
      >
        <img
          src={src}
          alt={filename}
          onError={onError}
          className="max-w-none"
          style={{
            transform: `scale(${zoom}) translate(${position.x / zoom}px, ${position.y / zoom}px)`,
            transition: isDragging ? "none" : "transform 0.1s ease-out",
          }}
        />
      </div>
    </div>
  );
};

export default PreviewModal;
