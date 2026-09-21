// Fase 0 regressions (audit 2026-09-21), expressed against the canonical client.
import { describe, it, expect } from 'vitest';
import { ApiService } from '../src/service/apiService.js';
import { getAccessToken } from '../src/auth/session.js';
import { mockFetch, jsonResponse } from './helpers.js';

const session = { access_token: 'access-1', token_type: 'Bearer', expires_in: 600, user: { id: 'u', username: 'alice', roles: ['player'], capabilities: ['agents:create'] } };

describe('P0 frontend regressions', () => {
  it('user status changes always send an explicit status', async () => {
    const { calls } = mockFetch({ 'PUT /users/u-1/status': () => jsonResponse(200, { id: 'u-1' }) });
    await ApiService.setUserStatus('u-1', 'active');
    expect(JSON.parse(calls[0].init.body)).toEqual({ status: 'active' });
    expect(() => ApiService.setUserStatus('u-1')).toThrow(/explicit status/);
    expect(() => ApiService.setAgentStatus('a-1')).toThrow(/explicit status/);
  });

  it('logout revokes the session on the server with the refresh cookie', async () => {
    const { calls } = mockFetch({ 'POST /auth/logout': () => new Response(null, { status: 204 }) });
    await ApiService.logout();
    expect(calls).toHaveLength(1);
    expect(calls[0].init.credentials).toBe('include');
    expect(calls[0].init.headers['X-Requested-With']).toBe('agentrix');
    expect(getAccessToken()).toBeNull();
  });

  it('never stores tokens in localStorage', async () => {
    mockFetch({ 'POST /auth/login': () => jsonResponse(200, session) });
    await ApiService.login('alice', 'secret');
    expect(getAccessToken()).toBe('access-1');
    expect(window.localStorage.length).toBe(0);
  });

  it('enrollment requires an explicit submission', () => {
    expect(() => ApiService.enroll('c', 'a')).toThrow(/submission_id/);
  });
});
