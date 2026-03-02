/**
 * HarmonyOS原生地图组件
 * 集成HarmonyOS Map Kit，支持地图显示、标记、交互
 * 前置条件（DevEco Studio）：
 * - build-profile.json5 的 capabilities 中声明 Map Kit
 * - DevEco Studio 中配置项目签名，并在 AppGallery Connect 注册证书指纹、启用 Map Kit 服务
 */

import { map, mapCommon, MapComponent } from '@kit.MapKit';
import { AsyncCallback } from '@kit.BasicServicesKit';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { image } from '@kit.ImageKit';


const DOMAIN_NUMBER: number = 0xFF11;
const TAG: string = 'NativeMapComponent';

type MapMarkerType = 'user' | 'group' | 'moment' | 'cluster';
type MapMarkerIconKey = MapMarkerType | 'center_pin';

export interface MapMarker {
  id: string;
  latitude: number;
  longitude: number;
  title?: string;
  snippet?: string;
  iconKey?: MapMarkerIconKey;
  type?: MapMarkerType;
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

/** 地图控制器操作接口，供父组件调用缩放、生命周期等控制 */
export interface NativeMapActions {
  zoomIn: () => void;
  zoomOut: () => void;
  /** 移动相机到指定坐标（保持当前缩放级别） */
  moveCamera: (latitude: number, longitude: number) => void;
  /** 页面重新可见时调用（配合 NavDestination.onShown） */
  show: () => void;
  /** 页面不可见时调用（配合 NavDestination.onHidden） */
  hide: () => void;
}

interface LongPressCapable {
  on(eventName: string, handler: (position: mapCommon.LatLng) => void): void;
}

interface CameraMoveCapable {
  on(eventName: string, handler: () => void): void;
}

@Component
export struct NativeMapComponent {
  @Watch('onDarkModeChange') @StorageProp('isDarkMode') isDarkMode: boolean = false;
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
  /** 是否在地图中心显示选点图钉 marker（由地图引擎管理坐标映射，缩放/布局变化时位置始终精确） */
  @Prop showCenterPin: boolean = false;
  /** 地图边界内边距（像素），用于把相机中心和内置控件推离被遮挡的区域 */
  @Watch('onContentPaddingChange') @Prop contentPaddingBottom: number = 0;
  @Watch('onContentPaddingChange') @Prop contentPaddingTop: number = 0;
  /** 地图厂商 Logo 底部偏移（像素），确保 logo 不被底部面板遮挡（合规要求） */
  @Watch('onLogoPaddingChange') @Prop logoPaddingBottom: number = 0;
  onMapLongPress?: (coordinate: MapCoordinate) => void;
  
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
  private centerPinMarker: map.Marker | null = null;
  private markerIconCache: Map<string, image.PixelMap> = new Map();
  // 双击检测
  private lastClickTime: number = 0;
  // 防止 @Watch 在父组件重渲染时误触发
  private lastRegionLat: number = NaN;
  private lastRegionLng: number = NaN;
  private lastPaddingTop: number = NaN;
  private lastPaddingBottom: number = NaN;
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
        await this.setupMap();
        
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
  
  aboutToDisappear() {
    for (const pixelMap of this.markerIconCache.values()) {
      pixelMap.release();
    }
    this.markerIconCache.clear();
  }

  private async initMarkerIcons(): Promise<void> {
    const iconKeys: string[] = ['user', 'group', 'moment', 'cluster', 'center_pin'];
    const iconFiles: string[] = [
      'marker_user.svg', 'marker_group.svg', 'marker_moment.svg',
      'marker_cluster.svg', 'marker_center_pin.svg',
    ];
    const hostContext = this.getUIContext().getHostContext();
    if (!hostContext) {
      hilog.error(DOMAIN_NUMBER, TAG, 'getHostContext 返回 null，无法加载 marker 图标');
      return;
    }
    const resourceMgr = hostContext.resourceManager;
    for (let i = 0; i < iconKeys.length; i++) {
      try {
        const rawData = await resourceMgr.getRawFileContent(iconFiles[i]);
        const imgSource = image.createImageSource(rawData.buffer);
        const pixelMap = await imgSource.createPixelMap();
        imgSource.release();
        this.markerIconCache.set(iconKeys[i], pixelMap);
      } catch (e) {
        hilog.error(DOMAIN_NUMBER, TAG, `加载 marker 图标失败 ${iconFiles[i]}: ${JSON.stringify(e)}`);
      }
    }
    hilog.info(DOMAIN_NUMBER, TAG, `marker 图标缓存完成，共 ${this.markerIconCache.size} 个`);
  }

  onContentPaddingChange() {
    if (this.contentPaddingTop === this.lastPaddingTop &&
      this.contentPaddingBottom === this.lastPaddingBottom) {
      return;
    }
    this.lastPaddingTop = this.contentPaddingTop;
    this.lastPaddingBottom = this.contentPaddingBottom;
    this.applyContentPadding();
  }

  onLogoPaddingChange() {
    this.applyLogoPadding();
  }

  private applyContentPadding() {
    if (this.isMapReady && this.mapController) {
      try {
        this.mapController.setPadding({
          left: 0,
          top: this.contentPaddingTop,
          right: 0,
          bottom: this.contentPaddingBottom
        });
      } catch (e) {
        hilog.warn(DOMAIN_NUMBER, TAG, `设置地图内边距失败: ${JSON.stringify(e)}`);
      }
    }
  }

  private applyLogoPadding() {
    if (this.isMapReady && this.mapController) {
      try {
        this.mapController.setLogoPadding({
          left: 0,
          top: 0,
          right: 0,
          bottom: this.logoPaddingBottom
        });
      } catch (e) {
        hilog.warn(DOMAIN_NUMBER, TAG, `设置地图 Logo 位置失败: ${JSON.stringify(e)}`);
      }
    }
  }

  /**
   * 监听深色模式变化，切换地图夜间样式
   */
  onDarkModeChange() {
    if (this.isMapReady && this.mapController) {
      try {
        this.mapController.setDayNightMode(
          this.isDarkMode ? mapCommon.DayNightMode.NIGHT : mapCommon.DayNightMode.DAY
        );
      } catch (e) {
        hilog.warn(DOMAIN_NUMBER, TAG, `切换地图夜间模式失败: ${JSON.stringify(e)}`);
      }
    }
  }

  /**
   * 监听标记变化并更新
   */
  onMarkersChange() {
    if (this.isMapReady && this.mapController) {
      this.updateMarkers();
    }
  }

  onRegionChangeProp() {
    if (!this.isMapReady || !this.mapController) {
      return;
    }
    if (this.mapRegion.latitude === this.lastRegionLat &&
      this.mapRegion.longitude === this.lastRegionLng) {
      return;
    }
    this.lastRegionLat = this.mapRegion.latitude;
    this.lastRegionLng = this.mapRegion.longitude;
    try {
      const pos = this.mapController.getCameraPosition();
      this.moveCamera(this.mapRegion.latitude, this.mapRegion.longitude, pos?.zoom ?? 15);
    } catch (_) {
      this.moveCamera(this.mapRegion.latitude, this.mapRegion.longitude);
    }
  }

  // 注意：Map Kit 在 MapComponent 从组件树移除时自动清理内部事件监听器，无需手动 off

  /** 放大地图（zoom + 1） */
  private doZoomIn(): void {
    if (!this.mapController) return;
    try {
      this.mapController.animateCamera(map.zoomIn(), 300);
    } catch (e) {
      hilog.warn(DOMAIN_NUMBER, TAG, `放大地图失败: ${JSON.stringify(e)}`);
    }
  }

  /** 缩小地图（zoom - 1） */
  private doZoomOut(): void {
    if (!this.mapController) return;
    try {
      this.mapController.animateCamera(map.zoomOut(), 300);
    } catch (e) {
      hilog.warn(DOMAIN_NUMBER, TAG, `缩小地图失败: ${JSON.stringify(e)}`);
    }
  }

  private registerLongPressHandler() {
    if (!this.mapController || !this.onMapLongPress) {
      return;
    }
    const controller = this.mapController as object as LongPressCapable;
    const candidateEvents: string[] = ['longPress', 'mapLongPress', 'longClick'];
    for (const name of candidateEvents) {
      try {
        controller.on(name, (position: mapCommon.LatLng) => {
          if (!this.onMapLongPress || !position) {
            return;
          }
          this.onMapLongPress({ latitude: position.latitude, longitude: position.longitude });
        });
        return;
      } catch (_) {
        // 忽略不支持的事件名称
      }
    }
  }

  private async setupMap() {
    if (!this.mapController) return;
    
    // 根据当前深浅模式设置地图夜间样式
    try {
      this.mapController.setDayNightMode(
        this.isDarkMode ? mapCommon.DayNightMode.NIGHT : mapCommon.DayNightMode.DAY
      );
    } catch (e) {
      hilog.warn(DOMAIN_NUMBER, TAG, `初始化地图夜间模式失败: ${JSON.stringify(e)}`);
    }

    // 设置定位启用
    this.mapController.setMyLocationEnabled(this.showUserLocation);
    this.mapController.setMyLocationControlsEnabled(this.showMyLocationControls);
    
    // 设置比例尺
    this.mapController.setScaleControlsEnabled(this.showScale);
    
    // 设置缩放按钮
    this.mapController.setZoomControlsEnabled(this.showZoomControls);

    // 设置指南针
    this.mapController.setCompassControlsEnabled(this.showCompass);

    // 社交地图场景：禁用旋转和倾斜手势，避免用户误操作导致地图方向混乱
    this.mapController.setRotateGesturesEnabled(false);
    this.mapController.setTiltGesturesEnabled(false);

    // 限制最小缩放级别（级别 3 ≈ 大洲级别，社交场景无需更小）
    this.mapController.setMinZoom(3);

    // 设置地图内边距，让控件避开覆盖层
    this.lastPaddingTop = this.contentPaddingTop;
    this.lastPaddingBottom = this.contentPaddingBottom;
    this.applyContentPadding();
    this.applyLogoPadding();

    // 记录初始 region，防止 @Watch 在后续父组件重渲染时误触发 moveCamera
    this.lastRegionLat = this.mapRegion.latitude;
    this.lastRegionLng = this.mapRegion.longitude;

    // 通知父组件地图控制能力已就绪
    if (this.onActionsReady) {
      const ctrl = this.mapController;
      const self = this;
      const actions: NativeMapActions = {
        zoomIn: (): void => this.doZoomIn(),
        zoomOut: (): void => this.doZoomOut(),
        moveCamera: (lat: number, lng: number): void => {
          if (ctrl) {
            try {
              const pos = ctrl.getCameraPosition();
              const cameraUpdate = map.newCameraPosition({
                target: { latitude: lat, longitude: lng },
                zoom: pos?.zoom ?? 15,
                tilt: 0,
                bearing: 0
              });
              ctrl.animateCamera(cameraUpdate, 500);
            } catch (e) {
              self.moveCamera(lat, lng);
            }
          }
        },
        show: (): void => { if (ctrl) { ctrl.show(); } },
        hide: (): void => { if (ctrl) { ctrl.hide(); } }
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

    this.registerLongPressHandler();

    // 监听相机空闲（用户停止拖拽/缩放后触发）
    try {
      this.mapController.on("cameraIdle", () => {
        if (!this.mapController) return;
        try {
          this.updateCenterPinPosition();
          const cameraPosition = this.mapController.getCameraPosition();
          if (cameraPosition?.target) {
            if (this.onCameraIdle) {
              this.onCameraIdle({
                latitude: cameraPosition.target.latitude,
                longitude: cameraPosition.target.longitude,
                zoom: cameraPosition.zoom
              });
            }
          }
        } catch (e) {
          hilog.warn(DOMAIN_NUMBER, TAG, `cameraIdle 处理失败: ${JSON.stringify(e)}`);
        }
      });
    } catch (e) {
      hilog.warn(DOMAIN_NUMBER, TAG, `注册 cameraIdle 事件失败: ${JSON.stringify(e)}`);
    }

    // 注册相机移动事件，使中心图钉实时跟随（cameraIdle 为保底更新）
    this.registerCameraMoveHandler();

    // 监听标记点击
    this.mapController.on("markerClick", (marker: map.Marker) => {
      const markerId = marker.getId();
      // 在 markers 中查找
      const target = this.markers.find(m => this.createdMarkers.get(m.id)?.getId() === markerId);
      if (target && this.onMarkerClick) {
        this.onMarkerClick(target);
      }
    });

    // 预加载 marker 图标为 PixelMap（绕过 $rawfile Resource 在 Map Kit 中的兼容问题）
    await this.initMarkerIcons();

    // 初始渲染标记
    this.updateMarkers();
  }

  private async updateMarkers() {
    if (!this.mapController) return;

    try {
      this.mapController.clear();
      this.createdMarkers.clear();
      this.centerPinMarker = null;
    } catch (e) {
      hilog.warn(DOMAIN_NUMBER, TAG, `清除标记失败: ${JSON.stringify(e)}`);
    }

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
          icon: this.getMarkerIconForMarker(markerData),
          anchorU: 0.5,
          anchorV: 1.0
        };
        const marker = await this.mapController.addMarker(markerOptions);
        this.createdMarkers.set(markerData.id, marker);
      } catch (error) {
        hilog.error(DOMAIN_NUMBER, TAG, `添加标记失败 ${markerData.id}: ${JSON.stringify(error)}`);
      }
    }

    if (this.showCenterPin) {
      await this.createCenterPinMarker();
    }
  }

  private getMarkerIconForMarker(markerData: MapMarker): image.PixelMap | Resource {
    const key = markerData.iconKey ?? markerData.type ?? 'user';
    const cached = this.markerIconCache.get(key);
    if (cached) {
      return cached;
    }
    hilog.warn(DOMAIN_NUMBER, TAG, `PixelMap 缓存未命中 key=${key}，降级为 $rawfile`);
    switch (key) {
      case 'group': return $rawfile('marker_group.svg');
      case 'moment': return $rawfile('marker_moment.svg');
      case 'cluster': return $rawfile('marker_cluster.svg');
      case 'center_pin': return $rawfile('marker_center_pin.svg');
      default: return $rawfile('marker_user.svg');
    }
  }

  private async createCenterPinMarker() {
    if (!this.mapController) return;
    try {
      const cameraPos = this.mapController.getCameraPosition();
      const target: mapCommon.LatLng = cameraPos?.target ??
        { latitude: this.mapRegion.latitude, longitude: this.mapRegion.longitude };
      const markerOptions: mapCommon.MarkerOptions = {
        position: target,
        clickable: false,
        icon: this.markerIconCache.get('center_pin') ?? $rawfile('marker_center_pin.svg'),
        anchorU: 0.5,
        anchorV: 1.0
      };
      this.centerPinMarker = await this.mapController.addMarker(markerOptions);
    } catch (e) {
      hilog.warn(DOMAIN_NUMBER, TAG, `创建中心图钉失败: ${JSON.stringify(e)}`);
    }
  }

  private updateCenterPinPosition() {
    if (!this.centerPinMarker || !this.mapController) return;
    try {
      const pos = this.mapController.getCameraPosition();
      if (pos?.target) {
        this.centerPinMarker.setPosition(pos.target);
      }
    } catch (_) {
      // 部分设备不支持 setPosition，忽略
    }
  }

  private registerCameraMoveHandler() {
    if (!this.mapController || !this.showCenterPin) return;
    const controller = this.mapController as object as CameraMoveCapable;
    const candidateEvents: string[] = ['cameraMove', 'cameraMoving'];
    for (const name of candidateEvents) {
      try {
        controller.on(name, () => {
          this.updateCenterPinPosition();
        });
        hilog.info(DOMAIN_NUMBER, TAG, `注册 ${name} 事件成功，中心图钉将实时跟随相机`);
        return;
      } catch (_) {
        // 忽略不支持的事件名称
      }
    }
    hilog.info(DOMAIN_NUMBER, TAG, '未找到 cameraMove 事件，中心图钉将在 cameraIdle 时更新');
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

          Text('请在 DevEco Studio 中配置项目签名\n并在 AppGallery Connect 中注册指纹、启用 Map Kit')
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .textAlign(TextAlign.Center)
            .lineHeight(18)
        }
        .width('100%')
        .height('100%')
        .justifyContent(FlexAlign.Center)
        .padding(32)
        .backgroundColor($r('app.color.bg_primary'))
      } else if (this.mapCallback) {
        MapComponent({
          mapOptions: {
            position: {
              target: {
                latitude: this.mapRegion.latitude,
                longitude: this.mapRegion.longitude
              },
              zoom: 15
            },
            minZoom: 3,
            dayNightMode: this.isDarkMode ? mapCommon.DayNightMode.NIGHT : mapCommon.DayNightMode.DAY
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
