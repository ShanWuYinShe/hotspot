/**
 * ListSkeleton —— 通用列表首屏骨架（列表型等待用；操作型等待仍用 SystemLoadingView）
 *
 * 5 行「圆头像 + 两行文字条」+ SkeletonShimmer，几何与分享/举报/公告等列表行卡一致；
 * 各列表页接入时只需一行 ListSkeleton({ rowCount: 5 })，避免每页重复写骨架结构。
 */
import { DesignConstants } from '../../constants/DesignConstants';
import { Skeleton } from './Skeleton';
import { SkeletonShimmer } from './SkeletonShimmer';

@Component
export struct ListSkeleton {
  @Prop rowCount: number = 5;
  @Prop avatarSize: number = DesignConstants.COMPONENT.AVATAR_SIZE.SMALL;

  private rowIndexes(): number[] {
    const indexes: number[] = [];
    for (let i = 0; i < this.rowCount; i++) {
      indexes.push(i);
    }
    return indexes;
  }

  build() {
    Column() {
      SkeletonShimmer({ active: true }) {
        Column({ space: DesignConstants.SPACING.SM }) {
          ForEach(this.rowIndexes(), (index: number) => {
            Row({ space: DesignConstants.SPACING.MD }) {
              Skeleton({
                blockWidth: this.avatarSize,
                blockHeight: this.avatarSize,
                cornerRadius: '50%'
              })
              Column({ space: DesignConstants.SPACING.XS }) {
                Skeleton({
                  blockWidth: '55%',
                  blockHeight: DesignConstants.FONT.SIZE.MD,
                  cornerRadius: DesignConstants.RADIUS.SM
                })
                Skeleton({
                  blockWidth: '34%',
                  blockHeight: DesignConstants.FONT.SIZE.XS,
                  cornerRadius: DesignConstants.RADIUS.SM
                })
              }
              .layoutWeight(1)
              .alignItems(HorizontalAlign.Start)
            }
            .width('100%')
            .padding({
              left: DesignConstants.SPACING.MD,
              right: DesignConstants.SPACING.MD,
              top: DesignConstants.SPACING.MD,
              bottom: DesignConstants.SPACING.MD
            })
          }, (index: number) => 'list-skeleton-' + index)
        }
        .width('100%')
      }
    }
    .width('100%')
    .layoutWeight(1)
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.SM })
  }
}
