import { GroupItem } from '../../../core/types/api';
import { interactionActions } from '../../../core/state/InteractionActions';
import { LocationUtil } from '../../../core/location/LocationUtil';
import { AppRouter, GroupDetailRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { NearbyImageCard } from '../../../components/NearbyImageCard';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';

/**
 * 附近群组卡 — 基于 NearbyImageCard 的图片卡（对齐前端 nearby/groups/shared/GroupCard）。
 * featured=false：3:4 网格卡（左上分类胶囊 + 群名/成员/距离/活动行 + 加入 pill）；
 * featured=true：全宽精选大卡（「本周精选」玻璃徽章 + 大号 footer）。
 * 成员头像堆叠前端使用群内好友头像列表，本端契约仅有 friends_in_group 计数，
 * 降级为成员数文本。加入/退出走 InteractionActions 乐观更新。
 */
@Reusable
@Component
export struct GroupCard {
  @Prop group: GroupItem;
  @Prop featured: boolean = false;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @State private joinPressed: boolean = false;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getGroupId(): string {
    return this.group.group?.group_id ?? '';
  }

  private hasJoined(): boolean {
    return this.group.viewer_state !== null && this.group.viewer_state !== undefined;
  }

  private isPrivate(): boolean {
    return this.group.join_method !== 'direct';
  }

  private getCategory(): string {
    const tags = this.group.tags ?? [];
    return tags.length > 0 ? tags[0] : '';
  }

  private getDistanceText(): string {
    const distance = this.group.location?.distance;
    if (distance === undefined || distance === null) {
      return '';
    }
    return LocationUtil.formatDistance(distance);
  }

  private formatMembers(): string {
    const count = this.group.group?.members_count ?? 0;
    if (count >= 10000) {
      return `${(count / 10000).toFixed(1)}w`;
    }
    if (count >= 1000) {
      return `${(count / 1000).toFixed(1)}k`;
    }
    return `${count}`;
  }

  private getActivityText(): string {
    const activityAt = this.group.next_activity_at;
    if (!activityAt) {
      return '';
    }
    const date = new Date(activityAt);
    if (Number.isNaN(date.getTime())) {
      return '';
    }
    return `${date.getMonth() + 1}月${date.getDate()}日活动`;
  }

  private openGroupDetail(): void {
    const groupId = this.getGroupId();
    if (groupId) {
      const params = new GroupDetailRouteParam();
      params.groupId = groupId;
      AppRouter.push(PageNames.GROUP_DETAIL, params);
    }
  }

  private toggleJoin(): void {
    const groupId = this.getGroupId();
    if (!groupId) {
      return;
    }
    if (this.hasJoined()) {
      interactionActions.leaveGroup(groupId);
    } else {
      interactionActions.joinGroup(groupId);
    }
  }

  @Builder
  private buildJoinButton() {
    Text(this.hasJoined() ? '已加入' : '加入')
      .id(`${NearbyImageCard.CHILD_CONTROL_PREFIX}join-${this.getGroupId()}`)
      .fontSize(DesignConstants.FONT.SIZE.XS)
      .fontWeight(this.hasJoined() ? DesignConstants.FONT.WEIGHT.REGULAR : DesignConstants.FONT.WEIGHT.SEMIBOLD)
      .fontColor(this.hasJoined() ?
        this.getPalette().textSecondary :
        this.getPalette().accentText)
      .backgroundColor(this.hasJoined() ?
        DesignConstants.COLOR.TRANSPARENT :
        this.getPalette().accentBackground)
      .border(this.hasJoined() ?
        { width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().borderColor } :
        { width: 0 })
      .borderRadius(DesignConstants.RADIUS.FULL)
      .padding({
        left: DesignConstants.SPACING.MD2,
        right: DesignConstants.SPACING.MD2,
        top: DesignConstants.SPACING.THREE,
        bottom: DesignConstants.SPACING.THREE
      })
      .shadow(this.hasJoined() ? undefined : this.getPalette().accentShadow)
      .scale(this.joinPressed ?
        { x: DesignConstants.INTERACTION.SCALE_PRESSED_PILL, y: DesignConstants.INTERACTION.SCALE_PRESSED_PILL } :
        { x: 1, y: 1 })
      .animation({ duration: DesignConstants.ANIMATION.DURATION_FAST, curve: Curve.EaseOut })
      .onTouch((event: TouchEvent) => {
        if (event.type === TouchType.Down) {
          this.joinPressed = true;
        } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
          this.joinPressed = false;
        }
      })
      .onClick(() => this.toggleJoin())
      .accessibilityText(this.hasJoined() ? `退出群组 ${this.group.group?.name ?? ''}` : `加入群组 ${this.group.group?.name ?? ''}`)
  }

  @Builder
  private buildNameLockIcon() {
    if (this.isPrivate()) {
      Text() {
        SymbolSpan($r(AppIcon.LOCK))
          .fontSize(DesignConstants.FONT.SIZE.XS2)
          .fontColor([this.getPalette().textSecondary])
      }
      .margin({ left: DesignConstants.SPACING.HALF })
    }
  }

  @Builder
  private buildCompactFooter() {
    Column({ space: DesignConstants.SPACING.XS }) {
      Row() {
        Text(this.group.group?.name ?? '')
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
          .fontColor(this.getPalette().textPrimary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
        this.buildNameLockIcon()
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)

      Row({ space: DesignConstants.SPACING.SM }) {
        Text() {
          SymbolSpan($r(AppIcon.GROUP))
            .fontSize(DesignConstants.FONT.SIZE.XS2)
            .fontColor([this.getPalette().textSecondary])
        }
        Text(`${this.formatMembers()} 人`)
          .fontSize(DesignConstants.FONT.SIZE.XS2)
          .fontColor(this.getPalette().textSecondary)

        Blank()

        if (this.getDistanceText().length > 0) {
          Row({ space: DesignConstants.SPACING.HALF }) {
            Text() {
              SymbolSpan($r(AppIcon.LOCATION))
                .fontSize(DesignConstants.FONT.SIZE.XS2)
                .fontColor([this.getPalette().accentText])
            }
            Text(this.getDistanceText())
              .fontSize(DesignConstants.FONT.SIZE.XS2)
              .fontColor(this.getPalette().accentText)
          }
          .alignItems(VerticalAlign.Center)
        }
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)

      Row() {
        if (this.getActivityText().length > 0) {
          Row({ space: DesignConstants.SPACING.HALF }) {
            Text() {
              SymbolSpan($r(AppIcon.TIME))
                .fontSize(DesignConstants.FONT.SIZE.XS2)
                .fontColor([this.getPalette().online])
            }
            Text(this.getActivityText())
              .fontSize(DesignConstants.FONT.SIZE.XS2)
              .fontColor(this.getPalette().online)
          }
          .alignItems(VerticalAlign.Center)
        }

        Blank()

        this.buildJoinButton()
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)
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

  @Builder
  private buildFeaturedFooter() {
    Column({ space: DesignConstants.SPACING.XS }) {
      Row() {
        Text(this.group.group?.name ?? '')
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
          .fontColor(this.getPalette().textPrimary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
        this.buildNameLockIcon()
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)

      if (this.group.description) {
        Text(this.group.description)
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .fontColor(this.getPalette().textSecondary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
      }

      Row({ space: DesignConstants.SPACING.MD2 }) {
        Row({ space: DesignConstants.SPACING.HALF }) {
          Text() {
            SymbolSpan($r(AppIcon.GROUP))
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor([this.getPalette().textSecondary])
          }
          Text(`${this.formatMembers()} 人`)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getPalette().textSecondary)
        }
        .alignItems(VerticalAlign.Center)

        Row({ space: DesignConstants.SPACING.HALF }) {
          Row()
            .width(DesignConstants.SPACING.SM2)
            .height(DesignConstants.SPACING.SM2)
            .borderRadius(DesignConstants.RADIUS.FULL)
            .backgroundColor(this.getPalette().online)
          Text(`${this.group.online_members_count ?? 0} 在线`)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getPalette().online)
        }
        .alignItems(VerticalAlign.Center)

        if (this.getDistanceText().length > 0) {
          Text(this.getDistanceText())
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getPalette().accentText)
        }
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)
      .margin({ top: DesignConstants.SPACING.SM })

      Row() {
        if (this.getActivityText().length > 0) {
          Row({ space: DesignConstants.SPACING.HALF }) {
            Text() {
              SymbolSpan($r(AppIcon.TIME))
                .fontSize(DesignConstants.FONT.SIZE.XS2)
                .fontColor([this.getPalette().accentText])
            }
            Text(this.getActivityText())
              .fontSize(DesignConstants.FONT.SIZE.XS2)
              .fontColor(this.getPalette().accentText)
          }
          .alignItems(VerticalAlign.Center)
        }

        Blank()

        this.buildJoinButton()
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)
      .margin({ top: DesignConstants.SPACING.SM })
    }
    .width('100%')
    .alignItems(HorizontalAlign.Start)
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.LG,
      bottom: DesignConstants.SPACING.LG
    })
  }

  build() {
    if (this.featured) {
      NearbyImageCard({
        imageUrl: this.group.group?.avatar_url ?? '',
        imageAlt: this.group.group?.name ?? '',
        fixedHeight: DesignConstants.COMPONENT.GROUP_FEATURED_HEIGHT,
        placeholderText: this.group.group?.name ?? '',
        badgeText: '本周精选',
        badgeIcon: AppIcon.STAR,
        badgeShape: 'pill',
        badgeVariant: 'glass',
        badgeAlign: 'topLeft',
        onPress: () => this.openGroupDetail()
      }) {
        this.buildFeaturedFooter()
      }
    } else {
      NearbyImageCard({
        imageUrl: this.group.group?.avatar_url ?? '',
        imageAlt: this.group.group?.name ?? '',
        cardAspectRatio: 0.75,
        placeholderText: this.group.group?.name ?? '',
        badgeText: this.getCategory(),
        badgeShape: 'pill',
        badgeVariant: 'accent',
        badgeAlign: 'topLeft',
        onPress: () => this.openGroupDetail()
      }) {
        this.buildCompactFooter()
      }
    }
  }
}
