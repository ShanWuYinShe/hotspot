/**
 * 附近的人 - 独立列表页
 * 对齐前端 /nearby/people：2 列 3:4 用户图片卡墙（桌面 3 列）+ 距离徽章 + 行内关注，
 * 搜索/排序/统计行保留；下拉刷新 + 上拉分页。数据来自 users/search。
 */

import { GlobalStateManager, KEY_BROWSE_ANCHOR, KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { PageShellContainer } from '../../../components/PageShellContainer';
import { isDesktopForm, BreakpointType } from '../../../core/responsive/Breakpoint';
import { LocationInfo, LocationUtil } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { AppRouter } from '../../../core/router/AppRouter';
import { PageNames, DefaultConfig } from '../../../constants/AppConstants';
import { UserItem, GeoPoint, SearchUsersRequest } from '../../../core/types/api';
import { UserCard } from '../components/UserCard';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemInput } from '../../../components/SystemInput';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { EmptyState, RefreshIndicator, LoadingMoreIndicator } from '../../../components/EmptyState';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { TabsPillGroup } from '../../../components/TabsPillGroup';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';

@Preview
@Component
export struct NearbyUsersListPage {
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;

  @State users: UserItem[] = [];
  @State isLoading: boolean = true;
  @State isRefreshing: boolean = false;
  @State isLoadingMore: boolean = false;
  @State hasMore: boolean = false;
  @State nextCursor: string = '';
  @State errorMessage: string = '';
  @State keyword: string = '';
  @State activeSort: string = 'distance';
  // 「在线」tab：users/search 接口无在线筛选/排序参数，但 UserItem 带 presence_status，
  // 因此「在线」为客户端过滤开关（不改变服务端 sort_by）
  @State onlineOnly: boolean = false;
  private usersDataSource: BasicDataSource<UserItem> = new BasicDataSource<UserItem>([]);
  private isDestroyed: boolean = false;
  private loadVersion: number = 0;

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getSurfaceBorder(): ResourceColor {
    return this.getCurrentThemePalette().borderColor;
  }

  private getReferencePoint(): GeoPoint | null {
    const ref = this.browseAnchor ?? this.currentLocation;
    if (ref) {
      return { latitude: ref.latitude, longitude: ref.longitude };
    }
    return null;
  }

  /** 位置行副标题（对齐前端 nearby 页 subtitle：{locationName} · 5km 内 / 5km 内） */
  private getLocationSubtitle(): string {
    const name = (this.browseAnchor ?? this.currentLocation)?.locationName ?? '';
    return name.length > 0 ? `${name} · 5km 内` : '5km 内';
  }

  aboutToAppear() {
    this.loadUsers();
  }

  aboutToDisappear() {
    this.isDestroyed = true;
  }

  private hasAnyData(): boolean {
    return this.users.length > 0;
  }

  /** 刷新列表数据源：onlineOnly 时按 presence_status 客户端过滤在线用户 */
  private refreshUsersDataSource(): void {
    if (this.onlineOnly) {
      this.usersDataSource.setData(this.users.filter((u: UserItem) => u.presence_status === 'online'));
    } else {
      this.usersDataSource.setData(this.users);
    }
  }

  private async loadUsers(refresh: boolean = false): Promise<void> {
    const ref = this.getReferencePoint();
    if (!ref) {
      this.isLoading = false;
      this.isRefreshing = false;
      this.errorMessage = '位置信息不可用，请在首页选择浏览位置';
      return;
    }
    // 请求版本号：每次进入递增，避免较慢的旧请求解析后覆盖较新的结果
    const capturedVersion = ++this.loadVersion;
    if (!refresh) { this.isLoading = true; }
    this.errorMessage = '';
    try {
      const me = GlobalStateManager.getInstance().getCurrentUser();
      const myId = me?.user_id;
      const request: SearchUsersRequest = {
        reference_point: ref,
        radius: DefaultConfig.NEARBY_RADIUS,
        sort_by: this.activeSort as 'distance' | 'relevance' | 'newest',
        keyword: this.keyword || undefined,
        pagination: { limit: DefaultConfig.PAGE_SIZE_LARGE }
      };
      const response = await LoadingUtil.withMinimumTime(() => ApiClient.searchUsers(request));
      if (this.isDestroyed) return;
      // 旧请求晚于新请求返回：丢弃结果，避免覆盖最新数据
      if (capturedVersion !== this.loadVersion) return;
      if (response.code === 0) {
        const items = (response.data?.items ?? []).filter((u: UserItem) => u.user?.user_id);
        this.users = typeof myId === 'string'
          ? items.filter((u: UserItem) => u.user!.user_id !== myId)
          : items;
        this.refreshUsersDataSource();
        this.hasMore = !!(response.data?.next_cursor);
        this.nextCursor = response.data?.next_cursor ?? '';
      } else {
        this.errorMessage = getErrorMessage(response, '加载失败');
        this.users = [];
        this.usersDataSource.clear();
      }
    } catch (error) {
      // 仅当当前请求仍为最新时才上报错误
      if (capturedVersion === this.loadVersion) {
        this.errorMessage = '加载附近用户失败';
      }
    } finally {
      // 仅当前最新请求可重置加载态，否则会把新请求的 loading 提前置 false
      if (capturedVersion === this.loadVersion) {
        this.isLoading = false;
        this.isRefreshing = false;
      }
    }
  }

  private async loadMoreUsers(): Promise<void> {
    if (this.isLoadingMore || !this.hasMore) return;
    const ref = this.getReferencePoint();
    if (!ref) return;
    this.isLoadingMore = true;
    try {
      const request: SearchUsersRequest = {
        reference_point: ref,
        radius: DefaultConfig.NEARBY_RADIUS,
        sort_by: this.activeSort as 'distance' | 'relevance' | 'newest',
        keyword: this.keyword || undefined,
        pagination: { limit: DefaultConfig.PAGE_SIZE_LARGE, cursor: this.nextCursor }
      };
      const response = await ApiClient.searchUsers(request);
      if (this.isDestroyed) return;
      if (response.code === 0) {
        const me = GlobalStateManager.getInstance().getCurrentUser();
        const myId = me?.user_id;
        const items = (response.data?.items ?? []).filter((u: UserItem) => u.user?.user_id);
        const filtered = typeof myId === 'string'
          ? items.filter((u: UserItem) => u.user!.user_id !== myId)
          : items;
        this.users = this.users.concat(filtered);
        // onlineOnly 模式下仅追加在线用户，保持当前过滤视图一致
        this.usersDataSource.appendData(
          this.onlineOnly ? filtered.filter((u: UserItem) => u.presence_status === 'online') : filtered
        );
        this.hasMore = !!(response.data?.next_cursor);
        this.nextCursor = response.data?.next_cursor ?? '';
      }
    } catch (error) {
      // 静默失败，用户可再次触底重试
    } finally {
      this.isLoadingMore = false;
    }
  }

  private getGridColumnCount(): number {
    // 对齐前端 /nearby/people gridColumns：mobile/tablet 2 列，desktop(≥1280) 3 列
    return isDesktopForm(this.currentBreakpoint) ? 3 : 2;
  }

  build() {
    NavDestination() {
    PageShellContainer({ variant: 'default', currentBreakpoint: this.currentBreakpoint, noHorizontalPadding: true }) {
      Column() {
        // 头部 —— 统一走 SystemPageHeader（返回 + 标题 24pt + 位置行副标题 + 计数 trailing，
        // 对齐前端 /nearby/people PageHeader：titleSize 2xl + MapPin 图标 + {位置} · 5km 内 + trailing N 人）
        SystemPageHeader({
          title: '附近的人',
          subtitle: this.getLocationSubtitle(),
          subtitleIcon: AppIcon.LOCATION,
          titleSize: DesignConstants.FONT.SIZE.XXL,
          titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM,
          showBackButton: true,
          onBack: () => { AppRouter.pop(); },
          backgroundVariant: 'transparent'
        }) {
          if (this.hasAnyData()) {
            Text(`${this.users.length} 人`)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor(this.getCurrentThemePalette().textSecondary)
          }
        }

        // 搜索栏（对齐前端 /users/nearby 搜索框）
        SystemInput({
          value: this.keyword,
          placeholder: '搜索附近用户',
          prefixIcon: AppIcon.SEARCH,
          showPrefixIcon: true,
          inputHeight: DesignConstants.COMPONENT.INPUT_HEIGHT,
          onChange: (value: string) => { this.keyword = value; },
          onSubmit: () => { this.loadUsers(true); }
        })
          .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.SM })
        .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })

        // 排序 TabsPillGroup（对齐前端 TabBar）
        TabsPillGroup({
          tabs: [{ label: '距离' }, { label: '在线' }, { label: '共同好友' }],
          activeIndex: this.onlineOnly ? 1 : (this.activeSort === 'relevance' ? 2 : 0),
          actions: {
            onChange: (index: number) => {
              // 「在线」非服务端排序（接口无在线参数），仅切换客户端过滤，无需重新请求
              if (index === 1) {
                if (!this.onlineOnly) {
                  this.onlineOnly = true;
                  this.refreshUsersDataSource();
                }
                return;
              }
              const sortMap: string[] = ['distance', 'distance', 'relevance'];
              const newSort = sortMap[index] || 'distance';
              const needReload = newSort !== this.activeSort;
              this.onlineOnly = false;
              if (needReload) {
                this.activeSort = newSort;
                this.loadUsers(true);
              } else {
                // 排序未变，仅退出在线过滤，直接恢复完整列表
                this.refreshUsersDataSource();
              }
            }
          }
        })
          .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.SM })

        // 内容区
        if (this.isLoading) {
          SystemLoadingView({ message: '正在加载附近的人...' })
            .layoutWeight(1)
        } else if (!this.hasAnyData() && this.errorMessage) {
          EmptyState({
            icon: AppIcon.WARNING,
            message: this.errorMessage,
            actionLabel: '重试',
            onAction: () => { this.loadUsers(); }
          })
            .layoutWeight(1)
        } else if (!this.hasAnyData()) {
          EmptyState({
            icon: AppIcon.COMPASS,
            title: '附近暂无其他用户',
            message: '可稍后再来，或发布一条动态让附近的人发现你',
            fillMode: true
          })
            .layoutWeight(1)
        } else {
          Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
            Column() {
              Grid() {
                LazyForEach(this.usersDataSource, (user: UserItem) => {
                  GridItem() {
                    UserCard({ user: user })
                  }
                }, (user: UserItem) => user.user?.user_id ?? '')
              }
              .width('100%')
              .layoutWeight(1)
              .columnsTemplate(this.getGridColumnCount() === 3 ? '1fr 1fr 1fr' : '1fr 1fr')
              .columnsGap(DesignConstants.SPACING.LG)
              .rowsGap(DesignConstants.SPACING.LG)
              .cachedCount(4)
              .scrollBar(BarState.Auto)
              .onReachEnd(() => { this.loadMoreUsers(); })

              // 分页状态条（Grid 外常驻，规避 GridItem 跨列占位）
              if (this.hasMore) {
                LoadingMoreIndicator()
              } else {
                Text('没有更多了')
                  .fontSize(DesignConstants.FONT.SIZE.XS)
                  .fontColor(this.getCurrentThemePalette().textTertiary)
                  .width('100%')
                  .textAlign(TextAlign.Center)
                  .padding({ top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.LG })
              }
            }
            .width('100%')
            .height('100%')
          }
          .layoutWeight(1)
          .onRefreshing(() => { this.loadUsers(true); })
        }
      }
      .width('100%')
      .height('100%')
    }
    }
    .hideTitleBar(true)
  }
}

@Builder
export function NearbyUsersListPageBuilder() {
  NearbyUsersListPage()
}