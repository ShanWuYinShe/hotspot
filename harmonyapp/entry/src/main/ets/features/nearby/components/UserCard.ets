import { UserItem } from '../../../core/types/api';
import { interactionActions } from '../../../core/state/InteractionActions';
import { UserRelationState } from '../../../core/state/InteractionStateStore';
import { LocationUtil } from '../../../core/location/LocationUtil';
import { AppRouter, ProfileRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { NearbyImageCard } from '../../../components/NearbyImageCard';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { LucideIcon } from '../../../components/LucideIcon';

/**
 * 附近用户卡 — 3:4 图片卡（头像即卡图，对齐前端 nearby/people/UserCard）。
 * 右上距离毛玻璃徽章；底部托盘 = 昵称 + bio 副行 + 在线点、
 * #tag 徽章 + 28vp 圆形关注钮（行内操作，不带动整卡按压）。
 * 关注态优先读 InteractionStateStore 乐观更新，回退响应 relation。
 */
@Reusable
@Component
export struct UserCard {
  @Prop user: UserItem;
  @Prop showDistance: boolean = true;
  @StorageLink('interaction_user_state') private userStateMap: Record<string, UserRelationState> = {};
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @State private followPressed: boolean = false;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getUserId(): string {
    return this.user.user?.user_id ?? '';
  }

  private isFollowing(): boolean {
    const state = this.userStateMap[this.getUserId()];
    return state?.relation?.is_following ?? this.user.relation?.is_following ?? false;
  }

  private isOnline(): boolean {
    return this.user.presence_status === 'online';
  }

  private getDistanceText(): string {
    const distance = this.user.location?.distance;
    if (distance === undefined || distance === null) {
      return '';
    }
    return LocationUtil.formatDistance(distance);
  }

  private getPrimaryTag(): string {
    const tags = this.user.tags ?? [];
    return tags.length > 0 ? tags[0] : '';
  }

  private openProfile(): void {
    const userId = this.getUserId();
    if (userId) {
      const params = new ProfileRouteParam();
      params.userId = userId;
      AppRouter.push(PageNames.PROFILE, params);
    }
  }

  private toggleFollow(): void {
    const userId = this.getUserId();
    if (userId) {
      interactionActions.setFollowUser(userId, !this.isFollowing());
    }
  }

  @Builder
  private buildFollowButton() {
    Row() {
      LucideIcon({
        icon: this.isFollowing() ? AppIcon.CHECK : AppIcon.ADD,
        size: DesignConstants.FONT.SIZE.SM,
        color: this.isFollowing() ? this.getPalette().accentText : DesignConstants.COLOR.WHITE
      })
    }
    .id(`${NearbyImageCard.CHILD_CONTROL_PREFIX}follow-${this.getUserId()}`)
    .width(DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT)
    .height(DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT)
    .borderRadius(DesignConstants.RADIUS.FULL)
    .justifyContent(FlexAlign.Center)
    .backgroundColor(this.isFollowing() ?
      this.getPalette().accentMuted :
      this.getPalette().accentBackground)
    .border(this.isFollowing() ?
      { width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().accentBackground } :
      { width: 0 })
    .shadow(this.isFollowing() ? undefined : this.getPalette().accentShadow)
    .scale(this.followPressed ?
      { x: DesignConstants.INTERACTION.SCALE_PRESSED_ICON, y: DesignConstants.INTERACTION.SCALE_PRESSED_ICON } :
      { x: 1, y: 1 })
    .animation({ duration: DesignConstants.ANIMATION.DURATION_FAST, curve: Curve.EaseOut })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.followPressed = true;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.followPressed = false;
      }
    })
    .onClick(() => this.toggleFollow())
    .accessibilityText(this.isFollowing() ?
      `取消关注 ${this.user.user?.nickname ?? ''}` :
      `关注 ${this.user.user?.nickname ?? ''}`)
  }

  build() {
    NearbyImageCard({
      imageUrl: this.user.user?.avatar_url ?? '',
      imageAlt: this.user.user?.nickname ?? '',
      cardAspectRatio: 0.75,
      placeholderText: this.user.user?.nickname ?? '',
      badgeText: this.showDistance ? this.getDistanceText() : '',
      badgeIcon: AppIcon.LOCATION,
      badgeShape: 'pill',
      badgeAlign: 'topRight',
      onPress: () => this.openProfile()
    }) {
      Column() {
        Row() {
          Column({ space: DesignConstants.SPACING.THREE }) {
            Text(this.user.user?.nickname ?? '')
              .fontSize(DesignConstants.FONT.SIZE.MD)
              .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
              .fontColor(this.getPalette().textPrimary)
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })

            if (this.user.bio) {
              Text(this.user.bio)
                .fontSize(DesignConstants.FONT.SIZE.XS)
                .fontColor(this.getPalette().textSecondary)
                .maxLines(1)
                .textOverflow({ overflow: TextOverflow.Ellipsis })
            }
          }
          .alignItems(HorizontalAlign.Start)
          .layoutWeight(1)

          if (this.isOnline()) {
            Row()
              .width(DesignConstants.SPACING.MD2)
              .height(DesignConstants.SPACING.MD2)
              .borderRadius(DesignConstants.RADIUS.FULL)
              .backgroundColor(this.getPalette().online)
              .border({
                width: DesignConstants.BORDER_WIDTH.STRONG,
                color: this.getPalette().surfaceElevated
              })
              .margin({ left: DesignConstants.SPACING.SM })
          }
        }
        .width('100%')
        .alignItems(VerticalAlign.Center)

        Row() {
          if (this.getPrimaryTag().length > 0) {
            Text(`#${this.getPrimaryTag()}`)
              .fontSize(DesignConstants.FONT.SIZE.XS2)
              .fontColor(this.getPalette().accentText)
              .backgroundColor(this.getPalette().accentMuted)
              .borderRadius(DesignConstants.RADIUS.FULL)
              .padding({
                left: DesignConstants.SPACING.SM2,
                right: DesignConstants.SPACING.SM2,
                top: DesignConstants.SPACING.HALF,
                bottom: DesignConstants.SPACING.HALF
              })
              .maxLines(1)
          }

          Blank()

          this.buildFollowButton()
        }
        .width('100%')
        .alignItems(VerticalAlign.Center)
        .margin({ top: DesignConstants.SPACING.SM })
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
