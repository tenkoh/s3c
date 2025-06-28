import { useToast } from "../contexts/ToastContext";
import { APIError } from "../services/api";

// Simple error display hook without retry functionality
export function useErrorToast() {
  const { showError, showWarning } = useToast();

  const displayError = (error: unknown, customTitle?: string): void => {
    console.error("Error displayed:", error);

    if (error instanceof APIError) {
      const title = customTitle || getErrorTitle(error);
      const message = getErrorMessage(error);

      if (error.severity === "warning") {
        showWarning(title, message);
      } else {
        showError(title, message);
      }
    } else if (error instanceof Error) {
      showError(customTitle || "Unexpected Error", error.message);
    } else {
      showError(customTitle || "Unknown Error", "An unexpected error occurred");
    }
  };

  return { displayError };
}

// Helper functions to extract meaningful information from errors
function getErrorTitle(error: APIError): string {
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
  let message = error.message;

  if (error.suggestion) {
    message += `\n\nSuggestion: ${error.suggestion}`;
  }

  if (error.requestId) {
    message += `\n\nRequest ID: ${error.requestId}`;
  }

  return message;
}