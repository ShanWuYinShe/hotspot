/**
 * 附近的人 - 独立列表页
 * 对齐前端 /users/nearby：用户列表（距离/在线）+ 下拉刷新 + 上拉分页。
 * 数据来自 users/search（reference_point + radius + sort_by=distance）。
 */

import { GlobalStateManager, KEY_BROWSE_ANCHOR, KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { ResponsiveContainer } from '../../../components/ResponsiveContainer';
import { LocationInfo, LocationUtil } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { AppRouter, ProfileRouteParam } from '../../../core/router/AppRouter';
import { PageNames, DefaultConfig } from '../../../constants/AppConstants';
import { UserItem, GeoPoint, SearchUsersRequest } from '../../../core/types/api';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { EmptyState, RefreshIndicator, LoadingMoreIndicator } from '../../../components/EmptyState';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';

@Preview
@Component
export struct NearbyUsersListPage {
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';
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
  @State onlineCount: number = 0;
  private usersDataSource: BasicDataSource<UserItem> = new BasicDataSource<UserItem>([]);
  private isDestroyed: boolean = false;

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getHeaderTopPadding(): number {
    return SafeAreaUtils.resolveTopPadding(this.safeAreaTop);
  }

  private getSurfaceBorder(): ResourceColor {
    return this.getCurrentThemePalette().borderColor;
  }

  private getSortLabel(): string {
    const labelMap: Record<string, string> = { 'distance': '距离', 'relevance': '共同好友', 'newest': '最新' };
    return labelMap[this.activeSort] || '距离';
  }

  private getReferencePoint(): GeoPoint | null {
    const ref = this.browseAnchor ?? this.currentLocation;
    if (ref) {
      return { latitude: ref.latitude, longitude: ref.longitude };
    }
    return null;
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

  private async loadUsers(refresh: boolean = false): Promise<void> {
    const ref = this.getReferencePoint();
    if (!ref) {
      this.isLoading = false;
      this.isRefreshing = false;
      this.errorMessage = '位置信息不可用，请在首页选择浏览位置';
      return;
    }
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
      if (response.code === 0) {
        const items = (response.data?.items ?? []).filter((u: UserItem) => u.user?.user_id);
        this.users = typeof myId === 'string'
          ? items.filter((u: UserItem) => u.user!.user_id !== myId)
          : items;
        this.usersDataSource.setData(this.users);
        this.hasMore = !!(response.data?.next_cursor);
        this.nextCursor = response.data?.next_cursor ?? '';
        this.onlineCount = this.users.filter((u: UserItem) => u.presence_status === 'online').length;
      } else {
        this.errorMessage = response.message || '加载失败';
        this.users = [];
        this.usersDataSource.clear();
      }
    } catch (error) {
      this.errorMessage = '加载附近用户失败';
    } finally {
      this.isLoading = false;
      this.isRefreshing = false;
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
        sort_by: 'distance',
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
        this.usersDataSource.appendData(filtered);
        this.hasMore = !!(response.data?.next_cursor);
        this.nextCursor = response.data?.next_cursor ?? '';
      }
    } catch (error) {
      // 静默失败，用户可再次触底重试
    } finally {
      this.isLoadingMore = false;
    }
  }

  private openProfile(userId: string): void {
    const params = new ProfileRouteParam();
    params.userId = userId;
    AppRouter.push(PageNames.PROFILE, params);
  }

  @Builder
  private buildUserRow(user: UserItem) {
    Row({ space: 12 }) {
      Stack({ alignContent: Alignment.BottomEnd }) {
        SystemAvatar({
          avatarUrl: user.user?.avatar_url ?? '',
          name: user.user?.nickname ?? '',
          avatarSize: 48,
          shape: 'circle'
        })
        if (user.presence_status === 'online') {
          Row()
            .width(12)
            .height(12)
            .borderRadius(DesignConstants.RADIUS.FULL)
            .backgroundColor(this.getCurrentThemePalette().success)
            .border({ width: 2, color: this.getCurrentThemePalette().surfaceBackground })
        }
      }
      .width(48)
      .height(48)

      Column({ space: 4 }) {
        Text(user.user?.nickname ?? '')
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontWeight(FontWeight.Medium)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        Row({ space: 8 }) {
          if (user.presence_status === 'online') {
            Text('在线')
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor(this.getCurrentThemePalette().success)
          }
          if (user.location?.distance !== undefined && user.location?.distance !== null) {
            Text(LocationUtil.formatDistance(user.location.distance))
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor(this.getCurrentThemePalette().textSecondary)
          }
        }
        .alignItems(VerticalAlign.Center)
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)

      Text() {
        SymbolSpan($r(AppIcon.ARROW_RIGHT))
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor([this.getCurrentThemePalette().textTertiary])
      }
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: 10, bottom: 10 })
    .onClick(() => {
      const userId = user.user?.user_id;
      if (userId) {
        this.openProfile(userId);
      }
    })
  }

  build() {
    NavDestination() {
    ResponsiveContainer() {
      Column() {
        // 头部：返回 + 标题
        Row({ space: 8 }) {
          Text() {
            SymbolSpan($r(AppIcon.ARROW_LEFT))
              .fontSize(DesignConstants.FONT.SIZE.XL_PLUS)
              .fontColor([this.getCurrentThemePalette().textPrimary])
          }
          .onClick(() => AppRouter.pop())

          Text('附近的人')
            .fontSize(DesignConstants.FONT.SIZE.XL)
            .fontWeight(FontWeight.Medium)
            .fontColor(this.getCurrentThemePalette().textPrimary)
        }
        .width('100%')
        .height(48)
        .alignItems(VerticalAlign.Center)
        .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.LG })
        .margin({ top: this.getHeaderTopPadding() })

        // 搜索栏（对齐前端 /users/nearby 搜索框）
        Row({ space: 10 }) {
          Text() {
            SymbolSpan($r(AppIcon.SEARCH))
              .fontSize(DesignConstants.FONT.SIZE.MD_PLUS)
              .fontColor([this.getCurrentThemePalette().textSecondary])
          }
          TextInput({ placeholder: '搜索附近用户', text: $$this.keyword })
            .fontSize(DesignConstants.FONT.SIZE.SM_PLUS)
            .fontColor(this.getCurrentThemePalette().textPrimary)
            .placeholderColor(this.getCurrentThemePalette().textSecondary)
            .layoutWeight(1)
            .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
            .onSubmit(() => { this.loadUsers(true); })
        }
        .height(DesignConstants.COMPONENT.INPUT_HEIGHT)
        .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
        .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.SM })
        .backgroundColor(this.getCurrentThemePalette().inputBackgroundColor)
        .borderRadius(DesignConstants.SHAPE.INPUT)
        .border({ width: 1, color: this.getSurfaceBorder() })

        // 排序 chips（对齐前端 sort chips：距离/在线/共同好友）
        Row({ space: DesignConstants.SPACING.SM }) {
          ForEach(['距离', '在线', '共同好友'], (chip: string) => {
            Text(chip)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor(chip === this.getSortLabel() ? this.getCurrentThemePalette().accentText : this.getCurrentThemePalette().textSecondary)
              .padding({ left: 12, right: 12, top: 6, bottom: 6 })
              .borderRadius(DesignConstants.SHAPE.CHIP)
              .backgroundColor(chip === this.getSortLabel() ? this.getCurrentThemePalette().accentBackground : this.getCurrentThemePalette().surfaceSubtle)
              .onClick(() => {
                const sortMap: Record<string, string> = { '距离': 'distance', '在线': 'distance', '共同好友': 'relevance' };
                const newSort = sortMap[chip] || 'distance';
                if (newSort !== this.activeSort) {
                  this.activeSort = newSort;
                  this.loadUsers(true);
                }
              })
          }, (chip: string) => chip)
        }
        .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.SM })

        // Hero 统计卡（对齐前端「附近概览」）
        if (this.hasAnyData()) {
          Row({ space: DesignConstants.SPACING.MD }) {
            Column({ space: 2 }) {
              Text(`${this.onlineCount}`)
                .fontSize(DesignConstants.FONT.SIZE.XL)
                .fontWeight(FontWeight.Bold)
                .fontColor(this.getCurrentThemePalette().success)
              Text('在线')
                .fontSize(DesignConstants.FONT.SIZE.XS)
                .fontColor(this.getCurrentThemePalette().textSecondary)
            }
            .alignItems(HorizontalAlign.Center)
            .layoutWeight(1)

            Column()
              .width(1).height(28)
              .backgroundColor(this.getCurrentThemePalette().borderColor)

            Column({ space: 2 }) {
              Text(`${this.users.length}`)
                .fontSize(DesignConstants.FONT.SIZE.XL)
                .fontWeight(FontWeight.Bold)
                .fontColor(this.getCurrentThemePalette().textPrimary)
              Text('总数')
                .fontSize(DesignConstants.FONT.SIZE.XS)
                .fontColor(this.getCurrentThemePalette().textSecondary)
            }
            .alignItems(HorizontalAlign.Center)
            .layoutWeight(1)

            Column()
              .width(1).height(28)
              .backgroundColor(this.getCurrentThemePalette().borderColor)

            Column({ space: 2 }) {
              Text('5km')
                .fontSize(DesignConstants.FONT.SIZE.XL)
                .fontWeight(FontWeight.Bold)
                .fontColor(this.getCurrentThemePalette().accentBackground)
              Text('半径')
                .fontSize(DesignConstants.FONT.SIZE.XS)
                .fontColor(this.getCurrentThemePalette().textSecondary)
            }
            .alignItems(HorizontalAlign.Center)
            .layoutWeight(1)
          }
          .width('100%')
          .padding(DesignConstants.SPACING.MD)
          .borderRadius(DesignConstants.SHAPE.CARD)
          .backgroundColor(this.getCurrentThemePalette().surfaceBackground)
          .border({ width: 1, color: this.getCurrentThemePalette().borderColor })
          .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.SM })
        }

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
            icon: AppIcon.RECOMMEND,
            message: '附近暂无其他用户，可稍后再来或调整浏览位置'
          })
            .layoutWeight(1)
        } else {
          Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
            List() {
              LazyForEach(this.usersDataSource, (user: UserItem) => {
                ListItem() {
                  this.buildUserRow(user)
                }
              }, (user: UserItem) => user.user?.user_id ?? '')

              if (this.hasMore) {
                ListItem() {
                  LoadingMoreIndicator()
                }
              } else {
                ListItem() {
                  Text('没有更多了')
                    .fontSize(DesignConstants.FONT.SIZE.XS)
                    .fontColor(this.getCurrentThemePalette().textTertiary)
                    .width('100%')
                    .textAlign(TextAlign.Center)
                    .padding({ top: 16, bottom: 16 })
                }
              }
            }
            .width('100%')
            .layoutWeight(1)
            .scrollBar(BarState.Auto)
            .onReachEnd(() => { this.loadMoreUsers(); })
          }
          .layoutWeight(1)
          .onRefreshing(() => { this.loadUsers(true); })
        }
      }
      .width('100%')
      .height('100%')
      .backgroundColor(this.getCurrentThemePalette().pageBackground)
    }
    }
    .hideTitleBar(true)
  }
}

@Builder
export function NearbyUsersListPageBuilder() {
  NearbyUsersListPage()
}
