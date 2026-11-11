/**
 * ProfileGuestView —— 个人中心游客态（对齐 web ProfileMobile.svelte 未登录分支 + 21-guest-me 截图）
 *
 * 结构：
 *   身份式游客卡（封面带 + 字母头像 + 「开启你的城市漫游档案」20/600 + 「访客浏览中」11
 *   + 描述 + 「附近漫游中」+ 三统计 0 + 登录/注册开启 primary + 随便逛逛 soft）
 *   → 「探索推荐」分区（图标 + 标题 14/600 + 副标题 11）
 *   → 3 行导流（探索附近瞬间 / 发现同好群组 / 真实社交雷达），行 = 图标砖 + 标题/描述 + chevron。
 *
 * 不再使用通用 spotlight 空态：游客「我的」按 web 展示个人中心全貌 + 可浏览内容导流，
 * 与「消息」等纯只读页的 NeedsAuthNotice 内联引导区分开。
 */

import { DesignConstants, lineHeightFor, textLineHeight } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LucideIcon } from '../../../components/LucideIcon';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemButton } from '../../../components/SystemButton';
import { ListGroup } from '../../../components/ListGroup';
import { Pressable } from '../../../components/Pressable';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';

interface GuestMetricItem {
  label: string;
  value: number;
}

interface ExploreItem {
  icon: AppIcon;
  title: string;
  description: string;
  onTap: () => void;
}

@Component
export struct ProfileGuestView {
  @Prop resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  onLogin: () => void = () => {};
  onBrowseMoments: () => void = () => {};
  onBrowseGroups: () => void = () => {};
  onBrowsePeople: () => void = () => {};

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  /** web IdentityHero 移动端封面 h-23 = 92px（DesignConstants 无同档，按 token 组合表达） */
  private getCoverHeight(): number {
    return DesignConstants.SPACING.XXL * 3 - DesignConstants.SPACING.XS;
  }

  private getAvatarOverlap(): number {
    return -(DesignConstants.SPACING.XXL + DesignConstants.SPACING.SM);
  }

  @Builder
  private metricColumn(metric: GuestMetricItem, showDivider: boolean) {
    Column({ space: DesignConstants.SPACING.SM }) {
      Text(`${metric.value}`)
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
    .border(showDivider ? {
      width: { left: DesignConstants.BORDER_WIDTH.HAIRLINE },
      color: { left: this.getPalette().borderColor }
    } : { width: 0 })
  }

  @Builder
  private exploreRow(item: ExploreItem) {
    Pressable({
      onPress: () => { item.onTap(); },
      pressScale: DesignConstants.INTERACTION.SCALE_PRESSED_INPUT_ROW,
      hoverFill: this.getPalette().surfaceSubtle
    }) {
      Row({ space: DesignConstants.SPACING.MD3 }) {
        Row() {
          LucideIcon({
            icon: item.icon,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_MEDIA_ACTION,
            color: this.getPalette().textAccentOnSoft
          })
        }
        .width(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL)
        .height(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL)
        .borderRadius(DesignConstants.RADIUS.SM)
        .backgroundColor(this.getPalette().accentMuted)
        .justifyContent(FlexAlign.Center)
        .alignItems(VerticalAlign.Center)

        Column({ space: DesignConstants.SPACING.HALF }) {
          Text(item.title)
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.getPalette().textPrimary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
          Text(item.description)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
            .fontColor(this.getPalette().textSecondary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)

        LucideIcon({
          icon: AppIcon.ARROW_RIGHT,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
          color: this.getPalette().textSecondary
        })
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)
      .padding({
        left: DesignConstants.SPACING.MD,
        right: DesignConstants.SPACING.MD,
        top: DesignConstants.SPACING.MD,
        bottom: DesignConstants.SPACING.MD
      })
    }
  }

  build() {
    Column() {
      // ===== 身份式游客卡 =====
      Column() {
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
          Row({ space: DesignConstants.SPACING.LG }) {
            Stack() {
              SystemAvatar({
                name: '开启你的城市漫游档案',
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
              Text('开启你的城市漫游档案')
                .fontSize(DesignConstants.FONT.SIZE.XL)
                .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XL))
                .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
                .fontColor(this.getPalette().textPrimary)
                .maxLines(2)
                .textOverflow({ overflow: TextOverflow.Ellipsis })
              Text('访客浏览中')
                .fontSize(DesignConstants.FONT.SIZE.XS)
                .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
                .fontColor(this.getPalette().textSecondary)
            }
            .alignItems(HorizontalAlign.Start)
            .layoutWeight(1)
            .padding({ bottom: DesignConstants.SPACING.XS })
          }
          .width('100%')
          .alignItems(VerticalAlign.Bottom)
          .margin({ top: this.getAvatarOverlap() })

          Text('记录街头巷尾的瞬间，发现同频伙伴，建立专属足迹。')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(lineHeightFor(DesignConstants.FONT.SIZE.SM, 1.625))
            .fontColor(this.getPalette().textSecondary)
            .width('100%')
            .margin({ top: DesignConstants.SPACING.MD })

          Row({ space: DesignConstants.SPACING.SM2 }) {
            LucideIcon({
              icon: AppIcon.LOCATION,
              iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
              color: this.getPalette().accentBackground
            })
            Text('附近漫游中')
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
              .fontColor(this.getPalette().textSecondary)
          }
          .alignItems(VerticalAlign.Center)
          .margin({ top: DesignConstants.SPACING.MD2 })

          Column() {
            Row() {
              this.metricColumn({ label: '瞬间', value: 0 }, false)
              this.metricColumn({ label: '群组', value: 0 }, true)
              this.metricColumn({ label: '好友', value: 0 }, true)
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

          Row({ space: DesignConstants.SPACING.SM }) {
            Column() {
              SystemButton({
                label: '登录 / 注册开启',
                variant: 'primary',
                buttonSize: 'default',
                fullWidth: true,
                onPress: () => { this.onLogin(); }
              })
            }
            .layoutWeight(1.25)

            Column() {
              SystemButton({
                label: '随便逛逛',
                variant: 'surface',
                buttonSize: 'default',
                fullWidth: true,
                onPress: () => { this.onBrowseMoments(); }
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

      // ===== 探索推荐 =====
      Row({ space: DesignConstants.SPACING.SM }) {
        LucideIcon({
          icon: AppIcon.SPARKLES,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
          color: this.getPalette().textSecondary
        })
        Column({ space: DesignConstants.SPACING.HALF }) {
          Text('探索推荐')
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
            .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
            .fontColor(this.getPalette().textPrimary)
          Text('发现城市里的精彩瞬间与同好')
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
            .fontColor(this.getPalette().textSecondary)
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)
      .margin({ top: DesignConstants.PAGE_SHELL.SECTION_GAP, bottom: DesignConstants.SPACING.MD })

      ListGroup() {
        this.exploreRow({
          icon: AppIcon.SPARKLES,
          title: '探索附近瞬间',
          description: '查看身边的光影与生活动态',
          onTap: () => { this.onBrowseMoments(); }
        })
        this.exploreRow({
          icon: AppIcon.GROUP,
          title: '发现同好群组',
          description: '加入本地摄影、夜跑与骑行俱乐部',
          onTap: () => { this.onBrowseGroups(); }
        })
        this.exploreRow({
          icon: AppIcon.PERSON,
          title: '真实社交雷达',
          description: '感知附近正在这片街区漫步的伙伴',
          onTap: () => { this.onBrowsePeople(); }
        })
      }
    }
    .width('100%')
    .alignItems(HorizontalAlign.Start)
  }
}
