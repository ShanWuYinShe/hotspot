/**
 * 通用空状态组件
 * 用于列表无数据、加载错误等场景的统一展示
 */

import { DesignConstants } from '../constants/DesignConstants';
import { AppIcon } from './SystemIcon';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';

@Reusable
@Component
export struct EmptyState {
  @Prop icon: AppIcon = AppIcon.EMPTY;
  @Prop message: string = '';
  @Prop actionLabel: string = '';
  onAction?: () => void;

  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Column() {
      Column({ space: DesignConstants.SPACING.LG }) {
        Stack() {
          Column()
            .width(64)
            .height(64)
            .borderRadius(DesignConstants.RADIUS.FULL)
            .backgroundColor(this.getPalette().surfaceBackground)
            .border({ width: 1, color: this.getPalette().floatingBorderColor })

          Text() {
            SymbolSpan($r(this.icon))
              .fontSize(DesignConstants.FONT.SIZE.XXL_PLUS)
              .fontColor([this.getPalette().textSecondary])
              .opacity(0.5)
          }
        }
        .width(64)
        .height(64)

        Text(this.message ?? '')
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .fontColor(this.getPalette().textSecondary)
          .textAlign(TextAlign.Center)
          .lineHeight(24)

        if ((this.actionLabel ?? '').length > 0 && this.onAction) {
          Button(this.actionLabel ?? '')
            .type(ButtonType.Normal)
            .stateEffect(true)
            .backgroundColor(this.getPalette().accentBackground)
            .fontColor(this.getPalette().accentText as ResourceColor)
            .height(DesignConstants.COMPONENT.BUTTON_HEIGHT)
            .borderRadius(DesignConstants.SHAPE.BUTTON)
            .border({ width: 1, color: this.getPalette().primaryButtonBorderColor })
            .shadow(this.getPalette().accentShadow)
            .padding({ left: DesignConstants.SPACING.XL, right: DesignConstants.SPACING.XL })
            .onClick(() => {
              this.onAction?.();
            })
        }
      }
      .width('100%')
      .padding(DesignConstants.SPACING.XXL)
      .borderRadius(DesignConstants.SHAPE.HERO_CARD)
      .backgroundColor(this.getPalette().surfaceBackground)
      .border({ width: 1, color: this.getPalette().floatingBorderColor })
      .shadow(this.getPalette().elevatedShadow)
    }
    .width('100%')
    .height('100%')
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

  build() {
    Row({ space: DesignConstants.SPACING.SM }) {
      LoadingProgress()
        .width(20)
        .height(20)
        .color(AppThemeManager.getSafeCurrentPalette().accentBackground)
        .accessibilityText('正在加载')
      Text(this.message)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor(AppThemeManager.getSafeCurrentPalette().textSecondary)
    }
    .width('100%')
    .justifyContent(FlexAlign.Center)
    .padding(DesignConstants.SPACING.MD)
    .borderRadius(DesignConstants.SHAPE.CHIP)
    .backgroundColor(AppThemeManager.getSafeCurrentPalette().surfaceBackground)
    .border({ width: 1, color: AppThemeManager.getSafeCurrentPalette().floatingBorderColor })
    .shadow(AppThemeManager.getSafeCurrentPalette().surfaceShadow)
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
      .width(24)
      .height(24)
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
