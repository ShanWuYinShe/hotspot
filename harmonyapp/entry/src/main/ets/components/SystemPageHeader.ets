/**
 * SystemPageHeader
 * - 统一页面标题栏样式（Apple-like / HarmonyOS Next）
 * - 只负责“标题 + 可选右侧操作”，避免每个页面重复写 Row/Text/Button 样式
 */

import { DesignConstants } from '../constants/DesignConstants';

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

  build() {
    Row() {
      Text(this.title)
        .fontSize(DesignConstants.FONT.SIZE.XL)
        .fontWeight(FontWeight.Bold)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
        .layoutWeight(1)

      if (this.action) {
        Button(this.action.label)
          .type(ButtonType.Normal)
          .backgroundColor(Color.Transparent)
          .fontColor(this.action.disabled
            ? $r('sys.color.ohos_id_color_text_secondary')
            : $r('sys.color.ohos_id_color_emphasize'))
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .enabled(!this.action.disabled)
          .onClick(async () => {
            try {
              await this.action?.onClick()
            } catch (_) {
              // 交由页面侧 toast/日志处理
            }
          })
      }
    }
    .width('100%')
    .height(56)
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
    .backgroundColor($r('sys.color.ohos_id_color_background'))
  }
}

