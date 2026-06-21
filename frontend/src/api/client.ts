const BASE_URL = import.meta.env.DEV ? (import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080') : '';

interface RequestOptions {
  method: string;
  body?: unknown;
  headers?: Record<string, string>;
}

function isBodyInit(body: unknown): body is BodyInit {
  if (
    typeof body === 'string' ||
    body instanceof FormData ||
    body instanceof Blob ||
    body instanceof URLSearchParams ||
    body instanceof ArrayBuffer ||
    ArrayBuffer.isView(body)
  ) {
    return true;
  }

  return typeof ReadableStream !== 'undefined' && body instanceof ReadableStream;
}

export class ApiError extends Error {
  status: number;
  constructor(message: string, status: number) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

/**
 * Determines whether the error is a network/offline failure.
 */
export function isNetworkError(err: unknown): boolean {
  if (!(err instanceof Error)) return false;
  // fetch throws TypeError for network failures
  if (err instanceof TypeError && err.message.includes('Failed to fetch')) return true;
  if (err.name === 'AbortError') return true;
  return false;
}

/**
 * Returns a user-friendly error message based on the error type.
 */
export function classifyError(err: unknown, t: (key: string) => string): string {
  if (err instanceof ApiError) {
    if (err.status === 401) return t('common.auth.expired');
    if (err.status >= 500) return t('common.error.server');
    return err.message;
  }
  if (isNetworkError(err)) return t('common.error.network');
  return err instanceof Error ? err.message : t('common.error.unknown');
}

export async function request(path: string, options: RequestOptions = { method: 'GET' }) {
  const url = `${BASE_URL}${path}`;
  const headers: Record<string, string> = { ...options.headers };
  let body: BodyInit | undefined;

  if (options.body !== undefined && options.body !== null) {
    if (isBodyInit(options.body)) {
      body = options.body;
    } else {
      headers['Content-Type'] = 'application/json';
      body = JSON.stringify(options.body);
    }
  }

  // Get token from localStorage
  const token = localStorage.getItem('sharelink_token');
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  try {
    const response = await fetch(url, {
      method: options.method,
      headers,
      body,
    });

    if (response.status === 401) {
      localStorage.removeItem('sharelink_token');
      // Redirect to login via SPA-compatible path change (no hard refresh)
      if (!window.location.pathname.endsWith('/login')) {
        window.history.pushState({}, '', '/login');
        window.dispatchEvent(new Event('sharelink_pushstate'));
      }
      throw new ApiError('Session expired', 401);
    }

    // Safely parse JSON only when content-type is JSON-like
    const contentType = response.headers.get('content-type') || '';
    let resData: Record<string, unknown>;
    if (contentType.includes('application/json')) {
      resData = await response.json() as Record<string, unknown>;
    } else {
      // Non-JSON response (e.g. HTML error page from proxy/CDN)
      if (!response.ok) {
        throw new ApiError(
          response.statusText || `HTTP ${response.status}`,
          response.status
        );
      }
      throw new ApiError('Unexpected non-JSON response', response.status);
    }

    if (!response.ok) {
      const msg = (resData.error as { message?: string })?.message || `HTTP ${response.status}`;
      throw new ApiError(msg, response.status);
    }

    if (!(resData.success as boolean)) {
      const msg = ((resData.error as { message?: string })?.message) || 'Request failed';
      throw new ApiError(msg, response.status);
    }

    return resData.data as any;
  } catch (error: unknown) {
    console.error(`API Error: ${url}`, error);
    throw error;
  }
}

export function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error ? error.message : fallback;
}

export const api = {
  get: (path: string) => request(path, { method: 'GET' }),
  post: (path: string, body?: unknown) => request(path, { method: 'POST', body }),
  put: (path: string, body?: unknown) => request(path, { method: 'PUT', body }),
  delete: (path: string) => request(path, { method: 'DELETE' }),
};
