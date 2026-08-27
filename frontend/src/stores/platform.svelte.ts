// Platform store — 响应式平台检测（mobile/tablet/desktop + 横竖屏 + 宽平板）。
// PlatformContext（runes 模块单例）。
// isWideTablet：≥960px 的平板为宽平板（可提升分栏 workspace / 浮动 Dock），
// 对齐 Figma Make PlatformContext 契约。
// 阈值 600/960/1280 同时是 styles/tailwind.css 的 CSS 断点刻度（sm/md/lg），
// 两处是同一套契约，改动必须同步。

export type Platform = 'mobile' | 'tablet' | 'desktop';
export type Orientation = 'portrait' | 'landscape';

export function detect(): { platform: Platform; orientation: Orientation; isWideTablet: boolean } {
  if (typeof window === 'undefined') {
    return { platform: 'mobile', orientation: 'portrait', isWideTablet: false };
  }
  const w = window.innerWidth;
  const h = window.innerHeight;
  const minDim = Math.min(w, h);
  const orientation: Orientation = w > h ? 'landscape' : 'portrait';

  let platform: Platform;
  if (w >= 1280) platform = 'desktop';
  else if (minDim >= 600) platform = 'tablet';
  else platform = 'mobile';

  const isWideTablet = platform === 'tablet' && w >= 960;
  return { platform, orientation, isWideTablet };
}

// 内部状态变量与导出对象重名会自引用（getter 内 `platform` 解析到导出对象本身），
// 故内部用 devicePlatform，导出对象命名为 platform（与 auth/composer/presence/ws 一致）。
let devicePlatform = $state<Platform>('mobile');
let orientation = $state<Orientation>('portrait');
let isWideTablet = $state(false);

function apply(value: { platform: Platform; orientation: Orientation; isWideTablet: boolean }) {
  devicePlatform = value.platform;
  orientation = value.orientation;
  isWideTablet = value.isWideTablet;
}

/** 由根布局调用一次：初始检测 + resize/orientationchange 监听（rAF 节流）。返回清理函数。 */
export function initPlatform(): () => void {
  apply(detect());
  let raf = 0;
  const handler = () => {
    cancelAnimationFrame(raf);
    raf = requestAnimationFrame(() => apply(detect()));
  };
  window.addEventListener('resize', handler);
  window.addEventListener('orientationchange', handler);
  return () => {
    window.removeEventListener('resize', handler);
    window.removeEventListener('orientationchange', handler);
    cancelAnimationFrame(raf);
  };
}

export const platform = {
  get platform() { return devicePlatform; },
  get orientation() { return orientation; },
  get isWideTablet() { return isWideTablet; },
};
