/**
 * ProfileCompletenessCard —— 个人中心「资料状态」卡（对齐 web components/profile/ProfileCompleteness.svelte）
 *
 * 结构：分区头（盾牌图标 + 标题 14/600 + 计数 11）→ 软岛卡
 *   未完成：完整度数字 24/600 + 「资料完整度」11 → 分段点进度条（5 段 6 高胶囊）
 *           → 提示 11 → 发丝线 → 「建议补全」10 眉标 → 缺失项行（描边圆 14 + 文案 11 + chevron）
 *   已完成：绿色软底圆 40（✓ 20）+「资料已完善」12/500 + 说明 11。
 *
 * 数据由调用方按资料接口实际字段组装（头像/昵称/简介/位置/群组），本组件只渲染。
 */

import { DesignConstants, lineHeightFor, textLineHeight } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LucideIcon } from '../../../components/LucideIcon';
import { Pressable } from '../../../components/Pressable';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { AppThemeVisuals } from '../../../core/theme/AppThemeVisuals';
import { DEFAULT_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';

export interface ProfileCompletenessItem {
  id: string;
  label: string;
  done: boolean;
}

@Component
export struct ProfileCompletenessCard {
  @Prop resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  @Prop items: ProfileCompletenessItem[] = [];
  onItemTap: (id: string) => void = () => {};

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getDoneCount(): number {
    let count = 0;
    for (const item of this.items) {
      if (item.done) {
        count += 1;
      }
    }
    return count;
  }

  private getPercent(): number {
    if (this.items.length === 0) {
      return 100;
    }
    return Math.round((this.getDoneCount() / this.items.length) * 100);
  }

  @Builder
  private missingRow(item: ProfileCompletenessItem) {
    Pressable({
      onPress: () => { this.onItemTap(item.id); },
      pressScale: DesignConstants.INTERACTION.SCALE_PRESSED_INPUT_ROW,
      hoverFill: this.getPalette().surfaceSubtle
    }) {
      Row({ space: DesignConstants.SPACING.SM }) {
        // web lucide Circle 14（空心圆）：本端用描边圆表达，避免引入新图标
        Column()
          .width(DesignConstants.COMPONENT.ICON_SIZE_2XS)
          .height(DesignConstants.COMPONENT.ICON_SIZE_2XS)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .border({ width: DesignConstants.BORDER_WIDTH.REGULAR, color: this.getPalette().borderColor })
        Text(item.label)
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
          .fontColor(this.getPalette().textPrimary)
          .layoutWeight(1)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
        LucideIcon({
          icon: AppIcon.ARROW_RIGHT,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
          color: this.getPalette().textSecondary
        })
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)
      .padding({
        top: DesignConstants.SPACING.SM,
        bottom: DesignConstants.SPACING.SM
      })
    }
  }

  @Builder
  private missingList() {
    Column({ space: DesignConstants.SPACING.HALF }) {
      ForEach(this.items.filter((item: ProfileCompletenessItem) => !item.done), (item: ProfileCompletenessItem) => {
        this.missingRow(item)
      }, (item: ProfileCompletenessItem) => item.id)
    }
    .width('100%')
    .margin({ top: DesignConstants.SPACING.SM2 })
  }

  build() {
    Column() {
      // 分区头：资料状态 + 完成计数（web SectionHead icon=ShieldCheck count=x/5 项）
      Row({ space: DesignConstants.SPACING.SM }) {
        LucideIcon({
          icon: AppIcon.SHIELD,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
          color: this.getPalette().textSecondary
        })
        Text('资料状态')
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
          .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
          .fontColor(this.getPalette().textPrimary)
        Text(this.getDoneCount().toString() + '/' + this.items.length.toString() + ' 项')
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getPalette().textSecondary)
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)
      .margin({ bottom: DesignConstants.SPACING.MD })

      // 软岛卡
      Column() {
        if (this.getDoneCount() === this.items.length) {
          Row({ space: DesignConstants.SPACING.MD }) {
            Row() {
              LucideIcon({
                icon: AppIcon.SUCCESS,
                iconSize: DesignConstants.COMPONENT.ICON_SIZE_SM,
                color: this.getPalette().textStatusOnlineOnSoft
              })
            }
            .width(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL)
            .height(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL)
            .borderRadius(DesignConstants.RADIUS.FULL)
            .backgroundColor(AppThemeVisuals.withAlpha(this.getPalette().success, 0.14))
            .justifyContent(FlexAlign.Center)
            .alignItems(VerticalAlign.Center)

            Column({ space: DesignConstants.SPACING.HALF }) {
              Text('资料已完善')
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
                .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                .fontColor(this.getPalette().textPrimary)
              Text('主页会更容易被附近的人记住，随时可以在编辑页更新。')
                .fontSize(DesignConstants.FONT.SIZE.XS)
                .lineHeight(lineHeightFor(DesignConstants.FONT.SIZE.XS, 1.625))
                .fontColor(this.getPalette().textSecondary)
            }
            .alignItems(HorizontalAlign.Start)
            .layoutWeight(1)
          }
          .width('100%')
          .alignItems(VerticalAlign.Center)
        } else {
          Row({ space: DesignConstants.SPACING.SM }) {
            Text(this.getPercent().toString() + '%')
              .fontSize(DesignConstants.FONT.SIZE.XXL)
              .lineHeight(DesignConstants.FONT.SIZE.XXL)
              .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
              .fontColor(this.getPalette().textPrimary)
            Text('资料完整度')
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
              .fontColor(this.getPalette().textSecondary)
              .margin({ bottom: DesignConstants.SPACING.HALF })
          }
          .width('100%')
          .alignItems(VerticalAlign.Bottom)

          // 分段点进度条（每项一段，已填 accent 实心 / 未填 surfaceSubtle）
          Row({ space: DesignConstants.SPACING.SM2 }) {
            ForEach(this.items, (item: ProfileCompletenessItem) => {
              Column()
                .layoutWeight(1)
                .height(DesignConstants.SPACING.SM2)
                .borderRadius(DesignConstants.RADIUS.FULL)
                .backgroundColor(item.done ? this.getPalette().accentBackground : this.getPalette().surfaceSubtle)
            }, (item: ProfileCompletenessItem) => item.id)
          }
          .width('100%')
          .margin({ top: DesignConstants.SPACING.MD })

          Text('再补几项就完整了——完善后的主页更容易被附近的人记住。')
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .lineHeight(lineHeightFor(DesignConstants.FONT.SIZE.XS, 1.625))
            .fontColor(this.getPalette().textSecondary)
            .width('100%')
            .margin({ top: DesignConstants.SPACING.MD })

          Column() {
            Text('建议补全')
              .fontSize(DesignConstants.FONT.SIZE.XS2)
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS2))
              .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
              .letterSpacing(DesignConstants.FONT.LETTER_SPACING.EYEBROW)
              .fontColor(this.getPalette().textSecondary)
              .width('100%')
            this.missingList()
          }
          .width('100%')
          .alignItems(HorizontalAlign.Start)
          .padding({ top: DesignConstants.SPACING.MD })
          .margin({ top: DesignConstants.SPACING.MD })
          .border({
            width: { top: DesignConstants.BORDER_WIDTH.HAIRLINE },
            color: { top: this.getPalette().borderColor }
          })
        }
      }
      .width('100%')
      .alignItems(HorizontalAlign.Start)
      .padding(DesignConstants.SPACING.LG)
      .borderRadius(DesignConstants.SHAPE.CARD)
      .backgroundColor(this.getPalette().surfaceIsland)
      .shadow(this.getPalette().surfaceShadow)
    }
    .width('100%')
    .alignItems(HorizontalAlign.Start)
  }
}
