import { useEffect, useState } from "react";
import CreateFolderModal from "../components/CreateFolderModal";
import PreviewModal from "../components/PreviewModal";
import { useToast } from "../contexts/ToastContext";
import { useErrorHandler } from "../hooks/useErrorHandler";
import { APIError, api } from "../services/api";
import {
  formatFileSize,
  getPreviewableType,
  type PreviewFile,
} from "../types/preview";

type S3Object = {
  key: string;
  size: number;
  lastModified: string;
  isFolder: boolean;
};

type ObjectsPageProps = {
  bucket: string;
  prefix?: string;
  onNavigate: (path: string) => void;
};

export function ObjectsPage({
  bucket,
  prefix = "",
  onNavigate,
}: ObjectsPageProps) {
  const [objects, setObjects] = useState<S3Object[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedKeys, setSelectedKeys] = useState<string[]>([]);
  const [, setContinuationToken] = useState<string>("");
  const [previewFile, setPreviewFile] = useState<PreviewFile | null>(null);
  const [isPreviewOpen, setIsPreviewOpen] = useState(false);
  const [isCreateFolderOpen, setIsCreateFolderOpen] = useState(false);
  const { showSuccess } = useToast();
  const { handleAPIError } = useErrorHandler();

  useEffect(() => {
    loadObjects();
  }, [bucket, prefix]);

  async function loadObjects() {
    setLoading(true);

    try {
      const result = await api.listObjects({
        bucket,
        prefix,
        delimiter: "/",
        maxKeys: 100,
      });

      setObjects(result.objects || []);
      setContinuationToken(result.nextContinuationToken || "");
    } catch (err) {
      if (err instanceof APIError) {
        handleAPIError(err, loadObjects, "Failed to Load Objects");
      } else {
        handleAPIError(
          new APIError("Failed to load objects"),
          loadObjects,
          "Loading Error",
        );
      }
    } finally {
      setLoading(false);
    }
  }

  function handleSelectObject(key: string, isFolder: boolean) {
    if (isFolder) {
      // Folders can only be selected individually
      setSelectedKeys(selectedKeys.includes(key) ? [] : [key]);
    } else {
      // Files can be multi-selected
      setSelectedKeys((prev) =>
        prev.includes(key)
          ? prev.filter((k) => k !== key)
          : [
              ...prev.filter(
                (k) => !objects.find((o) => o.key === k)?.isFolder,
              ),
              key,
            ],
      );
    }
  }

  function handleObjectClick(obj: S3Object) {
    if (obj.isFolder) {
      // Navigate into folder - always add trailing slash for folders
      const newPrefix = obj.key + "/";
      const newUrl = `/buckets/${encodeURIComponent(bucket)}/${encodeURIComponent(newPrefix)}`;
      onNavigate(newUrl);
    }
  }

  async function handleDownload() {
    if (selectedKeys.length === 0) return;

    try {
      const selectedObjects = objects.filter((o) =>
        selectedKeys.includes(o.key),
      );
      const hasFolder = selectedObjects.some((o) => o.isFolder);

      if (hasFolder && selectedKeys.length > 1) {
        handleAPIError(
          new APIError("Cannot download folders with other items"),
          undefined,
          "Download Error",
        );
        return;
      }

      const response = await api.downloadObjects({
        bucket,
        type: hasFolder ? "folder" : "files",
        keys: hasFolder ? undefined : selectedKeys,
        prefix: hasFolder ? selectedKeys[0] + "/" : undefined,
      });

      if (response.ok) {
        // Create download link
        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;

        // Get filename from Content-Disposition header or set default
        let filename: string;
        if (hasFolder) {
          filename = `${selectedKeys[0]?.replace("/", "") || "folder"}.zip`;
        } else if (selectedKeys.length > 1) {
          filename = "download.zip";
        } else {
          // Single file: extract filename from Content-Disposition header
          const contentDisposition = response.headers.get(
            "Content-Disposition",
          );
          if (contentDisposition) {
            filename =
              extractFilenameFromContentDisposition(contentDisposition) ||
              selectedKeys[0]?.split("/").pop() ||
              "download";
          } else {
            // Fallback to key basename
            filename = selectedKeys[0]?.split("/").pop() || "download";
          }
        }

        a.download = filename;
        a.click();
        window.URL.revokeObjectURL(url);
      } else {
        handleAPIError(
          new APIError("Download failed"),
          handleDownload,
          "Download Error",
        );
      }
    } catch (err) {
      if (err instanceof APIError) {
        handleAPIError(err, handleDownload, "Download Failed");
      } else {
        handleAPIError(
          new APIError("Download failed"),
          handleDownload,
          "Download Error",
        );
      }
    }
  }

  async function handleDelete() {
    if (selectedKeys.length === 0) return;

    if (
      !confirm(
        `Are you sure you want to delete ${selectedKeys.length} item(s)?`,
      )
    ) {
      return;
    }

    try {
      await api.deleteObjects({
        bucket,
        keys: selectedKeys,
      });

      showSuccess(
        "Objects Deleted",
        `Successfully deleted ${selectedKeys.length} item(s)`,
      );
      setSelectedKeys([]);
      loadObjects(); // Reload list
    } catch (err) {
      if (err instanceof APIError) {
        handleAPIError(err, handleDelete, "Delete Failed");
      } else {
        handleAPIError(
          new APIError("Delete failed"),
          handleDelete,
          "Delete Error",
        );
      }
    }
  }

  function handlePreview(obj: S3Object) {
    const previewType = getPreviewableType(obj.key, obj.size);

    if (previewType === "none") {
      handleAPIError(
        new APIError(`Preview not available for this file type or size`),
        undefined,
        "Preview Not Available",
      );
      return;
    }

    const previewFile: PreviewFile = {
      key: obj.key,
      size: obj.size,
      type: previewType,
    };

    setPreviewFile(previewFile);
    setIsPreviewOpen(true);
  }

  function closePreview() {
    setIsPreviewOpen(false);
    setPreviewFile(null);
  }

  function openCreateFolder() {
    setIsCreateFolderOpen(true);
  }

  function closeCreateFolder() {
    setIsCreateFolderOpen(false);
  }

  function handleFolderCreateSuccess() {
    loadObjects(); // Reload the objects list to show the new folder
  }

  function getParentPath() {
    if (!prefix) return "/";
    const parts = prefix.replace(/\/$/, "").split("/");
    parts.pop();
    return parts.length > 0
      ? `/buckets/${encodeURIComponent(bucket)}/${encodeURIComponent(parts.join("/") + "/")}`
      : `/buckets/${encodeURIComponent(bucket)}`;
  }

  function extractFilenameFromContentDisposition(
    contentDisposition: string,
  ): string | null {
    // First, try to extract RFC 5987 format: filename*=UTF-8''encoded-filename
    const rfc5987Match = contentDisposition.match(/filename\*=UTF-8''([^;]+)/);
    if (rfc5987Match?.[1]) {
      try {
        // URL decode the filename
        return decodeURIComponent(rfc5987Match[1]);
      } catch (e) {
        // Fall through to legacy format
      }
    }

    // Fallback to legacy format: filename="filename"
    const legacyMatch = contentDisposition.match(/filename="([^"]+)"/);
    if (legacyMatch?.[1]) {
      return legacyMatch[1];
    }

    return null;
  }

  return (
    <div>
      {/* Breadcrumb and Actions */}
      <div className="mb-6">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-2xl font-bold text-gray-900">
              {bucket}
              {prefix && ` / ${prefix.replace(/\/$/, "")}`}
            </h2>
            <p className="text-gray-600">{objects.length} items</p>
          </div>
        </div>

        {/* Action Row */}
        <div className="mt-4 flex items-center justify-between">
          {/* Back button */}
          <button
            onClick={() => onNavigate(getParentPath())}
            className="flex items-center px-3 py-2 text-gray-600 hover:text-gray-900 transition-colors"
          >
            <svg
              className="w-4 h-4 mr-1"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M15 19l-7-7 7-7"
              />
            </svg>
            Back
          </button>

          {/* Action buttons */}
          <div className="flex space-x-2">
            <button
              onClick={openCreateFolder}
              className="px-4 py-2 bg-purple-600 text-white rounded-md hover:bg-purple-700 transition-colors"
            >
              Create Folder
            </button>
            <button
              onClick={() => {
                const uploadPath = prefix
                  ? `/upload/${encodeURIComponent(bucket)}/${encodeURIComponent(prefix)}`
                  : `/upload/${encodeURIComponent(bucket)}`;
                onNavigate(uploadPath);
              }}
              className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
            >
              Upload
            </button>
            <button
              onClick={handleDownload}
              disabled={selectedKeys.length === 0}
              className="px-4 py-2 bg-green-600 text-white rounded-md hover:bg-green-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              Download ({selectedKeys.length})
            </button>
            <button
              onClick={handleDelete}
              disabled={selectedKeys.length === 0}
              className="px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              Delete ({selectedKeys.length})
            </button>
          </div>
        </div>
      </div>

      {/* Error Display */}

      {/* Objects List */}
      {loading ? (
        <div className="text-center py-8">
          <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          <p className="mt-2 text-gray-600">Loading objects...</p>
        </div>
      ) : objects.length === 0 ? (
        <div className="text-center py-8">
          <svg
            className="mx-auto h-12 w-12 text-gray-400"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z"
            />
          </svg>
          <h3 className="mt-2 text-lg font-medium text-gray-900">
            No objects found
          </h3>
          <p className="text-gray-600">This bucket or folder is empty.</p>
        </div>
      ) : (
        <div className="bg-white shadow rounded-lg overflow-hidden">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="w-8 px-6 py-3"></th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Name
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Size
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Modified
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {objects.map((obj) => (
                <tr key={obj.key} className="hover:bg-gray-50">
                  <td className="px-6 py-4">
                    <input
                      type="checkbox"
                      checked={selectedKeys.includes(obj.key)}
                      onChange={() => handleSelectObject(obj.key, obj.isFolder)}
                      className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                    />
                  </td>
                  <td className="px-6 py-4">
                    <button
                      onClick={() => handleObjectClick(obj)}
                      className="flex items-center text-left hover:text-blue-600 transition-colors"
                    >
                      {obj.isFolder ? (
                        <svg
                          className="h-5 w-5 text-blue-600 mr-2"
                          fill="currentColor"
                          viewBox="0 0 20 20"
                        >
                          <path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
                        </svg>
                      ) : (
                        <svg
                          className="h-5 w-5 text-gray-600 mr-2"
                          fill="currentColor"
                          viewBox="0 0 20 20"
                        >
                          <path
                            fillRule="evenodd"
                            d="M4 4a2 2 0 012-2h4.586A2 2 0 0112 2.586L15.414 6A2 2 0 0116 7.414V16a2 2 0 01-2 2H6a2 2 0 01-2-2V4z"
                            clipRule="evenodd"
                          />
                        </svg>
                      )}
                      <span className="text-sm font-medium text-gray-900">
                        {obj.key.split("/").pop() || obj.key}
                      </span>
                    </button>
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-500">
                    {obj.isFolder ? "-" : formatFileSize(obj.size)}
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-500">
                    {obj.isFolder
                      ? "-"
                      : new Date(obj.lastModified).toLocaleDateString()}
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-500">
                    {!obj.isFolder &&
                      getPreviewableType(obj.key, obj.size) !== "none" && (
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handlePreview(obj);
                          }}
                          className="text-blue-600 hover:text-blue-800 font-medium"
                          title="Preview file"
                        >
                          Preview
                        </button>
                      )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Preview Modal */}
      {previewFile && (
        <PreviewModal
          isOpen={isPreviewOpen}
          onClose={closePreview}
          file={previewFile}
          bucket={bucket}
        />
      )}

      {/* Create Folder Modal */}
      <CreateFolderModal
        isOpen={isCreateFolderOpen}
        onClose={closeCreateFolder}
        bucket={bucket}
        currentPrefix={prefix}
        onSuccess={handleFolderCreateSuccess}
      />
    </div>
  );
}
