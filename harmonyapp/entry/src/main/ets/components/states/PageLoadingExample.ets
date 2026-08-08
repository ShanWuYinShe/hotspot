/**
 * PageLoadingExample — Page 级加载 / 错误 / 成功三态包装示例
 *
 * 综合演示 Skeleton / SkeletonShimmer / ErrorBoundary 三个组件的协作模式。
 * 实际 Page 可参考此模式：
 *   1. isLoading=true → SkeletonShimmer { Skeleton 占位块 }
 *   2. err 有值 → ErrorBoundary 显示重试 UI
 *   3. 成功 → 渲染 child() 实际页面
 *
 * 用法（Page 内）：
 *   @State isLoading: boolean = true;
 *   @State pageError: string = '';
 *
 *   aboutToAppear() {
 *     this.loadData();
 *   }
 *
 *   build() {
 *     PageLoadingExample({
 *       isLoading: this.isLoading,
 *       err: this.pageError,
 *       onRetry: () => this.loadData()
 *     }) {
 *       this.actualPageContent()
 *     }
 *   }
 *
 * 注意：实际 Page 可拆为 PageLoadingExample + PageContent 两个 @Component，
 * 也可以直接 inline 此模式。本文件作为统一模式参考。
 */

import { DesignConstants } from '../../constants/DesignConstants';
import { Skeleton } from './Skeleton';
import { SkeletonShimmer } from './SkeletonShimmer';
import { ErrorBoundary } from './ErrorBoundary';

@Component
export struct PageLoadingExample {
  @BuilderParam child: () => void = this.defaultChild;
  @Prop isLoading: boolean = false;
  @Prop err: string = '';
  @Prop onRetry: () => void = () => {};

  @Builder
  defaultChild() {
    Column().width(0).height(0)
  }

  build() {
    if (this.isLoading) {
      // 加载态：SkeletonShimmer 包装 3 个 Skeleton 占位
      SkeletonShimmer({ active: true }) {
        Column({ space: DesignConstants.SPACING.LG }) {
          Skeleton({ width: '100%', height: 24, borderRadius: DesignConstants.RADIUS.MD })
          Skeleton({ width: '80%', height: 16, borderRadius: DesignConstants.RADIUS.SM })
          Skeleton({ width: '90%', height: 16, borderRadius: DesignConstants.RADIUS.SM })
          Skeleton({ width: '60%', height: 16, borderRadius: DesignConstants.RADIUS.SM })
          Skeleton({ width: 200, height: 200, borderRadius: DesignConstants.RADIUS.FULL })
        }
        .width('100%')
        .padding(DesignConstants.SPACING.LG)
      }
    } else {
      // 成功 / 错误态：ErrorBoundary 包装 child
      ErrorBoundary({
        err: this.err,
        onRetry: this.onRetry
      }) {
        if (this.child) {
          this.child();
        } else {
          this.defaultChild();
        }
      }
    }
  }
}
