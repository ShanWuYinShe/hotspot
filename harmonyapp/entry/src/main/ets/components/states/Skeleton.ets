/**
 * Skeleton — 骨架屏占位块
 *
 * 选型规则：列表型首屏等待用骨架屏（本组件，可配 SkeletonShimmer 动效），
 * 操作型等待（提交/动作反馈）用 SystemLoadingView。
 *
 * 用法：
 *   Skeleton({ blockWidth: '100%', blockHeight: 16, cornerRadius: 4 })
 *   Skeleton({ blockWidth: 200, blockHeight: 200, cornerRadius: '50%' })
 *
 * 视觉：使用 bg_tertiary（中性灰）作为占位色，
 * 配合 SkeletonShimmer 父容器使用可获得 shimmer 动效
 *
 * 与 frontend /frontend/src/lib/components/ui/skeleton.tsx 对齐
 *
 * 注：属性名不用 width/height/borderRadius/opacity——与 CustomComponent 基类
 * 方法同名导致 ArkTS 编译失败（此前零引用即因此从未编译通过）。
 */

import { DesignConstants } from '../../constants/DesignConstants';
import { AppThemeManager } from '../../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../core/theme/AppThemeTypes';

@Component
export struct Skeleton {
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  @Prop blockWidth: Length = '100%';
  @Prop blockHeight: Length = 16;
  @Prop cornerRadius: Length | string = DesignConstants.RADIUS.SM;
  @Prop block: boolean = false;

  build() {
    Column() {}
      .width(this.blockWidth)
      .height(this.block ? this.blockWidth : this.blockHeight)
      .borderRadius(this.cornerRadius)
      .backgroundColor(AppThemeManager.getPaletteById(this.resolvedThemeId).surfaceSubtle)
  }
}
