import { describe, it, expect } from 'vitest';
import { cn } from './utils';

describe('cn()', () => {
  it('合并普通类名', () => {
    expect(cn('a', 'b', undefined, null, false)).toBe('a b');
  });

  it('tailwind-merge 去重冲突类', () => {
    expect(cn('px-2', 'px-4')).toBe('px-4');
    expect(cn('bg-red-500', 'bg-blue-500')).toBe('bg-blue-500');
  });

  it('支持条件对象', () => {
    expect(cn({ 'a': true, 'b': false })).toBe('a');
  });
});
