/**
 * 附近页面 - 主标签页
 * 包含动态、用户、群组三个子标签，顶部位置栏和搜索/地图入口
 * 类似抖音风格的子标签切换
 */

import { GlobalStateManager, KEY_IS_LOGGED_IN, KEY_BROWSE_ANCHOR, KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { LocationInfo, LocationUtil } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { AppRouter, ChatRouteParam, LoginRouteParam, MomentDetailRouteParam, ProfileRouteParam } from '../../../core/router/AppRouter';
import { PageNames, DefaultConfig } from '../../../constants/AppConstants';
import {
  MomentItem, UserItem, GroupItem, GeoPoint, RelativeLocation,
  SearchMomentsRequest, SearchUsersRequest, SearchGroupsRequest,
  MomentImage, JoinGroupRequest,
  BlockRequest, UnblockRequest, CreateReportRequest, Relationship
} from '../../../core/types/api';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { MomentFeedCard } from '../../../components/MomentFeedCard';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { InteractionStateStore, UserRelationState } from '../../../core/state/InteractionStateStore';
import { interactionActions } from '../../../core/state/InteractionActions';
import { EmptyState } from '../../../components/EmptyState';
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
import { AppThemeVisuals } from '../../../core/theme/AppThemeVisuals';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';


const DOMAIN_NUMBER: number = 0xFF10;
const TAG: string = 'NearbyPage';

class GroupDetailParams {
  groupId: string = '';
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
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @StorageLink('interaction_user_state') userStateMap: Record<string, UserRelationState> = {};
  @StorageProp('isTablet') isTablet: boolean = false;

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
  private isDestroyed: boolean = false;

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
    this.isDestroyed = true;
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

  /** 判断当前是否无数据（用于决定 location 变化时是否需要自动加载） */
  private isCurrentTabEmpty(): boolean {
    return this.moments.length === 0 && this.users.length === 0 && this.groups.length === 0;
  }

  private getReferencePoint(): GeoPoint | null {
    const loc: LocationInfo | null = this.browseAnchor ?? this.currentLocation;
    if (!loc) return null;
    const point: GeoPoint = { latitude: loc.latitude, longitude: loc.longitude };
    return point;
  }

  private async loadCurrentTabData(refresh: boolean = false): Promise<void> {
    this.loadNearbyMoments(false, refresh);
    this.loadNearbyUsers(false, refresh);
    this.loadNearbyGroups(false, refresh);
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
      if (this.isDestroyed) return;
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
      if (this.isDestroyed) return;
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
      if (this.isDestroyed) return;
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
    Scroll() {
      Column() {
        this.buildHeroBanner()

        if (this.users.length > 0) {
          this.buildNearbyUsersHorizontal()
        }

        if (this.groups.length > 0) {
          this.buildActivityGrid()
        }

        if (this.moments.length > 0) {
          this.buildMomentsSection()
        }

        if (this.users.length === 0 && this.groups.length === 0 && this.moments.length === 0) {
          if (this.momentsLoading || this.usersLoading || this.groupsLoading) {
            SystemLoadingView({ message: '正在加载附近内容...' })
          } else if (this.momentsError.length > 0) {
            EmptyState({ icon: AppIcon.WARNING, message: this.momentsError, actionLabel: '重试', onAction: () => { this.loadNearbyMoments(); this.loadNearbyUsers(); this.loadNearbyGroups(); } })
          } else {
            EmptyState({ icon: AppIcon.NEARBY, message: '附近暂无内容\n发布第一条动态吧！' })
          }
        }
      }
      .width('100%')
      .constraintSize({ maxWidth: this.isTablet ? 720 : Infinity })
    }
    .width('100%')
    .height('100%')
    .scrollBar(BarState.Auto)
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
  }

  @Builder
  private HiddenTabBar() {
    Blank()
      .width(0)
      .height(0)
  }

  @Builder
  private buildHeroBanner() {
    Stack() {
      Column()
        .width('100%')
        .height(this.isTablet ? 260 : 200)
        .borderRadius(28)
        .linearGradient({
          direction: GradientDirection.BottomRight,
          colors: [[this.getCurrentThemePalette().accentBackground, 0.0],
            [this.getCurrentThemePalette().accentMuted, 0.6],
            [this.getCurrentThemePalette().surfaceSubtle, 1.0]]
        })

      Column() {
        Row() {
          Column({ space: 4 }) {
            Text('附近')
              .fontSize(22)
              .fontWeight(FontWeight.Bold)
              .fontColor(Color.White)
            Row({ space: 6 }) {
              Text() {
                SymbolSpan($r(AppIcon.LOCATION))
                  .fontSize(13)
                  .fontColor([Color.White])
              }
              Text(this.browseAnchor?.locationName ?? this.currentLocation?.locationName ?? '定位中...')
                .fontSize(13)
                .fontColor('rgba(255,255,255,0.85)')
            }
          }
          .alignItems(HorizontalAlign.Start)

          Blank()

          Column({ space: 2 }) {
            Text(this.nearbyDeviceCount > 0 ? `${this.nearbyDeviceCount}` : `${this.users.length}`)
              .fontSize(24)
              .fontWeight(FontWeight.Bold)
              .fontColor(Color.White)
            Text('人附近')
              .fontSize(10)
              .fontColor('rgba(255,255,255,0.8)')
          }
          .padding({ left: 16, right: 16, top: 12, bottom: 12 })
          .borderRadius(18)
          .backgroundColor('rgba(255,255,255,0.18)')
          .border({ width: 1, color: 'rgba(255,255,255,0.28)' })
        }
        .width('100%')
        .alignItems(VerticalAlign.Top)

        Blank()

        Row({ space: 12 }) {
          ForEach([
            { label: '活动中', value: `${this.groups.filter((g: GroupItem) => g.my_state?.is_joined).length}` },
            { label: '在线', value: `${this.users.filter((u: UserItem) => u.presence_status === 'online').length}` },
            { label: '动态', value: `${this.moments.length}` }
          ], (stat: { label: string; value: string }) => {
            Column({ space: 1 }) {
              Text(stat.value)
                .fontSize(16)
                .fontWeight(FontWeight.Bold)
                .fontColor(Color.White)
              Text(stat.label)
                .fontSize(10)
                .fontColor('rgba(255,255,255,0.75)')
            }
            .padding({ left: 12, right: 12, top: 8, bottom: 8 })
            .borderRadius(14)
            .backgroundColor('rgba(255,255,255,0.15)')
            .layoutWeight(1)
            .alignItems(HorizontalAlign.Center)
          }, (stat: { label: string }) => stat.label)
        }
        .width('100%')
      }
      .width('100%')
      .height(200)
      .padding({ left: 20, right: 20, top: 20, bottom: 20 })
      .borderRadius(28)
    }
    .width('100%')
    .margin({ left: 16, right: 16, top: this.getHeaderTopPadding(), bottom: 20 })
    .alignContent(Alignment.Center)
    .shadow(this.getCurrentThemePalette().elevatedShadow)
  }

  @Builder
  private buildNearbyUsersHorizontal() {
    Column() {
      Row() {
        Text('附近的人')
          .fontSize(16)
          .fontWeight(FontWeight.Medium)
          .fontColor(this.getCurrentThemePalette().textPrimary)

        Blank()

        Row({ space: 4 }) {
          Text('全部')
            .fontSize(12)
            .fontColor(this.getCurrentThemePalette().accentBackground)
          Text() {
            SymbolSpan($r(AppIcon.ARROW_RIGHT))
              .fontSize(13)
              .fontColor([this.getCurrentThemePalette().accentBackground])
          }
        }
      }
      .width('100%')
      .margin({ bottom: 12 })

      Scroll() {
        Row({ space: 12 }) {
          ForEach(this.users.slice(0, 10), (user: UserItem) => {
            this.buildHorizontalUserCard(user)
          }, (user: UserItem) => user.user?.user_id ?? '')
        }
      }
      .scrollable(ScrollDirection.Horizontal)
      .scrollBar(BarState.Off)
      .width('100%')
    }
    .width('100%')
    .padding({ left: 16, right: 16 })
    .margin({ bottom: 20 })
  }

  @Builder
  private buildHorizontalUserCard(user: UserItem) {
    Column({ space: 8 }) {
      Stack() {
        SystemAvatar({
          avatarUrl: user.user?.avatar_url ?? '',
          name: user.user?.nickname ?? '',
          avatarSize: 66,
          shape: 'squircle'
        })

        if (user.presence_status === 'online') {
          Row()
            .width(12)
            .height(12)
            .borderRadius(DesignConstants.RADIUS.FULL)
            .backgroundColor('#22c55e')
            .border({ width: 2, color: this.getCurrentThemePalette().surfaceBackground })
            .position({ x: 50, y: 50 })
        }

        if (user.location?.distance !== undefined && user.location?.distance !== null) {
          Row() {
            Text(LocationUtil.formatDistance(user.location.distance))
              .fontSize(9)
              .fontColor(Color.White)
          }
          .padding({ left: 8, right: 8, top: 2, bottom: 2 })
          .borderRadius(DesignConstants.RADIUS.FULL)
          .backgroundColor(this.getCurrentThemePalette().accentBackground)
          .shadow({ radius: 8, color: this.getCurrentThemePalette().accentShadow.color, offsetX: 0, offsetY: 2 })
          .position({ x: 10, y: 62 })
        }
      }
      .width(70)
      .height(76)

      Text(user.user?.nickname ?? '')
        .fontSize(12)
        .fontColor(this.getCurrentThemePalette().textPrimary)
        .maxLines(1)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
        .constraintSize({ maxWidth: 70 })
    }
    .alignItems(HorizontalAlign.Center)
    .onClick(() => {
      this.openUserProfile(user);
    })
  }

  @Builder
  private buildActivityGrid() {
    Column() {
      Row() {
        Text('附近群组')
          .fontSize(16)
          .fontWeight(FontWeight.Medium)
          .fontColor(this.getCurrentThemePalette().textPrimary)

        Blank()

        Row({ space: 4 }) {
          Text('查看更多')
            .fontSize(12)
            .fontColor(this.getCurrentThemePalette().accentBackground)
          Text() {
            SymbolSpan($r(AppIcon.ARROW_RIGHT))
              .fontSize(13)
              .fontColor([this.getCurrentThemePalette().accentBackground])
          }
        }
      }
      .width('100%')
      .margin({ bottom: 12 })

      Flex({ wrap: FlexWrap.Wrap, space: { main: LengthMetrics.vp(12), cross: LengthMetrics.vp(12) } }) {
        ForEach(this.groups.slice(0, this.isTablet ? 6 : 4), (group: GroupItem) => {
          this.buildActivityCard(group)
        }, (group: GroupItem) => group.group?.group_id ?? '')
      }
      .width('100%')
    }
    .width('100%')
    .padding({ left: 16, right: 16 })
    .margin({ bottom: 20 })
  }

  @Builder
  private buildActivityCard(group: GroupItem) {
    Column() {
      Row() {
        Text() {
          SymbolSpan($r(AppIcon.GROUP))
            .fontSize(28)
            .fontColor([this.getCurrentThemePalette().accentBackground])
        }
      }
      .width('100%')
      .height(64)
      .justifyContent(FlexAlign.Center)
      .backgroundColor(this.getCurrentThemePalette().accentMuted)
      .borderRadius({ topLeft: 22, topRight: 22 })

      Column({ space: 4 }) {
        Text(group.group?.name ?? '')
          .fontSize(13)
          .fontWeight(FontWeight.Medium)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        Row({ space: 6 }) {
          Text(`${formatCount(group.group?.members_count)} 人`)
            .fontSize(11)
            .fontColor(this.getCurrentThemePalette().textSecondary)
          if (group.location?.distance !== undefined) {
            Text(LocationUtil.formatDistance(group.location.distance))
              .fontSize(11)
              .fontColor(this.getCurrentThemePalette().accentBackground)
          }
        }
      }
      .padding({ left: 12, right: 12, top: 12, bottom: 12 })
      .alignItems(HorizontalAlign.Start)
    }
    .width(this.isTablet ? 'calc(33.3% - 8vp)' : 'calc(50% - 6vp)')
    .borderRadius(22)
    .backgroundColor(this.getCurrentThemePalette().surfaceBackground)
    .border({ width: 1, color: this.getCurrentThemePalette().borderColor })
    .shadow(this.getCurrentThemePalette().elevatedShadow)
    .onClick(() => {
      this.openGroupDetail(group);
    })
  }

  @Builder
  private buildMomentsSection() {
    Column() {
      Row() {
        Text('附近动态')
          .fontSize(16)
          .fontWeight(FontWeight.Medium)
          .fontColor(this.getCurrentThemePalette().textPrimary)

        Blank()

        Row({ space: 4 }) {
          Text('更多')
            .fontSize(12)
            .fontColor(this.getCurrentThemePalette().accentBackground)
          Text() {
            SymbolSpan($r(AppIcon.ARROW_RIGHT))
              .fontSize(13)
              .fontColor([this.getCurrentThemePalette().accentBackground])
          }
        }
      }
      .width('100%')
      .margin({ bottom: 12 })

      Column({ space: DesignConstants.SPACING.MD }) {
        if (this.isTablet) {
          Flex({ wrap: FlexWrap.Wrap, space: { main: LengthMetrics.vp(12), cross: LengthMetrics.vp(12) } }) {
            ForEach(this.moments.slice(0, 6), (moment: MomentItem) => {
              Column() {
                MomentFeedCard({
                  moment: moment,
                  onOpenDetail: (item: MomentItem) => this.openMomentDetail(item)
                })
              }
              .width('calc(50% - 6vp)')
            }, (moment: MomentItem) => moment.moment_id)
          }
        } else {
          ForEach(this.moments.slice(0, 5), (moment: MomentItem) => {
            MomentFeedCard({
              moment: moment,
              onOpenDetail: (item: MomentItem) => this.openMomentDetail(item)
            })
          }, (moment: MomentItem) => moment.moment_id)
        }
      }
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
    .margin({ bottom: this.getListBottomPadding() })
  }

  private openMomentDetail(moment: MomentItem): void {
    hilog.info(DOMAIN_NUMBER, TAG, `打开动态详情: ${moment.moment_id}`);
    const params = new MomentDetailRouteParam();
    params.momentId = moment.moment_id;
    AppRouter.push(PageNames.MOMENT_DETAIL, params);
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
      const loginParams = new LoginRouteParam();
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
      const loginParams = new LoginRouteParam();
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
      const loginParams = new LoginRouteParam();
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
      const loginParams = new LoginRouteParam();
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
      const loginParams = new LoginRouteParam();
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
