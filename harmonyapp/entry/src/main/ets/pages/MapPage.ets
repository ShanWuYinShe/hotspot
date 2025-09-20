/**
 * 地图页面 - 显示附近的用户、群组和瞬间
 * 使用HarmonyOS原生地图能力
 */

import { GlobalStateManager, GlobalState } from '../common/state/GlobalStateManager';
import { ApiClient } from '../common/api/ApiClient';
import { LocationUtil } from '../common/location/LocationUtil';
import { PageNames } from '../constants/AppConstants';
import { 
  SearchUserByLocationResponse, 
  SearchGroupByLocationResponse, 
  SearchMomentsByLocationResponse 
} from '../common/types/BackendApiTypes';
import { SystemLoadingView } from '../components/SystemLoadingView';
import { SystemToast } from '../components/SystemToast';
import { SystemAvatar } from '../components/SystemAvatar';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF12;
const TAG: string = 'MapPage';

interface MapMarker {
  id: string;
  type: 'user' | 'group' | 'moment';
  latitude: number;
  longitude: number;
  title: string;
  subtitle?: string;
  data: SearchUserByLocationResponse | SearchGroupByLocationResponse | SearchMomentsByLocationResponse;
}

interface MapCenter {
  latitude: number;
  longitude: number;
}

@Entry
@Component
struct MapPage {
  @State globalState: GlobalState = GlobalStateManager.getInstance().getState();
  private stateManager = GlobalStateManager.getInstance();
  @State markers: MapMarker[] = [];
  @State isLoading: boolean = false;
  @State errorMessage: string = '';
  @State selectedMarker: MapMarker | null = null;
  @State showMarkerDetail: boolean = false;
  @State mapCenter: MapCenter | null = null;
  @State currentFilter: 'all' | 'users' | 'groups' | 'moments' = 'all';

  async aboutToAppear() {
    hilog.info(DOMAIN_NUMBER, TAG, '地图页面初始化');
    
    // 订阅状态变化
    this.stateManager.subscribe('map_page', (state: GlobalState) => {
      this.globalState = state;
    });

    // 设置地图中心
    if (this.globalState.currentLocation) {
      this.mapCenter = {
        latitude: this.globalState.currentLocation.latitude,
        longitude: this.globalState.currentLocation.longitude
      };
      await this.loadMapData();
    } else {
      await this.getCurrentLocation();
    }
  }

  aboutToDisappear() {
    // 取消订阅
    this.stateManager.unsubscribe('map_page');
  }

  /**
   * 获取当前位置
   */
  private async getCurrentLocation() {
    try {
      this.isLoading = true;
      const location = await LocationUtil.getCurrentLocation();
      if (location) {
        this.mapCenter = {
          latitude: location.latitude,
          longitude: location.longitude
        };
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

  /**
   * 加载地图数据
   */
  private async loadMapData() {
    if (!this.mapCenter) return;

    this.isLoading = true;
    this.errorMessage = '';
    this.markers = [];

    try {
      // 并行加载所有数据
      const responses = await Promise.all([
        this.loadNearbyUsers(),
        this.loadNearbyGroups(),
        this.loadNearbyMoments()
      ]);
      const usersResponse = responses[0];
      const groupsResponse = responses[1];
      const momentsResponse = responses[2];

      hilog.info(DOMAIN_NUMBER, TAG, `地图数据加载完成: ${this.markers.length} 个标记点`);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载地图数据失败: ${error}`);
      this.errorMessage = '加载地图数据失败';
    } finally {
      this.isLoading = false;
    }
  }

  /**
   * 加载附近用户
   */
  private async loadNearbyUsers(): Promise<void> {
    if (!this.mapCenter || (this.currentFilter !== 'all' && this.currentFilter !== 'users')) {
      return;
    }

    try {
      const response = await ApiClient.searchUsersByLocation({
        latitude: this.mapCenter.latitude,
        longitude: this.mapCenter.longitude,
        radius: 2000,
        limit: 50
      });

      if (response.code === 0 && response.data) {
        const userMarkers: MapMarker[] = response.data.items.map((user): MapMarker => ({
          id: `user_${user.user_id}`,
          type: 'user',
          latitude: user.latitude || this.mapCenter!.latitude,
          longitude: user.longitude || this.mapCenter!.longitude,
          title: user.nickname,
          subtitle: user.bio || '用户',
          data: user
        }));
        this.markers = [...this.markers, ...userMarkers];
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载附近用户失败: ${error}`);
    }
  }

  /**
   * 加载附近群组
   */
  private async loadNearbyGroups(): Promise<void> {
    if (!this.mapCenter || (this.currentFilter !== 'all' && this.currentFilter !== 'groups')) {
      return;
    }

    try {
      const response = await ApiClient.searchGroupsByLocation({
        latitude: this.mapCenter.latitude,
        longitude: this.mapCenter.longitude,
        radius: 5000,
        limit: 50
      });

      if (response.code === 0 && response.data) {
        const groupMarkers: MapMarker[] = response.data.items.map((group): MapMarker => ({
          id: `group_${group.group_id}`,
          type: 'group',
          latitude: group.latitude,
          longitude: group.longitude,
          title: group.name,
          subtitle: `${group.member_count} 成员`,
          data: group
        }));
        this.markers = [...this.markers, ...groupMarkers];
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载附近群组失败: ${error}`);
    }
  }

  /**
   * 加载附近瞬间
   */
  private async loadNearbyMoments(): Promise<void> {
    if (!this.mapCenter || (this.currentFilter !== 'all' && this.currentFilter !== 'moments')) {
      return;
    }

    try {
      const response = await ApiClient.searchMomentsByLocation({
        latitude: this.mapCenter.latitude,
        longitude: this.mapCenter.longitude,
        radius: 3000,
        limit: 50
      });

      if (response.code === 0 && response.data) {
        const momentMarkers: MapMarker[] = response.data.items.map((moment): MapMarker => ({
          id: `moment_${moment.moment_id}`,
          type: 'moment',
          latitude: moment.latitude,
          longitude: moment.longitude,
          title: moment.nickname,
          subtitle: moment.description.length > 20 ? 
            moment.description.substring(0, 20) + '...' : moment.description,
          data: moment
        }));
        this.markers = [...this.markers, ...momentMarkers];
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载附近瞬间失败: ${error}`);
    }
  }

  /**
   * 切换筛选器
   */
  private async switchFilter(filter: 'all' | 'users' | 'groups' | 'moments') {
    if (this.currentFilter === filter) return;
    
    this.currentFilter = filter;
    await this.loadMapData();
  }

  /**
   * 处理标记点击
   */
  private handleMarkerClick(marker: MapMarker) {
    this.selectedMarker = marker;
    this.showMarkerDetail = true;
  }

  /**
   * 导航到详情页面
   */
  private navigateToDetail() {
    if (!this.selectedMarker) return;

    const marker = this.selectedMarker;
    this.showMarkerDetail = false;

    switch (marker.type) {
      case 'user':
        // 导航到用户资料页面或开始聊天
        SystemToast.show(`查看用户: ${marker.title}`);
        break;
      case 'group':
        // 导航到群组详情页面
        this.getUIContext().getRouter().pushUrl({
          url: `pages/${PageNames.GROUP_DETAIL}`,
          params: { groupId: (marker.data as SearchGroupByLocationResponse).group_id }
        });
        break;
      case 'moment':
        // 显示瞬间详情
        SystemToast.show(`查看瞬间: ${marker.title}`);
        break;
    }
  }

  build() {
    Column() {
      // 顶部工具栏
      Row() {
        Text('地图')
          .fontSize(20)
          .fontWeight(FontWeight.Bold)
          .layoutWeight(1)

        Button('刷新')
          .fontSize(14)
          .backgroundColor(Color.Transparent)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .onClick(() => this.loadMapData())
      }
      .width('100%')
      .height(56)
      .padding({ left: 16, right: 16 })
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      // 筛选器
      Row() {
        Button('全部')
          .fontSize(12)
          .backgroundColor(this.currentFilter === 'all' ? 
            $r('sys.color.ohos_id_color_emphasize') : Color.Transparent)
          .fontColor(this.currentFilter === 'all' ? 
            Color.White : $r('sys.color.ohos_id_color_text_primary'))
          .onClick(() => this.switchFilter('all'))

        Button('用户')
          .fontSize(12)
          .backgroundColor(this.currentFilter === 'users' ? 
            $r('sys.color.ohos_id_color_emphasize') : Color.Transparent)
          .fontColor(this.currentFilter === 'users' ? 
            Color.White : $r('sys.color.ohos_id_color_text_primary'))
          .onClick(() => this.switchFilter('users'))

        Button('群组')
          .fontSize(12)
          .backgroundColor(this.currentFilter === 'groups' ? 
            $r('sys.color.ohos_id_color_emphasize') : Color.Transparent)
          .fontColor(this.currentFilter === 'groups' ? 
            Color.White : $r('sys.color.ohos_id_color_text_primary'))
          .onClick(() => this.switchFilter('groups'))

        Button('瞬间')
          .fontSize(12)
          .backgroundColor(this.currentFilter === 'moments' ? 
            $r('sys.color.ohos_id_color_emphasize') : Color.Transparent)
          .fontColor(this.currentFilter === 'moments' ? 
            Color.White : $r('sys.color.ohos_id_color_text_primary'))
          .onClick(() => this.switchFilter('moments'))
      }
      .width('100%')
      .padding({ left: 16, right: 16, top: 8, bottom: 8 })
      .justifyContent(FlexAlign.SpaceAround)

      // 地图区域
      Stack() {
        if (this.isLoading) {
          SystemLoadingView({ message: '加载地图数据中...' })
        } else if (this.errorMessage) {
          Column() {
            Text('😕')
              .fontSize(48)
              .margin({ bottom: 16 })
            
            Text(this.errorMessage)
              .fontSize(16)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
              .margin({ bottom: 24 })
            
            Button('重试')
              .onClick(() => this.loadMapData())
          }
          .justifyContent(FlexAlign.Center)
          .alignItems(HorizontalAlign.Center)
          .width('100%')
          .height('100%')
        } else {
          // 地图占位符 - 实际项目中应该集成真实的地图组件
          Column() {
            Text('🗺️')
              .fontSize(64)
              .margin({ bottom: 16 })
            
            Text(`地图中心: ${this.mapCenter?.latitude.toFixed(4)}, ${this.mapCenter?.longitude.toFixed(4)}`)
              .fontSize(14)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
              .margin({ bottom: 8 })
            
            Text(`${this.markers.length} 个标记点`)
              .fontSize(14)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
              .margin({ bottom: 16 })

            // 标记点列表（简化显示）
            if (this.markers.length > 0) {
              List() {
                ForEach(this.markers.slice(0, 5), (marker: MapMarker) => {
                  ListItem() {
                    Row() {
                      Text(this.getMarkerIcon(marker.type))
                        .fontSize(20)
                        .margin({ right: 12 })
                      
                      Column() {
                        Text(marker.title)
                          .fontSize(14)
                          .fontWeight(FontWeight.Medium)
                          .maxLines(1)
                          .textOverflow({ overflow: TextOverflow.Ellipsis })
                        
                        if (marker.subtitle) {
                          Text(marker.subtitle)
                            .fontSize(12)
                            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                            .maxLines(1)
                            .textOverflow({ overflow: TextOverflow.Ellipsis })
                        }
                      }
                      .alignItems(HorizontalAlign.Start)
                      .layoutWeight(1)
                    }
                    .width('100%')
                    .padding(12)
                    .backgroundColor($r('sys.color.ohos_id_color_background_transparent'))
                    .borderRadius(8)
                    .onClick(() => this.handleMarkerClick(marker))
                  }
                })
              }
              .width('90%')
              .height(200)
            }
          }
          .justifyContent(FlexAlign.Center)
          .alignItems(HorizontalAlign.Center)
          .width('100%')
          .height('100%')
          .backgroundColor($r('sys.color.ohos_id_color_background_transparent'))
        }
      }
      .layoutWeight(1)
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .bindSheet($$this.showMarkerDetail, this.buildMarkerDetailSheet(), {
      height: 300,
      showClose: true,
      dragBar: true
    })
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

  /**
   * 构建标记类型相关内容
   */
  @Builder
  buildMarkerTypeContent() {
    if (this.selectedMarker) {
      if (this.selectedMarker.type === 'group') {
        Column() {
          Text(`成员数: ${(this.selectedMarker.data as SearchGroupByLocationResponse).member_count}`)
            .fontSize(14)
            .margin({ bottom: 8 })
          
          if ((this.selectedMarker.data as SearchGroupByLocationResponse).description) {
            Text((this.selectedMarker.data as SearchGroupByLocationResponse).description!)
              .fontSize(14)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
              .margin({ bottom: 12 })
          }
        }
        .alignItems(HorizontalAlign.Start)
      } else if (this.selectedMarker.type === 'moment') {
        Column() {
          Text(`点赞: ${(this.selectedMarker.data as SearchMomentsByLocationResponse).likes_count} | 评论: ${(this.selectedMarker.data as SearchMomentsByLocationResponse).comments_count}`)
            .fontSize(14)
            .margin({ bottom: 8 })
          
          Text((this.selectedMarker.data as SearchMomentsByLocationResponse).location_name)
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .margin({ bottom: 12 })
        }
        .alignItems(HorizontalAlign.Start)
      }
    }
  }

  /**
   * 构建标记详情弹窗
   */
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
          .fontSize(18)
          .fontWeight(FontWeight.Bold)
          .layoutWeight(1)
      }
      .width('100%')
      .padding(16)

      // 详情内容
      Column() {
        if (this.selectedMarker.subtitle) {
          Text(this.selectedMarker.subtitle)
            .fontSize(14)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
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
          .layoutWeight(1)
          .onClick(() => this.navigateToDetail())
        
        Button('取消')
          .backgroundColor(Color.Transparent)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
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


