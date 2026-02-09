/**
 * 全局搜索页面
 * 重构：使用 HarmonyOS 原生 Tabs 组件
 */

import { GlobalStateManager, KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { ApiClient } from '../../../core/api/ApiClient';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { PageNames } from '../../../constants/AppConstants';
import { UserSearchItem, GroupSearchItem, MomentSearchItem } from '../../../core/types/api';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon, SystemIcon, IconSize, IconButton } from '../../../components/SystemIcon';
import { SystemCard } from '../../../components/SystemCard';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';

const DOMAIN_NUMBER: number = 0xFF10;
const TAG: string = 'SearchPage';

@Entry
@Component
struct SearchPage {
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  
  @State currentIndex: number = 0;
  @State keyword: string = '';
  @State isLoading: boolean = false;
  @State userResults: UserSearchItem[] = [];
  @State groupResults: GroupSearchItem[] = [];
  @State momentResults: MomentSearchItem[] = [];
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
    this.groupResults = [];
    this.momentResults = [];

    try {
      const lat = this.currentLocation.latitude;
      const lng = this.currentLocation.longitude;

      // 并行搜索所有类型，或者根据当前tab搜索
      // 为了简单体验，这里一次性搜索所有类型，或者也可以只搜索当前tab
      // 考虑到Tabs可以滑动，预加载比较好。
      
      const p1 = ApiClient.searchUsersByKeyword({ 
        kw: this.keyword, 
        reference_point: { latitude: lat, longitude: lng }
      });
      const p2 = ApiClient.searchGroupsByKeyword({ 
        kw: this.keyword, 
        reference_point: { latitude: lat, longitude: lng }
      });
      const p3 = ApiClient.searchMomentsByKeyword({ 
        kw: this.keyword, 
        reference_point: { latitude: lat, longitude: lng }
      });

      const results = await Promise.all([p1, p2, p3]);
      const res1 = results[0];
      const res2 = results[1];
      const res3 = results[2];

      // 过滤掉“自己”（避免搜索结果里出现当前用户）
      const myId = await AuthAccountManager.getCurrentUserId();
      if (res1.code === 0 && res1.data) {
        this.userResults = (myId !== null && typeof myId === 'number')
          ? res1.data.items.filter((u: UserSearchItem) => u.user_id !== myId)
          : res1.data.items;
      }
      if (res2.code === 0 && res2.data) this.groupResults = res2.data.items;
      if (res3.code === 0 && res3.data) this.momentResults = res3.data.items;

    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `搜索失败: ${error}`);
      SystemToast.showError('搜索失败', this.getUIContext());
    } finally {
      this.isLoading = false;
    }
  }

  build() {
    Column() {
      // 搜索栏
      Row() {
        IconButton({
          icon: AppIcon.BACK,
          iconSize: IconSize.LARGE,
          iconColor: DesignConstants.COLOR.TEXT_PRIMARY_LIGHT,
          bgColor: Color.Transparent,
          onButtonClick: () => {
            const routerInstance = this.getUIContext().getRouter();
            routerInstance.back();
          }
        })

        TextInput({ placeholder: '搜索用户、群组、瞬间...', text: this.keyword })
          .layoutWeight(1)
          .height(40)
          .backgroundColor(DesignConstants.COLOR.TERTIARY_BG_LIGHT)
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
      .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)

      if (this.isLoading) {
        SystemLoadingView({ message: '搜索中...' }).layoutWeight(1)
      } else {
        Tabs({ barPosition: BarPosition.Start, controller: this.controller, index: this.currentIndex }) {
          TabContent() {
            this.ResultList(this.userResults, 'user')
          }.tabBar(this.TabBuilder(0, '用户'))

          TabContent() {
            this.ResultList(this.groupResults, 'group')
          }.tabBar(this.TabBuilder(1, '群组'))

          TabContent() {
            this.ResultList(this.momentResults, 'moment')
          }.tabBar(this.TabBuilder(2, '瞬间'))
        }
        .vertical(false)
        .barMode(BarMode.Fixed)
        .barWidth('100%')
        .barHeight(56)
        .animationDuration(200)
        .onChange((index: number) => {
          this.currentIndex = index;
        })
        .width('100%')
        .layoutWeight(1)
        .backgroundColor(DesignConstants.COLOR.SECONDARY_BG_LIGHT)
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor(DesignConstants.COLOR.SECONDARY_BG_LIGHT)
  }

  @Builder TabBuilder(index: number, name: string) {
    Column() {
      Text(name)
        .fontColor(this.currentIndex === index ? DesignConstants.COLOR.BRAND_PRIMARY : DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
        .fontSize(16)
        .fontWeight(this.currentIndex === index ? FontWeight.Medium : FontWeight.Normal)
        .lineHeight(22)
        .margin({ top: 17, bottom: 7 })
      Divider()
        .strokeWidth(2)
        .color(DesignConstants.COLOR.BRAND_PRIMARY)
        .opacity(this.currentIndex === index ? 1 : 0)
        .width(20)
    }.width('100%')
  }

  @Builder
  ResultList(data: any[], type: 'user' | 'group' | 'moment') {
    if (this.hasSearched && data.length === 0) {
      Column({ space: 16 }) {
        Text('🔍').fontSize(48)
        Text('未找到相关结果').fontSize(16).fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
      }
      .width('100%').height('100%').justifyContent(FlexAlign.Center)
    } else {
      List() {
        ForEach(data, (item: any) => {
          ListItem() {
            if (type === 'user') this.buildUserItem(item as UserSearchItem);
            else if (type === 'group') this.buildGroupItem(item as GroupSearchItem);
            else this.buildMomentItem(item as MomentSearchItem);
          }
        })
      }
      .width('100%').height('100%')
      .padding({ left: 16, right: 16, top: 16 })
    }
  }

  @Builder
  buildUserItem(item: UserSearchItem) {
    Row() {
      SystemAvatar({
        avatarUrl: item.avatar_url,
        name: item.nickname,
        avatarSize: 50
      })
      .margin({ right: 12 })

      Column({ space: 4 }) {
        Text(item.nickname)
          .fontSize(16)
          .fontWeight(FontWeight.Medium)
          .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
        
        if (item.bio) {
          Text(item.bio)
            .fontSize(14)
            .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)
    }
    .width('100%')
    .padding(16)
    .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
    .borderRadius(DesignConstants.RADIUS.LG)
    .margin({ bottom: 12 })
  }

  @Builder
  buildGroupItem(item: GroupSearchItem) {
    Row() {
      SystemAvatar({
        avatarUrl: item.avatar_url,
        name: item.name,
        avatarSize: 50,
        shape: 'square'
      })
      .margin({ right: 12 })

      Column({ space: 4 }) {
        Text(item.name)
          .fontSize(16)
          .fontWeight(FontWeight.Medium)
          .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
        
        Text(`${item.members_count} 成员`)
          .fontSize(12)
          .fontColor(DesignConstants.COLOR.TEXT_TERTIARY_LIGHT)

        if (item.description) {
          Text(item.description)
            .fontSize(14)
            .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)
    }
    .width('100%')
    .padding(16)
    .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
    .borderRadius(DesignConstants.RADIUS.LG)
    .margin({ bottom: 12 })
    .onClick(async () => {
      const routerInstance = this.getUIContext().getRouter();
      await routerInstance.pushUrl({
        url: `features/group/pages/${PageNames.GROUP_DETAIL}`,
        params: { groupId: item.group_id }
      });
    })
  }

  @Builder
  buildMomentItem(item: MomentSearchItem) {
    Column({ space: 8 }) {
      Row() {
        SystemAvatar({
          avatarUrl: item.user.avatar_url,
          name: item.user.nickname,
          avatarSize: 32
        })
        .margin({ right: 8 })

        Text(item.user.nickname)
          .fontSize(14)
          .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
      }

      Text(item.description)
        .fontSize(16)
        .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
        .maxLines(3)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
    }
    .width('100%')
    .padding(16)
    .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
    .borderRadius(DesignConstants.RADIUS.LG)
    .margin({ bottom: 12 })
  }
}
