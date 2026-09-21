/**
 * Session state kept in memory only. Nothing sensitive is written to
 * localStorage: the access token dies with the tab and the refresh token is
 * an HttpOnly cookie managed by the browser.
 */
import { cookieRequest } from '../api/client.js';

let accessToken = null;
let currentUser = null;
let refreshing = null;
const listeners = new Set();

function emit(event) {
  listeners.forEach((listener) => {
    try {
      listener(event, { user: currentUser });
    } catch (err) {
      console.error('session listener failed', err);
    }
  });
}

export function subscribe(listener) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function getAccessToken() {
  return accessToken;
}

export function getCurrentUser() {
  return currentUser;
}

/** Stores a session DTO returned by login or refresh. */
export function setSession(session) {
  accessToken = session?.access_token || null;
  currentUser = session?.user || null;
  emit('changed');
}

export function clearSession() {
  accessToken = null;
  currentUser = null;
  emit('changed');
}

/** Marks the session as expired after a failed refresh. */
export function expireSession() {
  const hadSession = Boolean(accessToken);
  clearSession();
  if (hadSession) emit('expired');
}

async function doRefresh() {
  try {
    const session = await cookieRequest('/auth/refresh');
    setSession(session);
    return true;
  } catch {
    return false;
  }
}

/**
 * Rotates the refresh cookie once for all concurrent callers of this tab.
 * Across tabs, the Web Locks API serializes rotations so two tabs never
 * present the same single-use refresh token.
 */
export function refreshSession() {
  if (!refreshing) {
    const run = typeof navigator !== 'undefined' && navigator.locks?.request
      ? navigator.locks.request('agentrix-refresh', doRefresh)
      : doRefresh();
    refreshing = Promise.resolve(run).finally(() => {
      refreshing = null;
    });
  }
  return refreshing;
}

/** Capabilities come exclusively from the backend (/me or session DTO). */
export function hasCapability(user, capability) {
  return Boolean(user && Array.isArray(user.capabilities) && user.capabilities.includes(capability));
}

export function hasAnyCapability(user, ...capabilities) {
  return capabilities.some((c) => hasCapability(user, c));
}
