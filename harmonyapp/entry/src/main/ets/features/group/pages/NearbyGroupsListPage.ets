/**
 * 附近的群组 - 独立列表页
 * 对齐前端 /groups/nearby：群组列表（距离/成员数）+ 下拉刷新 + 上拉分页。
 * 数据来自 groups/search（reference_point + radius + sort_by=distance）。
 */

import { GlobalStateManager, KEY_BROWSE_ANCHOR, KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { ResponsiveContainer } from '../../../components/ResponsiveContainer';
import { LocationInfo, LocationUtil } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { AppRouter, GroupDetailRouteParam } from '../../../core/router/AppRouter';
import { PageNames, DefaultConfig } from '../../../constants/AppConstants';
import { GroupItem, GeoPoint, SearchGroupsRequest } from '../../../core/types/api';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { EmptyState, RefreshIndicator, LoadingMoreIndicator } from '../../../components/EmptyState';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { formatCount } from '../../../core/utils/FormatUtil';

@Preview
@Component
export struct NearbyGroupsListPage {
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';
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
  private groupsDataSource: BasicDataSource<GroupItem> = new BasicDataSource<GroupItem>([]);
  private isDestroyed: boolean = false;

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getHeaderTopPadding(): number {
    return SafeAreaUtils.resolveTopPadding(this.safeAreaTop);
  }

  private getReferencePoint(): GeoPoint | null {
    const ref = this.browseAnchor ?? this.currentLocation;
    if (ref) {
      return { latitude: ref.latitude, longitude: ref.longitude };
    }
    return null;
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
        this.groupsDataSource.setData(this.groups);
        this.hasMore = !!(response.data?.next_cursor);
        this.nextCursor = response.data?.next_cursor ?? '';
      } else {
        this.errorMessage = response.message || '加载失败';
        this.groups = [];
        this.groupsDataSource.clear();
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
        pagination: { limit: DefaultConfig.PAGE_SIZE_LARGE, cursor: this.nextCursor }
      };
      const response = await ApiClient.searchGroups(request);
      if (this.isDestroyed) return;
      if (response.code === 0) {
        const items = (response.data?.items ?? []).filter((g: GroupItem) => g.group?.group_id);
        this.groups = this.groups.concat(items);
        this.groupsDataSource.appendData(items);
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

  @Builder
  private buildGroupRow(group: GroupItem) {
    Row({ space: 12 }) {
      Row() {
        Text() {
          SymbolSpan($r(AppIcon.GROUP))
            .fontSize(DesignConstants.FONT.SIZE.XL)
            .fontColor([this.getCurrentThemePalette().accentBackground])
        }
      }
      .width(48)
      .height(48)
      .justifyContent(FlexAlign.Center)
      .borderRadius(DesignConstants.RADIUS.LG)
      .backgroundColor(this.getCurrentThemePalette().accentMuted)

      Column({ space: 4 }) {
        Text(group.group?.name ?? '')
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontWeight(FontWeight.Medium)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        Row({ space: 8 }) {
          Text(`${formatCount(group.group?.members_count)} 人`)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getCurrentThemePalette().textTertiary)
          if (group.location?.distance !== undefined && group.location?.distance !== null) {
            Text(LocationUtil.formatDistance(group.location.distance))
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor(this.getCurrentThemePalette().accentText)
          }
        }
        .alignItems(VerticalAlign.Center)

        if (group.description) {
          Text(group.description)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getCurrentThemePalette().textSecondary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
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
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: 12, bottom: 12 })
    .onClick(() => {
      const groupId = group.group?.group_id;
      if (groupId) {
        this.openGroupDetail(groupId);
      }
    })
  }

  build() {
    NavDestination() {
    ResponsiveContainer() {
      Column() {
        // 头部 —— 统一走 SystemPageHeader（返回 + 标题）
        SystemPageHeader({
          title: '附近的群组',
          showBackButton: true,
          onBack: () => { AppRouter.pop(); },
          titleWeight: FontWeight.Medium,
          backgroundVariant: 'transparent'
        })

        // 搜索栏（对齐前端 /groups/nearby）
        Row({ space: 10 }) {
          Text() {
            SymbolSpan($r(AppIcon.SEARCH))
              .fontSize(DesignConstants.FONT.SIZE.MD_PLUS)
              .fontColor([this.getCurrentThemePalette().textSecondary])
          }
          TextInput({ placeholder: '搜索群组', text: $$this.keyword })
            .fontSize(DesignConstants.FONT.SIZE.SM_PLUS)
            .fontColor(this.getCurrentThemePalette().textPrimary)
            .placeholderColor(this.getCurrentThemePalette().textSecondary)
            .layoutWeight(1)
            .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
            .onSubmit(() => { this.loadGroups(true); })
        }
        .height(DesignConstants.COMPONENT.INPUT_HEIGHT)
        .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
        .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.SM })
        .backgroundColor(this.getCurrentThemePalette().inputBackgroundColor)
        .borderRadius(DesignConstants.SHAPE.INPUT)
        .border({ width: 1, color: this.getCurrentThemePalette().borderColor })

        // 分类 chips（对齐前端 filter categories）
        Row({ space: DesignConstants.SPACING.SM }) {
          ForEach(this.TAGS, (tag: string) => {
            Text(tag)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor(tag === this.activeTag ? this.getCurrentThemePalette().accentText : this.getCurrentThemePalette().textSecondary)
              .padding({ left: 12, right: 12, top: 6, bottom: 6 })
              .borderRadius(DesignConstants.SHAPE.CHIP)
              .backgroundColor(tag === this.activeTag ? this.getCurrentThemePalette().accentBackground : this.getCurrentThemePalette().surfaceSubtle)
              .onClick(() => {
                if (tag !== this.activeTag) {
                  this.activeTag = tag;
                  this.loadGroups(true);
                }
              })
          }, (tag: string) => tag)
        }
        .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.SM })

        // Hero 统计卡（对齐前端「附近概览」）
        if (this.hasAnyData()) {
          Row({ space: DesignConstants.SPACING.MD }) {
            Column({ space: 2 }) {
              Text(`${this.groups.length}`)
                .fontSize(DesignConstants.FONT.SIZE.XL)
                .fontWeight(FontWeight.Bold)
                .fontColor(this.getCurrentThemePalette().accentBackground)
              Text('群组')
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
          SystemLoadingView({ message: '正在加载附近的群组...' })
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
            List() {
              LazyForEach(this.groupsDataSource, (group: GroupItem) => {
                ListItem() {
                  this.buildGroupRow(group)
                }
              }, (group: GroupItem) => group.group?.group_id ?? '')

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
            .onReachEnd(() => { this.loadMoreGroups(); })
          }
          .layoutWeight(1)
          .onRefreshing(() => { this.loadGroups(true); })
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
export function NearbyGroupsListPageBuilder() {
  NearbyGroupsListPage()
}
