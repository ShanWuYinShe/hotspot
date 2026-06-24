/**
 * SkeletonShimmer — 骨架屏 shimmer 动效包装
 *
 * 用法：
 *   SkeletonShimmer({ active: this.isLoading }) {
 *     Column() {
 *       Skeleton({ width: '100%', height: 16 })
 *       Skeleton({ width: '80%', height: 16 }).margin({ top: 8 })
 *     }
 *   }
 *
 * 视觉：使用 animateTo 实现透明度 0.4 ↔ 1.0 循环，
 * 1.5s 周期 + Smooth 曲线，与 frontend Tailwind animate-pulse 风格对齐
 *
 * 与 frontend /frontend/src/lib/components/ui/skeleton.tsx 对齐
 */

@Component
export struct SkeletonShimmer {
  @Prop active: boolean = true;
  @BuilderParam child: () => void = this.defaultChild;

  @State opacity: number = 1.0;

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
    .opacity(this.opacity)
    .width('100%')
  }

  @Builder
  defaultChild() {
    Column().width(0).height(0)
  }

  private startShimmer() {
    this.opacity = 1.0;
    animateTo({
      duration: 1500,
      curve: Curve.Smooth,
      iterations: -1,
      playMode: PlayMode.Alternate
    }, () => {
      this.opacity = 0.4;
    });
  }

  private cancelAnimation() {
    animateTo({ duration: 0 }, () => {});
  }
}
