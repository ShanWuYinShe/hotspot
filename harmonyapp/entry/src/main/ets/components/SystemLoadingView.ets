/**
 * 系统级加载组件
 * 使用 HarmonyOS 原生 LoadingProgress，保持系统一致性与简洁视觉。
 */

import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';

@Component
export struct SystemLoadingView {
  @Prop message: string = '加载中...';
  @Prop showProgress: boolean = true;
  @Prop showSystemInfo: boolean = false;
  @Prop progressType: 'indeterminate' | 'determinate' = 'indeterminate';
  @Prop maxProgress: number = 100;
  @Prop externalProgress: number = 0;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Column() {
      Column({ space: DesignConstants.SPACING.MD }) {
        if (this.showProgress) {
          Stack() {
            Column()
              .width(DesignConstants.COMPONENT.LOADING_CONTAINER_SIZE)
              .height(DesignConstants.COMPONENT.LOADING_CONTAINER_SIZE)
              .borderRadius(DesignConstants.RADIUS.FULL)
              .backgroundColor(this.getPalette().surfaceBackground)
              .shadow(this.getPalette().controlShadow)
              .border({ width: 1, color: this.getPalette().floatingBorderColor })

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
      .padding(DesignConstants.SPACING.XXL)
      .borderRadius(DesignConstants.SHAPE.HERO_CARD)
      .backgroundColor(this.getPalette().surfaceBackground)
      .border({ width: 1, color: this.getPalette().floatingBorderColor })
      .shadow(this.getPalette().elevatedShadow)
    }
    .width('100%')
    .height('100%')
    .justifyContent(FlexAlign.Center)
    .backgroundColor(this.getPalette().pageBackground)
    .padding(DesignConstants.SPACING.XXL)
    .accessibilityText(this.message)
  }
}
