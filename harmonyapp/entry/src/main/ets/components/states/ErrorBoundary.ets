/**
 * ErrorBoundary — 错误状态包装（ArkUI 风格）
 *
 * ArkUI 没有 React ErrorBoundary 概念；此组件是"包装器"模式：
 * 上层在 onError / try-catch 捕获错误后，将 err 状态传给本组件，
 * 内部根据 err 是否存在渲染 child() 或 fallback UI。
 *
 * 用法：
 *   ErrorBoundary({
 *     err: this.pageError,
 *     onRetry: () => this.retry()
 *   }) {
 *     this.actualPageContent()
 *   }
 *
 * 真正的错误捕获：
 * - Page 渲染错误：父组件用 try/catch 包裹业务方法
 * - 资源加载错误：Image / Video 等组件的 .onError 事件
 * - 全局错误：EntryAbility 通过 errorManager.on('error', ...) 监听
 *
 * 与 frontend /frontend/src/lib/components/ui/ErrorBoundary.tsx 对齐
 */

import { DesignConstants } from '../../constants/DesignConstants';

@Component
export struct ErrorBoundary {
  @BuilderParam child: () => void = this.defaultChild;
  @Prop err: string = '';
  @Prop onRetry: () => void = () => {};

  @Builder
  defaultChild() {
    Column().width(0).height(0)
  }

  build() {
    if (this.err) {
      Column({ space: 16 }) {
        Text() {
          SymbolSpan($r('sys.symbol.exclamationmark_triangle_fill'))
            .fontSize(DesignConstants.FONT.SIZE.HERO)
            .fontColor([$r('app.color.color_danger')])
        }

        Text('加载失败')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Medium)
          .fontColor($r('app.color.brand_primary'))

        Text(this.err)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor($r('app.color.chip_inactive_text'))
          .textAlign(TextAlign.Center)
          .constraintSize({ maxWidth: 280 })

        Button('重试')
          .type(ButtonType.Normal)
          .height(40)
          .backgroundColor($r('app.color.brand_primary'))
          .fontColor($r('app.color.bg_primary'))
          .borderRadius(DesignConstants.RADIUS.MD)
          .padding({ left: 24, right: 24 })
          .onClick(() => {
            this.onRetry();
          })
      }
      .width('100%')
      .height('100%')
      .justifyContent(FlexAlign.Center)
      .backgroundColor($r('app.color.bg_primary'))
    } else {
      if (this.child) {
        this.child();
      } else {
        this.defaultChild();
      }
    }
  }
}
