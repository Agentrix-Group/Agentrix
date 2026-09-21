import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';

/**
 * useResource loads data with an AbortController that is aborted on unmount
 * or when dependencies change. `reload` refetches; `setData` allows local
 * updates after mutations.
 */
export function useResource(loader, deps = []) {
  const [state, setState] = useState({ data: null, error: null, loading: true });
  const [version, setVersion] = useState(0);
  const loaderRef = useRef(loader);
  loaderRef.current = loader;

  useEffect(() => {
    const controller = new AbortController();
    setState((s) => ({ ...s, loading: true, error: null }));
    Promise.resolve(loaderRef.current(controller.signal))
      .then((data) => {
        if (!controller.signal.aborted) setState({ data, error: null, loading: false });
      })
      .catch((error) => {
        if (!controller.signal.aborted) setState({ data: null, error, loading: false });
      });
    return () => controller.abort();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, version]);

  const reload = useCallback(() => setVersion((v) => v + 1), []);
  const setData = useCallback((updater) => setState((s) => ({ ...s, data: typeof updater === 'function' ? updater(s.data) : updater })), []);
  return { ...state, reload, setData };
}

/** usePolling calls fn every interval while `active` is true. */
export function usePolling(fn, interval, active) {
  const fnRef = useRef(fn);
  fnRef.current = fn;
  useEffect(() => {
    if (!active) return undefined;
    const timer = setInterval(() => fnRef.current(), interval);
    return () => clearInterval(timer);
  }, [interval, active]);
}

export function usePrefersReducedMotion() {
  const query = '(prefers-reduced-motion: reduce)';
  const get = () => typeof window !== 'undefined' && typeof window.matchMedia === 'function' && window.matchMedia(query).matches;
  const [reduced, setReduced] = useState(get);
  useEffect(() => {
    if (typeof window.matchMedia !== 'function') return undefined;
    const media = window.matchMedia(query);
    const onChange = () => setReduced(media.matches);
    media.addEventListener?.('change', onChange);
    return () => media.removeEventListener?.('change', onChange);
  }, []);
  return reduced;
}

/**
 * useFlip animates list items (keyed by data-flip-key) from their previous
 * position to the new one (FLIP). Disabled with prefers-reduced-motion.
 */
export function useFlip(containerRef, keys) {
  const positions = useRef(new Map());
  const reduced = usePrefersReducedMotion();
  useLayoutEffect(() => {
    const container = containerRef.current;
    if (!container) return;
    const nodes = container.querySelectorAll('[data-flip-key]');
    const next = new Map();
    nodes.forEach((node) => {
      const key = node.getAttribute('data-flip-key');
      const top = node.getBoundingClientRect().top;
      next.set(key, top);
      const previous = positions.current.get(key);
      if (!reduced && previous !== undefined && previous !== top && typeof node.animate === 'function') {
        node.animate([{ transform: `translateY(${previous - top}px)` }, { transform: 'translateY(0)' }],
          { duration: 320, easing: 'cubic-bezier(0.2, 0, 0, 1)' });
      }
    });
    positions.current = next;
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [keys, reduced]);
}
