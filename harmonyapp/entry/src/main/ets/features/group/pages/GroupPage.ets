/**
 * 群组页面
 * 使用 HarmonyOS 原生能力 + 系统颜色资源（支持深色模式）
 */

import { GlobalStateManager, KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { PageNames } from '../../../constants/AppConstants';
import { GroupSearchItem, SearchGroupsByLocationRequest } from '../../../core/types/api';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { IntelligentLocationCard } from '../../../components/IntelligentLocationCard';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';

@Component
export struct GroupPage {
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;

  private stateManager = GlobalStateManager.getInstance();
  @State groups: GroupSearchItem[] = [];
  @State isLoading: boolean = false;
  @State errorMessage: string = '';

  async aboutToAppear() {
    await this.loadNearbyGroups();
  }

  aboutToDisappear() {}

  private async loadNearbyGroups() {
    if (!this.currentLocation) {
      this.errorMessage = '请先获取位置信息';
      return;
    }

    this.isLoading = true;
    this.errorMessage = '';

    try {
      const request: SearchGroupsByLocationRequest = {
        reference_point: {
          latitude: this.currentLocation.latitude,
          longitude: this.currentLocation.longitude
        },
        radius: 5000,
        limit: 50
      };
      const response = await ApiClient.searchGroupsByLocation(request);

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
      try {
        const routerInstance = this.getUIContext().getRouter();
        await routerInstance.pushUrl({
          url: `features/group/pages/${PageNames.GROUP_DETAIL}`,
          params: { groupId: group.group_id }
        });
      } catch (error) {
        SystemToast.showError('跳转失败', this.getUIContext());
      }
      return;
    }

    try {
      await ApiClient.joinGroup({ group_id: group.group_id });
      SystemToast.show('加入群组成功', this.getUIContext());
      await this.loadNearbyGroups();
    } catch (error) {
      const errMsg = error instanceof Error ? error.message : '加入群组失败';
      if (errMsg.includes('1305')) {
        SystemToast.show('您已是该群成员', this.getUIContext());
      } else {
        SystemToast.showError(errMsg, this.getUIContext());
      }
    }
  }

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
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Bold)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .layoutWeight(1)

        Button('创建')
          .type(ButtonType.Capsule)
          .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
          .fontColor(Color.White)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .height(32)
          .onClick(() => this.createGroup())
      }
      .width('100%')
      .height(56)
      .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      // 内容区域
      if (this.isLoading && this.groups.length === 0) {
        SystemLoadingView({ message: '正在加载附近群组...' })
      } else if (this.errorMessage && this.groups.length === 0) {
        this.buildEmptyState(AppIcon.WARNING, this.errorMessage, '重试', () => { this.loadNearbyGroups(); })
      } else if (this.groups.length === 0) {
        Refresh({ refreshing: $$this.isLoading, builder: this.RefreshBuilder() }) {
          this.buildEmptyState(AppIcon.GROUP, '附近暂无群组\n快来创建第一个吧！', undefined, undefined)
        }
        .onRefreshing(() => { this.loadNearbyGroups(); })
      } else {
        Refresh({ refreshing: $$this.isLoading, builder: this.RefreshBuilder() }) {
          this.buildGroupList()
        }
        .onRefreshing(() => { this.loadNearbyGroups(); })
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
  }

  @Builder
  private buildEmptyState(icon: AppIcon, message: string, actionLabel: string | undefined, action: (() => void) | undefined) {
    Column({ space: DesignConstants.SPACING.LG }) {
      Text() {
        SymbolSpan($r(icon))
          .fontSize(48)
          .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
      }

      Text(message)
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        .textAlign(TextAlign.Center)
        .lineHeight(24)

      if (actionLabel && action) {
        Button(actionLabel)
          .type(ButtonType.Capsule)
          .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
          .fontColor(Color.White)
          .height(40)
          .onClick(() => { action(); })
      }
    }
    .width('100%')
    .height('100%')
    .justifyContent(FlexAlign.Center)
    .padding(DesignConstants.SPACING.XXL)
  }

  @Builder
  RefreshBuilder() {
    Row() {
      LoadingProgress().width(24).height(24)
      Text('正在刷新...')
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .margin({ left: DesignConstants.SPACING.SM })
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
    }
    .width('100%')
    .justifyContent(FlexAlign.Center)
    .padding(DesignConstants.SPACING.SM)
  }

  @Builder
  private buildGroupList() {
    List({ space: DesignConstants.SPACING.MD }) {
      ForEach(this.groups, (group: GroupSearchItem) => {
        ListItem() {
          this.buildGroupCard(group)
        }
      }, (group: GroupSearchItem) => group.group_id)
    }
    .width('100%')
    .layoutWeight(1)
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.LG })
    .scrollBar(BarState.Auto)
  }

  @Builder
  private buildGroupCard(group: GroupSearchItem) {
    Column({ space: DesignConstants.SPACING.MD }) {
      // 群组基本信息
      Row({ space: DesignConstants.SPACING.MD }) {
        SystemAvatar({
          avatarUrl: group.avatar_url || '',
          name: group.name,
          avatarSize: 48,
          shape: 'squircle'
        })

        Column({ space: 4 }) {
          Text(group.name)
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontWeight(FontWeight.Medium)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })

          Row({ space: DesignConstants.SPACING.XS }) {
            Text() {
              SymbolSpan($r(AppIcon.PERSON))
                .fontSize(12)
                .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
            }
            Text(`${group.members_count}人`)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          }

          if (group.description) {
            Text(group.description)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
              .maxLines(2)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
          }
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)

        Button('加入')
          .type(ButtonType.Capsule)
          .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
          .fontColor(Color.White)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .height(32)
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
          altitude: 0
        })
      }
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .borderRadius(DesignConstants.RADIUS.LG)
    .onClick(async () => {
      try {
        const routerInstance = this.getUIContext().getRouter();
        await routerInstance.pushUrl({
          url: `features/group/pages/${PageNames.GROUP_DETAIL}`,
          params: { groupId: group.group_id }
        });
      } catch (error) {
        SystemToast.showError('跳转失败', this.getUIContext());
      }
    })
  }
}
