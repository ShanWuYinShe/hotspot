import { UserItem } from '../../../core/types/api';
import { interactionActions } from '../../../core/state/InteractionActions';
import { UserRelationState, interactionEventHub} from '../../../core/state/InteractionStateStore'
import { LocationUtil } from '../../../core/location/LocationUtil';
import { AppRouter, ProfileRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { DesignConstants, textLineHeight } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { NearbyImageCard } from '../../../components/NearbyImageCard';
import { Pressable } from '../../../components/Pressable';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { LucideIcon } from '../../../components/LucideIcon';

/**
 * 附近用户卡 — portrait 3:4 图片卡,头像即卡图,对齐前端 nearby/people/UserCard——
 * 右上距离毛玻璃徽章;底部托盘 = 昵称 + bio 副行 + 在线点、#tag 徽章 + 28vp 圆形关注钮。
 * (原 compact 竖排白卡随前端 /nearby 聚合门户删除一并移除,附近页改用本卡双列卡墙)
 * 关注态优先读 InteractionStateStore 乐观更新,回退响应 relation。
 */
@Reusable
@Component
export struct UserCard {
  @Prop user: UserItem;
  @Prop showDistance: boolean = true;
  // 仅持本卡关系态：整 Map 订阅会让任意用户关注动作触发全部卡重算。
  // 事件总线按 user_id 过滤定向更新（快照冷启动播种见 aboutToAppear）。
  @State private socialState: UserRelationState | undefined = undefined;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  @State private followInFlight: boolean = false;
  /** R2 整卡图片卡：是否有图（false = fallback，footer 换主题文字色） */
  @State private mediaActive: boolean = true;

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
    this.followInFlight = true;
    try {
      await interactionActions.setFollowUser(this.getUserId(), !this.isFollowing());
    } finally {
      this.followInFlight = false;
    }
  }

  @Builder
  private buildFollowButton() {
    // 按压反馈统一走 Pressable（P3-1）：原双态按压（未关注缩放/已关注边框加深）
    // 收敛为统一 spring 缩放手感；id 保留在内部 Row 供 NearbyImageCard
    // CHILD_CONTROL_PREFIX 约定识别行内子控件（属性放内容组件上——自定义组件
    // 尾随闭包后不可链式属性）
    Pressable({
      onPress: () => this.toggleFollow(),
      pressScale: DesignConstants.INTERACTION.SCALE_PRESSED_ICON,
      // 行内小控件不得撑满整行：fillWidth 缺省 true，会把 28vp 圆形关注钮的
      // 按压容器拉成整行宽（卡片里看起来是一条通栏底色块，且热区覆盖整行）
      fillWidth: false
    }) {
      Row() {
        LucideIcon({
          // 图标语义对齐前端 FollowButton(iconOnly)：未关注 UserPlus、已关注 Check
          icon: this.isFollowing() ? AppIcon.CHECK : AppIcon.PERSON_ADD,
          // web FollowButton iconOnly：图标 size-4(16)
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
          color: this.mediaActive ? DesignConstants.COLOR.WHITE :
            (this.isFollowing() ? this.getPalette().textSecondary : this.getPalette().accentBackground)
        })
      }
      .id(`${NearbyImageCard.CHILD_CONTROL_PREFIX}follow-${this.getUserId()}`)
      // web：size-9 = 36vp 圆钮（28vp 偏小，与 16vp 图标比例不匹配）
      .width(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
      .height(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .justifyContent(FlexAlign.Center)
      // 未关注 = accent-soft 淡底、无描边；已关注 = 透明底 + 强描边（web outline 变体）
      // 对齐 web FollowButton onMedia：两态均为强遮罩底 + 白色描边（未操作 40% / 已操作 70%），
      // 形状线索 = UserPlus ↔ Check，不只靠颜色
      .backgroundColor(this.mediaActive
        ? DesignConstants.COLOR.MEDIA_SCRIM_STRONG
        : (this.isFollowing() ? DesignConstants.COLOR.TRANSPARENT : this.getPalette().accentMuted))
      .border({
        width: this.mediaActive ? DesignConstants.BORDER_WIDTH.HAIRLINE :
          (this.isFollowing() ? DesignConstants.BORDER_WIDTH.HAIRLINE : 0),
        color: this.mediaActive
          ? (this.isFollowing() ? DesignConstants.COLOR.MEDIA_BORDER_WHITE_STRONG : DesignConstants.COLOR.MEDIA_BORDER_WHITE_SOFT)
          : this.getPalette().accentBackground
      })
      .accessibilityText(this.isFollowing() ?
        `取消关注 ${this.user.user?.nickname ?? ''}` :
        `关注 ${this.user.user?.nickname ?? ''}`)
      // 触控热区：视觉 36vp，responseRegion 外扩到 44vp（不占布局）
      .responseRegion({
        x: -(DesignConstants.LAYOUT.MIN_HIT_TARGET - DesignConstants.LAYOUT.ICON_BUTTON_SIZE) / 2,
        y: -(DesignConstants.LAYOUT.MIN_HIT_TARGET - DesignConstants.LAYOUT.ICON_BUTTON_SIZE) / 2,
        width: DesignConstants.LAYOUT.MIN_HIT_TARGET,
        height: DesignConstants.LAYOUT.MIN_HIT_TARGET
      })
    }
  }
  build() {
    NearbyImageCard({
      imageUrl: this.user.user?.avatar_url ?? '',
      imageAlt: this.user.user?.nickname ?? '',
      cardAspectRatio: 0.75,
      cardRadius: DesignConstants.SHAPE.CARD,
      // R2 整卡图片卡：图片铺满 3:4，信息覆盖图面（对齐 BRIEF §A 网格卡）
      mediaOverlay: true,
      placeholderText: this.user.user?.nickname ?? '',
      badgeText: this.showDistance ? this.getDistanceText() : '',
      badgeIcon: AppIcon.LOCATION,
      badgeShape: 'pill',
      badgeAlign: 'topRight',
      onMediaStateChange: (active: boolean) => {
        this.mediaActive = active;
      },
      onPress: () => this.openProfile()
    }) {
      Column({ space: DesignConstants.SPACING.THREE }) {
        // 在线点紧贴昵称右侧（对齐前端 nearby/people/UserCard：name+dot 同行左对齐）
        // web：名称 truncate 后**在线点紧跟其右**（不是被顶到行尾）——
        // 原 layoutWeight(1) 把点推到卡片右缘，与前端「绿点贴着昵称」的语义不符
        Row({ space: DesignConstants.SPACING.SM2 }) {
          Text(this.user.user?.nickname ?? '')
            .fontSize(DesignConstants.FONT.SIZE.SM) // 对齐前端 nearby/people 卡片名 text-sm（项目口径 12px）
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
            .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
            .fontColor(this.mediaActive ? DesignConstants.COLOR.WHITE : this.getPalette().textPrimary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
            .flexShrink(1)

          if (this.isOnline()) {
            // 前端 nearby/people：size-2(8) + ring-2；本端统一 ONLINE_DOT_MD(10，含 2vp 描边)
            Row()
              .flexShrink(0)
              .width(DesignConstants.COMPONENT.ONLINE_DOT_MD)
              .height(DesignConstants.COMPONENT.ONLINE_DOT_MD)
              .borderRadius(DesignConstants.RADIUS.FULL)
              .backgroundColor(this.getPalette().online)
              .border({
                width: DesignConstants.BORDER_WIDTH.STRONG,
                color: this.mediaActive ? DesignConstants.COLOR.MEDIA_SCRIM_STRONG : this.getPalette().surfaceElevated
              })
          }
        }
        .width('100%')
        .alignItems(VerticalAlign.Center)

        if (this.user.bio) {
          Text(this.user.bio)
            .fontSize(DesignConstants.FONT.SIZE.XS2) // 对齐前端次要行 text-xxs(10)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS2))
            .fontColor(this.mediaActive ? DesignConstants.COLOR.WHITE : this.getPalette().textSecondary)
            .opacity(this.mediaActive ? DesignConstants.OPACITY.SUBTLE : 1)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        Row() {
          if (this.getPrimaryTag().length > 0) {
            Text(`#${this.getPrimaryTag()}`)
              .fontSize(DesignConstants.FONT.SIZE.XS2)
              .fontColor(this.mediaActive ? DesignConstants.COLOR.WHITE : this.getPalette().accentBackground)
              .backgroundColor(this.mediaActive ? DesignConstants.COLOR.MEDIA_SCRIM_STRONG : this.getPalette().accentMuted)
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
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: DesignConstants.SPACING.MD,
        bottom: DesignConstants.SPACING.LG
      })
    }
  }
}
