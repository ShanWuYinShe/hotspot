/**
 * GroupMemberRow —— 群成员列表行（头像 + 昵称 + 角色胶囊 + 加入时间 + 操作菜单）。
 * 从 GroupDetailPage.buildMemberItem 抽取；主题与角色文案/颜色经 props 注入，
 * 菜单项由页面构建（动作闭包留在页面）。
 */

import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { AppThemePalette } from '../../../core/theme/AppThemeTypes';
import { GroupMemberItem } from '../../../core/types/api';
import { formatDate } from '../../../core/utils/FormatUtil';
import { LucideIcon } from '../../../components/LucideIcon';

@Component
export struct GroupMemberRow {
  @Prop member: GroupMemberItem;
  @Prop roleText: string = '';
  @Prop roleColor: ResourceColor = '';
  @Prop showMenu: boolean = false;
  palette: AppThemePalette | null = null;
  surfaceBackground: string = '';
  /** 页面构建的菜单（bindMenu 需要 MenuElement[]，动作闭包在页面侧） */
  menuItems: MenuElement[] = [];

  build() {
    Row() {
      SystemAvatar({
        src: this.member.user?.avatar_url ?? '',
        name: this.member.user?.nickname ?? '',
        avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL,
      })

      Column() {
        Text(this.member.user?.nickname ?? '')
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.palette?.textPrimary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        Row() {
          Text(this.roleText)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.roleColor)
            .backgroundColor(this.palette?.surfaceSubtle)
            .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM, top: DesignConstants.SPACING.HALF, bottom: DesignConstants.SPACING.HALF })
            .borderRadius(DesignConstants.SHAPE.CHIP)

          if (this.member.joined_at) {
            Text(`加入于 ${formatDate(this.member.joined_at)}`)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor(this.palette?.textSecondary)
              .margin({ left: DesignConstants.SPACING.SM })
          }
        }
        .margin({ top: DesignConstants.SPACING.XS })
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)
      .margin({ left: DesignConstants.SPACING.MD })

      if (this.showMenu) {
        LucideIcon({
            icon: AppIcon.MORE,
            iconSize: DesignConstants.FONT.SIZE.LG_PLUS,
            color: this.palette?.textSecondary
          })
        .padding(DesignConstants.SPACING.SM)
        .borderRadius(DesignConstants.RADIUS.SM)
        .bindMenu(this.menuItems)
      }
    }
    .width('100%')
    .padding(DesignConstants.SPACING.MD)
    .backgroundColor(this.surfaceBackground)
    .borderRadius(DesignConstants.RADIUS.XL)
    .shadow(this.palette?.controlShadow)
    .margin({ bottom: DesignConstants.SPACING.SM })
  }
}
