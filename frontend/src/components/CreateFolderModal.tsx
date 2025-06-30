import type React from "react";
import { useEffect, useState } from "react";
import { useToast } from "../contexts/ToastContext";
import { useErrorHandler } from "../hooks/useErrorHandler";
import { APIError, api } from "../services/api";

type CreateFolderModalProps = {
  isOpen: boolean;
  onClose: () => void;
  bucket: string;
  currentPrefix?: string;
  onSuccess: () => void;
};

const CreateFolderModal: React.FC<CreateFolderModalProps> = ({
  isOpen,
  onClose,
  bucket,
  currentPrefix = "",
  onSuccess,
}) => {
  const [folderName, setFolderName] = useState("");
  const [isCreating, setIsCreating] = useState(false);
  const { showSuccess } = useToast();
  const { handleAPIError } = useErrorHandler();

  // Reset form when modal opens/closes
  useEffect(() => {
    if (isOpen) {
      setFolderName("");
      setIsCreating(false);
    }
  }, [isOpen]);

  // Handle ESC key to close modal
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape" && !isCreating) {
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
  }, [isOpen, isCreating, onClose]);

  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget && !isCreating) {
      onClose();
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!folderName.trim()) {
      handleAPIError(
        new APIError("Folder name cannot be empty"),
        undefined,
        "Invalid Input",
      );
      return;
    }

    setIsCreating(true);

    try {
      // Construct the full prefix for the folder
      const fullPrefix = currentPrefix
        ? `${currentPrefix.replace(/\/+$/, "")}/${folderName.trim()}`
        : folderName.trim();

      await api.createFolder(bucket, fullPrefix);

      showSuccess(
        "Folder Created",
        `Successfully created folder "${folderName.trim()}"`,
      );
      onSuccess(); // Trigger parent to reload objects
      onClose(); // Close modal
    } catch (err) {
      if (err instanceof APIError) {
        handleAPIError(err, () => handleSubmit(e), "Create Folder Failed");
      } else {
        handleAPIError(
          new APIError("Failed to create folder"),
          () => handleSubmit(e),
          "Create Folder Error",
        );
      }
    } finally {
      setIsCreating(false);
    }
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    // Allow alphanumeric, spaces, hyphens, underscores, and basic punctuation
    // Prevent forward slashes to avoid nested path confusion in input
    const sanitized = value.replace(/[/\\]/g, "");
    setFolderName(sanitized);
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
      <div className="bg-white rounded-lg shadow-xl max-w-md w-full">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b">
          <h2 className="text-lg font-semibold text-gray-900">
            Create New Folder
          </h2>
          <button
            type="button"
            onClick={onClose}
            disabled={isCreating}
            className="text-gray-400 hover:text-gray-600 focus:outline-none disabled:opacity-50"
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
        <form onSubmit={handleSubmit} className="p-6">
          <div className="mb-4">
            <label
              htmlFor="folderName"
              className="block text-sm font-medium text-gray-700 mb-2"
            >
              Folder Name
            </label>
            <input
              type="text"
              id="folderName"
              value={folderName}
              onChange={handleInputChange}
              placeholder="Enter folder name"
              disabled={isCreating}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:bg-gray-100 disabled:cursor-not-allowed"
              maxLength={255}
            />
            <p className="mt-1 text-xs text-gray-500">
              Folder will be created{" "}
              {currentPrefix
                ? `in "${currentPrefix}"`
                : "in the root of the bucket"}
            </p>
          </div>

          {/* Footer */}
          <div className="flex justify-end space-x-3">
            <button
              type="button"
              onClick={onClose}
              disabled={isCreating}
              className="px-4 py-2 text-gray-700 bg-gray-100 hover:bg-gray-200 rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isCreating || !folderName.trim()}
              className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {isCreating ? (
                <span className="flex items-center">
                  <svg
                    className="animate-spin -ml-1 mr-2 h-4 w-4 text-white"
                    fill="none"
                    viewBox="0 0 24 24"
                    aria-label="Loading"
                  >
                    <title>Loading</title>
                    <circle
                      className="opacity-25"
                      cx="12"
                      cy="12"
                      r="10"
                      stroke="currentColor"
                      strokeWidth="4"
                    ></circle>
                    <path
                      className="opacity-75"
                      fill="currentColor"
                      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                    ></path>
                  </svg>
                  Creating...
                </span>
              ) : (
                "Create Folder"
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default CreateFolderModal;
