/**
 * 极简群组页面
 * 使用HarmonyOS原生能力，去除复杂抽象
 */

import { GlobalStateManager, GlobalState } from '../common/state/GlobalStateManager';
import { ApiClient } from '../common/api/ApiClient';
import { LocationUtil } from '../common/location/LocationUtil';
import { PageNames } from '../constants/AppConstants';
import { GroupSearchItem } from '../common/types/api';
import { SystemAvatar } from '../components/SystemAvatar';
import { IntelligentLocationCard } from '../components/IntelligentLocationCard';
import { SystemLoadingView } from '../components/SystemLoadingView';
import { SystemToast } from '../components/SystemToast';


@Entry
@Component
struct GroupPage {
  @State globalState: GlobalState = GlobalStateManager.getInstance().getState();
  private stateManager = GlobalStateManager.getInstance();
  @State groups: GroupSearchItem[] = [];
  @State isLoading: boolean = false;
  @State errorMessage: string = '';

  async aboutToAppear() {
    // 订阅状态变化
    this.stateManager.subscribe('group_page', (state: GlobalState) => {
      this.globalState = state;
    });
    await this.loadNearbyGroups();
  }

  aboutToDisappear() {
    // 取消订阅
    this.stateManager.unsubscribe('group_page');
  }

  /**
   * 加载附近群组
   */
  private async loadNearbyGroups() {
    if (!this.globalState.currentLocation) {
      this.errorMessage = '请先获取位置信息';
      return;
    }

    this.isLoading = true;
    this.errorMessage = '';

    try {
      const response = await ApiClient.searchGroupsByLocation({
        latitude: this.globalState.currentLocation.latitude,
        longitude: this.globalState.currentLocation.longitude,
        radius: 5000, // 5km 范围
        limit: 50     // 最多50个群组
      });

      if (response.code === 0 && response.data) {
        this.groups = response.data.items;
      } else {
        this.errorMessage = response.message || '加载群组失败';
        this.groups = [];
      }
    } catch (error) {
      this.errorMessage = error instanceof Error ? error.message : '加载群组失败';
    } finally {
      this.isLoading = false;
    }
  }

  /**
   * 加入群组
   */
  private async joinGroup(group: GroupSearchItem) {
    try {
      await ApiClient.joinGroup({ id: group.id });
      SystemToast.show('加入群组成功', this.getUIContext());
      await this.loadNearbyGroups(); // 刷新列表
    } catch (error) {
      SystemToast.showError(error instanceof Error ? error.message : '加入群组失败', this.getUIContext());
    }
  }

  /**
   * 创建群组
   */
  private async createGroup() {
    if (!this.globalState.currentLocation) {
      SystemToast.show('请先获取位置信息', this.getUIContext());
      return;
    }

    try {
      const routerInstance = this.getUIContext().getRouter();
      await routerInstance.pushUrl({ url: `pages/${PageNames.CREATE_GROUP}` });
    } catch (error) {
      SystemToast.showError('跳转失败', this.getUIContext());
    }
  }

  build() {
    Column() {
      // 标题栏
      Row() {
        Text('附近群组')
          .fontSize(20)
          .fontWeight(FontWeight.Bold)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .layoutWeight(1)

        Button('创建')
          .type(ButtonType.Capsule)
          .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
          .fontColor(Color.White)
          .fontSize(14)
          .padding({
            left: 16,
            right: 16,
            top: 8,
            bottom: 8
          })
          .onClick(() => this.createGroup())
      }
      .width('100%')
      .padding(16)
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      // 内容区域
      if (this.isLoading) {
        SystemLoadingView({ message: '正在加载附近群组...' })
      } else if (this.errorMessage) {
        // 内联错误视图
        Column({ space: 16 }) {
          Text('⚠️')
            .fontSize(48)

          Text(this.errorMessage)
            .fontSize(16)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .textAlign(TextAlign.Center)

          Button('重试')
            .type(ButtonType.Capsule)
            .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
            .fontColor(Color.White)
            .onClick(() => this.loadNearbyGroups())
        }
        .width('100%')
        .height('100%')
        .justifyContent(FlexAlign.Center)
        .padding(24)
      } else if (this.groups.length === 0) {
        // 内联空状态视图
        Column({ space: 16 }) {
          Text('👥')
            .fontSize(48)

          Text('附近暂无群组\n快来创建第一个吧！')
            .fontSize(16)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .textAlign(TextAlign.Center)
        }
        .width('100%')
        .height('100%')
        .justifyContent(FlexAlign.Center)
      } else {
        this.buildGroupList()
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
  }

  @Builder
  private buildGroupList() {
    List({ space: 12 }) {
      ForEach(this.groups, (group: SearchGroupByLocationResponse) => {
        ListItem() {
          this.buildGroupCard(group)
        }
      }, (group: SearchGroupByLocationResponse) => group.group_id)
    }
    .width('100%')
    .layoutWeight(1)
    .padding({
      left: 16,
      right: 16,
      top: 8,
      bottom: 16
    })
    .scrollBar(BarState.Auto)
  }

  @Builder
  private buildGroupCard(group: SearchGroupByLocationResponse) {
    Column({ space: 12 }) {
      // 群组基本信息
      Row({ space: 12 }) {
        SystemAvatar({
          avatarUrl: group.avatar_url || '',
          nickname: group.name,
          avatarSize: 48
        })

        Column({ space: 4 }) {
          Text(group.name)
            .fontSize(16)
            .fontWeight(FontWeight.Medium)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })

          Text(`${group.member_count}人 · ${group.creator_name}创建`)
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))

          if (group.description) {
            Text(group.description)
              .fontSize(14)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
              .maxLines(2)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
          }
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)

        // 加入按钮
        if (!group.is_member) {
          Button('加入')
            .type(ButtonType.Capsule)
            .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
            .fontColor(Color.White)
            .fontSize(12)
            .padding({
              left: 12,
              right: 12,
              top: 6,
              bottom: 6
            })
            .onClick(() => this.joinGroup(group))
        } else {
          Text('已加入')
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .padding({
              left: 12,
              right: 12,
              top: 6,
              bottom: 6
            })
        }
      }
      .width('100%')
      .alignItems(VerticalAlign.Top)

      // 位置信息
      IntelligentLocationCard({
        latitude: group.latitude,
        longitude: group.longitude,
        locationName: group.location_name,
        distance: group.distance
      })
    }
    .width('100%')
    .padding(16)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .borderRadius(12)
    .onClick(async () => {
      try {
        const routerInstance = this.getUIContext().getRouter();
        await routerInstance.pushUrl({
          url: `pages/${PageNames.GROUP_DETAIL}`,
          params: { groupId: group.group_id }
        });
      } catch (error) {
        SystemToast.showError('跳转失败', this.getUIContext());
      }
    })
  }
}
