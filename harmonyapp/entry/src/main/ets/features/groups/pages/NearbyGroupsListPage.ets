/**
 * 附近的群组 - 独立列表页
 * 对齐前端 /nearby/groups：2 列 3:4 群组图片卡墙（桌面 3 列）+ 移动端首卡 featured
 * 全宽精选大卡；搜索/分类/统计行保留；下拉刷新 + 上拉分页。
 * 数据来自 groups/search（reference_point + radius + sort_by=distance）。
 */

import { GlobalStateManager, KEY_BROWSE_ANCHOR, KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { PageShellContainer } from '../../../components/PageShellContainer';
import { isDesktopForm, BreakpointType } from '../../../core/responsive/Breakpoint';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { AppRouter, GroupDetailRouteParam } from '../../../core/router/AppRouter';
import { PageNames, DefaultConfig } from '../../../constants/AppConstants';
import { GroupItem, GeoPoint, SearchGroupsRequest } from '../../../core/types/api';
import { GroupCard } from '../components/GroupCard';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemInput } from '../../../components/SystemInput';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { EmptyState, RefreshIndicator, LoadingMoreIndicator } from '../../../components/EmptyState';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { TabsPillGroup, TabItem } from '../../../components/TabsPillGroup';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';

@Preview
@Component
export struct NearbyGroupsListPage {
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;

  @State groups: GroupItem[] = [];
  @State isLoading: boolean = true;
  @State isRefreshing: boolean = false;
  @State isLoadingMore: boolean = false;
  @State hasMore: boolean = false;
  @State nextCursor: string = '';
  @State errorMessage: string = '';
  @State keyword: string = '';
  @State activeTag: string = '全部';
  private readonly TAGS: string[] = ['全部', '摄影', '音乐', '户外', '阅读', '美食', '健身', '游戏'];
  private isDestroyed: boolean = false;

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getReferencePoint(): GeoPoint | null {
    const ref = this.browseAnchor ?? this.currentLocation;
    if (ref) {
      return { latitude: ref.latitude, longitude: ref.longitude };
    }
    return null;
  }

  /** 当前列表群组在线成员总数（对齐前端 group-data.ts totalOnline = Σ online_members_count） */
  private getOnlineTotalCount(): number {
    let total = 0;
    for (let index = 0; index < this.groups.length; index++) {
      const count = this.groups[index].online_members_count;
      if (count !== null && count !== undefined) {
        total += count;
      }
    }
    return total;
  }

  /** 位置行副标题（对齐前端 nearby 页 subtitle：{locationName} · 5km 内 / 5km 内） */
  private getLocationSubtitle(): string {
    const name = (this.browseAnchor ?? this.currentLocation)?.locationName ?? '';
    return name.length > 0 ? `${name} · 5km 内` : '5km 内';
  }

  aboutToAppear() {
    this.loadGroups();
  }

  aboutToDisappear() {
    this.isDestroyed = true;
  }

  private hasAnyData(): boolean {
    return this.groups.length > 0;
  }

  private async loadGroups(refresh: boolean = false): Promise<void> {
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
      const request: SearchGroupsRequest = {
        reference_point: ref,
        radius: DefaultConfig.NEARBY_RADIUS,
        sort_by: 'distance',
        keyword: this.keyword || undefined,
        tag: (this.activeTag !== '全部' ? this.activeTag : undefined) as string | undefined,
        pagination: { limit: DefaultConfig.PAGE_SIZE_LARGE }
      };
      const response = await LoadingUtil.withMinimumTime(() => ApiClient.searchGroups(request));
      if (this.isDestroyed) return;
      if (response.code === 0) {
        const items = (response.data?.items ?? []).filter((g: GroupItem) => g.group?.group_id);
        this.groups = items;
        this.hasMore = !!(response.data?.next_cursor);
        this.nextCursor = response.data?.next_cursor ?? '';
      } else {
        this.errorMessage = getErrorMessage(response, '加载失败');
        this.groups = [];
      }
    } catch (error) {
      this.errorMessage = '加载附近群组失败';
    } finally {
      this.isLoading = false;
      this.isRefreshing = false;
    }
  }

  private async loadMoreGroups(): Promise<void> {
    if (this.isLoadingMore || !this.hasMore) return;
    const ref = this.getReferencePoint();
    if (!ref) return;
    this.isLoadingMore = true;
    try {
      const request: SearchGroupsRequest = {
        reference_point: ref,
        radius: DefaultConfig.NEARBY_RADIUS,
        sort_by: 'distance',
        // 翻页必须透传当前生效的筛选条件，否则翻页后 keyword/tag 筛选失效
        keyword: this.keyword || undefined,
        tag: (this.activeTag !== '全部' ? this.activeTag : undefined) as string | undefined,
        pagination: { limit: DefaultConfig.PAGE_SIZE_LARGE, cursor: this.nextCursor }
      };
      const response = await ApiClient.searchGroups(request);
      if (this.isDestroyed) return;
      if (response.code === 0) {
        const items = (response.data?.items ?? []).filter((g: GroupItem) => g.group?.group_id);
        this.groups = this.groups.concat(items);
        this.hasMore = !!(response.data?.next_cursor);
        this.nextCursor = response.data?.next_cursor ?? '';
      }
    } catch (error) {
      // 静默失败，用户可再次触底重试
    } finally {
      this.isLoadingMore = false;
    }
  }

  private openGroupDetail(groupId: string): void {
    const params = new GroupDetailRouteParam();
    params.groupId = groupId;
    AppRouter.push(PageNames.GROUP_DETAIL, params);
  }

  private isDesktop(): boolean {
    return isDesktopForm(this.currentBreakpoint);
  }

  /** 网格卡分列数：mobile/tablet 2 列，desktop(≥1280) 3 列（对齐前端 /nearby/groups） */
  private getColumnCount(): number {
    return this.isDesktop() ? 3 : 2;
  }

  /** featured 首卡仅移动/平板端（对齐前端移动端首卡精选，桌面 BSP 无 featured） */
  private getFeaturedGroup(): GroupItem | null {
    if (this.isDesktop() || this.groups.length === 0) {
      return null;
    }
    return this.groups[0];
  }

  /** featured 之后的卡按列数分块（Scroll + ForEach chunk，分页规模可控） */
  private getWallGroups(): GroupItem[] {
    const featured = this.getFeaturedGroup();
    return featured ? this.groups.slice(1) : this.groups;
  }

  private getWallRows(): GroupItem[][] {
    const columns = this.getColumnCount();
    const items = this.getWallGroups();
    const rows: GroupItem[][] = [];
    for (let index = 0; index < items.length; index += columns) {
      rows.push(items.slice(index, index + columns));
    }
    return rows;
  }

  build() {
    NavDestination() {
    PageShellContainer({ variant: 'default', currentBreakpoint: this.currentBreakpoint}) {
      Column() {
        // 头部 —— 统一走 SystemPageHeader（返回 + 标题 24pt + 位置行副标题 + 计数 trailing，
        // 对齐前端 /nearby/groups PageHeader：titleSize 2xl + MapPin 图标 + {位置} · 5km 内 + trailing N 个活跃群组）
        SystemPageHeader({
          title: '附近群组',
          subtitle: this.getLocationSubtitle(),
          subtitleIcon: AppIcon.LOCATION,
          titleSize: DesignConstants.FONT.SIZE.XXL,
          titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM,
          showBackButton: true,
          onBack: () => { AppRouter.pop(); },
          backgroundVariant: 'transparent'
        }) {
          if (this.hasAnyData()) {
            Text(`${this.groups.length} 个活跃群组`)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor(this.getCurrentThemePalette().textSecondary)
          }
        }

        // 搜索栏（对齐前端 /groups/nearby；SystemInput 自带 16 圆角 + hairline
        // 边框，外层不得再叠 border——否则圆角输入框外套直角框形成双重外框）
        SystemInput({
          value: this.keyword,
          placeholder: '搜索群组',
          prefixIcon: AppIcon.SEARCH,
          showPrefixIcon: true,
          inputHeight: DesignConstants.COMPONENT.INPUT_HEIGHT,
          onChange: (value: string) => { this.keyword = value; },
          onSubmit: () => { this.loadGroups(true); }
        })
          .margin({ bottom: DesignConstants.SPACING.SM })

        // 分类 TabsPillGroup（对齐前端 TabBar）
        TabsPillGroup({
          tabs: this.TAGS.map((tag: string) => ({ label: tag } as TabItem)),
          activeIndex: Math.max(0, this.TAGS.indexOf(this.activeTag)),
          actions: {
            onChange: (index: number) => {
              const tag = this.TAGS[index];
              if (tag && tag !== this.activeTag) {
                this.activeTag = tag;
                this.loadGroups(true);
              }
            }
          }
        })
          .padding({ bottom: DesignConstants.SPACING.SM })

        // 内容区
        if (this.isLoading) {
          SystemLoadingView({ message: '正在加载附近的群组…' })
            .layoutWeight(1)
        } else if (!this.hasAnyData() && this.errorMessage) {
          EmptyState({
            icon: AppIcon.WARNING,
            message: this.errorMessage,
            actionLabel: '重试',
            onAction: () => { this.loadGroups(); }
          })
            .layoutWeight(1)
        } else if (!this.hasAnyData()) {
          EmptyState({
            icon: AppIcon.GROUP,
            message: '附近暂无群组，可稍后再来或调整浏览位置'
          })
            .layoutWeight(1)
        } else {
          Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
            Scroll() {
              Column() {
                // section header（对齐前端：eyebrow「此刻附近群组」+ status-online trailing）
                Row() {
                  Text('此刻附近群组')
                    .fontSize(DesignConstants.FONT.SIZE.XS2)
                    .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                    .fontColor(this.getCurrentThemePalette().textSecondary)
                    .letterSpacing(DesignConstants.FONT.LETTER_SPACING.EYEBROW)
                    .textCase(TextCase.UpperCase)
                  Blank()
                  Text(`${this.getOnlineTotalCount()} 人正在群聊`)
                    .fontSize(DesignConstants.FONT.SIZE.XS)
                    .fontColor(this.getCurrentThemePalette().online)
                }
                .width('100%')
                .padding({ bottom: DesignConstants.SPACING.SM })
                .alignItems(VerticalAlign.Center)

                // 移动/平板端首卡 featured 全宽精选大卡
                if (this.getFeaturedGroup()) {
                  GroupCard({ group: this.getFeaturedGroup() as GroupItem, featured: true })
                    .margin({ bottom: DesignConstants.SPACING.LG })
                }

                // 2/3 列图片卡墙
                ForEach(this.getWallRows(), (rowGroups: GroupItem[]) => {
                  Row({ space: DesignConstants.SPACING.LG }) {
                    ForEach(rowGroups, (group: GroupItem) => {
                      GroupCard({ group: group })
                        .layoutWeight(1)
                    }, (group: GroupItem) => group.group?.group_id ?? '')
                  }
                  .width('100%')
                  .alignItems(VerticalAlign.Top)
                  .margin({ bottom: DesignConstants.SPACING.LG })
                }, (rowGroups: GroupItem[], index: number) => `row-${index}`)

                // 分页状态条
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
              .constraintSize({ minHeight: '100%' })
            }
            .width('100%')
            .layoutWeight(1)
            .scrollBar(BarState.Off)
            .onReachEnd(() => { this.loadMoreGroups(); })
            .edgeEffect(EdgeEffect.None)
          }
          .layoutWeight(1)
          .onRefreshing(() => { this.loadGroups(true); })
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
export function NearbyGroupsListPageBuilder() {
  NearbyGroupsListPage()
}