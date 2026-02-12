/**
 * HarmonyOS原生地图组件
 * 集成HarmonyOS Map Kit，支持地图显示、标记、交互
 * 注意：使用Map Kit需要在AppGallery Connect中配置指纹，并在module.json5中配置client_id
 */

import { map, mapCommon, MapComponent } from '@kit.MapKit';
import { AsyncCallback } from '@kit.BasicServicesKit';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { LocationInfo } from '../core/location/LocationUtil';
import { DesignConstants } from '../constants/DesignConstants';

const DOMAIN_NUMBER: number = 0xFF11;
const TAG: string = 'NativeMapComponent';

export interface MapMarker {
  id: string;
  latitude: number;
  longitude: number;
  title?: string;
  snippet?: string;
  icon?: string; // 暂不支持自定义图标路径，仅作为预留
  type?: 'user' | 'group' | 'moment' | 'cluster';
}

export interface MapRegion {
  latitude: number;
  longitude: number;
  latitudeDelta: number;
  longitudeDelta: number;
}

export interface MapCoordinate {
  latitude: number;
  longitude: number;
}

/** 摄像头信息（含可选 zoom） */
export interface MapCameraPosition extends MapCoordinate {
  zoom?: number;
}

/** 地图控制器操作接口，供父组件调用缩放等控制 */
export interface NativeMapActions {
  zoomIn: () => void;
  zoomOut: () => void;
}

@Component
export struct NativeMapComponent {
  @Watch('onRegionChangeProp') @Prop mapRegion: MapRegion = {
    latitude: 39.9042,
    longitude: 116.4074,
    latitudeDelta: 0.01,
    longitudeDelta: 0.01
  };
  @Watch('onMarkersChange') @Prop markers: MapMarker[] = [];
  @Prop showUserLocation: boolean = true;
  @Prop showCompass: boolean = true;
  @Prop showScale: boolean = true;
  @Prop showZoomControls: boolean = false;
  /** 是否显示 Map Kit 内置的「我的位置」按钮 */
  @Prop showMyLocationControls: boolean = true;
  
  // 回调函数
  onMapReady?: () => void;
  onMarkerClick?: (marker: MapMarker) => void;
  onMapClick?: (coordinate: MapCoordinate) => void;
  /** 双击地图回调 */
  onMapDoubleClick?: (coordinate: MapCoordinate) => void;
  /** 相机停止移动回调（用户停止拖拽/缩放后触发） */
  onCameraIdle?: (camera: MapCameraPosition) => void;
  /** 地图控制器就绪回调，用于父组件获取缩放等操作能力 */
  onActionsReady?: (actions: NativeMapActions) => void;
  
  // 地图控制器
  private mapController: map.MapComponentController | null = null;
  private mapCallback?: AsyncCallback<map.MapComponentController>;
  private createdMarkers: Map<string, map.Marker> = new Map();
  // 双击检测
  private lastClickTime: number = 0;

  // 内部状态
  @State isMapReady: boolean = false;
  @State mapError: string = '';

  aboutToAppear() {
    // 初始化地图回调
    this.mapCallback = async (err, mapController) => {
      if (!err && mapController) {
        this.mapController = mapController;
        this.isMapReady = true;
        this.mapError = '';
        hilog.info(DOMAIN_NUMBER, TAG, 'MapComponentController 初始化成功');
        
        // 初始化地图设置
        this.setupMap();
        
        // 触发回调
        if (this.onMapReady) {
          this.onMapReady();
        }
      } else {
        const errMsg = JSON.stringify(err);
        hilog.error(DOMAIN_NUMBER, TAG, `MapComponentController 初始化失败: ${errMsg}`);
        this.mapError = '地图加载失败，请检查 AppGallery Connect 配置';
      }
    };
  }
  
  /**
   * 监听标记变化并更新
   */
  onMarkersChange() {
    if (this.isMapReady && this.mapController) {
      this.updateMarkers();
    }
  }

  /**
   * 监听区域变化
   */
  onRegionChangeProp() {
    if (this.isMapReady && this.mapController) {
      this.moveCamera(this.mapRegion.latitude, this.mapRegion.longitude);
    }
  }

  // 注意：Map Kit 在 MapComponent 从组件树移除时自动清理内部事件监听器，无需手动 off

  /** 放大地图（zoom + 1） */
  private doZoomIn(): void {
    if (!this.mapController) return;
    try {
      const pos = this.mapController.getCameraPosition();
      const newZoom = Math.min(20, pos.zoom + 1);
      const cameraUpdate = map.newCameraPosition({
        target: pos.target,
        zoom: newZoom,
        tilt: pos.tilt,
        bearing: pos.bearing
      });
      this.mapController.animateCamera(cameraUpdate, 300);
    } catch (e) {
      hilog.warn(DOMAIN_NUMBER, TAG, `放大地图失败: ${JSON.stringify(e)}`);
    }
  }

  /** 缩小地图（zoom - 1） */
  private doZoomOut(): void {
    if (!this.mapController) return;
    try {
      const pos = this.mapController.getCameraPosition();
      const newZoom = Math.max(3, pos.zoom - 1);
      const cameraUpdate = map.newCameraPosition({
        target: pos.target,
        zoom: newZoom,
        tilt: pos.tilt,
        bearing: pos.bearing
      });
      this.mapController.animateCamera(cameraUpdate, 300);
    } catch (e) {
      hilog.warn(DOMAIN_NUMBER, TAG, `缩小地图失败: ${JSON.stringify(e)}`);
    }
  }

  private setupMap() {
    if (!this.mapController) return;
    
    // 设置定位启用
    this.mapController.setMyLocationEnabled(this.showUserLocation);
    this.mapController.setMyLocationControlsEnabled(this.showMyLocationControls);
    
    // 设置比例尺
    this.mapController.setScaleControlsEnabled(this.showScale);
    
    // 设置缩放按钮
    this.mapController.setZoomControlsEnabled(this.showZoomControls);

    // 通知父组件地图控制能力已就绪
    if (this.onActionsReady) {
      const actions: NativeMapActions = {
        zoomIn: (): void => this.doZoomIn(),
        zoomOut: (): void => this.doZoomOut()
      };
      this.onActionsReady(actions);
    }

    // 监听地图点击（含双击检测）
    this.mapController.on("mapClick", (position: mapCommon.LatLng) => {
      const now = Date.now();
      const coord: MapCoordinate = { latitude: position.latitude, longitude: position.longitude };

      if (now - this.lastClickTime < 400) {
        // 双击：两次点击间隔 < 400ms
        this.lastClickTime = 0;
        if (this.onMapDoubleClick) {
          this.onMapDoubleClick(coord);
        }
      } else {
        this.lastClickTime = now;
        if (this.onMapClick) {
          this.onMapClick(coord);
        }
      }
    });

    // 监听相机空闲（用户停止拖拽/缩放后触发）
    try {
      this.mapController.on("cameraIdle", () => {
        if (this.onCameraIdle && this.mapController) {
          try {
            const cameraPosition = this.mapController.getCameraPosition();
            if (cameraPosition && cameraPosition.target) {
              this.onCameraIdle({
                latitude: cameraPosition.target.latitude,
                longitude: cameraPosition.target.longitude,
                zoom: cameraPosition.zoom
              });
            }
          } catch (e) {
            hilog.warn(DOMAIN_NUMBER, TAG, `cameraIdle 获取相机位置失败: ${JSON.stringify(e)}`);
          }
        }
      });
    } catch (e) {
      hilog.warn(DOMAIN_NUMBER, TAG, `注册 cameraIdle 事件失败: ${JSON.stringify(e)}`);
    }

    // 监听标记点击
    this.mapController.on("markerClick", (marker: map.Marker) => {
      const markerId = marker.getId();
      // 在 markers 中查找
      const target = this.markers.find(m => this.createdMarkers.get(m.id)?.getId() === markerId);
      if (target && this.onMarkerClick) {
        this.onMarkerClick(target);
      }
    });

    // 初始渲染标记
    this.updateMarkers();
  }

  private async updateMarkers() {
    if (!this.mapController) return;
    
    // 简单实现：清除所有旧标记，添加新标记
    // 生产环境应做 Diff 更新以提升性能
    
    // 1. 清除旧标记
    try {
        // 由于没有直接 clearAll 接口，或者需要遍历清除
        // 这里简化处理，如果 Map Kit 支持 clear 则调用，否则遍历
        this.mapController.clear(); 
        this.createdMarkers.clear();
    } catch(e) {
        hilog.warn(DOMAIN_NUMBER, TAG, `清除标记失败: ${JSON.stringify(e)}`);
    }

    // 2. 添加新标记
    for (const markerData of this.markers) {
      try {
        const markerOptions: mapCommon.MarkerOptions = {
          position: {
            latitude: markerData.latitude,
            longitude: markerData.longitude
          },
          title: markerData.title,
          snippet: markerData.snippet,
          clickable: true,
          // 可以根据 markerData.type 设置不同的颜色或图标
          // icon: ... 
        };
        
        const marker = await this.mapController.addMarker(markerOptions);
        this.createdMarkers.set(markerData.id, marker);
      } catch (error) {
        hilog.error(DOMAIN_NUMBER, TAG, `添加标记失败 ${markerData.id}: ${JSON.stringify(error)}`);
      }
    }
  }

  /**
   * 移动相机到指定位置
   */
  public moveCamera(latitude: number, longitude: number, zoom: number = 15) {
    if (this.mapController) {
      const cameraUpdate = map.newCameraPosition({
        target: {
          latitude: latitude,
          longitude: longitude
        },
        zoom: zoom,
        tilt: 0,
        bearing: 0
      });
      this.mapController.animateCamera(cameraUpdate, 1000); // 1000ms 动画
    }
  }

  build() {
    Stack() {
      if (this.mapError.length > 0) {
        // 地图加载失败提示
        Column({ space: 12 }) {
          Text() {
            SymbolSpan($r('sys.symbol.map'))
              .fontSize(48)
              .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
          }

          Text('地图服务不可用')
            .fontSize(16)
            .fontWeight(FontWeight.Medium)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))

          Text(this.mapError)
            .fontSize(13)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .textAlign(TextAlign.Center)

          Text('请在 module.json5 中配置 client_id\n并在 AppGallery Connect 中启用 Map Kit 服务')
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .textAlign(TextAlign.Center)
            .lineHeight(18)
        }
        .width('100%')
        .height('100%')
        .justifyContent(FlexAlign.Center)
        .padding(32)
        .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
      } else if (this.mapCallback) {
        MapComponent({ 
          mapOptions: {
            position: {
              target: {
                latitude: this.mapRegion.latitude,
                longitude: this.mapRegion.longitude
              },
              zoom: 15
            }
          }, 
          mapCallback: this.mapCallback 
        })
        .width('100%')
        .height('100%')
      } else {
        Column() {
          LoadingProgress()
            .width(40)
            .height(40)
          Text('加载地图中...')
            .fontSize(14)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .margin({ top: 12 })
        }
        .width('100%')
        .height('100%')
        .justifyContent(FlexAlign.Center)
      }
    }
    .width('100%')
    .height('100%')
  }
}
