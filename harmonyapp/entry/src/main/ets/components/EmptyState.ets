/**
 * 通用空状态组件
 * 用于列表无数据、加载错误等场景的统一展示
 */

import { DesignConstants } from '../constants/DesignConstants';
import { AppIcon } from './SystemIcon';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { SystemButton } from './SystemButton';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { LucideIcon } from './LucideIcon';

@Reusable
@Component
export struct EmptyState {
  @Prop icon: AppIcon = AppIcon.EMPTY;
  /** 主标题（大字，对齐前端 EmptyState title） */
  @Prop title: string = '';
  /** 说明文字（小字，向后兼容：等价于前端 description） */
  @Prop message: string = '';
  @Prop actionLabel: string = '';
  /** 整页空态：撑开 min-height 垂直居中（对齐前端 fill 模式，用于下方无并列内容的空态） */
  @Prop fillMode: boolean = false;
  /** 极简空态：不画卡片背景/边框/阴影（对齐前端 bare 形态，用于 tab 内容区/列表内空态） */
  @Prop bare: boolean = false;
  onAction?: () => void;

  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Column() {
      Column({ space: DesignConstants.SPACING.LG }) {
        // bare：无圆形容器，图标直接渲染 + muted 文案（对齐前端 EmptyState bare 契约）
        if (this.bare) {
          LucideIcon({
            icon: this.icon,
            iconSize: DesignConstants.FONT.SIZE.XXL,
            color: this.getPalette().textSecondary
          })
          .opacity(DesignConstants.OPACITY.MUTED)
        } else {
          Stack() {
            // 图标底改 accent 淡底、glyph 用主题色（对齐前端空态的柔和 accent 图标，
            // 消除此前灰底灰字 + muted 叠加导致的「禁止符号」观感）
            Column()
              .width(DesignConstants.COMPONENT.EMPTY_ICON_SIZE)
              .height(DesignConstants.COMPONENT.EMPTY_ICON_SIZE)
              .borderRadius(DesignConstants.RADIUS.FULL)
              .backgroundColor(this.getPalette().accentMuted)
              .shadow(this.getPalette().accentShadow)

            LucideIcon({
              icon: this.icon,
              iconSize: DesignConstants.FONT.SIZE.XXL,
              color: this.getPalette().accentBackground
            })
            .opacity(DesignConstants.OPACITY.EMPHASIZED)
          }
          .width(DesignConstants.COMPONENT.EMPTY_ICON_SIZE)
          .height(DesignConstants.COMPONENT.EMPTY_ICON_SIZE)
        }

        // 主标题（大字）—— 对齐前端 EmptyState title；
        // bare 变体 title 用 textSecondary（对齐前端 bare 契约 muted 文案）
        if ((this.title ?? '').length > 0) {
          Text(this.title)
            .fontSize(DesignConstants.FONT.SIZE.LG_PLUS) // 18
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.bare ? this.getPalette().textSecondary : this.getPalette().textPrimary)
            .textAlign(TextAlign.Center)
        }

        // 说明文字（小字）
        if ((this.message ?? '').length > 0) {
          Text(this.message)
            .fontSize(DesignConstants.FONT.SIZE.SM) // 12
            .fontColor(this.getPalette().textSecondary)
            .textAlign(TextAlign.Center)
            .lineHeight(DesignConstants.FONT.LINE_HEIGHT.SM)
        }

        if ((this.actionLabel ?? '').length > 0 && this.onAction) {
          SystemButton({
            label: this.actionLabel ?? '',
            variant: 'primary',
            buttonSize: 'default',
            onPress: () => {
              this.onAction?.();
            }
          })
        }
      }
      .width('100%')
      .padding(this.bare ? 0 : DesignConstants.SPACING.XXL)
      .borderRadius(this.bare ? 0 : DesignConstants.SHAPE.HERO_CARD)
      .backgroundColor(this.bare ? DesignConstants.COLOR.TRANSPARENT : this.getPalette().surfaceBackground)
      .border(this.bare
        ? { width: 0 }
        : { width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().floatingBorderColor })
      .shadow(this.bare
        ? { radius: 0, color: DesignConstants.COLOR.TRANSPARENT, offsetX: 0, offsetY: 0 }
        : this.getPalette().elevatedShadow)
    }
    .width('100%')
    .height('100%')
    .constraintSize({ minHeight: this.fillMode ? 420 : 0 })
    .justifyContent(FlexAlign.Center)
    .padding(DesignConstants.SPACING.XL)
  }
}


/**
 * 加载更多指示器
 * 用于列表底部的加载更多状态展示
 */
@Reusable
@Component
export struct LoadingMoreIndicator {
  @Prop message: string = '加载更多...';
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Row({ space: DesignConstants.SPACING.SM }) {
      LoadingProgress()
        .width(DesignConstants.COMPONENT.LOADING_ICON_SIZE)
        .height(DesignConstants.COMPONENT.LOADING_ICON_SIZE)
        .color(this.getPalette().accentBackground)
        .accessibilityText('正在加载')
      Text(this.message)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor(this.getPalette().textSecondary)
    }
    .width('100%')
    .justifyContent(FlexAlign.Center)
    .padding(DesignConstants.SPACING.MD)
    .borderRadius(DesignConstants.SHAPE.CHIP)
    .backgroundColor(this.getPalette().surfaceBackground)
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().floatingBorderColor })
    .shadow(this.getPalette().surfaceShadow)
    .accessibilityText(this.message)
  }
}

/**
 * 下拉刷新指示器
 * 用于 Refresh 组件的 builder 参数
 */
@Builder
export function RefreshIndicator() {
  Row() {
    LoadingProgress()
      .width(DesignConstants.COMPONENT.ICON_SIZE_MD)
      .height(DesignConstants.COMPONENT.ICON_SIZE_MD)
      .color(AppThemeManager.getSafeCurrentPalette().accentBackground)
      .accessibilityText('正在刷新')
    Text('正在刷新...')
      .fontSize(DesignConstants.FONT.SIZE.MD)
      .margin({ left: DesignConstants.SPACING.SM })
      .fontColor(AppThemeManager.getSafeCurrentPalette().textSecondary)
  }
  .width('100%')
  .justifyContent(FlexAlign.Center)
  .padding(DesignConstants.SPACING.SM)
  .accessibilityText('正在刷新')
}
