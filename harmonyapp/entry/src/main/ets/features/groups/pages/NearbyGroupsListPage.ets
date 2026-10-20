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
import { DesignConstants, textLineHeight } from '../../../constants/DesignConstants';
import { ListEnd } from '../../../components/ListEnd';
import { AppIcon } from '../../../components/SystemIcon';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { EmptyState, RefreshIndicator, LoadingMoreIndicator } from '../../../components/EmptyState';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SurfaceIconButton } from '../../../components/SurfaceIconButton';
import { TabsPillGroup, TabItem } from '../../../components/TabsPillGroup';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';

@Preview
@Component
export struct NearbyGroupsListPage {
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  @StorageLink(KEY_BROWSE_ANCHOR) @Watch('onBrowseAnchorChanged') browseAnchor: LocationInfo | null = null;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;

  @State groups: GroupItem[] = [];
  @State isLoading: boolean = true;
  @State isRefreshing: boolean = false;
  @State isLoadingMore: boolean = false;
  @State hasMore: boolean = false;
  @State nextCursor: string = '';
  @State errorMessage: string = '';
  @State activeTag: string = '全部';
  // 附近域统一分类词库（对齐前端 nearby/shared/categories.ts NEARBY_CATEGORIES：
  // 全部/摄影/户外/旅行/音乐/美食/阅读/健身 —— 附近的人与附近群组共用同一套，
  // 此前两端词库割裂：本端群组用「游戏」、人用「运动/艺术」，前端均无对应项，选中必空结果）
  private readonly TAGS: string[] = ['全部', '摄影', '户外', '旅行', '音乐', '美食', '阅读', '健身'];
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

  /**
   * 浏览锚点变化 → 若当前处于「未选位置」错误态则自动重载（与 NearbyUsersListPage 同因同治）。
   * 用 @Watch 而非 onPageShow：本项目导航走 AppRouter，页面生命周期回调不触发（实测）。
   */
  onBrowseAnchorChanged(): void {
    if (this.errorMessage.includes('位置信息不可用') && this.getReferencePoint()) {
      void this.loadGroups();
    }
  }

  aboutToAppear() {
    // 同 NearbyUsersListPage：@StorageLink(browseAnchor/currentLocation) 绑定晚于
    // aboutToAppear，同步发请求会读到 null 锚点而落到「位置信息不可用」错误态。
    setTimeout(() => {
      if (!this.isDestroyed) {
        this.loadGroups();
      }
    }, 0);
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
        // 翻页必须透传当前生效的筛选条件，否则翻页后 tag 筛选失效
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
        // ── 页头区块（对齐前端 PageHeader：标题行 + `below` 分类胶囊行，分隔线画在 below 之下）──
        Column() {
        SystemPageHeader({
          title: '附近群组',
          subtitle: this.getLocationSubtitle(),
          subtitleIcon: AppIcon.LOCATION,
          titleSize: DesignConstants.FONT.SIZE.XXL,
          titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM,
          showBackButton: true,
          onBack: () => { AppRouter.pop(); },
          backgroundVariant: 'transparent',
          hasBelowContent: true,
          showDivider: false
        }) {
          // trailing：搜索/地图两个页面级工具入口（对齐前端 /nearby/groups PageHeader
          // trailing，搜索在移动端显示、桌面隐藏；本端统一显示）
          Row({ space: DesignConstants.SPACING.SM }) {
            SurfaceIconButton({
              icon: AppIcon.SEARCH,
              accessibilityLabel: '搜索',
              kind: 'neutral',
              buttonSize: DesignConstants.LAYOUT.ICON_BUTTON_SIZE,
              onPress: () => AppRouter.push(PageNames.SEARCH)
            })
            SurfaceIconButton({
              icon: AppIcon.MAP,
              accessibilityLabel: '地图浏览',
              kind: 'neutral',
              buttonSize: DesignConstants.LAYOUT.ICON_BUTTON_SIZE,
              onPress: () => AppRouter.push(PageNames.LOCATION_PICKER)
            })
          }
        }

        // 分类胶囊行 = 页头 below（前端 header 内 mt-3 pb-2 的 TabBar；紧凑横向滚动胶囊）
        Scroll() {
        TabsPillGroup({
          tabs: this.TAGS.map((tag: string) => ({ label: tag } as TabItem)),
          freeWidth: true,
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
        }
        .scrollable(ScrollDirection.Horizontal)
        .scrollBar(BarState.Off)
        .align(Alignment.Start)
        .width('100%')
        .padding({ top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.SM })
        }
        .width('100%')
        .padding({ bottom: DesignConstants.SPACING.MD3 })
        .border({
          width: { bottom: DesignConstants.BORDER_WIDTH.HAIRLINE },
          color: this.getCurrentThemePalette().borderColor
        })

        // ── 正文（对齐前端内容列 pt-4 pb-6）──
        // 内容区
        if (this.isLoading) {
          SystemLoadingView({ message: '正在加载附近的群组…' })
            .layoutWeight(1)
        } else if (!this.hasAnyData() && this.errorMessage) {
          // 错误分流动作：位置不可用属「重试不可解」死路（同 r70/r15 模式），
          // 引导直达位置设置页；其余错误维持重试
          EmptyState({
            icon: AppIcon.WARNING,
            message: this.errorMessage,
            actionLabel: this.errorMessage.includes('位置信息不可用') ? '去设置位置' : '重试',
            onAction: (): void => {
              if (this.errorMessage.includes('位置信息不可用')) {
                AppRouter.push(PageNames.LOCATION_PICKER);
              } else {
                this.loadGroups();
              }
            }
          })
            .layoutWeight(1)
        } else if (!this.hasAnyData()) {
          // 空态引导（r41/r42）：仅静态文本无下一步路径，补「创建群组」直达
          EmptyState({
            icon: AppIcon.GROUP,
            message: '附近暂无群组，可稍后再来或调整浏览位置',
            actionLabel: '创建群组',
            onAction: () => { AppRouter.push(PageNames.CREATE_GROUP); }
          })
            .layoutWeight(1)
        } else {
          Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
            Scroll() {
              Column() {
                // section header（对齐前端：eyebrow「此刻附近群组」+ status-online trailing，
                // 前端内容列 pt-4(16)、眉标行 mb-3(12)）
                Row() {
                  Text('此刻附近群组')
                    .fontSize(DesignConstants.FONT.SIZE.XS2)
                    .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS2))
                    .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                    .fontColor(this.getCurrentThemePalette().textSecondary)
                    .letterSpacing(DesignConstants.FONT.LETTER_SPACING.EYEBROW)
                    .textCase(TextCase.UpperCase)
                  Blank()
                  Text(`${this.getOnlineTotalCount()} 人正在群聊`)
                    .fontSize(DesignConstants.FONT.SIZE.XS)
                    .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
                    .fontColor(this.getCurrentThemePalette().online)
                }
                .width('100%')
                .alignItems(VerticalAlign.Center)
                // 眉标行 → 精选卡 12（前端 mb-3）
                .margin({ bottom: DesignConstants.SPACING.MD })

                // 移动/平板端首卡 featured 全宽精选大卡（前端 mb-5=20）
                if (this.getFeaturedGroup()) {
                  GroupCard({ group: this.getFeaturedGroup() as GroupItem, featured: true })
                    .margin({ bottom: DesignConstants.SPACING.LG2 })
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
                                    ListEnd()
                }
              }
              .width('100%')
              // 内容列顶距 16（对齐前端 StandardPageLayout pt-4）
              .padding({ top: DesignConstants.SPACING.LG })
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