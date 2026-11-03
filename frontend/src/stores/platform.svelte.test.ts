// Platform store 测试：mobile/tablet/desktop 断点 + 横竖屏 + 宽平板。
import { describe, it, expect } from 'vitest';
import { detect, uiScaleFor } from './platform.svelte';

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
    setViewport(568, 320);
    expect(detect()).toEqual({ platform: 'mobile', orientation: 'landscape', isWideTablet: false });
  });

  it('宽度 600..1023 为 tablet', () => {
    setViewport(768, 1024);
    expect(detect()).toEqual({ platform: 'tablet', orientation: 'portrait', isWideTablet: false });
    setViewport(960, 600);
    expect(detect()).toEqual({ platform: 'tablet', orientation: 'landscape', isWideTablet: true });
  });

  it('宽度 ≥1024 为 desktop', () => {
    setViewport(1024, 768);
    expect(detect()).toEqual({ platform: 'desktop', orientation: 'landscape', isWideTablet: false });
    setViewport(1440, 900);
    expect(detect()).toEqual({ platform: 'desktop', orientation: 'landscape', isWideTablet: false });
  });
});

describe('uiScaleFor()', () => {
  it('≤1920 维持 1×（1920 基准零变化）', () => {
    expect(uiScaleFor(390)).toBe(1);
    expect(uiScaleFor(1440)).toBe(1);
    expect(uiScaleFor(1920)).toBe(1);
  });

  it('超宽屏按 1920 基准线性放大，4K 封顶 2×', () => {
    expect(uiScaleFor(2560)).toBeCloseTo(4 / 3, 5);
    expect(uiScaleFor(3440)).toBeCloseTo(3440 / 1920, 5);
    expect(uiScaleFor(3840)).toBe(2);
    expect(uiScaleFor(5120)).toBe(2);
  });
});
