/**
 * ProfileDataSection —— 个人中心「个人数据」分组（分组小标 + 卡片化数据行）。
 * 从 ProfilePage.buildProfileDataList/buildProfileDataRow 抽取；计数经 props 注入，
 * 主题按 resolvedThemeId 自解析（与 ProfileContentAtoms 同规），行内导航走 AppRouter；无本地状态。
 */

import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LucideIcon } from '../../../components/LucideIcon';
import { Pressable } from '../../../components/Pressable';
import { AppRouter } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { formatCount } from '../../../core/utils/FormatUtil';

@Component
export struct ProfileDataSection {
  @Prop resolvedThemeId: string = 'vibrant_light';
  @Prop myGroupsCount: number = 0;
  @Prop myFriendsCount: number = 0;
  @Prop myFollowingCount: number = 0;
  @Prop myFollowersCount: number = 0;
  /** 追加动作行（更新位置/升级/设置/退出登录等）：与数据行同卡拼接，顶部 hairline 分隔 */
  @BuilderParam actionRows: () => void;
  @Prop hasActionRows: boolean = false;

  @Builder
  private dataRow(icon: AppIcon, label: string, detail: string, onTap: () => void, divider: boolean) {
    // 对齐前端 ProfileMobile：圆形 icon-badge(32) + 左侧两行（标题/副标题）+ 右箭头；
    // 按压反馈统一走 Pressable（P3-1），键鼠 hover 淡填充由 hoverFill 提供
    Pressable({
      onPress: () => { onTap(); },
      pressScale: DesignConstants.INTERACTION.SCALE_PRESSED_INPUT_ROW,
      hoverFill: AppThemeManager.getPaletteById(this.resolvedThemeId).accentMuted
    }) {
      Row({ space: DesignConstants.SPACING.MD }) {
        Row() {
          LucideIcon({
            icon: icon,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
            color: AppThemeManager.getPaletteById(this.resolvedThemeId).accentBackground
          })
        }
        .width(DesignConstants.COMPONENT.AVATAR_SIZE.XS)
        .height(DesignConstants.COMPONENT.AVATAR_SIZE.XS)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .backgroundColor(AppThemeManager.getPaletteById(this.resolvedThemeId).accentMuted)
        .justifyContent(FlexAlign.Center)

        Column({ space: DesignConstants.SPACING.HALF }) {
          Text(label)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textPrimary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
          Text(detail)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)

        LucideIcon({
          icon: AppIcon.ARROW_RIGHT,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
          color: AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary
        })
      }
      .width('100%')
      .padding({
        left: DesignConstants.SPACING.LG2,
        right: DesignConstants.SPACING.LG2,
        top: DesignConstants.SPACING.MD3,
        bottom: DesignConstants.SPACING.MD3
      })
      .border(divider ? {
        width: { bottom: DesignConstants.BORDER_WIDTH.HAIRLINE },
        color: { bottom: AppThemeManager.getPaletteById(this.resolvedThemeId).borderColor }
      } : { width: 0 })
    }
  }

  build() {
    Column() {
      // 分组小标（对齐前端 ProfileMobile 改版：eyebrow + 副文案在列表外，列表去外框）
      Text('个人数据')
        .fontSize(DesignConstants.FONT.SIZE.XS2)
        .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
        .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary)
        .letterSpacing(DesignConstants.FONT.LETTER_SPACING.LABEL)
        .textCase(TextCase.UpperCase)
        .width('100%')
        .padding({ left: DesignConstants.SPACING.SM2, bottom: DesignConstants.SPACING.SM })
      Text('在这里查看你的内容与关系')
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary)
        .width('100%')
        .padding({ left: DesignConstants.SPACING.SM2, bottom: DesignConstants.SPACING.SM })

      // 数据行白卡容器（表面分层：底色差 + 阴影浮起，无边框线，对齐前端 card-raised）
      Column() {
        this.dataRow(AppIcon.PHOTO, '我的瞬间', '查看已发布的记录', () => {
          AppRouter.push(PageNames.MOMENT);
        }, true)
        this.dataRow(AppIcon.GROUP, '我的群组', `${formatCount(this.myGroupsCount)} 个正在参与`, () => {
          AppRouter.push(PageNames.GROUP);
        }, false)
        this.dataRow(AppIcon.PERSON, '好友', `${formatCount(this.myFriendsCount)} 位好友`, () => {
          AppRouter.push(PageNames.RELATIONSHIP_HUB);
        }, false)
        this.dataRow(AppIcon.STAR, '关注', `${formatCount(this.myFollowingCount)} 人`, () => {
          AppRouter.push(PageNames.RELATIONSHIP_HUB);
        }, false)
        this.dataRow(AppIcon.HEART, '粉丝', `${formatCount(this.myFollowersCount)} 人`, () => {
          AppRouter.push(PageNames.RELATIONSHIP_HUB);
        }, false)
        if (this.hasActionRows) {
          // 动作行容器：顶部 hairline 与数据行分隔（行内各自再带底部线）
          Column() {
            this.actionRows()
          }
          .width('100%')
          .border({
            width: { top: DesignConstants.BORDER_WIDTH.HAIRLINE },
            color: { top: AppThemeManager.getPaletteById(this.resolvedThemeId).borderColor }
          })
        }
      }
      .width('100%')
      .backgroundColor(AppThemeManager.getPaletteById(this.resolvedThemeId).surfaceBackground)
      .borderRadius(DesignConstants.SHAPE.CARD)
      .shadow(AppThemeManager.getPaletteById(this.resolvedThemeId).surfaceShadow)
      .clip(true)
    }
    .width('100%')
    .alignItems(HorizontalAlign.Start)
    .margin({ top: DesignConstants.SPACING.LG })
  }
}
