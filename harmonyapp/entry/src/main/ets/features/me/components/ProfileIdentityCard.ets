/**
 * ProfileIdentityCard —— 个人中心身份封面卡（对齐 web components/profile/IdentityHero.svelte 移动端）
 *
 * 结构（R13「叙事封面」母题 B）：
 *   封面带 92（主题 accent-soft 渐变，无图时不铺图）
 *   → 头像 68 + 4 ring（压封面 -40）+ 昵称 20/600 + handle 11
 *   → 简介 12 + 位置行（MapPin 14 + 11）
 *   → 发丝线 + 三列统计（数字 24/600 leading-none + 标签 11/500，列间发丝竖线）
 *   → 动作整行（编辑资料 primary + 设置 soft，等宽 42 高，落在大拇指区）
 *
 * 说明：web 身份卡封面高度 92 在 DesignConstants 中无同档 token，按 4 基准间距组合表达；
 * 头像取 AVATAR_SIZE.PROFILE(68)，与 web ProfileMobile 68 同档（IdentityHero 移动端 72 的
 * 平台适配差异，圆环 +4 后整体视觉尺寸一致）。
 */

import { DesignConstants, lineHeightFor, textLineHeight } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LucideIcon } from '../../../components/LucideIcon';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemButton } from '../../../components/SystemButton';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { formatCount } from '../../../core/utils/FormatUtil';
import { DEFAULT_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';

interface ProfileMetricItem {
  label: string;
  value: number;
}

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

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  /** web IdentityHero 移动端封面 h-23 = 92px（DesignConstants 无同档，按 token 组合表达） */
  private getCoverHeight(): number {
    return DesignConstants.SPACING.XXL * 3 - DesignConstants.SPACING.XS;
  }

  /** 头像环压封面 -40（web -mt-9 36 + ring-4） */
  private getAvatarOverlap(): number {
    return -(DesignConstants.SPACING.XXL + DesignConstants.SPACING.SM);
  }

  @Builder
  private metricColumn(metric: ProfileMetricItem, showDivider: boolean) {
    Column({ space: DesignConstants.SPACING.SM }) {
      Text(formatCount(metric.value))
        // web text-2xl(24) font-semibold leading-none tabular-nums
        .fontSize(DesignConstants.FONT.SIZE.XXL)
        .lineHeight(DesignConstants.FONT.SIZE.XXL)
        .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
        .fontColor(this.getPalette().textPrimary)
      Text(metric.label)
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
        .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
        .fontColor(this.getPalette().textSecondary)
    }
    .layoutWeight(1)
    .alignItems(HorizontalAlign.Center)
    .justifyContent(FlexAlign.Center)
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG
    })
    // web divide-x divide-border-subtle：列间发丝竖线（首列不画）
    .border(showDivider ? {
      width: { left: DesignConstants.BORDER_WIDTH.HAIRLINE },
      color: { left: this.getPalette().borderColor }
    } : { width: 0 })
  }

  build() {
    Column() {
      // ===== 封面带（web h-23；主题 accent-soft → 表面色 柔和渐变） =====
      Column()
        .width('100%')
        .height(this.getCoverHeight())
        .linearGradient({
          direction: GradientDirection.Bottom,
          colors: [
            [this.getPalette().accentMuted, 0.0],
            [this.getPalette().surfaceIsland, 1.0]
          ]
        })

      Column() {
        // ===== 身份行：头像压封面 + 昵称/handle（底对齐） =====
        Row({ space: DesignConstants.SPACING.LG }) {
          Stack() {
            SystemAvatar({
              avatarUrl: this.avatarUrl,
              name: this.nickname,
              avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.PROFILE,
              shape: 'circle',
              decorative: true
            })
          }
          .padding(DesignConstants.SPACING.XS)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .backgroundColor(this.getPalette().pageBackground)
          .shadow(this.getPalette().surfaceShadow)

          Column({ space: DesignConstants.SPACING.HALF }) {
            Text(this.nickname || '未设置昵称')
              .fontSize(DesignConstants.FONT.SIZE.XL)
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XL))
              .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
              .fontColor(this.getPalette().textPrimary)
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
            if (this.nickname) {
              Text(`@${this.nickname}`)
                .fontSize(DesignConstants.FONT.SIZE.XS)
                .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
                .fontColor(this.getPalette().textSecondary)
                .maxLines(1)
                .textOverflow({ overflow: TextOverflow.Ellipsis })
            }
          }
          .alignItems(HorizontalAlign.Start)
          .layoutWeight(1)
          .padding({ bottom: DesignConstants.SPACING.XS })
        }
        .width('100%')
        .alignItems(VerticalAlign.Bottom)
        .margin({ top: this.getAvatarOverlap() })

        // ===== 简介（12 / leading-relaxed） =====
        if (this.bio) {
          Text(this.bio)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(lineHeightFor(DesignConstants.FONT.SIZE.SM, 1.625))
            .fontColor(this.getPalette().textSecondary)
            .width('100%')
            .margin({ top: DesignConstants.SPACING.MD })
        }

        // ===== 位置 =====
        Row({ space: DesignConstants.SPACING.SM2 }) {
          LucideIcon({
            icon: AppIcon.LOCATION,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
            color: this.getPalette().accentBackground
          })
          Text(this.locationName || '未设置位置')
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
            .fontColor(this.getPalette().textSecondary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .alignItems(VerticalAlign.Center)
        .margin({ top: DesignConstants.SPACING.MD2 })

        // ===== 发丝线 + 三列统计（web border-t mt-4 pt-4） =====
        Column() {
          Row() {
            this.metricColumn({ label: '瞬间', value: this.momentsCount }, false)
            this.metricColumn({ label: '粉丝', value: this.followersCount }, true)
            this.metricColumn({ label: '群组', value: this.groupsCount }, true)
          }
          .width('100%')
          .alignItems(VerticalAlign.Center)
        }
        .width('100%')
        .padding({ top: DesignConstants.SPACING.LG })
        .margin({ top: DesignConstants.SPACING.LG })
        .border({
          width: { top: DesignConstants.BORDER_WIDTH.HAIRLINE },
          color: { top: this.getPalette().borderColor }
        })

        // ===== 动作行：编辑资料 primary + 设置 soft（web 卡内底部整行） =====
        Row({ space: DesignConstants.SPACING.SM }) {
          Column() {
            SystemButton({
              label: '编辑资料',
              icon: AppIcon.EDIT,
              showIcon: true,
              variant: 'primary',
              buttonSize: 'default',
              fullWidth: true,
              onPress: () => { this.onEditProfile(); }
            })
          }
          .layoutWeight(1)

          Column() {
            SystemButton({
              label: '设置',
              icon: AppIcon.SETTINGS,
              showIcon: true,
              variant: 'surface',
              buttonSize: 'default',
              fullWidth: true,
              onPress: () => { this.onOpenSettings(); }
            })
          }
          .layoutWeight(1)
        }
        .width('100%')
        .alignItems(VerticalAlign.Center)
        .margin({ top: DesignConstants.SPACING.LG })
      }
      .width('100%')
      .alignItems(HorizontalAlign.Start)
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        bottom: DesignConstants.SPACING.LG2
      })
    }
    .width('100%')
    .alignItems(HorizontalAlign.Start)
    .borderRadius(DesignConstants.SHAPE.CARD)
    .backgroundColor(this.getPalette().surfaceIsland)
    .shadow(this.getPalette().surfaceShadow)
    .clip(true)
  }
}
