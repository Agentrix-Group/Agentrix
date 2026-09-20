import React, { createContext, useContext, useEffect, useState, useCallback, useMemo } from 'react';

const RouterContext = createContext(null);

/**
 * Route pattern matching helper
 * E.g. pattern '/replays/:id' matches '/replays/abc-123' -> { match: true, params: { id: 'abc-123' } }
 */
export function matchPattern(path, pattern) {
  const pathClean = path.replace(/\/+$/, '') || '/';
  const patternClean = pattern.replace(/\/+$/, '') || '/';

  if (pathClean === patternClean) {
    return { match: true, params: {} };
  }

  const pathParts = pathClean.split('/').filter(Boolean);
  const patternParts = patternClean.split('/').filter(Boolean);

  if (pathParts.length !== patternParts.length) {
    return { match: false, params: {} };
  }

  const params = {};
  for (let i = 0; i < patternParts.length; i++) {
    const pPart = patternParts[i];
    const actual = pathParts[i];
    if (pPart.startsWith(':')) {
      const paramName = pPart.slice(1);
      params[paramName] = decodeURIComponent(actual);
    } else if (pPart !== actual) {
      return { match: false, params: {} };
    }
  }

  return { match: true, params };
}

export function parseLocation(pathname) {
  const clean = pathname || (typeof window !== 'undefined' ? window.location.pathname : '/');
  return clean === '' ? '/' : clean;
}

export function Router({ children }) {
  const [currentPath, setCurrentPath] = useState(() => {
    if (typeof window !== 'undefined') {
      return parseLocation(window.location.pathname);
    }
    return '/';
  });

  useEffect(() => {
    if (typeof window === 'undefined') return;

    const handlePopState = () => {
      setCurrentPath(parseLocation(window.location.pathname));
    };

    window.addEventListener('popstate', handlePopState);
    return () => window.removeEventListener('popstate', handlePopState);
  }, []);

  const navigate = useCallback((to, { replace = false } = {}) => {
    if (typeof window === 'undefined') return;

    const target = parseLocation(to);
    if (target === currentPath) return;

    if (replace) {
      window.history.replaceState({}, '', target);
    } else {
      window.history.pushState({}, '', target);
    }
    setCurrentPath(target);
    window.scrollTo(0, 0);
  }, [currentPath]);

  // Derived route and params
  const { route, params } = useMemo(() => {
    const replayMatch = matchPattern(currentPath, '/replays/:id') || matchPattern(currentPath, '/viewer/:id');
    if (replayMatch.match) {
      return { route: 'viewer', params: replayMatch.params };
    }
    if (currentPath === '/viewer' || currentPath === '/replays') {
      return { route: 'viewer', params: {} };
    }
    const matchDetail = matchPattern(currentPath, '/matches/:id');
    if (matchDetail.match) {
      return { route: 'matches', params: matchDetail.params };
    }
    if (currentPath === '/matches') {
      return { route: 'matches', params: {} };
    }
    if (currentPath === '/rankings') {
      return { route: 'rankings', params: {} };
    }
    if (currentPath === '/agents') {
      return { route: 'agents', params: {} };
    }
    if (currentPath === '/auth') {
      return { route: 'auth', params: {} };
    }
    return { route: 'home', params: {} };
  }, [currentPath]);

  const value = useMemo(() => ({
    currentPath,
    route,
    params,
    navigate,
  }), [currentPath, route, params, navigate]);

  return (
    <RouterContext.Provider value={value}>
      {children}
    </RouterContext.Provider>
  );
}

export function useRouter() {
  const context = useContext(RouterContext);
  if (!context) {
    throw new Error('useRouter must be used within a <Router>');
  }
  return context;
}

export function Link({ to, children, className = '', onClick, replace = false, ...rest }) {
  const { currentPath, navigate } = useRouter();
  const isActive = currentPath === to;

  const handleClick = (e) => {
    if (onClick) onClick(e);
    // Don't intercept if modified click (Ctrl+click, Meta+click) or if default prevented
    if (!e.defaultPrevented && e.button === 0 && !e.metaKey && !e.ctrlKey && !e.altKey && !e.shiftKey) {
      e.preventDefault();
      navigate(to, { replace });
    }
  };

  return (
    <a
      href={to}
      className={className}
      onClick={handleClick}
      aria-current={isActive ? 'page' : undefined}
      {...rest}
    >
      {children}
    </a>
  );
}
