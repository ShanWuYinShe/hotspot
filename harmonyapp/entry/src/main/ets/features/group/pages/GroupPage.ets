/**
 * 群组页面
 * 使用 HarmonyOS 原生能力 + 系统颜色资源（支持深色模式）
 */

import { GlobalStateManager, KEY_CURRENT_LOCATION, KEY_BROWSE_ANCHOR } from '../../../core/state/GlobalStateManager';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { PageNames, DefaultConfig } from '../../../constants/AppConstants';
import { GroupItem, SearchGroupsRequest } from '../../../core/types/api';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { IntelligentLocationCard } from '../../../components/IntelligentLocationCard';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { AppRouter, LoginRouteParam, GroupDetailRouteParam } from '../../../core/router/AppRouter';

@Component
export struct GroupPage {
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;

  private stateManager = GlobalStateManager.getInstance();
  @State groups: GroupItem[] = [];
  private groupsDataSource: BasicDataSource<GroupItem> = new BasicDataSource<GroupItem>([]);
  @State isLoading: boolean = false;
  @State errorMessage: string = '';
  @State hasMore: boolean = false;
  @State nextCursor: string = '';
  @State isLoadingMore: boolean = false;

  async aboutToAppear() {
    await this.loadNearbyGroups();
  }

  aboutToDisappear() {}

  /** 获取搜索参考点：优先浏览锚点，回退到当前定位 */
  private getRefLocation(): LocationInfo | null {
    return this.browseAnchor ?? this.currentLocation;
  }

  private async loadNearbyGroups() {
    const refLoc = this.getRefLocation();
    if (!refLoc) {
      return;
    }

    this.isLoading = true;
    this.errorMessage = '';

    try {
      const request: SearchGroupsRequest = {
        reference_point: {
          latitude: refLoc.latitude,
          longitude: refLoc.longitude
        },
        radius: DefaultConfig.NEARBY_RADIUS,
        sort_by: 'distance',
        pagination: { limit: DefaultConfig.PAGE_SIZE_LARGE }
      };
      const response = await LoadingUtil.withMinimumTime(() => ApiClient.searchGroups(request));

      if (response.code === 0 && response.data) {
        this.groups = response.data.items;
        this.groupsDataSource.setData(this.groups);
        this.hasMore = !!(response.data.next_cursor);
        this.nextCursor = response.data.next_cursor ?? '';
      } else {
        this.errorMessage = response.message || '加载群组失败';
        this.groups = [];
        this.groupsDataSource.clear();
      }
    } catch (error) {
      this.errorMessage = error instanceof Error ? error.message : '加载群组失败';
    } finally {
      this.isLoading = false;
    }
  }

  private async loadMoreGroups(): Promise<void> {
    const refLoc = this.getRefLocation();
    if (this.isLoadingMore || !this.hasMore || !refLoc) return;
    this.isLoadingMore = true;
    try {
      const request: SearchGroupsRequest = {
        reference_point: {
          latitude: refLoc.latitude,
          longitude: refLoc.longitude
        },
        radius: DefaultConfig.NEARBY_RADIUS,
        sort_by: 'distance',
        pagination: { limit: DefaultConfig.PAGE_SIZE_LARGE, cursor: this.nextCursor }
      };
      const response = await ApiClient.searchGroups(request);
      if (response.code === 0 && response.data) {
        this.groups = this.groups.concat(response.data.items);
        this.groupsDataSource.appendData(response.data.items);
        this.hasMore = !!(response.data.next_cursor);
        this.nextCursor = response.data.next_cursor ?? '';
      }
    } catch (error) {
      // 静默失败，用户可再次触底重试
    } finally {
      this.isLoadingMore = false;
    }
  }

  private async joinGroup(group: GroupItem) {
    const currentUser = this.stateManager.getCurrentUser();
    if (!currentUser) {
      SystemToast.showWarning('请先登录后再加入群组', this.getUIContext());
      const loginParams = new LoginRouteParam();
      loginParams.reason = 'action_required';
      loginParams.action = '加入群组';
      AppRouter.push(PageNames.LOGIN, loginParams);
      return;
    }

    if (group.has_password) {
      SystemToast.show('此群组需要密码，请进入详情页加入', this.getUIContext());
      const detailParams = new GroupDetailRouteParam();
      detailParams.groupId = group.group.group_id;
      AppRouter.push(PageNames.GROUP_DETAIL, detailParams);
      return;
    }

    try {
      const response = await ApiClient.joinGroup({ group_id: group.group.group_id });
      if (response.code === 0 && response.data) {
        if (response.data.status === 'joined') {
          SystemToast.show('加入群组成功', this.getUIContext());
          await this.loadNearbyGroups();
        } else if (response.data.status === 'pending_approval') {
          SystemToast.show('已提交申请，等待管理员审批', this.getUIContext());
          await this.loadNearbyGroups();
        }
      } else if (response.code === 1305) {
        SystemToast.show('您已是该群成员', this.getUIContext());
        await this.loadNearbyGroups();
      } else {
        SystemToast.showError(response.message || '加入群组失败', this.getUIContext());
      }
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
      const loginParams = new LoginRouteParam();
      loginParams.reason = 'action_required';
      loginParams.action = '创建群组';
      AppRouter.push(PageNames.LOGIN, loginParams);
      return;
    }

    AppRouter.push(PageNames.CREATE_GROUP);
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
          .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else if (this.errorMessage && this.groups.length === 0) {
        Column() {
          this.buildEmptyState(AppIcon.WARNING, this.errorMessage, '重试', () => { this.loadNearbyGroups(); })
        }
        .layoutWeight(1)
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else if (this.groups.length === 0) {
        Refresh({ refreshing: $$this.isLoading, builder: this.RefreshBuilder() }) {
          this.buildEmptyState(AppIcon.GROUP, '附近暂无群组\n快来创建第一个吧！', undefined, undefined)
        }
        .onRefreshing(() => { this.loadNearbyGroups(); })
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else {
        Refresh({ refreshing: $$this.isLoading, builder: this.RefreshBuilder() }) {
          this.buildGroupList()
        }
        .onRefreshing(() => { this.loadNearbyGroups(); })
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
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
      LazyForEach(this.groupsDataSource, (group: GroupItem) => {
        ListItem() {
          this.buildGroupCard(group)
        }
      }, (group: GroupItem) => group.group?.group_id ?? '')
      if (this.isLoadingMore) {
        ListItem() {
          Row({ space: DesignConstants.SPACING.SM }) {
            LoadingProgress().width(20).height(20)
            Text('加载更多...')
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          }
          .width('100%')
          .justifyContent(FlexAlign.Center)
          .padding(DesignConstants.SPACING.MD)
        }
      }
    }
    .width('100%')
    .layoutWeight(1)
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.LG })
    .scrollBar(BarState.Auto)
    .onReachEnd(() => { this.loadMoreGroups(); })
  }

  @Builder
  private buildGroupCard(group: GroupItem) {
    Column({ space: DesignConstants.SPACING.MD }) {
      // 群组基本信息
      Row({ space: DesignConstants.SPACING.MD }) {
        SystemAvatar({
          avatarUrl: group.group.avatar_url || '',
          name: group.group.name,
          avatarSize: 48,
          shape: 'squircle'
        })

        Column({ space: 4 }) {
          Text(group.group.name)
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
            Text(`${group.group.members_count}人`)
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
      if (group.location?.point) {
        IntelligentLocationCard({
          latitude: group.location.point.latitude,
          longitude: group.location.point.longitude,
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
    .onClick(() => {
      const detailParams = new GroupDetailRouteParam();
      detailParams.groupId = group.group.group_id;
      AppRouter.push(PageNames.GROUP_DETAIL, detailParams);
    })
  }
}
