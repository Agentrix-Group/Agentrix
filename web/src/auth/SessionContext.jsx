import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { ApiService } from '../service/apiService.js';
import { getCurrentUser, hasCapability, hasAnyCapability, subscribe } from './session.js';

const SessionContext = createContext(null);

export function SessionProvider({ children }) {
  const [user, setUser] = useState(getCurrentUser());
  const [isLoading, setIsLoading] = useState(true);
  const [expired, setExpired] = useState(false);

  useEffect(() => {
    const unsubscribe = subscribe((event, state) => {
      setUser(state.user);
      if (event === 'expired') setExpired(true);
    });
    // A reload loses the in-memory access token: restore it from the cookie.
    ApiService.restore().finally(() => setIsLoading(false));
    return unsubscribe;
  }, []);

  const login = useCallback(async (username, password) => {
    const session = await ApiService.login(username, password);
    setExpired(false);
    return session;
  }, []);

  const logout = useCallback(async () => {
    await ApiService.logout();
    setExpired(false);
  }, []);

  const value = useMemo(() => ({
    currentUser: user,
    isAuthenticated: Boolean(user),
    isLoading,
    sessionExpired: expired,
    dismissExpired: () => setExpired(false),
    login,
    logout,
    can: (capability) => hasCapability(user, capability),
    canAny: (...capabilities) => hasAnyCapability(user, ...capabilities),
  }), [user, isLoading, expired, login, logout]);

  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

export function useSession() {
  const context = useContext(SessionContext);
  if (!context) throw new Error('useSession must be used within <SessionProvider>');
  return context;
}
