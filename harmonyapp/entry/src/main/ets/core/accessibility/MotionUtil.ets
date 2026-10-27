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
  /** R7 动效语义（与 web enter 180/exit 150/pop 220 对齐；不新增设计 token） */
  static readonly ENTER_MS: number = 180;
  static readonly ENTER_OFFSET_VP: number = 4;
  static readonly POP_MS: number = 220;
  static readonly POP_SCALE: number = 1.15;

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

  // ==================== R7：内容进场 / 状态 pop ====================

  /** 内容进场位移动量：reduced 时 0（直接终态） */
  static enterOffset(): number {
    return MotionUtil.reduced() ? 0 : MotionUtil.ENTER_OFFSET_VP;
  }

  /**
   * 内容进场：reduced 直接 apply 终态；否则 180ms ease-out opacity/translate 动画。
   * 仅用于数据就绪/内容替换的内容层，首屏与系统转场不调用。
   */
  static contentEnter(context: UIContext, apply: () => void): void {
    if (MotionUtil.reduced()) {
      apply();
      return;
    }
    context.animateTo({ duration: MotionUtil.ENTER_MS, curve: Curve.EaseOut }, apply);
  }

  /**
   * 状态 pop：1→1.15→1，220ms（keyframe 两段各 110ms）；reduced 直接 setSettle 终态。
   * 仅「未激活→激活」调用；取消激活不播。
   */
  static statePop(context: UIContext, setPeak: () => void, setSettle: () => void): void {
    if (MotionUtil.reduced()) {
      setSettle();
      return;
    }
    context.keyframeAnimateTo({ delay: 0 }, [
      { duration: MotionUtil.POP_MS / 2, curve: Curve.EaseOut, event: setPeak },
      { duration: MotionUtil.POP_MS / 2, curve: Curve.EaseInOut, event: setSettle }
    ]);
  }

  /** pop 缩放值：reduced 或未 pop 时 1 */
  static popScale(popped: boolean): number {
    return popped && !MotionUtil.reduced() ? MotionUtil.POP_SCALE : 1;
  }
}
