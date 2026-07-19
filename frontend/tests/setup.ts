/**
 * Vitest global setup
 *
 * Provides the minimal browser APIs that jsdom lacks and that SvelteKit
 * components / libs touch at import time. Kept deliberately small — extend
 * only when a real test starts failing on a missing global.
 */
import { afterEach, vi } from 'vitest';

// jsdom has no matchMedia. Svelte components (and some libraries) read it at
// module load (e.g. prefers-color-scheme). Provide a noop stand-in.
if (!('matchMedia' in window) || typeof window.matchMedia !== 'function') {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    configurable: true,
    value: (query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => false
    })
  });
}

// IntersectionObserver is absent in jsdom and lazy-mounted components use it.
if (typeof window.IntersectionObserver === 'undefined') {
  class MockIntersectionObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
    takeRecords() {
      return [];
    }
  }
  Object.defineProperty(window, 'IntersectionObserver', {
    writable: true,
    configurable: true,
    value: MockIntersectionObserver
  });
}

// ResizeObserver is absent in jsdom; some UI primitives call it on mount.
if (typeof window.ResizeObserver === 'undefined') {
  class MockResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
  Object.defineProperty(window, 'ResizeObserver', {
    writable: true,
    configurable: true,
    value: MockResizeObserver
  });
}

// crypto.randomUUID is widely available in modern Node; jsdom exposes crypto
// too. We don't polyfill here — if a test fails we'll add it.

// Restore all vi mocks / stubs between tests so leaky global state from one
// test (e.g. a stubbed fetch) doesn't poison the next.
afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});
