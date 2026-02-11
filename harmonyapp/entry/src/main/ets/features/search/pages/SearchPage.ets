/**
 * 全局搜索页面
 * 重构：使用 HarmonyOS 原生 Tabs 组件
 */

import { GlobalStateManager, KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { ApiClient } from '../../../core/api/ApiClient';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { PageNames } from '../../../constants/AppConstants';
import { AppRouter, GroupDetailRouteParam } from '../../../core/router/AppRouter';
import { UserItem, GroupItem, MomentItem, GeoPoint } from '../../../core/types/api';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon, SystemIcon, IconSize, IconButton } from '../../../components/SystemIcon';
import { SystemCard } from '../../../components/SystemCard';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';

const DOMAIN_NUMBER: number = 0xFF10;
const TAG: string = 'SearchPage';

@Component
export struct SearchPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  
  @State currentIndex: number = 0;
  @State keyword: string = '';
  @State isLoading: boolean = false;
  @State userResults: UserItem[] = [];
  private userResultsDataSource: BasicDataSource<UserItem> = new BasicDataSource<UserItem>([]);
  @State groupResults: GroupItem[] = [];
  private groupResultsDataSource: BasicDataSource<GroupItem> = new BasicDataSource<GroupItem>([]);
  @State momentResults: MomentItem[] = [];
  private momentResultsDataSource: BasicDataSource<MomentItem> = new BasicDataSource<MomentItem>([]);
  @State hasSearched: boolean = false;
  private controller: TabsController = new TabsController();

  private async doSearch() {
    if (!this.keyword.trim()) return;
    if (!this.currentLocation) {
      SystemToast.show('请先获取位置信息', this.getUIContext());
      return;
    }

    this.isLoading = true;
    this.hasSearched = true;
    this.userResults = [];
    this.userResultsDataSource.clear();
    this.groupResults = [];
    this.groupResultsDataSource.clear();
    this.momentResults = [];
    this.momentResultsDataSource.clear();

    const referencePoint: GeoPoint = {
      latitude: this.currentLocation.latitude,
      longitude: this.currentLocation.longitude
    };

    try {
      await LoadingUtil.withMinimumTime(async () => {

        const p1 = ApiClient.searchUsers({
          keyword: this.keyword,
          reference_point: referencePoint
        });
        const p2 = ApiClient.searchGroups({
          keyword: this.keyword,
          reference_point: referencePoint
        });
        const p3 = ApiClient.searchMoments({
          keyword: this.keyword,
          reference_point: referencePoint
        });

        const results = await Promise.all([p1, p2, p3]);
        const res1 = results[0];
        const res2 = results[1];
        const res3 = results[2];

        const myId = await AuthAccountManager.getCurrentUserId();
        if (res1.code === 0 && res1.data) {
          const validUsers = res1.data.items.filter((u: UserItem) => u.user?.user_id);
          this.userResults = (myId !== null && typeof myId === 'string')
            ? validUsers.filter((u: UserItem) => u.user.user_id !== myId)
            : validUsers;
          this.userResultsDataSource.setData(this.userResults);
        }
        if (res2.code === 0 && res2.data) {
          this.groupResults = res2.data.items.filter((g: GroupItem) => g.group?.group_id);
          this.groupResultsDataSource.setData(this.groupResults);
        }
        if (res3.code === 0 && res3.data) {
          this.momentResults = res3.data.items.filter((m: MomentItem) => m.moment_id && m.user);
          this.momentResultsDataSource.setData(this.momentResults);
        }
      });
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `搜索失败: ${error}`);
      SystemToast.showError('搜索失败', this.getUIContext());
    } finally {
      this.isLoading = false;
    }
  }

  build() {
    NavDestination() {
    Column() {
      Row() {
        IconButton({
          icon: AppIcon.BACK,
          iconSize: IconSize.LARGE,
          iconColor: $r('sys.color.ohos_id_color_text_primary'),
          bgColor: Color.Transparent,
          onButtonClick: () => {
            AppRouter.pop();
          }
        })

        TextInput({ placeholder: '搜索用户、群组、瞬间...', text: this.keyword })
          .layoutWeight(1)
          .height(40)
          .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
          .borderRadius(20)
          .padding({ left: 16, right: 16 })
          .margin({ left: 8, right: 16 })
          .onChange((value) => {
            this.keyword = value;
          })
          .onSubmit(() => {
            this.doSearch();
          })
      }
      .width('100%')
      .height(56)
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      if (this.isLoading) {
        SystemLoadingView({ message: '搜索中...' }).layoutWeight(1)
      } else {
        Tabs({ barPosition: BarPosition.Start, controller: this.controller, index: this.currentIndex }) {
          TabContent() {
            this.UserResultList()
          }.tabBar(this.TabBuilder(0, '用户'))

          TabContent() {
            this.GroupResultList()
          }.tabBar(this.TabBuilder(1, '群组'))

          TabContent() {
            this.MomentResultList()
          }.tabBar(this.TabBuilder(2, '瞬间'))
        }
        .vertical(false)
        .barMode(BarMode.Fixed)
        .barWidth('100%')
        .barHeight(56)
        .animationDuration(300)
        .onChange((index: number) => {
          this.currentIndex = index;
        })
        .width('100%')
        .layoutWeight(1)
        .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
    })
  }

  @Builder TabBuilder(index: number, name: string) {
    Column() {
      Text(name)
        .fontColor(this.currentIndex === index ? $r('sys.color.ohos_id_color_emphasize') : $r('sys.color.ohos_id_color_text_secondary'))
        .fontSize(16)
        .fontWeight(this.currentIndex === index ? FontWeight.Medium : FontWeight.Normal)
        .lineHeight(22)
        .margin({ top: 17, bottom: 7 })
      Divider()
        .strokeWidth(2)
        .color($r('sys.color.ohos_id_color_emphasize'))
        .opacity(this.currentIndex === index ? 1 : 0)
        .width(this.currentIndex === index ? 20 : 0)
        .animation({ duration: 250, curve: Curve.FastOutSlowIn })
    }.width('100%')
  }

  @Builder
  UserResultList() {
    if (this.hasSearched && this.userResults.length === 0) {
      Column({ space: 16 }) {
        Text('🔍').fontSize(48)
        Text('未找到相关结果').fontSize(16).fontColor($r('sys.color.ohos_id_color_text_secondary'))
      }
      .width('100%').height('100%').justifyContent(FlexAlign.Center)
    } else {
      List() {
        LazyForEach(this.userResultsDataSource, (item: UserItem) => {
          ListItem() {
            this.buildUserItem(item)
          }
        }, (item: UserItem) => item.user?.user_id ?? '')
      }
      .width('100%').height('100%')
      .padding({ left: 16, right: 16, top: 16 })
    }
  }

  @Builder
  GroupResultList() {
    if (this.hasSearched && this.groupResults.length === 0) {
      Column({ space: 16 }) {
        Text('🔍').fontSize(48)
        Text('未找到相关结果').fontSize(16).fontColor($r('sys.color.ohos_id_color_text_secondary'))
      }
      .width('100%').height('100%').justifyContent(FlexAlign.Center)
    } else {
      List() {
        LazyForEach(this.groupResultsDataSource, (item: GroupItem) => {
          ListItem() {
            this.buildGroupItem(item)
          }
        }, (item: GroupItem) => item.group?.group_id ?? '')
      }
      .width('100%').height('100%')
      .padding({ left: 16, right: 16, top: 16 })
    }
  }

  @Builder
  MomentResultList() {
    if (this.hasSearched && this.momentResults.length === 0) {
      Column({ space: 16 }) {
        Text('🔍').fontSize(48)
        Text('未找到相关结果').fontSize(16).fontColor($r('sys.color.ohos_id_color_text_secondary'))
      }
      .width('100%').height('100%').justifyContent(FlexAlign.Center)
    } else {
      List() {
        LazyForEach(this.momentResultsDataSource, (item: MomentItem) => {
          ListItem() {
            this.buildMomentItem(item)
          }
        }, (item: MomentItem) => item.moment_id)
      }
      .width('100%').height('100%')
      .padding({ left: 16, right: 16, top: 16 })
    }
  }

  @Builder
  buildUserItem(item: UserItem) {
    Row() {
      SystemAvatar({
        avatarUrl: item.user?.avatar_url ?? '',
        name: item.user?.nickname ?? '',
        avatarSize: 50
      })
      .margin({ right: 12 })

      Column({ space: 4 }) {
        Text(item.user?.nickname ?? '')
          .fontSize(16)
          .fontWeight(FontWeight.Medium)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
        
        if (item.bio) {
          Text(item.bio)
            .fontSize(14)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)
    }
    .width('100%')
    .padding(16)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .borderRadius(DesignConstants.RADIUS.LG)
    .margin({ bottom: 12 })
  }

  @Builder
  buildGroupItem(item: GroupItem) {
    Row() {
      SystemAvatar({
        avatarUrl: item.group?.avatar_url ?? '',
        name: item.group?.name ?? '',
        avatarSize: 50,
        shape: 'square'
      })
      .margin({ right: 12 })

      Column({ space: 4 }) {
        Text(item.group?.name ?? '')
          .fontSize(16)
          .fontWeight(FontWeight.Medium)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
        
        Text(`${item.group?.members_count ?? 0} 成员`)
          .fontSize(12)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))

        if (item.description) {
          Text(item.description)
            .fontSize(14)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)
    }
    .width('100%')
    .padding(16)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .borderRadius(DesignConstants.RADIUS.LG)
    .margin({ bottom: 12 })
    .onClick(() => {
      const p = new GroupDetailRouteParam();
      p.groupId = item.group?.group_id ?? '';
      AppRouter.push(PageNames.GROUP_DETAIL, p);
    })
  }

  @Builder
  buildMomentItem(item: MomentItem) {
    Column({ space: 8 }) {
      Row() {
        SystemAvatar({
          avatarUrl: item.user?.avatar_url ?? '',
          name: item.user?.nickname ?? '',
          avatarSize: 32
        })
        .margin({ right: 8 })

        Text(item.user?.nickname ?? '')
          .fontSize(14)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
      }

      Text(item.description)
        .fontSize(16)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
        .maxLines(3)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
    }
    .width('100%')
    .padding(16)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .borderRadius(DesignConstants.RADIUS.LG)
    .margin({ bottom: 12 })
  }
}

@Builder export function SearchPageBuilder() {
  SearchPage();
}
