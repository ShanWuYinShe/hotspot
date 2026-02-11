/**
 * 地图页面 - 显示附近的用户、群组和瞬间
 * 使用 HarmonyOS 原生地图能力 + 系统颜色资源
 */

import { GlobalStateManager, KEY_BROWSE_ANCHOR, KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { ApiClient } from '../../../core/api/ApiClient';
import { LocationUtil, LocationInfo } from '../../../core/location/LocationUtil';
import { PageNames } from '../../../constants/AppConstants';
import {
  UserItem,
  GroupItem,
  MomentItem,
  SearchUsersRequest,
  SearchGroupsRequest,
  SearchMomentsRequest,
  GeoPoint,
  RelativeLocation
} from '../../../core/types/api';
import { SystemToast } from '../../../components/SystemToast';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { NativeMapComponent, NativeMapActions, MapMarker, MapRegion } from '../../../components/NativeMapComponent';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { AppRouter, LocationPickerRouteParam, GroupDetailRouteParam } from '../../../core/router/AppRouter';

const DOMAIN_NUMBER: number = 0xFF12;
const TAG: string = 'MapPage';

interface ExtendedMapMarker extends MapMarker {
  type: 'user' | 'group' | 'moment';
  data: UserItem | GroupItem | MomentItem;
}

interface MapCenter {
  latitude: number;
  longitude: number;
}

@Component
export struct MapPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @State markers: ExtendedMapMarker[] = [];
  @State isLoading: boolean = false;
  @State errorMessage: string = '';
  @State selectedMarker: ExtendedMapMarker | null = null;
  @State showMarkerDetail: boolean = false;
  @State mapCenter: MapCenter | null = null;
  @State currentFilter: 'all' | 'users' | 'groups' | 'moments' = 'all';

  @State mapRegion: MapRegion = {
    latitude: 39.9042,
    longitude: 116.4074,
    latitudeDelta: 0.01,
    longitudeDelta: 0.01
  };
  private mapActions: NativeMapActions | null = null;

  async aboutToAppear() {
    hilog.info(DOMAIN_NUMBER, TAG, '地图页面初始化');
    const center = this.browseAnchor || this.currentLocation;
    if (center) {
      this.updateMapCenter(center.latitude, center.longitude);
      await this.loadMapData();
    } else {
      await this.getCurrentLocation();
    }
  }

  private openBrowseAnchorPicker(): void {
    const pickerParams = new LocationPickerRouteParam();
    pickerParams.mode = 'browse_anchor';
    pickerParams.title = '设置浏览锚点';
    pickerParams.confirmText = '设为浏览锚点';
    pickerParams.hintText = '点击地图设置浏览锚点';
    pickerParams.resultKey = KEY_BROWSE_ANCHOR;
    AppRouter.push(PageNames.LOCATION_PICKER, pickerParams);
  }

  private jumpToMyLocation(): void {
    if (!this.currentLocation) {
      this.getCurrentLocation();
      return;
    }
    this.updateMapCenter(this.currentLocation.latitude, this.currentLocation.longitude);
    this.loadMapData();
  }

  private updateMapCenter(lat: number, lng: number) {
    this.mapCenter = { latitude: lat, longitude: lng };
    this.mapRegion = {
      latitude: lat,
      longitude: lng,
      latitudeDelta: 0.01,
      longitudeDelta: 0.01
    };
  }

  private async loadMapData() {
    if (!this.mapCenter) return;

    this.isLoading = true;
    this.markers = [];

    try {
      const promises: Promise<void>[] = [];

      if (this.currentFilter === 'all' || this.currentFilter === 'users') {
        promises.push(this.loadNearbyUsers());
      }
      if (this.currentFilter === 'all' || this.currentFilter === 'groups') {
        promises.push(this.loadNearbyGroups());
      }
      if (this.currentFilter === 'all' || this.currentFilter === 'moments') {
        promises.push(this.loadNearbyMoments());
      }

      await LoadingUtil.withMinimumTime(() => Promise.all(promises));
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载地图数据失败: ${error}`);
      SystemToast.showError('加载地图数据失败', this.getUIContext());
    } finally {
      this.isLoading = false;
    }
  }

  private switchFilter(filter: 'all' | 'users' | 'groups' | 'moments') {
    if (this.currentFilter === filter) return;
    this.currentFilter = filter;
    this.loadMapData();
  }

  private async loadNearbyUsers(): Promise<void> {
    if (!this.mapCenter) return;
    try {
      const myId = await AuthAccountManager.getCurrentUserId();
      const refPoint: GeoPoint = { latitude: this.mapCenter.latitude, longitude: this.mapCenter.longitude };
      const request: SearchUsersRequest = {
        reference_point: refPoint,
        radius: 2000,
        pagination: { limit: 50 }
      };
      const response = await ApiClient.searchUsers(request);

      if (response.code === 0 && response.data) {
        const users = response.data.items
          .filter((u: UserItem) => (myId ? u.user?.user_id !== String(myId) : true))
          .filter((u: UserItem) => u.location?.point);

        const userMarkers: ExtendedMapMarker[] = users.map((user: UserItem): ExtendedMapMarker => {
          const pt = user.location!.point;
          return {
            id: `user_${user.user?.user_id ?? ''}`,
            type: 'user',
            latitude: pt.latitude,
            longitude: pt.longitude,
            title: user.user?.nickname ?? '用户',
            snippet: user.bio ?? '用户',
            data: user
          };
        });
        this.markers.push(...userMarkers);
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载附近用户失败: ${error}`);
    }
  }

  private async loadNearbyGroups(): Promise<void> {
    if (!this.mapCenter) return;
    try {
      const refPoint: GeoPoint = { latitude: this.mapCenter.latitude, longitude: this.mapCenter.longitude };
      const request: SearchGroupsRequest = {
        reference_point: refPoint,
        radius: 5000,
        pagination: { limit: 50 }
      };
      const response = await ApiClient.searchGroups(request);

      if (response.code === 0 && response.data) {
        const groupMarkers: ExtendedMapMarker[] = response.data.items
          .filter((g: GroupItem) => g.location?.point)
          .map((group: GroupItem): ExtendedMapMarker => {
            const pt = group.location!.point;
            return {
              id: `group_${group.group?.group_id ?? ''}`,
              type: 'group',
              latitude: pt.latitude,
              longitude: pt.longitude,
              title: group.group?.name ?? '群组',
              snippet: `${group.group?.members_count ?? 0} 成员`,
              data: group
            };
          });
        this.markers.push(...groupMarkers);
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载附近群组失败: ${error}`);
    }
  }

  private async loadNearbyMoments(): Promise<void> {
    if (!this.mapCenter) return;
    try {
      const refPoint: GeoPoint = { latitude: this.mapCenter.latitude, longitude: this.mapCenter.longitude };
      const request: SearchMomentsRequest = {
        reference_point: refPoint,
        radius: 3000,
        pagination: { limit: 50 }
      };
      const response = await ApiClient.searchMoments(request);

      if (response.code === 0 && response.data) {
        const momentMarkers: ExtendedMapMarker[] = response.data.items
          .filter((m: MomentItem) => m.location?.point)
          .map((moment: MomentItem): ExtendedMapMarker => {
            const pt = moment.location!.point;
            return {
              id: `moment_${moment.moment_id}`,
              type: 'moment',
              latitude: pt.latitude,
              longitude: pt.longitude,
              title: moment.user?.nickname ?? '瞬间',
              snippet: moment.description.length > 20 ?
                moment.description.substring(0, 20) + '...' : moment.description,
              data: moment
            };
          });
        this.markers.push(...momentMarkers);
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载附近瞬间失败: ${error}`);
    }
  }

  private handleMarkerClick(marker: MapMarker) {
    this.selectedMarker = marker as ExtendedMapMarker;
    this.showMarkerDetail = true;
  }

  private async getCurrentLocation() {
    try {
      this.isLoading = true;
      const location = await LocationUtil.getCurrentLocation();
      if (location) {
        this.updateMapCenter(location.latitude, location.longitude);
        await this.loadMapData();
      } else {
        this.errorMessage = '无法获取当前位置';
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `获取位置失败: ${error}`);
      this.errorMessage = '获取位置失败';
    } finally {
      this.isLoading = false;
    }
  }

  private navigateToDetail() {
    if (!this.selectedMarker) return;

    const marker = this.selectedMarker;
    this.showMarkerDetail = false;

    switch (marker.type) {
      case 'user':
        SystemToast.show(`查看用户: ${marker.title}`, this.getUIContext());
        break;
      case 'group':
        const detailParams = new GroupDetailRouteParam();
        detailParams.groupId = (marker.data as GroupItem).group?.group_id ?? '';
        AppRouter.push(PageNames.GROUP_DETAIL, detailParams);
        break;
      case 'moment':
        SystemToast.show(`查看瞬间: ${marker.title}`, this.getUIContext());
        break;
    }
  }

  private getMarkerIcon(type: 'user' | 'group' | 'moment'): AppIcon {
    switch (type) {
      case 'user': return AppIcon.PERSON;
      case 'group': return AppIcon.GROUP;
      case 'moment': return AppIcon.CAMERA;
      default: return AppIcon.LOCATION;
    }
  }

  build() {
    NavDestination() {
      Column() {
      // 顶部工具栏
      Row() {
        // 返回按钮
        Text() {
          SymbolSpan($r(AppIcon.BACK))
            .fontSize(22)
            .fontColor([$r('sys.color.ohos_id_color_text_primary')])
        }
        .width(40)
        .height(40)
        .textAlign(TextAlign.Center)
        .onClick(() => {
          AppRouter.pop();
        })

        Text('地图')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Bold)
          .layoutWeight(1)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))

        Row({ space: DesignConstants.SPACING.SM }) {
          // 锚点
          Text() {
            SymbolSpan($r(AppIcon.LOCATION))
              .fontSize(22)
              .fontColor([$r('sys.color.ohos_id_color_text_primary')])
          }
          .onClick(() => this.openBrowseAnchorPicker())

          // 我的位置
          Text() {
            SymbolSpan($r(AppIcon.MAP))
              .fontSize(22)
              .fontColor([$r('sys.color.ohos_id_color_text_primary')])
          }
          .onClick(() => this.jumpToMyLocation())

          // 刷新
          Text() {
            SymbolSpan($r(AppIcon.REFRESH))
              .fontSize(22)
              .fontColor([$r('sys.color.ohos_id_color_text_primary')])
          }
          .onClick(() => this.loadMapData())
        }
      }
      .width('100%')
      .height(56)
      .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      // 筛选器
      Row({ space: DesignConstants.SPACING.SM }) {
        this.FilterChip('全部', 'all')
        this.FilterChip('用户', 'users')
        this.FilterChip('群组', 'groups')
        this.FilterChip('瞬间', 'moments')
      }
      .width('100%')
      .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
      .justifyContent(FlexAlign.Start)
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      // 地图区域
      Stack() {
        if (this.errorMessage) {
          Column({ space: DesignConstants.SPACING.LG }) {
            Text() {
              SymbolSpan($r(AppIcon.WARNING))
                .fontSize(48)
                .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
            }

            Text(this.errorMessage)
              .fontSize(DesignConstants.FONT.SIZE.LG)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))

            Button('重试')
              .type(ButtonType.Capsule)
              .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
              .fontColor(Color.White)
              .height(40)
              .onClick(() => this.getCurrentLocation())
          }
          .justifyContent(FlexAlign.Center)
          .alignItems(HorizontalAlign.Center)
          .width('100%')
          .height('100%')
        } else {
          NativeMapComponent({
            mapRegion: this.mapRegion,
            markers: this.markers,
            showUserLocation: true,
            showZoomControls: false,
            onMapClick: (coord) => {},
            onMarkerClick: (marker) => { this.handleMarkerClick(marker); },
            onActionsReady: (actions: NativeMapActions) => { this.mapActions = actions; }
          })

          // 右下角浮动控制按钮组：缩放 + 定位
          Column({ space: 12 }) {
            // 放大按钮
            Column() {
              Text('+')
                .fontSize(22)
                .fontWeight(FontWeight.Medium)
                .fontColor($r('sys.color.ohos_id_color_text_primary'))
            }
            .width(44)
            .height(44)
            .justifyContent(FlexAlign.Center)
            .backgroundColor($r('sys.color.ohos_id_color_background'))
            .borderRadius(DesignConstants.RADIUS.FULL)
            .shadow({ radius: 8, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 2 })
            .onClick(() => { this.mapActions?.zoomIn(); })

            // 缩小按钮
            Column() {
              Text('−')
                .fontSize(22)
                .fontWeight(FontWeight.Medium)
                .fontColor($r('sys.color.ohos_id_color_text_primary'))
            }
            .width(44)
            .height(44)
            .justifyContent(FlexAlign.Center)
            .backgroundColor($r('sys.color.ohos_id_color_background'))
            .borderRadius(DesignConstants.RADIUS.FULL)
            .shadow({ radius: 8, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 2 })
            .onClick(() => { this.mapActions?.zoomOut(); })

            // 回到当前位置按钮
            Column() {
              Text() {
                SymbolSpan($r(AppIcon.LOCATION))
                  .fontSize(22)
                  .fontColor([$r('sys.color.ohos_id_color_emphasize')])
              }
            }
            .width(44)
            .height(44)
            .justifyContent(FlexAlign.Center)
            .backgroundColor($r('sys.color.ohos_id_color_background'))
            .borderRadius(DesignConstants.RADIUS.FULL)
            .shadow({ radius: 8, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 2 })
            .onClick(() => { this.jumpToMyLocation(); })
          }
          .position({ right: 16, bottom: 24 })

          if (this.isLoading) {
            Column() {
              LoadingProgress()
                .width(48)
                .height(48)
                .color($r('sys.color.ohos_id_color_emphasize'))
            }
            .width('100%')
            .height('100%')
            .justifyContent(FlexAlign.Center)
            .backgroundColor($r('app.color.color_overlay_medium'))
            .hitTestBehavior(HitTestMode.Transparent)
          }
        }
      }
      .layoutWeight(1)
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
    .bindSheet($$this.showMarkerDetail, this.buildMarkerDetailSheet(), {
      height: SheetSize.MEDIUM,
      showClose: false,
      dragBar: true,
      backgroundColor: $r('sys.color.ohos_id_color_background')
    })
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
    })
  }

  @Builder
  FilterChip(text: string, filter: 'all' | 'users' | 'groups' | 'moments') {
    Text(text)
      .fontSize(DesignConstants.FONT.SIZE.SM)
      .fontColor(this.currentFilter === filter ? Color.White : $r('sys.color.ohos_id_color_text_primary'))
      .backgroundColor(this.currentFilter === filter ?
        $r('sys.color.ohos_id_color_emphasize') : $r('sys.color.ohos_id_color_component_normal'))
      .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
      .borderRadius(DesignConstants.RADIUS.FULL)
      .onClick(() => this.switchFilter(filter))
  }

  @Builder
  buildMarkerTypeContent() {
    if (this.selectedMarker) {
      if (this.selectedMarker.type === 'group') {
        Column({ space: DesignConstants.SPACING.SM }) {
          Text(`成员数: ${(this.selectedMarker.data as GroupItem).group?.members_count ?? 0}`)
            .fontSize(DesignConstants.FONT.SIZE.MD)

          if ((this.selectedMarker.data as GroupItem).description) {
            Text((this.selectedMarker.data as GroupItem).description!)
              .fontSize(DesignConstants.FONT.SIZE.MD)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          }
        }
        .alignItems(HorizontalAlign.Start)
      } else if (this.selectedMarker.type === 'moment') {
        Column({ space: DesignConstants.SPACING.SM }) {
          Text(`点赞: ${(this.selectedMarker.data as MomentItem).likes_count} | 评论: ${(this.selectedMarker.data as MomentItem).comments_count}`)
            .fontSize(DesignConstants.FONT.SIZE.MD)

          if ((this.selectedMarker.data as MomentItem).location?.location_name) {
            Text((this.selectedMarker.data as MomentItem).location!.location_name)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          }
        }
        .alignItems(HorizontalAlign.Start)
      }
    }
  }

  @Builder
  buildMarkerDetailSheet() {
    if (this.selectedMarker) {
      Column() {
        // 标题栏
        Row({ space: DesignConstants.SPACING.MD }) {
          Text() {
            SymbolSpan($r(this.getMarkerIcon(this.selectedMarker.type)))
              .fontSize(24)
              .fontColor([$r('sys.color.ohos_id_color_emphasize')])
          }

          Text(this.selectedMarker.title)
            .fontSize(DesignConstants.FONT.SIZE.XL)
            .fontWeight(FontWeight.Bold)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
            .layoutWeight(1)
        }
        .width('100%')
        .padding(DesignConstants.SPACING.LG)

        // 详情内容
        Column({ space: DesignConstants.SPACING.SM }) {
          if (this.selectedMarker.snippet) {
            Text(this.selectedMarker.snippet)
              .fontSize(DesignConstants.FONT.SIZE.MD)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          }

          this.buildMarkerTypeContent()
        }
        .alignItems(HorizontalAlign.Start)
        .width('100%')
        .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })

        // 操作按钮
        Row({ space: DesignConstants.SPACING.MD }) {
          Button('查看详情')
            .type(ButtonType.Capsule)
            .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
            .fontColor(Color.White)
            .layoutWeight(1)
            .height(44)
            .onClick(() => this.navigateToDetail())

          Button('取消')
            .type(ButtonType.Capsule)
            .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
            .layoutWeight(1)
            .height(44)
            .onClick(() => this.showMarkerDetail = false)
        }
        .width('100%')
        .padding(DesignConstants.SPACING.LG)
        .justifyContent(FlexAlign.SpaceAround)
      }
      .width('100%')
    }
  }
}

@Builder
export function MapPageBuilder() {
  MapPage()
}
