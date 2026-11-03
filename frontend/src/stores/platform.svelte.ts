// Platform store — 响应式平台检测（mobile/tablet/desktop + 横竖屏 + 宽平板）。
// PlatformContext（runes 模块单例）。
// isWideTablet：≥960px 的平板为宽平板（可提升分栏 workspace / 浮动 Dock），
// 对齐 Figma Make PlatformContext 契约。
// 阈值 600/960/1024 同时是 styles/tailwind.css 的 CSS 断点刻度（sm/md/lg），
// 两处是同一套契约，改动必须同步。

export type Platform = 'mobile' | 'tablet' | 'desktop';
export type Orientation = 'portrait' | 'landscape';

export function detect(): { platform: Platform; orientation: Orientation; isWideTablet: boolean } {
  if (typeof window === 'undefined') {
    return { platform: 'mobile', orientation: 'portrait', isWideTablet: false };
  }
  const w = window.innerWidth;
  const h = window.innerHeight;
  const orientation: Orientation = w > h ? 'landscape' : 'portrait';

  let platform: Platform;
  if (w >= 1024) platform = 'desktop';
  else if (w >= 600) platform = 'tablet';
  else platform = 'mobile';

  const isWideTablet = platform === 'tablet' && w >= 960;
  return { platform, orientation, isWideTablet };
}

/**
 * 大屏 UI 缩放倍数（与 shared.css 根字号规则同一公式）：
 * 以 1920 为设计基准线性放大，1×（≤1920）→ 2×（≥3840 封顶）。
 * 供数值型内联尺寸（lucide size={n}、Avatar 等）通过 --cst-ui-scale 同步缩放。
 */
export function uiScaleFor(width: number): number {
  return Math.min(2, Math.max(1, width / 1920));
}

/** 把当前缩放倍数写回 :root（CSS 无法用 calc 求长度比值，故由 JS 同步）。 */
function syncUiScale(): void {
  if (typeof document === 'undefined') return;
  const scale = uiScaleFor(window.innerWidth);
  uiScale = scale;
  document.documentElement.style.setProperty('--cst-ui-scale', scale.toFixed(4));
}

// 内部状态变量与导出对象重名会自引用（getter 内 `platform` 解析到导出对象本身），
// 故内部用 devicePlatform，导出对象命名为 platform（与 auth/composer/presence/ws 一致）。
let devicePlatform = $state<Platform>('mobile');
let orientation = $state<Orientation>('portrait');
let isWideTablet = $state(false);
// 大屏 UI 缩放倍数（1× → 2×）：CSS 根字号负责 rem 缩放，此值供 JS/内联 px 场景读取。
let uiScale = $state(1);

function apply(value: { platform: Platform; orientation: Orientation; isWideTablet: boolean }) {
  devicePlatform = value.platform;
  orientation = value.orientation;
  isWideTablet = value.isWideTablet;
}

/** 由根布局调用一次：初始检测 + resize/orientationchange 监听（rAF 节流）。返回清理函数。 */
export function initPlatform(): () => void {
  apply(detect());
  syncUiScale();
  let raf = 0;
  const handler = () => {
    cancelAnimationFrame(raf);
    raf = requestAnimationFrame(() => {
      apply(detect());
      syncUiScale();
    });
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
  get uiScale() { return uiScale; },
};
