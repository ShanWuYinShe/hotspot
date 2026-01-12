/**
 * 极简群组页面
 * 使用HarmonyOS原生能力，去除复杂抽象
 * 优化：添加左滑退出群组功能
 */

import { GlobalStateManager, KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { LocationUtil } from '../../../core/location/LocationUtil';
import { PageNames } from '../../../constants/AppConstants';
import { GroupSearchItem } from '../../../core/types/api';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { IntelligentLocationCard } from '../../../components/IntelligentLocationCard';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants } from '../../../constants/DesignConstants';
import { SystemCard } from '../../../components/SystemCard';

export struct GroupPage {
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  
  private stateManager = GlobalStateManager.getInstance();
  @State groups: GroupSearchItem[] = [];
  @State isLoading: boolean = false;
  @State errorMessage: string = '';

  async aboutToAppear() {
    await this.loadNearbyGroups();
  }

  aboutToDisappear() {
    // 
  }

  /**
   * 加载附近群组
   */
  private async loadNearbyGroups() {
    if (!this.currentLocation) {
      this.errorMessage = '请先获取位置信息';
      return;
    }

    this.isLoading = true;
    this.errorMessage = '';

    try {
      const response = await ApiClient.searchGroupsByLocation({
        latitude: this.currentLocation.latitude,
        longitude: this.currentLocation.longitude,
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
    const currentUser = this.stateManager.getCurrentUser();
    if (!currentUser) {
      SystemToast.showWarning('请先登录后再加入群组', this.getUIContext());
      try {
        const routerInstance = this.getUIContext().getRouter();
        await routerInstance.pushUrl({
          url: `features/auth/pages/${PageNames.LOGIN}`,
          params: { reason: 'action_required', action: '加入群组' }
        });
      } catch (error) {
        // ignore
      }
      return;
    }

    if (group.has_password) {
      SystemToast.show('此群组需要密码，请进入详情页加入', this.getUIContext());
      // 跳转到详情页
      try {
        const routerInstance = this.getUIContext().getRouter();
        await routerInstance.pushUrl({
          url: `features/group/pages/${PageNames.GROUP_DETAIL}`,
          params: { groupId: group.group_chat_id }
        });
      } catch (error) {
        SystemToast.showError('跳转失败', this.getUIContext());
      }
      return;
    }

    try {
      await ApiClient.joinGroup({ id: group.group_chat_id });
      SystemToast.show('加入群组成功', this.getUIContext());
      // 自动刷新列表以更新状态（如果需要）或者跳转到群组聊天页面
      await this.loadNearbyGroups(); 
    } catch (error) {
      // 错误处理优化
      const errMsg = error instanceof Error ? error.message : '加入群组失败';
      if (errMsg.includes('1305')) { // 假设 1305 是 "已在群组中"
          SystemToast.show('您已是该群成员', this.getUIContext());
      } else {
          SystemToast.showError(errMsg, this.getUIContext());
      }
    }
  }

  /**
   * 创建群组
   */
  private async createGroup() {
    const currentUser = this.stateManager.getCurrentUser();
    if (!currentUser) {
      SystemToast.showWarning('请先登录后再创建群组', this.getUIContext());
      try {
        const routerInstance = this.getUIContext().getRouter();
        await routerInstance.pushUrl({
          url: `features/auth/pages/${PageNames.LOGIN}`,
          params: { reason: 'action_required', action: '创建群组' }
        });
      } catch (error) {
        // ignore
      }
      return;
    }

    if (!this.currentLocation) {
      SystemToast.show('请先获取位置信息', this.getUIContext());
      return;
    }

    try {
      const routerInstance = this.getUIContext().getRouter();
      await routerInstance.pushUrl({ url: `features/group/pages/${PageNames.CREATE_GROUP}` });
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
          .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
          .layoutWeight(1)

        Button('创建')
          .type(ButtonType.Capsule)
          .backgroundColor(DesignConstants.COLOR.BRAND_PRIMARY)
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
      .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)

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
            .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
            .textAlign(TextAlign.Center)

          Button('重试')
            .type(ButtonType.Capsule)
            .backgroundColor(DesignConstants.COLOR.BRAND_PRIMARY)
            .fontColor(Color.White)
            .onClick(() => this.loadNearbyGroups())
        }
        .width('100%')
        .height('100%')
        .justifyContent(FlexAlign.Center)
        .padding(24)
      } else if (this.groups.length === 0) {
        // 内联空状态视图
        Refresh({ refreshing: $$this.isLoading, builder: this.RefreshBuilder() }) {
          Column({ space: 16 }) {
            Text('👥')
              .fontSize(48)

            Text('附近暂无群组\n快来创建第一个吧！')
              .fontSize(16)
              .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
              .textAlign(TextAlign.Center)
          }
          .width('100%')
          .height('100%')
          .justifyContent(FlexAlign.Center)
        }
        .onRefreshing(() => {
          this.loadNearbyGroups();
        })
      } else {
        Refresh({ refreshing: $$this.isLoading, builder: this.RefreshBuilder() }) {
          this.buildGroupList()
        }
        .onRefreshing(() => {
          this.loadNearbyGroups();
        })
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor(DesignConstants.COLOR.SECONDARY_BG_LIGHT)
  }

  @Builder
  RefreshBuilder() {
    Row() {
      LoadingProgress().width(24).height(24)
      Text('正在刷新...').fontSize(14).margin({left: 8}).fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
    }
    .width('100%')
    .justifyContent(FlexAlign.Center)
    .padding(10)
  }

  @Builder
  private buildGroupList() {
    List({ space: 12 }) {
      ForEach(this.groups, (group: GroupSearchItem) => {
        ListItem() {
          this.buildGroupCard(group)
        }
      }, (group: GroupSearchItem) => group.group_chat_id)
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
  private buildGroupCard(group: GroupSearchItem) {
    SystemCard({
      content: () => {
        this.GroupCardContent(group);
      },
      backgroundColor: DesignConstants.COLOR.PRIMARY_BG_LIGHT
    })
    .onClick(async () => {
      try {
        const routerInstance = this.getUIContext().getRouter();
        await routerInstance.pushUrl({
          url: `features/group/pages/${PageNames.GROUP_DETAIL}`,
          params: { groupId: group.group_chat_id }
        });
      } catch (error) {
        SystemToast.showError('跳转失败', this.getUIContext());
      }
    })
  }

  @Builder
  private GroupCardContent(group: GroupSearchItem) {
    Column({ space: 12 }) {
      // 群组基本信息
      Row({ space: 12 }) {
        SystemAvatar({
          avatarUrl: group.avatar_url || '',
          nickname: group.name,
          size: 48
        })

        Column({ space: 4 }) {
          Text(group.name)
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontWeight(FontWeight.Medium)
            .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })

          Text(`${group.members_count}人`)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)

          if (group.description) {
            Text(group.description)
              .fontSize(DesignConstants.FONT.SIZE.MD)
              .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
              .maxLines(2)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
          }
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)

        // 加入按钮
        Button('加入')
          .type(ButtonType.Capsule)
          .backgroundColor(DesignConstants.COLOR.BRAND_PRIMARY)
          .fontColor(Color.White)
          .fontSize(12)
          .padding({
            left: 12,
            right: 12,
            top: 6,
            bottom: 6
          })
          .onClick(() => this.joinGroup(group))
      }
      .width('100%')
      .alignItems(VerticalAlign.Top)

      // 位置信息
      if (group.location) {
        IntelligentLocationCard({
          latitude: group.location.latitude,
          longitude: group.location.longitude,
          locationName: group.location.location_name,
          distance: group.location.distance,
          altitude: 0 // 列表数据暂时不包含海拔信息
        })
      }
    }
    .width('100%')
  }
}
