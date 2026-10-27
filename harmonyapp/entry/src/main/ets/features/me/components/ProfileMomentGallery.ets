/**
 * ProfileMomentGallery —— 个人页「我的瞬间」九宫格画廊（帖子 tab 网格 +
 * 喜欢/收藏 tab 空态）。从 ProfilePage.buildFigmaGallery 抽取；
 * 缩略图归一化与按列分行逻辑随组件内聚，列数经 props 注入（随断点）。
 */

import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { MediaUrlUtil } from '../../../core/utils/MediaUrlUtil';
import { ImageUrlUtil } from '../../../core/utils/ImageUrlUtil';
import { MomentItem } from '../../../core/types/api';
import { LucideIcon } from '../../../components/LucideIcon';
import { DEFAULT_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { MotionUtil } from '../../../core/accessibility/MotionUtil';

@Component
export struct ProfileMomentGallery {
  /** 0=帖子 1=喜欢 2=收藏 */
  @Prop selectedTab: number = 0;
  @Prop moments: MomentItem[] = [];
  /** 网格列数（宽屏 master-detail 3 列，窄屏 2 列，对齐前端 grid-cols-2 md:grid-cols-3） */
  @Prop columns: number = 2;
  @Prop resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  @State private loadedThumbs: string[] = [];
  @State private failedThumbs: string[] = [];

  private palette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private momentThumb(moment: MomentItem): string {
    const imgs = moment.images;
    if (imgs !== undefined && imgs !== null && imgs.length > 0) {
      const first = imgs[0];
      // normalize：相对 /s3/ 路径与本地 host 绝对地址统一改写（与其余页面一致，
      // 兜底 RDB 缓存/历史数据未经 HttpUtil 网络层改写的场景）
      return ImageUrlUtil.optimized(MediaUrlUtil.normalize(first.thumbnail_url || first.url), 480);
    }
    return '';
  }

  // 我的瞬间九宫格：按列数分行（避开 ArkUI 不支持的 calc，用 Row + layoutWeight 等宽）。
  private momentRows(): MomentItem[][] {
    const cols: number = this.columns;
    const items: MomentItem[] = this.moments.slice(0, cols * 3); // 最多 3 行
    const rows: MomentItem[][] = [];
    for (let i = 0; i < items.length; i += cols) {
      rows.push(items.slice(i, i + cols));
    }
    return rows;
  }

  @Builder
  private momentTile(moment: MomentItem) {
    Column() {
      Stack() {
        // R8 占位层：图标常驻，加载中/失败不出现纯灰块
        Column() {
          LucideIcon({
            icon: AppIcon.IMAGES,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_MD,
            color: this.palette().textTertiary
          })
        }
        .width('100%')
        .height('100%')
        .justifyContent(FlexAlign.Center)
        .backgroundColor(this.palette().surfaceSubtle)
        .borderRadius(DesignConstants.RADIUS.LG)
        if (this.failedThumbs.indexOf(this.momentThumb(moment)) < 0) {
          Image(this.momentThumb(moment))
            .width('100%')
            .height('100%')
            .objectFit(ImageFit.Cover)
            .borderRadius(DesignConstants.RADIUS.LG)
            .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.palette().borderColor })
            .opacity(this.loadedThumbs.indexOf(this.momentThumb(moment)) >= 0 || MotionUtil.reduced() ? 1 : 0)
            .animation({
              duration: MotionUtil.reduced() ? 0 : DesignConstants.ANIMATION.DURATION_SHORT,
              curve: DesignConstants.ANIMATION.CURVE_EASE_OUT
            })
            .onComplete(() => {
              const key: string = this.momentThumb(moment);
              if (this.loadedThumbs.indexOf(key) >= 0) return;
              const next: string[] = this.loadedThumbs.slice();
              next.push(key);
              this.loadedThumbs = next;
            })
            .onError(() => {
              const key: string = this.momentThumb(moment);
              if (this.failedThumbs.indexOf(key) >= 0) return;
              const next: string[] = this.failedThumbs.slice();
              next.push(key);
              this.failedThumbs = next;
            })
        }
      }
      .width('100%')
      .aspectRatio(1)
    }
    .layoutWeight(1)
  }

  @Builder
  private momentsRow(row: MomentItem[], rowIndex: number) {
    Row({ space: DesignConstants.SPACING.SM }) {
      ForEach(row, (m: MomentItem) => {
        this.momentTile(m)
      }, (m: MomentItem) => m.moment_id)
      ForEach(Array.from({ length: this.columns - row.length }, (_: Object, padIndex: number) => padIndex), (_: number) => {
        Column().layoutWeight(1)
      }, (_: number, padIndex: number) => `pad_${rowIndex}_${padIndex}`)
    }
    .width('100%')
  }

  build() {
    if (this.selectedTab === 0) {
      // 帖子 tab
      if (this.moments.length === 0) {
        Column({ space: DesignConstants.SPACING.MD }) {
          LucideIcon({
              icon: AppIcon.EMPTY,
              iconSize: DesignConstants.COMPONENT.ICON_SIZE_SM,
              color: this.palette().textSecondary
            })
          .opacity(DesignConstants.OPACITY.MUTED)

          Text('还没有发布帖子')
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor(this.palette().textSecondary)
        }
        .width('100%')
        .height(DesignConstants.COMPONENT.EMPTY_ICON_SIZE) // 64 对齐前端空态容器档
        .justifyContent(FlexAlign.Center)
      } else {
        Column({ space: DesignConstants.SPACING.SM }) {
          ForEach(this.momentRows(), (row: MomentItem[], rowIndex: number) => {
            this.momentsRow(row, rowIndex)
          }, (row: MomentItem[], rowIndex: number) => `${rowIndex}`)
        }
        .width('100%')
      }
    } else {
      // 喜欢 / 收藏 tab
      Column({ space: DesignConstants.SPACING.MD }) {
        LucideIcon({
            icon: this.selectedTab === 1 ? AppIcon.HEART : AppIcon.BOOKMARK,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_SM,
            color: this.palette().textSecondary
          })
        .opacity(DesignConstants.OPACITY.MUTED)

        Text(this.selectedTab === 1 ? '喜欢的帖子将在这里展示' : '收藏的帖子将在这里展示')
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor(this.palette().textSecondary)
      }
      .width('100%')
      .height(DesignConstants.COMPONENT.EMPTY_ICON_SIZE) // 64 对齐前端空态容器档
      .justifyContent(FlexAlign.Center)
    }
  }
}
