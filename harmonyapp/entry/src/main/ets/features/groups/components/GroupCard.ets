import { GroupItem } from '../../../core/types/api';
import { interactionActions } from '../../../core/state/InteractionActions';
import { interactionEventHub, GroupMemberState, GroupMembershipState } from '../../../core/state/InteractionStateStore';
import { LocationUtil } from '../../../core/location/LocationUtil';
import { AppRouter, GroupDetailRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { MotionUtil } from '../../../core/accessibility/MotionUtil';
import { DesignConstants, textLineHeight } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { NearbyImageCard } from '../../../components/NearbyImageCard';
import { Pressable } from '../../../components/Pressable';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { LucideIcon } from '../../../components/LucideIcon';
import { SystemDialog } from '../../../components/SystemDialog';

/**
 * 群组卡（附近域） — 两种形态，均对齐 web 移动端：
 * - card（默认）：白底名片卡，对齐 web /nearby/groups 双列卡。20 圆角白卡 + ring 描边 +
 *   shadow-xs；左上分类 chip（深色强遮罩白字）、右上私密锁圆；居中字母圆（accent-soft 底）
 *   + 群名 14/600 + 成员/距离 meta 行 + 底部「加入」pill（视觉 36vp、热区 44vp）。
 *   不再整卡铺封面图（web 该卡的无图形态，避免 3:4 图片卡与名片卡两种语言并存）。
 * - carousel：附近枢纽「附近群组」横滑卡（对齐 web GroupCarousel 140×140 方形图卡）：
 *   封面铺满 + 底部姓名/成员/距离托盘 + 右上隐私圆徽章。
 * 加入/退出走 InteractionActions 乐观更新（写 InteractionStateStore 并广播
 * group_membership_updated），本卡订阅该事件刷新加入态——viewer_state 只是搜索响应的
 * 静态快照，加入后不会自动更新。
 */
@Reusable
@Component
export struct GroupCard {
  @Prop group: GroupItem;
  /** 形态：card = 白底名片卡（列表双列）；carousel = 枢纽横滑方形图卡 */
  @Prop variant: string = 'card';
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  /** 群组名片居中字母圆直径（对齐 web 群卡占位圆 size-16=64；AVATAR 刻度无 64 档，
   *  按布局几何登记在该组件内） */
  private static readonly AVATAR_SIZE: number = 64;

  // 加入态（对齐 UserCard / MomentFeedCard 的 store 订阅范式）：
  // 事件到达本群时写入 @State 触发重渲染，真值判定见 hasJoined()。
  @State private memberState: GroupMemberState | undefined = undefined;
  /** memberState 归属的群 id：@Reusable 复用不重放 aboutToAppear，@State 可能仍属于
   *  上一个群，故仅在与当前群 id 一致时才采信，否则现查 store / 回退响应快照。 */
  private memberStateGroupId: string = '';
  /** carousel 图卡：是否有封面图（false = fallback，底部托盘换主题文字色） */
  @State private mediaActive: boolean = true;
  @State private showLeaveConfirm: boolean = false;
  @State private joinPopped: boolean = false;

  aboutToAppear(): void {
    this.memberState = interactionEventHub.getGroupState(this.getGroupId())?.my_state;
    this.memberStateGroupId = this.getGroupId();
    // 横滑卡：无封面时底部托盘走主题文字色（有封面时由 NearbyImageCard 回调同步）
    this.mediaActive = (this.group.group?.avatar_url ?? '').length > 0;
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

  private getGroupName(): string {
    return this.group.group?.name ?? '';
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

  private toggleJoin(): void {
    const groupId = this.getGroupId();
    if (!groupId) {
      return;
    }
    if (this.hasJoined()) {
      this.showLeaveConfirm = true;
    } else {
      // R7 加入 pop（仅激活方向）
      MotionUtil.statePop(this.getUIContext(),
        () => { this.joinPopped = true; },
        () => { this.joinPopped = false; });
      interactionActions.joinGroup(groupId);
    }
  }

  /** 名片卡高度：由 token 档位推导（padding + 顶栏 chip + 字母圆/群名/meta + 底部 pill），
   *  ArkUI 的 height 为 border-box（含 padding）；末尾留 XS 余量吸收行高取整，保证不裁切。 */
  private getCardHeight(): number {
    return DesignConstants.SPACING.MD2 * 2
      + DesignConstants.SPACING.LG2
      + GroupCard.AVATAR_SIZE
      + DesignConstants.SPACING.SM2
      + textLineHeight(DesignConstants.FONT.SIZE.MD)
      + DesignConstants.SPACING.HALF
      + textLineHeight(DesignConstants.FONT.SIZE.XS)
      + DesignConstants.COMPONENT.BUTTON_HEIGHT_SM
      + DesignConstants.SPACING.XS;
  }

  private getJoinForeground(): ResourceColor {
    return this.hasJoined() ? this.getPalette().textSecondary : this.getPalette().accentBackground;
  }

  @Builder
  private buildJoinPill() {
    // 按压反馈统一走 Pressable（springMotion 手感）；fillWidth:false 自适应 pill 内容宽
    Pressable({
      onPress: () => this.toggleJoin(),
      pressScale: DesignConstants.INTERACTION.SCALE_PRESSED_PILL,
      fillWidth: false
    }) {
      Row({ space: DesignConstants.SPACING.HALF }) {
        if (this.hasJoined()) {
          LucideIcon({
            icon: AppIcon.CHECK,
            iconSize: DesignConstants.FONT.SIZE.XS,
            color: this.getJoinForeground()
          })
        }
        Text(this.hasJoined() ? '已加入' : '加入')
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
          .fontWeight(this.hasJoined() ? DesignConstants.FONT.WEIGHT.MEDIUM : DesignConstants.FONT.WEIGHT.SEMIBOLD)
          .fontColor(this.getJoinForeground())
          .maxLines(1)
      }
      .height(DesignConstants.COMPONENT.BUTTON_HEIGHT_SM)
      .padding({ left: DesignConstants.SPACING.MD3, right: DesignConstants.SPACING.MD3 })
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(this.hasJoined() ? this.getPalette().surfaceSubtle : this.getPalette().accentMuted)
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)
      .scale({ x: MotionUtil.popScale(this.joinPopped), y: MotionUtil.popScale(this.joinPopped) })
      .accessibilityText(this.hasJoined() ? `退出群组 ${this.getGroupName()}` : `加入群组 ${this.getGroupName()}`)
      // 触控热区：视觉 36vp，responseRegion 纵向外扩到 MIN_HIT_TARGET(44)（不占布局）
      .responseRegion({
        x: 0,
        y: -(DesignConstants.LAYOUT.MIN_HIT_TARGET - DesignConstants.COMPONENT.BUTTON_HEIGHT_SM) / 2,
        width: '100%',
        height: DesignConstants.LAYOUT.MIN_HIT_TARGET
      })
    }
  }

  @Builder
  private buildPrivacyBadge() {
    if (this.isPrivate()) {
      Row() {
        LucideIcon({
          icon: AppIcon.LOCK,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
          color: DesignConstants.COLOR.WHITE
        })
      }
      .width(DesignConstants.LAYOUT.UNREAD_BADGE_SIZE)
      .height(DesignConstants.LAYOUT.UNREAD_BADGE_SIZE)
      .borderRadius(DesignConstants.RADIUS.FULL)
      // 深色强遮罩圆徽章（对齐 web media-badge--icon；纯装饰，不进读屏）
      .backgroundColor(DesignConstants.COLOR.MEDIA_SCRIM_STRONG)
      .justifyContent(FlexAlign.Center)
      .accessibilityLevel('no')
    }
  }

  @Builder
  private buildCategoryChip() {
    if (this.getCategory().length > 0) {
      Row() {
        Text(this.getCategory())
          .fontSize(DesignConstants.FONT.SIZE.XS2)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS2))
          .fontColor(DesignConstants.COLOR.WHITE)
          .maxLines(1)
      }
      .height(DesignConstants.SPACING.LG2)
      .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM })
      // 深色强遮罩底白字（对齐 web 群卡分类角标 media-badge）
      .backgroundColor(DesignConstants.COLOR.MEDIA_SCRIM_STRONG)
      .borderRadius(DesignConstants.RADIUS.FULL)
    }
  }

  @Builder
  private buildCardTopRow() {
    Row() {
      this.buildCategoryChip()
      Blank()
      this.buildPrivacyBadge()
    }
    .width('100%')
    .height(DesignConstants.SPACING.LG2)
    .alignItems(VerticalAlign.Center)
  }

  @Builder
  private buildCardCenter() {
    Column() {
      // 居中字母圆：群头像缺省时 SystemAvatar 用 accent-soft 底 + 首字母兜底
      SystemAvatar({
        avatarUrl: this.group.group?.avatar_url ?? '',
        name: this.getGroupName(),
        avatarSize: GroupCard.AVATAR_SIZE,
        shape: 'circle',
        decorative: true
      })
        .shadow(this.getPalette().surfaceShadow)

      Text(this.getGroupName())
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
        .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
        .fontColor(this.getPalette().textPrimary)
        .maxLines(1)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
        .textAlign(TextAlign.Center)
        .width('100%')
        .margin({ top: DesignConstants.SPACING.SM2 })

      // meta 行：左「成员数」/ 右「距离」（对齐 web 群卡 meta 行）
      Row() {
        Row({ space: DesignConstants.SPACING.HALF }) {
          LucideIcon({
            icon: AppIcon.GROUP,
            iconSize: DesignConstants.FONT.SIZE.XS,
            color: this.getPalette().textSecondary
          })
          Text(`${this.formatMembers()} 人`)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
            .fontColor(this.getPalette().textSecondary)
            .maxLines(1)
        }
        .alignItems(VerticalAlign.Center)

        Blank()

        if (this.getDistanceText().length > 0) {
          Row({ space: DesignConstants.SPACING.HALF }) {
            LucideIcon({
              icon: AppIcon.LOCATION,
              iconSize: DesignConstants.FONT.SIZE.XS,
              color: this.getPalette().accentBackground
            })
            Text(this.getDistanceText())
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
              .fontColor(this.getPalette().accentBackground)
              .maxLines(1)
          }
          .alignItems(VerticalAlign.Center)
        }
      }
      .width('100%')
      .margin({ top: DesignConstants.SPACING.HALF })
    }
    .width('100%')
    .layoutWeight(1)
    .justifyContent(FlexAlign.Center)
  }

  @Builder
  private buildCardBottomRow() {
    Row() {
      if (this.getActivityText().length > 0) {
        Row({ space: DesignConstants.SPACING.HALF }) {
          LucideIcon({
            icon: AppIcon.TIME,
            iconSize: DesignConstants.FONT.SIZE.XS2,
            color: this.getPalette().online
          })
          Text(this.getActivityText())
            .fontSize(DesignConstants.FONT.SIZE.XS2)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS2))
            .fontColor(this.getPalette().online)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .alignItems(VerticalAlign.Center)
        .flexShrink(1)
      }

      Blank()

      this.buildJoinPill()
    }
    .width('100%')
    .height(DesignConstants.COMPONENT.BUTTON_HEIGHT_SM)
    .alignItems(VerticalAlign.Center)
  }

  @Builder
  private buildNameCard() {
    Pressable({
      onPress: () => this.openGroupDetail(),
      pressScale: DesignConstants.INTERACTION.SCALE_PRESSED_CARD,
      hoverFill: this.getPalette().surfaceSubtle
    }) {
      Column() {
        this.buildCardTopRow()
        this.buildCardCenter()
        this.buildCardBottomRow()
      }
      .width('100%')
      .height(this.getCardHeight())
      .padding(DesignConstants.SPACING.MD2)
      .backgroundColor(this.getPalette().surfaceIsland)
      // web 群卡：ring-1 border-strong + rounded-lg(20) + shadow-xs
      .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().borderColor })
      .borderRadius(DesignConstants.SHAPE.CARD)
      .shadow(this.getPalette().surfaceShadow)
      .clip(true)
      .accessibilityText(`查看群组 ${this.getGroupName()}`)
    }
  }

  @Builder
  private buildCarouselFooter() {
    Column({ space: DesignConstants.SPACING.HALF }) {
      Text(this.getGroupName())
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
        .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
        .fontColor(this.isOnMedia() ? DesignConstants.COLOR.WHITE : this.getPalette().textPrimary)
        .maxLines(1)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
        .width('100%')

      Row() {
        Row({ space: DesignConstants.SPACING.HALF }) {
          LucideIcon({
            icon: AppIcon.GROUP,
            iconSize: DesignConstants.FONT.SIZE.XS2,
            color: this.isOnMedia() ? DesignConstants.COLOR.WHITE : this.getPalette().textSecondary
          })
          Text(`${this.formatMembers()} 成员`)
            .fontSize(DesignConstants.FONT.SIZE.XS2)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS2))
            .fontColor(this.isOnMedia() ? DesignConstants.COLOR.WHITE : this.getPalette().textSecondary)
            .opacity(this.isOnMedia() ? DesignConstants.OPACITY.EMPHASIZED : 1)
            .maxLines(1)
        }
        .alignItems(VerticalAlign.Center)

        Blank()

        if (this.getDistanceText().length > 0) {
          Row({ space: DesignConstants.SPACING.HALF }) {
            LucideIcon({
              icon: AppIcon.LOCATION,
              iconSize: DesignConstants.FONT.SIZE.XS2,
              color: this.isOnMedia() ? DesignConstants.COLOR.WHITE : this.getPalette().accentBackground
            })
            Text(this.getDistanceText())
              .fontSize(DesignConstants.FONT.SIZE.XS2)
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS2))
              .fontColor(this.isOnMedia() ? DesignConstants.COLOR.WHITE : this.getPalette().accentBackground)
              .maxLines(1)
          }
          .alignItems(VerticalAlign.Center)
        }
      }
      .width('100%')
    }
    .width('100%')
    .alignItems(HorizontalAlign.Start)
    .padding({
      left: DesignConstants.SPACING.MD2,
      right: DesignConstants.SPACING.MD2,
      top: DesignConstants.SPACING.SM,
      bottom: DesignConstants.SPACING.SM
    })
  }

  /** 群名字面圆首字（无封面横滑卡的占位字母） */
  /** 横滑卡当前是否「压在封面图上」（决定底部托盘白字/主题字）：无封面或图加载失败走主题字 */
  private isOnMedia(): boolean {
    return (this.group.group?.avatar_url ?? '').length > 0 && this.mediaActive;
  }

  private getInitialChar(): string {
    const name: string = this.getGroupName();
    return name.length > 0 ? name.substring(0, 1) : '群';
  }

  /** 无封面横滑卡：accent 渐变底 + 居中字母圆（对齐 web GroupCarousel 占位形态） */
  @Builder
  private buildCarouselFallback() {
    Column() {
      Stack() {
        Column()
          .width('100%')
          .height('100%')
          .linearGradient({
            angle: 135,
            colors: [
              [this.getPalette().accentMuted, 0],
              [this.getPalette().surfaceIsland, 1]
            ] as [ResourceColor, number][]
          })

        Row() {
          Text(this.getInitialChar())
            .fontSize(DesignConstants.FONT.SIZE.XXXL)
            .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
            .fontColor(this.getPalette().accentBackground)
        }
        .width(GroupCard.AVATAR_SIZE)
        .height(GroupCard.AVATAR_SIZE)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .backgroundColor(this.getPalette().accentMuted)
        .justifyContent(FlexAlign.Center)
      }
      .width('100%')
      .layoutWeight(1)

      this.buildCarouselFooter()
    }
    .width(DesignConstants.COMPONENT.GROUP_CARD_WIDTH)
    .height(DesignConstants.COMPONENT.GROUP_CARD_WIDTH)
    .backgroundColor(this.getPalette().surfaceIsland)
    .borderRadius(DesignConstants.SHAPE.CARD)
    .clip(true)
  }

  @Builder
  private buildCarouselCard() {
    Column() {
      if ((this.group.group?.avatar_url ?? '').length === 0) {
        this.buildCarouselFallback()
      } else if (this.isPrivate()) {
        // 右上隐私圆徽章（仅私密群）：公开群组不渲染角标（web 用 globe，
        // 本端 AppIcon 无 globe 符号，登记为平台差异）
        NearbyImageCard({
          imageUrl: this.group.group?.avatar_url ?? '',
          imageAlt: this.getGroupName(),
          cardAspectRatio: 1,
          cardRadius: DesignConstants.SHAPE.CARD,
          mediaOverlay: true,
          fallbackIcon: AppIcon.GROUP,
          placeholderText: this.getGroupName(),
          badgeShape: 'circle',
          badgeIcon: AppIcon.LOCK,
          badgeAlign: 'topRight',
          onMediaStateChange: (active: boolean) => {
            this.mediaActive = active;
          },
          onPress: () => this.openGroupDetail()
        }) {
          this.buildCarouselFooter()
        }
      } else {
        NearbyImageCard({
          imageUrl: this.group.group?.avatar_url ?? '',
          imageAlt: this.getGroupName(),
          cardAspectRatio: 1,
          cardRadius: DesignConstants.SHAPE.CARD,
          mediaOverlay: true,
          fallbackIcon: AppIcon.GROUP,
          placeholderText: this.getGroupName(),
          onMediaStateChange: (active: boolean) => {
            this.mediaActive = active;
          },
          onPress: () => this.openGroupDetail()
        }) {
          this.buildCarouselFooter()
        }
      }
    }
    .width(DesignConstants.COMPONENT.GROUP_CARD_WIDTH)
  }

  build() {
    Stack() {
      if (this.variant === 'carousel') {
        this.buildCarouselCard()
      } else {
        this.buildNameCard()
      }

      // 退出群组二次确认（destructive 防误触）
      SystemDialog({
        visible: this.showLeaveConfirm,
        title: '退出群组',
        description: `退出「${this.getGroupName() || '该群组'}」后需要重新申请加入`,
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
    .width('100%')
  }
}
