import { useEffect, useReducer } from "react";
import { APIError, api } from "../services/api";

export type Bucket = {
  name: string;
};

// Union type for bucket state management
export type BucketState =
  | { status: "idle" }
  | { status: "loading" }
  | { status: "success"; buckets: Bucket[] }
  | { status: "error"; error: APIError }
  | { status: "disconnected" };

// Actions for state management
type BucketAction =
  | { type: "START_LOADING" }
  | { type: "LOAD_SUCCESS"; buckets: Bucket[] }
  | { type: "LOAD_ERROR"; error: APIError }
  | { type: "SET_DISCONNECTED" }
  | { type: "START_CREATE_BUCKET" }
  | { type: "CREATE_BUCKET_SUCCESS"; bucketName: string }
  | { type: "CREATE_BUCKET_ERROR"; error: APIError };

function bucketReducer(state: BucketState, action: BucketAction): BucketState {
  switch (action.type) {
    case "START_LOADING":
      return { status: "loading" };

    case "LOAD_SUCCESS":
      return { status: "success", buckets: action.buckets };

    case "LOAD_ERROR":
      return { status: "error", error: action.error };

    case "SET_DISCONNECTED":
      return { status: "disconnected" };

    case "START_CREATE_BUCKET":
      // Keep current state but could be extended for optimistic updates
      return state;

    case "CREATE_BUCKET_SUCCESS":
      // Add new bucket to existing list if in success state
      if (state.status === "success") {
        return {
          status: "success",
          buckets: [...state.buckets, { name: action.bucketName }].sort((a, b) =>
            a.name.localeCompare(b.name)
          ),
        };
      }
      return state;

    case "CREATE_BUCKET_ERROR":
      return { status: "error", error: action.error };

    default:
      return state;
  }
}

export interface UseBucketsReturn {
  state: BucketState;
  actions: {
    refresh: () => Promise<void>;
    createBucket: (name: string) => Promise<void>;
  };
}

export function useBuckets(): UseBucketsReturn {
  const [state, dispatch] = useReducer(bucketReducer, { status: "idle" });

  const loadBuckets = async (): Promise<void> => {
    dispatch({ type: "START_LOADING" });

    try {
      // First check health
      await api.health();

      // Then load buckets
      const result = await api.listBuckets();
      const buckets = result.buckets.map((name: string) => ({ name }));
      dispatch({ type: "LOAD_SUCCESS", buckets });
    } catch (err) {
      if (err instanceof APIError) {
        if (err.message.includes("not configured")) {
          dispatch({ type: "SET_DISCONNECTED" });
        } else {
          dispatch({ type: "LOAD_ERROR", error: err });
        }
      } else {
        dispatch({
          type: "LOAD_ERROR",
          error: new APIError("Failed to connect to server"),
        });
      }
    }
  };

  const createBucket = async (name: string): Promise<void> => {
    dispatch({ type: "START_CREATE_BUCKET" });

    try {
      await api.createBucket(name);
      dispatch({ type: "CREATE_BUCKET_SUCCESS", bucketName: name });
    } catch (err) {
      if (err instanceof APIError) {
        dispatch({ type: "CREATE_BUCKET_ERROR", error: err });
      } else {
        dispatch({
          type: "CREATE_BUCKET_ERROR",
          error: new APIError("Failed to create bucket"),
        });
      }
      throw err; // Re-throw to allow caller to handle UI feedback
    }
  };

  // Simple useEffect with no complex dependencies
  useEffect(() => {
    loadBuckets();
  }, []);

  return {
    state,
    actions: {
      refresh: loadBuckets,
      createBucket,
    },
  };
}