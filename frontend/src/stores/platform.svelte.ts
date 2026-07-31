// Platform store — 响应式平台检测（mobile/tablet/desktop + 横竖屏）。
// 迁移自 React PlatformContext（runes 模块单例）。

export type Platform = 'mobile' | 'tablet' | 'desktop';
export type Orientation = 'portrait' | 'landscape';

export function detect(): { platform: Platform; orientation: Orientation } {
  if (typeof window === 'undefined') {
    return { platform: 'mobile', orientation: 'portrait' };
  }
  const w = window.innerWidth;
  const h = window.innerHeight;
  const minDim = Math.min(w, h);
  const orientation: Orientation = w > h ? 'landscape' : 'portrait';

  let platform: Platform;
  if (w >= 1280) platform = 'desktop';
  else if (minDim >= 600) platform = 'tablet';
  else platform = 'mobile';

  return { platform, orientation };
}

let platform = $state<Platform>('mobile');
let orientation = $state<Orientation>('portrait');

function apply(value: { platform: Platform; orientation: Orientation }) {
  platform = value.platform;
  orientation = value.orientation;
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

export const platformStore = {
  get platform() { return platform; },
  get orientation() { return orientation; },
};
