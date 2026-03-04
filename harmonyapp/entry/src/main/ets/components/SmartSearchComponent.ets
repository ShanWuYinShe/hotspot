/**
 * 智能搜索组件
 * 支持用户、群组、瞬间的统一搜索
 */

import { ApiClient } from '../core/api/ApiClient';
import { LoadingUtil } from '../core/utils/LoadingUtil';
import { 
  UserItem, 
  GroupItem, 
  MomentItem
} from '../core/types/api';
import { KEY_CURRENT_LOCATION, KEY_BROWSE_ANCHOR } from '../core/state/GlobalStateManager';
import { GlobalStateManager } from '../core/state/GlobalStateManager';
import { LocationInfo } from '../core/location/LocationUtil';
import { SystemAvatar } from './SystemAvatar';
import { DesignConstants } from '../constants/DesignConstants';
import { SystemToast } from './SystemToast';
import { PageNames } from '../constants/AppConstants';
import { AppRouter, GroupDetailRouteParam } from '../core/router/AppRouter';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF13;
const TAG: string = 'SmartSearchComponent';

interface SearchResult {
  id: string;
  type: 'user' | 'group' | 'moment';
  title: string;
  subtitle: string;
  data: UserItem | GroupItem | MomentItem;
}

@Component
export struct SmartSearchComponent {
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  
  @State searchText: string = '';
  @State searchResults: SearchResult[] = [];
  @State isSearching: boolean = false;
  @State showResults: boolean = false;
  @State searchHistory: string[] = [];
  @State currentFilter: 'all' | 'users' | 'groups' | 'moments' = 'all';
  
  private searchTimer: number = -1;

  aboutToAppear() {
    this.loadSearchHistory();
  }

  aboutToDisappear() {
    if (this.searchTimer !== -1) {
      clearTimeout(this.searchTimer);
    }
  }

  private loadSearchHistory() {
    this.searchHistory = ['附近咖啡厅', '技术交流群', '摄影爱好者'];
  }

  private async performSearch(query: string) {
    if (!query.trim()) {
      this.searchResults = [];
      this.showResults = false;
      return;
    }

    this.isSearching = true;
    this.showResults = true;

    try {
      await LoadingUtil.withMinimumTime(async () => {
        hilog.info(DOMAIN_NUMBER, TAG, `执行搜索: ${query}, 筛选: ${this.currentFilter}`);

        const searchPromises: Promise<void>[] = [];

        if (this.currentFilter === 'all' || this.currentFilter === 'users') {
          searchPromises.push(this.searchUsers(query));
        }

        if (this.currentFilter === 'all' || this.currentFilter === 'groups') {
          searchPromises.push(this.searchGroups(query));
        }

        if (this.currentFilter === 'all' || this.currentFilter === 'moments') {
          searchPromises.push(this.searchMoments(query));
        }

        await Promise.all(searchPromises);
        this.saveSearchHistory(query);
      }, LoadingUtil.DEFAULT_ACTION_MS);

    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `搜索失败: ${error}`);
      SystemToast.showError('搜索失败，请重试', this.getUIContext());
    } finally {
      this.isSearching = false;
    }
  }

  private async searchUsers(query: string) {
    const refLoc: LocationInfo | null = this.browseAnchor ?? this.currentLocation;
    if (!refLoc) {
      return;
    }

    try {
      const me = GlobalStateManager.getInstance().getCurrentUser();
      const myId = me?.user_id;
      const response = await ApiClient.searchUsers({
        keyword: query,
        reference_point: {
          latitude: refLoc.latitude,
          longitude: refLoc.longitude
        },
        pagination: { limit: 10 }
      });

      if (response.code === 0) {
        const items = response.data?.items ?? [];
        const validUsers = items.filter((u: UserItem) => u.user?.user_id);
        const users = typeof myId === 'string'
          ? validUsers.filter((u: UserItem) => u.user.user_id !== myId)
          : validUsers;

        const userResults: SearchResult[] = users.map((user: UserItem): SearchResult => ({
          id: `user_${user.user.user_id}`,
          type: 'user',
          title: user.user.nickname ?? '',
          subtitle: user.bio || '用户',
          data: user
        }));

        this.searchResults = [...this.searchResults, ...userResults];
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `搜索用户失败: ${error}`);
    }
  }

  private async searchGroups(query: string) {
    const refLoc: LocationInfo | null = this.browseAnchor ?? this.currentLocation;
    if (!refLoc) {
      return;
    }

    try {
      const response = await ApiClient.searchGroups({
        keyword: query,
        reference_point: {
          latitude: refLoc.latitude,
          longitude: refLoc.longitude
        },
        pagination: { limit: 10 }
      });

      if (response.code === 0) {
        const items = response.data?.items ?? [];
        const validGroups = items.filter((g: GroupItem) => g.group?.group_id);
        const groupResults: SearchResult[] = validGroups.map((group: GroupItem): SearchResult => ({
          id: `group_${group.group.group_id}`,
          type: 'group',
          title: group.group.name ?? '',
          subtitle: `${group.group.members_count ?? 0} 成员`,
          data: group
        }));

        this.searchResults = [...this.searchResults, ...groupResults];
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `搜索群组失败: ${error}`);
    }
  }

  private async searchMoments(query: string) {
    const refLoc: LocationInfo | null = this.browseAnchor ?? this.currentLocation;
    if (!refLoc) {
      return;
    }

    try {
      const response = await ApiClient.searchMoments({
        keyword: query,
        reference_point: {
          latitude: refLoc.latitude,
          longitude: refLoc.longitude
        },
        pagination: { limit: 10 }
      });

      if (response.code === 0) {
        const items = response.data?.items ?? [];
        const validMoments = items.filter((m: MomentItem) => m.moment_id && m.user);
        const momentResults: SearchResult[] = validMoments.map((moment: MomentItem): SearchResult => ({
          id: `moment_${moment.moment_id}`,
          type: 'moment',
          title: moment.user?.nickname ?? '',
          subtitle: (moment.description?.length ?? 0) > 30 ?
            moment.description.substring(0, 30) + '...' : (moment.description ?? ''),
          data: moment
        }));

        this.searchResults = [...this.searchResults, ...momentResults];
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `搜索瞬间失败: ${error}`);
    }
  }

  private saveSearchHistory(query: string) {
    if (!this.searchHistory.includes(query)) {
      this.searchHistory.unshift(query);
      if (this.searchHistory.length > 10) {
        this.searchHistory = this.searchHistory.slice(0, 10);
      }
    }
  }

  private onSearchTextChange(value: string) {
    this.searchText = value;

    if (this.searchTimer !== -1) {
      clearTimeout(this.searchTimer);
    }

    this.searchTimer = setTimeout(() => {
      this.searchResults = [];
      this.performSearch(value);
    }, 300);
  }

  private handleResultClick(result: SearchResult) {
    this.showResults = false;

    switch (result.type) {
      case 'user':
        SystemToast.show(`查看用户: ${result.title}`, this.getUIContext());
        break;
      case 'group':
        const group = result.data as GroupItem;
        if (!group.group?.group_id) break;
        const p = new GroupDetailRouteParam();
        p.groupId = group.group.group_id;
        AppRouter.push(PageNames.GROUP_DETAIL, p);
        break;
      case 'moment':
        SystemToast.show(`查看瞬间: ${result.title}`, this.getUIContext());
        break;
    }
  }

  @Builder
  private FilterChip(label: string, filter: string) {
    Text(label)
      .fontSize(DesignConstants.FONT.SIZE.SM)
      .fontWeight(this.currentFilter === filter ? FontWeight.Medium : FontWeight.Normal)
      .fontColor(this.currentFilter === filter
        ? $r('sys.color.ohos_id_color_emphasize')
        : $r('sys.color.ohos_id_color_text_secondary'))
      .height(30)
      .layoutWeight(1)
      .textAlign(TextAlign.Center)
      .backgroundColor(this.currentFilter === filter
        ? $r('app.color.chip_selected_bg')
        : Color.Transparent)
      .borderRadius(DesignConstants.RADIUS.XL)
      .padding({ left: 2, right: 2 })
      .onClick(() => {
        this.switchFilter(filter as 'all' | 'users' | 'groups' | 'moments');
      })
  }

  private switchFilter(filter: 'all' | 'users' | 'groups' | 'moments') {
    if (this.currentFilter === filter) {
      return;
    }

    this.currentFilter = filter;
    if (this.searchText.trim()) {
      this.searchResults = [];
      this.performSearch(this.searchText);
    }
  }

  build() {
    Column() {
      Search({ value: this.searchText, placeholder: '搜索用户、群组、瞬间...' })
        .searchButton('搜索')
        .onChange((value: string) => this.onSearchTextChange(value))
        .onSubmit((value: string) => this.performSearch(value))
        .onFocus(() => {
          if (!this.searchText.trim() && this.searchHistory.length > 0) {
            this.showResults = true;
          }
        })
        .width('100%')
        .padding({ left: 12, right: 12, top: 8, bottom: 8 })

      if (this.showResults) {
        Row({ space: 0 }) {
          this.FilterChip('全部', 'all')
          this.FilterChip('用户', 'users')
          this.FilterChip('群组', 'groups')
          this.FilterChip('瞬间', 'moments')
        }
        .padding(3)
        .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
          bottom: DesignConstants.SPACING.SM })
        .backgroundColor($r('app.color.bg_tertiary'))
        .borderRadius(DesignConstants.RADIUS.XL)
        .transition(TransitionEffect.OPACITY.animation({ duration: 250, curve: Curve.FastOutSlowIn }))
      }

      if (this.showResults) {
        List() {
          if (this.searchText.trim() === '' && this.searchHistory.length > 0) {
            ListItem() {
              Text('搜索历史')
                .fontSize(14)
                .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                .padding({
                  left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
                  top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.SM
                })
            }

            ForEach(this.searchHistory, (historyItem: string) => {
              ListItem() {
                Row() {
                  Text('🕐')
                    .fontSize(16)
                    .margin({ right: 12 })

                  Text(historyItem)
                    .fontSize(14)
                    .layoutWeight(1)

                  Button('✕')
                    .type(ButtonType.Capsule)
                    .width(24)
                    .height(24)
                    .fontSize(12)
                    .backgroundColor(Color.Transparent)
                    .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                    .onClick(() => {
                      const index = this.searchHistory.indexOf(historyItem);
                      if (index > -1) {
                        this.searchHistory.splice(index, 1);
                      }
                    })
                }
                .width('100%')
                .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
                .onClick(() => {
                  this.searchText = historyItem;
                  this.performSearch(historyItem);
                })
              }
            }, (item: string) => item)
          } else if (this.searchResults.length > 0) {
            ForEach(this.searchResults, (result: SearchResult) => {
              ListItem() {
                Row() {
                  Text(this.getResultIcon(result.type))
                    .fontSize(20)
                    .margin({ right: 12 })

                  Column() {
                    Text(result.title)
                      .fontSize(14)
                      .fontWeight(FontWeight.Medium)
                      .maxLines(1)
                      .textOverflow({ overflow: TextOverflow.Ellipsis })
                      .alignSelf(ItemAlign.Start)

                    Text(result.subtitle)
                      .fontSize(12)
                      .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                      .maxLines(1)
                      .textOverflow({ overflow: TextOverflow.Ellipsis })
                      .alignSelf(ItemAlign.Start)
                  }
                  .alignItems(HorizontalAlign.Start)
                  .layoutWeight(1)
                }
                .width('100%')
                .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
                .onClick(() => this.handleResultClick(result))
              }
            }, (result: SearchResult) => result.id)
          } else if (this.searchText.trim() !== '' && !this.isSearching) {
            ListItem() {
              Column() {
                Text('😕')
                  .fontSize(32)
                  .margin({ bottom: 8 })

                Text('未找到相关内容')
                  .fontSize(14)
                  .fontColor($r('sys.color.ohos_id_color_text_secondary'))
              }
              .width('100%')
              .padding(40)
              .justifyContent(FlexAlign.Center)
              .alignItems(HorizontalAlign.Center)
            }
          }
        }
        .width('100%')
        .layoutWeight(1)
        .backgroundColor($r('app.color.bg_secondary'))
        .borderRadius({ topLeft: DesignConstants.RADIUS.XXL, topRight: DesignConstants.RADIUS.XXL })
        .border({ width: 1, color: $r('sys.color.ohos_id_color_list_separator') })
        .shadow({
          radius: 8,
          color: $r('app.color.color_shadow'),
          offsetX: 0,
          offsetY: -2
        })
        .transition(TransitionEffect.OPACITY.animation({ duration: 250, curve: Curve.FastOutSlowIn }))
      }
    }
    .width('100%')
    .height('100%')
    .onClick(() => {
      if (this.showResults) {
        this.showResults = false;
      }
    })
  }

  private getResultIcon(type: 'user' | 'group' | 'moment'): string {
    switch (type) {
      case 'user':
        return '👤';
      case 'group':
        return '👥';
      case 'moment':
        return '📸';
      default:
        return '🔍';
    }
  }
}
