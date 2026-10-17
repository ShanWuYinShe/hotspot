/**
 * ProfileIdentityCard —— 个人中心身份卡（头像 + 昵称/handle/位置 + 简介 + 操作行 + 三列统计）。
 * 结构对齐前端 frontend/src/components/profile/mobile/ProfileMobile.svelte（移动端唯一形态）：
 *   头像 xl(80) 圆环 / 昵称 text-xl(20) / handle text-xs / 位置行 icon 14 + text-xs /
 *   简介 text-xs(条件渲染) / 「编辑资料(flex-1) + 设置」卡内操作行 / 三列统计（数值 text-2xl，标签 text-xs）。
 * 2026-10-17 移动端对齐修正（P0-4 + token 漂移）：
 *   1. 移除头像相机角钮：前端同屏只保留一处「编辑资料」入口（镜像入口重复且 affordance 误导），
 *      头像更换统一走「编辑资料」；
 *   2. 「编辑资料」从顶部小钮下移为卡内底部操作行（flex-1），并补「设置」入口——前端
 *      「移动端单手黄金操作区」，设置入口同页只留这一处；
 *   3. 昵称 LG(16)→XL(20)、统计数值 XL(20)→XXL(24)、统计标签 SM(12)→XS(11)、
 *      定位图标 FONT.SIZE.XS(11)→COMPONENT.ICON_SIZE_2XS(14)（图标尺寸禁用字号 token）、
 *      简介 SM(12)→XS(11)、头像 PROFILE(68)→LARGE(80)、卡补发丝描边（前端 border-border-default）；
 *   4. 简介改为条件渲染（前端 {#if bio}）：无简介不再显示占位行。
 * 资料字段以扁平 props 注入（避免跨文件传递可空复合对象），主题按 resolvedThemeId 自解析，
 * 动作回调上抛；无本地状态。
 */

import { DesignConstants, lineHeightFor, textLineHeight } from '../../../constants/DesignConstants';
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
  onEditProfile: () => void = () => {};
  onOpenSettings: () => void = () => {};

  @Builder
  private statItem(label: string, value: number) {
    Column({ space: DesignConstants.SPACING.HALF }) {
      // 数值 text-2xl(24)+SEMIBOLD、标签 text-xs(11)：对齐前端 ProfileMobile 统计条
      Text(formatCount(value))
        .fontSize(DesignConstants.FONT.SIZE.XXL)
        // 前端 text-2xl(24) 行高 32；ArkUI 默认只有 ≈28，统计条会矮一截
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XXL))
        .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
        .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textPrimary)
      Text(label)
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
        .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary)
    }
    .layoutWeight(1)
    .justifyContent(FlexAlign.Center)
    .alignItems(HorizontalAlign.Center)
    .padding({ top: DesignConstants.SPACING.MD3, bottom: DesignConstants.SPACING.MD3 })
  }

  /** 卡内操作行按钮：前端 border + bg-surface-primary + rounded-md + shadow-raised-sm + text-xs(11) + icon 14 */
  @Builder
  private cardAction(icon: AppIcon, label: string, accessibilityLabel: string, onTap: () => void, grow: boolean) {
    Button() {
      Row({ space: DesignConstants.SPACING.SM2 }) {
        LucideIcon({
          icon: icon,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
          color: AppThemeManager.getPaletteById(this.resolvedThemeId).textPrimary
        })
        Text(label)
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textPrimary)
      }
      .alignItems(VerticalAlign.Center)
    }
    .type(ButtonType.Normal)
    .layoutWeight(grow ? 1 : 0)
    .backgroundColor(AppThemeManager.getPaletteById(this.resolvedThemeId).surfaceBackground)
    .border({
      width: DesignConstants.BORDER_WIDTH.HAIRLINE,
      color: AppThemeManager.getPaletteById(this.resolvedThemeId).borderColor
    })
    .borderRadius(DesignConstants.SHAPE.INPUT)
    .shadow(AppThemeManager.getPaletteById(this.resolvedThemeId).controlShadow)
    .padding({
      left: DesignConstants.SPACING.MD,
      right: DesignConstants.SPACING.MD,
      top: DesignConstants.SPACING.SM,
      bottom: DesignConstants.SPACING.SM
    })
    .onClick(() => { onTap(); })
    .accessibilityText(accessibilityLabel)
  }

  build() {
    Column() {
      Row({ space: DesignConstants.SPACING.MD }) {
        SystemAvatar({
          avatarUrl: this.avatarUrl,
          name: this.nickname,
          avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.LARGE,
          shape: 'circle',
          showBorder: true
        })

        Column({ space: DesignConstants.SPACING.THREE }) {
          Text(this.nickname || '未设置昵称')
            .fontSize(DesignConstants.FONT.SIZE.XL)
            // 前端 text-xl(20) 行高 28（Tailwind 20×1.4）；ArkUI 默认只有 ≈23.5
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XL))
            .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
            .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textPrimary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
          if (this.nickname) {
            // 对齐前端：handle 显示 @昵称（前端 ProfileMobile 用 nickname，非 user_id）
            Text(`@${this.nickname}`)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
              .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary)
          }
          Row({ space: DesignConstants.SPACING.SM2 }) {
            LucideIcon({
              icon: AppIcon.LOCATION,
              iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
              color: AppThemeManager.getPaletteById(this.resolvedThemeId).accentBackground
            })
            Text(this.locationName || '未设置位置')
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
              .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary)
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
          }
          .alignItems(VerticalAlign.Center)
          .margin({ top: DesignConstants.SPACING.SM })
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)
      }
      .width('100%')
      .alignItems(VerticalAlign.Top)

      if (this.bio) {
        Text(this.bio)
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary)
          // 简介属长文：原 MD(22)/12→SM(12) 后按 --leading-copy(1.75) 收敛；字号对齐前端 text-xs(11)
          .lineHeight(lineHeightFor(DesignConstants.FONT.SIZE.XS, DesignConstants.FONT.LINE_RATIO.COPY))
          .width('100%')
          .margin({ top: DesignConstants.SPACING.LG })
      }

      // 移动端单手操作区：编辑资料（撑满）+ 设置（自适应），前端 mt-4 pt-1 flex gap-2
      Row({ space: DesignConstants.SPACING.SM }) {
        this.cardAction(AppIcon.EDIT, '编辑资料', '编辑资料', this.onEditProfile, true)
        this.cardAction(AppIcon.SETTINGS, '设置', '系统设置', this.onOpenSettings, false)
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)
      .margin({ top: DesignConstants.SPACING.LG })

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
    // 前端卡片壳：白卡 + 发丝描边 + 主题投影（原端侧只有投影，缺描边）
    .border({
      width: DesignConstants.BORDER_WIDTH.HAIRLINE,
      color: AppThemeManager.getPaletteById(this.resolvedThemeId).borderColor
    })
    .shadow(AppThemeManager.getPaletteById(this.resolvedThemeId).surfaceShadow)
  }
}
