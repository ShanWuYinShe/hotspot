/**
 * 首页 - 对齐前端 Home 页面布局
 * 粘性头部 + 地图卡片 + 附近用户列表 + 活动网格
 */

import { GlobalStateManager, KEY_IS_LOGGED_IN, KEY_BROWSE_ANCHOR, KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { LocationInfo, LocationUtil } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { AppRouter, ChatRouteParam, LoginRouteParam, MomentDetailRouteParam, ProfileRouteParam } from '../../../core/router/AppRouter';
import { PageNames, DefaultConfig } from '../../../constants/AppConstants';
import {
  MomentItem, UserItem, GroupItem, GeoPoint,
  SearchMomentsRequest, SearchUsersRequest, SearchGroupsRequest,
  JoinGroupRequest,
  BlockRequest, UnblockRequest, CreateReportRequest, Relationship
} from '../../../core/types/api';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { MomentFeedCard } from '../../../components/MomentFeedCard';
import { SystemToast } from '../../../components/SystemToast';
import { InteractionStateStore, UserRelationState } from '../../../core/state/InteractionStateStore';
import { interactionActions } from '../../../core/state/InteractionActions';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { WidgetDataService } from '../../../core/services/WidgetDataService';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { formatCount } from '../../../core/utils/FormatUtil';
import pasteboard from '@ohos.pasteboard';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';

const DOMAIN_NUMBER: number = 0xFF10;
const TAG: string = 'NearbyPage';

interface StatItem {
  label: string;
  value: string;
}

@Preview
@Component
export struct NearbyPage {
  @StorageLink(KEY_IS_LOGGED_IN) isLoggedIn: boolean = false;
  @StorageLink(KEY_CURRENT_LOCATION) @Watch('onCurrentLocationChange') currentLocation: LocationInfo | null = null;
  @StorageLink(KEY_BROWSE_ANCHOR) @Watch('onBrowseAnchorChange') browseAnchor: LocationInfo | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';
  @StorageLink('interaction_user_state') userStateMap: Record<string, UserRelationState> = {};
  @StorageProp('isTablet') isTablet: boolean = false;

  @State moments: MomentItem[] = [];
  @State users: UserItem[] = [];
  @State groups: GroupItem[] = [];

  @State momentsLoading: boolean = true;
  @State usersLoading: boolean = true;
  @State groupsLoading: boolean = true;

  @State momentsError: string = '';
  @State usersError: string = '';
  @State groupsError: string = '';

  @State userActionPendingIds: string[] = [];
  @State groupActionPendingIds: string[] = [];

  private interactionStore = InteractionStateStore.getInstance();
  private isDestroyed: boolean = false;
  private initialLoadDone: boolean = false;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  async aboutToAppear() {
    SystemUiController.syncWithCurrentTheme();
    this.loadNearbyMoments();
    this.loadNearbyUsers();
    this.loadNearbyGroups();
    this.initialLoadDone = true;
  }

  aboutToDisappear() {
    this.isDestroyed = true;
  }

  private onCurrentLocationChange(): void {
    if (!this.initialLoadDone || this.browseAnchor || !this.currentLocation) return;
    if (this.moments.length === 0 && this.users.length === 0 && this.groups.length === 0) {
      this.loadNearbyMoments();
      this.loadNearbyUsers();
      this.loadNearbyGroups();
    }
  }

  private onBrowseAnchorChange(): void {
    if (!this.initialLoadDone) return;
    this.loadNearbyMoments();
    this.loadNearbyUsers();
    this.loadNearbyGroups();
  }

  private getReferencePoint(): GeoPoint | null {
    const loc: LocationInfo | null = this.browseAnchor ?? this.currentLocation;
    if (!loc) return null;
    return { latitude: loc.latitude, longitude: loc.longitude };
  }

  private getHeaderTopPadding(): number {
    return SafeAreaUtils.resolveTopPadding(this.safeAreaTop);
  }

  // ==================== 数据加载 ====================

  private async loadNearbyMoments(): Promise<void> {
    const ref = this.getReferencePoint();
    if (!ref) { this.momentsLoading = false; return; }
    this.momentsLoading = true;
    this.momentsError = '';
    try {
      const request: SearchMomentsRequest = {
        reference_point: ref, radius: DefaultConfig.NEARBY_RADIUS,
        sort_by: 'distance', pagination: { limit: 10 }
      };
      const response = await LoadingUtil.withMinimumTime(() => ApiClient.searchMoments(request));
      if (this.isDestroyed) return;
      if (response.code === 0) {
        const items = response.data?.items ?? [];
        this.moments = items.filter((m: MomentItem) => m.moment_id && m.user);
        this.moments.forEach((item: MomentItem) => this.interactionStore.patchMomentFromItem(item));
      } else {
        this.momentsError = response.message || '加载失败';
      }
    } catch {
      this.momentsError = '加载附近动态失败';
    } finally {
      this.momentsLoading = false;
    }
  }

  private async loadNearbyUsers(): Promise<void> {
    const ref = this.getReferencePoint();
    if (!ref) { this.usersLoading = false; return; }
    this.usersLoading = true;
    this.usersError = '';
    try {
      const request: SearchUsersRequest = {
        reference_point: ref, radius: DefaultConfig.NEARBY_RADIUS,
        sort_by: 'distance', pagination: { limit: 10 }
      };
      const response = await LoadingUtil.withMinimumTime(() => ApiClient.searchUsers(request));
      if (this.isDestroyed) return;
      if (response.code === 0) {
        const items = response.data?.items ?? [];
        const me = GlobalStateManager.getInstance().getCurrentUser();
        const myId = me?.user_id;
        const valid = items.filter((u: UserItem) => u.user?.user_id);
        this.users = typeof myId === 'string'
          ? valid.filter((u: UserItem) => u.user.user_id !== myId)
          : valid;
        this.users.forEach((u: UserItem) => this.interactionStore.patchUserFromItem(u));
      } else {
        this.usersError = response.message || '加载失败';
      }
    } catch {
      this.usersError = '加载附近用户失败';
    } finally {
      this.usersLoading = false;
    }
  }

  private async loadNearbyGroups(): Promise<void> {
    const ref = this.getReferencePoint();
    if (!ref) { this.groupsLoading = false; return; }
    this.groupsLoading = true;
    this.groupsError = '';
    try {
      const request: SearchGroupsRequest = {
        reference_point: ref, radius: DefaultConfig.NEARBY_RADIUS,
        sort_by: 'distance', pagination: { limit: 10 }
      };
      const response = await LoadingUtil.withMinimumTime(() => ApiClient.searchGroups(request));
      if (this.isDestroyed) return;
      if (response.code === 0) {
        const items = response.data?.items ?? [];
        this.groups = items.filter((g: GroupItem) => g.group?.group_id);
      } else {
        this.groupsError = response.message || '加载失败';
      }
    } catch {
      this.groupsError = '加载附近群组失败';
    } finally {
      this.groupsLoading = false;
    }
  }

  // ==================== 操作方法 ====================

  private isUserActionPending(userId: string): boolean {
    return this.userActionPendingIds.indexOf(userId) >= 0;
  }

  private setUserActionPending(userId: string, pending: boolean): void {
    const next: string[] = [];
    for (let i = 0; i < this.userActionPendingIds.length; i++) {
      if (this.userActionPendingIds[i] !== userId) next.push(this.userActionPendingIds[i]);
    }
    if (pending) next.push(userId);
    this.userActionPendingIds = next;
  }

  private isUserFollowing(userId: string): boolean {
    const state = this.userStateMap[userId];
    return state?.relation?.is_following ?? false;
  }

  private ensureUserRelation(user: UserItem): Relationship {
    if (!user.relation) {
      user.relation = { is_following: false, is_followed_by: false, is_blocked: false };
    }
    return user.relation;
  }

  private async followUser(user: UserItem): Promise<void> {
    if (!user.user?.user_id) return;
    const userId = user.user.user_id;
    if (this.isUserActionPending(userId)) return;
    this.setUserActionPending(userId, true);
    try {
      const isFollowing = this.isUserFollowing(userId);
      const success = await interactionActions.setFollowUser(userId, !isFollowing, '关注用户');
      if (success) {
        this.patchUserFollowing(userId, !isFollowing);
        SystemToast.showSuccess(isFollowing ? '已取消关注' : '已关注', this.getUIContext());
      } else {
        SystemToast.showError('操作失败', this.getUIContext());
      }
    } finally {
      this.setUserActionPending(userId, false);
    }
  }

  private patchUserFollowing(userId: string, isFollowing: boolean): void {
    const users = this.users;
    for (let i = 0; i < users.length; i++) {
      if ((users[i].user?.user_id ?? '') !== userId) continue;
      this.ensureUserRelation(users[i]).is_following = isFollowing;
      break;
    }
    this.users = users;
  }

  private async joinGroup(group: GroupItem): Promise<void> {
    if (!this.isLoggedIn) {
      SystemToast.showWarning('请先登录后再加入群组', this.getUIContext());
      return;
    }
    const groupId = group.group?.group_id ?? '';
    if (!groupId || group.my_state?.is_joined) {
      if (group.my_state?.is_joined) this.enterGroupChat(group);
      return;
    }
    try {
      const joinReq: JoinGroupRequest = { group_id: groupId };
      const response = await ApiClient.joinGroup(joinReq);
      if (response.code === 0 && response.data) {
        if (response.data.status === 'joined') {
          InteractionStateStore.getInstance().updateGroupMembership(groupId, { is_member: true });
          SystemToast.showSuccess('加入成功', this.getUIContext());
          this.loadNearbyGroups();
        } else if (response.data.status === 'pending_approval') {
          SystemToast.show('已提交申请，等待审批', this.getUIContext());
        }
      } else {
        SystemToast.showError(response.message || '加入失败', this.getUIContext());
      }
    } catch {
      SystemToast.showError('加入群组失败', this.getUIContext());
    }
  }

  private openUserProfile(user: UserItem): void {
    const userId = user.user?.user_id;
    if (!userId) return;
    const params = new ProfileRouteParam();
    params.userId = userId;
    AppRouter.push(PageNames.PROFILE, params);
  }

  private openMomentDetail(moment: MomentItem): void {
    const params = new MomentDetailRouteParam();
    params.momentId = moment.moment_id;
    AppRouter.push(PageNames.MOMENT_DETAIL, params);
  }

  private enterGroupChat(group: GroupItem): void {
    if (!this.isLoggedIn) return;
    const groupId = group.group?.group_id;
    if (!groupId) return;
    const params = new ChatRouteParam();
    params.chatId = groupId;
    params.chatType = 'group_chat';
    params.title = group.group?.name ?? '群聊';
    AppRouter.push(PageNames.CHAT, params);
  }

  private async copyText(text: string, successText: string): Promise<void> {
    if (!text) return;
    try {
      const systemPasteboard = pasteboard.getSystemPasteboard();
      const pasteData = pasteboard.createData(pasteboard.MIMETYPE_TEXT_PLAIN, text);
      await systemPasteboard.setData(pasteData);
      SystemToast.show(successText, this.getUIContext());
    } catch {
      SystemToast.showError('复制失败', this.getUIContext());
    }
  }

  // ==================== UI ====================

  build() {
    Column() {
      // 粘性头部
      this.buildStickyHeader()

      // 内容区
      Scroll() {
        Column() {
          this.buildMapCard()
          this.buildNearbyPeople()
          this.buildActivityGrid()
          this.buildMomentsSection()
        }
        .width('100%')
        .padding({ top: 16, bottom: 24 })
      }
      .layoutWeight(1)
      .scrollBar(BarState.Auto)
      .align(Alignment.TopStart)
    }
    .width('100%')
    .height('100%')
    .constraintSize({ maxWidth: this.isTablet ? 720 : Infinity })
    .backgroundColor(this.getPalette().pageBackground)
  }

  // ==================== 粘性头部 ====================

  @Builder
  private buildStickyHeader() {
    Row() {
      Column({ space: 2 }) {
        Text('首页')
          .fontSize(22)
          .fontWeight(FontWeight.Normal)
          .fontColor(this.getPalette().textPrimary)

        Text('发现周边的有趣人和事')
          .fontSize(12)
          .fontColor(this.getPalette().textSecondary)
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)

      Row({ space: 6 }) {
        Text() {
          SymbolSpan($r(AppIcon.LOCATION))
            .fontSize(12)
            .fontColor([this.getPalette().accentBackground])
        }

        Text(this.browseAnchor?.locationName ?? this.currentLocation?.locationName ?? '定位中...')
          .fontSize(11)
          .fontColor(this.getPalette().textSecondary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
      }
      .padding({ left: 12, right: 14, top: 6, bottom: 6 })
      .borderRadius(DesignConstants.RADIUS.FULL)
      .border({ width: 1, color: this.getPalette().borderColor })
      .backgroundColor(this.getPalette().surfaceBackground)
    }
    .width('100%')
    .padding({ left: 16, right: 16, top: this.getHeaderTopPadding(), bottom: 12 })
    .backgroundColor(this.getPalette().pageBackground)
    .border({ width: { bottom: 1 }, color: this.getPalette().borderColor })
  }

  // ==================== 地图卡片 ====================

  @Builder
  private buildMapCard() {
    Stack({ alignContent: Alignment.Center }) {
      // 底层：135° 品牌微染 → 米白 对角渐变（对齐前端 surface-secondary → surface-bg）
      Column()
        .width('100%')
        .height('100%')
        .linearGradient({
          angle: 135,
          colors: [
            [this.getPalette().heroGlow, 0.0],
            [this.getPalette().surfaceBackground, 1.0]
          ]
        })

      // 顶层：底部轻微加深，提升浮层对比
      Column()
        .width('100%')
        .height('100%')
        .linearGradient({
          direction: GradientDirection.Bottom,
          colors: [
            ['rgba(0,0,0,0.00)', 0.45],
            ['rgba(0,0,0,0.14)', 1.0]
          ]
        })

      // 中心脉冲定位点（44 圈包 14 点）
      Stack() {
        Row()
          .width(14)
          .height(14)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .backgroundColor(this.getPalette().accentBackground)
          .shadow(this.getPalette().accentShadow)
      }
      .width(44)
      .height(44)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(this.getPalette().surfaceBackground)
      .border({ width: 1, color: this.getPalette().floatingBorderColor })

      // 右下控制按钮（定位 + 地图）：嵌套全屏 Stack 贴右下（ArkUI 不支持 calc，用 alignContent 对齐前端 bottom-3 right-3）
      Stack({ alignContent: Alignment.BottomEnd }) {
        Column({ space: DesignConstants.SPACING.SM }) {
          this.buildMapIconButton(AppIcon.LOCATION)
          this.buildMapIconButton(AppIcon.MAP)
        }
      }
      .width('100%')
      .height('100%')
      .padding({ right: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })

      // 左下人数统计：嵌套全屏 Stack 贴左下（对齐前端 bottom-3 left-3）
      Stack({ alignContent: Alignment.BottomStart }) {
        Row({ space: DesignConstants.SPACING.XS }) {
          Text(`${this.users.length}`)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getPalette().accentBackground)
            .fontWeight(FontWeight.Bold)
          Text('人附近')
            .fontSize(10)
            .fontColor(this.getPalette().textSecondary)
        }
        .padding({ left: 10, right: 10, top: 4, bottom: 4 })
        .borderRadius(DesignConstants.RADIUS.FULL)
        .backgroundColor(this.getPalette().surfaceBackground)
        .border({ width: 1, color: this.getPalette().floatingBorderColor })
      }
      .width('100%')
      .height('100%')
      .padding({ left: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
    }
    .width('100%')
    .aspectRatio(2.4)
    .borderRadius(DesignConstants.SHAPE.HERO_CARD)
    .clip(true)
    .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: 20 })
    .border({ width: 1, color: this.getPalette().borderColor })
  }

  @Builder
  private buildMapIconButton(icon: AppIcon) {
    Row() {
      Text() {
        SymbolSpan($r(icon))
          .fontSize(14)
          .fontColor([this.getPalette().textPrimary])
      }
    }
    .width(34)
    .height(34)
    .justifyContent(FlexAlign.Center)
    .borderRadius(DesignConstants.SHAPE.ICON_TILE)
    .backgroundColor(this.getPalette().surfaceBackground)
    .border({ width: 1, color: this.getPalette().floatingBorderColor })
  }

  // ==================== 区块标题 (eyebrow) ====================

  @Builder
  private buildSectionHeader(title: string, linkText: string, onViewAll: () => void) {
    Row() {
      Text(title)
        .fontSize(10)
        .fontColor(this.getPalette().textSecondary)
        .textCase(TextCase.UpperCase)
        .letterSpacing(1.4)
        .opacity(0.75)

      Blank()

      Row({ space: 2 }) {
        Text(linkText)
          .fontSize(12)
          .fontColor(this.getPalette().accentBackground)
        Text() {
          SymbolSpan($r(AppIcon.ARROW_RIGHT))
            .fontSize(13)
            .fontColor([this.getPalette().accentBackground])
        }
      }
      .onClick(onViewAll)
    }
    .width('100%')
    .margin({ bottom: 12 })
  }

  // ==================== 附近的人 ====================

  @Builder
  private buildNearbyPeople() {
    Column() {
      this.buildSectionHeader('附近的人', '查看全部', () => {
        // navigate to nearby users
      })

      if (!this.usersLoading && this.users.length === 0) {
        Text('附近暂无其他用户，可稍后再来或调整浏览位置。')
          .fontSize(12)
          .fontColor(this.getPalette().textSecondary)
          .width('100%')
          .margin({ top: 2 })
      } else {
        Scroll() {
          Row({ space: 12 }) {
            if (this.usersLoading) {
              ForEach(['1', '2', '3', '4', '5'], (_: string) => {
              Column({ space: 6 }) {
                Row()
                  .width(66)
                  .height(66)
                  .borderRadius(DesignConstants.RADIUS.FULL)
                  .backgroundColor(this.getPalette().surfaceSubtle)
              }
              .alignItems(HorizontalAlign.Center)
            }, (_: string) => _)
          } else {
            ForEach(this.users.slice(0, 8), (user: UserItem) => {
              Column({ space: 6 }) {
                Stack({ alignContent: Alignment.Center }) {
                  SystemAvatar({
                    avatarUrl: user.user?.avatar_url ?? '',
                    name: user.user?.nickname ?? '',
                    avatarSize: 80,
                    shape: 'circle'
                  })
                  .border({ width: 2, color: this.getPalette().borderColor })
                  .borderRadius(DesignConstants.RADIUS.FULL)

                  // 在线状态点：贴头像右下角（用 alignContent 取代硬编码坐标）
                  if (user.presence_status === 'online') {
                    Stack({ alignContent: Alignment.BottomEnd }) {
                      Row()
                        .width(12)
                        .height(12)
                        .borderRadius(DesignConstants.RADIUS.FULL)
                        .backgroundColor(DesignConstants.COLOR.SUCCESS)
                        .border({ width: 2, color: this.getPalette().surfaceBackground })
                    }
                    .width('100%')
                    .height('100%')
                  }

                  // 距离徽标：贴头像底部居中（取代百分比 + translate 硬定位）
                  if (user.location?.distance !== undefined && user.location?.distance !== null) {
                    Stack({ alignContent: Alignment.Bottom }) {
                      Row() {
                        Text(LocationUtil.formatDistance(user.location.distance))
                          .fontSize(9)
                          .fontColor(this.getPalette().accentText)
                      }
                      .padding({ left: 6, right: 6, top: 1, bottom: 1 })
                      .borderRadius(DesignConstants.RADIUS.FULL)
                      .backgroundColor(this.getPalette().accentBackground)
                      .shadow(this.getPalette().accentShadow)
                    }
                    .width('100%')
                    .height('100%')
                  }
                }

                Text(user.user?.nickname ?? '')
                  .fontSize(12)
                  .fontColor(this.getPalette().textPrimary)
                  .maxLines(1)
                  .textOverflow({ overflow: TextOverflow.Ellipsis })
                  .constraintSize({ maxWidth: 70 })
                  .margin({ top: 4 })
              }
              .alignItems(HorizontalAlign.Center)
              .onClick(() => {
                const userId = user.user?.user_id;
                if (userId) {
                  const params = new ProfileRouteParam();
                  params.userId = userId;
                  AppRouter.push(PageNames.PROFILE, params);
                }
              })
            }, (user: UserItem) => user.user?.user_id ?? '')
          }
        }
      }
      .scrollable(ScrollDirection.Horizontal)
      .scrollBar(BarState.Off)
      .width('100%')
      }
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
    .margin({ bottom: 20 })
  }

  // ==================== 附近活动 ====================

  @Builder
  private buildActivityGrid() {
    if (this.groupsLoading || this.groups.length > 0) {
      Column() {
        this.buildSectionHeader('附近活动', '查看全部', () => {
        })

        Column({ space: DesignConstants.SPACING.MD }) {
          if (this.groupsLoading && this.groups.length === 0) {
            Row({ space: DesignConstants.SPACING.MD }) {
              Column()
                .layoutWeight(1)
                .height(120)
                .borderRadius(DesignConstants.SHAPE.CARD)
                .backgroundColor(this.getPalette().surfaceSubtle)
              Column()
                .layoutWeight(1)
                .height(120)
                .borderRadius(DesignConstants.SHAPE.CARD)
                .backgroundColor(this.getPalette().surfaceSubtle)
            }
            .width('100%')
          } else {
            this.buildActivityRow(this.groups.slice(0, 2))
            if (this.groups.length > 2) {
              this.buildActivityRow(this.groups.slice(2, 4))
            }
          }
        }
        .width('100%')
      }
      .width('100%')
      .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
      .margin({ bottom: 20 })
    }
  }

  // 两列等宽卡片行（前端 grid-cols-2 的 ArkUI 等价：Row + layoutWeight，避开 ArkUI 不支持的 calc）
  @Builder
  private buildActivityRow(items: GroupItem[]) {
    Row({ space: DesignConstants.SPACING.MD }) {
      ForEach(items, (group: GroupItem) => {
        Column() {
          this.buildActivityCard(group)
        }
        .layoutWeight(1)
      }, (group: GroupItem) => group.group?.group_id ?? '')
      if (items.length === 1) {
        Column().layoutWeight(1)
      }
    }
    .width('100%')
  }

  @Builder
  private buildActivityCard(group: GroupItem) {
    Column() {
      Row() {
        Text() {
          SymbolSpan($r(AppIcon.GROUP))
            .fontSize(28)
            .fontColor([this.getPalette().accentBackground])
        }
      }
      .width('100%')
      .height(64)
      .justifyContent(FlexAlign.Center)
      .backgroundColor(this.getPalette().accentMuted)
      .borderRadius({ topLeft: DesignConstants.SHAPE.CARD, topRight: DesignConstants.SHAPE.CARD })

      Column({ space: 4 }) {
        Text(group.group?.name ?? '')
          .fontSize(13)
          .fontWeight(FontWeight.Medium)
          .fontColor(this.getPalette().textPrimary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        Row({ space: 6 }) {
          Text(`${formatCount(group.group?.members_count)} 人`)
            .fontSize(11)
            .fontColor(this.getPalette().textTertiary)
          if (group.location?.distance !== undefined) {
            Text(LocationUtil.formatDistance(group.location.distance))
              .fontSize(11)
              .fontColor(this.getPalette().accentBackground)
          }
        }
      }
      .padding({ left: 12, right: 12, top: 12, bottom: 12 })
      .alignItems(HorizontalAlign.Start)
    }
    .width('100%')
    .borderRadius(DesignConstants.SHAPE.CARD)
    .backgroundColor(this.getPalette().surfaceBackground)
    .border({ width: 1, color: this.getPalette().borderColor })
    .shadow(this.getPalette().elevatedShadow)
    .onClick(() => {
      if (group.group?.group_id) {
        const params = new GroupDetailParams();
        params.groupId = group.group.group_id;
        AppRouter.push(PageNames.GROUP_DETAIL, params);
      }
    })
  }

  // ==================== 附近动态 ====================

  @Builder
  private buildMomentsSection() {
    if (this.momentsLoading || this.moments.length > 0) {
      Column() {
        this.buildSectionHeader('附近动态', '查看全部', () => {
        })

        if (this.momentsLoading && this.moments.length === 0) {
          Column({ space: 12 }) {
            ForEach(['1', '2'], (_: string) => {
              Column()
                .width('100%')
                .height(96)
                .borderRadius(DesignConstants.SHAPE.CARD)
                .backgroundColor(this.getPalette().surfaceSubtle)
            }, (_: string) => _)
          }
          .width('100%')
        } else {
          Column({ space: 16 }) {
            ForEach(this.moments.slice(0, 5), (moment: MomentItem) => {
              MomentFeedCard({
                moment: moment,
                onOpenDetail: (item: MomentItem) => this.openMomentDetail(item)
              })
            }, (moment: MomentItem) => moment.moment_id)
          }
          .width('100%')
        }
      }
      .width('100%')
      .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
      .margin({ bottom: 24 })
    }
  }
}

class GroupDetailParams {
  groupId: string = '';
}
