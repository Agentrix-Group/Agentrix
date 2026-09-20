import React, { createContext, useContext, useEffect, useState, useCallback, useMemo } from 'react';
import { ApiService } from '../service/apiService.js';
import {
  getAccessToken,
  clearTokens,
  onSessionExpired,
  hasCapability as checkCapability,
  canRunMatch as checkCanRunMatch,
  canScheduleMatch as checkCanScheduleMatch,
  canCreateAgent as checkCanCreateAgent,
  canUploadSubmission as checkCanUploadSubmission,
} from './session.js';

const SessionContext = createContext(null);

export function SessionProvider({ children }) {
  const [currentUser, setCurrentUser] = useState(null);
  const [isLoading, setIsLoading] = useState(true);
  const [sessionExpired, setSessionExpired] = useState(false);

  const loadCurrentUser = useCallback(async () => {
    const token = getAccessToken();
    if (!token) {
      setCurrentUser(null);
      setIsLoading(false);
      return;
    }

    try {
      const user = await ApiService.getCurrentUser();
      setCurrentUser(user);
    } catch {
      clearTokens();
      setCurrentUser(null);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    loadCurrentUser();

    // Listen to 401 expiration events emitted by HTTP client
    const unsubscribe = onSessionExpired(() => {
      setCurrentUser(null);
      setSessionExpired(true);
    });

    return unsubscribe;
  }, [loadCurrentUser]);

  const login = useCallback(async (username, password) => {
    setSessionExpired(false);
    const res = await ApiService.login(username, password);
    if (res?.participant) {
      setCurrentUser(res.participant);
    } else {
      await loadCurrentUser();
    }
    return res;
  }, [loadCurrentUser]);

  const logout = useCallback(() => {
    ApiService.logout();
    setCurrentUser(null);
    setSessionExpired(false);
  }, []);

  const dismissExpiredNotice = useCallback(() => {
    setSessionExpired(false);
  }, []);

  const hasCapability = useCallback((cap) => {
    return checkCapability(currentUser, cap);
  }, [currentUser]);

  const canRunMatch = useCallback(() => {
    return checkCanRunMatch(currentUser);
  }, [currentUser]);

  const canScheduleMatch = useCallback(() => {
    return checkCanScheduleMatch(currentUser);
  }, [currentUser]);

  const canCreateAgent = useCallback(() => {
    return checkCanCreateAgent(currentUser);
  }, [currentUser]);

  const canUploadSubmission = useCallback(() => {
    return checkCanUploadSubmission(currentUser);
  }, [currentUser]);

  const value = useMemo(() => ({
    currentUser,
    isAuthenticated: Boolean(currentUser),
    isLoading,
    sessionExpired,
    login,
    logout,
    dismissExpiredNotice,
    hasCapability,
    canRunMatch,
    canScheduleMatch,
    canCreateAgent,
    canUploadSubmission,
    reloadSession: loadCurrentUser,
  }), [
    currentUser,
    isLoading,
    sessionExpired,
    login,
    logout,
    dismissExpiredNotice,
    hasCapability,
    canRunMatch,
    canScheduleMatch,
    canCreateAgent,
    canUploadSubmission,
    loadCurrentUser,
  ]);

  return (
    <SessionContext.Provider value={value}>
      {children}
    </SessionContext.Provider>
  );
}

export function useSession() {
  const context = useContext(SessionContext);
  if (!context) {
    throw new Error('useSession must be used within a <SessionProvider>');
  }
  return context;
}
