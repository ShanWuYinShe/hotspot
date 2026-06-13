import { browser } from '$app/environment';

export type Platform = 'mobile' | 'tablet' | 'desktop';
export type Orientation = 'portrait' | 'landscape';

/** ≥ 此视口宽度即视为"宽屏"（可承载双栏 / master-detail）。覆盖横屏及大尺寸平板与桌面。 */
const WIDE_MIN_WIDTH = 960;

class PlatformStore {
  platform = $state<Platform>('mobile');
  orientation = $state<Orientation>('portrait');
  isMobile = $state(true);
  isTablet = $state(false);
  isDesktop = $state(false);
  /** 当前视口宽度（px）。 */
  width = $state(0);
  /**
   * 是否为"宽屏"：视口宽度 ≥ 960px（桌面恒为真，横屏 / 大尺寸平板亦为真）。
   * 用于决定 **内容布局** 是否采用双栏 / master-detail；导航 chrome（侧边栏 vs 底栏）
   * 仍由 isDesktop / isTablet 决定。详见 frontend/CLAUDE.md「三档设备一致性」。
   */
  isWide = $state(false);

  #initialized = false;
  #raf = 0;

  constructor() {
    if (!browser) return;
    this.update();
    const schedule = () => {
      cancelAnimationFrame(this.#raf);
      this.#raf = requestAnimationFrame(() => this.update());
    };
    window.addEventListener('resize', schedule);
    window.addEventListener('orientationchange', schedule);
  }

  private update() {
    const w = window.innerWidth;
    const h = window.innerHeight;
    const minDim = Math.min(w, h);
    let p: Platform;
    if (w >= 1280) p = 'desktop';
    else if (minDim >= 600) p = 'tablet';
    else p = 'mobile';

    this.platform = p;
    this.orientation = w > h ? 'landscape' : 'portrait';
    this.isMobile = p === 'mobile';
    this.isTablet = p === 'tablet';
    this.isDesktop = p === 'desktop';
    this.width = w;
    this.isWide = w >= WIDE_MIN_WIDTH;
  }

  init() {
    if (this.#initialized) return;
    this.#initialized = true;
    this.update();
  }
}

export const platformStore = new PlatformStore();
