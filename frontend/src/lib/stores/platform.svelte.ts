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
   *
   * ⚠️ 业务页优先用下方语义化派生量（useMasterDetail 等）或容器查询原语（.density-grid /
   * .sidebar-layout / .switcher，见 app.css），而非直接读 isWide。直接读 isWide 仅用于
   * 「信息架构级」差异（master-detail vs 长滚动这类容器查询无法消解的结构差异）。
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

  // ── 语义化派生量：表达「布局意图」而非「设备类型」。
  //    业务页应优先引用这些意图名（或直接用容器查询原语），让代码读起来是
  //    「我要双栏」而不是「我在桌面上」。

  /** 采用 master-detail 信息架构（左导航 + 右内容区，点击切换）。
   *  用于 settings / profile 等存在「双栏 master-detail vs 单列长滚动」结构差异的页面。
   *  这是信息架构级决策，容器查询无法消解——窄屏需要一镜到底的长滚动体验。 */
  get useMasterDetail() {
    return this.isWide;
  }

  /** 采用常驻侧栏导航 chrome（桌面专属：顶栏 + 192px 侧边栏）。 */
  get useSidebarChrome() {
    return this.isDesktop;
  }

  /** 桌面侧栏已直达某入口，页面内不再重复放置该入口（如 nearby 页的返回/创建按钮、
   *  profile 的 personal hub 块）。这是「入口去重」的视口级语义。 */
  get hideRedundantEntry() {
    return this.isDesktop;
  }

  init() {
    if (this.#initialized) return;
    this.#initialized = true;
    this.update();
  }
}

export const platformStore = new PlatformStore();
