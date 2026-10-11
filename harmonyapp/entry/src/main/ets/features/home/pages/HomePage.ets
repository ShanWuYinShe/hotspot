/**
 * 首页 - 对齐前端 Home 页面布局
 * 粘性头部 + 地图卡片 + 附近用户列表 + 活动网格
 */

import { GlobalStateManager, KEY_IS_LOGGED_IN, KEY_BROWSE_ANCHOR, KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { useMasterDetail } from '../../../core/responsive/Breakpoint';
import { LocationInfo, LocationUtil } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { AppRouter, ChatRouteParam, GroupDetailRouteParam, LoginRouteParam, MomentDetailRouteParam, ProfileRouteParam, SearchRouteParam } from '../../../core/router/AppRouter';
import { PageNames, DefaultConfig } from '../../../constants/AppConstants';
import { Pressable } from '../../../components/Pressable';
import { NearbyImageCard } from '../../../components/NearbyImageCard';
import {
  MomentItem, UserItem, GroupItem, GeoPoint,
  SearchMomentsRequest, SearchUsersRequest, SearchGroupsRequest,
  EnterGroupRequest,
  BlockRequest, CreateReportRequest, Relationship
} from '../../../core/types/api';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemToast } from '../../../components/SystemToast';
import { LoginGate } from '../../../core/state/LoginGate';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SystemButton } from '../../../components/SystemButton';
import { SurfaceIconButton } from '../../../components/SurfaceIconButton';
import { InteractionStateStore, UserRelationState } from '../../../core/state/InteractionStateStore';
import { interactionActions } from '../../../core/state/InteractionActions';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LucideIcon } from '../../../components/LucideIcon';
import { EmptyState, RefreshIndicator } from '../../../components/EmptyState';
import { Skeleton } from '../../../components/states/Skeleton';
import { SkeletonShimmer } from '../../../components/states/SkeletonShimmer';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { WidgetDataService } from '../../../core/services/WidgetDataService';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { formatCount } from '../../../core/utils/FormatUtil';
import pasteboard from '@ohos.pasteboard';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { AppThemeVisuals } from '../../../core/theme/AppThemeVisuals';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';
import { curves } from '@kit.ArkUI';

const DOMAIN_NUMBER: number = 0xFF10;
const TAG: string = 'NearbyPage';

interface StatItem {
  label: string;
  value: string;
}

@Preview
@Component
export struct HomePage {
  @StorageLink(KEY_IS_LOGGED_IN) isLoggedIn: boolean = false;
  @StorageLink(KEY_CURRENT_LOCATION) @Watch('onCurrentLocationChange') currentLocation: LocationInfo | null = null;
  @StorageLink(KEY_BROWSE_ANCHOR) @Watch('onBrowseAnchorChange') browseAnchor: LocationInfo | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @StorageLink('interaction_user_state') userStateMap: Record<string, UserRelationState> = {};
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';

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
  @State viewAllPressedIndex: number = -1;
  @State locationCardPressed: boolean = false;
  @State pressedUserKey: string = '';
  @State pressedMomentKey: string = '';
  @State isRefreshing: boolean = false;

  private interactionStore = InteractionStateStore.getInstance();
  private isDestroyed: boolean = false;
  private initialLoadDone: boolean = false;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getOnlineCount(): number {
    if (!this.users || this.users.length === 0) return 0;
    return this.users.filter((u: UserItem) => u.presence_status === 'online').length;
  }

  // HeroStats 副标题（对齐前端 rangeLabel：移动端拼入定位名）
  private getHeroRangeLabel(): string {
    const locationName = this.browseAnchor?.locationName ?? this.currentLocation?.locationName ?? '';
    return locationName ? `${locationName} · 5km 内 · 持续更新` : '5km 内 · 持续更新';
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

  // ==================== 数据加载 ====================

  private async loadNearbyMoments(): Promise<void> {
    this.momentsLoading = true;
    this.momentsError = '';
    try {
      // 对齐前端 fetchHomeFeed：首页「附近活动」数据源为热门瞬间流
      // searchMoments({ sort_by: 'popular', limit: 12 })，不按位置过滤
      const request: SearchMomentsRequest = {
        sort_by: 'popular',
        pagination: { limit: 12 }
      };
      const response = await LoadingUtil.withMinimumTime(() => ApiClient.searchMoments(request));
      if (this.isDestroyed) return;
      if (response.code === 0) {
        const items = response.data?.items ?? [];
        this.moments = items.filter((m: MomentItem) => m.moment_id && m.user);
        this.moments.forEach((item: MomentItem) => this.interactionStore.patchMomentFromItem(item));
      } else {
        this.momentsError = getErrorMessage(response, '加载失败');
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
        sort_by: 'distance', pagination: { limit: 12 }
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
        this.usersError = getErrorMessage(response, '加载失败');
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
        sort_by: 'distance', pagination: { limit: 12 }
      };
      const response = await LoadingUtil.withMinimumTime(() => ApiClient.searchGroups(request));
      if (this.isDestroyed) return;
      if (response.code === 0) {
        const items = response.data?.items ?? [];
        this.groups = items.filter((g: GroupItem) => g.group?.group_id);
      } else {
        this.groupsError = getErrorMessage(response, '加载失败');
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
    // 未登录：引导登录（登录后可回跳补做），避免直接调 API 得到「操作失败」
    if (!LoginGate.requireAction('关注用户', this.getUIContext(), {
      restoreAction: 'follow_user',
      restoreTargetId: user.user.user_id,
      restoreTitle: user.user.nickname ?? ''
    })) {
      return;
    }
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
    // 未登录：Toast + 直达登录页（带补做意图），不再只弹 Toast 让用户自己找入口
    if (!LoginGate.requireAction('加入群组', this.getUIContext())) {
      return;
    }
    const groupId = group.group?.group_id ?? '';
    if (!groupId || group.viewer_state != null) {
      if (group.viewer_state != null) this.enterGroupChat(group);
      return;
    }
    // in-flight 去重：连点重复提交加入请求
    if (this.groupActionPendingIds.indexOf(groupId) >= 0) {
      return;
    }
    this.groupActionPendingIds = [...this.groupActionPendingIds, groupId];
    try {
      const joinReq: EnterGroupRequest = { group_id: groupId };
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
        SystemToast.showError(getErrorMessage(response, '加入失败'), this.getUIContext());
      }
    } catch {
      SystemToast.showError('加入群组失败', this.getUIContext());
    } finally {
      this.groupActionPendingIds = this.groupActionPendingIds.filter((id: string) => id !== groupId);
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
  // 横向留白由容器 gutter 统一提供（不再 noHorizontalPadding + 页面自加 LG2）：
  // 首页此前用 20、列表页用 16、个人页用 32，同机滑动时各页左边缘不一致
  PageShellContainer({ variant: 'default', currentBreakpoint: this.currentBreakpoint }) {
    Column() {
      // 粘性头部
      this.buildStickyHeader()

      // 整页下拉刷新（P2-2）：Refresh 包内容区（sticky header 保持粘性在外）
      Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
        Column() {
      if (this.getReferencePoint() === null) {
        // 未设置浏览位置：聚合页无数据源，引导先去设置位置（避免误导性「附近没有内容」）
        EmptyState({
          icon: AppIcon.LOCATION,
          title: '设置你的浏览位置',
          message: '选择一个位置后，可查看附近的实时动态、人与群组。',
          fillMode: true,
          actionLabel: '设置浏览位置',
          onAction: () => AppRouter.push(PageNames.LOCATION_PICKER)
        })
      } else if (this.isHomePageAllFailed()) {
        // 整页兜底：三个接口都失败（对齐前端 LoadError 契约，提供统一重试；
        // P1-2 错误态收敛 EmptyState）
        EmptyState({
          icon: AppIcon.WARNING,
          title: '首页数据加载失败',
          message: '网络似乎不太顺畅，请稍后重试。',
          fillMode: true,
          actionLabel: '重试',
          onAction: () => this.reloadHome()
        })
      } else if (this.isHomePageAllEmpty()) {
        // 整页空态：附近没有任何内容（对齐前端 EmptyState「附近还没有内容」）
        EmptyState({
          icon: AppIcon.NEARBY,
          title: '附近还没有内容',
          message: '去发布第一条动态，或到附近的人认识新朋友。',
          fillMode: true,
          actionLabel: '去附近看看',
          onAction: () => AppRouter.push(PageNames.NEARBY)
        })
      } else {
        this.buildHomeBody()
      }
        }
        .width('100%')
        .height('100%')
      }
      .width('100%')
      .layoutWeight(1)
      .onRefreshing(() => { this.refreshHome(); })
    }
    .width('100%')
    .height('100%')
  }
  }

  // ==================== 整页状态判定 ====================

  private isInitialLoading(): boolean {
    return this.momentsLoading && this.usersLoading && this.groupsLoading;
  }

  private isHomePageAllFailed(): boolean {
    return !this.isInitialLoading()
      && this.momentsError.length > 0 && this.usersError.length > 0 && this.groupsError.length > 0;
  }

  private isHomePageAllEmpty(): boolean {
    return !this.isInitialLoading()
      && this.momentsError.length === 0 && this.usersError.length === 0 && this.groupsError.length === 0
      && this.moments.length === 0 && this.users.length === 0 && this.groups.length === 0;
  }

  private reloadHome(): void {
    this.loadNearbyMoments();
    this.loadNearbyUsers();
    this.loadNearbyGroups();
  }

  /** 下拉刷新（P2-2）：三个分区并行重载，全部落地后收起刷新指示器 */
  private async refreshHome(): Promise<void> {
    try {
      await Promise.all([this.loadNearbyMoments(), this.loadNearbyUsers(), this.loadNearbyGroups()]);
    } finally {
      this.isRefreshing = false;
    }
  }

  @Builder
  private buildHomeBody() {
    if (useMasterDetail(this.currentBreakpoint)) {
      // ======== 宽屏（master-detail）：双栏工作区（对齐前端 desktop grid）========
      // 左栏：HeroStats 统计 + 附近活动 + 附近群组横滑
      Row() {
        // 左栏
        Scroll() {
          Column({ space: DesignConstants.SPACING.LG }) {
            this.buildHeroStats()
            this.buildMomentsSection()
            this.buildGroupCarousel()
          }
          .width('100%')
          .padding({ top: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.XL })
        }
        .layoutWeight(1)
        .scrollBar(BarState.Off)
        .align(Alignment.TopStart)

        // 分割线
        Column()
          .width(DesignConstants.BORDER_WIDTH.HAIRLINE)
          .height('100%')
          .backgroundColor(this.getPalette().borderColor)

        // 右栏（固定宽度，对齐前端右栏语义：完整人物列表）
        Scroll() {
          Column({ space: DesignConstants.SPACING.LG }) {
            this.buildNearbyPeople()
          }
          .width('100%')
          .padding({ top: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.XL })
        }
        .width(DesignConstants.LAYOUT.HOME_COLUMN_WIDTH)
        .scrollBar(BarState.Off)
        .align(Alignment.TopStart)
      }
      .layoutWeight(1)
    } else {
      // ======== 窄屏（<960）：单栏纵向堆叠 ========
      // 区块顺序对齐前端 homeContent：HeroStats → 附近活动 → 群组 → 人物推荐
      Scroll() {
        Column() {
          this.buildHeroStats()
          this.buildMomentsSection()
          this.buildGroupCarousel()
          this.buildNearbyPeople()
        }
        .width('100%')
        .padding({ top: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.XL })
      }
      .layoutWeight(1)
      .scrollBar(BarState.Off)
      .align(Alignment.TopStart)
    }
  }

  // ==================== 粘性头部 ====================

  @Builder
  private buildStickyHeader() {
    // 统一走 SystemPageHeader（标题 + 副标题 + 位置 chip trailing）；
    // 顶级 Tab 页标题对齐前端 titleSize=3xl（30）+ semibold（移动端，+page.svelte）
    SystemPageHeader({
      title: '首页',
      subtitle: '发现周边有趣的人和事',
      titleSize: DesignConstants.FONT.SIZE.XXL,
      titleWeight: DesignConstants.FONT.WEIGHT.SEMIBOLD
    }) {
      // trailing: 在线计数 chip + 搜索/通知/地图入口（对齐前端 PageHeader trailing）。
      // 溢出优先级（对齐前端弹性契约）：trailing 簇 flexShrink(0) 锁定不被压缩，
      // 窄屏下由标题副标题先省略让位（SystemPageHeader 内 Text 已 maxLines(1)+Ellipsis），
      // chip 文本同样 maxLines(1) 单行不折行——此前 trailing 无收缩保护，
      // 窄屏下 chip 会被压折行（对齐前端 360px 下「4 在线」折成两行的同款问题）。
      Row({ space: DesignConstants.SPACING.SM2 }) {
        // 0 人在线时隐藏计数 chip：空数据不广告（对齐前端隐藏空态的原则）
        if (this.getOnlineCount() > 0) {
          Row({ space: DesignConstants.SPACING.XS }) {
            Row()
              .width(DesignConstants.SPACING.SM2)
              .height(DesignConstants.SPACING.SM2)
              .borderRadius(DesignConstants.RADIUS.FULL)
              .backgroundColor(this.getPalette().online)

            Text(`${this.getOnlineCount()} 在线`)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
              .fontColor(this.getPalette().textSecondary)
              .maxLines(1)
          }
          .flexShrink(0)
          .padding({ left: DesignConstants.SPACING.SM2, right: DesignConstants.SPACING.SM2, top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
          .borderRadius(DesignConstants.RADIUS.FULL)
          .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().borderColor })
          .backgroundColor(this.getPalette().surfaceBackground)
          .shadow(this.getPalette().surfaceShadow) // 对齐前端 shadow-raised-xs
        }

        // 搜索/通知/地图入口：中性（透明底 + 主文字色图标），对齐前端 PageHeader trailing
        // （affa411a 为非桌面端补齐通知 Bell 快捷入口，本端同步；搜索/通知/地图三连）
        SurfaceIconButton({
          icon: AppIcon.SEARCH,
          kind: 'neutral',
          buttonSize: 36,
          glyphSize: DesignConstants.COMPONENT.ICON_SIZE_SM,
          onPress: () => AppRouter.push(PageNames.SEARCH)
        })

        SurfaceIconButton({
          icon: AppIcon.NOTIFICATION,
          kind: 'neutral',
          buttonSize: 36,
          glyphSize: DesignConstants.COMPONENT.ICON_SIZE_SM,
          onPress: () => AppStorage.setOrCreate<boolean>('openNotificationCenter', true)
        })

        SurfaceIconButton({
          icon: AppIcon.MAP,
          kind: 'neutral',
          buttonSize: 36,
          glyphSize: DesignConstants.COMPONENT.ICON_SIZE_SM,
          onPress: () => AppRouter.push(PageNames.LOCATION_PICKER)
        })
      }
      .flexShrink(0)
    }
  }

  // ==================== 地图卡片 ====================

  @Builder
  private buildHeroStats() {
    // 实时附近统计卡（对齐前端 HeroStats.svelte 当前实现：白卡 + 发丝描边 + raised 投影，
    // 无 ambient 光晕——前端 09-07 去炫光治理已删除 glow 层，本端同步删除）
    Column() {
      // Header row：图标 + 标题/范围 + 右箭头
      // 弹性契约（对齐前端 HeroStats）：左侧文本簇 layoutWeight(1) 先收缩，
      // 右箭头 flexShrink(0) 锁定；定位名过长时副标题单行省略，不顶出箭头。
      // （此前靠 Blank 撑开，文本簇无宽度约束，副标题 maxLines(1) 的省略永远触发不了，
      // 长定位名会把箭头顶出卡外——对齐前端同款问题。）
      Row() {
        Row({ space: DesignConstants.SPACING.SM2 }) {
          // brand 圆形图标（对齐前端 size-9.5 圆 + accent 实底，无投影）
          Row() {
            LucideIcon({
              icon: AppIcon.NEARBY,
              iconSize: DesignConstants.COMPONENT.ICON_SIZE_SM,
              paletteRole: 'accentText'
            })
          }
          .width(DesignConstants.COMPONENT.BRAND_ICON_SIZE)
          .height(DesignConstants.COMPONENT.BRAND_ICON_SIZE)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .backgroundColor(this.getPalette().accentBackground)
          .justifyContent(FlexAlign.Center)

          Column() {
            Text('实时附近')
              .fontSize(DesignConstants.FONT.SIZE.SM) // 12 对齐前端 text-sm
              .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
              .fontColor(this.getPalette().textPrimary)
            // 副标题：定位名 · 范围 · 更新节奏（对齐前端 HeroStats rangeLabel）
            Text(this.getHeroRangeLabel())
              .fontSize(DesignConstants.FONT.SIZE.XS) // 11 对齐前端 text-xs
              .fontColor(this.getPalette().textSecondary)
              .margin({ top: 1 })
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
          }
          .alignItems(HorizontalAlign.Start)
          .layoutWeight(1)
        }
        .alignItems(VerticalAlign.Center)
        .layoutWeight(1)

        Text() {
          LucideIcon({
            icon: AppIcon.ARROW_RIGHT,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
            paletteRole: 'textSecondary'
          })
        }
        .flexShrink(0)
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)

      // Stats grid：三段统计 + 竖分割线（对齐前端 bg-border-default gap-px 网格语义）
      // 数字统一 text-2xl(24) bold，对齐前端 display 化 stats（0d528548）
      Row({ space: DesignConstants.SPACING.MD }) {
        // 人附近（对齐前端：显示实际返回条数，limit 12）
        Column({ space: DesignConstants.SPACING.SM2 }) {
          Text(`${this.users.length}`)
            .fontSize(DesignConstants.FONT.SIZE.XXL)
            .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
            .fontColor(this.getPalette().textPrimary)
          Text('人附近')
            .fontSize(DesignConstants.FONT.SIZE.XS2) // 10 对齐前端 text-xxs
            .fontColor(this.getPalette().textSecondary)
        }
        .alignItems(HorizontalAlign.Center)
        .layoutWeight(1)

        // 分割竖线
        Column()
          .width(DesignConstants.BORDER_WIDTH.HAIRLINE)
          .height(DesignConstants.COMPONENT.AVATAR_SIZE.XS)
          .backgroundColor(this.getPalette().borderColor)

        // 活动中（对齐前端 heroStats.active = moments.length）
        Column({ space: DesignConstants.SPACING.SM2 }) {
          Text(`${this.moments.length}`)
            .fontSize(DesignConstants.FONT.SIZE.XXL)
            .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
            .fontColor(this.getPalette().textPrimary)
          Text('动态')
            .fontSize(DesignConstants.FONT.SIZE.XS2)
            .fontColor(this.getPalette().textSecondary)
        }
        .alignItems(HorizontalAlign.Center)
        .layoutWeight(1)

        // 分割竖线
        Column()
          .width(DesignConstants.BORDER_WIDTH.HAIRLINE)
          .height(DesignConstants.COMPONENT.AVATAR_SIZE.XS)
          .backgroundColor(this.getPalette().borderColor)

        // 在线
        Column({ space: DesignConstants.SPACING.SM2 }) {
          Text(`${this.getOnlineCount()}`)
            .fontSize(DesignConstants.FONT.SIZE.XXL)
            .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
            .fontColor(this.getPalette().textPrimary)
          Text('在线')
            .fontSize(DesignConstants.FONT.SIZE.XS2)
            .fontColor(this.getPalette().textSecondary)
        }
        .alignItems(HorizontalAlign.Center)
        .layoutWeight(1)
      }
      .width('100%')
      .margin({ top: DesignConstants.SPACING.LG })
    }
    .width('100%')
    // 页边距对齐前端 px-5(20)：卡本体不可贴屏幕边缘（此前 100% 宽导致边框贴边不可见）
    // 对齐前端 HeroStats 卡：py-4.5(18→LG 档) px-5(LG2) + rounded-xl(24) + raised-sm 投影
    .padding({ top: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.LG })
    .borderRadius(DesignConstants.SHAPE.HERO_CARD)
    .backgroundColor(this.getPalette().surfaceBackground)
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().borderColor })
    .shadow(this.getPalette().surfaceShadow)
    .scale(this.locationCardPressed
      ? { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD }
      : { x: 1, y: 1 })
    .animation({
      curve: curves.springMotion(
        DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
        DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
      )
    })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.locationCardPressed = true;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.locationCardPressed = false;
      }
    })
    .onClick(() => AppRouter.push(PageNames.NEARBY_USERS))
    // 首块与后续区块同节奏：统一底距，使下一个区块标题的上方间距与其他区块一致
    .margin({ bottom: DesignConstants.PAGE_SHELL.SECTION_GAP })
  }

  // ==================== 区块标题 (eyebrow) ====================

  @Builder
  private buildSectionHeader(title: string, linkText: string, pressedIndex: number, onViewAll: () => void) {
    // 对齐前端 SectionHead.svelte：分区标题用真实标题层级（15px semibold），
    // eyebrow 仅用于表单/分组 label；右侧 link-btn 文字链接 + 右箭头
    Row() {
      Text(title)
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
        .fontColor(this.getPalette().textPrimary)

      Blank()

      Row({ space: DesignConstants.SPACING.HALF }) {
        Text(linkText)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getPalette().accentBackground)
        LucideIcon({
          icon: AppIcon.ARROW_RIGHT,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
          color: this.getPalette().accentBackground
        })
      }
      .scale(this.viewAllPressedIndex === pressedIndex
        ? { x: DesignConstants.INTERACTION.SCALE_PRESSED_PILL, y: DesignConstants.INTERACTION.SCALE_PRESSED_PILL }
        : { x: 1, y: 1 })
      .animation({
        curve: curves.springMotion(
          DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
          DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
        )
      })
      .onTouch((event: TouchEvent) => {
        if (event.type === TouchType.Down) {
          this.viewAllPressedIndex = pressedIndex;
        } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
          this.viewAllPressedIndex = -1;
        }
      })
      .onClick(onViewAll)
      // 触控热区：行高 ~20vp，垂直外扩到 MIN_HIT_TARGET（responseRegion 不占布局）
      .responseRegion({
        x: -DesignConstants.SPACING.MD,
        y: -(DesignConstants.LAYOUT.MIN_HIT_TARGET - DesignConstants.COMPONENT.FILTER_CHIP_HEIGHT) / 2,
        width: '120%',
        height: DesignConstants.LAYOUT.MIN_HIT_TARGET
      })
    }
    .width('100%')
    // 区块标题 → 其内容：SECTION_INNER_GAP(12)，与区块之间的 SECTION_GAP(24) 形成
    // 2:1 层次——标题才在视觉上归属下方内容，而不是飘在两块之间
    .margin({ bottom: DesignConstants.PAGE_SHELL.SECTION_INNER_GAP })
  }

  // ==================== 附近的人 ====================

  @Builder
  private buildNearbyPeople() {
    Column() {
      this.buildSectionHeader('附近的人', '更多', 0, () => {
        AppRouter.push(PageNames.NEARBY_USERS);
      })

      if (!this.usersLoading && this.usersError.length > 0) {
        // 加载失败：错误文案 + 重试（P1-2 错误态收敛 EmptyState bare，区块级极简）
        EmptyState({
          icon: AppIcon.WARNING,
          message: this.usersError,
          actionLabel: '重试',
          onAction: () => { this.loadNearbyUsers(); },
          bare: true
        })
        .width('100%')
        .margin({ top: DesignConstants.SPACING.SM })
      } else if (!this.usersLoading && this.users.length === 0) {
        // 空态（对齐前端 EmptyState bare：极简形态，替代单行灰字；审查 P2 空态统一）
        EmptyState({
          icon: AppIcon.PERSON,
          message: '附近暂无其他用户，可稍后再来或调整浏览位置。',
          bare: true
        })
          .width('100%')
          .margin({ top: DesignConstants.SPACING.MD })
      } else if (this.usersLoading && this.users.length === 0) {
        // loading skeleton（行骨架：头像圆 + 两行文字；P1-3 收敛 Skeleton/SkeletonShimmer）
        Column() {
          SkeletonShimmer({ active: true }) {
            Column({ space: DesignConstants.SPACING.MD }) {
              ForEach(['1', '2', '3'], (_: string) => {
                Row({ space: DesignConstants.SPACING.MD }) {
                  Skeleton({
                    blockWidth: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL,
                    blockHeight: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL,
                    cornerRadius: DesignConstants.RADIUS.FULL
                  })
                  Column({ space: DesignConstants.SPACING.XS }) {
                    Skeleton({ blockWidth: '65%', blockHeight: DesignConstants.FONT.SIZE.SM })
                    Skeleton({ blockWidth: '45%', blockHeight: DesignConstants.FONT.SIZE.XS })
                  }
                  .alignItems(HorizontalAlign.Start)
                  .layoutWeight(1)
                }
                .width('100%')
              }, (_: string) => _)
            }
            .width('100%')
          }
        }
        .width('100%')
        .padding({ top: DesignConstants.SPACING.HALF })
      } else {
        // 竖排用户行列表（对齐前端 NearbyUserRows：ListContainer bordered 白卡容器
        // ——在线角标 + #tag · 距离副标题 + 关注按钮；移动端取前 4 位）。
        // 白卡容器契约（154ace61）：rounded-md(16) + border-subtle + raised-sm + p-1.5(6)
        Column() {
          ForEach(this.nearbyHomeUsers(), (user: UserItem) => {
            this.buildNearbyUserRow(user)
          }, (user: UserItem) => user.user?.user_id ?? '')
        }
        .width('100%')
        .padding(DesignConstants.SPACING.SM2)
        .borderRadius(DesignConstants.SHAPE.INPUT)
        .backgroundColor(this.getPalette().surfaceBackground)
        .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().borderColor })
        .shadow(this.getPalette().surfaceShadow)
      }
    }
    .width('100%')
    // 区块之间统一用「底距」：下一区块标题的上方间距恒为 SECTION_GAP(24)，
    // 不随各卡片自身 padding 变化（否则同类标题的上下间距会不一致）
    .margin({ bottom: DesignConstants.PAGE_SHELL.SECTION_GAP })
  }

  // 首页人物推荐条数（对齐前端 +page.svelte：移动端仅展示前 4 位，宽屏完整列表）
  private nearbyHomeUsers(): UserItem[] {
    if (useMasterDetail(this.currentBreakpoint)) return this.users;
    return this.users.slice(0, 4);
  }

  @Builder
  private buildNearbyUserRow(user: UserItem) {
    Row({ space: DesignConstants.SPACING.MD }) {
      Stack({ alignContent: Alignment.BottomEnd }) {
        SystemAvatar({
          avatarUrl: user.user?.avatar_url ?? '',
          name: user.user?.nickname ?? '',
          avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL_PLUS, // 44 对齐前端 tone='row' avatar 44
          shape: 'circle'
        })
        if (user.presence_status === 'online') {
          Row()
            .width(DesignConstants.SPACING.MD)
            .height(DesignConstants.SPACING.MD)
            .borderRadius(DesignConstants.RADIUS.FULL)
            .backgroundColor(this.getPalette().online)
            .border({ width: DesignConstants.BORDER_WIDTH.STRONG, color: this.getPalette().surfaceBackground })
        }
      }
      .width(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL_PLUS)
      .height(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL_PLUS)

      Column({ space: DesignConstants.SPACING.XS }) {
        Text(user.user?.nickname ?? '')
          .fontSize(DesignConstants.FONT.SIZE.SM) // 12 对齐前端 UserCardRow text-sm
          .fontWeight(DesignConstants.FONT.WEIGHT.REGULAR)
          .fontColor(this.getPalette().textPrimary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        // 副标题：#tag · 距离（对齐前端 UserCardRow subtitle 契约）
        Row({ space: DesignConstants.SPACING.SM2 }) {
          if (user.tags && user.tags.length > 0) {
            Text(`#${user.tags[0]}`)
              .fontSize(DesignConstants.FONT.SIZE.XS2)
              .fontColor(this.getPalette().accentBackground)
            Text('·')
              .fontSize(DesignConstants.FONT.SIZE.XS2)
              .opacity(DesignConstants.OPACITY.MUTED)
              .fontColor(this.getPalette().textSecondary)
          }
          if (user.location?.distance !== undefined && user.location?.distance !== null) {
            Text(LocationUtil.formatDistance(user.location.distance))
              .fontSize(DesignConstants.FONT.SIZE.XS2)
              .fontColor(this.getPalette().textSecondary)
          }
        }
        .alignItems(VerticalAlign.Center)
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)

      SystemButton({
        label: this.isUserFollowing(user.user?.user_id ?? '') ? '已关注' : '关注',
        variant: this.isUserFollowing(user.user?.user_id ?? '') ? 'outline' : 'primary',
        buttonSize: 'small',
        disabled: this.isUserActionPending(user.user?.user_id ?? ''),
        onPress: () => { this.followUser(user); }
      })
    }
    .width('100%')
    // 白卡容器内行：px-2.5 py-3（对齐前端 UserCardRow ListItem p-3 扣除容器 p-1.5）
    .padding({ left: DesignConstants.SPACING.MD2, right: DesignConstants.SPACING.MD2, top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
    // 整行按压反馈（对齐 tap-97 语义）
    .scale(this.pressedUserKey === (user.user?.user_id ?? '') ?
      { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD } :
      { x: 1, y: 1 })
    .animation({ curve: curves.springMotion(
      DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
      DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
    ) })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.pressedUserKey = user.user?.user_id ?? '';
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.pressedUserKey = '';
      }
    })
    .onClick(() => this.openUserProfile(user))
  }

  // ==================== 附近活动 ====================

  @Builder
  private buildGroupCarousel() {
    // 附近群组横滑条（对齐前端 GroupCarousel：水平滑动的群组卡片条）
    Column() {
      this.buildSectionHeader('附近群组', '更多', 1, () => {
        AppRouter.push(PageNames.NEARBY_GROUPS);
      })

      if (!this.groupsLoading && this.groupsError.length > 0) {
        // 加载失败：错误文案 + 重试（对齐前端 LoadError 契约，禁止静默转空态；
        // P1-2 错误态收敛 EmptyState bare，区块级极简）
        EmptyState({
          icon: AppIcon.WARNING,
          message: this.groupsError,
          actionLabel: '重试',
          onAction: () => { this.loadNearbyGroups(); },
          bare: true
        })
        .width('100%')
        .margin({ top: DesignConstants.SPACING.SM })
      } else if (!this.groupsLoading && this.groups.length === 0) {
        Text('附近暂时没有活跃的群组。')
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getPalette().textSecondary)
          .width('100%')
          .margin({ top: DesignConstants.SPACING.HALF })
      } else {
        Scroll() {
          Row({ space: DesignConstants.SPACING.MD }) {
            if (this.groupsLoading && this.groups.length === 0) {
              // loading skeleton（P1-3 收敛 Skeleton；横滚 Row 内不用 SkeletonShimmer
              // 包装——其根节点 width('100%') 会破坏横向滚动的内容宽度）
              Skeleton({
                blockWidth: DesignConstants.COMPONENT.GROUP_CARD_WIDTH,
                blockHeight: DesignConstants.COMPONENT.BANNER_HEIGHT,
                cornerRadius: DesignConstants.SHAPE.CARD
              })
              Skeleton({
                blockWidth: DesignConstants.COMPONENT.GROUP_CARD_WIDTH,
                blockHeight: DesignConstants.COMPONENT.BANNER_HEIGHT,
                cornerRadius: DesignConstants.SHAPE.CARD
              })
            } else {
              ForEach(this.groups, (group: GroupItem) => {
                // 固定卡宽容器：NearbyImageCard 依赖外部宽度测量，横滚 Row 不提供宽度约束
                Column() {
                  this.buildGroupCarouselCard(group)
                }
                .width(DesignConstants.COMPONENT.GROUP_CARD_WIDTH)
              }, (group: GroupItem) => group.group?.group_id ?? '')
            }
          }
          // 横滚容器：Row 宽度自适应内容，钉死 100% 会把溢出卡片裁掉（只显示首卡）
          .padding({ left: DesignConstants.SPACING.XS })
        }
        .align(Alignment.Start)
        .scrollBar(BarState.Off)
        .scrollable(ScrollDirection.Horizontal)
        .width('100%')
      }
    }
    .width('100%')
    // 区块之间统一用「底距」：下一区块标题的上方间距恒为 SECTION_GAP(24)，
    // 不随各卡片自身 padding 变化（否则同类标题的上下间距会不一致）
    .margin({ bottom: DesignConstants.PAGE_SHELL.SECTION_GAP })
  }

  @Builder
  private buildGroupCarouselCard(group: GroupItem) {
    // 140×140 banner 图方卡 + 隐私圆徽章 + 图上 card-menu 托盘（对齐前端 GroupCarousel）。
    // 导航必须走 NearbyImageCard 自身的 onPress：该组件根节点有自己的 onClick，
    // 会吞掉外层 Pressable 的点击（点击落在卡上时 Pressable.onClick 不触发）
    NearbyImageCard({
      imageUrl: group.group?.avatar_url ?? '',
      imageAlt: group.group?.name ?? '',
      cardAspectRatio: 1,
      placeholderText: group.group?.name ?? '',
      // 私有群（密码/邀请加入）显示锁圆徽；公开群 pill 空文本即不渲染
      badgeText: '',
      badgeIcon: AppIcon.LOCK,
      badgeShape: group.join_method !== 'direct' ? 'circle' : 'pill',
      badgeVariant: 'glass',
      badgeAlign: 'topRight',
      onPress: () => {
        const groupId = group.group?.group_id;
        if (groupId) {
          const params = new GroupDetailRouteParam();
          params.groupId = groupId;
          AppRouter.push(PageNames.GROUP_DETAIL, params);
        }
      }
    }) {
          Column({ space: DesignConstants.SPACING.THREE }) {
            Text(group.group?.name ?? '')
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
              .fontColor(this.getPalette().textPrimary)
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
              .width('100%')

            Row({ space: DesignConstants.SPACING.SM }) {
              Text(`${formatCount(group.group?.members_count ?? 0)} 人`)
                .fontSize(DesignConstants.FONT.SIZE.XS2)
                .fontColor(this.getPalette().textSecondary)
              if (group.location?.distance !== undefined && group.location?.distance !== null) {
                Text(LocationUtil.formatDistance(group.location.distance))
                  .fontSize(DesignConstants.FONT.SIZE.XS2)
                  .fontColor(this.getPalette().accentBackground)
              }
            }
            .width('100%')
          }
          .width('100%')
          .alignItems(HorizontalAlign.Start)
          .padding({
            left: DesignConstants.SPACING.MD,
            right: DesignConstants.SPACING.MD,
            top: DesignConstants.SPACING.SM2,
            bottom: DesignConstants.SPACING.SM2
          })
        }
  }

  // ==================== 附近动态 ====================

  @Builder
  private buildMomentsSection() {
    Column() {
      this.buildSectionHeader('附近活动', '更多', 2, () => {
        AppRouter.push(PageNames.NEARBY_MOMENTS);
      })

      if (!this.momentsLoading && this.momentsError.length > 0) {
        // 加载失败：错误文案 + 重试（对齐前端 LoadError 契约，禁止静默转空态；
        // P1-2 错误态收敛 EmptyState bare，区块级极简）
        EmptyState({
          icon: AppIcon.WARNING,
          message: this.momentsError,
          actionLabel: '重试',
          onAction: () => { this.loadNearbyMoments(); },
          bare: true
        })
        .width('100%')
        .margin({ top: DesignConstants.SPACING.SM })
      } else if (this.momentsLoading && this.moments.length === 0) {
        Column({ space: DesignConstants.SPACING.MD }) {
          ForEach(['1', '2', '3'], (_: string) => {
            Row({ space: DesignConstants.SPACING.MD }) {
              Column()
                .width(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL_PLUS)
                .height(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL_PLUS)
                .borderRadius(DesignConstants.RADIUS.SM)
                .backgroundColor(this.getPalette().surfaceSubtle)
              Column()
                .width('60%')
                .height(DesignConstants.FONT.SIZE.SM)
                .borderRadius(DesignConstants.RADIUS.SM)
                .backgroundColor(this.getPalette().surfaceSubtle)
            }
            .width('100%')
          }, (_: string) => _)
        }
        .width('100%')
      } else if (!this.momentsLoading && this.moments.length === 0) {
        Text('附近暂时没有动态，去发布第一条吧。')
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getPalette().textSecondary)
          .width('100%')
          .margin({ top: DesignConstants.SPACING.HALF })
      } else {
        // 紧凑活动行列表（对齐前端 momentRows：ListContainer bordered 白卡容器
        // + ActivityRow 色块 icon + 描述 + 地名 + chevron；
        // 首页为大纲页，移动端取前 3 条，完整卡片流在附近动态专区）。
        // 白卡容器契约（154ace61）：rounded-md(16) + border-subtle + raised-sm + p-1.5(6)
        Column() {
          ForEach(this.homeActivityRows(), (moment: MomentItem, index: number) => {
            this.buildActivityRow(moment, index)
          }, (moment: MomentItem) => moment.moment_id)
        }
        .width('100%')
        .padding(DesignConstants.SPACING.SM2)
        .borderRadius(DesignConstants.SHAPE.INPUT)
        .backgroundColor(this.getPalette().surfaceBackground)
        .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().borderColor })
        .shadow(this.getPalette().surfaceShadow)
      }
    }
    .width('100%')
    // 区块之间统一用「底距」：下一区块标题的上方间距恒为 SECTION_GAP(24)，
    // 不随各卡片自身 padding 变化（否则同类标题的上下间距会不一致）
    .margin({ bottom: DesignConstants.PAGE_SHELL.SECTION_GAP })
  }

  // 首页附近动态条数（对齐前端：移动端 3 条，宽屏完整列表）
  private homeActivityRows(): MomentItem[] {
    if (useMasterDetail(this.currentBreakpoint)) return this.moments;
    return this.moments.slice(0, 3);
  }

  // 活动行配色轮换（对齐前端 ActivityRow ICONS/COLORS/BGS 四档循环）：
  // [warning+咖啡, accent+音乐, marker-group+人群, danger+星标]，淡底全部走 token。
  private activityAccent(index: number): string {
    const accents: string[] = [
      this.getPalette().warning,
      this.getPalette().accentBackground,
      this.getPalette().mapGroup,
      this.getPalette().error
    ];
    return accents[index % 4];
  }

  private activitySoftBg(index: number): string {
    const bgs: string[] = [
      this.getPalette().warningSoft,
      this.getPalette().accentMuted,
      AppThemeVisuals.withAlpha(this.getPalette().mapGroup, 0.12),
      this.getPalette().dangerSoft
    ];
    return bgs[index % 4];
  }

  private activityIcon(index: number): AppIcon {
    const icons: AppIcon[] = [AppIcon.COFFEE, AppIcon.MUSIC, AppIcon.GROUP, AppIcon.STAR];
    return icons[index % 4];
  }

  @Builder
  private buildActivityRow(moment: MomentItem, index: number) {
    Row({ space: DesignConstants.SPACING.MD }) {
      Column() {
        LucideIcon({
          icon: this.activityIcon(index),
          iconSize: DesignConstants.FONT.SIZE.LG,
          color: this.activityAccent(index)
        })
      }
      // 对齐前端 ActivityRow size-8.5(34px) 图标圆底
      .width(DesignConstants.COMPONENT.ACTIVITY_ICON_TILE_SIZE)
      .height(DesignConstants.COMPONENT.ACTIVITY_ICON_TILE_SIZE)
      .borderRadius(DesignConstants.RADIUS.SM)
      .backgroundColor(this.activitySoftBg(index))
      .justifyContent(FlexAlign.Center)

      Column({ space: DesignConstants.SPACING.XS }) {
        Text(moment.description || '分享了一条动态')
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor(this.getPalette().textPrimary)
          .maxLines(2)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
        Text(moment.location?.location_name ?? '')
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getPalette().textSecondary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
          .margin({ top: 1 })
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)

      LucideIcon({
        icon: AppIcon.ARROW_RIGHT,
        iconSize: DesignConstants.FONT.SIZE.MD,
        paletteRole: 'textSecondary'
      })
    }
    .width('100%')
    // 白卡容器内行：px-3 py-2.5（对齐前端 ActivityRow ListItem px-4 py-3 扣除容器 p-1.5）
    .padding({ left: DesignConstants.SPACING.MD2, right: DesignConstants.SPACING.MD2, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
    // 整行按压反馈（对齐 tap-97 语义）
    .scale(this.pressedMomentKey === moment.moment_id ?
      { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD } :
      { x: 1, y: 1 })
    .animation({ curve: curves.springMotion(
      DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
      DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
    ) })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.pressedMomentKey = moment.moment_id;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.pressedMomentKey = '';
      }
    })
    .onClick(() => this.openMomentDetail(moment))
  }
}

class GroupDetailParams {
  groupId: string = '';
}