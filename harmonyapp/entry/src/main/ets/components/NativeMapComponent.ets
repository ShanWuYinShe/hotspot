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
  type?: 'user' | 'group' | 'moment';
}

export interface MapRegion {
  latitude: number;
  longitude: number;
  latitudeDelta: number;
  longitudeDelta: number;
}

@Component
export struct NativeMapComponent {
  @Prop mapRegion: MapRegion = {
    latitude: 39.9042,
    longitude: 116.4074,
    latitudeDelta: 0.01,
    longitudeDelta: 0.01
  };
  @Prop markers: MapMarker[] = [];
  @Prop showUserLocation: boolean = true;
  @Prop showCompass: boolean = true;
  @Prop showScale: boolean = true;
  @Prop showZoomControls: boolean = false;
  
  // 回调函数
  onMapReady?: () => void;
  onMarkerClick?: (marker: MapMarker) => void;
  onMapClick?: (coordinate: { latitude: number; longitude: number }) => void;
  
  // 地图控制器
  private mapController: map.MapComponentController | null = null;
  private mapCallback?: AsyncCallback<map.MapComponentController>;
  private createdMarkers: Map<string, map.Marker> = new Map();

  // 内部状态
  @State isMapReady: boolean = false;

  aboutToAppear() {
    // 初始化地图回调
    this.mapCallback = async (err, mapController) => {
      if (!err && mapController) {
        this.mapController = mapController;
        this.isMapReady = true;
        hilog.info(DOMAIN_NUMBER, TAG, 'MapComponentController 初始化成功');
        
        // 初始化地图设置
        this.setupMap();
        
        // 触发回调
        if (this.onMapReady) {
          this.onMapReady();
        }
      } else {
        hilog.error(DOMAIN_NUMBER, TAG, `MapComponentController 初始化失败: ${JSON.stringify(err)}`);
      }
    };
  }
  
  /**
   * 监听标记变化并更新
   */
  @Watch('onMarkersChange')
  onMarkersChange() {
    if (this.isMapReady && this.mapController) {
      this.updateMarkers();
    }
  }

  /**
   * 监听区域变化
   */
  @Watch('onRegionChange')
  onRegionChangeProp() {
    if (this.isMapReady && this.mapController) {
       // 可以在这里平滑移动到新区域，但为了避免冲突，通常由外部控制是否移动
    }
  }

  private setupMap() {
    if (!this.mapController) return;
    
    // 设置定位启用
    this.mapController.setMyLocationEnabled(this.showUserLocation);
    this.mapController.setMyLocationControlsEnabled(true);
    
    // 设置指南针
    this.mapController.setCompassEnabled(this.showCompass);
    
    // 设置比例尺
    this.mapController.setScaleControlsEnabled(this.showScale);
    
    // 设置缩放按钮
    this.mapController.setZoomControlsEnabled(this.showZoomControls);

    // 监听地图点击
    this.mapController.on("mapClick", (position: mapCommon.LatLng) => {
      if (this.onMapClick) {
        this.onMapClick({ latitude: position.latitude, longitude: position.longitude });
      }
    });

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
      if (this.mapCallback) {
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
        LoadingProgress()
      }
    }
    .width('100%')
    .height('100%')
  }
}
