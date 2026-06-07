/**
 * 响应式断点（HarmonyOS ArkUI 声明式栅格）
 *
 * 使用方式：
 *   GridRow({ ...GRID_ROW_OPTIONS }) {
 *     GridCol({ span: { sm: 12, md: 6, lg: 4 } }) { ... }
 *   }
 *
 *   if (GRID_ROW_OPTIONS.currentBreakpoint === 'lg' || GRID_ROW_OPTIONS.currentBreakpoint === 'xl') {
 *     // desktop 布局
 *   }
 *
 * 设计动机：
 *   - HarmonyOS 4+ 提供 GridRow/GridCol 声明式栅格（breakpoints / columns / gutter），
 *     内部自动响应屏幕尺寸变化，无需写 mediaQuery 监听
 *   - 与 frontend 768/1024/1280 断点风格对齐（lg 切桌面 = frontend `lg:` 断点），
 *     但用 HarmonyOS 标准命名 xs/sm/md/lg/xl/xxl 与原生栅格无缝集成
 *   - 旧版 mediaQuery 监听实现（BreakpointState.ets）已删除
 */

// 断点阈值（vp）— 与 HarmonyOS ArkUI GridRow breakpoints 标准对齐
export const BREAKPOINT_XS_MAX_VP: number = 319;   // < 320vp = xs
export const BREAKPOINT_SM_MAX_VP: number = 599;   // 320-599vp = sm
export const BREAKPOINT_MD_MAX_VP: number = 839;   // 600-839vp = md（手机横屏 / 平板竖屏）
export const BREAKPOINT_LG_MAX_VP: number = 1199;  // 840-1199vp = lg（桌面 / 平板横屏）
export const BREAKPOINT_XL_MAX_VP: number = 1599;  // 1200-1599vp = xl
// >= 1600vp = xxl（宽屏）

/**
 * HarmonyOS 命名断点（与 ArkUI GridRow.currentBreakpoint 一致）
 * - xs: < 320vp（紧凑手机）
 * - sm: 320-599vp（标准手机竖屏）
 * - md: 600-839vp（手机横屏 / 平板竖屏）
 * - lg: 840-1199vp（**桌面** / 平板横屏，与 frontend `lg:` 断点对齐）
 * - xl: 1200-1599vp（宽屏桌面，与 frontend `xl:` 对齐）
 * - xxl: >= 1600vp（超宽屏）
 */
export type BreakpointType = 'xs' | 'sm' | 'md' | 'lg' | 'xl' | 'xxl';

/**
 * 业务视角的端形态
 * - MOBILE: xs / sm / md（手机，**含平板竖屏**，与 frontend TabletLayout 一致）
 * - DESKTOP: lg / xl / xxl（桌面 / 折叠屏 / PC）
 */
export type DeviceForm = 'mobile' | 'desktop';

export function isMobileForm(bp: BreakpointType): boolean {
  return bp === 'xs' || bp === 'sm' || bp === 'md';
}

export function isDesktopForm(bp: BreakpointType): boolean {
  return bp === 'lg' || bp === 'xl' || bp === 'xxl';
}

/**
 * ArkUI GridRow breakpoints 配置（vp 单位）
 * 与 frontend lg=1024 风格对齐：lg (840vp) 起视为桌面。
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
export const GRID_COLUMNS: Record<BreakpointType, number> = {
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
export const GRID_GUTTER_X: Record<BreakpointType, number> = {
  xs: 8,
  sm: 12,
  md: 16,
  lg: 24,
  xl: 32,
  xxl: 32
};

export const GRID_GUTTER_Y: Record<BreakpointType, number> = {
  xs: 8,
  sm: 12,
  md: 16,
  lg: 24,
  xl: 32,
  xxl: 32
};

/**
 * 容器最大宽度（vp）— 与 frontend content max-width 对齐
 * 桌面端限制内容最大宽度，避免超宽屏铺满
 */
export const CONTAINER_MAX_WIDTH_VP: Record<BreakpointType, number> = {
  xs: 0,    // 充满
  sm: 0,
  md: 720,
  lg: 960,
  xl: 1200,
  xxl: 1440
};
