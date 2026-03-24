/**
 * 附近页面 - 主标签页
 * 包含动态、用户、群组三个子标签，顶部位置栏和搜索/地图入口
 * 类似抖音风格的子标签切换
 */

import { GlobalStateManager, KEY_IS_LOGGED_IN, KEY_BROWSE_ANCHOR, KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { LocationInfo, LocationUtil } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { AppRouter, ChatRouteParam, MomentDetailRouteParam, ProfileRouteParam } from '../../../core/router/AppRouter';
import { PageNames, DefaultConfig } from '../../../constants/AppConstants';
import {
  MomentItem, UserItem, GroupItem, GeoPoint, RelativeLocation,
  SearchMomentsRequest, SearchUsersRequest, SearchGroupsRequest,
  MomentImage, JoinGroupRequest,
  BlockRequest, UnblockRequest, CreateReportRequest, Relationship
} from '../../../core/types/api';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { TabsPillGroup } from '../../../components/TabsPillGroup';
import { MomentFeedCard } from '../../../components/MomentFeedCard';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { InteractionStateStore, UserRelationState } from '../../../core/state/InteractionStateStore';
import { interactionActions } from '../../../core/state/InteractionActions';
import { LocationBar } from '../../../components/LocationBar';
import { EmptyState, LoadingMoreIndicator, RefreshIndicator } from '../../../components/EmptyState';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { DistributedManager } from '../../../core/distributed/DistributedManager';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { WidgetDataService } from '../../../core/services/WidgetDataService';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { MediaUrlUtil } from '../../../core/utils/MediaUrlUtil';
import { formatCount, formatRelativeTime } from '../../../core/utils/FormatUtil';
import pasteboard from '@ohos.pasteboard';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';


const DOMAIN_NUMBER: number = 0xFF10;
const TAG: string = 'NearbyPage';

class GroupDetailParams {
  groupId: string = '';
}

class LoginRedirectParams {
  reason: string = '';
  action: string = '';
}

@Preview
@Component
export struct NearbyPage {
  @StorageLink(KEY_IS_LOGGED_IN) isLoggedIn: boolean = false;
  @StorageLink(KEY_CURRENT_LOCATION) @Watch('onCurrentLocationChange') currentLocation: LocationInfo | null = null;
  @StorageLink(KEY_BROWSE_ANCHOR) @Watch('onBrowseAnchorChange') browseAnchor: LocationInfo | null = null;
  @StorageLink('nearbyDeviceCount') nearbyDeviceCount: number = 0;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';
  @StorageLink('interaction_user_state') userStateMap: Record<string, UserRelationState> = {};

  @State currentSubTab: number = 0;
  private subTabController: TabsController = new TabsController();
  /** 标记 aboutToAppear 是否已完成首次数据加载，避免 @Watch 重复触发 */
  private initialLoadDone: boolean = false;

  // 动态数据
  @State moments: MomentItem[] = [];
  private momentsDataSource: BasicDataSource<MomentItem> = new BasicDataSource<MomentItem>([]);
  private interactionStore = InteractionStateStore.getInstance();
  @State momentsLoading: boolean = true;
  @State momentsRefreshing: boolean = false;
  @State momentsError: string = '';
  @State momentsHasMore: boolean = false;
  @State momentsNextCursor: string = '';
  @State momentsLoadingMore: boolean = false;

  // 用户数据
  @State users: UserItem[] = [];
  private usersDataSource: BasicDataSource<UserItem> = new BasicDataSource<UserItem>([]);
  @State usersLoading: boolean = true;
  @State usersRefreshing: boolean = false;
  @State usersError: string = '';
  @State usersHasMore: boolean = false;
  @State usersNextCursor: string = '';
  @State usersLoadingMore: boolean = false;

  // 群组数据
  @State groups: GroupItem[] = [];
  private groupsDataSource: BasicDataSource<GroupItem> = new BasicDataSource<GroupItem>([]);
  @State groupsLoading: boolean = true;
  @State groupsRefreshing: boolean = false;
  @State groupsError: string = '';
  @State groupsHasMore: boolean = false;
  @State groupsNextCursor: string = '';
  @State groupsLoadingMore: boolean = false;
  @State userActionPendingIds: string[] = [];
  @State groupActionPendingIds: string[] = [];

  async aboutToAppear() {
    SystemUiController.syncWithCurrentTheme();
    // 预加载所有 tab 数据（并行），切换 tab 时数据已就绪，无需再显示 loading
    this.loadNearbyMoments();
    this.loadNearbyUsers();
    this.loadNearbyGroups();
    this.initialLoadDone = true;

    // Start device discovery (non-blocking)
    try {
      const dm = DistributedManager.getInstance();
      dm.startDeviceDiscovery();
    } catch {
      hilog.warn(DOMAIN_NUMBER, TAG, '设备发现启动失败');
    }
  }

  private getSurfaceBackground(): string {
    return this.getCurrentThemePalette().surfaceBackground;
  }

  private getSurfaceSubtle(): string {
    return this.getCurrentThemePalette().surfaceSubtle;
  }

  private getSurfaceBorder(): string {
    return this.getCurrentThemePalette().borderColor;
  }

  private getAccentBackground(): string {
    return this.getCurrentThemePalette().accentBackground;
  }

  private getAccentText(): string {
    return this.getCurrentThemePalette().accentText;
  }

  aboutToDisappear() {
    try {
      DistributedManager.getInstance().stopDeviceDiscovery();
    } catch {
      // ignore
    }
  }

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  /**
   * currentLocation 变化时：仅当没有设置浏览锚点且当前数据为空/出错时自动刷新
   * 典型场景：页面加载时 GPS 尚未定位，定位完成后自动加载附近数据
   */
  private onCurrentLocationChange(): void {
    if (!this.initialLoadDone) return;
    // 已设置浏览锚点时，currentLocation 变化不影响搜索中心
    if (this.browseAnchor) return;
    // 新位置为 null 则无需处理
    if (!this.currentLocation) return;
    // 仅当当前 tab 无数据或有错误时才自动刷新（避免频繁 GPS 更新导致反复请求）
    if (this.isCurrentTabEmpty()) {
      hilog.info(DOMAIN_NUMBER, TAG, '位置可用，自动加载附近数据');
      this.loadCurrentTabData();
    }
  }

  /**
   * browseAnchor 变化时：用户主动切换了搜索中心，始终刷新当前 tab 数据
   */
  private onBrowseAnchorChange(): void {
    if (!this.initialLoadDone) return;
    hilog.info(DOMAIN_NUMBER, TAG, `浏览锚点变化 -> ${this.browseAnchor ? this.browseAnchor.locationName : '已清除'}`);
    this.loadNearbyMoments();
    this.loadNearbyUsers();
    this.loadNearbyGroups();
  }

  /** 判断当前 tab 是否无数据（用于决定 location 变化时是否需要自动加载） */
  private isCurrentTabEmpty(): boolean {
    switch (this.currentSubTab) {
      case 0: return this.moments.length === 0;
      case 1: return this.users.length === 0;
      case 2: return this.groups.length === 0;
      default: return true;
    }
  }

  private getReferencePoint(): GeoPoint | null {
    const loc: LocationInfo | null = this.browseAnchor ?? this.currentLocation;
    if (!loc) return null;
    const point: GeoPoint = { latitude: loc.latitude, longitude: loc.longitude };
    return point;
  }

  private async loadCurrentTabData(refresh: boolean = false): Promise<void> {
    switch (this.currentSubTab) {
      case 0:
        await this.loadNearbyMoments(false, refresh);
        break;
      case 1:
        await this.loadNearbyUsers(false, refresh);
        break;
      case 2:
        await this.loadNearbyGroups(false, refresh);
        break;
    }
  }

  private getHeaderTopPadding(): number {
    return SafeAreaUtils.resolveTopPadding(this.safeAreaTop);
  }

  private getListBottomPadding(): number {
    // 底部安全区域已由 Tab Bar 处理，列表只需少量视觉边距
    return DesignConstants.SPACING.MD;
  }

  private isUserActionPending(userId: string): boolean {
    return this.userActionPendingIds.indexOf(userId) >= 0;
  }

  private isGroupActionPending(groupId: string): boolean {
    return this.groupActionPendingIds.indexOf(groupId) >= 0;
  }

  private isUserBlocked(user: UserItem): boolean {
    return user.relation?.is_blocked ?? false;
  }

  private isGroupJoined(group: GroupItem): boolean {
    return group.my_state?.is_joined ?? false;
  }

  private switchSubTab(index: number): void {
    if (this.currentSubTab === index) {
      return;
    }
    this.currentSubTab = index;
    // 在 Tabs 动画开始前预设目标 tab 的 loading 状态，
    // 避免 Tabs 捕获旧内容（错误/空状态）用于过渡动画
    this.prepareTabLoading(index);
    this.subTabController.changeIndex(index);
    // 兜底触发加载：部分设备上 controller 切换不一定触发 onChange
    this.loadCurrentTabData(true);
  }

  /** 为目标 tab 预设 loading 状态（仅当该 tab 无数据或存在错误时） */
  private prepareTabLoading(index: number): void {
    switch (index) {
      case 0:
        if (this.moments.length === 0 || this.momentsError.length > 0) {
          this.momentsLoading = true;
          this.momentsError = '';
        }
        break;
      case 1:
        if (this.users.length === 0 || this.usersError.length > 0) {
          this.usersLoading = true;
          this.usersError = '';
        }
        break;
      case 2:
        if (this.groups.length === 0 || this.groupsError.length > 0) {
          this.groupsLoading = true;
          this.groupsError = '';
        }
        break;
    }
  }

  // ==================== 加载附近动态 ====================
  private async loadNearbyMoments(loadMore: boolean = false, refresh: boolean = false): Promise<void> {
    const ref = this.getReferencePoint();
    if (!ref) {
      if (!loadMore) { this.momentsLoading = false; this.momentsRefreshing = false; }
      return;
    }
    if (loadMore) {
      if (this.momentsLoadingMore || !this.momentsHasMore) return;
      this.momentsLoadingMore = true;
    } else {
      if (!refresh || this.moments.length === 0) { this.momentsLoading = true; }
      this.momentsError = '';
    }
    try {
      const request: SearchMomentsRequest = {
        reference_point: ref,
        radius: DefaultConfig.NEARBY_RADIUS,
        sort_by: 'distance',
        pagination: { limit: DefaultConfig.PAGE_SIZE, cursor: loadMore ? this.momentsNextCursor : undefined }
      };
      const response = loadMore
        ? await ApiClient.searchMoments(request)
        : await LoadingUtil.withMinimumTime(() => ApiClient.searchMoments(request));
      if (response.code === 0) {
        const items = response.data?.items ?? [];
        const validItems = items.filter((m: MomentItem) => m.moment_id && m.user);
        if (loadMore) {
          this.moments = this.moments.concat(validItems);
          this.momentsDataSource.appendData(validItems);
        } else {
          this.moments = validItems;
          this.momentsDataSource.setData(validItems);
        }
        validItems.forEach((item: MomentItem) => this.interactionStore.patchMomentFromItem(item));
        this.momentsHasMore = !!(response.data?.next_cursor);
        this.momentsNextCursor = response.data?.next_cursor ?? '';
        try {
          WidgetDataService.getInstance().updateStats('nearbyMomentCount', this.moments.length);
        } catch {
          // 小组件数据更新失败不影响主流程
        }
      } else if (!loadMore) {
        this.momentsError = response.message || '加载失败';
      }
    } catch {
      hilog.error(DOMAIN_NUMBER, TAG, '加载附近动态失败');
      if (!loadMore) {
        this.momentsError = '加载附近动态失败';
      }
    } finally {
      if (loadMore) {
        this.momentsLoadingMore = false;
      } else {
        this.momentsLoading = false;
        this.momentsRefreshing = false;
      }
    }
  }

  // ==================== 加载附近用户 ====================
  private async loadNearbyUsers(loadMore: boolean = false, refresh: boolean = false): Promise<void> {
    const ref = this.getReferencePoint();
    if (!ref) {
      if (!loadMore) { this.usersLoading = false; this.usersRefreshing = false; }
      return;
    }
    if (loadMore) {
      if (this.usersLoadingMore || !this.usersHasMore) return;
      this.usersLoadingMore = true;
    } else {
      if (!refresh || this.users.length === 0) { this.usersLoading = true; }
      this.usersError = '';
    }
    try {
      const request: SearchUsersRequest = {
        reference_point: ref,
        radius: DefaultConfig.NEARBY_RADIUS,
        sort_by: 'distance',
        pagination: { limit: DefaultConfig.PAGE_SIZE, cursor: loadMore ? this.usersNextCursor : undefined }
      };
      const response = loadMore
        ? await ApiClient.searchUsers(request)
        : await LoadingUtil.withMinimumTime(() => ApiClient.searchUsers(request));
      if (response.code === 0) {
        const items = response.data?.items ?? [];
        const me = GlobalStateManager.getInstance().getCurrentUser();
        const myId = me?.user_id;
        const validItemsAll = items.filter((u: UserItem) => u.user?.user_id);
        const validItems = typeof myId === 'string'
          ? validItemsAll.filter((u: UserItem) => u.user.user_id !== myId)
          : validItemsAll;
        if (loadMore) {
          this.users = this.users.concat(validItems);
          this.usersDataSource.appendData(validItems);
        } else {
          this.users = validItems;
          this.usersDataSource.setData(validItems);
        }
        validItems.forEach((item: UserItem) => this.interactionStore.patchUserFromItem(item));
        this.usersHasMore = !!(response.data?.next_cursor);
        this.usersNextCursor = response.data?.next_cursor ?? '';
        try {
          WidgetDataService.getInstance().updateStats('nearbyUserCount', this.users.length);
        } catch {
          // 小组件数据更新失败不影响主流程
        }
      } else if (!loadMore) {
        this.usersError = response.message || '加载失败';
      }
    } catch {
      hilog.error(DOMAIN_NUMBER, TAG, '加载附近用户失败');
      if (!loadMore) {
        this.usersError = '加载附近用户失败';
      }
    } finally {
      if (loadMore) {
        this.usersLoadingMore = false;
      } else {
        this.usersLoading = false;
        this.usersRefreshing = false;
      }
    }
  }

  // ==================== 加载附近群组 ====================
  private async loadNearbyGroups(loadMore: boolean = false, refresh: boolean = false): Promise<void> {
    const ref = this.getReferencePoint();
    if (!ref) {
      if (!loadMore) { this.groupsLoading = false; this.groupsRefreshing = false; }
      return;
    }
    if (loadMore) {
      if (this.groupsLoadingMore || !this.groupsHasMore) return;
      this.groupsLoadingMore = true;
    } else {
      if (!refresh || this.groups.length === 0) { this.groupsLoading = true; }
      this.groupsError = '';
    }
    try {
      const request: SearchGroupsRequest = {
        reference_point: ref,
        radius: DefaultConfig.NEARBY_RADIUS,
        sort_by: 'distance',
        pagination: { limit: DefaultConfig.PAGE_SIZE, cursor: loadMore ? this.groupsNextCursor : undefined }
      };
      const response = loadMore
        ? await ApiClient.searchGroups(request)
        : await LoadingUtil.withMinimumTime(() => ApiClient.searchGroups(request));
      if (response.code === 0) {
        const items = response.data?.items ?? [];
        const validItems = items.filter((g: GroupItem) => g.group?.group_id);
        if (loadMore) {
          this.groups = this.groups.concat(validItems);
          this.groupsDataSource.appendData(validItems);
        } else {
          this.groups = validItems;
          this.groupsDataSource.setData(validItems);
        }
        this.groupsHasMore = !!(response.data?.next_cursor);
        this.groupsNextCursor = response.data?.next_cursor ?? '';
        try {
          WidgetDataService.getInstance().updateStats('nearbyGroupCount', this.groups.length);
        } catch {
          // 小组件数据更新失败不影响主流程
        }
      } else if (!loadMore) {
        this.groupsError = response.message || '加载失败';
      }
    } catch {
      hilog.error(DOMAIN_NUMBER, TAG, '加载附近群组失败');
      if (!loadMore) {
        this.groupsError = '加载附近群组失败';
      }
    } finally {
      if (loadMore) {
        this.groupsLoadingMore = false;
      } else {
        this.groupsLoading = false;
        this.groupsRefreshing = false;
      }
    }
  }

  build() {
    Column() {
      this.buildHeaderSurface()

      Tabs({ controller: this.subTabController }) {
        TabContent() {
          this.buildMomentsContent()
        }
        .tabBar(this.HiddenTabBar())

        TabContent() {
          this.buildUsersContent()
        }
        .tabBar(this.HiddenTabBar())

        TabContent() {
          this.buildGroupsContent()
        }
        .tabBar(this.HiddenTabBar())
      }
      .barMode(BarMode.Fixed)
      .barHeight(0)
      .animationDuration(400)
      .onChange((index: number) => {
        if (this.currentSubTab === index) {
          return;
        }
        this.prepareTabLoading(index);
        this.currentSubTab = index;
        this.loadCurrentTabData(true);
      })
      .layoutWeight(1)
    }
    .width('100%')
    .height('100%')
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
  }

  @Builder
  private HiddenTabBar() {
    Blank()
      .width(0)
      .height(0)
  }

  @Builder
  private buildHeaderSurface() {
    Column({ space: DesignConstants.SPACING.SM }) {
      LocationBar({ showSearchButton: true })

      if (this.nearbyDeviceCount > 0) {
        Row({ space: DesignConstants.SPACING.SM }) {
          Text() {
            SymbolSpan($r(AppIcon.DISTRIBUTED))
              .fontSize(DesignConstants.FONT.SIZE.MD)
              .fontColor([this.getCurrentThemePalette().accentBackground])
          }
          Text(`发现 ${this.nearbyDeviceCount} 台附近设备正在使用 GeoTalk`)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getCurrentThemePalette().textSecondary)
        }
        .width('100%')
        .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD,
          top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
        .backgroundColor(Color.Transparent)
        .transition(
          TransitionEffect.OPACITY.animation({ duration: 280, curve: Curve.FastOutSlowIn })
            .combine(TransitionEffect.move(TransitionEdge.TOP).animation({ duration: 280, curve: Curve.FastOutSlowIn }))
        )
      }

      TabsPillGroup({
        tabs: [
          { label: '动态', icon: AppIcon.CAMERA },
          { label: '用户', icon: AppIcon.PERSON },
          { label: '群组', icon: AppIcon.GROUP }
        ],
        activeIndex: this.currentSubTab,
        gap: DesignConstants.SPACING.XS,
        actions: {
          onChange: (index: number) => this.switchSubTab(index)
        }
      })
    }
    .width('100%')
    .padding({
      top: this.getHeaderTopPadding(),
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      bottom: DesignConstants.SPACING.SM
    })
    .backgroundColor(Color.Transparent)
    .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])
  }

  // ==================== 动态内容 ====================
  @Builder
  buildMomentsContent() {
    Column() {
      if (this.momentsLoading) {
        SystemLoadingView({ message: '正在加载附近动态...' })
      } else if (this.momentsError.length > 0 && this.moments.length === 0) {
        EmptyState({ icon: AppIcon.WARNING, message: this.momentsError, actionLabel: '重试', onAction: () => { this.loadNearbyMoments(); } })
      } else if (this.moments.length === 0) {
        EmptyState({ icon: AppIcon.CAMERA, message: '附近暂无动态\n去发布第一条吧！' })
      } else {
        Refresh({ refreshing: $$this.momentsRefreshing, builder: RefreshIndicator }) {
          List({ space: DesignConstants.SPACING.MD }) {
            LazyForEach(this.momentsDataSource, (moment: MomentItem) => {
              ListItem() {
                MomentFeedCard({
                  moment: moment,
                  onOpenDetail: (item: MomentItem) => this.openMomentDetail(item)
                })
              }
            }, (moment: MomentItem) => moment.moment_id)
            if (this.momentsLoadingMore) {
              ListItem() {
                LoadingMoreIndicator()
              }
            }
          }
          .padding({
            left: DesignConstants.SPACING.LG,
            right: DesignConstants.SPACING.LG,
            top: DesignConstants.SPACING.SM,
            bottom: this.getListBottomPadding()
          })
          .width('100%')
          .scrollBar(BarState.Auto)
          .onReachEnd(() => { this.loadNearbyMoments(true); })
        }
        .layoutWeight(1)
        .onRefreshing(() => { this.loadNearbyMoments(false, true); })
        .transition(
          TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
            .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
        )
      }
    }
    .width('100%')
    .height('100%')
  }

  private openMomentDetail(moment: MomentItem): void {
    hilog.info(DOMAIN_NUMBER, TAG, `打开动态详情: ${moment.moment_id}`);
    const params = new MomentDetailRouteParam();
    params.momentId = moment.moment_id;
    AppRouter.push(PageNames.MOMENT_DETAIL, params);
  }

  // ==================== 用户内容 ====================
  @Builder
  buildUsersContent() {
    Column() {
      if (this.usersLoading) {
        SystemLoadingView({ message: '正在加载附近用户...' })
          .transition(
            TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
              .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
          )
      } else if (this.usersError.length > 0 && this.users.length === 0) {
        EmptyState({ icon: AppIcon.WARNING, message: this.usersError, actionLabel: '重试', onAction: () => { this.loadNearbyUsers(); } })
          .transition(
            TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
              .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
          )
      } else if (this.users.length === 0) {
        EmptyState({ icon: AppIcon.PERSON, message: '附近暂无其他用户' })
          .transition(
            TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
              .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
          )
      } else {
        Refresh({ refreshing: $$this.usersRefreshing, builder: RefreshIndicator }) {
          List() {
            LazyForEach(this.usersDataSource, (user: UserItem) => {
              ListItem() {
                this.buildUserCard(user)
              }
            }, (user: UserItem) => user.user?.user_id ?? '')
            if (this.usersLoadingMore) {
              ListItem() {
                LoadingMoreIndicator()
              }
            }
          }
          .padding({
            left: DesignConstants.SPACING.LG,
            right: DesignConstants.SPACING.LG,
            top: DesignConstants.SPACING.SM,
            bottom: this.getListBottomPadding()
          })
          .scrollBar(BarState.Auto)
          .onReachEnd(() => { this.loadNearbyUsers(true); })
        }
        .layoutWeight(1)
        .onRefreshing(() => { this.loadNearbyUsers(false, true); })
        .transition(
          TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
            .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
        )
      }
    }
    .width('100%')
    .height('100%')
  }

  @Builder
  buildUserCard(user: UserItem) {
    Column({ space: DesignConstants.SPACING.SM }) {
      Row({ space: 6 }) {
        Row({ space: DesignConstants.SPACING.XS }) {
          Text() {
            SymbolSpan($r(AppIcon.LOCATION))
              .fontSize(10)
              .fontColor([this.getCurrentThemePalette().accentBackground])
          }
          Text('附近用户')
            .fontSize(10)
            .fontColor(this.getCurrentThemePalette().textSecondary)
        }
        .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM, top: 3, bottom: 3 })
               .backgroundColor(this.getSurfaceSubtle())
        .borderRadius(DesignConstants.RADIUS.XL)

        Blank().layoutWeight(1)

        if (user.location?.distance !== undefined && user.location?.distance !== null) {
          Text(LocationUtil.formatDistance(user.location.distance))
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getCurrentThemePalette().accentBackground)
            .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM, top: 3, bottom: 3 })
            .backgroundColor(this.getSurfaceSubtle())
            .borderRadius(DesignConstants.RADIUS.XL)
        }
      }
      .width('100%')

      Row({ space: DesignConstants.SPACING.SM }) {
        Row({ space: DesignConstants.SPACING.SM }) {
          SystemAvatar({
            avatarUrl: user.user?.avatar_url ?? '',
            name: user.user?.nickname ?? '',
            avatarSize: 44,
            shape: 'circle'
          })

          Column({ space: DesignConstants.SPACING.XS }) {
            Text(user.user?.nickname ?? '')
              .fontSize(15)
              .fontWeight(FontWeight.Medium)
              .fontColor(this.getCurrentThemePalette().textPrimary)
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })

            Row({ space: DesignConstants.SPACING.XS }) {
              if (user.presence_status === 'online') {
                Row()
                  .width(6)
                  .height(6)
                  .borderRadius(DesignConstants.RADIUS.FULL)
                  .backgroundColor(this.getCurrentThemePalette().accentBackground)
                Text('在线')
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor(this.getCurrentThemePalette().accentBackground)
              } else {
                Row()
                  .width(6)
                  .height(6)
                  .borderRadius(DesignConstants.RADIUS.FULL)
                  .backgroundColor(this.getCurrentThemePalette().textTertiary)
                Text('离线')
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor(this.getCurrentThemePalette().textSecondary)
              }
            }
            .alignItems(VerticalAlign.Center)
          }
          .alignItems(HorizontalAlign.Start)
          .layoutWeight(1)
        }
        .layoutWeight(1)
        .onClick(() => {
          this.openUserProfile(user);
        })

        Row() {
          Text() {
            SymbolSpan($r(AppIcon.MORE))
              .fontSize(DesignConstants.FONT.SIZE.LG)
              .fontColor([this.getCurrentThemePalette().textSecondary])
          }
        }
        .width(30)
        .height(30)
        .borderRadius(DesignConstants.RADIUS.XL)
        .backgroundColor(this.getSurfaceSubtle())
        .border({ width: 1, color: this.getSurfaceBorder() })
        .justifyContent(FlexAlign.Center)
        .alignItems(VerticalAlign.Center)
        .bindMenu([
          {
            value: '私聊',
            action: () => this.startPrivateChat(user)
          },
          {
            value: this.isUserFollowing(user.user?.user_id ?? '') ? '取消关注' : '关注',
            action: () => this.followUser(user)
          },
          {
            value: this.isUserBlocked(user) ? '取消拉黑' : '拉黑',
            action: () => {
              if (this.isUserBlocked(user)) {
                this.unblockUser(user);
              } else {
                this.blockUser(user);
              }
            }
          },
          {
            value: '举报用户',
            action: () => this.reportUser(user)
          },
          {
            value: '复制用户ID',
            action: () => this.copyText(user.user?.user_id ?? '', '用户ID已复制')
          }
        ])
      }
      .width('100%')

      if (user.bio) {
        Text(user.bio)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .lineHeight(17)
          .fontColor(this.getCurrentThemePalette().textSecondary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
          .width('100%')
      } else {
        Text('这个人很神秘，暂时还没有留下简介')
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .lineHeight(17)
          .fontColor(this.getCurrentThemePalette().textSecondary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
          .width('100%')
      }

      if ((user.social_context?.mutual_friends_count ?? 0) > 0 || (user.tags && user.tags.length > 0)) {
        Flex({ wrap: FlexWrap.Wrap }) {
          if ((user.social_context?.mutual_friends_count ?? 0) > 0) {
            Text(`共同好友 ${formatCount(user.social_context?.mutual_friends_count)}`)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor(this.getCurrentThemePalette().textSecondary)
              .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM, top: 3, bottom: 3 })
              .margin({ right: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
               .backgroundColor(this.getSurfaceSubtle())
               .borderRadius(DesignConstants.RADIUS.XL)
          }
           ForEach((user.tags ?? []).slice(0, 2), (tag: string) => {
            Text(`#${tag}`)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor(this.getCurrentThemePalette().textSecondary)
              .padding({ left: 7, right: 7, top: 3, bottom: 3 })
              .margin({ right: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
               .backgroundColor(this.getSurfaceSubtle())
              .borderRadius(DesignConstants.RADIUS.XL)
          }, (tag: string) => tag)
        }
        .width('100%')
      }

      Row({ space: DesignConstants.SPACING.SM }) {
        Button(this.isUserActionPending(user.user?.user_id ?? '') ? '处理中...' :
          (this.isUserFollowing(user.user?.user_id ?? '') ? '已关注' : '关注'))
          .type(ButtonType.Capsule)
          .layoutWeight(1)
          .height(40)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .enabled(!this.isUserActionPending(user.user?.user_id ?? '') && !this.isUserBlocked(user))
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontWeight(FontWeight.Medium)
          .backgroundColor(this.isUserFollowing(user.user?.user_id ?? '') ? this.getSurfaceSubtle() : this.getAccentBackground())
          .fontColor(this.isUserFollowing(user.user?.user_id ?? '') ? this.getAccentBackground() : this.getAccentText())
          .shadow(this.isUserFollowing(user.user?.user_id ?? '') ? { radius: 0, color: '#00000000', offsetX: 0, offsetY: 0 } : this.getCurrentThemePalette().controlShadow)
          .border(this.isUserFollowing(user.user?.user_id ?? '') ? { width: 0 } : { width: 1, color: this.getSurfaceBorder() })
          .opacity(this.isUserActionPending(user.user?.user_id ?? '') || this.isUserBlocked(user) ? 0.58 : 1)
          .onClick(() => { this.followUser(user); })

        Button('私聊')
          .type(ButtonType.Capsule)
          .layoutWeight(1)
          .height(40)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .enabled(!this.isUserBlocked(user))
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontWeight(FontWeight.Medium)
          .backgroundColor(this.getSurfaceSubtle())
          .fontColor(this.getCurrentThemePalette().accentBackground)
          .border({ width: 1, color: this.getSurfaceBorder() })
          .opacity(this.isUserBlocked(user) ? 0.58 : 1)
          .onClick(() => { this.startPrivateChat(user); })
      }
      .width('100%')
    }
    .width('100%')
    .alignItems(HorizontalAlign.Start)
    .padding({
      left: DesignConstants.SPACING.XL,
      right: DesignConstants.SPACING.XL,
      top: DesignConstants.SPACING.LG,
      bottom: DesignConstants.SPACING.LG
    })
    .backgroundColor(this.getSurfaceBackground())
    .borderRadius(DesignConstants.RADIUS.XXL + 6)
    .shadow(this.getCurrentThemePalette().elevatedShadow)
    .border({ width: 1, color: this.getSurfaceBorder() })
    .margin({ bottom: DesignConstants.SPACING.MD })
  }

  // ==================== 群组内容 ====================
  @Builder
  buildGroupsContent() {
    Column() {
      if (this.groupsLoading) {
        SystemLoadingView({ message: '正在加载附近群组...' })
          .transition(
            TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
              .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
          )
      } else if (this.groupsError.length > 0 && this.groups.length === 0) {
        EmptyState({ icon: AppIcon.WARNING, message: this.groupsError, actionLabel: '重试', onAction: () => { this.loadNearbyGroups(); } })
          .transition(
            TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
              .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
          )
      } else if (this.groups.length === 0) {
        EmptyState({ icon: AppIcon.GROUP, message: '附近暂无群组' })
          .transition(
            TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
              .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
          )
      } else {
        Refresh({ refreshing: $$this.groupsRefreshing, builder: RefreshIndicator }) {
          List() {
            LazyForEach(this.groupsDataSource, (group: GroupItem) => {
              ListItem() {
                this.buildGroupCard(group)
              }
            }, (group: GroupItem) => group.group?.group_id ?? '')
            if (this.groupsLoadingMore) {
              ListItem() {
                LoadingMoreIndicator()
              }
            }
          }
          .padding({
            left: DesignConstants.SPACING.LG,
            right: DesignConstants.SPACING.LG,
            top: DesignConstants.SPACING.SM,
            bottom: this.getListBottomPadding()
          })
          .scrollBar(BarState.Auto)
          .onReachEnd(() => { this.loadNearbyGroups(true); })
        }
        .layoutWeight(1)
        .onRefreshing(() => { this.loadNearbyGroups(false, true); })
        .transition(
          TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
            .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
        )
      }
    }
    .width('100%')
    .height('100%')
  }

  @Builder
  buildGroupCard(group: GroupItem) {
    Column({ space: DesignConstants.SPACING.MD }) {
      Row({ space: 6 }) {
        Row({ space: DesignConstants.SPACING.XS }) {
          Text() {
            SymbolSpan($r(AppIcon.GROUP))
              .fontSize(10)
              .fontColor([this.getCurrentThemePalette().accentBackground])
          }
          Text('附近群组')
            .fontSize(10)
            .fontColor(this.getCurrentThemePalette().textSecondary)
        }
        .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM, top: 3, bottom: 3 })
        .backgroundColor(this.getSurfaceSubtle())
        .borderRadius(DesignConstants.RADIUS.XL)

        Blank().layoutWeight(1)

        if ((group.location as RelativeLocation)?.distance !== undefined &&
          (group.location as RelativeLocation)?.distance !== null) {
          Text(LocationUtil.formatDistance((group.location as RelativeLocation).distance!))
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getCurrentThemePalette().accentBackground)
            .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM, top: 3, bottom: 3 })
            .backgroundColor(this.getSurfaceSubtle())
            .borderRadius(DesignConstants.RADIUS.XL)
        }
      }
      .width('100%')

      Row({ space: DesignConstants.SPACING.SM }) {
        Row({ space: DesignConstants.SPACING.MD }) {
          SystemAvatar({
            avatarUrl: group.group?.avatar_url ?? '',
            name: group.group?.name ?? '',
            avatarSize: 48,
            shape: 'circle'
          })

          Column({ space: DesignConstants.SPACING.XS }) {
            Text(group.group?.name ?? '')
              .fontSize(15)
              .fontWeight(FontWeight.Medium)
              .fontColor(this.getCurrentThemePalette().textPrimary)
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })

            Row({ space: 6 }) {
              Text(`${formatCount(group.group?.members_count)} 人`)
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .fontColor(this.getCurrentThemePalette().textSecondary)
              if (group.has_password) {
                Text('需口令')
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor(this.getCurrentThemePalette().textSecondary)
              }
            }
            .alignItems(VerticalAlign.Center)
          }
          .alignItems(HorizontalAlign.Start)
          .layoutWeight(1)
        }
        .layoutWeight(1)

        Row() {
          Text() {
            SymbolSpan($r(AppIcon.MORE))
              .fontSize(DesignConstants.FONT.SIZE.LG)
              .fontColor([this.getCurrentThemePalette().textSecondary])
          }
        }
        .width(30)
        .height(30)
        .borderRadius(DesignConstants.RADIUS.XL)
        .backgroundColor(this.getSurfaceSubtle())
        .border({ width: 1, color: this.getSurfaceBorder() })
        .justifyContent(FlexAlign.Center)
        .alignItems(VerticalAlign.Center)
        .bindMenu([
          {
            value: '查看详情',
            action: () => this.openGroupDetail(group)
          },
          {
            value: this.isGroupJoined(group) ? '进入群聊' : '加入群组',
            action: () => this.joinGroup(group)
          },
          {
            value: '举报群组',
            action: () => this.reportGroup(group)
          },
          {
            value: '复制群组ID',
            action: () => this.copyText(group.group?.group_id ?? '', '群组ID已复制')
          }
        ])
      }
      .width('100%')

      if (group.description) {
        Text(group.description)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .lineHeight(17)
          .fontColor(this.getCurrentThemePalette().textSecondary)
          .maxLines(2)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
          .width('100%')
      } else {
        Text('这个群还没有群介绍')
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .lineHeight(17)
          .fontColor(this.getCurrentThemePalette().textSecondary)
          .maxLines(2)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
          .width('100%')
      }

      if (group.tags && group.tags.length > 0) {
        Flex({ wrap: FlexWrap.Wrap }) {
          ForEach(group.tags.slice(0, 3), (tag: string) => {
            Text(`#${tag}`)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor(this.getCurrentThemePalette().textSecondary)
            .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM, top: 3, bottom: 3 })
            .backgroundColor(this.getSurfaceSubtle())
            .borderRadius(DesignConstants.RADIUS.XL)
            .margin({ right: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
        }, (tag: string) => tag)
      }
      .width('100%')
      }

      Row({ space: DesignConstants.SPACING.SM }) {
        Button(this.isGroupActionPending(group.group?.group_id ?? '') ? '处理中...' :
          (this.isGroupJoined(group) ? '进入群聊' : '加入群组'))
          .type(ButtonType.Capsule)
          .layoutWeight(1)
          .height(40)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .enabled(!this.isGroupActionPending(group.group?.group_id ?? ''))
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontWeight(FontWeight.Medium)
          .backgroundColor(this.isGroupJoined(group) ? this.getSurfaceSubtle() : this.getCurrentThemePalette().accentBackground)
          .fontColor(this.isGroupJoined(group) ? this.getCurrentThemePalette().accentBackground : this.getCurrentThemePalette().accentText)
          .shadow(this.isGroupJoined(group) ? { radius: 0, color: '#00000000', offsetX: 0, offsetY: 0 } : this.getCurrentThemePalette().controlShadow)
          .border(this.isGroupJoined(group) ? { width: 0 } : { width: 1, color: this.getSurfaceBorder() })
          .opacity(this.isGroupActionPending(group.group?.group_id ?? '') ? 0.58 : 1)
          .onClick(() => { this.joinGroup(group); })

        Button('详情')
          .type(ButtonType.Capsule)
          .layoutWeight(1)
          .height(40)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontWeight(FontWeight.Medium)
          .backgroundColor(this.getSurfaceSubtle())
          .fontColor(this.getCurrentThemePalette().accentBackground)
          .border({ width: 1, color: this.getSurfaceBorder() })
          .onClick(() => { this.openGroupDetail(group); })
      }
      .width('100%')
    }
    .width('100%')
    .alignItems(HorizontalAlign.Start)
    .padding({
      left: DesignConstants.SPACING.MD,
      right: DesignConstants.SPACING.MD,
      top: DesignConstants.SPACING.MD,
      bottom: DesignConstants.SPACING.MD
    })
    .backgroundColor(this.getSurfaceBackground())
    .borderRadius(DesignConstants.RADIUS.XXL)
    .shadow(this.getCurrentThemePalette().elevatedShadow)
    .border({ width: 1, color: this.getSurfaceBorder() })
    .margin({ bottom: DesignConstants.SPACING.MD })
  }

  // ==================== 操作方法 ====================
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

  private isUserFollowing(userId: string): boolean {
    const state = this.userStateMap[userId];
    if (state?.relation) {
      return state.relation.is_following;
    }
    return false;
  }

  private async joinGroup(group: GroupItem): Promise<void> {
    if (!this.isLoggedIn) {
      SystemToast.showWarning('请先登录后再加入群组', this.getUIContext());
      const loginParams = new LoginRedirectParams();
      loginParams.reason = 'action_required';
      loginParams.action = '加入群组';
      AppRouter.push(PageNames.LOGIN, loginParams);
      return;
    }

    const groupId = group.group?.group_id ?? '';
    if (!groupId) return;
    if (this.isGroupJoined(group)) {
      this.enterGroupChat(group);
      return;
    }
    if (this.isGroupActionPending(groupId)) return;
    this.setGroupActionPending(groupId, true);

    try {
      const joinReq: JoinGroupRequest = { group_id: groupId };
      const response = await ApiClient.joinGroup(joinReq);
      if (response.code === 0 && response.data) {
        if (response.data.status === 'joined') {
          InteractionStateStore.getInstance().updateGroupMembership(groupId, { is_member: true });
          SystemToast.showSuccess('加入成功', this.getUIContext());
          await this.loadNearbyGroups();
        } else if (response.data.status === 'pending_approval') {
          SystemToast.show('已提交申请，等待审批', this.getUIContext());
        }
      } else if (response.code === 10400) {
        SystemToast.show(response.message || '加入失败', this.getUIContext());
      } else {
        SystemToast.showError(response.message || '加入失败', this.getUIContext());
      }
    } catch {
      SystemToast.showError('加入群组失败', this.getUIContext());
    } finally {
      this.setGroupActionPending(groupId, false);
    }
  }

  private openUserProfile(user: UserItem): void {
    const userId = user.user?.user_id;
    if (!userId) {
      return;
    }
    const params = new ProfileRouteParam();
    params.userId = userId;
    AppRouter.push(PageNames.PROFILE, params);
  }

  private startPrivateChat(user: UserItem): void {
    if (!this.isLoggedIn) {
      SystemToast.showWarning('请先登录后再发起私聊', this.getUIContext());
      const loginParams = new LoginRedirectParams();
      loginParams.reason = 'action_required';
      loginParams.action = '发起私聊';
      AppRouter.push(PageNames.LOGIN, loginParams);
      return;
    }
    const userId = user.user?.user_id;
    if (!userId) return;
    const params = new ChatRouteParam();
    params.chatId = userId;
    params.chatType = 'private_chat';
    params.targetId = userId;
    params.title = user.user?.nickname ?? '私聊';
    AppRouter.push(PageNames.CHAT, params);
  }

  private enterGroupChat(group: GroupItem): void {
    if (!this.isLoggedIn) {
      SystemToast.showWarning('请先登录后再进入群聊', this.getUIContext());
      const loginParams = new LoginRedirectParams();
      loginParams.reason = 'action_required';
      loginParams.action = '进入群聊';
      AppRouter.push(PageNames.LOGIN, loginParams);
      return;
    }
    const groupId = group.group?.group_id;
    if (!groupId) return;
    const params = new ChatRouteParam();
    params.chatId = groupId;
    params.chatType = 'group_chat';
    params.title = group.group?.name ?? '群聊';
    AppRouter.push(PageNames.CHAT, params);
  }

  private async reportUser(user: UserItem): Promise<void> {
    const userId = user.user?.user_id;
    if (!userId) return;
    if (!this.isLoggedIn) {
      SystemToast.showWarning('请先登录后再举报', this.getUIContext());
      const loginParams = new LoginRedirectParams();
      loginParams.reason = 'action_required';
      loginParams.action = '举报用户';
      AppRouter.push(PageNames.LOGIN, loginParams);
      return;
    }
    try {
      const req: CreateReportRequest = {
        target_type: 'user',
        target_id: userId,
        reason: 'other'
      };
      const response = await ApiClient.createReport(req);
      if (response.code === 0) {
        SystemToast.showSuccess('举报已提交', this.getUIContext());
      } else {
        SystemToast.showError(response.message || '举报失败', this.getUIContext());
      }
    } catch {
      SystemToast.showError('举报失败', this.getUIContext());
    }
  }

  private async reportGroup(group: GroupItem): Promise<void> {
    const groupId = group.group?.group_id;
    if (!groupId) return;
    if (!this.isLoggedIn) {
      SystemToast.showWarning('请先登录后再举报', this.getUIContext());
      const loginParams = new LoginRedirectParams();
      loginParams.reason = 'action_required';
      loginParams.action = '举报群组';
      AppRouter.push(PageNames.LOGIN, loginParams);
      return;
    }
    try {
      const req: CreateReportRequest = {
        target_type: 'group',
        target_id: groupId,
        reason: 'other'
      };
      const response = await ApiClient.createReport(req);
      if (response.code === 0) {
        SystemToast.showSuccess('举报已提交', this.getUIContext());
      } else {
        SystemToast.showError(response.message || '举报失败', this.getUIContext());
      }
    } catch {
      SystemToast.showError('举报失败', this.getUIContext());
    }
  }

  private async blockUser(user: UserItem): Promise<void> {
    const userId = user.user?.user_id;
    if (!userId) return;
    if (!this.isLoggedIn) {
      SystemToast.showWarning('请先登录后再拉黑', this.getUIContext());
      return;
    }
    if (this.isUserActionPending(userId)) return;
    this.setUserActionPending(userId, true);
    try {
      const response = await ApiClient.blockUser({ user_id: userId } as BlockRequest);
      if (response.code === 0) {
        this.patchUserBlocked(userId, true);
        SystemToast.showSuccess('已拉黑', this.getUIContext());
      } else {
        SystemToast.showError(response.message || '拉黑失败', this.getUIContext());
      }
    } catch {
      SystemToast.showError('拉黑失败', this.getUIContext());
    } finally {
      this.setUserActionPending(userId, false);
    }
  }

  private async unblockUser(user: UserItem): Promise<void> {
    const userId = user.user?.user_id;
    if (!userId) return;
    if (!this.isLoggedIn) {
      SystemToast.showWarning('请先登录后再取消拉黑', this.getUIContext());
      return;
    }
    if (this.isUserActionPending(userId)) return;
    this.setUserActionPending(userId, true);
    try {
      const response = await ApiClient.unblockUser({ user_id: userId } as UnblockRequest);
      if (response.code === 0) {
        this.patchUserBlocked(userId, false);
        SystemToast.showSuccess('已取消拉黑', this.getUIContext());
      } else {
        SystemToast.showError(response.message || '取消拉黑失败', this.getUIContext());
      }
    } catch {
      SystemToast.showError('取消拉黑失败', this.getUIContext());
    } finally {
      this.setUserActionPending(userId, false);
    }
  }

  private ensureUserRelation(user: UserItem): Relationship {
    if (!user.relation) {
      const relation: Relationship = {
        is_following: false,
        is_followed_by: false,
        is_blocked: false
      };
      user.relation = relation;
    }
    return user.relation;
  }

  private patchUserFollowing(userId: string, isFollowing: boolean): void {
    const users = this.users;
    for (let i = 0; i < users.length; i++) {
      if ((users[i].user?.user_id ?? '') !== userId) {
        continue;
      }
      const relation = this.ensureUserRelation(users[i]);
      relation.is_following = isFollowing;
      break;
    }
    this.users = users;
    this.usersDataSource.setData(users);
  }

  private patchUserBlocked(userId: string, isBlocked: boolean): void {
    const users = this.users;
    for (let i = 0; i < users.length; i++) {
      if ((users[i].user?.user_id ?? '') !== userId) {
        continue;
      }
      const relation = this.ensureUserRelation(users[i]);
      relation.is_blocked = isBlocked;
      break;
    }
    this.users = users;
    this.usersDataSource.setData(users);
  }

  private setUserActionPending(userId: string, pending: boolean): void {
    const next: string[] = [];
    for (let i = 0; i < this.userActionPendingIds.length; i++) {
      const id = this.userActionPendingIds[i];
      if (id !== userId) {
        next.push(id);
      }
    }
    if (pending) {
      next.push(userId);
    }
    this.userActionPendingIds = next;
  }

  private setGroupActionPending(groupId: string, pending: boolean): void {
    const next: string[] = [];
    for (let i = 0; i < this.groupActionPendingIds.length; i++) {
      const id = this.groupActionPendingIds[i];
      if (id !== groupId) {
        next.push(id);
      }
    }
    if (pending) {
      next.push(groupId);
    }
    this.groupActionPendingIds = next;
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

  private openGroupDetail(group: GroupItem): void {
    if (!group.group?.group_id) return;
    const params = new GroupDetailParams();
    params.groupId = group.group.group_id;
    AppRouter.push(PageNames.GROUP_DETAIL, params);
  }

}
