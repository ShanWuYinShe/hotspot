/**
 * SystemPageHeader
 * - 统一页面标题栏样式（Apple-like / HarmonyOS Next）
 * - 只负责“标题 + 可选右侧操作”，避免每个页面重复写 Row/Text/Button 样式
 */

import { DesignConstants } from '../constants/DesignConstants';
import { AppIcon } from './SystemIcon';
import { SafeAreaUtils } from '../core/utils/SafeAreaUtils';

export interface PageHeaderAction {
  label: string;
  onClick: () => void | Promise<void>;
  // 轻量扩展：有需要再加 icon / disabled / loading
  disabled?: boolean;
}

@Component
export struct SystemPageHeader {
  title: string = '';
  action?: PageHeaderAction;
  showBackButton: boolean = false;
  onBack?: () => void;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  surfaceColor?: ResourceColor;

  private getHeaderTopPadding(): number {
    return SafeAreaUtils.resolveTopPadding(this.safeAreaTop);
  }

  private getBackgroundColor(): ResourceColor {
    return this.surfaceColor ? this.surfaceColor : $r('app.color.bg_primary');
  }

  build() {
    Row({ space: DesignConstants.SPACING.SM }) {
      if (this.showBackButton) {
        Button() {
          Text() {
            SymbolSpan($r(AppIcon.BACK))
              .fontSize(24)
              .fontColor([$r('sys.color.ohos_id_color_text_primary')])
          }
        }
        .type(ButtonType.Normal)
        .backgroundColor($r('app.color.bg_tertiary'))
        .borderRadius(DesignConstants.RADIUS.XXL)
        .border({ width: 1, color: $r('sys.color.ohos_id_color_list_separator') })
        .width(40)
        .height(40)
        .onClick(() => {
          if (this.onBack) {
            this.onBack();
          }
        })
      } else {
        Blank().width(16)
      }

      Text(this.title)
        .fontSize(DesignConstants.FONT.SIZE.XL)
        .fontWeight(FontWeight.Medium)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
        .layoutWeight(1)

      if (this.action) {
        Button(this.action.label)
          .type(ButtonType.Normal)
          .backgroundColor($r('app.color.bg_secondary'))
          .borderRadius(DesignConstants.RADIUS.XXL)
          .border({ width: 1, color: $r('sys.color.ohos_id_color_list_separator') })
          .height(36)
          .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD })
          .fontColor(this.action.disabled
            ? $r('sys.color.ohos_id_color_text_secondary')
            : $r('sys.color.ohos_id_color_emphasize'))
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .enabled(!this.action.disabled)
          .onClick(async () => {
            try {
              await this.action?.onClick()
            } catch (_) {
            }
          })
      } else {
        Blank().width(24)
      }
    }
    .width('100%')
    .alignItems(VerticalAlign.Center)
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: this.getHeaderTopPadding(),
      bottom: DesignConstants.SPACING.MD
    })
    .backgroundColor(this.getBackgroundColor())
    .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])
  }
}
