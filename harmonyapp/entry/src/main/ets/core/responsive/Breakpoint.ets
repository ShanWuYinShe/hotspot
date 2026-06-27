/**
 * 响应式断点（HarmonyOS ArkUI 声明式栅格）
 *
 * 使用方式：
 *   GridRow({ ...GRID_ROW_OPTIONS }) {
 *     GridCol({ span: { sm: 12, md: 6, lg: 4 } }) { ... }
 *   }
 *
 *   if (isDesktopForm(bp)) {
 *     // 桌面 chrome（顶栏+侧边栏）
 *   }
 *   if (isWideForm(bp)) {
 *     // 内容双栏 / master-detail（与 chrome 解耦）
 *   }
 *
 * 设计动机（对齐前端 platform.svelte.ts 三轴模型）：
 *   - 底层 6 档（xs..xxl）供 ArkUI GridRow/ResponsiveContainer 栅格使用
 *   - 业务语义三轴：
 *       · isTablet（≥600vp，EntryAbility 维护）：PhoneLayout 内切药丸底栏（对齐前端 tablet 档）
 *       · isWideForm（≥960vp）：内容区是否双栏 / master-detail（对齐前端 isWide=960）
 *       · isDesktopForm（≥1280vp）：是否切桌面 chrome 顶栏+侧边栏（对齐前端 desktop=1280）
 *   - 关键原则（对齐前端 ui-layout.md）：chrome 归 isDesktopForm，内容双栏归 isWideForm，
 *     二者解耦——960–1279vp 是「平板药丸底栏 + 内容可双栏」，不是「桌面侧边栏」。
 *   - 旧版 mediaQuery 监听实现（BreakpointState.ets）已删除
 */

// 底层断点阈值（vp）— 与 HarmonyOS ArkUI GridRow breakpoints 标准对齐
export const BREAKPOINT_XS_MAX_VP: number = 319;   // < 320vp = xs
export const BREAKPOINT_SM_MAX_VP: number = 599;   // 320-599vp = sm
export const BREAKPOINT_MD_MAX_VP: number = 839;   // 600-839vp = md（手机横屏 / 平板竖屏）
export const BREAKPOINT_LG_MAX_VP: number = 1199;  // 840-1199vp = lg（平板横屏 / 小尺寸桌面窗口）
export const BREAKPOINT_XL_MAX_VP: number = 1599;  // 1200-1599vp = xl（桌面）
// >= 1600vp = xxl（宽屏）

// 业务语义阈值（vp）— 对齐前端 platform.svelte.ts
// vp 与 CSS px 同为逻辑单位（EntryAbility 的 widthVp = px*160/densityDPI 即逻辑宽度），故 1:1 对应。
export const WIDE_MIN_VP: number = 960;    // ≥960vp：内容区双栏 / master-detail（对齐前端 isWide=960）
export const DESKTOP_MIN_VP: number = 1280; // ≥1280vp：桌面 chrome 顶栏+侧边栏（对齐前端 desktop=1280）

/**
 * HarmonyOS 命名断点（与 ArkUI GridRow.currentBreakpoint 一致）
 * - xs: < 320vp（紧凑手机）
 * - sm: 320-599vp（标准手机竖屏）
 * - md: 600-839vp（手机横屏 / 平板竖屏）
 * - lg: 840-1199vp（平板横屏 / 小尺寸桌面窗口）
 * - xl: 1200-1599vp（宽屏桌面）
 * - xxl: >= 1600vp（超宽屏）
 */
export type BreakpointType = 'xs' | 'sm' | 'md' | 'lg' | 'xl' | 'xxl';

/**
 * 业务视角的端形态（对齐前端 mobile/tablet/desktop 三档 platform）
 * - MOBILE: xs / sm / md / lg（< 1280vp，手机 + 平板，走 PhoneLayout）
 * - DESKTOP: xl / xxl 中 ≥1280vp（桌面 / PC，走 DesktopLayout）
 * 注：lg(840-1199) 归 MOBILE 形态但 isTablet(≥600) 在其内切药丸底栏。
 */
export type DeviceForm = 'mobile' | 'desktop';

export interface BreakpointNumberMap {
  xs: number;
  sm: number;
  md: number;
  lg: number;
  xl: number;
  xxl: number;
}

/** 手机/平板形态（< 1280vp）：走 PhoneLayout（含平板药丸底栏）。对齐前端 !isDesktop。 */
export function isMobileForm(bp: BreakpointType): boolean {
  return !isDesktopForm(bp);
}

/** 桌面形态（≥ 1280vp）：走 DesktopLayout 顶栏+侧边栏。对齐前端 isDesktop(≥1280)。 */
export function isDesktopForm(bp: BreakpointType): boolean {
  return bp === 'xl' || bp === 'xxl' || getBreakpointVp(bp) >= DESKTOP_MIN_VP;
}

/** 宽屏内容形态（≥ 960vp）：内容区双栏 / master-detail。对齐前端 isWide(≥960)。与 chrome 解耦。 */
export function isWideForm(bp: BreakpointType): boolean {
  return getBreakpointVp(bp) >= WIDE_MIN_VP;
}

// ── 语义化派生量：表达「布局意图」而非「设备类型」（对齐前端 platformStore 同名派生量）。
//    业务页应优先引用这些语义名，而非裸用 isWideForm/isDesktopForm，让代码读起来是
//    「我要双栏」而不是「我在桌面上」。

/** 采用 master-detail 信息架构（双栏 / split-pane）。
 *  对齐前端 platformStore.useMasterDetail（= isWide ≥960）。
 *  用于 discover / message / nearby / profile 等存在「双栏 vs 单列」布局差异的页面。 */
export function useMasterDetail(bp: BreakpointType): boolean {
  return isWideForm(bp);
}

/** 采用常驻侧栏导航 chrome（桌面顶栏+侧边栏）。
 *  对齐前端 platformStore.useSidebarChrome（= isDesktop ≥1280）。
 *  用于 Layout / Index 的 chrome 装配分支。 */
export function useSidebarChrome(bp: BreakpointType): boolean {
  return isDesktopForm(bp);
}

/** 桌面侧栏已直达某入口，页面内隐藏冗余入口（返回/创建按钮等）。
 *  对齐前端 platformStore.hideRedundantEntry（= isDesktop ≥1280）。 */
export function hideRedundantEntry(bp: BreakpointType): boolean {
  return isDesktopForm(bp);
}

/** 断点档位对应的最小 vp（用于语义阈值判断） */
function getBreakpointVp(bp: BreakpointType): number {
  switch (bp) {
    case 'xxl': return 1600;
    case 'xl': return 1200;
    case 'lg': return 840;
    case 'md': return 600;
    case 'sm': return 320;
    default: return 0; // xs
  }
}

/**
 * 根据屏幕宽度（vp）计算当前断点类型
 * @param widthVp 屏幕宽度（vp）
 * @returns 当前断点类型
 */
export function getBreakpointType(widthVp: number): BreakpointType {
  if (widthVp >= BREAKPOINT_XL_MAX_VP + 1) {
    return 'xxl';
  }
  if (widthVp >= BREAKPOINT_LG_MAX_VP + 1) {
    return 'xl';
  }
  if (widthVp >= BREAKPOINT_MD_MAX_VP + 1) {
    return 'lg';
  }
  if (widthVp >= BREAKPOINT_XS_MAX_VP + 1) {
    return 'md';
  }
  if (widthVp >= BREAKPOINT_XS_MAX_VP - 300) {
    return 'sm';
  }
  return 'xs';
}

/**
 * ArkUI GridRow breakpoints 配置（vp 单位）— ArkUI 栅格标准档位。
 * 业务语义阈值（isWideForm≥960 / isDesktopForm≥1280）见文件顶部常量。
 */
export const GRID_BREAKPOINTS_VALUE: string[] = [
  '320vp',
  '600vp',
  '840vp',
  '1200vp',
  '1600vp'
];

/**
 * ArkUI GridRow columns 配置：每个断点对应的栅格总列数
 */
export const GRID_COLUMNS: BreakpointNumberMap = {
  xs: 4,
  sm: 4,
  md: 8,
  lg: 12,
  xl: 12,
  xxl: 12
};

/**
 * ArkUI GridRow gutter 配置：每个断点对应的列间距
 */
export const GRID_GUTTER_X: BreakpointNumberMap = {
  xs: 8,
  sm: 12,
  md: 16,
  lg: 24,
  xl: 32,
  xxl: 32
};

export const GRID_GUTTER_Y: BreakpointNumberMap = {
  xs: 8,
  sm: 12,
  md: 16,
  lg: 24,
  xl: 32,
  xxl: 32
};

/**
 * 容器最大宽度（vp）— 与 frontend app.css 布局 token 严格对齐：
 *   --measure-narrow(720) / --measure-medium(960) / --measure-default(1180) / --measure-wide(1320)
 * 桌面端限制内容最大宽度，避免超宽屏铺满。
 */
export const CONTAINER_MAX_WIDTH_VP: BreakpointNumberMap = {
  xs: 0,    // 充满（手机竖屏）
  sm: 0,    // 充满
  md: 720,  // 手机横屏 / 平板竖屏 → --measure-narrow
  lg: 960,  // 平板横屏 → --measure-medium
  xl: 1180, // 桌面 → --measure-default
  xxl: 1320 // 宽屏桌面 → --measure-wide
};
