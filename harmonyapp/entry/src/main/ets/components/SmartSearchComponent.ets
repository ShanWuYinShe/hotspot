/**
 * 智能搜索组件
 * 支持用户、群组、瞬间的统一搜索
 */

import { ApiClient } from '../core/api/ApiClient';
import { 
  UserSearchItem, 
  GroupSearchItem, 
  MomentSearchItem
} from '../core/types/api';
import { KEY_CURRENT_LOCATION } from '../core/state/GlobalStateManager';
import { GlobalStateManager } from '../core/state/GlobalStateManager';
import { LocationInfo } from '../core/location/LocationUtil';
import { SystemAvatar } from './SystemAvatar';
import { SystemToast } from './SystemToast';
import { PageNames } from '../constants/AppConstants';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF13;
const TAG: string = 'SmartSearchComponent';

interface SearchResult {
  id: string;
  type: 'user' | 'group' | 'moment';
  title: string;
  subtitle: string;
  data: UserSearchItem | GroupSearchItem | MomentSearchItem;
}

@Component
export struct SmartSearchComponent {
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  
  @State searchText: string = '';
  @State searchResults: SearchResult[] = [];
  @State isSearching: boolean = false;
  @State showResults: boolean = false;
  @State searchHistory: string[] = [];
  @State currentFilter: 'all' | 'users' | 'groups' | 'moments' = 'all';
  
  private searchTimer: number = -1;

  aboutToAppear() {
    // 加载搜索历史
    this.loadSearchHistory();
  }

  aboutToDisappear() {
    // 清理定时器
    if (this.searchTimer !== -1) {
      clearTimeout(this.searchTimer);
    }
  }

  /**
   * 加载搜索历史
   */
  private loadSearchHistory() {
    // 从本地存储加载搜索历史
    // 这里可以集成 PreferencesUtil
    this.searchHistory = ['附近咖啡厅', '技术交流群', '摄影爱好者'];
  }

  /**
   * 执行搜索
   */
  private async performSearch(query: string) {
    if (!query.trim()) {
      this.searchResults = [];
      this.showResults = false;
      return;
    }

    this.isSearching = true;
    this.showResults = true;

    try {
      hilog.info(DOMAIN_NUMBER, TAG, `执行搜索: ${query}, 筛选: ${this.currentFilter}`);

      // 并行搜索不同类型的内容
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

      // 保存搜索历史
      this.saveSearchHistory(query);

    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `搜索失败: ${error}`);
      SystemToast.showError('搜索失败，请重试', this.getUIContext());
    } finally {
      this.isSearching = false;
    }
  }

  /**
   * 搜索用户
   * 使用 searchUsersByKeyword API
   */
  private async searchUsers(query: string) {
    if (!this.currentLocation) {
      return;
    }

    try {
      const me = GlobalStateManager.getInstance().getCurrentUser();
      const myId = me?.user_id;
      const response = await ApiClient.searchUsersByKeyword({
        kw: query,
        latitude: this.currentLocation.latitude,
        longitude: this.currentLocation.longitude,
        limit: 10
      });

      if (response.code === 0 && response.data) {
        const users = typeof myId === 'number'
          ? response.data.items.filter((u: UserSearchItem) => u.user_id !== myId)
          : response.data.items;

        const userResults: SearchResult[] = users.map((user: UserSearchItem): SearchResult => ({
          id: `user_${user.user_id}`,
          type: 'user',
          title: user.nickname,
          subtitle: user.bio || '用户',
          data: user
        }));

        this.searchResults = [...this.searchResults, ...userResults];
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `搜索用户失败: ${error}`);
    }
  }

  /**
   * 搜索群组
   * 使用 searchGroupsByKeyword API
   */
  private async searchGroups(query: string) {
    if (!this.currentLocation) {
      return;
    }

    try {
      const response = await ApiClient.searchGroupsByKeyword({
        kw: query,
        latitude: this.currentLocation.latitude,
        longitude: this.currentLocation.longitude,
        limit: 10
      });

      if (response.code === 0 && response.data) {
        const groupResults: SearchResult[] = response.data.items.map((group: GroupSearchItem): SearchResult => ({
          id: `group_${group.group_chat_id}`,
          type: 'group',
          title: group.name,
          subtitle: `${group.members_count} 成员`,
          data: group
        }));

        this.searchResults = [...this.searchResults, ...groupResults];
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `搜索群组失败: ${error}`);
    }
  }

  /**
   * 搜索瞬间
   * 使用 searchMomentsByKeyword API
   */
  private async searchMoments(query: string) {
    if (!this.currentLocation) {
      return;
    }

    try {
      // 使用关键字搜索
      const response = await ApiClient.searchMomentsByKeyword({
        kw: query,
        latitude: this.currentLocation.latitude,
        longitude: this.currentLocation.longitude,
        limit: 10
      });

      if (response.code === 0 && response.data) {
        const momentResults: SearchResult[] = response.data.items.map((moment: MomentSearchItem): SearchResult => ({
          id: `moment_${moment.moment_id}`,
          type: 'moment',
          title: moment.user.nickname,
          subtitle: moment.description.length > 30 ?
            moment.description.substring(0, 30) + '...' : moment.description,
          data: moment
        }));

        this.searchResults = [...this.searchResults, ...momentResults];
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `搜索瞬间失败: ${error}`);
    }
  }

  /**
   * 保存搜索历史
   */
  private saveSearchHistory(query: string) {
    if (!this.searchHistory.includes(query)) {
      this.searchHistory.unshift(query);
      if (this.searchHistory.length > 10) {
        this.searchHistory = this.searchHistory.slice(0, 10);
      }
      // 这里可以保存到本地存储
    }
  }

  /**
   * 处理搜索输入变化
   */
  private onSearchTextChange(value: string) {
    this.searchText = value;

    // 清除之前的定时器
    if (this.searchTimer !== -1) {
      clearTimeout(this.searchTimer);
    }

    // 防抖搜索
    this.searchTimer = setTimeout(() => {
      this.searchResults = []; // 清空之前的结果
      this.performSearch(value);
    }, 300);
  }

  /**
   * 处理搜索结果点击
   */
  private handleResultClick(result: SearchResult) {
    this.showResults = false;

    switch (result.type) {
      case 'user':
        // 导航到用户资料或开始聊天
        SystemToast.show(`查看用户: ${result.title}`, this.getUIContext());
        break;
      case 'group':
        // 导航到群组详情
        const group = result.data as GroupSearchItem;
        try {
          this.getUIContext().getRouter().pushUrl({
            url: `features/group/pages/${PageNames.GROUP_DETAIL}`,
            params: { groupId: group.group_chat_id }
          });
        } catch (error) {
          hilog.error(DOMAIN_NUMBER, TAG, `跳转群组详情失败: ${error}`);
        }
        break;
      case 'moment':
        // 显示瞬间详情
        SystemToast.show(`查看瞬间: ${result.title}`, this.getUIContext());
        break;
    }
  }

  /**
   * 切换筛选器
   */
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
      // 搜索框
      Row() {
        TextInput({
          placeholder: '搜索用户、群组、瞬间...',
          text: this.searchText
        })
          .layoutWeight(1)
          .onChange((value: string) => this.onSearchTextChange(value))
          .onFocus(() => {
            if (!this.searchText.trim() && this.searchHistory.length > 0) {
              this.showResults = true;
            }
          })

        if (this.isSearching) {
          LoadingProgress()
            .width(24)
            .height(24)
            .margin({ left: 8 })
        } else {
          Button('🔍')
            .width(40)
            .height(40)
            .fontSize(16)
            .backgroundColor(Color.Transparent)
            .margin({ left: 8 })
            .onClick(() => this.performSearch(this.searchText))
        }
      }
      .width('100%')
      .padding(12)

      // 筛选器
      if (this.showResults) {
        Row() {
          Button('全部')
            .fontSize(12)
            .backgroundColor(this.currentFilter === 'all' ?
            $r('sys.color.ohos_id_color_emphasize') : Color.Transparent)
            .fontColor(this.currentFilter === 'all' ?
            Color.White : $r('sys.color.ohos_id_color_text_primary'))
            .onClick(() => this.switchFilter('all'))

          Button('用户')
            .fontSize(12)
            .backgroundColor(this.currentFilter === 'users' ?
            $r('sys.color.ohos_id_color_emphasize') : Color.Transparent)
            .fontColor(this.currentFilter === 'users' ?
            Color.White : $r('sys.color.ohos_id_color_text_primary'))
            .onClick(() => this.switchFilter('users'))

          Button('群组')
            .fontSize(12)
            .backgroundColor(this.currentFilter === 'groups' ?
            $r('sys.color.ohos_id_color_emphasize') : Color.Transparent)
            .fontColor(this.currentFilter === 'groups' ?
            Color.White : $r('sys.color.ohos_id_color_text_primary'))
            .onClick(() => this.switchFilter('groups'))

          Button('瞬间')
            .fontSize(12)
            .backgroundColor(this.currentFilter === 'moments' ?
            $r('sys.color.ohos_id_color_emphasize') : Color.Transparent)
            .fontColor(this.currentFilter === 'moments' ?
            Color.White : $r('sys.color.ohos_id_color_text_primary'))
            .onClick(() => this.switchFilter('moments'))
        }
        .width('100%')
        .padding({ left: 12, right: 12, bottom: 8 })
        .justifyContent(FlexAlign.SpaceAround)
      }

      // 搜索结果或历史
      if (this.showResults) {
        List() {
          if (this.searchText.trim() === '' && this.searchHistory.length > 0) {
            // 显示搜索历史
            ListItem() {
              Text('搜索历史')
                .fontSize(14)
                .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                .padding(12)
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
                .padding(12)
                .onClick(() => {
                  this.searchText = historyItem;
                  this.performSearch(historyItem);
                })
              }
            })
          } else if (this.searchResults.length > 0) {
            // 显示搜索结果
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
                .padding(12)
                .onClick(() => this.handleResultClick(result))
              }
            })
          } else if (this.searchText.trim() !== '' && !this.isSearching) {
            // 无搜索结果
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
        .backgroundColor($r('sys.color.ohos_id_color_background'))
        .borderRadius({ topLeft: 12, topRight: 12 })
        .shadow({
          radius: 8,
          color: Color.Black,
          offsetX: 0,
          offsetY: -2
        })
      }
    }
    .width('100%')
    .height('100%')
    .onClick(() => {
      // 点击空白区域隐藏搜索结果
      if (this.showResults) {
        this.showResults = false;
      }
    })
  }

  /**
   * 获取结果图标
   */
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
