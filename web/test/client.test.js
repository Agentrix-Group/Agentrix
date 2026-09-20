import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  api,
  request,
  buildUrl,
  ApiClientError,
  NetworkError,
  TimeoutError,
} from '../src/api/client.js';

describe('Central HTTP Client (ADR-0009 / F0.4)', () => {
  const originalFetch = global.fetch;

  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    global.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  it('buildUrl handles path and query parameters safely', () => {
    expect(buildUrl('/contests')).toBe('/contests');
    expect(buildUrl('matches')).toBe('/matches');
    expect(buildUrl('/matches', { contest_id: 'c-1', limit: 10 })).toBe('/matches?contest_id=c-1&limit=10');
    expect(buildUrl('/agents', { owner_user_id: 'user 1 & 2' })).toBe('/agents?owner_user_id=user+1+%26+2');
    expect(buildUrl('/contests', { empty: '', undef: undefined, nul: null })).toBe('/contests');
  });

  it('does NOT attach Content-Type: application/json on GET requests', async () => {
    let capturedHeaders = null;
    global.fetch = vi.fn().mockImplementation((url, options) => {
      capturedHeaders = options.headers;
      return Promise.resolve({
        ok: true,
        status: 200,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: () => Promise.resolve({ success: true }),
      });
    });

    await api.get('/contests');
    expect(capturedHeaders['Content-Type']).toBeUndefined();
    expect(capturedHeaders['Accept']).toContain('application/json');
  });

  it('attaches Content-Type: application/json on POST requests with JSON payload', async () => {
    let capturedHeaders = null;
    let capturedBody = null;
    global.fetch = vi.fn().mockImplementation((url, options) => {
      capturedHeaders = options.headers;
      capturedBody = options.body;
      return Promise.resolve({
        ok: true,
        status: 201,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: () => Promise.resolve({ id: '1' }),
      });
    });

    await api.post('/contests', { name: 'Test Contest' });
    expect(capturedHeaders['Content-Type']).toBe('application/json');
    expect(capturedBody).toBe('{"name":"Test Contest"}');
  });

  it('attaches Authorization header when token exists in localStorage', async () => {
    localStorage.setItem('agentrix_token', 'valid-jwt-token-123');
    let capturedHeaders = null;
    global.fetch = vi.fn().mockImplementation((url, options) => {
      capturedHeaders = options.headers;
      return Promise.resolve({
        ok: true,
        status: 200,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: () => Promise.resolve({ user: 'tester' }),
      });
    });

    await api.get('/me');
    expect(capturedHeaders['Authorization']).toBe('Bearer valid-jwt-token-123');
  });

  it('handles 204 No Content gracefully without failing on response.json()', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 204,
      headers: new Headers(),
    });

    const result = await api.delete('/matches/m-123');
    expect(result).toBeNull();
  });

  it('extracts correlation ID from response headers upon API failure', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 404,
      statusText: 'Not Found',
      headers: new Headers({
        'content-type': 'application/json',
        'x-correlation-id': 'req-corr-98765',
      }),
      json: () => Promise.resolve({ errorCode: 'RESOURCE_NOT_FOUND', message: 'Match not found' }),
    });

    try {
      await api.get('/matches/missing-id');
      expect.fail('Should have thrown ApiClientError');
    } catch (err) {
      expect(err).toBeInstanceOf(ApiClientError);
      expect(err.status).toBe(404);
      expect(err.isNotFound).toBe(true);
      expect(err.correlationId).toBe('req-corr-98765');
      expect(err.errorCode).toBe('RESOURCE_NOT_FOUND');
    }
  });

  it('correctly classifies authentication and forbidden errors', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 401,
      statusText: 'Unauthorized',
      headers: new Headers(),
      json: () => Promise.resolve({ message: 'Token expired' }),
    });

    try {
      await api.get('/me');
      expect.fail('Should have thrown ApiClientError');
    } catch (err) {
      expect(err).toBeInstanceOf(ApiClientError);
      expect(err.isAuth).toBe(true);
      expect(err.isForbidden).toBe(false);
    }
  });

  it('translates network failure into NetworkError', async () => {
    global.fetch = vi.fn().mockRejectedValue(new TypeError('Failed to fetch'));

    try {
      await api.get('/contests');
      expect.fail('Should have thrown NetworkError');
    } catch (err) {
      expect(err).toBeInstanceOf(NetworkError);
      expect(err.isNetworkError).toBe(true);
      expect(err.code).toBe('NETWORK_ERROR');
    }
  });

  it('translates timeout into TimeoutError', async () => {
    global.fetch = vi.fn().mockImplementation((url, options) => {
      return new Promise((resolve, reject) => {
        options.signal.addEventListener('abort', () => {
          const abortError = new Error('The operation was aborted.');
          abortError.name = 'AbortError';
          reject(abortError);
        });
      });
    });

    try {
      await request('/long-running', { timeout: 50 });
      expect.fail('Should have timed out');
    } catch (err) {
      expect(err).toBeInstanceOf(TimeoutError);
      expect(err.isTimeout).toBe(true);
      expect(err.timeoutMs).toBe(50);
    }
  });
});
