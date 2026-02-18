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

import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { EmptyState, LoadingMoreIndicator, RefreshIndicator } from '../../../components/EmptyState';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { AppRouter, LoginRouteParam, GroupDetailRouteParam } from '../../../core/router/AppRouter';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';

@Component
export struct GroupPage {
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;

  private stateManager = GlobalStateManager.getInstance();
  @State groups: GroupItem[] = [];
  private groupsDataSource: BasicDataSource<GroupItem> = new BasicDataSource<GroupItem>([]);
  @State isLoading: boolean = true;
  @State isRefreshing: boolean = false;
  @State errorMessage: string = '';
  @State hasMore: boolean = false;
  @State nextCursor: string = '';
  @State isLoadingMore: boolean = false;

  private getHeaderTopPadding(): number {
    return (this.safeAreaTop > 0 ? this.safeAreaTop : DesignConstants.SPACING.LG) + DesignConstants.SPACING.SM;
  }

  async aboutToAppear() {
    SystemUiController.useSurface({
      baseColor: DesignConstants.COLOR.PRIMARY_BG_LIGHT,      preferDarkContent: true
    });
    await this.loadNearbyGroups();
  }

  aboutToDisappear() {}

  /** 获取搜索参考点：优先浏览锚点，回退到当前定位 */
  private getRefLocation(): LocationInfo | null {
    return this.browseAnchor ?? this.currentLocation;
  }

  private async loadNearbyGroups(refresh: boolean = false) {
    const refLoc = this.getRefLocation();
    if (!refLoc) {
      this.isLoading = false;
      this.isRefreshing = false;
      return;
    }

    if (!refresh) { this.isLoading = true; }
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

      if (response.code === 0) {
        this.groups = response.data?.items ?? [];
        this.groupsDataSource.setData(this.groups);
        this.hasMore = !!(response.data?.next_cursor);
        this.nextCursor = response.data?.next_cursor ?? '';
      } else {
        this.errorMessage = response.message || '加载群组失败';
        this.groups = [];
        this.groupsDataSource.clear();
      }
    } catch (error) {
      this.errorMessage = error instanceof Error ? error.message : '加载群组失败';
    } finally {
      this.isLoading = false;
      this.isRefreshing = false;
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
      if (response.code === 0) {
        const items = response.data?.items ?? [];
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

  private formatDistance(meters: number): string {
    if (meters < 1000) {
      return `${Math.round(meters)}m`;
    }
    return `${(meters / 1000).toFixed(1)}km`;
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
      // 标题栏 - 使用 SafeArea 避免与状态栏重叠
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
      .alignItems(VerticalAlign.Center)
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: this.getHeaderTopPadding(),
        bottom: DesignConstants.SPACING.SM
      })
      .backgroundColor($r('app.color.bg_secondary'))
      .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])

      // 内容区域
      if (this.isLoading) {
        SystemLoadingView({ message: '正在加载附近群组...' })
          .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else if (this.errorMessage && this.groups.length === 0) {
        EmptyState({ icon: AppIcon.WARNING, message: this.errorMessage, actionLabel: '重试', onAction: () => { this.loadNearbyGroups(); } })
          .layoutWeight(1)
          .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else if (this.groups.length === 0) {
        Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
          EmptyState({ icon: AppIcon.GROUP, message: '附近暂无群组\n快来创建第一个吧！' })
        }
        .onRefreshing(() => { this.loadNearbyGroups(true); })
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else {
        Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
          this.buildGroupList()
        }
        .layoutWeight(1)
        .onRefreshing(() => { this.loadNearbyGroups(true); })
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('app.color.bg_primary'))
  }

  @Builder
  private buildGroupList() {
    List() {
      LazyForEach(this.groupsDataSource, (group: GroupItem) => {
        ListItem() {
          this.buildGroupCard(group)
        }
      }, (group: GroupItem) => group.group?.group_id ?? '')
      if (this.isLoadingMore) {
        ListItem() {
          LoadingMoreIndicator()
        }
      }
    }
    .width('100%')
    .padding({ bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, 0) })
    .divider({ strokeWidth: 0.5, color: $r('app.color.divider'), startMargin: 76, endMargin: 0 })
    .scrollBar(BarState.Auto)
    .onReachEnd(() => { this.loadMoreGroups(); })
  }

  @Builder
  private buildGroupCard(group: GroupItem) {
    Row({ space: 12 }) {
      SystemAvatar({
        avatarUrl: group.group.avatar_url || '',
        name: group.group.name,
        avatarSize: 48,
        shape: 'squircle'
      })

      Column({ space: 2 }) {
        Text(group.group.name)
          .fontSize(16)
          .fontWeight(FontWeight.Medium)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        Row({ space: 8 }) {
          Text(`${group.group.members_count}人`)
            .fontSize(13)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))

          if (group.location?.distance !== undefined && group.location?.distance !== null) {
            Text(this.formatDistance(group.location.distance))
              .fontSize(13)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          }
        }

        if (group.description) {
          Text(group.description)
            .fontSize(13)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)

      Button('加入')
        .type(ButtonType.Capsule)
        .backgroundColor(Color.Transparent)
        .fontColor($r('sys.color.ohos_id_color_emphasize'))
        .fontSize(13)
        .fontWeight(FontWeight.Medium)
        .height(32)
        .borderWidth(1)
        .borderColor($r('sys.color.ohos_id_color_emphasize'))
        .onClick(() => this.joinGroup(group))
    }
    .width('100%')
    .padding({ left: 16, right: 16, top: 12, bottom: 12 })
    .alignItems(VerticalAlign.Center)
    .onClick(() => {
      const detailParams = new GroupDetailRouteParam();
      detailParams.groupId = group.group.group_id;
      AppRouter.push(PageNames.GROUP_DETAIL, detailParams);
    })
  }
}
