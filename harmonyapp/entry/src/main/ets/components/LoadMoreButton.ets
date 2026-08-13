import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { AppIcon } from './SystemIcon';

/**
 * 加载更多指示器（对齐前端 load-more-button.svelte）
 * - loading=true：旋转图标 + "加载中"
 * - loading=false 且 hasMore=true：向下箭头 + "点击加载更多"（可点击）
 * - hasMore=false：不渲染
 */
@Reusable
@Component
export struct LoadMoreButton {
  @Prop loading: boolean = false;
  @Prop hasMore: boolean = false;
  @Prop loadingText: string = '加载中';
  @Prop actionText: string = '点击加载更多';
  onLoadMore?: () => void;

  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    if (this.hasMore || this.loading) {
      Row() {
        if (this.loading) {
          LoadingProgress()
            .width(DesignConstants.COMPONENT.LOADING_ICON_SIZE)
            .height(DesignConstants.COMPONENT.LOADING_ICON_SIZE)
            .color(this.getPalette().textSecondary)
            .margin({ right: DesignConstants.SPACING.XS })
          Text(this.loadingText)
            .fontSize(DesignConstants.FONT.SIZE.SM) // 13
            .fontColor(this.getPalette().textSecondary)
        } else {
          Text() {
            SymbolSpan($r(AppIcon.ARROW_DOWN))
              .fontSize(DesignConstants.FONT.SIZE.XL)
              .fontColor([this.getPalette().textSecondary])
          }
          .margin({ right: DesignConstants.SPACING.XS })
          Text(this.actionText)
            .fontSize(DesignConstants.FONT.SIZE.XS) // 11
            .fontColor(this.getPalette().textSecondary)
        }
      }
      .width('100%')
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)
      .padding({ top: DesignConstants.SPACING.XL, bottom: DesignConstants.SPACING.XL })
      .onClick(() => {
        if (!this.loading && this.hasMore && this.onLoadMore) {
          this.onLoadMore();
        }
      })
    }
  }
}
