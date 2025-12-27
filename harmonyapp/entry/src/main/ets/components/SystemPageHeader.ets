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
        .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
        .layoutWeight(1)

      if (this.action) {
        Button(this.action.label)
          .type(ButtonType.Normal)
          .backgroundColor(Color.Transparent)
          .fontColor(this.action.disabled ? DesignConstants.COLOR.TEXT_TERTIARY_LIGHT : DesignConstants.COLOR.BRAND_PRIMARY)
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
    .padding(DesignConstants.SPACING.LG)
    .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
  }
}

