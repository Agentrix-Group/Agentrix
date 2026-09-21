import { afterEach, beforeEach, vi } from 'vitest';
import { cleanup } from '@testing-library/react';
import i18n from '../src/i18n/index.js';

class LocalStorageMock {
  constructor() { this.store = {}; }
  get length() { return Object.keys(this.store).length; }
  key(i) { return Object.keys(this.store)[i] ?? null; }
  clear() { this.store = {}; }
  getItem(k) { return Object.prototype.hasOwnProperty.call(this.store, k) ? this.store[k] : null; }
  setItem(k, v) { this.store[k] = String(v); }
  removeItem(k) { delete this.store[k]; }
}
Object.defineProperty(window, 'localStorage', { value: new LocalStorageMock(), configurable: true });

let reducedMotion = false;
export function setReducedMotion(value) { reducedMotion = value; }
window.matchMedia = (query) => ({
  matches: query.includes('prefers-reduced-motion') ? reducedMotion : false,
  media: query,
  addEventListener: () => {},
  removeEventListener: () => {},
});
window.scrollTo = () => {};
HTMLCanvasElement.prototype.getContext = function getContext() {
  const noop = () => {};
  return new Proxy({}, { get: (_, prop) => (prop === 'createLinearGradient' ? () => ({ addColorStop: noop }) : noop), set: () => true });
};

beforeEach(() => { i18n.changeLanguage('es'); });

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  window.localStorage.clear();
});
