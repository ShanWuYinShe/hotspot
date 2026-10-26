import { GroupItem } from '../../../core/types/api';
import { interactionActions } from '../../../core/state/InteractionActions';
import { interactionEventHub, GroupMemberState, GroupMembershipState } from '../../../core/state/InteractionStateStore';
import { LocationUtil } from '../../../core/location/LocationUtil';
import { AppRouter, GroupDetailRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { NearbyImageCard } from '../../../components/NearbyImageCard';
import { Pressable } from '../../../components/Pressable';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { LucideIcon } from '../../../components/LucideIcon';
import { SystemDialog } from '../../../components/SystemDialog';

/**
 * 附近群组卡 — 基于 NearbyImageCard 的图片卡（对齐前端 nearby/groups/shared/GroupCard）。
 * featured=false：3:4 网格卡（左上分类胶囊 + 群名/成员/距离/活动行 + 加入 pill）；
 * featured=true：全宽精选大卡（「本周精选」玻璃徽章 + 大号 footer）。
 * 成员头像堆叠前端使用群内好友头像列表，本端契约仅有 friends_in_group 计数，
 * 降级为成员数文本。加入/退出走 InteractionActions 乐观更新（写 InteractionStateStore
 * 并发 group_membership_updated 事件），本卡订阅该事件刷新加入态——viewer_state 只是
 * 搜索响应的静态快照，加入后不会自动更新。
 */
@Reusable
@Component
export struct GroupCard {
  @Prop group: GroupItem;
  @Prop featured: boolean = false;
  /** R2 整卡图片卡：是否有图（false = fallback，footer 换主题文字色） */
  @State private mediaActive: boolean = true;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  // 加入态（对齐 UserCard / MomentFeedCard 的 store 订阅范式）：
  // 事件到达本群时写入 @State 触发重渲染，真值判定见 hasJoined()。
  @State private memberState: GroupMemberState | undefined = undefined;
  /** memberState 归属的群 id：@Reusable 复用不重放 aboutToAppear，@State 可能仍属于
   *  上一个群，故仅在与当前群 id 一致时才采信，否则现查 store / 回退响应快照。 */
  private memberStateGroupId: string = '';

  aboutToAppear(): void {
    this.memberState = interactionEventHub.getGroupState(this.getGroupId())?.my_state;
    this.memberStateGroupId = this.getGroupId();
    interactionEventHub.on('group_membership_updated', this.handleGroupMembershipUpdate);
  }

  aboutToDisappear(): void {
    interactionEventHub.off('group_membership_updated', this.handleGroupMembershipUpdate);
  }

  private handleGroupMembershipUpdate = (payload: Object): void => {
    const state = payload as GroupMembershipState;
    if (state && state.group_id === this.getGroupId()) {
      this.memberState = state.my_state;
      this.memberStateGroupId = state.group_id;
    }
  };

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getGroupId(): string {
    return this.group.group?.group_id ?? '';
  }

  private hasJoined(): boolean {
    const groupId = this.getGroupId();
    // ① 本卡 @State：事件已按 group_id 定向写入本群（加入/退出是乐观更新，
    //    InteractionActions 首发写 store 并广播），@State 变更即触发重渲染。
    if (this.memberStateGroupId === groupId && this.memberState) {
      return this.memberState.is_member === true;
    }
    // ② 现查 store：@Reusable 复用不重放 aboutToAppear/@State 可能仍属于上一个群，
    //    组件晚建（store 先于卡片写入）时也走这一级；store 是乐观更新的单一事实源。
    const live = interactionEventHub.getGroupState(groupId)?.my_state;
    if (live) {
      return live.is_member === true;
    }
    // ③ 兜底：搜索响应的 viewer_state 静态快照（加入后不会自动刷新，故仅作最后一级）
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

  /** 退出群组二次确认可见性（destructive 防误触：已加入态一点即静默退群） */
  @State private showLeaveConfirm: boolean = false;

  private toggleJoin(): void {
    const groupId = this.getGroupId();
    if (!groupId) {
      return;
    }
    if (this.hasJoined()) {
      this.showLeaveConfirm = true;
    } else {
      interactionActions.joinGroup(groupId);
    }
  }

  @Builder
  private buildJoinButton() {
    // 按压反馈统一走 Pressable（P3-1，springMotion 手感）；fillWidth:false 自适应
    // 药丸内容宽（默认撑满会破坏 footer Row 的 Blank 布局）；id 保留在内部 Text
    // 供 NearbyImageCard CHILD_CONTROL_PREFIX 约定识别行内子控件
    Pressable({
      onPress: () => this.toggleJoin(),
      pressScale: DesignConstants.INTERACTION.SCALE_PRESSED_PILL,
      fillWidth: false
    }) {
      Text(this.hasJoined() ? '已加入' : '加入')
        .id(`${NearbyImageCard.CHILD_CONTROL_PREFIX}join-${this.getGroupId()}`)
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .fontWeight(this.hasJoined() ? DesignConstants.FONT.WEIGHT.REGULAR : DesignConstants.FONT.WEIGHT.SEMIBOLD)
        .fontColor(this.hasJoined() ?
          (this.mediaActive ? DesignConstants.COLOR.WHITE : this.getPalette().textSecondary) :
          this.getPalette().accentText)
        .backgroundColor(this.hasJoined()
          ? (this.mediaActive ? DesignConstants.COLOR.MEDIA_SCRIM_STRONG : DesignConstants.COLOR.TRANSPARENT)
          : this.getPalette().accentBackground)
        .border(this.hasJoined()
          ? { width: DesignConstants.BORDER_WIDTH.HAIRLINE,
            color: this.mediaActive ? DesignConstants.COLOR.MEDIA_BORDER_WHITE_SOFT : this.getPalette().borderColor } :
          { width: 0 })
        .borderRadius(DesignConstants.RADIUS.FULL)
        .padding({
          left: DesignConstants.SPACING.MD,
          right: DesignConstants.SPACING.MD,
          top: DesignConstants.SPACING.THREE,
          bottom: DesignConstants.SPACING.THREE
        })
        .shadow(this.hasJoined() ? undefined : this.getPalette().surfaceShadow)
        .accessibilityText(this.hasJoined() ? `退出群组 ${this.group.group?.name ?? ''}` : `加入群组 ${this.group.group?.name ?? ''}`)
        // 触控热区：视觉高 ~18vp（Text+3vp padding），responseRegion 外扩到 44vp
        .responseRegion({
          x: -DesignConstants.SPACING.MD,
          y: -(DesignConstants.LAYOUT.MIN_HIT_TARGET - DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT) / 2,
          width: '130%',
          height: DesignConstants.LAYOUT.MIN_HIT_TARGET
        })
    }
  }

  @Builder
  private buildNameLockIcon() {
    if (this.isPrivate()) {
      LucideIcon({
          icon: AppIcon.LOCK,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
          color: this.mediaActive ? DesignConstants.COLOR.WHITE : this.getPalette().textSecondary
        })
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
          .fontColor((this.mediaActive ? DesignConstants.COLOR.WHITE : this.getPalette().textPrimary))
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
        this.buildNameLockIcon()
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)

      Row({ space: DesignConstants.SPACING.SM }) {
        LucideIcon({
            icon: AppIcon.GROUP,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
            color: (this.mediaActive ? DesignConstants.COLOR.WHITE : this.getPalette().textSecondary)
          })
        Text(`${this.formatMembers()} 人`)
          .fontSize(DesignConstants.FONT.SIZE.XS2)
          .fontColor((this.mediaActive ? DesignConstants.COLOR.WHITE : this.getPalette().textSecondary))

        Blank()

        if (this.getDistanceText().length > 0) {
          Row({ space: DesignConstants.SPACING.HALF }) {
            LucideIcon({
                icon: AppIcon.LOCATION,
                iconSize: DesignConstants.FONT.SIZE.XS2,
                color: (this.mediaActive ? DesignConstants.COLOR.WHITE : this.getPalette().accentBackground)
              })
            Text(this.getDistanceText())
              .fontSize(DesignConstants.FONT.SIZE.XS2)
              .fontColor((this.mediaActive ? DesignConstants.COLOR.WHITE : this.getPalette().accentBackground))
          }
          .alignItems(VerticalAlign.Center)
        }
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)

      Row() {
        if (this.getActivityText().length > 0) {
          Row({ space: DesignConstants.SPACING.HALF }) {
            LucideIcon({
                icon: AppIcon.TIME,
                iconSize: DesignConstants.FONT.SIZE.XS2,
                color: (this.mediaActive ? DesignConstants.COLOR.WHITE : this.getPalette().online)
              })
            Text(this.getActivityText())
              .fontSize(DesignConstants.FONT.SIZE.XS2)
              .fontColor((this.mediaActive ? DesignConstants.COLOR.WHITE : this.getPalette().online))
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
      top: DesignConstants.SPACING.MD,
      bottom: DesignConstants.SPACING.MD
    })
  }

  @Builder
  private buildFeaturedFooter() {
    Column({ space: DesignConstants.SPACING.XS }) {
      Row() {
        Text(this.group.group?.name ?? '')
          // 前端 featured 群名 text-lg = 18px（本端 18 档是 LG_PLUS；LG=16 是平台适配档）
          .fontSize(DesignConstants.FONT.SIZE.LG_PLUS)
          .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
          .fontColor((this.mediaActive ? DesignConstants.COLOR.WHITE : this.getPalette().textPrimary))
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
        this.buildNameLockIcon()
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)

      if (this.group.description) {
        Text(this.group.description)
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .fontColor((this.mediaActive ? DesignConstants.COLOR.WHITE : this.getPalette().textSecondary))
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
      }

      Row({ space: DesignConstants.SPACING.MD }) {
        Row({ space: DesignConstants.SPACING.HALF }) {
          LucideIcon({
              icon: AppIcon.GROUP,
              iconSize: DesignConstants.FONT.SIZE.XS,
              color: (this.mediaActive ? DesignConstants.COLOR.WHITE : this.getPalette().textSecondary)
            })
          Text(`${this.formatMembers()} 人`)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor((this.mediaActive ? DesignConstants.COLOR.WHITE : this.getPalette().textSecondary))
        }
        .alignItems(VerticalAlign.Center)

        Row({ space: DesignConstants.SPACING.HALF }) {
          Row()
            .width(DesignConstants.COMPONENT.INDICATOR_DOT_SIZE)
            .height(DesignConstants.COMPONENT.INDICATOR_DOT_SIZE)
            .borderRadius(DesignConstants.RADIUS.FULL)
            .backgroundColor((this.mediaActive ? DesignConstants.COLOR.WHITE : this.getPalette().online))
          Text(`${this.group.online_members_count ?? 0} 在线`)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor((this.mediaActive ? DesignConstants.COLOR.WHITE : this.getPalette().online))
        }
        .alignItems(VerticalAlign.Center)

        if (this.getDistanceText().length > 0) {
          Text(this.getDistanceText())
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor((this.mediaActive ? DesignConstants.COLOR.WHITE : this.getPalette().accentBackground))
        }
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)
      .margin({ top: DesignConstants.SPACING.SM })

      Row() {
        if (this.getActivityText().length > 0) {
          Row({ space: DesignConstants.SPACING.HALF }) {
            LucideIcon({
                icon: AppIcon.TIME,
                iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
                color: (this.mediaActive ? DesignConstants.COLOR.WHITE : this.getPalette().online)
              })
            Text(this.getActivityText())
              .fontSize(DesignConstants.FONT.SIZE.XS2)
              .fontColor((this.mediaActive ? DesignConstants.COLOR.WHITE : this.getPalette().online))
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
    Stack() {
      if (this.featured) {
      NearbyImageCard({
        imageUrl: this.group.group?.avatar_url ?? '',
        imageAlt: this.group.group?.name ?? '',
        // 无封面时：降高 + **不传 placeholderText**（NearbyImageCard 在该字段为空时不画占位圆，
        // 否则圆会被 footer 压住裁切——2026-10-21 端上实测）。留一条紧凑渐变条即可。
        fixedHeight: (this.group.group?.avatar_url ?? '').length > 0
          ? DesignConstants.COMPONENT.GROUP_FEATURED_HEIGHT
          : DesignConstants.COMPONENT.GROUP_FEATURED_HEIGHT_NO_COVER,
        placeholderText: (this.group.group?.avatar_url ?? '').length > 0
          ? (this.group.group?.name ?? '')
          : '',
        badgeText: '本周精选',
        badgeIcon: AppIcon.STAR,
        badgeShape: 'pill',
        badgeVariant: 'glass',
        badgeAlign: 'topLeft',
        cardRadius: DesignConstants.SHAPE.CARD,
        mediaOverlay: true,
        fallbackIcon: AppIcon.GROUP,
        onMediaStateChange: (active: boolean) => {
          this.mediaActive = active;
        },
        onPress: () => this.openGroupDetail()
      }) {
        this.buildFeaturedFooter()
      }
    } else {
      NearbyImageCard({
        imageUrl: this.group.group?.avatar_url ?? '',
        imageAlt: this.group.group?.name ?? '',
        // 前端 nearby/groups/shared/GroupCard 非精选卡在双列网格（手机端）是 **aspect 0.75 (3:4) 竖版**
        // 保证封面图片有充足高度（140vp+），避免 4/3 横版导致底部 footer 挤占 70% 空间仅剩 40vp 图条
        cardAspectRatio: 0.75,
        placeholderText: this.group.group?.name ?? '',
        badgeText: this.getCategory(),
        badgeShape: 'pill',
        // 分类胶囊走 glass（图上毛玻璃），对齐前端：accent 实底收给主按钮，
        // 卡片角标用弱对比玻璃底，避免每张卡一个高饱和色块
        badgeVariant: 'glass',
        badgeAlign: 'topLeft',
        cardRadius: DesignConstants.SHAPE.CARD,
        mediaOverlay: true,
        fallbackIcon: AppIcon.GROUP,
        onMediaStateChange: (active: boolean) => {
          this.mediaActive = active;
        },
        onPress: () => this.openGroupDetail()
      }) {
        this.buildCompactFooter()
      }
      }
      // 退出群组二次确认（destructive 防误触）
      SystemDialog({
        visible: this.showLeaveConfirm,
        title: '退出群组',
        description: `退出「${this.group.group?.name ?? '该群组'}」后需要重新申请加入`,
        confirmText: '退出',
        confirmVariant: 'danger',
        onCancel: () => { this.showLeaveConfirm = false; },
        onConfirm: () => {
          this.showLeaveConfirm = false;
          const groupId = this.getGroupId();
          if (groupId) {
            interactionActions.leaveGroup(groupId);
          }
        }
      })
    }
  }
}
