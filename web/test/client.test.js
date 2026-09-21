import { describe, it, expect, vi } from 'vitest';
import { request, ApiError } from '../src/api/client.js';
import { ApiService } from '../src/service/apiService.js';
import { setSession, getAccessToken, subscribe, clearSession } from '../src/auth/session.js';
import { mockFetch, jsonResponse, errorResponse } from './helpers.js';

const user = { id: 'u', username: 'alice', roles: ['player'], capabilities: [] };

describe('transport', () => {
  it('maps the error envelope to ApiError', async () => {
    mockFetch({ 'GET /matches/x': errorResponse(409, 'run_in_progress', 'busy', { run_id: 'r1' }) });
    const err = await request('/matches/x').catch((e) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect(err.status).toBe(409);
    expect(err.code).toBe('run_in_progress');
    expect(err.details).toEqual({ run_id: 'r1' });
    expect(err.requestId).toBe('req-1');
  });

  it('sends credentials and the bearer token', async () => {
    setSession({ access_token: 'tok', user });
    const { calls } = mockFetch({ 'GET /me': jsonResponse(200, user) });
    await request('/me');
    expect(calls[0].init.credentials).toBe('include');
    expect(calls[0].init.headers.Authorization).toBe('Bearer tok');
    clearSession();
  });

  it('refreshes once for concurrent 401s and retries', async () => {
    setSession({ access_token: 'old', user });
    let refreshes = 0;
    const { calls } = mockFetch({
      'GET /me': ({ init }) => (init.headers.Authorization === 'Bearer new' ? jsonResponse(200, user) : errorResponse(401, 'invalid_token')),
      'POST /auth/refresh': () => { refreshes += 1; return jsonResponse(200, { access_token: 'new', user }); },
    });
    const results = await Promise.all([request('/me'), request('/me'), request('/me')]);
    expect(results.every((r) => r.id === 'u')).toBe(true);
    expect(refreshes).toBe(1);
    const refreshCall = calls.find((c) => c.path === '/auth/refresh');
    expect(refreshCall.init.headers['X-Requested-With']).toBe('agentrix');
    expect(getAccessToken()).toBe('new');
    clearSession();
  });

  it('expires the session when refresh fails', async () => {
    setSession({ access_token: 'old', user });
    const events = [];
    const unsubscribe = subscribe((event) => events.push(event));
    mockFetch({ 'GET /me': errorResponse(401, 'session_revoked'), 'POST /auth/refresh': errorResponse(401, 'invalid_refresh_token') });
    const err = await request('/me').catch((e) => e);
    expect(err.status).toBe(401);
    expect(getAccessToken()).toBeNull();
    expect(events).toContain('expired');
    unsubscribe();
  });

  it('reports non-JSON bodies instead of crashing (JSON.parse bug regression)', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response('<html>', { status: 200, headers: { 'Content-Type': 'text/html' } }));
    const err = await request('/contests').catch((e) => e);
    expect(err.code).toBe('unexpected_content');
  });

  it('reports network failures as ApiError status 0', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new TypeError('Failed to fetch'));
    const err = await request('/contests').catch((e) => e);
    expect(err.isNetwork).toBe(true);
  });

  it('scheduling a run always carries an Idempotency-Key', async () => {
    const { calls } = mockFetch({ 'POST /matches/m/runs': jsonResponse(202, { run_id: 'r', job_id: 'j' }) });
    await ApiService.scheduleRun('m');
    await ApiService.scheduleRun('m', 'fixed-key-123');
    expect(calls[0].init.headers['Idempotency-Key']).toMatch(/.{8,}/);
    expect(calls[1].init.headers['Idempotency-Key']).toBe('fixed-key-123');
  });

  it('aborts when the caller aborts', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation((_, init) => new Promise((_, reject) => {
      init.signal.addEventListener('abort', () => reject(new DOMException('aborted', 'AbortError')));
    }));
    const controller = new AbortController();
    const pending = request('/contests', { signal: controller.signal });
    controller.abort();
    await expect(pending).rejects.toThrow();
  });
});
