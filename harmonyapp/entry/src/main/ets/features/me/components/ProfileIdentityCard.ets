/**
 * ProfileIdentityCard —— 个人中心身份卡（头像 + 昵称/handle/位置 + 简介 + 三列统计）。
 * 从 ProfilePage.buildIdentityCard/buildIdentityStat 抽取；资料字段以扁平 props 注入
 * （避免跨文件传递可空复合对象），主题按 resolvedThemeId 自解析（与 ProfileContentAtoms 同规），
 * 动作回调上抛；无本地状态。
 */

import { DesignConstants, lineHeightFor } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LucideIcon } from '../../../components/LucideIcon';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { formatCount } from '../../../core/utils/FormatUtil';
import { DEFAULT_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';

@Component
export struct ProfileIdentityCard {
  @Prop resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  @Prop avatarUrl: string = '';
  @Prop nickname: string = '';
  @Prop locationName: string = '';
  @Prop bio: string = '';
  @Prop momentsCount: number = 0;
  @Prop followersCount: number = 0;
  @Prop groupsCount: number = 0;
  onChangeAvatar: () => void = () => {};
  onEditProfile: () => void = () => {};

  @Builder
  private statItem(label: string, value: number) {
    Row() {
      // Gemini 审查（2026-10-14）：数值与标签层级拉不开——数字升 XL(20)+SEMIBOLD
    // （与群组详情 StatsCell 同款统计字阶，跨页统一），标签升 SM(12) 次要色
    Column({ space: DesignConstants.SPACING.XS }) {
        Text(formatCount(value))
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
          .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textPrimary)
        Text(label)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary)
      }
      .layoutWeight(1)
      .justifyContent(FlexAlign.Center)
      .alignItems(HorizontalAlign.Center)
    }
    .layoutWeight(1)
    .padding({ top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.SM })
  }

  build() {
    Column() {
      Row({ space: DesignConstants.SPACING.MD }) {
        Stack({ alignContent: Alignment.BottomEnd }) {
          SystemAvatar({
            avatarUrl: this.avatarUrl,
            name: this.nickname,
            avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.PROFILE,
            shape: 'circle',
            showBorder: true
          })

          // 相机角钮（24vp accent 实底，点击换头像）
          Button() {
            LucideIcon({
              icon: AppIcon.CAMERA,
              iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
              paletteRole: 'accentText'
            })
          }
          .type(ButtonType.Normal)
          .width(DesignConstants.COMPONENT.ICON_SIZE_SM)
          .height(DesignConstants.COMPONENT.ICON_SIZE_SM)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .backgroundColor(AppThemeManager.getPaletteById(this.resolvedThemeId).accentBackground)
          .border({
            width: DesignConstants.BORDER_WIDTH.STRONG,
            color: AppThemeManager.getPaletteById(this.resolvedThemeId).pageBackground
          })
          .onClick(() => this.onChangeAvatar())
          .accessibilityText('更换头像')
          // 触控热区：视觉 24 → 命中区 44（responseRegion 不占布局）
          .responseRegion({
            x: -DesignConstants.SPACING.SM,
            y: -DesignConstants.SPACING.SM,
            width: DesignConstants.LAYOUT.MIN_HIT_TARGET,
            height: DesignConstants.LAYOUT.MIN_HIT_TARGET
          })
        }

        Column({ space: DesignConstants.SPACING.THREE }) {
          Text(this.nickname || '未命名用户')
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
            .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textPrimary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
          if (this.nickname) {
            // 对齐前端：handle 显示 @昵称（前端 ProfileMobile 用 nickname，非 user_id）
            Text(`@${this.nickname}`)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
              .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary)
          }
          if (this.locationName) {
            Row({ space: DesignConstants.SPACING.HALF }) {
              LucideIcon({
                icon: AppIcon.LOCATION,
                iconSize: DesignConstants.FONT.SIZE.XS,
                color: AppThemeManager.getPaletteById(this.resolvedThemeId).accentBackground
              })
              Text(this.locationName)
                .fontSize(DesignConstants.FONT.SIZE.XS)
                .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary)
            }
            .alignItems(VerticalAlign.Center)
          }
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)

        // 编辑资料描边小钮
        Button('编辑资料')
          .type(ButtonType.Normal)
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textPrimary)
          .backgroundColor(AppThemeManager.getPaletteById(this.resolvedThemeId).surfaceBackground)
          .border({
            width: DesignConstants.BORDER_WIDTH.HAIRLINE,
            color: AppThemeManager.getPaletteById(this.resolvedThemeId).borderColor
          })
          .borderRadius(DesignConstants.SHAPE.INPUT)
          .padding({
            left: DesignConstants.SPACING.MD2,
            right: DesignConstants.SPACING.MD2,
            top: DesignConstants.SPACING.SM,
            bottom: DesignConstants.SPACING.SM
          })
          .onClick(() => this.onEditProfile())
      }
      .width('100%')
      .alignItems(VerticalAlign.Top)

      if (this.bio) {
        Text(this.bio)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary)
          // 简介属长文：原 MD(22)/12=1.83 偏松 → 对齐前端 --leading-copy(1.75)=21
          .lineHeight(lineHeightFor(DesignConstants.FONT.SIZE.SM, DesignConstants.FONT.LINE_RATIO.COPY))
          .width('100%')
          .margin({ top: DesignConstants.SPACING.LG })
      } else {
        // 对齐前端：无简介时显示占位文案（自己的资料页用第一人称）
        Text('还没有填写简介，介绍一下自己吧')
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary)
          .width('100%')
          .margin({ top: DesignConstants.SPACING.LG })
      }

      // 卡底三列统计（border-t，列间留白——竖线在小屏三列里显得切割局促）
      Row() {
        this.statItem('瞬间', this.momentsCount)
        this.statItem('关注者', this.followersCount)
        this.statItem('群组', this.groupsCount)
      }
      .width('100%')
      .border({
        width: { top: DesignConstants.BORDER_WIDTH.HAIRLINE },
        color: { top: AppThemeManager.getPaletteById(this.resolvedThemeId).borderColor }
      })
      .margin({ top: DesignConstants.SPACING.LG })
    }
    .width('100%')
    .alignItems(HorizontalAlign.Start)
    .padding(DesignConstants.SPACING.LG2)
    .borderRadius(DesignConstants.SHAPE.CARD)
    .backgroundColor(AppThemeManager.getPaletteById(this.resolvedThemeId).surfaceBackground)
    .shadow(AppThemeManager.getPaletteById(this.resolvedThemeId).surfaceShadow)
  }
}
