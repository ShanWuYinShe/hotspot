/**
 * 地图页面 - 显示附近的用户、群组和瞬间
 * 使用HarmonyOS原生地图能力
 */

import { GlobalStateManager, KEY_BROWSE_ANCHOR, KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { ApiClient } from '../../../core/api/ApiClient';
import { LocationUtil, LocationInfo } from '../../../core/location/LocationUtil';
import { PageNames } from '../../../constants/AppConstants';
import { 
  UserSearchItem, 
  RelativeLocation,
  GroupSearchItem, 
  MomentSearchItem,
  SearchUsersByLocationRequest,
  SearchGroupsByLocationRequest,
  SearchMomentsByLocationRequest
} from '../../../core/types/api';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon, SystemIcon } from '../../../components/SystemIcon';
import { NativeMapComponent, MapMarker, MapRegion } from '../../../components/NativeMapComponent';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';

const DOMAIN_NUMBER: number = 0xFF12;
const TAG: string = 'MapPage';

// 扩展 MapMarker 接口以适配 NativeMapComponent
interface ExtendedMapMarker extends MapMarker {
  type: 'user' | 'group' | 'moment';
  data: UserSearchItem | GroupSearchItem | MomentSearchItem;
}

interface MapCenter {
  latitude: number;
  longitude: number;
}

@Component
export struct MapPage {
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

  async aboutToAppear() {
    hilog.info(DOMAIN_NUMBER, TAG, '地图页面初始化');
    
    // 设置地图中心
    const center = this.browseAnchor || this.currentLocation;
    if (center) {
      this.updateMapCenter(center.latitude, center.longitude);
      await this.loadMapData();
    } else {
      await this.getCurrentLocation();
    }
  }

  private openBrowseAnchorPicker(): void {
    try {
      const routerInstance = this.getUIContext().getRouter();
      routerInstance.pushUrl({
        url: `features/map/pages/${PageNames.LOCATION_PICKER}`,
        params: {
          mode: 'browse_anchor',
          title: '设置浏览锚点',
          confirmText: '设为浏览锚点',
          hintText: '点击地图设置浏览锚点',
          resultKey: KEY_BROWSE_ANCHOR
        }
      });
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `打开浏览锚点选择失败: ${error}`);
      SystemToast.showError('打开锚点选择失败', this.getUIContext());
    }
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
    this.markers = []; // Clear existing
    
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
      
      await Promise.all(promises);
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

  /**
   * 加载附近用户
   */
  private async loadNearbyUsers(): Promise<void> {
    if (!this.mapCenter) return;

    try {
      const me = GlobalStateManager.getInstance().getCurrentUser();
      const myId = me?.user_id;
      const request: SearchUsersByLocationRequest = {
        reference_point: {
          latitude: this.mapCenter.latitude,
          longitude: this.mapCenter.longitude
        },
        radius: 2000,
        limit: 50
      };
      const response = await ApiClient.searchUsersByLocation(request);

      if (response.code === 0 && response.data) {
        const users = response.data.items
          .filter((u: UserSearchItem) => (typeof myId === 'number' ? u.user_id !== myId : true))
          // 位置信息可能因隐私返回 null/undefined，此处直接过滤避免运行时崩溃
          .filter((u: UserSearchItem) => !!u.location);

        const userMarkers: ExtendedMapMarker[] = users.map((user: UserSearchItem): ExtendedMapMarker => ({
          id: `user_${user.user_id}`,
          type: 'user',
          latitude: (user.location as RelativeLocation).latitude,
          longitude: (user.location as RelativeLocation).longitude,
          title: user.nickname,
          snippet: user.bio || '用户',
          data: user
        }));
        this.markers.push(...userMarkers);
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载附近用户失败: ${error}`);
    }
  }

  /**
   * 加载附近群组
   */
  private async loadNearbyGroups(): Promise<void> {
    if (!this.mapCenter) return;

    try {
      const request: SearchGroupsByLocationRequest = {
        reference_point: {
          latitude: this.mapCenter.latitude,
          longitude: this.mapCenter.longitude
        },
        radius: 5000,
        limit: 50
      };
      const response = await ApiClient.searchGroupsByLocation(request);

      if (response.code === 0 && response.data) {
        const groupMarkers: ExtendedMapMarker[] = response.data.items.map((group: GroupSearchItem): ExtendedMapMarker => ({
          id: `group_${group.group_id}`,
          type: 'group',
          latitude: group.location!.latitude,
          longitude: group.location!.longitude,
          title: group.name,
          snippet: `${group.members_count} 成员`,
          data: group
        }));
        this.markers.push(...groupMarkers);
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载附近群组失败: ${error}`);
    }
  }

  /**
   * 加载附近瞬间
   */
  private async loadNearbyMoments(): Promise<void> {
    if (!this.mapCenter) return;

    try {
      const request: SearchMomentsByLocationRequest = {
        reference_point: {
          latitude: this.mapCenter.latitude,
          longitude: this.mapCenter.longitude
        },
        radius: 3000,
        limit: 50
      };
      const response = await ApiClient.searchMomentsByLocation(request);

      if (response.code === 0 && response.data) {
        const momentMarkers: ExtendedMapMarker[] = response.data.items.map((moment: MomentSearchItem): ExtendedMapMarker => ({
          id: `moment_${moment.moment_id}`,
          type: 'moment',
          latitude: moment.location!.latitude,
          longitude: moment.location!.longitude,
          title: moment.user.nickname,
          snippet: moment.description.length > 20 ? 
            moment.description.substring(0, 20) + '...' : moment.description,
          data: moment
        }));
        this.markers.push(...momentMarkers);
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载附近瞬间失败: ${error}`);
    }
  }

  /**
   * 处理标记点击
   */
  private handleMarkerClick(marker: MapMarker) {
    // 强制类型转换，因为我们在load时确保了类型是ExtendedMapMarker
    this.selectedMarker = marker as ExtendedMapMarker;
    this.showMarkerDetail = true;
  }
  
  /**
   * 获取当前位置
   */
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
        // 导航到用户资料页面或开始聊天
        SystemToast.show(`查看用户: ${marker.title}`, this.getUIContext());
        break;
      case 'group':
        // 导航到群组详情页面
        try {
          this.getUIContext().getRouter().pushUrl({
            url: `features/group/pages/${PageNames.GROUP_DETAIL}`,
            params: { groupId: (marker.data as GroupSearchItem).group_id }
          });
        } catch (error) {
          hilog.error(DOMAIN_NUMBER, TAG, `跳转群组详情失败: ${error}`);
        }
        break;
      case 'moment':
        // 显示瞬间详情
        SystemToast.show(`查看瞬间: ${marker.title}`, this.getUIContext());
        break;
    }
  }

  /**
   * 获取标记图标
   */
  private getMarkerIcon(type: 'user' | 'group' | 'moment'): string {
    switch (type) {
      case 'user': return '👤';
      case 'group': return '👥';
      case 'moment': return '📸';
      default: return '📍';
    }
  }

  build() {
    Column() {
      // 顶部工具栏
      Row() {
        Text('地图')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Bold)
          .layoutWeight(1)
          .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)

        Button('锚点')
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .backgroundColor(Color.Transparent)
          .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
          .onClick(() => this.openBrowseAnchorPicker())

        Button('我的位置')
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .backgroundColor(Color.Transparent)
          .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
          .onClick(() => this.jumpToMyLocation())

        Button('刷新')
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .backgroundColor(Color.Transparent)
          .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
          .onClick(() => this.loadMapData())
      }
      .width('100%')
      .height(56)
      .padding({ left: 16, right: 16 })
      .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)

      // 筛选器
      Row() {
        this.FilterButton('全部', 'all')
        this.FilterButton('用户', 'users')
        this.FilterButton('群组', 'groups')
        this.FilterButton('瞬间', 'moments')
      }
      .width('100%')
      .padding({ left: 16, right: 16, top: 8, bottom: 8 })
      .justifyContent(FlexAlign.SpaceAround)

      // 地图区域
      Stack() {
        if (this.errorMessage) {
           Column() {
            Text('😕')
              .fontSize(48)
              .margin({ bottom: 16 })
            
            Text(this.errorMessage)
              .fontSize(DesignConstants.FONT.SIZE.LG)
              .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
              .margin({ bottom: 24 })
            
            Button('重试')
              .type(ButtonType.Capsule)
              .backgroundColor(DesignConstants.COLOR.BRAND_PRIMARY)
              .fontColor(Color.White)
              .onClick(() => this.getCurrentLocation())
          }
          .justifyContent(FlexAlign.Center)
          .alignItems(HorizontalAlign.Center)
          .width('100%')
          .height('100%')
        } else {
          // 使用原生地图组件
          NativeMapComponent({
            mapRegion: this.mapRegion,
            markers: this.markers,
            showUserLocation: true,
            showZoomControls: false,
            onMapClick: (coord) => {
              // 处理地图点击
            },
            onMarkerClick: (marker) => {
              this.handleMarkerClick(marker);
            }
          })
          
          // 加载遮罩
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
             .backgroundColor('rgba(255,255,255,0.5)')
             .hitTestBehavior(HitTestMode.Transparent)
          }
        }
      }
      .layoutWeight(1)
    }
    .width('100%')
    .height('100%')
    .backgroundColor(DesignConstants.COLOR.SECONDARY_BG_LIGHT)
    .bindSheet($$this.showMarkerDetail, this.buildMarkerDetailSheet(), {
      height: SheetSize.MEDIUM,
      // 允许拖拽/点击外部关闭时，不显示右上角关闭按钮（避免冗余）
      showClose: false,
      dragBar: true,
      backgroundColor: DesignConstants.COLOR.PRIMARY_BG_LIGHT
    })
  }

  @Builder
  FilterButton(text: string, filter: 'all' | 'users' | 'groups' | 'moments') {
    Button(text)
      .fontSize(12)
      .height(28)
      .backgroundColor(this.currentFilter === filter ? 
        DesignConstants.COLOR.BRAND_PRIMARY : Color.Transparent)
      .fontColor(this.currentFilter === filter ? 
        Color.White : DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
      .onClick(() => this.switchFilter(filter))
  }

  @Builder
  buildMarkerTypeContent() {
    if (this.selectedMarker) {
      if (this.selectedMarker.type === 'group') {
        Column() {
          Text(`成员数: ${(this.selectedMarker.data as GroupSearchItem).members_count}`)
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .margin({ bottom: 8 })
          
          if ((this.selectedMarker.data as GroupSearchItem).description) {
            Text((this.selectedMarker.data as GroupSearchItem).description!)
              .fontSize(DesignConstants.FONT.SIZE.MD)
              .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
              .margin({ bottom: 12 })
          }
        }
        .alignItems(HorizontalAlign.Start)
      } else if (this.selectedMarker.type === 'moment') {
        Column() {
          Text(`点赞: ${(this.selectedMarker.data as MomentSearchItem).likes_count} | 评论: ${(this.selectedMarker.data as MomentSearchItem).comments_count}`)
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .margin({ bottom: 8 })
          
          Text((this.selectedMarker.data as MomentSearchItem).location!.location_name)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
            .margin({ bottom: 12 })
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
        Row() {
          Text(this.getMarkerIcon(this.selectedMarker.type))
            .fontSize(24)
            .margin({ right: 12 })
          
          Text(this.selectedMarker.title)
            .fontSize(DesignConstants.FONT.SIZE.XL)
            .fontWeight(FontWeight.Bold)
            .layoutWeight(1)
        }
        .width('100%')
        .padding(16)

        // 详情内容
        Column() {
          if (this.selectedMarker.snippet) {
            Text(this.selectedMarker.snippet)
              .fontSize(DesignConstants.FONT.SIZE.MD)
              .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
              .margin({ bottom: 12 })
          }

          // 根据类型显示不同内容
          this.buildMarkerTypeContent()
        }
        .alignItems(HorizontalAlign.Start)
        .width('100%')
        .padding({ left: 16, right: 16 })

        // 操作按钮
        Row() {
          Button('查看详情')
            .type(ButtonType.Capsule)
            .backgroundColor(DesignConstants.COLOR.BRAND_PRIMARY)
            .fontColor(Color.White)
            .layoutWeight(1)
            .onClick(() => this.navigateToDetail())
          
          Button('取消')
            .type(ButtonType.Capsule)
            .backgroundColor(Color.Transparent)
            .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
            .onClick(() => this.showMarkerDetail = false)
        }
        .width('100%')
        .padding(16)
        .justifyContent(FlexAlign.SpaceAround)
      }
      .width('100%')
    }
  }
}
