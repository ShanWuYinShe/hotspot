/**
 * Skeleton — 骨架屏占位块
 *
 * 用法：
 *   Skeleton({ width: '100%', height: 16, borderRadius: 4 })
 *   Skeleton({ width: 200, height: 200, borderRadius: '50%' })
 *
 * 视觉：使用 bg_tertiary（中性灰）作为占位色，
 * 配合 SkeletonShimmer 父容器使用可获得 shimmer 动效
 *
 * 与 frontend /frontend/src/lib/components/ui/skeleton.tsx 对齐
 */

import { DesignConstants } from '../../constants/DesignConstants';
import { AppThemeManager } from '../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../core/theme/AppThemeTypes';

@Component
export struct Skeleton {
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @Prop width: Length = '100%';
  @Prop height: Length = 16;
  @Prop borderRadius: Length | string = DesignConstants.RADIUS.SM;
  @Prop block: boolean = false;

  build() {
    Column() {}
      .width(this.width)
      .height(this.block ? this.width : this.height)
      .borderRadius(typeof this.borderRadius === 'string' ? 0 : this.borderRadius as Length)
      .backgroundColor(AppThemeManager.getPaletteById(this.resolvedThemeId).surfaceSubtle)
  }
}
