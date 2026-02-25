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
  MomentItem, UserItem, GroupItem, GeoPoint, RelativeLocation,
  SearchMomentsRequest, SearchUsersRequest, SearchGroupsRequest,
  MomentImage, FollowRequest, JoinGroupRequest
} from '../../../core/types/api';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { LocationBar } from '../../../components/LocationBar';
import { EmptyState, LoadingMoreIndicator } from '../../../components/EmptyState';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { DistributedManager } from '../../../core/distributed/DistributedManager';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { WidgetDataService } from '../../../core/services/WidgetDataService';
import { SystemUiController } from '../../../core/utils/SystemUiController';


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
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;

  @State currentSubTab: number = 0;
  private subTabController: TabsController = new TabsController();
  /** 标记 aboutToAppear 是否已完成首次数据加载，避免 @Watch 重复触发 */
  private initialLoadDone: boolean = false;

  // 动态数据
  @State moments: MomentItem[] = [];
  private momentsDataSource: BasicDataSource<MomentItem> = new BasicDataSource<MomentItem>([]);
  @State momentsLoading: boolean = true;
  @State momentsRefreshing: boolean = false;
  @State momentsError: string = '';
  @State momentsHasMore: boolean = false;
  @State momentsNextCursor: string = '';
  @State momentsLoadingMore: boolean = false;

  // 用户数据
  @State users: UserItem[] = [];
  private usersDataSource: BasicDataSource<UserItem> = new BasicDataSource<UserItem>([]);
  @State usersLoading: boolean = true;
  @State usersRefreshing: boolean = false;
  @State usersError: string = '';
  @State usersHasMore: boolean = false;
  @State usersNextCursor: string = '';
  @State usersLoadingMore: boolean = false;

  // 群组数据
  @State groups: GroupItem[] = [];
  private groupsDataSource: BasicDataSource<GroupItem> = new BasicDataSource<GroupItem>([]);
  @State groupsLoading: boolean = true;
  @State groupsRefreshing: boolean = false;
  @State groupsError: string = '';
  @State groupsHasMore: boolean = false;
  @State groupsNextCursor: string = '';
  @State groupsLoadingMore: boolean = false;

  async aboutToAppear() {
    SystemUiController.useSurface({
      baseColor: DesignConstants.COLOR.PRIMARY_BG_LIGHT,      preferDarkContent: true
    });
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
    this.loadNearbyMoments();
    this.loadNearbyUsers();
    this.loadNearbyGroups();
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

  private async loadCurrentTabData(refresh: boolean = false): Promise<void> {
    switch (this.currentSubTab) {
      case 0:
        await this.loadNearbyMoments(false, refresh);
        break;
      case 1:
        await this.loadNearbyUsers(false, refresh);
        break;
      case 2:
        await this.loadNearbyGroups(false, refresh);
        break;
    }
  }

  private getHeaderTopPadding(): number {
    const safeTop = this.safeAreaTop > 0 ? this.safeAreaTop : DesignConstants.SPACING.XXL;
    return safeTop + DesignConstants.SPACING.SM;
  }

  private getListBottomPadding(): number {
    // 底部安全区域已由 Tab Bar 处理，列表只需少量视觉边距
    return DesignConstants.SPACING.MD;
  }

  private switchSubTab(index: number): void {
    if (this.currentSubTab === index) {
      return;
    }
    // 在 Tabs 动画开始前预设目标 tab 的 loading 状态，
    // 避免 Tabs 捕获旧内容（错误/空状态）用于过渡动画
    this.prepareTabLoading(index);
    this.subTabController.changeIndex(index);
  }

  /** 为目标 tab 预设 loading 状态（仅当该 tab 无数据或存在错误时） */
  private prepareTabLoading(index: number): void {
    switch (index) {
      case 0:
        if (this.moments.length === 0 || this.momentsError.length > 0) {
          this.momentsLoading = true;
          this.momentsError = '';
        }
        break;
      case 1:
        if (this.users.length === 0 || this.usersError.length > 0) {
          this.usersLoading = true;
          this.usersError = '';
        }
        break;
      case 2:
        if (this.groups.length === 0 || this.groupsError.length > 0) {
          this.groupsLoading = true;
          this.groupsError = '';
        }
        break;
    }
  }

  private openLocationPicker(): void {
    AppRouter.push(PageNames.LOCATION_PICKER);
  }

  // ==================== 加载附近动态 ====================
  private async loadNearbyMoments(loadMore: boolean = false, refresh: boolean = false): Promise<void> {
    const ref = this.getReferencePoint();
    if (!ref) {
      if (!loadMore) { this.momentsLoading = false; this.momentsRefreshing = false; }
      return;
    }
    if (loadMore) {
      if (this.momentsLoadingMore || !this.momentsHasMore) return;
      this.momentsLoadingMore = true;
    } else {
      if (!refresh || this.moments.length === 0) { this.momentsLoading = true; }
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
      if (response.code === 0) {
        const items = response.data?.items ?? [];
        const validItems = items.filter((m: MomentItem) => m.moment_id && m.user);
        if (loadMore) {
          this.moments = this.moments.concat(validItems);
          this.momentsDataSource.appendData(validItems);
        } else {
          this.moments = validItems;
          this.momentsDataSource.setData(validItems);
        }
        this.momentsHasMore = !!(response.data?.next_cursor);
        this.momentsNextCursor = response.data?.next_cursor ?? '';
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
        this.momentsRefreshing = false;
      }
    }
  }

  // ==================== 加载附近用户 ====================
  private async loadNearbyUsers(loadMore: boolean = false, refresh: boolean = false): Promise<void> {
    const ref = this.getReferencePoint();
    if (!ref) {
      if (!loadMore) { this.usersLoading = false; this.usersRefreshing = false; }
      return;
    }
    if (loadMore) {
      if (this.usersLoadingMore || !this.usersHasMore) return;
      this.usersLoadingMore = true;
    } else {
      if (!refresh || this.users.length === 0) { this.usersLoading = true; }
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
      if (response.code === 0) {
        const items = response.data?.items ?? [];
        const validItems = items.filter((u: UserItem) => u.user?.user_id);
        if (loadMore) {
          this.users = this.users.concat(validItems);
          this.usersDataSource.appendData(validItems);
        } else {
          this.users = validItems;
          this.usersDataSource.setData(validItems);
        }
        this.usersHasMore = !!(response.data?.next_cursor);
        this.usersNextCursor = response.data?.next_cursor ?? '';
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
        this.usersRefreshing = false;
      }
    }
  }

  // ==================== 加载附近群组 ====================
  private async loadNearbyGroups(loadMore: boolean = false, refresh: boolean = false): Promise<void> {
    const ref = this.getReferencePoint();
    if (!ref) {
      if (!loadMore) { this.groupsLoading = false; this.groupsRefreshing = false; }
      return;
    }
    if (loadMore) {
      if (this.groupsLoadingMore || !this.groupsHasMore) return;
      this.groupsLoadingMore = true;
    } else {
      if (!refresh || this.groups.length === 0) { this.groupsLoading = true; }
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
      if (response.code === 0) {
        const items = response.data?.items ?? [];
        const validItems = items.filter((g: GroupItem) => g.group?.group_id);
        if (loadMore) {
          this.groups = this.groups.concat(validItems);
          this.groupsDataSource.appendData(validItems);
        } else {
          this.groups = validItems;
          this.groupsDataSource.setData(validItems);
        }
        this.groupsHasMore = !!(response.data?.next_cursor);
        this.groupsNextCursor = response.data?.next_cursor ?? '';
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
        this.groupsRefreshing = false;
      }
    }
  }

  build() {
    Column() {
      this.buildHeaderSurface()

      Tabs({ controller: this.subTabController }) {
        TabContent() {
          this.buildMomentsContent()
        }
        .tabBar(this.HiddenTabBar())

        TabContent() {
          this.buildUsersContent()
        }
        .tabBar(this.HiddenTabBar())

        TabContent() {
          this.buildGroupsContent()
        }
        .tabBar(this.HiddenTabBar())
      }
      .barMode(BarMode.Fixed)
      .barHeight(0)
      .animationDuration(400)
      .onChange((index: number) => {
        this.prepareTabLoading(index);
        this.getUIContext().animateTo({ duration: 300, curve: Curve.FastOutSlowIn }, () => {
          this.currentSubTab = index;
        });
        this.loadCurrentTabData(true);
      })
      .layoutWeight(1)
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('app.color.bg_primary'))
  }

  @Builder
  private HiddenTabBar() {
    Blank()
      .width(0)
      .height(0)
  }

  @Builder
  private buildHeaderSurface() {
    Column({ space: DesignConstants.SPACING.SM }) {
      LocationBar({ showSearchButton: true })

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
        .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD,
          top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
        .backgroundColor($r('app.color.bg_primary'))
        .borderRadius(DesignConstants.RADIUS.LG)
        .transition(TransitionEffect.OPACITY.animation({ duration: 250, curve: Curve.FastOutSlowIn }))
      }

      Row({ space: 0 }) {
        this.SubTabChip('动态', 0, AppIcon.CAMERA)
        this.SubTabChip('用户', 1, AppIcon.PERSON)
        this.SubTabChip('群组', 2, AppIcon.GROUP)
      }
      .width('100%')
      .padding(3)
      .backgroundColor($r('app.color.bg_tertiary'))
      .borderRadius(DesignConstants.RADIUS.FULL)
    }
    .width('100%')
    .padding({
      top: this.getHeaderTopPadding(),
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      bottom: DesignConstants.SPACING.SM
    })
    .backgroundColor($r('app.color.bg_secondary'))
    .borderRadius({ bottomLeft: DesignConstants.RADIUS.XL, bottomRight: DesignConstants.RADIUS.XL })
    .shadow({ radius: 8, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 2 })
    .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])
  }

  @Builder
  private SubTabChip(label: string, index: number, icon: AppIcon) {
    Row({ space: 4 }) {
      Text() {
        SymbolSpan($r(icon))
          .fontSize(14)
          .fontColor([this.currentSubTab === index
            ? $r('sys.color.ohos_id_color_emphasize')
            : $r('sys.color.ohos_id_color_text_secondary')])
      }
      Text(label)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontWeight(this.currentSubTab === index ? FontWeight.Medium : FontWeight.Normal)
        .fontColor(this.currentSubTab === index
          ? $r('sys.color.ohos_id_color_emphasize')
          : $r('sys.color.ohos_id_color_text_secondary'))
    }
    .height(32)
    .layoutWeight(1)
    .justifyContent(FlexAlign.Center)
    .backgroundColor(this.currentSubTab === index
      ? $r('app.color.chip_selected_bg')
      : Color.Transparent)
    .borderRadius(DesignConstants.RADIUS.FULL)
    .onClick(() => {
      this.switchSubTab(index);
    })
  }

  // @Builder
  // private buildMapEntryButton() {
  //   Column() {
  //     Text() {
  //       SymbolSpan($r(AppIcon.MAP))
  //         .fontSize(20)
  //         .fontColor([$r('sys.color.ohos_id_color_text_primary')])
  //     }
  //   }
  //   .width(44)
  //   .height(44)
  //   .justifyContent(FlexAlign.Center)
  //   .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
  //   .borderRadius(DesignConstants.RADIUS.FULL)
  //   .shadow({ radius: 10, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 3 })
  //   .onClick(() => {
  //     this.openLocationPicker();
  //   })
  // }

  // ==================== 动态内容 ====================
  @Builder
  buildMomentsContent() {
    Column() {
      if (this.momentsLoading) {
        SystemLoadingView({ message: '正在加载附近动态...' })
      } else if (this.momentsError.length > 0 && this.moments.length === 0) {
        EmptyState({ icon: AppIcon.WARNING, message: this.momentsError, actionLabel: '重试', onAction: () => { this.loadNearbyMoments(); } })
      } else if (this.moments.length === 0) {
        EmptyState({ icon: AppIcon.CAMERA, message: '附近暂无动态\n去发布第一条吧！' })
      } else {
        Refresh({ refreshing: $$this.momentsRefreshing }) {
          WaterFlow({ footer: (): void => this.buildWaterFlowFooter() }) {
            LazyForEach(this.momentsDataSource, (moment: MomentItem, index: number) => {
              FlowItem() {
                this.buildMomentCard(moment, index)
              }
            }, (moment: MomentItem) => moment.moment_id)
          }
          .columnsTemplate('repeat(auto-fill, 160vp)')
          .columnsGap(8)
          .rowsGap(8)
          .width('100%')
          .padding({ left: 8, right: 8, top: 8, bottom: this.getListBottomPadding() })
          .cachedCount(8)
          .onReachEnd(() => { this.loadNearbyMoments(true); })
        }
        .layoutWeight(1)
        .onRefreshing(() => { this.loadNearbyMoments(false, true); })
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      }
    }
    .width('100%')
    .height('100%')
  }

  // 计算卡片图片高度，根据图片比例动态调整（小红书风格）
  private calculateImageHeight(moment: MomentItem): number {
    if (!moment.images || moment.images.length === 0) {
      return 160;  // 无图默认高度
    }

    const image = moment.images[0];
    if (image.width && image.height && image.width > 0) {
      // 列宽约为 160vp，根据原始比例计算高度
      const aspectRatio = image.height / image.width;
      const calculatedHeight = 160 * aspectRatio;
      // 限制高度范围：最小 120，最大 280
      return Math.max(120, Math.min(calculatedHeight, 280));
    }

    return 180;  // 默认高度
  }

  @State pressedIndex: number = -1  // 记录被按下的卡片索引

  @Builder
  buildMomentCard(moment: MomentItem, index: number) {
    Column({ space: 0 }) {
      // 图片 (封面) - 根据原始比例动态调整高度
      if (moment.images && moment.images.length > 0) {
        Image(moment.images[0].url)
          .width('100%')
          .height(this.calculateImageHeight(moment))
          .objectFit(ImageFit.Cover)
          .borderRadius({ topLeft: 12, topRight: 12 })
          .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
      } else {
        // 无图占位 - 纯文字卡片
        Column() {
          Text(moment.description)
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .maxLines(5)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .width('100%')
        .height(160)
        .padding(12)
        .justifyContent(FlexAlign.Center)
        .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
        .borderRadius({ topLeft: 12, topRight: 12 })
      }

      // 内容区域 - 小红书风格
      Column({ space: 4 }) {
        // 文本描述
        if (moment.description && moment.images && moment.images.length > 0) {
          Text(moment.description)
            .fontSize(13)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
            .maxLines(2)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
            .lineHeight(18)
        }

        // 用户信息 - 更紧凑的布局
        Row({ space: 4 }) {
          SystemAvatar({
            avatarUrl: moment.user?.avatar_url ?? '',
            name: moment.user?.nickname ?? '',
            avatarSize: 16
          })
          Text(moment.user?.nickname ?? '未知用户')
            .fontSize(11)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
            .layoutWeight(1)

          // 点赞数 - 放在右侧
          Row({ space: 2 }) {
            Text() {
              SymbolSpan($r(AppIcon.HEART))
                .fontSize(11)
                .fontColor([$r('sys.color.ohos_id_color_text_tertiary')])
            }
            Text(`${moment.likes_count}`)
              .fontSize(11)
              .fontColor($r('sys.color.ohos_id_color_text_tertiary'))
          }
        }
        .width('100%')
        .alignItems(VerticalAlign.Center)
      }
      .padding(8)
    }
    .width('100%')
    .backgroundColor($r('app.color.bg_secondary'))
    .borderRadius(DesignConstants.RADIUS.LG)
    .shadow({ radius: 6, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 2 })
    // 点击反馈动画
    .scale(this.pressedIndex === index ? { x: 0.97, y: 0.97 } : { x: 1, y: 1 })
    .animation({ duration: 100, curve: Curve.EaseOut })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.pressedIndex = index;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.pressedIndex = -1;
      }
    })
    .onClick(() => {
      // 打开动态详情
      this.openMomentDetail(moment);
    })
  }

  private openMomentDetail(moment: MomentItem): void {
    hilog.info(DOMAIN_NUMBER, TAG, `打开动态详情: ${moment.moment_id}`);
    // TODO: 创建 MomentDetailPage 并实现跳转
    SystemToast.show('动态详情页即将上线', this.getUIContext());
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
    Column() {
      if (this.usersLoading) {
        SystemLoadingView({ message: '正在加载附近用户...' })
          .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else if (this.usersError.length > 0 && this.users.length === 0) {
        EmptyState({ icon: AppIcon.WARNING, message: this.usersError, actionLabel: '重试', onAction: () => { this.loadNearbyUsers(); } })
          .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else if (this.users.length === 0) {
        EmptyState({ icon: AppIcon.PERSON, message: '附近暂无其他用户' })
          .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else {
        Refresh({ refreshing: $$this.usersRefreshing }) {
          List() {
            LazyForEach(this.usersDataSource, (user: UserItem) => {
              ListItem() {
                this.buildUserCard(user)
              }
            }, (user: UserItem) => user.user?.user_id ?? '')
            if (this.usersLoadingMore) {
              ListItem() {
                LoadingMoreIndicator()
              }
            }
          }
          .width('100%')
          .padding({ bottom: this.getListBottomPadding() })
          .divider({ strokeWidth: 0.5, color: $r('app.color.divider'), startMargin: 76, endMargin: 0 })
          .scrollBar(BarState.Auto)
          .onReachEnd(() => { this.loadNearbyUsers(true); })
        }
        .layoutWeight(1)
        .onRefreshing(() => { this.loadNearbyUsers(false, true); })
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      }
    }
    .width('100%')
    .height('100%')
  }

  @Builder
  buildUserCard(user: UserItem) {
    Row({ space: 12 }) {
      SystemAvatar({
        avatarUrl: user.user?.avatar_url ?? '',
        name: user.user?.nickname ?? '',
        avatarSize: 48
      })

      Column({ space: 2 }) {
        Text(user.user?.nickname ?? '')
          .fontSize(16)
          .fontWeight(FontWeight.Medium)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        if (user.bio) {
          Text(user.bio)
            .fontSize(13)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        } else if (user.location?.distance !== undefined && user.location?.distance !== null) {
          Text(this.formatDistance(user.location.distance))
            .fontSize(13)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)

      Button('关注')
        .type(ButtonType.Capsule)
        .backgroundColor(Color.Transparent)
        .fontColor($r('sys.color.ohos_id_color_emphasize'))
        .fontSize(13)
        .fontWeight(FontWeight.Medium)
        .height(32)
        .borderWidth(1)
        .borderColor($r('sys.color.ohos_id_color_emphasize'))
        .onClick(() => { this.followUser(user); })
    }
    .width('100%')
    .padding({ left: 16, right: 16, top: 12, bottom: 12 })
    .alignItems(VerticalAlign.Center)
  }

  // ==================== 群组内容 ====================
  @Builder
  buildGroupsContent() {
    Column() {
      if (this.groupsLoading) {
        SystemLoadingView({ message: '正在加载附近群组...' })
          .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else if (this.groupsError.length > 0 && this.groups.length === 0) {
        EmptyState({ icon: AppIcon.WARNING, message: this.groupsError, actionLabel: '重试', onAction: () => { this.loadNearbyGroups(); } })
          .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else if (this.groups.length === 0) {
        EmptyState({ icon: AppIcon.GROUP, message: '附近暂无群组' })
          .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else {
        Refresh({ refreshing: $$this.groupsRefreshing }) {
          List() {
            LazyForEach(this.groupsDataSource, (group: GroupItem) => {
              ListItem() {
                this.buildGroupCard(group)
              }
            }, (group: GroupItem) => group.group?.group_id ?? '')
            if (this.groupsLoadingMore) {
              ListItem() {
                LoadingMoreIndicator()
              }
            }
          }
          .width('100%')
          .padding({ bottom: this.getListBottomPadding() })
          .divider({ strokeWidth: 0.5, color: $r('app.color.divider'), startMargin: 76, endMargin: 0 })
          .scrollBar(BarState.Auto)
          .onReachEnd(() => { this.loadNearbyGroups(true); })
        }
        .layoutWeight(1)
        .onRefreshing(() => { this.loadNearbyGroups(false, true); })
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      }
    }
    .width('100%')
    .height('100%')
  }

  @Builder
  buildGroupCard(group: GroupItem) {
    Row({ space: 12 }) {
      SystemAvatar({
        avatarUrl: group.group?.avatar_url ?? '',
        name: group.group?.name ?? '',
        avatarSize: 48,
        shape: 'squircle'
      })

      Column({ space: 2 }) {
        Text(group.group?.name ?? '')
          .fontSize(16)
          .fontWeight(FontWeight.Medium)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        Row({ space: 8 }) {
          Text(`${group.group?.members_count ?? 0}人`)
            .fontSize(13)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))

          if ((group.location as RelativeLocation)?.distance !== undefined &&
            (group.location as RelativeLocation)?.distance !== null) {
            Text(this.formatDistance((group.location as RelativeLocation).distance!))
              .fontSize(13)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          }
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
        .onClick(() => { this.joinGroup(group); })
    }
    .width('100%')
    .padding({ left: 16, right: 16, top: 12, bottom: 12 })
    .alignItems(VerticalAlign.Center)
    .onClick(() => { this.openGroupDetail(group); })
  }

  @Builder
  buildWaterFlowFooter() {
    LoadingMoreIndicator()
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
