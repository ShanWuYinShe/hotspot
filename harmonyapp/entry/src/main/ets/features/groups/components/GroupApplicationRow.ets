/**
 * GroupApplicationRow —— 入群申请行（申请人 + 留言 + 批准/拒绝）。
 * 从 GroupDetailPage.buildApplicationItem 抽取；动作经回调上抛页面。
 */

import { DesignConstants } from '../../../constants/DesignConstants';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemButton } from '../../../components/SystemButton';
import { AppThemePalette } from '../../../core/theme/AppThemeTypes';
import { GroupApplicationItem } from '../../../core/types/api';
import { formatDateTime } from '../../../core/utils/FormatUtil';

@Component
export struct GroupApplicationRow {
  @Prop application: GroupApplicationItem;
  @Prop handling: boolean = false;
  palette: AppThemePalette | null = null;
  surfaceBackground: string = '';
  onApprove: () => void = () => {};
  onReject: () => void = () => {};

  build() {
    Column({ space: DesignConstants.SPACING.SM }) {
      Row() {
        SystemAvatar({
          src: this.application.applicant?.avatar_url ?? '',
          name: this.application.applicant?.nickname ?? '',
          avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL,
          userInfo: this.application.applicant
        })

        Column() {
          Text(this.application.applicant?.nickname ?? '未知用户')
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.palette?.textPrimary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })

          Text(`申请于 ${formatDateTime(this.application.created_at)}`)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.palette?.textSecondary)
            .margin({ top: DesignConstants.SPACING.XS })
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)
        .margin({ left: DesignConstants.SPACING.MD })
      }
      .width('100%')

      if (this.application.message) {
        Text(this.application.message)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.palette?.textSecondary)
          .width('100%')
      }

      Row({ space: DesignConstants.SPACING.SM }) {
        SystemButton({
          label: this.handling ? '处理中…' : '批准',
          variant: 'primary',
          buttonSize: 'large',
          disabled: this.handling,
          onPress: () => { this.onApprove(); }
        })
          .layoutWeight(1)

        SystemButton({
          label: '拒绝',
          variant: 'surface',
          buttonSize: 'large',
          disabled: this.handling,
          onPress: () => { this.onReject(); }
        })
          .layoutWeight(1)
      }
      .width('100%')
    }
    .width('100%')
    .padding(DesignConstants.SPACING.MD)
    .backgroundColor(this.surfaceBackground)
    .borderRadius(DesignConstants.RADIUS.XL)
    .shadow(this.palette?.controlShadow)
    .margin({ bottom: DesignConstants.SPACING.SM })
  }
}
