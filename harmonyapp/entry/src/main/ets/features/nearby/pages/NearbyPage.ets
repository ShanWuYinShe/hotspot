/**
 * 附近页面 - 主标签页
 * 包含动态、用户、群组三个子标签，顶部位置栏和搜索/地图入口
 * 类似抖音风格的子标签切换
 */

import { GlobalStateManager, KEY_IS_LOGGED_IN, KEY_BROWSE_ANCHOR, KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { AppRouter } from '../../../core/router/AppRouter';
import { PageNames, DefaultConfig } from '../../../constants/AppConstants';
import {
  MomentItem, UserItem, GroupItem, GeoPoint,
  SearchMomentsRequest, SearchUsersRequest, SearchGroupsRequest,
  MomentImage, FollowRequest, JoinGroupRequest
} from '../../../core/types/api';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { LocationBar } from '../../../components/LocationBar';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { DistributedManager } from '../../../core/distributed/DistributedManager';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { WidgetDataService } from '../../../core/services/WidgetDataService';

const DOMAIN_NUMBER: number = 0xFF10;
const TAG: string = 'NearbyPage';

class GroupDetailParams {
  groupId: string = '';
}

class LoginRedirectParams {
  reason: string = '';
  action: string = '';
}

@Component
export struct NearbyPage {
  @StorageLink(KEY_IS_LOGGED_IN) isLoggedIn: boolean = false;
  @StorageLink(KEY_CURRENT_LOCATION) @Watch('onCurrentLocationChange') currentLocation: LocationInfo | null = null;
  @StorageLink(KEY_BROWSE_ANCHOR) @Watch('onBrowseAnchorChange') browseAnchor: LocationInfo | null = null;
  @StorageLink('nearbyDeviceCount') nearbyDeviceCount: number = 0;

  @State currentSubTab: number = 0;
  private subTabController: TabsController = new TabsController();
  /** 标记 aboutToAppear 是否已完成首次数据加载，避免 @Watch 重复触发 */
  private initialLoadDone: boolean = false;

  // 动态数据
  @State moments: MomentItem[] = [];
  private momentsDataSource: BasicDataSource<MomentItem> = new BasicDataSource<MomentItem>([]);
  @State momentsLoading: boolean = false;
  @State momentsError: string = '';
  @State momentsHasMore: boolean = false;
  @State momentsNextCursor: string = '';
  @State momentsLoadingMore: boolean = false;

  // 用户数据
  @State users: UserItem[] = [];
  private usersDataSource: BasicDataSource<UserItem> = new BasicDataSource<UserItem>([]);
  @State usersLoading: boolean = false;
  @State usersError: string = '';
  @State usersHasMore: boolean = false;
  @State usersNextCursor: string = '';
  @State usersLoadingMore: boolean = false;

  // 群组数据
  @State groups: GroupItem[] = [];
  private groupsDataSource: BasicDataSource<GroupItem> = new BasicDataSource<GroupItem>([]);
  @State groupsLoading: boolean = false;
  @State groupsError: string = '';
  @State groupsHasMore: boolean = false;
  @State groupsNextCursor: string = '';
  @State groupsLoadingMore: boolean = false;

  async aboutToAppear() {
    // 预加载所有 tab 数据（并行），切换 tab 时数据已就绪，无需再显示 loading
    this.loadNearbyMoments();
    this.loadNearbyUsers();
    this.loadNearbyGroups();
    this.initialLoadDone = true;

    // Start device discovery (non-blocking)
    try {
      const dm = DistributedManager.getInstance();
      dm.startDeviceDiscovery();
    } catch {
      hilog.warn(DOMAIN_NUMBER, TAG, '设备发现启动失败');
    }
  }

  aboutToDisappear() {
    try {
      DistributedManager.getInstance().stopDeviceDiscovery();
    } catch {
      // ignore
    }
  }

  /**
   * currentLocation 变化时：仅当没有设置浏览锚点且当前数据为空/出错时自动刷新
   * 典型场景：页面加载时 GPS 尚未定位，定位完成后自动加载附近数据
   */
  private onCurrentLocationChange(): void {
    if (!this.initialLoadDone) return;
    // 已设置浏览锚点时，currentLocation 变化不影响搜索中心
    if (this.browseAnchor) return;
    // 新位置为 null 则无需处理
    if (!this.currentLocation) return;
    // 仅当当前 tab 无数据或有错误时才自动刷新（避免频繁 GPS 更新导致反复请求）
    if (this.isCurrentTabEmpty()) {
      hilog.info(DOMAIN_NUMBER, TAG, '位置可用，自动加载附近数据');
      this.loadCurrentTabData();
    }
  }

  /**
   * browseAnchor 变化时：用户主动切换了搜索中心，始终刷新当前 tab 数据
   */
  private onBrowseAnchorChange(): void {
    if (!this.initialLoadDone) return;
    hilog.info(DOMAIN_NUMBER, TAG, `浏览锚点变化 -> ${this.browseAnchor ? this.browseAnchor.locationName : '已清除'}`);
    this.loadCurrentTabData();
  }

  /** 判断当前 tab 是否无数据（用于决定 location 变化时是否需要自动加载） */
  private isCurrentTabEmpty(): boolean {
    switch (this.currentSubTab) {
      case 0: return this.moments.length === 0;
      case 1: return this.users.length === 0;
      case 2: return this.groups.length === 0;
      default: return true;
    }
  }

  private getReferencePoint(): GeoPoint | null {
    const loc: LocationInfo | null = this.browseAnchor ?? this.currentLocation;
    if (!loc) return null;
    const point: GeoPoint = { latitude: loc.latitude, longitude: loc.longitude };
    return point;
  }

  private async loadCurrentTabData(): Promise<void> {
    switch (this.currentSubTab) {
      case 0:
        await this.loadNearbyMoments();
        break;
      case 1:
        await this.loadNearbyUsers();
        break;
      case 2:
        await this.loadNearbyGroups();
        break;
    }
  }

  // ==================== 加载附近动态 ====================
  private async loadNearbyMoments(loadMore: boolean = false): Promise<void> {
    const ref = this.getReferencePoint();
    if (!ref) {
      return;
    }
    if (loadMore) {
      if (this.momentsLoadingMore || !this.momentsHasMore) return;
      this.momentsLoadingMore = true;
    } else {
      this.momentsLoading = true;
      this.momentsError = '';
    }
    try {
      const request: SearchMomentsRequest = {
        reference_point: ref,
        radius: DefaultConfig.NEARBY_RADIUS,
        sort_by: 'distance',
        pagination: { limit: DefaultConfig.PAGE_SIZE, cursor: loadMore ? this.momentsNextCursor : undefined }
      };
      const response = loadMore
        ? await ApiClient.searchMoments(request)
        : await LoadingUtil.withMinimumTime(() => ApiClient.searchMoments(request));
      if (response.code === 0 && response.data) {
        const validItems = response.data.items.filter((m: MomentItem) => m.moment_id && m.user);
        if (loadMore) {
          this.moments = this.moments.concat(validItems);
          this.momentsDataSource.appendData(validItems);
        } else {
          this.moments = validItems;
          this.momentsDataSource.setData(validItems);
        }
        this.momentsHasMore = response.data.pagination?.has_more ?? false;
        this.momentsNextCursor = response.data.pagination?.next_cursor ?? '';
        try {
          WidgetDataService.getInstance().updateStats('nearbyMomentCount', this.moments.length);
        } catch {
          // 小组件数据更新失败不影响主流程
        }
      } else if (!loadMore) {
        this.momentsError = response.message || '加载失败';
      }
    } catch {
      hilog.error(DOMAIN_NUMBER, TAG, '加载附近动态失败');
      if (!loadMore) {
        this.momentsError = '加载附近动态失败';
      }
    } finally {
      if (loadMore) {
        this.momentsLoadingMore = false;
      } else {
        this.momentsLoading = false;
      }
    }
  }

  // ==================== 加载附近用户 ====================
  private async loadNearbyUsers(loadMore: boolean = false): Promise<void> {
    const ref = this.getReferencePoint();
    if (!ref) {
      return;
    }
    if (loadMore) {
      if (this.usersLoadingMore || !this.usersHasMore) return;
      this.usersLoadingMore = true;
    } else {
      this.usersLoading = true;
      this.usersError = '';
    }
    try {
      const request: SearchUsersRequest = {
        reference_point: ref,
        radius: DefaultConfig.NEARBY_RADIUS,
        sort_by: 'distance',
        pagination: { limit: DefaultConfig.PAGE_SIZE, cursor: loadMore ? this.usersNextCursor : undefined }
      };
      const response = loadMore
        ? await ApiClient.searchUsers(request)
        : await LoadingUtil.withMinimumTime(() => ApiClient.searchUsers(request));
      if (response.code === 0 && response.data) {
        const validItems = response.data.items.filter((u: UserItem) => u.user?.user_id);
        if (loadMore) {
          this.users = this.users.concat(validItems);
          this.usersDataSource.appendData(validItems);
        } else {
          this.users = validItems;
          this.usersDataSource.setData(validItems);
        }
        this.usersHasMore = response.data.pagination?.has_more ?? false;
        this.usersNextCursor = response.data.pagination?.next_cursor ?? '';
        try {
          WidgetDataService.getInstance().updateStats('nearbyUserCount', this.users.length);
        } catch {
          // 小组件数据更新失败不影响主流程
        }
      } else if (!loadMore) {
        this.usersError = response.message || '加载失败';
      }
    } catch {
      hilog.error(DOMAIN_NUMBER, TAG, '加载附近用户失败');
      if (!loadMore) {
        this.usersError = '加载附近用户失败';
      }
    } finally {
      if (loadMore) {
        this.usersLoadingMore = false;
      } else {
        this.usersLoading = false;
      }
    }
  }

  // ==================== 加载附近群组 ====================
  private async loadNearbyGroups(loadMore: boolean = false): Promise<void> {
    const ref = this.getReferencePoint();
    if (!ref) {
      return;
    }
    if (loadMore) {
      if (this.groupsLoadingMore || !this.groupsHasMore) return;
      this.groupsLoadingMore = true;
    } else {
      this.groupsLoading = true;
      this.groupsError = '';
    }
    try {
      const request: SearchGroupsRequest = {
        reference_point: ref,
        radius: DefaultConfig.NEARBY_RADIUS,
        sort_by: 'distance',
        pagination: { limit: DefaultConfig.PAGE_SIZE, cursor: loadMore ? this.groupsNextCursor : undefined }
      };
      const response = loadMore
        ? await ApiClient.searchGroups(request)
        : await LoadingUtil.withMinimumTime(() => ApiClient.searchGroups(request));
      if (response.code === 0 && response.data) {
        const validItems = response.data.items.filter((g: GroupItem) => g.group?.group_id);
        if (loadMore) {
          this.groups = this.groups.concat(validItems);
          this.groupsDataSource.appendData(validItems);
        } else {
          this.groups = validItems;
          this.groupsDataSource.setData(validItems);
        }
        this.groupsHasMore = response.data.pagination?.has_more ?? false;
        this.groupsNextCursor = response.data.pagination?.next_cursor ?? '';
        try {
          WidgetDataService.getInstance().updateStats('nearbyGroupCount', this.groups.length);
        } catch {
          // 小组件数据更新失败不影响主流程
        }
      } else if (!loadMore) {
        this.groupsError = response.message || '加载失败';
      }
    } catch {
      hilog.error(DOMAIN_NUMBER, TAG, '加载附近群组失败');
      if (!loadMore) {
        this.groupsError = '加载附近群组失败';
      }
    } finally {
      if (loadMore) {
        this.groupsLoadingMore = false;
      } else {
        this.groupsLoading = false;
      }
    }
  }

  build() {
    Column() {
      // 顶部位置栏
      LocationBar({ showSearchButton: true })

      // 附近设备发现指示器
      if (this.nearbyDeviceCount > 0) {
        Row({ space: DesignConstants.SPACING.SM }) {
          Text() {
            SymbolSpan($r('sys.symbol.wifi'))
              .fontSize(14)
              .fontColor([$r('sys.color.ohos_id_color_emphasize')])
          }
          Text(`发现 ${this.nearbyDeviceCount} 台附近设备正在使用 GeoTalk`)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
        .width('100%')
        .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
        .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      }

      // 子标签栏 + 内容
      Tabs({ barPosition: BarPosition.Start, controller: this.subTabController }) {
        // 子标签 0: 动态
        TabContent() {
          this.buildMomentsContent()
        }
        .tabBar(this.SubTabBuilder('动态', 0))

        // 子标签 1: 用户
        TabContent() {
          this.buildUsersContent()
        }
        .tabBar(this.SubTabBuilder('用户', 1))

        // 子标签 2: 群组
        TabContent() {
          this.buildGroupsContent()
        }
        .tabBar(this.SubTabBuilder('群组', 2))
      }
      .barMode(BarMode.Fixed)
      .barHeight(44)
      .animationDuration(500)
      .onChange((index: number) => {
        this.getUIContext().animateTo({ duration: 400, curve: Curve.FastOutSlowIn }, () => {
          this.currentSubTab = index;
        });
        this.loadCurrentTabData();
      })
      .layoutWeight(1)
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
  }

  // ==================== 子标签构建器 ====================
  @Builder
  SubTabBuilder(title: string, index: number) {
    Column() {
      Text(title)
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .fontWeight(this.currentSubTab === index ? FontWeight.Bold : FontWeight.Normal)
        .fontColor(this.currentSubTab === index
          ? $r('sys.color.ohos_id_color_text_primary')
          : $r('sys.color.ohos_id_color_text_secondary'))

      // 下划线指示器（带缩放+颜色过渡动画）
      Divider()
        .width(this.currentSubTab === index ? 24 : 0)
        .strokeWidth(3)
        .color(this.currentSubTab === index
          ? $r('sys.color.ohos_id_color_emphasize')
          : Color.Transparent)
        .margin({ top: 6 })
        .lineCap(LineCapStyle.Round)
        .animation({ duration: 400, curve: Curve.FastOutSlowIn })
    }
    .width('100%')
    .height('100%')
    .justifyContent(FlexAlign.Center)
  }

  // ==================== 动态内容 ====================
  @Builder
  buildMomentsContent() {
    if (this.momentsLoading && this.moments.length === 0) {
      SystemLoadingView({ message: '正在加载附近动态...' })
    } else if (this.momentsError.length > 0 && this.moments.length === 0) {
      this.buildEmptyState(AppIcon.WARNING, this.momentsError, '重试', () => { this.loadNearbyMoments(); })
    } else if (this.moments.length === 0) {
      this.buildEmptyState(AppIcon.CAMERA, '附近暂无动态\n去发布第一条吧！', undefined, undefined)
    } else {
      Refresh({ refreshing: $$this.momentsLoading }) {
        List({ space: DesignConstants.SPACING.SM }) {
          LazyForEach(this.momentsDataSource, (moment: MomentItem) => {
            ListItem() {
              this.buildMomentCard(moment)
            }
          }, (moment: MomentItem) => moment.moment_id)
          // 底部加载更多指示器
          if (this.momentsLoadingMore) {
            ListItem() {
              this.buildLoadingMoreIndicator()
            }
          }
        }
        .width('100%')
        .layoutWeight(1)
        .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
          top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.LG })
        .scrollBar(BarState.Auto)
        .onReachEnd(() => { this.loadNearbyMoments(true); })
      }
      .onRefreshing(() => { this.loadNearbyMoments(); })
      .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
    }
  }

  @Builder
  buildMomentCard(moment: MomentItem) {
    Column({ space: DesignConstants.SPACING.MD }) {
      // 用户信息
      Row({ space: DesignConstants.SPACING.SM }) {
        SystemAvatar({
          avatarUrl: moment.user?.avatar_url ?? '',
          name: moment.user?.nickname ?? '',
          avatarSize: 40
        })
        Column({ space: 2 }) {
          Text(moment.user?.nickname ?? '')
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontWeight(FontWeight.Medium)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
          if (moment.location?.location_name) {
            Text(moment.location.location_name)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          }
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)
      }
      .width('100%')

      // 文本内容
      Text(moment.description)
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
        .lineHeight(22)
        .width('100%')

      // 图片网格
      if (moment.images && moment.images.length > 0) {
        this.buildImageGrid(moment.images)
      }

      // 互动栏
      Row() {
        this.buildStatItem(AppIcon.HEART, `${moment.likes_count}`)
        this.buildStatItem(AppIcon.MESSAGE, `${moment.comments_count}`)
      }
      .width('100%')
      .justifyContent(FlexAlign.Start)
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .borderRadius(DesignConstants.RADIUS.LG)
  }

  @Builder
  buildImageGrid(images: MomentImage[]) {
    if (images.length === 1) {
      Image(images[0].url)
        .width('100%')
        .constraintSize({ maxHeight: 240 })
        .objectFit(ImageFit.Cover)
        .borderRadius(DesignConstants.RADIUS.MD)
    } else {
      Flex({ wrap: FlexWrap.Wrap }) {
        ForEach(images.slice(0, 9), (img: MomentImage) => {
          Image(img.url)
            .width(images.length <= 4 ? '48%' : '31%')
            .aspectRatio(1)
            .objectFit(ImageFit.Cover)
            .borderRadius(DesignConstants.RADIUS.SM)
            .margin(2)
        }, (img: MomentImage, index: number) => `${img.url}_${index}`)
      }
      .width('100%')
    }
  }

  @Builder
  buildStatItem(icon: AppIcon, text: string) {
    Row({ space: 4 }) {
      Text() {
        SymbolSpan($r(icon))
          .fontSize(16)
          .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
      }
      Text(text)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
    }
    .margin({ right: 24 })
  }

  // ==================== 用户内容 ====================
  @Builder
  buildUsersContent() {
    if (this.usersLoading && this.users.length === 0) {
      SystemLoadingView({ message: '正在加载附近用户...' })
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
    } else if (this.usersError.length > 0 && this.users.length === 0) {
      Stack() {
        this.buildEmptyState(AppIcon.WARNING, this.usersError, '重试', () => { this.loadNearbyUsers(); })
      }
      .width('100%')
      .height('100%')
      .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
    } else if (this.users.length === 0) {
      Stack() {
        this.buildEmptyState(AppIcon.PERSON, '附近暂无其他用户', undefined, undefined)
      }
      .width('100%')
      .height('100%')
      .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
    } else {
      Refresh({ refreshing: $$this.usersLoading }) {
        List({ space: DesignConstants.SPACING.SM }) {
          LazyForEach(this.usersDataSource, (user: UserItem) => {
            ListItem() {
              this.buildUserCard(user)
            }
          }, (user: UserItem) => user.user?.user_id ?? '')
          if (this.usersLoadingMore) {
            ListItem() {
              this.buildLoadingMoreIndicator()
            }
          }
        }
        .width('100%')
        .layoutWeight(1)
        .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
          top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.LG })
        .scrollBar(BarState.Auto)
        .onReachEnd(() => { this.loadNearbyUsers(true); })
      }
      .onRefreshing(() => { this.loadNearbyUsers(); })
      .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
    }
  }

  @Builder
  buildUserCard(user: UserItem) {
    Row({ space: DesignConstants.SPACING.MD }) {
      SystemAvatar({
        avatarUrl: user.user?.avatar_url ?? '',
        name: user.user?.nickname ?? '',
        avatarSize: 52
      })

      Column({ space: 4 }) {
        Text(user.user?.nickname ?? '')
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .fontWeight(FontWeight.Medium)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))

        if (user.bio) {
          Text(user.bio)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .maxLines(2)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }

        if (user.location?.location_name) {
          Row({ space: 4 }) {
            Text() {
              SymbolSpan($r(AppIcon.LOCATION))
                .fontSize(12)
                .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
            }
            Text(user.location.location_name)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))

            if (user.location.distance !== undefined && user.location.distance !== null) {
              Text(this.formatDistance(user.location.distance))
                .fontSize(DesignConstants.FONT.SIZE.XS)
                .fontColor($r('sys.color.ohos_id_color_emphasize'))
            }
          }
        }
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)

      // 关注按钮
      Button('关注')
        .type(ButtonType.Capsule)
        .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
        .fontColor(Color.White)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .height(32)
        .onClick(() => { this.followUser(user); })
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .borderRadius(DesignConstants.RADIUS.LG)
  }

  // ==================== 群组内容 ====================
  @Builder
  buildGroupsContent() {
    if (this.groupsLoading && this.groups.length === 0) {
      SystemLoadingView({ message: '正在加载附近群组...' })
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
    } else if (this.groupsError.length > 0 && this.groups.length === 0) {
      Stack() {
        this.buildEmptyState(AppIcon.WARNING, this.groupsError, '重试', () => { this.loadNearbyGroups(); })
      }
      .width('100%')
      .height('100%')
      .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
    } else if (this.groups.length === 0) {
      Stack() {
        this.buildEmptyState(AppIcon.GROUP, '附近暂无群组', undefined, undefined)
      }
      .width('100%')
      .height('100%')
      .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
    } else {
      Refresh({ refreshing: $$this.groupsLoading }) {
        List({ space: DesignConstants.SPACING.SM }) {
          LazyForEach(this.groupsDataSource, (group: GroupItem) => {
            ListItem() {
              this.buildGroupCard(group)
            }
          }, (group: GroupItem) => group.group?.group_id ?? '')
          if (this.groupsLoadingMore) {
            ListItem() {
              this.buildLoadingMoreIndicator()
            }
          }
        }
        .width('100%')
        .layoutWeight(1)
        .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
          top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.LG })
        .scrollBar(BarState.Auto)
        .onReachEnd(() => { this.loadNearbyGroups(true); })
      }
      .onRefreshing(() => { this.loadNearbyGroups(); })
      .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
    }
  }

  @Builder
  buildGroupCard(group: GroupItem) {
    Row({ space: DesignConstants.SPACING.MD }) {
      SystemAvatar({
        avatarUrl: group.group?.avatar_url ?? '',
        name: group.group?.name ?? '',
        avatarSize: 52,
        shape: 'squircle'
      })

      Column({ space: 4 }) {
        Text(group.group?.name ?? '')
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .fontWeight(FontWeight.Medium)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        Row({ space: DesignConstants.SPACING.SM }) {
          Text(`${group.group?.members_count ?? 0}人`)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))

          if (group.location?.location_name) {
            Text(group.location.location_name)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
              .constraintSize({ maxWidth: '50%' })
          }
        }

        if (group.description) {
          Text(group.description)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .maxLines(1)
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
        .onClick(() => { this.joinGroup(group); })
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .borderRadius(DesignConstants.RADIUS.LG)
    .onClick(() => { this.openGroupDetail(group); })
  }

  // ==================== 底部加载更多指示器 ====================
  @Builder
  buildLoadingMoreIndicator() {
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

  // ==================== 通用空状态 ====================
  @Builder
  buildEmptyState(icon: AppIcon, message: string, actionLabel: string | undefined, action: (() => void) | undefined) {
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

  // ==================== 操作方法 ====================
  private async followUser(user: UserItem): Promise<void> {
    if (!user.user?.user_id) return;
    if (!this.isLoggedIn) {
      SystemToast.showWarning('请先登录后再关注', this.getUIContext());
      const loginParams = new LoginRedirectParams();
      loginParams.reason = 'action_required';
      loginParams.action = '关注用户';
      AppRouter.push(PageNames.LOGIN, loginParams);
      return;
    }
    try {
      const followReq: FollowRequest = { user_id: user.user.user_id };
      const response = await ApiClient.followUser(followReq);
      if (response.code === 0) {
        SystemToast.showSuccess('已关注', this.getUIContext());
      } else {
        SystemToast.showError(response.message || '关注失败', this.getUIContext());
      }
    } catch {
      SystemToast.showError('关注失败', this.getUIContext());
    }
  }

  private async joinGroup(group: GroupItem): Promise<void> {
    if (!this.isLoggedIn) {
      SystemToast.showWarning('请先登录后再加入群组', this.getUIContext());
      const loginParams = new LoginRedirectParams();
      loginParams.reason = 'action_required';
      loginParams.action = '加入群组';
      AppRouter.push(PageNames.LOGIN, loginParams);
      return;
    }
    try {
      const joinReq: JoinGroupRequest = { group_id: group.group?.group_id ?? '' };
      const response = await ApiClient.joinGroup(joinReq);
      if (response.code === 0 && response.data) {
        if (response.data.status === 'joined') {
          SystemToast.showSuccess('加入成功', this.getUIContext());
          await this.loadNearbyGroups();
        } else if (response.data.status === 'pending_approval') {
          SystemToast.show('已提交申请，等待审批', this.getUIContext());
        }
      } else if (response.code === 1305) {
        SystemToast.show('您已是该群成员', this.getUIContext());
      } else {
        SystemToast.showError(response.message || '加入失败', this.getUIContext());
      }
    } catch {
      SystemToast.showError('加入群组失败', this.getUIContext());
    }
  }

  private openGroupDetail(group: GroupItem): void {
    if (!group.group?.group_id) return;
    const params = new GroupDetailParams();
    params.groupId = group.group.group_id;
    AppRouter.push(PageNames.GROUP_DETAIL, params);
  }

  private formatDistance(meters: number): string {
    if (meters < 1000) {
      return `${Math.round(meters)}m`;
    }
    return `${(meters / 1000).toFixed(1)}km`;
  }
}
