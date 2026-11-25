/**
 * CardGridSkeleton —— 名片卡墙首屏骨架（2/3 列白底卡，居中对齐头像+两行文字+底部 pill）
 *
 * 用于「附近的人 / 附近的群组」名片卡墙的网络等待：行式 ListSkeleton 与卡墙几何不匹配，
 * 单独用本组件保持「加载前骨架 == 加载后卡片」同构，避免二次位移；SkeletonShimmer 统一脉冲，
 * reduced-motion 下静止。
 *
 * 用法：
 *   CardGridSkeleton({ columns: this.getColumnCount(), rows: 3 })
 */
import { DesignConstants } from '../../constants/DesignConstants';
import { AppThemeManager } from '../../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../core/theme/AppThemeTypes';
import { Skeleton } from './Skeleton';
import { SkeletonShimmer } from './SkeletonShimmer';

@Component
export struct CardGridSkeleton {
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  @Prop columns: number = 2;
  @Prop rows: number = 3;
  @Prop cardHeight: number = 188;

  private indexes(count: number): number[] {
    const list: number[] = [];
    for (let i = 0; i < count; i++) {
      list.push(i);
    }
    return list;
  }

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Column() {
      SkeletonShimmer({ active: true }) {
        Column({ space: DesignConstants.SPACING.MD }) {
          ForEach(this.indexes(this.rows), (rowIndex: number) => {
            Row({ space: DesignConstants.SPACING.MD }) {
              ForEach(this.indexes(this.columns), (colIndex: number) => {
                Column({ space: DesignConstants.SPACING.SM }) {
                  Skeleton({
                    blockWidth: DesignConstants.COMPONENT.AVATAR_SIZE.MEDIUM,
                    blockHeight: DesignConstants.COMPONENT.AVATAR_SIZE.MEDIUM,
                    cornerRadius: DesignConstants.RADIUS.FULL
                  })
                  Skeleton({ blockWidth: '52%', blockHeight: DesignConstants.FONT.SIZE.MD, cornerRadius: DesignConstants.RADIUS.SM })
                  Skeleton({ blockWidth: '36%', blockHeight: DesignConstants.FONT.SIZE.XS, cornerRadius: DesignConstants.RADIUS.SM })
                  Skeleton({ blockWidth: '44%', blockHeight: DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT, cornerRadius: DesignConstants.RADIUS.FULL })
                    .margin({ top: DesignConstants.SPACING.XS })
                }
                .layoutWeight(1)
                .height(this.cardHeight)
                .justifyContent(FlexAlign.Center)
                .alignItems(HorizontalAlign.Center)
                .padding(DesignConstants.SPACING.MD)
                .backgroundColor(this.getPalette().surfaceIsland)
                .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().borderColor })
                .borderRadius(DesignConstants.SHAPE.CARD)
              }, (colIndex: number) => 'card-grid-skeleton-' + rowIndex + '-' + colIndex)
            }
            .width('100%')
            .alignItems(VerticalAlign.Top)
          }, (rowIndex: number) => 'card-grid-skeleton-row-' + rowIndex)
        }
        .width('100%')
      }
    }
    .width('100%')
    .layoutWeight(1)
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.LG
    })
  }
}
