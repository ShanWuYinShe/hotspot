/**
 * 响应式断点常量与工具函数
 *
 * 断点值与 frontend `use-mobile.ts` MOBILE_BREAKPOINT=768 / Tailwind md=768 / lg=1024 / xl=1280 对齐，
 * 保证跨端（Web + 鸿蒙）在 phone / tablet / desktop / wide 四态的视觉与布局一致。
 *
 * 鸿蒙 vp 单位：1vp ≈ 1.5px @ 160dpi baseline；与 px 在常见 DPR 下数值近似，便于与 CSS 断点直接对齐。
 *
 * 资源子目录约定（HarmonyOS 标准）：
 *   base/      — phone 默认
 *   phone/     — 显式手机（< 768vp）
 *   tablet/    — 平板（768-1023vp，**复用 phone 布局**，与 frontend TabletLayout 一致）
 *   2in1/      — 桌面 / 折叠屏（≥ 1024vp，独立布局）
 */

// 断点阈值（vp）
export const BREAKPOINT_MOBILE_MAX_VP: number = 768; // < 768vp = mobile
export const BREAKPOINT_TABLET_MAX_VP: number = 1023; // 768-1023vp = tablet
export const BREAKPOINT_DESKTOP_MIN_VP: number = 1024; // >= 1024vp = desktop
export const BREAKPOINT_WIDE_MIN_VP: number = 1280; // >= 1280vp = wide

/**
 * 断点类型枚举
 * - MOBILE:  < 768vp（手机，单独布局）
 * - TABLET:  768-1023vp（平板，**复用 mobile 布局**，与 frontend TabletLayout 一致）
 * - DESKTOP: 1024-1279vp（桌面，多列布局 + 顶部 nav）
 * - WIDE:    >= 1280vp（宽屏，桌面布局 + 2 列 grid 变体）
 */
export enum BreakpointType {
  MOBILE = 'mobile',
  TABLET = 'tablet',
  DESKTOP = 'desktop',
  WIDE = 'wide',
}

/**
 * 根据屏幕宽度（vp）计算当前断点类型
 * @param widthVp 屏幕宽度（vp）
 * @returns 当前断点类型
 */
export function getBreakpointType(widthVp: number): BreakpointType {
  if (widthVp >= BREAKPOINT_WIDE_MIN_VP) {
    return BreakpointType.WIDE;
  }
  if (widthVp >= BREAKPOINT_DESKTOP_MIN_VP) {
    return BreakpointType.DESKTOP;
  }
  if (widthVp >= BREAKPOINT_MOBILE_MAX_VP) {
    return BreakpointType.TABLET;
  }
  return BreakpointType.MOBILE;
}

/**
 * 判断是否为手机 + 平板（即"窄屏"—— 768vp 以下）
 * 与 frontend 768px 以下一致；tablet 在布局上与 mobile 复用同一套。
 */
export function isNarrowScreen(breakpoint: BreakpointType): boolean {
  return breakpoint === BreakpointType.MOBILE || breakpoint === BreakpointType.TABLET;
}

/**
 * 判断是否为桌面（>= 1024vp）
 */
export function isDesktopScreen(breakpoint: BreakpointType): boolean {
  return breakpoint === BreakpointType.DESKTOP || breakpoint === BreakpointType.WIDE;
}
