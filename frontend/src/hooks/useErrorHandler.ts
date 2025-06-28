import { useToast } from "../contexts/ToastContext";
import { APIError } from "../services/api";

export const useErrorHandler = () => {
  const { showError, showWarning } = useToast();

  const handleError = (error: unknown, customTitle?: string) => {
    console.error("Error handled:", error);

    if (error instanceof APIError) {
      // Use structured error information
      const title = customTitle || getErrorTitle(error);
      const message = getErrorMessage(error);

      // Show appropriate toast based on severity
      if (error.severity === "warning") {
        showWarning(title, message);
      } else {
        showError(title, message, {
          retryable: error.retryable,
          // Note: onRetry would need to be provided by the calling component
        });
      }
    } else if (error instanceof Error) {
      // Generic error
      showError(customTitle || "Unexpected Error", error.message);
    } else {
      // Unknown error type
      showError(customTitle || "Unknown Error", "An unexpected error occurred");
    }
  };

  const handleAPIError = (
    error: APIError,
    retryFn?: () => void,
    customTitle?: string,
  ) => {
    console.error("API Error handled:", error);

    const title = customTitle || getErrorTitle(error);
    const message = getErrorMessage(error);

    // Show appropriate toast based on severity
    if (error.severity === "warning") {
      showWarning(title, message);
    } else {
      showError(title, message, {
        retryable: error.retryable && !!retryFn,
        onRetry: retryFn,
      });
    }
  };

  return { handleError, handleAPIError };
};

// Helper functions to extract meaningful information from errors
function getErrorTitle(error: APIError): string {
  // Use category and code to create user-friendly titles
  switch (error.category) {
    case "validation":
      return "Invalid Input";
    case "s3":
      return "S3 Operation Failed";
    case "config":
      return "Configuration Error";
    case "network":
      return "Network Error";
    case "internal":
      return "Internal Error";
    default:
      return "Error";
  }
}

function getErrorMessage(error: APIError): string {
  // Combine error message with suggestion if available
  let message = error.message;

  if (error.suggestion) {
    message += `\n\nSuggestion: ${error.suggestion}`;
  }

  // Add request ID for debugging if available
  if (error.requestId) {
    message += `\n\nRequest ID: ${error.requestId}`;
  }

  return message;
}
