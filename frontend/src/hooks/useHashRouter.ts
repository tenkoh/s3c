import { useState, useEffect } from 'react';

export type RouteParams = {
  [key: string]: string;
};

export type ParsedRoute = {
  path: string;
  params: RouteParams;
  query: URLSearchParams;
};

/**
 * Custom hook for hash-based routing without React Router
 * Handles URLs like: #/buckets/my-bucket?page=2
 */
export function useHashRouter(): [ParsedRoute, (path: string) => void] {
  const [route, setRoute] = useState<ParsedRoute>(() => parseHash());

  function parseHash(): ParsedRoute {
    const hash = window.location.hash.slice(1) || '/';
    const [pathPart, queryPart] = hash.split('?');
    
    return {
      path: pathPart || '/',
      params: {},
      query: new URLSearchParams(queryPart || '')
    };
  }

  function navigate(path: string) {
    window.location.hash = path;
  }

  useEffect(() => {
    function handleHashChange() {
      setRoute(parseHash());
    }

    window.addEventListener('hashchange', handleHashChange);
    return () => window.removeEventListener('hashchange', handleHashChange);
  }, []);

  return [route, navigate];
}

/**
 * Helper to match route patterns like '/buckets/:bucket' or '/buckets/:bucket/*'
 */
export function matchRoute(pattern: string, path: string): RouteParams | null {
  const patternParts = pattern.split('/');
  const pathParts = path.split('/');

  // Handle wildcard patterns (e.g., '/buckets/:bucket/*')
  const hasWildcard = pattern.endsWith('/*');
  
  if (hasWildcard) {
    // For wildcard patterns, path must be at least as long as pattern (minus wildcard)
    if (pathParts.length < patternParts.length - 1) {
      return null;
    }
  } else {
    // For exact patterns, lengths must match
    if (patternParts.length !== pathParts.length) {
      return null;
    }
  }

  const params: RouteParams = {};

  // Process parts up to wildcard (or all parts if no wildcard)
  const partsToProcess = hasWildcard ? patternParts.length - 1 : patternParts.length;

  for (let i = 0; i < partsToProcess; i++) {
    const patternPart = patternParts[i];
    const pathPart = pathParts[i];

    if (patternPart?.startsWith(':')) {
      // Parameter part
      const paramName = patternPart.slice(1);
      params[paramName] = decodeURIComponent(pathPart || '');
    } else if (patternPart !== pathPart) {
      // Static part doesn't match
      return null;
    }
  }

  // Handle wildcard part - capture remaining path
  if (hasWildcard && pathParts.length > partsToProcess) {
    const remainingPath = pathParts.slice(partsToProcess).join('/');
    params['*'] = decodeURIComponent(remainingPath);
  }

  return params;
}