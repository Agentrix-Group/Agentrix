/**
 * Robust API Client for Agentrix Backend
 * Complies with ADR-0009 and resolves F0.3 / F0.4.
 */

import {
  getAccessToken,
  getRefreshToken,
  refreshAuthTokens,
  notifySessionExpired,
} from '../auth/session.js';

const API_HOST = (import.meta.env?.VITE_API_URL || '').replace(/\/+$/, '');
export const BASE_URL = `${API_HOST}/api/v1`;
const DEFAULT_TIMEOUT_MS = 10000;

export class ApiClientError extends Error {
  constructor(message, { status, statusText, data = {}, correlationId = null, url = '' } = {}) {
    super(message || (data && data.message) || `HTTP ${status}: ${statusText}`);
    this.name = 'ApiClientError';
    this.status = status;
    this.statusText = statusText;
    this.data = data || {};
    this.correlationId = correlationId || data?.correlation_id || data?.correlationId || null;
    this.url = url;
    this.errorCode = data?.errorCode || data?.error_code || null;
    this.isAuth = status === 401;
    this.isForbidden = status === 403;
    this.isNotFound = status === 404;
    this.isConflict = status === 409;
    this.isValidation = status === 422 || (status === 400 && Boolean(data?.errors || data?.details));
    this.isServer = status >= 500;
  }

  getUserMessage(t) {
    if (this.errorCode && typeof t === 'function') {
      const translation = t(`errors:codes.${this.errorCode}`, { defaultValue: null });
      if (translation && translation !== `errors:codes.${this.errorCode}`) {
        return translation;
      }
    }
    return this.data?.message || this.message || 'Error de comunicación con el servidor';
  }
}

export class NetworkError extends Error {
  constructor(message, originalError = null) {
    super(message || 'No se pudo establecer conexión con el servidor. Verifique su red.');
    this.name = 'NetworkError';
    this.isNetworkError = true;
    this.code = 'NETWORK_ERROR';
    this.originalError = originalError;
  }
}

export class TimeoutError extends Error {
  constructor(timeoutMs) {
    super(`La solicitud excedió el tiempo límite de espera (${timeoutMs}ms).`);
    this.name = 'TimeoutError';
    this.isTimeout = true;
    this.code = 'TIMEOUT_ERROR';
    this.timeoutMs = timeoutMs;
  }
}

export function buildUrl(endpoint, queryParams = {}) {
  const cleanEndpoint = endpoint.startsWith('/') ? endpoint : `/${endpoint}`;
  const keys = Object.keys(queryParams).filter(
    (k) => queryParams[k] !== undefined && queryParams[k] !== null && queryParams[k] !== ''
  );
  if (keys.length === 0) {
    return cleanEndpoint;
  }
  const searchParams = new URLSearchParams();
  for (const key of keys) {
    searchParams.append(key, String(queryParams[key]));
  }
  return `${cleanEndpoint}?${searchParams.toString()}`;
}

export async function request(endpoint, options = {}) {
  const token = getAccessToken();
  const timeoutMs = options.timeout ?? DEFAULT_TIMEOUT_MS;
  const method = (options.method || 'GET').toUpperCase();

  const headers = {
    Accept: 'application/json, text/plain, */*',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...options.headers,
  };

  // Do NOT add Content-Type: application/json for GET or HEAD requests (F0.4)
  if (method !== 'GET' && method !== 'HEAD' && options.body && !(options.body instanceof FormData)) {
    if (!headers['Content-Type']) {
      headers['Content-Type'] = 'application/json';
    }
  }

  const controller = new AbortController();
  let timedOut = false;
  const timer = setTimeout(() => {
    timedOut = true;
    controller.abort();
  }, timeoutMs);

  // Link external signal if provided
  if (options.signal) {
    options.signal.addEventListener('abort', () => controller.abort());
  }

  const fullUrl = `${BASE_URL}${endpoint}`;

  try {
    const response = await fetch(fullUrl, {
      ...options,
      method,
      headers,
      signal: controller.signal,
    });

    clearTimeout(timer);

    const correlationId =
      response.headers.get('x-correlation-id') ||
      response.headers.get('x-request-id') ||
      null;

    // Handle 401 with coordinated refresh token renewal (F0.3)
    if (
      response.status === 401 &&
      !options._retry &&
      !endpoint.includes('/auth/login') &&
      !endpoint.includes('/auth/refresh')
    ) {
      const refreshToken = getRefreshToken();
      if (refreshToken) {
        try {
          const newTokens = await refreshAuthTokens(BASE_URL);
          if (newTokens?.access_token) {
            return await request(endpoint, {
              ...options,
              _retry: true,
              headers: {
                ...options.headers,
                Authorization: `Bearer ${newTokens.access_token}`,
              },
            });
          }
        } catch {
          // Refresh failed; notify session expired and proceed with original 401 error
          notifySessionExpired();
        }
      }
    }

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new ApiClientError(errorData.message || response.statusText, {
        status: response.status,
        statusText: response.statusText,
        data: errorData,
        correlationId,
        url: fullUrl,
      });
    }

    // Handle 204 No Content or empty responses gracefully
    if (response.status === 204) {
      return null;
    }

    const contentType = response.headers.get('content-type') || '';
    if (contentType.includes('application/json')) {
      return await response.json().catch(() => null);
    }
    const text = await response.text();
    return text ? JSON.parse(text).catch(() => text) : null;
  } catch (err) {
    clearTimeout(timer);
    if (err instanceof ApiClientError) {
      throw err;
    }
    if (timedOut || err.name === 'AbortError') {
      if (timedOut) {
        throw new TimeoutError(timeoutMs);
      }
      throw err;
    }
    throw new NetworkError(err.message, err);
  }
}

export async function requestText(endpoint, options = {}) {
  const token = getAccessToken();
  const timeoutMs = options.timeout ?? DEFAULT_TIMEOUT_MS;
  const controller = new AbortController();
  let timedOut = false;
  const timer = setTimeout(() => {
    timedOut = true;
    controller.abort();
  }, timeoutMs);

  const fullUrl = `${BASE_URL}${endpoint}`;
  try {
    const response = await fetch(fullUrl, {
      ...options,
      headers: {
        Accept: 'application/x-ndjson, text/plain, */*',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...options.headers,
      },
      signal: controller.signal,
    });
    clearTimeout(timer);

    if (
      response.status === 401 &&
      !options._retry &&
      !endpoint.includes('/auth/login') &&
      !endpoint.includes('/auth/refresh')
    ) {
      const refreshToken = getRefreshToken();
      if (refreshToken) {
        try {
          const newTokens = await refreshAuthTokens(BASE_URL);
          if (newTokens?.access_token) {
            return await requestText(endpoint, {
              ...options,
              _retry: true,
              headers: {
                ...options.headers,
                Authorization: `Bearer ${newTokens.access_token}`,
              },
            });
          }
        } catch {
          notifySessionExpired();
        }
      }
    }

    if (!response.ok) {
      const correlationId =
        response.headers.get('x-correlation-id') ||
        response.headers.get('x-request-id') ||
        null;
      throw new ApiClientError(`HTTP ${response.status}: ${response.statusText}`, {
        status: response.status,
        statusText: response.statusText,
        correlationId,
        url: fullUrl,
      });
    }
    return await response.text();
  } catch (err) {
    clearTimeout(timer);
    if (err instanceof ApiClientError) throw err;
    if (timedOut || err.name === 'AbortError') {
      if (timedOut) throw new TimeoutError(timeoutMs);
      throw err;
    }
    throw new NetworkError(err.message, err);
  }
}

export async function requestForm(endpoint, formData, options = {}) {
  const token = getAccessToken();
  const timeoutMs = options.timeout ?? DEFAULT_TIMEOUT_MS * 3; // 30s for uploads
  const controller = new AbortController();
  let timedOut = false;
  const timer = setTimeout(() => {
    timedOut = true;
    controller.abort();
  }, timeoutMs);

  const fullUrl = `${BASE_URL}${endpoint}`;
  try {
    const response = await fetch(fullUrl, {
      ...options,
      method: 'POST',
      body: formData,
      headers: {
        Accept: 'application/json, text/plain, */*',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...options.headers,
      },
      signal: controller.signal,
    });
    clearTimeout(timer);

    if (
      response.status === 401 &&
      !options._retry &&
      !endpoint.includes('/auth/login') &&
      !endpoint.includes('/auth/refresh')
    ) {
      const refreshToken = getRefreshToken();
      if (refreshToken) {
        try {
          const newTokens = await refreshAuthTokens(BASE_URL);
          if (newTokens?.access_token) {
            return await requestForm(endpoint, formData, {
              ...options,
              _retry: true,
              headers: {
                ...options.headers,
                Authorization: `Bearer ${newTokens.access_token}`,
              },
            });
          }
        } catch {
          notifySessionExpired();
        }
      }
    }

    const correlationId =
      response.headers.get('x-correlation-id') ||
      response.headers.get('x-request-id') ||
      null;

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new ApiClientError(errorData.message || response.statusText, {
        status: response.status,
        statusText: response.statusText,
        data: errorData,
        correlationId,
        url: fullUrl,
      });
    }
    return await response.json().catch(() => null);
  } catch (err) {
    clearTimeout(timer);
    if (err instanceof ApiClientError) throw err;
    if (timedOut || err.name === 'AbortError') {
      if (timedOut) throw new TimeoutError(timeoutMs);
      throw err;
    }
    throw new NetworkError(err.message, err);
  }
}

export const api = {
  get: (url, options = {}) => request(url, { ...options, method: 'GET' }),
  post: (url, body, options = {}) =>
    request(url, { ...options, method: 'POST', body: JSON.stringify(body) }),
  put: (url, body, options = {}) =>
    request(url, { ...options, method: 'PUT', body: JSON.stringify(body) }),
  patch: (url, body, options = {}) =>
    request(url, { ...options, method: 'PATCH', body: body ? JSON.stringify(body) : undefined }),
  delete: (url, options = {}) => request(url, { ...options, method: 'DELETE' }),
  text: (url, options = {}) => requestText(url, options),
  form: (url, data, options = {}) => requestForm(url, data, options),
};
