import { UserItem } from '../../../core/types/api';
import { interactionActions } from '../../../core/state/InteractionActions';
import { UserRelationState, interactionEventHub} from '../../../core/state/InteractionStateStore'
import { LocationUtil } from '../../../core/location/LocationUtil';
import { AppRouter, ProfileRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { NearbyImageCard } from '../../../components/NearbyImageCard';
import { Pressable } from '../../../components/Pressable';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { LucideIcon } from '../../../components/LucideIcon';

/**
 * 附近用户卡 — 双形态（对齐前端两处消费场景）：
 * - portrait（默认）：3:4 图片卡，头像即卡图，对齐前端 nearby/people/UserCard——
 *   右上距离毛玻璃徽章；底部托盘 = 昵称 + bio 副行 + 在线点、#tag 徽章 + 28vp 圆形关注钮。
 * - compact：竖排白卡，对齐前端 nearby/+page「身边的新朋友」横滑卡——
 *   avatar 48 + 名字 + 距离 + #tag 胶囊 + 全宽关注钮（rounded-md 16）。
 * 关注态优先读 InteractionStateStore 乐观更新，回退响应 relation。
 */
@Reusable
@Component
export struct UserCard {
  @Prop user: UserItem;
  @Prop showDistance: boolean = true;
  /** 卡形态：portrait=3:4 图片卡（people 专区）；compact=竖排白卡（附近首页横滑） */
  @Prop mode: string = 'portrait';
  // 仅持本卡关系态：整 Map 订阅会让任意用户关注动作触发全部卡重算。
  // 事件总线按 user_id 过滤定向更新（快照冷启动播种见 aboutToAppear）。
  @State private socialState: UserRelationState | undefined = undefined;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  @State private followInFlight: boolean = false;

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

  private handleUserStateUpdate(payload: Object): void {
    const state = payload as UserRelationState;
    if (state && state.user_id === this.getUserId()) {
      this.socialState = state;
    }
  }

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
      pressScale: DesignConstants.INTERACTION.SCALE_PRESSED_ICON
    }) {
      Row() {
        LucideIcon({
          icon: this.isFollowing() ? AppIcon.CHECK : AppIcon.ADD,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
          // Tonal 化（Gemini 审查：实体蓝泛滥，关注为列表次级动作）：
          // 未关注 = accentMuted 底 + accentText 图标，实体蓝让位全局主流程
          color: this.getPalette().accentText
        })
      }
      .id(`${NearbyImageCard.CHILD_CONTROL_PREFIX}follow-${this.getUserId()}`)
      .width(DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT)
      .height(DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .justifyContent(FlexAlign.Center)
      .backgroundColor(this.getPalette().accentMuted)
      .border(this.isFollowing() ?
        { width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().borderColor } :
        { width: 0 })
      .accessibilityText(this.isFollowing() ?
        `取消关注 ${this.user.user?.nickname ?? ''}` :
        `关注 ${this.user.user?.nickname ?? ''}`)
      // 触控热区：视觉 28vp，responseRegion 外扩到 44vp（不占布局）
      .responseRegion({
        x: -(DesignConstants.LAYOUT.MIN_HIT_TARGET - DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT) / 2,
        y: -(DesignConstants.LAYOUT.MIN_HIT_TARGET - DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT) / 2,
        width: DesignConstants.LAYOUT.MIN_HIT_TARGET,
        height: DesignConstants.LAYOUT.MIN_HIT_TARGET
      })
    }
  }

  /** compact 形态全宽关注钮（对齐前端 nearby/+page w-full rounded-md py-1 text-xxs） */
  @Builder
  private buildCompactFollowButton() {
    // 按压反馈统一走 Pressable（P3-1，springMotion 手感）
    Pressable({
      onPress: () => this.toggleFollow(),
      pressScale: DesignConstants.INTERACTION.SCALE_PRESSED_PILL
    }) {
      Row({ space: DesignConstants.SPACING.HALF }) {
        if (this.isFollowing()) {
          LucideIcon({
            icon: AppIcon.CHECK,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
            paletteRole: 'textSecondary'
          })
          Text('已关注')
            .fontSize(DesignConstants.FONT.SIZE.XS2)
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.getPalette().textSecondary)
        } else {
          Text('关注')
            .fontSize(DesignConstants.FONT.SIZE.XS2)
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.getPalette().accentText)
        }
      }
      .width('100%')
      .height(DesignConstants.COMPONENT.FILTER_CHIP_HEIGHT)
      .justifyContent(FlexAlign.Center)
      .borderRadius(DesignConstants.SHAPE.INPUT)
      // Tonal 化（Gemini 审查：同圆形钮，未关注 accentMuted 底 + accentText 字）
      .backgroundColor(this.isFollowing() ? this.getPalette().surfaceSubtle : this.getPalette().accentMuted)
      .border(this.isFollowing() ?
        { width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().borderColor } :
        { width: 0 })
      .accessibilityText(this.isFollowing() ?
        `取消关注 ${this.user.user?.nickname ?? ''}` :
        `关注 ${this.user.user?.nickname ?? ''}`)
    }
  }

  /** compact 竖排白卡（对齐前端 nearby/+page 身边的新朋友卡：p-3 avatar48 名字/距离/tag 竖排） */
  @Builder
  private buildCompactCard() {
    Column({ space: DesignConstants.SPACING.SM }) {
      Stack({ alignContent: Alignment.BottomEnd }) {
        SystemAvatar({
          avatarUrl: this.user.user?.avatar_url ?? '',
          name: this.user.user?.nickname ?? '',
          avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.MEDIUM, // 48 对齐前端 size-12
          shape: 'circle'
        })
        if (this.isOnline()) {
          Row()
            .width(DesignConstants.SPACING.MD)
            .height(DesignConstants.SPACING.MD)
            .borderRadius(DesignConstants.RADIUS.FULL)
            .backgroundColor(this.getPalette().online)
            .border({ width: DesignConstants.BORDER_WIDTH.STRONG, color: this.getPalette().surfaceBackground })
        }
      }

      Text(this.user.user?.nickname ?? '')
        .fontSize(DesignConstants.FONT.SIZE.XS) // 11 对齐前端 text-xs font-semibold
        .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
        .fontColor(this.getPalette().textPrimary)
        .maxLines(1)
        .textOverflow({ overflow: TextOverflow.Ellipsis })

      // 距离行（对齐前端 MapPin size-2.5 + text-xxs）
      if (this.showDistance && this.getDistanceText().length > 0) {
        Row({ space: DesignConstants.SPACING.HALF }) {
          LucideIcon({
            icon: AppIcon.LOCATION,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
            paletteRole: 'accent'
          })
          Text(this.getDistanceText())
            .fontSize(DesignConstants.FONT.SIZE.XS2)
            .fontColor(this.getPalette().textSecondary)
        }
        .alignItems(VerticalAlign.Center)
      }

      if (this.getPrimaryTag().length > 0) {
        Text(`#${this.getPrimaryTag()}`)
          .fontSize(DesignConstants.FONT.SIZE.XS2)
          .fontColor(this.getPalette().accentBackground)
          .backgroundColor(this.getPalette().accentMuted)
          .borderRadius(DesignConstants.RADIUS.SM)
          .padding({ left: DesignConstants.SPACING.SM2, right: DesignConstants.SPACING.SM2, top: 1, bottom: 1 })
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
      }

      this.buildCompactFollowButton()
    }
    .width('100%')
    .alignItems(HorizontalAlign.Center)
    // 卡本体点击进主页（对齐前端整卡 button → /users/[id]；关注钮 stopPropagation 语义
    // 由 ArkUI 子组件 onClick 不冒泡保证）
    .onClick(() => this.openProfile())
    .padding(DesignConstants.SPACING.MD) // p-3(12) 对齐前端
    .backgroundColor(this.getPalette().surfaceBackground)
    .borderRadius(DesignConstants.SHAPE.HERO_CARD) // 24 对齐前端 rounded-xl
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().borderColor })
    .shadow(this.getPalette().surfaceShadow)
  }

  build() {
    if (this.mode === 'compact') {
      this.buildCompactCard()
    } else {
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
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
                .fontColor(this.getPalette().textPrimary)
                .maxLines(1)
                .textOverflow({ overflow: TextOverflow.Ellipsis })

              if (this.user.bio) {
                Text(this.user.bio)
                  .fontSize(DesignConstants.FONT.SIZE.XS2)
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
                .fontColor(this.getPalette().accentBackground)
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
}
