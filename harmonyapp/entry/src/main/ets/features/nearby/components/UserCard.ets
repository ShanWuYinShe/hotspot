import { UserItem } from '../../../core/types/api';
import { interactionActions } from '../../../core/state/InteractionActions';
import { UserRelationState, interactionEventHub } from '../../../core/state/InteractionStateStore';
import { LocationUtil } from '../../../core/location/LocationUtil';
import { AppRouter, ProfileRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { DesignConstants, textLineHeight } from '../../../constants/DesignConstants';
import { MotionUtil } from '../../../core/accessibility/MotionUtil';
import { AppIcon } from '../../../components/SystemIcon';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { Pressable } from '../../../components/Pressable';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { LucideIcon } from '../../../components/LucideIcon';

/**
 * 附近用户卡 — web 白底名片卡（对齐前端 components/data-display/UserCard.svelte 1x1 变体）：
 * - card：双列网格卡。20 圆角白卡 + ring 描边 + shadow-xs；顶栏分类 chip + 距离，
 *   中部居中头像（字母 fallback = accent-soft 圆）/姓名 14·600/副行 11 次色，
 *   底部居中关注 pill（视觉 36vp，热区外扩 44vp）。
 * - row：附近枢纽「附近的人」行式列表行（透明底，由外层软岛 ListGroup 承担外框）：
 *   avatar 44 + 名/副行 + 右侧关注 pill。
 * 关注态优先读 InteractionStateStore 乐观更新，回退响应 relation；
 * 事件总线按 user_id 过滤定向更新（快照冷启动播种见 aboutToAppear）。
 */
@Reusable
@Component
export struct UserCard {
  @Prop user: UserItem;
  @Prop showDistance: boolean = true;
  /** 形态：card = 白底名片卡（列表页双列）；row = 枢纽行式列表行 */
  @Prop variant: string = 'card';
  // 仅持本卡关系态：整 Map 订阅会让任意用户关注动作触发全部卡重算。
  @State private socialState: UserRelationState | undefined = undefined;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  @State private followInFlight: boolean = false;
  @State private followPopped: boolean = false;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  aboutToAppear(): void {
    // 播种：组件创建晚于状态写入时从 store 快照恢复
    const existing = interactionEventHub.getUserState(this.getUserId());
    if (existing) {
      this.socialState = existing;
    }
    interactionEventHub.on('user_relation_updated', this.handleUserStateUpdate);
  }

  aboutToDisappear(): void {
    interactionEventHub.off('user_relation_updated', this.handleUserStateUpdate);
  }

  private handleUserStateUpdate = (payload: Object): void => {
    const state = payload as UserRelationState;
    if (state && state.user_id === this.getUserId()) {
      this.socialState = state;
    }
  };

  private getUserId(): string {
    return this.user.user?.user_id ?? '';
  }

  private getNickname(): string {
    return this.user.user?.nickname ?? '';
  }

  private isFollowing(): boolean {
    const state = this.socialState;
    return state?.relation?.is_following ?? this.user.relation?.is_following ?? false;
  }

  private isOnline(): boolean {
    return this.user.presence_status === 'online';
  }

  private getPrimaryTag(): string {
    return this.user.tags && this.user.tags.length > 0 ? this.user.tags[0] : '';
  }

  /** 顶栏左侧分类 chip：无标签时回退「#热点邻居」（对齐 web UserCard 缺省档） */
  private getBadgeText(): string {
    const tag = this.getPrimaryTag();
    return tag.length > 0 ? `#${tag}` : '#热点邻居';
  }

  /** 副行：简介缺省「热点邻居」（对齐 web note || '热点邻居'） */
  private getSubtitle(): string {
    const bio = this.user.bio;
    return bio && bio.length > 0 ? bio : '热点邻居';
  }

  private getDistanceText(): string {
    if (this.user.location?.distance === undefined || this.user.location?.distance === null) {
      return '';
    }
    return LocationUtil.formatDistance(this.user.location.distance);
  }

  private openProfile(): void {
    const params = new ProfileRouteParam();
    params.userId = this.getUserId();
    AppRouter.push(PageNames.PROFILE, params);
  }

  private async toggleFollow(): Promise<void> {
    if (this.followInFlight) return;
    const wasFollowing: boolean = this.isFollowing();
    if (!wasFollowing) {
      // R7 关注 pop：仅未激活→激活方向
      MotionUtil.statePop(this.getUIContext(),
        () => { this.followPopped = true; },
        () => { this.followPopped = false; });
    }
    this.followInFlight = true;
    try {
      await interactionActions.setFollowUser(this.getUserId(), !wasFollowing);
    } finally {
      this.followInFlight = false;
    }
  }

  /** 名片卡高度：由 token 档位推导（padding + 顶栏 chip + 头像/姓名/副行 + 底部 pill），
   *  ArkUI 的 height 为 border-box（含 padding）；末尾留 XS 余量吸收行高取整，保证不裁切。 */
  private getCardHeight(): number {
    return DesignConstants.SPACING.MD * 2
      + DesignConstants.SPACING.LG2
      + DesignConstants.COMPONENT.AVATAR_SIZE.MEDIUM
      + DesignConstants.SPACING.SM
      + textLineHeight(DesignConstants.FONT.SIZE.MD)
      + DesignConstants.SPACING.HALF
      + textLineHeight(DesignConstants.FONT.SIZE.XS)
      + DesignConstants.COMPONENT.BUTTON_HEIGHT_SM
      + DesignConstants.SPACING.XS;
  }

  private getFollowForeground(): ResourceColor {
    return this.isFollowing() ? this.getPalette().textSecondary : this.getPalette().accentBackground;
  }

  @Builder
  private buildFollowPill() {
    Pressable({
      onPress: () => this.toggleFollow(),
      pressScale: DesignConstants.INTERACTION.SCALE_PRESSED_PILL,
      // 行内小控件不得撑满整行：fillWidth 缺省 true 会把 pill 的按压容器拉成整行宽
      fillWidth: false
    }) {
      Row({ space: DesignConstants.SPACING.XS }) {
        LucideIcon({
          // 图标语义对齐前端 FollowButton：未关注 UserPlus、已关注 Check
          icon: this.isFollowing() ? AppIcon.CHECK : AppIcon.PERSON_ADD,
          // 与同档文字（XS=11）并排时允许用字号档作图标尺寸（skill 已登记例外）
          iconSize: DesignConstants.FONT.SIZE.XS,
          color: this.getFollowForeground()
        })
        Text(this.isFollowing() ? '已关注' : '关注')
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
          .fontWeight(this.isFollowing() ? DesignConstants.FONT.WEIGHT.MEDIUM : DesignConstants.FONT.WEIGHT.SEMIBOLD)
          .fontColor(this.getFollowForeground())
          .maxLines(1)
      }
      .height(DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT)
      .padding({ left: DesignConstants.SPACING.MD3, right: DesignConstants.SPACING.MD3 })
      .borderRadius(DesignConstants.RADIUS.FULL)
      // 未关注 = accent-soft 淡底（引导点击）；已关注 = 中性软填（回到信息态），对齐 web FollowButton
      .backgroundColor(this.isFollowing() ? this.getPalette().surfaceSubtle : this.getPalette().accentMuted)
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)
      .scale({ x: MotionUtil.popScale(this.followPopped), y: MotionUtil.popScale(this.followPopped) })
      .accessibilityText(this.isFollowing() ? `取消关注 ${this.getNickname()}` : `关注 ${this.getNickname()}`)
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
  private buildCardTopRow() {
    Row() {
      // 左上分类 chip（10px neutral 底），无标签回退 #热点邻居
      Row() {
        Text(this.getBadgeText())
          .fontSize(DesignConstants.FONT.SIZE.XS2)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS2))
          .fontColor(this.getPalette().textSecondary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
      }
      .height(DesignConstants.SPACING.LG2)
      .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM })
      .backgroundColor(this.getPalette().surfaceSubtle)
      .borderRadius(DesignConstants.RADIUS.FULL)

      Blank()

      // 右上距离（MapPin + 10px 次色）
      if (this.showDistance && this.getDistanceText().length > 0) {
        Row({ space: DesignConstants.SPACING.HALF }) {
          LucideIcon({
            icon: AppIcon.LOCATION,
            iconSize: DesignConstants.FONT.SIZE.XS2,
            color: this.getPalette().accentBackground
          })
          Text(this.getDistanceText())
            .fontSize(DesignConstants.FONT.SIZE.XS2)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS2))
            .fontColor(this.getPalette().textSecondary)
            .maxLines(1)
        }
        .alignItems(VerticalAlign.Center)
      }
    }
    .width('100%')
    .height(DesignConstants.SPACING.LG2)
    .alignItems(VerticalAlign.Center)
  }

  @Builder
  private buildCardCenter() {
    Column() {
      // 居中头像：字母 fallback = accent-soft 圆（SystemAvatar 内建占位/渐显/失败降级）
      SystemAvatar({
        avatarUrl: this.user.user?.avatar_url ?? '',
        name: this.getNickname(),
        avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.MEDIUM,
        isOnline: this.isOnline(),
        onlineDotSize: DesignConstants.COMPONENT.ONLINE_DOT_LG,
        shape: 'circle',
        decorative: true
      })

      Text(this.getNickname())
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
        .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
        .fontColor(this.getPalette().textPrimary)
        .maxLines(1)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
        .textAlign(TextAlign.Center)
        .width('100%')
        .margin({ top: DesignConstants.SPACING.SM })

      Text(this.getSubtitle())
        .fontSize(DesignConstants.FONT.SIZE.XS2)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS2))
        .fontColor(this.getPalette().textSecondary)
        .maxLines(1)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
        .textAlign(TextAlign.Center)
        .width('100%')
        .margin({ top: DesignConstants.SPACING.HALF })
    }
    .width('100%')
    .layoutWeight(1)
    .justifyContent(FlexAlign.Center)
  }

  @Builder
  private buildCard() {
    Pressable({
      onPress: () => this.openProfile(),
      pressScale: DesignConstants.INTERACTION.SCALE_PRESSED_CARD,
      hoverFill: this.getPalette().surfaceSubtle
    }) {
      Column() {
        this.buildCardTopRow()
        this.buildCardCenter()

        Row() {
          this.buildFollowPill()
        }
        .width('100%')
        .justifyContent(FlexAlign.Center)
      }
      .width('100%')
      .height(this.getCardHeight())
      // web 默认竖卡 p-3(12) / sm:p-3.5；移动端 12
      .padding(DesignConstants.SPACING.MD)
      .backgroundColor(this.getPalette().surfaceIsland)
      // web 名片卡：ring-1 border-strong + rounded-lg(20) + shadow-xs
      .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().borderColor })
      .borderRadius(DesignConstants.SHAPE.CARD)
      .shadow(this.getPalette().surfaceShadow)
      .clip(true)
      .accessibilityText(`查看${this.getNickname()}的主页`)
    }
  }

  @Builder
  private buildRow() {
    Pressable({
      onPress: () => this.openProfile(),
      pressScale: DesignConstants.INTERACTION.SCALE_PRESSED_INPUT_ROW,
      hoverFill: this.getPalette().surfaceSubtle
    }) {
      Row({ space: DesignConstants.SPACING.MD }) {
        SystemAvatar({
          avatarUrl: this.user.user?.avatar_url ?? '',
          name: this.getNickname(),
          avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL_PLUS,
          isOnline: this.isOnline(),
          onlineDotSize: DesignConstants.COMPONENT.ONLINE_DOT_MD,
          shape: 'circle',
          decorative: true
        })

        Column() {
          Text(this.getNickname())
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.getPalette().textPrimary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })

          Row({ space: DesignConstants.SPACING.SM2 }) {
            if (this.getPrimaryTag().length > 0) {
              Text(`#${this.getPrimaryTag()}`)
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
                .fontColor(this.getPalette().accentBackground)
                .maxLines(1)
            }
            if (this.getPrimaryTag().length > 0 && this.showDistance && this.getDistanceText().length > 0) {
              Text('·')
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
                .fontColor(this.getPalette().textSecondary)
                .opacity(DesignConstants.OPACITY.MUTED)
                .maxLines(1)
            }
            if (this.showDistance && this.getDistanceText().length > 0) {
              Text(this.getDistanceText())
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
                .fontColor(this.getPalette().textSecondary)
                .maxLines(1)
            }
          }
          .alignItems(VerticalAlign.Center)
          .margin({ top: DesignConstants.SPACING.HALF })
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)

        this.buildFollowPill()
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)
      .padding({
        left: DesignConstants.SPACING.MD,
        right: DesignConstants.SPACING.MD,
        top: DesignConstants.SPACING.MD,
        bottom: DesignConstants.SPACING.MD
      })
    }
  }

  build() {
    if (this.variant === 'row') {
      this.buildRow()
    } else {
      this.buildCard()
    }
  }
}
