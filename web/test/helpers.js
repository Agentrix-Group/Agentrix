import { vi } from 'vitest';

export function jsonResponse(status, body, headers = {}) {
  return new Response(body === undefined ? null : JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json', ...headers },
  });
}

export function errorResponse(status, code, message = code, details) {
  return jsonResponse(status, { error: { code, message, details, request_id: 'req-1' } });
}

/** Routes fetch calls by "METHOD path" (path relative to /api/v1). */
export function mockFetch(routes) {
  const calls = [];
  const spy = vi.spyOn(globalThis, 'fetch').mockImplementation(async (url, init = {}) => {
    const path = String(url).replace(/^.*\/api\/v1/, '');
    const method = (init.method || 'GET').toUpperCase();
    calls.push({ method, path, init });
    const handler = routes[`${method} ${path.split('?')[0]}`] || routes[`${method} ${path}`];
    if (!handler) return errorResponse(404, 'route_not_found');
    return typeof handler === 'function' ? handler({ method, path, init, calls }) : handler.clone();
  });
  return { spy, calls };
}
