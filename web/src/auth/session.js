/**
 * Central Session and Token Manager for Agentrix
 * Implements Sprint FQ-1 requirements (ADR-0009 / F0.3 / F1.4).
 */

export const ACCESS_TOKEN_KEY = 'agentrix_token';
export const REFRESH_TOKEN_KEY = 'agentrix_refresh_token';

export function getAccessToken() {
  try {
    return localStorage.getItem(ACCESS_TOKEN_KEY) || null;
  } catch {
    return null;
  }
}

export function setAccessToken(token) {
  try {
    if (token) {
      localStorage.setItem(ACCESS_TOKEN_KEY, token);
    } else {
      localStorage.removeItem(ACCESS_TOKEN_KEY);
    }
  } catch {
    // Ignore localStorage errors (e.g. private mode quota)
  }
}

export function getRefreshToken() {
  try {
    return localStorage.getItem(REFRESH_TOKEN_KEY) || null;
  } catch {
    return null;
  }
}

export function setRefreshToken(token) {
  try {
    if (token) {
      localStorage.setItem(REFRESH_TOKEN_KEY, token);
    } else {
      localStorage.removeItem(REFRESH_TOKEN_KEY);
    }
  } catch {
    // Ignore localStorage errors
  }
}

export function saveTokens(tokens) {
  if (tokens?.access_token) {
    setAccessToken(tokens.access_token);
  }
  if (tokens?.refresh_token) {
    setRefreshToken(tokens.refresh_token);
  }
}

export function clearTokens() {
  try {
    localStorage.removeItem(ACCESS_TOKEN_KEY);
    localStorage.removeItem(REFRESH_TOKEN_KEY);
  } catch {
    // Ignore errors
  }
}

// Session expired listeners
const sessionExpiredListeners = new Set();

export function onSessionExpired(callback) {
  sessionExpiredListeners.add(callback);
  return () => sessionExpiredListeners.delete(callback);
}

export function notifySessionExpired() {
  clearTokens();
  for (const callback of sessionExpiredListeners) {
    try {
      callback();
    } catch (e) {
      console.error('[Agentrix Session] Error in sessionExpired callback:', e);
    }
  }
}

// Coordinated concurrent token refresh (F0.3)
let activeRefreshPromise = null;

export async function refreshAuthTokens(baseUrl) {
  if (activeRefreshPromise) {
    return activeRefreshPromise;
  }

  const refreshToken = getRefreshToken();
  if (!refreshToken) {
    notifySessionExpired();
    throw new Error('No refresh token available');
  }

  activeRefreshPromise = (async () => {
    try {
      const response = await fetch(`${baseUrl}/auth/refresh`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Accept: 'application/json',
        },
        body: JSON.stringify({ refresh_token: refreshToken }),
      });

      if (!response.ok) {
        throw new Error(`Refresh failed with status ${response.status}`);
      }

      const data = await response.json();
      if (!data?.access_token) {
        throw new Error('Invalid refresh token response');
      }

      saveTokens(data);
      return data;
    } catch (err) {
      notifySessionExpired();
      throw err;
    } finally {
      activeRefreshPromise = null;
    }
  })();

  return activeRefreshPromise;
}

/**
 * Capability-based Authorization Helpers (F1.4)
 * Allows dynamic capability resolution from participant object.
 */
export function hasCapability(user, capability) {
  if (!user) return false;

  // Direct capability list from backend (/me or /auth/login)
  if (Array.isArray(user.capabilities) && user.capabilities.includes(capability)) {
    return true;
  }

  // Fallback role-based capabilities mapping
  const role = user.role_id || user.role;
  if (role === 'admin') {
    return true; // Admin has all capabilities
  }

  if (role === 'organizer') {
    return [
      'matches:run',
      'matches:schedule',
      'contests:create',
      'contests:view',
      'matches:view',
      'rankings:view',
      'replays:view',
      'admin:access',
    ].includes(capability);
  }

  if (role === 'participant' || role === 'pilot' || role === 'player') {
    return [
      'agents:create',
      'submissions:upload',
      'contests:enroll',
      'contests:view',
      'matches:view',
      'rankings:view',
      'replays:view',
    ].includes(capability);
  }

  // Default spectator capabilities
  return ['contests:view', 'matches:view', 'rankings:view', 'replays:view'].includes(capability);
}

export function canRunMatch(user) {
  return hasCapability(user, 'matches:run');
}

export function canScheduleMatch(user) {
  return hasCapability(user, 'matches:schedule');
}

export function canCreateAgent(user) {
  return hasCapability(user, 'agents:create');
}

export function canUploadSubmission(user) {
  return hasCapability(user, 'submissions:upload');
}
