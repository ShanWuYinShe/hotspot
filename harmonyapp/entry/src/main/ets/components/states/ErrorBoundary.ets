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
import { AppIcon } from '../SystemIcon';
import { AppThemeManager } from '../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../core/theme/AppThemeTypes';

@Component
export struct ErrorBoundary {
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @BuilderParam child: () => void = this.defaultChild;
  @Prop err: string = '';
  @Prop onRetry: () => void = () => {};

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  @Builder
  defaultChild() {
    Column().width(0).height(0)
  }

  build() {
    if (this.err) {
      Column({ space: DesignConstants.SPACING.LG }) {
        Text() {
          SymbolSpan($r(AppIcon.WARNING))
            .fontSize(DesignConstants.FONT.SIZE.HERO)
            .fontColor([this.getPalette().error])
        }

        Text('加载失败')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getPalette().accentBackground)

        Text(this.err)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor(this.getPalette().textSecondary)
          .textAlign(TextAlign.Center)
          .constraintSize({ maxWidth: 280 })

        Button('重试')
          .type(ButtonType.Normal)
          .height(DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT_MD)
          .backgroundColor(this.getPalette().accentBackground)
          .fontColor(this.getPalette().accentText)
          .borderRadius(DesignConstants.RADIUS.MD)
          .padding({ left: DesignConstants.SPACING.XL, right: DesignConstants.SPACING.XL })
          .onClick(() => {
            this.onRetry();
          })
      }
      .width('100%')
      .height('100%')
      .justifyContent(FlexAlign.Center)
      .backgroundColor(this.getPalette().pageBackground)
    } else {
      if (this.child) {
        this.child();
      } else {
        this.defaultChild();
      }
    }
  }
}
