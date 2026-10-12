import { MomentItem } from '../../../core/types/api';
import { interactionActions } from '../../../core/state/InteractionActions';
import {
  interactionEventHub,
  MomentSocialState,
  isMomentLikedByReactions,
  sumReactionCounts
} from '../../../core/state/InteractionStateStore';
import { AppRouter, MomentDetailRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { NearbyImageCard } from '../../../components/NearbyImageCard';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { curves } from '@kit.ArkUI';
import { LucideIcon } from '../../../components/LucideIcon';

/**
 * 动态瀑布墙卡 — 4:3 图片卡（对齐前端 nearby/moments/shared/MomentCard）。
 * 左上 tag 胶囊（accent 实底）；底部托盘 = 作者行（22vp 头像 + 昵称 + 位置）、
 * 单行正文、操作行（点赞 count-pop 弹跳 / 评论数 / 收藏）。
 * 点赞/收藏走 InteractionActions 乐观更新；操作行为行内控件，不带动整卡按压。
 */
@Reusable
@Component
export struct MomentWallCard {
  @Prop moment: MomentItem;
  onOpenDetail?: (moment: MomentItem) => void;
  onImagePreview?: (momentId: string, images: string[], index: number) => void;
  // 同 MomentFeedCard：仅持有本卡社交态，事件总线按 moment_id 定向更新
  //（整 Map 订阅会让任意卡片点赞引发全部墙卡重算）。
  @State private socialState: MomentSocialState | undefined = undefined;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @State private likePopped: boolean = false;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getMomentState(): MomentSocialState | undefined {
    return this.socialState;
  }

  aboutToAppear(): void {
    this.socialState = interactionEventHub.getMomentState(this.moment.moment_id);
    interactionEventHub.on('moment_updated', this.handleMomentStateUpdate);
  }

  aboutToDisappear(): void {
    interactionEventHub.off('moment_updated', this.handleMomentStateUpdate);
  }

  private handleMomentStateUpdate(payload: Object): void {
    const state = payload as MomentSocialState;
    if (state && state.moment_id === this.moment.moment_id) {
      this.socialState = state;
    }
  }

  private isLikedByMe(): boolean {
    const state = this.getMomentState();
    return isMomentLikedByReactions(state?.reactions ?? this.moment.reactions);
  }

  private isFavorited(): boolean {
    const state = this.getMomentState();
    return state?.is_favorited ?? this.moment.is_favorited ?? false;
  }

  private getLikeCount(): number {
    const state = this.getMomentState();
    return sumReactionCounts(state?.reactions ?? this.moment.reactions);
  }

  private getCommentCount(): number {
    const state = this.getMomentState();
    return state?.comments_count ?? this.moment.comments_count ?? 0;
  }

  private getPrimaryTag(): string {
    const tags = this.moment.tags ?? [];
    return tags.length > 0 ? tags[0] : '';
  }

  private getCoverUrl(): string {
    const images = this.moment.images ?? [];
    return images.length > 0 ? (images[0].url ?? '') : '';
  }

  private openDetail(): void {
    if (this.onOpenDetail) {
      this.onOpenDetail(this.moment);
      return;
    }
    const params = new MomentDetailRouteParam();
    params.momentId = this.moment.moment_id;
    AppRouter.push(PageNames.MOMENT_DETAIL, params);
  }

  private onLikeClick(): void {
    // count-pop：点赞瞬间计数放大弹跳后回落（对齐前端 cst-count-pop 1→1.3→1；
    // 原实现翻转布尔永不回落，计数停留在缩小态）
    this.likePopped = true;
    setTimeout(() => {
      this.likePopped = false;
    }, DesignConstants.ANIMATION.DURATION_FAST);
    interactionActions.toggleMomentLike(this.moment.moment_id);
  }

  @Builder
  private buildAuthorRow() {
    Row({ space: DesignConstants.SPACING.SM }) {
      SystemAvatar({
        avatarUrl: this.moment.user?.avatar_url ?? '',
        name: this.moment.user?.nickname ?? '',
        avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.XXS,
        shape: 'circle'
      })

      Text(this.moment.user?.nickname ?? '')
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
        .fontColor(this.getPalette().textPrimary)
        .maxLines(1)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
        .layoutWeight(1)

      if (this.moment.location?.location_name) {
        Row({ space: DesignConstants.SPACING.HALF }) {
          LucideIcon({
              icon: AppIcon.LOCATION,
              iconSize: DesignConstants.FONT.SIZE.XS2,
              color: this.getPalette().textSecondary
            })
          Text(this.moment.location.location_name)
            .fontSize(DesignConstants.FONT.SIZE.XS2)
            .fontColor(this.getPalette().textSecondary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .alignItems(VerticalAlign.Center)
        .constraintSize({ maxWidth: '50%' })
      }
    }
    .width('100%')
    .alignItems(VerticalAlign.Center)
  }

  @Builder
  private buildActions() {
    Row() {
      Row({ space: DesignConstants.SPACING.SM2 }) {
        // 点赞（liked = accent + 实心 + count-pop 弹跳）
        Row({ space: DesignConstants.SPACING.HALF }) {
          LucideIcon({
            icon: this.isLikedByMe() ? AppIcon.HEART_FILL : AppIcon.HEART,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
            color: this.isLikedByMe() ? this.getPalette().accentBackground : this.getPalette().textSecondary
          })
          Text(`${this.getLikeCount()}`)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.isLikedByMe() ? this.getPalette().accentBackground : this.getPalette().textSecondary)
            .scale(this.likePopped ?
              { x: DesignConstants.INTERACTION.SCALE_LIKE_POP, y: DesignConstants.INTERACTION.SCALE_LIKE_POP } :
              { x: 1, y: 1 })
            .animation({
              duration: DesignConstants.ANIMATION.DURATION_FAST,
              curve: DesignConstants.ANIMATION.CURVE_EASE_OUT
            })
        }
        .alignItems(VerticalAlign.Center)
        .id(`${NearbyImageCard.CHILD_CONTROL_PREFIX}like-${this.moment.moment_id}`)
        .onClick(() => this.onLikeClick())
        .accessibilityText(this.isLikedByMe() ? '取消点赞' : '点赞')

        // 评论数（点击进详情评论区）
        Row({ space: DesignConstants.SPACING.HALF }) {
          LucideIcon({
              icon: AppIcon.MESSAGE,
              iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
              color: this.getPalette().textSecondary
            })
          Text(`${this.getCommentCount()}`)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getPalette().textSecondary)
        }
        .alignItems(VerticalAlign.Center)
        .id(`${NearbyImageCard.CHILD_CONTROL_PREFIX}comment-${this.moment.moment_id}`)
        .onClick(() => this.openDetail())
        .accessibilityText('查看评论')
        .responseRegion({
          x: -DesignConstants.SPACING.SM,
          y: -DesignConstants.SPACING.SM,
          width: '120%',
          height: DesignConstants.LAYOUT.MIN_HIT_TARGET
        })
      }
      .alignItems(VerticalAlign.Center)

      Blank()

      // 收藏
      LucideIcon({
        icon: this.isFavorited() ? AppIcon.STAR_FILL : AppIcon.STAR,
        iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
        color: this.isFavorited() ? this.getPalette().accentBackground : this.getPalette().textSecondary
      })
      .id(`${NearbyImageCard.CHILD_CONTROL_PREFIX}favorite-${this.moment.moment_id}`)
      .onClick(() => {
        interactionActions.toggleMomentFavorite(this.moment.moment_id);
      })
      .accessibilityText(this.isFavorited() ? '取消收藏' : '收藏')
      .responseRegion({
        x: -DesignConstants.SPACING.MD2,
        y: -DesignConstants.SPACING.MD2,
        width: DesignConstants.LAYOUT.MIN_HIT_TARGET,
        height: DesignConstants.LAYOUT.MIN_HIT_TARGET
      })
    }
    .width('100%')
    .alignItems(VerticalAlign.Center)
  }

  build() {
    NearbyImageCard({
      imageUrl: this.getCoverUrl(),
      imageAlt: this.getPrimaryTag() || this.moment.description || this.moment.user?.nickname || '',
      cardAspectRatio: 1.333,
      placeholderText: this.moment.user?.nickname ?? '',
      badgeText: this.getPrimaryTag(),
      badgeShape: 'pill',
      badgeVariant: 'accent',
      badgeAlign: 'topLeft',
      onPress: () => this.openDetail()
    }) {
      Column({ space: DesignConstants.SPACING.THREE }) {
        this.buildAuthorRow()

        if (this.moment.description) {
          Text(this.moment.description)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getPalette().textSecondary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }

        this.buildActions()
      }
      .width('100%')
      .alignItems(HorizontalAlign.Start)
      .padding({
        left: DesignConstants.SPACING.MD,
        right: DesignConstants.SPACING.MD,
        top: DesignConstants.SPACING.MD2,
        bottom: DesignConstants.SPACING.MD2
      })
    }
  }
}
