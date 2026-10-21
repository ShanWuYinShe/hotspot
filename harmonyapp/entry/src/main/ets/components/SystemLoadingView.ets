/**
 * 系统级加载组件
 * 使用 HarmonyOS 原生 LoadingProgress，保持系统一致性与简洁视觉。
 */

import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';

@Component
export struct SystemLoadingView {
  @Prop message: string = '加载中…';
  @Prop showProgress: boolean = true;
  @Prop progressType: 'indeterminate' | 'determinate' = 'indeterminate';
  @Prop maxProgress: number = 100;
  @Prop externalProgress: number = 0;
  /**
   * card：是否画「圆角卡片 + 图标徽章」外框。默认 **false（无卡）** —— 页面级加载态应与页面
   * 背景一体，而不是在屏幕中间悬浮一张卡片（2026-10-21 端上反馈：加载框很丑、和 UI 不是一体）。
   * 只有浮层/媒体之上的局促场景才需要传 true。
   */
  @Prop card: boolean = false;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Column() {
      Column({ space: DesignConstants.SPACING.MD }) {
        if (this.showProgress) {
          Stack() {
            // 无卡形态不画圆形徽章：加载图标直接落在页面上（与无卡错误态一致）
            Column()
              .width(DesignConstants.COMPONENT.LOADING_CONTAINER_SIZE)
              .height(DesignConstants.COMPONENT.LOADING_CONTAINER_SIZE)
              .borderRadius(DesignConstants.RADIUS.FULL)
              .backgroundColor(this.card ? this.getPalette().surfaceBackground : DesignConstants.COLOR.TRANSPARENT)
              .shadow(this.card ? this.getPalette().controlShadow : {
                radius: 0,
                color: DesignConstants.COLOR.TRANSPARENT,
                offsetX: 0,
                offsetY: 0
              })
              .border(this.card
                ? { width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().floatingBorderColor }
                : { width: 0 })

            if (this.progressType === 'determinate') {
              Progress({
                value: this.externalProgress,
                total: this.maxProgress,
                type: ProgressType.Ring
              })
                .width(DesignConstants.COMPONENT.LOADING_GLYPH_SIZE)
                .height(DesignConstants.COMPONENT.LOADING_GLYPH_SIZE)
                .color(this.getPalette().accentBackground)
                .style({ strokeWidth: DesignConstants.COMPONENT.PROGRESS_STROKE_WIDTH })
            } else {
              LoadingProgress()
                .width(DesignConstants.COMPONENT.LOADING_GLYPH_SIZE)
                .height(DesignConstants.COMPONENT.LOADING_GLYPH_SIZE)
                .color(this.getPalette().accentBackground)
                .accessibilityText('正在加载')
            }
          }
          .width(DesignConstants.COMPONENT.LOADING_CONTAINER_SIZE)
          .height(DesignConstants.COMPONENT.LOADING_CONTAINER_SIZE)
        }

        Text(this.message)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor(this.getPalette().textSecondary)
          .textAlign(TextAlign.Center)

        if (this.showProgress && this.progressType === 'determinate') {
          Text(`${Math.round(this.externalProgress)}%`)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getPalette().textTertiary)
            .textAlign(TextAlign.Center)
        }
      }
      .padding(this.card ? DesignConstants.SPACING.XXL : 0)
      .borderRadius(this.card ? DesignConstants.SHAPE.HERO_CARD : 0)
      .backgroundColor(this.card ? this.getPalette().surfaceBackground : DesignConstants.COLOR.TRANSPARENT)
      .border(this.card
        ? { width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().floatingBorderColor }
        : { width: 0 })
      .shadow(this.card ? this.getPalette().surfaceShadow : {
        radius: 0,
        color: DesignConstants.COLOR.TRANSPARENT,
        offsetX: 0,
        offsetY: 0
      })
    }
    .width('100%')
    .height('100%')
    .justifyContent(FlexAlign.Center)
    .backgroundColor(this.card ? this.getPalette().pageBackground : DesignConstants.COLOR.TRANSPARENT)
    .padding(DesignConstants.SPACING.XXL)
    .accessibilityText(this.message)
  }
}
