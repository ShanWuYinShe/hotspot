/**
 * MotionUtil —— 减少动态（reduced-motion）统一入口（r4）
 *
 * 数据源：AccessibilityManager.shouldReduceMotion()（系统「减少动态/移除动画」+ 应用内关闭动画）。
 * 约束：
 * - 只在 shouldReduceMotion() === true 时把位移/缩放/脉冲动画时长置 0（透明度过渡保留）；
 * - false 时返回原参数，行为与 r3 完全一致；
 * - 不新增动画类型，仅统一读取开关。
 */
import { curves } from '@kit.ArkUI';
import { AccessibilityManager } from './AccessibilityManager';

export interface MotionSpringParam {
  curve: ICurve;
  duration?: number;
}

export class MotionUtil {
  static reduced(): boolean {
    return AccessibilityManager.getInstance().shouldReduceMotion();
  }

  /** 按压/悬停弹簧动画参数：减少动态时 duration=0（无缩放过渡） */
  static spring(response: number, damping: number): MotionSpringParam {
    const param: MotionSpringParam = { curve: curves.springMotion(response, damping) };
    if (MotionUtil.reduced()) {
      param.duration = 0;
    }
    return param;
  }

  /** 按压缩放：减少动态时恒为 1（不位移/不缩放） */
  static pressScale(pressed: boolean, scale: number): number {
    return pressed && !MotionUtil.reduced() ? scale : 1;
  }

  /** 骨架脉冲：减少动态/系统关闭动画时返回 false（骨架静止） */
  static shimmerActive(): boolean {
    return !MotionUtil.reduced();
  }
}
