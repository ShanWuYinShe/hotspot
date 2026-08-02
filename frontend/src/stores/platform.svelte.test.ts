// Platform store 测试：mobile/tablet/desktop 断点 + 横竖屏 + 宽平板。
import { describe, it, expect } from 'vitest';
import { detect } from './platform.svelte';

function setViewport(w: number, h: number) {
  Object.defineProperty(window, 'innerWidth', { configurable: true, value: w });
  Object.defineProperty(window, 'innerHeight', { configurable: true, value: h });
}

describe('detect()', () => {
  it('宽度 <600 为 mobile portrait', () => {
    setViewport(375, 812);
    expect(detect()).toEqual({ platform: 'mobile', orientation: 'portrait', isWideTablet: false });
  });

  it('宽度 <600 横屏为 mobile landscape', () => {
    setViewport(667, 375);
    expect(detect()).toEqual({ platform: 'mobile', orientation: 'landscape', isWideTablet: false });
  });

  it('minDim ≥600 且宽度 <1280 为 tablet', () => {
    setViewport(1024, 768);
    expect(detect()).toEqual({ platform: 'tablet', orientation: 'landscape', isWideTablet: true });
    setViewport(768, 1024);
    expect(detect()).toEqual({ platform: 'tablet', orientation: 'portrait', isWideTablet: false });
  });

  it('宽度 ≥1280 为 desktop', () => {
    setViewport(1440, 900);
    expect(detect()).toEqual({ platform: 'desktop', orientation: 'landscape', isWideTablet: false });
  });
});
