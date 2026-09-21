import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';

const RouterContext = createContext(null);

/** Route table: the only paths the SPA serves; anything else is notFound. */
export const ROUTES = [
  { name: 'home', pattern: '/' },
  { name: 'contests', pattern: '/contests' },
  { name: 'contest', pattern: '/contests/:id' },
  { name: 'matches', pattern: '/matches' },
  { name: 'match', pattern: '/matches/:id' },
  { name: 'rankings', pattern: '/rankings' },
  { name: 'agents', pattern: '/agents' },
  { name: 'agent', pattern: '/agents/:id' },
  { name: 'admin', pattern: '/admin' },
  { name: 'auth', pattern: '/auth' },
  { name: 'replay', pattern: '/replays/:id' },
];

export function matchPattern(path, pattern) {
  const clean = (value) => (value.replace(/\/+$/, '') || '/');
  const pathParts = clean(path).split('/').filter(Boolean);
  const patternParts = clean(pattern).split('/').filter(Boolean);
  if (pathParts.length !== patternParts.length) return null;
  const params = {};
  for (let i = 0; i < patternParts.length; i += 1) {
    const expected = patternParts[i];
    const actual = pathParts[i];
    if (expected.startsWith(':')) {
      try {
        params[expected.slice(1)] = decodeURIComponent(actual);
      } catch {
        return null;
      }
    } else if (expected !== actual) {
      return null;
    }
  }
  return params;
}

export function resolveRoute(pathname) {
  for (const route of ROUTES) {
    const params = matchPattern(pathname || '/', route.pattern);
    if (params) return { route: route.name, params };
  }
  return { route: 'notFound', params: {} };
}

function currentLocation() {
  if (typeof window === 'undefined') return { pathname: '/', search: '' };
  return { pathname: window.location.pathname || '/', search: window.location.search || '' };
}

export function Router({ children }) {
  const [location, setLocation] = useState(currentLocation);

  useEffect(() => {
    const onPop = () => setLocation(currentLocation());
    window.addEventListener('popstate', onPop);
    return () => window.removeEventListener('popstate', onPop);
  }, []);

  const navigate = useCallback((to, { replace = false } = {}) => {
    const url = new URL(to, window.location.origin);
    const target = url.pathname + url.search;
    if (target === window.location.pathname + window.location.search) return;
    if (replace) window.history.replaceState({}, '', target);
    else window.history.pushState({}, '', target);
    setLocation({ pathname: url.pathname, search: url.search });
    if (typeof window.scrollTo === 'function') window.scrollTo(0, 0);
  }, []);

  const value = useMemo(() => {
    const { route, params } = resolveRoute(location.pathname);
    const query = Object.fromEntries(new URLSearchParams(location.search));
    return { path: location.pathname, route, params, query, navigate };
  }, [location, navigate]);

  return <RouterContext.Provider value={value}>{children}</RouterContext.Provider>;
}

export function useRouter() {
  const context = useContext(RouterContext);
  if (!context) throw new Error('useRouter must be used within <Router>');
  return context;
}

export function Link({ to, children, className = '', onClick, replace = false, ...rest }) {
  const { path, navigate } = useRouter();
  const handleClick = (event) => {
    if (onClick) onClick(event);
    if (!event.defaultPrevented && event.button === 0 && !event.metaKey && !event.ctrlKey && !event.altKey && !event.shiftKey) {
      event.preventDefault();
      navigate(to, { replace });
    }
  };
  return (
    <a href={to} className={className} onClick={handleClick} aria-current={path === to ? 'page' : undefined} {...rest}>
      {children}
    </a>
  );
}
