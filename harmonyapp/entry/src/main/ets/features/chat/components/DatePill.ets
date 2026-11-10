/**
 * DatePill —— 聊天页日期分隔标签（纯展示）。
 * 从 ChatPage.buildDatePill 抽取；主题对象经 props 注入，无本地状态。
 *
 * 对齐前端 ChatDatePill：中性轻量分节标签——无底色、无投影、无圆角控件外观，
 * 只有 11vp 次色文字居中（py-1.5）。旧实现是「白底 + 描边 + 胶囊」，在聊天画布上
 * 看起来像一个可点的按钮（前端同款降噪已记录）。
 */

import { DesignConstants } from '../../../constants/DesignConstants';
import { AppThemePalette } from '../../../core/theme/AppThemeTypes';

@Component
export struct DatePill {
  @Prop label: string;
  @Prop palette: AppThemePalette;

  build() {
    Row() {
      Text(this.label)
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
        .fontColor(this.palette.textSecondary)
        .maxLines(1)
    }
    .width('100%')
    .justifyContent(FlexAlign.Center)
    // web py-1.5(6)：日期退成「阅读坐标」而不是控件
    .padding({
      top: DesignConstants.SPACING.SM2,
      bottom: DesignConstants.SPACING.SM2
    })
  }
}
