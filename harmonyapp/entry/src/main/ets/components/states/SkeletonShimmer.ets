/**
 * SkeletonShimmer — 骨架屏 shimmer 动效包装
 *
 * 用法：
 *   SkeletonShimmer({ active: this.isLoading }) {
 *     Column() {
 *       Skeleton({ blockWidth: '100%', blockHeight: 16 })
 *       Skeleton({ blockWidth: '80%', blockHeight: 16 }).margin({ top: DesignConstants.SPACING.SM })
 *     }
 *   }
 *
 * 视觉：使用 animateTo 实现透明度 0.4 ↔ 1.0 循环，
 * 1.5s 周期 + Smooth 曲线，与 frontend Tailwind animate-pulse 风格对齐
 *
 * 与 frontend /frontend/src/lib/components/ui/skeleton.tsx 对齐
 */

import { DesignConstants } from '../../constants/DesignConstants';

@Component
export struct SkeletonShimmer {
  @Prop active: boolean = true;
  @BuilderParam child: () => void = this.defaultChild;

  @State shimmerOpacity: number = 1.0;

  aboutToAppear() {
    if (this.active) {
      this.startShimmer();
    }
  }

  aboutToDisappear() {
    this.cancelAnimation();
  }

  build() {
    Column() {
      if (this.child) {
        this.child();
      } else {
        this.defaultChild();
      }
    }
    .opacity(this.shimmerOpacity)
    .width('100%')
  }

  @Builder
  defaultChild() {
    Column().width(0).height(0)
  }

  private startShimmer() {
    this.shimmerOpacity = 1.0;
    animateTo({
      duration: DesignConstants.ANIMATION.DURATION_SLOW, // 脉冲/shimmer 循环周期
      curve: DesignConstants.ANIMATION.CURVE_SMOOTH,
      iterations: -1,
      playMode: PlayMode.Alternate
    }, () => {
      this.shimmerOpacity = 0.4;
    });
  }

  private cancelAnimation() {
    animateTo({ duration: 0 }, () => {});
  }
}
