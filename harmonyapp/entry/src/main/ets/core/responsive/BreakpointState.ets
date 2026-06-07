/**
 * 响应式断点状态（@Observed 单例 + mediaQuery 监听）
 *
 * 用法：
 *   @State breakpoint: BreakpointState = new BreakpointState();
 *
 *   aboutToAppear() {
 *     attachBreakpointListener(this.breakpoint);
 *   }
 *
 *   aboutToDisappear() {
 *     detachBreakpointListener(this.breakpoint);
 *   }
 *
 *   build() {
 *     if (isDesktopScreen(this.breakpoint.current)) {
 *       // desktop 布局
 *     } else {
 *       // mobile + tablet 布局（共用）
 *     }
 *   }
 */

import mediaQuery from '@ohos.mediaquery';
import display from '@ohos.display';
import {
  BreakpointType,
  getBreakpointType
} from './Breakpoint';

@Observed
export class BreakpointState {
  @Track current: BreakpointType = BreakpointType.MOBILE;
  @Track widthVp: number = 0;

  constructor() {
    // 初始化时立即读取一次当前屏幕宽度
    this.refresh();
  }

  /**
   * 主动刷新当前屏幕宽度（vp）并更新断点类型
   */
  refresh(): void {
    const widthVp = getScreenWidthVp();
    this.widthVp = widthVp;
    this.current = getBreakpointType(widthVp);
  }
}

/**
 * 获取当前屏幕宽度（vp）
 * display.getDefaultDisplaySync().width 返回 px，densityDPI 用于换算
 */
function getScreenWidthVp(): number {
  try {
    const displayInfo = display.getDefaultDisplaySync();
    const px = displayInfo.width;
    const density = displayInfo.densityDPI;
    if (density <= 0) {
      return px;
    }
    // 1vp = (density / 160) px ⇒ vp = px / (density / 160) = px * 160 / density
    return Math.floor((px * 160) / density);
  } catch (err) {
    return 0;
  }
}

// 单例 listener 与注册状态集合（多 component 复用同一 listener）
let mediaListener: mediaQuery.MediaQueryListener | null = null;
const registeredStates: Set<BreakpointState> = new Set();

function notifyAll(widthVp: number): void {
  const next = getBreakpointType(widthVp);
  registeredStates.forEach((state) => {
    state.widthVp = widthVp;
    state.current = next;
  });
}

/**
 * 注册断点状态到全局监听
 * 首次注册时创建 mediaQuery listener（监听屏幕宽度变化）
 */
export function attachBreakpointListener(state: BreakpointState): void {
  if (registeredStates.has(state)) {
    return;
  }
  registeredStates.add(state);
  state.refresh();

  if (mediaListener === null) {
    // HarmonyOS mediaQuery 支持 (orientation: landscape) 等
    // 用 (width >= 0vp) 等价于任何宽度变化都触发（实际由 on('change') 推送新 width）
    mediaListener = mediaQuery.matchMediaSync('(width >= 0vp)');
    mediaListener.on('change', (_result: mediaQuery.MediaQueryResult) => {
      const widthVp = getScreenWidthVp();
      notifyAll(widthVp);
    });
  }
}

/**
 * 注销断点状态
 * listener 保持单例（其他 component 可能仍在使用）
 */
export function detachBreakpointListener(state: BreakpointState): void {
  registeredStates.delete(state);
}
