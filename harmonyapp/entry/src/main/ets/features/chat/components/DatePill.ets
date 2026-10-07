/**
 * DatePill —— 聊天页日期分隔胶囊（纯展示）。
 * 从 ChatPage.buildDatePill 抽取；主题对象与 surface 边框色经 props 注入，
 * 无本地状态。样式对齐 DesignConstants token（对齐前端 groupMessagesByDay 胶囊）。
 */

import { DesignConstants } from '../../../constants/DesignConstants';
import { AppThemePalette } from '../../../core/theme/AppThemeTypes';

@Component
export struct DatePill {
  @Prop label: string;
  @Prop palette: AppThemePalette;
  @Prop surfaceBorder: string;

  build() {
    Row() {
      Text(this.label)
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .fontColor(this.palette.textSecondary)
        .backgroundColor(this.palette.surfaceSubtle)
        .padding({
          left: DesignConstants.SPACING.MD,
          right: DesignConstants.SPACING.MD,
          top: DesignConstants.SPACING.XS,
          bottom: DesignConstants.SPACING.XS
        })
        .borderRadius(DesignConstants.RADIUS.FULL)
        .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.surfaceBorder })
    }
    .width('100%')
    .justifyContent(FlexAlign.Center)
    .margin({ top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
  }
}
