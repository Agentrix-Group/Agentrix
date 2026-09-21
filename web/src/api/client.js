/**
 * HTTP transport for the Agentrix API.
 *
 * - The access token lives only in memory (auth/session.js); the refresh
 *   token is an HttpOnly cookie the browser sends to /api/v1/auth/*.
 * - Requests use `credentials: 'include'` so the cookie also works when the
 *   API is served from another origin allowed by CORS.
 * - A 401 triggers one coordinated refresh and a single retry.
 * - Errors are always ApiError instances built from the API envelope
 *   {"error": {"code", "message", "details", "request_id"}}.
 */
import { getAccessToken, refreshSession, expireSession } from '../auth/session.js';

const API_HOST = (import.meta.env?.VITE_API_URL || '').replace(/\/+$/, '');
export const BASE_URL = `${API_HOST}/api/v1`;
export const CSRF_HEADER = 'X-Requested-With';
export const CSRF_VALUE = 'agentrix';
const DEFAULT_TIMEOUT_MS = 15000;

export class ApiError extends Error {
  constructor({ status = 0, code = 'network_error', message = '', details = null, requestId = null } = {}) {
    super(message || code);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.details = details;
    this.requestId = requestId;
  }

  get isUnauthorized() { return this.status === 401; }
  get isForbidden() { return this.status === 403; }
  get isNotFound() { return this.status === 404; }
  get isConflict() { return this.status === 409; }
  get isValidation() { return this.status === 422 || this.status === 400; }
  get isNetwork() { return this.status === 0; }
}

export function buildQuery(params = {}) {
  const search = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') search.append(key, String(value));
  });
  const text = search.toString();
  return text ? `?${text}` : '';
}

async function toApiError(response) {
  let body = null;
  const text = await response.text();
  if (text) {
    try {
      body = JSON.parse(text);
    } catch {
      body = null;
    }
  }
  const envelope = body?.error;
  return new ApiError({
    status: response.status,
    code: envelope?.code || `http_${response.status}`,
    message: envelope?.message || response.statusText || `HTTP ${response.status}`,
    details: envelope?.details || null,
    requestId: envelope?.request_id || response.headers.get('x-request-id'),
  });
}

function linkSignals(external, timeoutMs) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(new DOMException('timeout', 'TimeoutError')), timeoutMs);
  const onAbort = () => controller.abort(external.reason);
  if (external) {
    if (external.aborted) controller.abort(external.reason);
    else external.addEventListener('abort', onAbort, { once: true });
  }
  return {
    signal: controller.signal,
    cleanup: () => {
      clearTimeout(timer);
      if (external) external.removeEventListener('abort', onAbort);
    },
  };
}

/**
 * request performs one API call.
 * @param {string} path  path below /api/v1 (e.g. "/contests")
 * @param {object} options { method, json, form, headers, signal, timeout, raw, auth }
 */
export async function request(path, options = {}) {
  const { method = 'GET', json, form, signal, timeout = DEFAULT_TIMEOUT_MS, raw = false, retry = true } = options;
  const headers = { Accept: 'application/json', ...(options.headers || {}) };
  const token = getAccessToken();
  if (token && options.auth !== false) headers.Authorization = `Bearer ${token}`;
  let body;
  if (json !== undefined) {
    headers['Content-Type'] = 'application/json';
    body = JSON.stringify(json);
  } else if (form) {
    body = form;
  }
  const { signal: linked, cleanup } = linkSignals(signal, timeout);
  let response;
  try {
    response = await fetch(`${BASE_URL}${path}`, { method, headers, body, signal: linked, credentials: 'include' });
  } catch (err) {
    cleanup();
    if (signal?.aborted) throw err;
    const timedOut = linked.aborted && linked.reason?.name === 'TimeoutError';
    throw new ApiError({ status: 0, code: timedOut ? 'timeout' : 'network_error', message: timedOut ? 'The request timed out' : 'The server could not be reached' });
  }
  cleanup();

  if (response.status === 401 && retry && token && options.auth !== false) {
    const refreshed = await refreshSession();
    if (refreshed) return request(path, { ...options, retry: false });
    expireSession();
  }
  if (!response.ok) throw await toApiError(response);
  if (raw) return response;
  if (response.status === 204) return null;
  const text = await response.text();
  if (!text) return null;
  const type = response.headers.get('content-type') || '';
  if (!type.includes('application/json')) {
    throw new ApiError({ status: response.status, code: 'unexpected_content', message: 'The server returned a non-JSON response' });
  }
  return JSON.parse(text);
}

/** Calls an endpoint authenticated by the refresh cookie (CSRF header). */
export async function cookieRequest(path, { signal } = {}) {
  let response;
  try {
    response = await fetch(`${BASE_URL}${path}`, {
      method: 'POST',
      headers: { Accept: 'application/json', [CSRF_HEADER]: CSRF_VALUE },
      credentials: 'include',
      signal,
    });
  } catch {
    throw new ApiError({ status: 0, code: 'network_error', message: 'The server could not be reached' });
  }
  if (!response.ok) throw await toApiError(response);
  if (response.status === 204) return null;
  return response.json();
}

export function newIdempotencyKey() {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') return crypto.randomUUID();
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('');
}
