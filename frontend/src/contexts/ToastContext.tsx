import type React from "react";
import {
  createContext,
  type ReactNode,
  useContext,
  useState,
} from "react";

export type ToastType = "success" | "error" | "warning" | "info";

export type Toast = {
  id: string;
  type: ToastType;
  title: string;
  message?: string;
  duration?: number;
  retryable?: boolean;
  onRetry?: () => void;
};

type ToastContextType = {
  toasts: Toast[];
  addToast: (toast: Omit<Toast, "id">) => void;
  removeToast: (id: string) => void;
  showSuccess: (title: string, message?: string) => void;
  showError: (
    title: string,
    message?: string,
    options?: { retryable?: boolean; onRetry?: () => void },
  ) => void;
  showWarning: (title: string, message?: string) => void;
  showInfo: (title: string, message?: string) => void;
};

const ToastContext = createContext<ToastContextType | undefined>(undefined);

export const useToast = () => {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error("useToast must be used within a ToastProvider");
  }
  return context;
};

type ToastProviderProps = {
  children: ReactNode;
};

export const ToastProvider: React.FC<ToastProviderProps> = ({ children }) => {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const generateId = () => Math.random().toString(36).substring(2, 11);

  const removeToast = (id: string) => {
    setToasts((prev) => prev.filter((toast) => toast.id !== id));
  };

  const addToast = (toast: Omit<Toast, "id">) => {
    const newToast: Toast = {
      ...toast,
      id: generateId(),
      duration: toast.duration ?? (toast.type === "error" ? 8000 : 5000), // Errors stay longer
    };

    setToasts((prev) => [...prev, newToast]);

    // Auto-remove toast after duration
    if (newToast.duration && newToast.duration > 0) {
      setTimeout(() => {
        removeToast(newToast.id);
      }, newToast.duration);
    }
  };

  const showSuccess = (title: string, message?: string) => {
    addToast({ type: "success", title, message });
  };

  const showError = (
    title: string,
    message?: string,
    options?: { retryable?: boolean; onRetry?: () => void },
  ) => {
    addToast({
      type: "error",
      title,
      message,
      retryable: options?.retryable,
      onRetry: options?.onRetry,
    });
  };

  const showWarning = (title: string, message?: string) => {
    addToast({ type: "warning", title, message });
  };

  const showInfo = (title: string, message?: string) => {
    addToast({ type: "info", title, message });
  };

  const value: ToastContextType = {
    toasts,
    addToast,
    removeToast,
    showSuccess,
    showError,
    showWarning,
    showInfo,
  };

  return (
    <ToastContext.Provider value={value}>{children}</ToastContext.Provider>
  );
};
