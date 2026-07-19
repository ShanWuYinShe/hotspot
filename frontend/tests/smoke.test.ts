import { describe, it, expect } from 'vitest';
import { AUTH, API_BASE } from '$lib/api/endpoints';

describe('vitest setup smoke test', () => {
  it('resolves $lib alias via SvelteKit vite plugin', () => {
    expect(API_BASE).toBe('/api/v1');
    expect(AUTH.LOGIN).toBe('/api/v1/auth/login');
  });
});
