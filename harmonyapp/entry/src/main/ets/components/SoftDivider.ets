/**
 * 软渐变分割线，对齐前端 .divider-soft（frontend/src/app.css）：
 * linear-gradient(to right, transparent 0% → borderColor 20% → borderColor 80% → transparent 100%)，
 * 默认高 1px。两端淡出、中间实色，比纯色 Divider 更柔和。
 *
 * 用法：SoftDivider({ color: this.getSurfaceBorder() })
 */

import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';

@Component
export struct SoftDivider {
  /** 默认取当前主题 borderColor；调用方可按上下文显式传色 */
  @Prop color: ResourceColor = AppThemeManager.getSafeCurrentPalette().borderColor;
  @Prop thickness: number = 1;

  build() {
    Row()
      .width('100%')
      .height(this.thickness)
      .linearGradient({
        direction: GradientDirection.Right,
        colors: [
          [DesignConstants.COLOR.TRANSPARENT, 0.0],
          [this.color, 0.2],
          [this.color, 0.8],
          [DesignConstants.COLOR.TRANSPARENT, 1.0]
        ]
      })
  }
}
