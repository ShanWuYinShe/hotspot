import { KEY_BROWSE_ANCHOR, KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { LocationInfo, LocationUtil } from '../../../core/location/LocationUtil';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { AnchorHistoryUtil } from '../../../core/utils/AnchorHistoryUtil';
import { NativeMapComponent, MapRegion, MapCoordinate, MapMarker, MapCameraPosition } from '../../../components/NativeMapComponent';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { curves } from '@kit.ArkUI';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { ApiClient } from '../../../core/api/ApiClient';
import { MapRenderItem, MapRenderRequest, MapViewport } from '../../../core/types/api';
import { AppRouter, GroupDetailRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { CoordinateConverter } from '../../../core/geo/CoordinateConverter';

const DOMAIN_NUMBER: number = 0xFF13;
const TAG: string = 'AnchorPickPage';

interface BoundsDelta {
  latDelta: number;
  lonDelta: number;
}

@Component
export struct AnchorPickPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;

  @State mapRegion: MapRegion = {
    latitude: 35.154921,
    longitude: 112.68253,
    latitudeDelta: 0.01,
    longitudeDelta: 0.01
  };

  @State locationName: string = '';
  @State centerLat: number = 0;
  @State centerLng: number = 0;
  @State isLoading: boolean = false;
  @State isPinAnimating: boolean = false;
  @State searchText: string = '';
  @State isSearching: boolean = false;
  @State showSearchBar: boolean = false;
  @State searchResults: LocationInfo[] = [];
  @State mapMarkers: MapMarker[] = [];
  @State showMarkerDetail: boolean = false;
  @State showUsers: boolean = true;
  @State showGroups: boolean = true;
  @State showMoments: boolean = true;
  @State showClusters: boolean = true;
  @State isRenderLoading: boolean = false;
  @State renderError: string = '';
  @State mapToastMessage: string = '';

  private renderItemsMap: Map<string, MapRenderItem> = new Map();
  private selectedRenderItem: MapRenderItem | null = null;
  private resolveTimerId: number = 0;
  private isFirstCameraIdle: boolean = true;
  private skipNextResolve: boolean = false;
  private searchDebounceTimerId: number = 0;
  private mapZoom: number = 15;
  private initialised: boolean = false;
  private renderItems: MapRenderItem[] = [];
  private lastViewportSignature: string = '';

  async aboutToAppear() {
    this.initialised = false;
  }

  private initFromParams(param: Record<string, Object> | undefined): void {
    this.initLocationAsync();
  }

  private async initLocationAsync(): Promise<void> {
    if (this.initialised) {
      return;
    }

    let target: LocationInfo | null = AppStorage.get<LocationInfo | null>(KEY_BROWSE_ANCHOR) ?? null;
    if (!target && this.currentLocation) {
      target = this.currentLocation;
    }

    if (!target) {
      target = await CoordinateConverter.getConvertedLocation();
    } else {
      const converted = CoordinateConverter.convertWgs84ToGcj02(target.latitude, target.longitude);
      target = {
        latitude: converted.latitude,
        longitude: converted.longitude,
        accuracy: target.accuracy,
        altitude: target.altitude,
        speed: target.speed,
        bearing: target.bearing,
        timestamp: target.timestamp,
        locationName: target.locationName
      };
    }

    if (target) {
      this.setCenter(target.latitude, target.longitude);
      await this.resolveLocation(target.latitude, target.longitude);
    } else {
      hilog.warn(DOMAIN_NUMBER, TAG, '无法获取当前位置');
    }

    this.initialised = true;
    await this.fetchMapRenderData();
  }

  aboutToDisappear() {
    if (this.resolveTimerId > 0) {
      clearTimeout(this.resolveTimerId);
      this.resolveTimerId = 0;
    }
    if (this.searchDebounceTimerId > 0) {
      clearTimeout(this.searchDebounceTimerId);
      this.searchDebounceTimerId = 0;
    }
  }

  /**
   * 回到我的当前位置：通过 geoLocationManager 获取 WGS84 坐标后转换为 GCJ02。
   * 不依赖 Map Kit 的「我的位置」功能。
   */
  private async recenterToCurrentLocation(): Promise<void> {
    const location = await CoordinateConverter.getConvertedLocation();
    if (!location) {
      SystemToast.show('无法获取当前位置', this.getUIContext());
      return;
    }
    this.skipNextResolve = true;
    this.setCenter(location.latitude, location.longitude);
    await this.resolveLocation(location.latitude, location.longitude);
  }

  private setCenter(lat: number, lng: number) {
    this.centerLat = lat;
    this.centerLng = lng;
    this.mapRegion = { latitude: lat, longitude: lng, latitudeDelta: 0.01, longitudeDelta: 0.01 };
  }

  private onCameraIdle(camera: MapCameraPosition): void {
    if (this.isFirstCameraIdle) {
      this.isFirstCameraIdle = false;
    }

    this.centerLat = camera.latitude;
    this.centerLng = camera.longitude;
    if (camera.zoom) {
      this.mapZoom = camera.zoom;
    }

    const shouldResolve = !this.skipNextResolve;
    if (this.skipNextResolve) {
      this.skipNextResolve = false;
    }

    if (shouldResolve) {
      if (this.resolveTimerId > 0) {
        clearTimeout(this.resolveTimerId);
      }
      this.isLoading = true;
      this.resolveTimerId = setTimeout(() => {
        this.resolveLocation(camera.latitude, camera.longitude);
      }, 400);
    }

    this.fetchMapRenderData();
  }

  private onMapDoubleClick(coord: MapCoordinate): void {
    this.getUIContext().animateTo({ duration: 200, curve: Curve.FastOutSlowIn }, () => {
      this.isPinAnimating = true;
    });
    setTimeout(() => {
      this.getUIContext().animateTo({ duration: 500, curve: curves.springMotion(0.6, 0.8) }, () => {
        this.isPinAnimating = false;
      });
    }, 200);
    this.setCenter(coord.latitude, coord.longitude);
  }

  private async resolveLocation(lat: number, lng: number): Promise<void> {
    this.isLoading = true;
    try {
      const addr = await LoadingUtil.withMinimumTime(() => LocationUtil.reverseGeocode(lat, lng), LoadingUtil.DEFAULT_ACTION_MS);
      this.locationName = addr?.fullAddress || '未知位置';
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, `解析位置失败: ${JSON.stringify(error)}`);
      if (this.locationName.length === 0) {
        this.locationName = '未知位置';
      }
    } finally {
      this.isLoading = false;
    }
  }

  private async doSearch(): Promise<void> {
    const keyword = this.searchText.trim();
    if (keyword.length === 0) return;

    this.isSearching = true;
    this.searchResults = [];
    try {
      const results = await LoadingUtil.withMinimumTime(() => LocationUtil.geocodeMultiple(keyword, 10), LoadingUtil.DEFAULT_ACTION_MS);
      if (results.length > 0) {
        this.searchResults = this.sortByProximity(results);
        this.getUIContext().getFocusController().clearFocus();
      } else {
        SystemToast.show('未找到该地点，请换个关键词试试', this.getUIContext());
      }
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, `搜索地点失败: ${JSON.stringify(error)}`);
      SystemToast.show('搜索失败，请稍后重试', this.getUIContext());
    } finally {
      this.isSearching = false;
    }
  }

  private sortByProximity(results: LocationInfo[]): LocationInfo[] {
    const refLat = this.centerLat || (this.currentLocation?.latitude ?? 0);
    const refLng = this.centerLng || (this.currentLocation?.longitude ?? 0);
    return results.sort((a, b) => {
      const dLatA = a.latitude - refLat;
      const dLngA = a.longitude - refLng;
      const dLatB = b.latitude - refLat;
      const dLngB = b.longitude - refLng;
      return (dLatA * dLatA + dLngA * dLngA) - (dLatB * dLatB + dLngB * dLngB);
    });
  }

  private selectSearchResult(result: LocationInfo): void {
    this.skipNextResolve = true;
    this.setCenter(result.latitude, result.longitude);
    this.locationName = result.locationName || '';
    this.centerLat = result.latitude;
    this.centerLng = result.longitude;
    this.getUIContext().animateTo({ duration: 250, curve: Curve.FastOutSlowIn }, () => {
      this.searchResults = [];
      this.showSearchBar = false;
      this.searchText = '';
    });
  }

  private confirm() {
    if (this.locationName.length === 0 || this.isLoading) {
      SystemToast.show('请等待位置识别完成', this.getUIContext());
      return;
    }
    const location: LocationInfo = {
      latitude: this.centerLat,
      longitude: this.centerLng,
      locationName: this.locationName
    };
    AppStorage.setOrCreate<LocationInfo | null>(KEY_BROWSE_ANCHOR, location);
    AnchorHistoryUtil.addToHistory(this.centerLat, this.centerLng, this.locationName);
    AppRouter.pop();
  }

  private async fetchMapRenderData(): Promise<void> {
    if (!this.initialised) {
      return;
    }
    const viewport = this.buildViewport();
    const signature = `${viewport.bounds.north.toFixed(6)}|${viewport.bounds.south.toFixed(6)}|${viewport.bounds.east.toFixed(6)}|${viewport.bounds.west.toFixed(6)}|${viewport.zoom.toFixed(2)}`;
    if (signature === this.lastViewportSignature && this.renderItems.length > 0) {
      this.updateMarkers();
      return;
    }
    this.lastViewportSignature = signature;
    this.isRenderLoading = true;
    this.renderError = '';
    try {
      const request: MapRenderRequest = { viewport };
      const response = await ApiClient.renderMap(request);
      if (response.code === 0 && response.data) {
        this.renderItems = response.data.items ?? [];
        this.mapToastMessage = response.data.toast_message ?? '';
        this.updateMarkers();
      } else {
        this.renderError = '地图数据加载失败';
      }
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, `render map failed: ${JSON.stringify(error)}`);
      this.renderError = '地图数据加载失败';
    } finally {
      this.isRenderLoading = false;
    }
  }

  private buildViewport(): MapViewport {
    const lat = this.centerLat || this.mapRegion.latitude;
    const lng = this.centerLng || this.mapRegion.longitude;
    const zoom = Math.max(1, Math.min(20, this.mapZoom));
    const delta: BoundsDelta = this.computeBoundsDelta(zoom);
    return {
      bounds: {
        north: Math.min(90, lat + delta.latDelta / 2),
        south: Math.max(-90, lat - delta.latDelta / 2),
        east: Math.min(180, lng + delta.lonDelta / 2),
        west: Math.max(-180, lng - delta.lonDelta / 2)
      },
      zoom
    };
  }

  private computeBoundsDelta(zoom: number): BoundsDelta {
    const latDelta = 180 / Math.pow(2, zoom);
    const lonDelta = 360 / Math.pow(2, zoom);
    return { latDelta, lonDelta };
  }

  private updateMarkers(): void {
    this.renderItemsMap.clear();
    const nextMarkers: MapMarker[] = [];
    for (const item of this.renderItems) {
      this.renderItemsMap.set(item.id, item);
      if (!this.shouldDisplayItem(item)) {
        continue;
      }
      nextMarkers.push({
        id: item.id,
        latitude: item.location.latitude,
        longitude: item.location.longitude,
        title: this.getMarkerTitle(item),
        snippet: this.getMarkerSnippet(item),
        type: item.type === 'cluster' ? 'cluster' : item.type
      });
    }
    this.mapMarkers = nextMarkers;
  }

  private shouldDisplayItem(item: MapRenderItem): boolean {
    switch (item.type) {
      case 'user':
        return this.showUsers;
      case 'group':
        return this.showGroups;
      case 'moment':
        return this.showMoments;
      case 'cluster':
        return this.showClusters;
      default:
        return true;
    }
  }

  private getMarkerTitle(item: MapRenderItem): string {
    if (item.type === 'cluster' && item.cluster) {
      return item.cluster.label || `${item.cluster.count} 个内容`;
    }
    return item.entity?.title || '';
  }

  private getMarkerSnippet(item: MapRenderItem): string {
    if (item.type === 'cluster' && item.cluster) {
      return `共 ${item.cluster.count} 项`;
    }
    return item.entity?.subtitle || '';
  }

  private onMarkerItemClick(marker: MapMarker): void {
    const item = this.renderItemsMap.get(marker.id);
    if (!item) {
      return;
    }
    if (item.type === 'group' && item.entity) {
      const param = new GroupDetailRouteParam();
      param.groupId = item.entity.payload_id;
      AppRouter.push(PageNames.GROUP_DETAIL, param as Object);
    }
  }

  build() {
    NavDestination() {
      Column() {
        // ===== 顶部导航栏 =====
        Row() {
          Column() {
            Text() {
              SymbolSpan($r(AppIcon.BACK))
                .fontSize(22)
                .fontColor([$r('sys.color.ohos_id_color_text_primary')])
            }
          }
          .width(44)
          .height(44)
          .justifyContent(FlexAlign.Center)
          .onClick(() => { AppRouter.pop(); })

          if (this.showSearchBar) {
            TextInput({ placeholder: '搜索地点名称', text: this.searchText })
              .id('anchorPickSearchInput')
              .layoutWeight(1)
              .height(36)
              .borderRadius(DesignConstants.RADIUS.LG)
              .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
              .fontSize(DesignConstants.FONT.SIZE.MD)
              .onChange((value: string) => {
                this.searchText = value;
                if (this.searchDebounceTimerId > 0) {
                  clearTimeout(this.searchDebounceTimerId);
                  this.searchDebounceTimerId = 0;
                }
                if (value.trim().length > 0) {
                  this.searchDebounceTimerId = setTimeout(() => {
                    this.doSearch();
                  }, 500);
                } else {
                  this.searchResults = [];
                }
              })
              .onSubmit(() => {
                if (this.searchDebounceTimerId > 0) {
                  clearTimeout(this.searchDebounceTimerId);
                  this.searchDebounceTimerId = 0;
                }
                this.doSearch();
              })
              .transition(TransitionEffect.OPACITY.combine(
                TransitionEffect.translate({ x: 40 })
              ).animation({ duration: 250, curve: Curve.FastOutSlowIn }))

            if (this.isSearching) {
              LoadingProgress()
                .width(24)
                .height(24)
                .color($r('sys.color.ohos_id_color_emphasize'))
                .margin({ left: DesignConstants.SPACING.SM })
                .transition(TransitionEffect.OPACITY.animation({ duration: 250, curve: Curve.FastOutSlowIn }))
            } else {
              Text('取消')
                .fontSize(DesignConstants.FONT.SIZE.MD)
                .fontColor($r('sys.color.ohos_id_color_emphasize'))
                .margin({ left: DesignConstants.SPACING.SM })
                .transition(TransitionEffect.OPACITY.animation({ duration: 250, curve: Curve.FastOutSlowIn }))
                .onClick(() => {
                  this.getUIContext().animateTo({ duration: 250, curve: Curve.FastOutSlowIn }, () => {
                    this.showSearchBar = false;
                    this.searchText = '';
                    this.searchResults = [];
                  });
                })
            }
          } else {
            Text('浏览锚点')
              .fontSize(DesignConstants.FONT.SIZE.XL)
              .fontWeight(FontWeight.Bold)
              .fontColor($r('sys.color.ohos_id_color_text_primary'))
              .layoutWeight(1)
              .textAlign(TextAlign.Center)
              .transition(TransitionEffect.OPACITY.animation({ duration: 250, curve: Curve.FastOutSlowIn }))

            Column() {
              Text() {
                SymbolSpan($r(AppIcon.SEARCH))
                  .fontSize(22)
                  .fontColor([$r('sys.color.ohos_id_color_text_primary')])
              }
            }
            .width(44)
            .height(44)
            .justifyContent(FlexAlign.Center)
            .onClick(() => {
              this.getUIContext().animateTo({ duration: 250, curve: Curve.FastOutSlowIn }, () => {
                this.showSearchBar = true;
              });
              setTimeout(() => {
                this.getUIContext().getFocusController().requestFocus('anchorPickSearchInput');
              }, 300);
            })
          }
        }
        .width('100%')
        .height(56)
        .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
        .backgroundColor($r('sys.color.ohos_id_color_background'))

        // ===== 地图区域 + 覆盖层 =====
        Stack({ alignContent: Alignment.Center }) {
          NativeMapComponent({
            mapRegion: this.mapRegion,
            markers: this.mapMarkers,
            showCompass: true,
            showScale: true,
            showZoomControls: true,
            showMyLocationControls: false,
            onCameraIdle: (camera: MapCameraPosition) => { this.onCameraIdle(camera); },
            onMapDoubleClick: (coord: MapCoordinate) => { this.onMapDoubleClick(coord); },
            onMarkerClick: (marker: MapMarker) => { this.onMarkerItemClick(marker); }
          })

          // 中心选择器 — 地图钉
          Column() {
            Stack({ alignContent: Alignment.Center }) {
              Column()
                .width(40)
                .height(40)
                .borderRadius(20)
                .backgroundColor($r('app.color.color_pin_halo'))
              Column()
                .width(24)
                .height(24)
                .borderRadius(12)
                .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
                .border({ width: 3, color: $r('sys.color.ohos_id_color_background') })
                .shadow({ radius: 6, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 2 })
            }

            Column()
              .width(3)
              .height(14)
              .borderRadius(1.5)
              .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
              .margin({ top: -2 })

            Column()
              .width(10)
              .height(4)
              .borderRadius(5)
              .backgroundColor($r('app.color.color_overlay_light'))
              .margin({ top: 6 })
          }
          .offset({ y: this.isPinAnimating ? -30 : -18 })
          .scale({ x: this.isPinAnimating ? 1.15 : 1.0, y: this.isPinAnimating ? 1.15 : 1.0 })
          .animation({ duration: 500, curve: curves.springMotion(0.6, 0.8) })

          // ===== 筛选标签栏 =====
          Row({ space: DesignConstants.SPACING.SM }) {
            this.buildFilterChip('用户', this.showUsers, () => {
              this.showUsers = !this.showUsers;
              this.updateMarkers();
            })
            this.buildFilterChip('群组', this.showGroups, () => {
              this.showGroups = !this.showGroups;
              this.updateMarkers();
            })
            this.buildFilterChip('动态', this.showMoments, () => {
              this.showMoments = !this.showMoments;
              this.updateMarkers();
            })
          }
          .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
          .position({ top: DesignConstants.SPACING.MD, left: 0 })

          // 自定义「回到我的位置」按钮（替代 Map Kit 内置按钮）
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
          .borderRadius(22)
          .backgroundColor($r('sys.color.ohos_id_color_background'))
          .shadow({ radius: 8, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 2 })
          .onClick(() => { this.recenterToCurrentLocation(); })
          .position({ right: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.LG })

          // 地图提示消息
          if (this.mapToastMessage.length > 0) {
            Text(this.mapToastMessage)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor($r('sys.color.ohos_id_color_text_primary'))
              .backgroundColor($r('sys.color.ohos_id_color_background'))
              .borderRadius(DesignConstants.RADIUS.MD)
              .padding({
                left: DesignConstants.SPACING.MD,
                right: DesignConstants.SPACING.MD,
                top: DesignConstants.SPACING.XS,
                bottom: DesignConstants.SPACING.XS
              })
              .shadow({ radius: 6, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 2 })
              .position({ top: DesignConstants.SPACING.MD })
              .transition(TransitionEffect.OPACITY.animation({ duration: 250, curve: Curve.FastOutSlowIn }))
          }

          // 搜索结果列表覆盖层
          if (this.searchResults.length > 0) {
            Column() {
              List({ space: 0 }) {
                ForEach(this.searchResults, (item: LocationInfo, index: number) => {
                  ListItem() {
                    Row({ space: DesignConstants.SPACING.MD }) {
                      Column() {
                        Text() {
                          SymbolSpan($r(AppIcon.LOCATION))
                            .fontSize(20)
                            .fontColor([$r('sys.color.ohos_id_color_emphasize')])
                        }
                      }
                      .width(36)
                      .height(36)
                      .justifyContent(FlexAlign.Center)
                      .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
                      .borderRadius(DesignConstants.RADIUS.MD)

                      Column({ space: 2 }) {
                        Text(item.locationName || '未知地点')
                          .fontSize(DesignConstants.FONT.SIZE.MD)
                          .fontColor($r('sys.color.ohos_id_color_text_primary'))
                          .maxLines(2)
                          .textOverflow({ overflow: TextOverflow.Ellipsis })
                        Text(`${item.latitude.toFixed(6)}, ${item.longitude.toFixed(6)}`)
                          .fontSize(DesignConstants.FONT.SIZE.SM)
                          .fontColor($r('sys.color.ohos_id_color_text_tertiary'))
                      }
                      .layoutWeight(1)
                      .alignItems(HorizontalAlign.Start)
                    }
                    .width('100%')
                    .padding({
                      left: DesignConstants.SPACING.LG,
                      right: DesignConstants.SPACING.LG,
                      top: DesignConstants.SPACING.MD,
                      bottom: DesignConstants.SPACING.MD
                    })
                    .onClick(() => {
                      this.selectSearchResult(item);
                    })
                  }
                }, (item: LocationInfo, index: number) => `search_${index}_${item.latitude}_${item.longitude}`)
              }
              .width('100%')
              .divider({ strokeWidth: 0.5, color: $r('sys.color.ohos_id_color_list_separator') })
            }
            .width('100%')
            .constraintSize({ maxHeight: '70%' })
            .backgroundColor($r('sys.color.ohos_id_color_background'))
            .borderRadius({ bottomLeft: DesignConstants.RADIUS.LG, bottomRight: DesignConstants.RADIUS.LG })
            .shadow({ radius: 12, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 4 })
            .transition(
              TransitionEffect.OPACITY.combine(
                TransitionEffect.translate({ y: -20 })
              ).animation({ duration: 250, curve: Curve.FastOutSlowIn })
            )
            .position({ top: 0, left: 0 })
          }
        }
        .width('100%')
        .layoutWeight(1)

        // ===== 底部面板：位置名称 + 确认按钮 =====
        Column() {
          Column()
            .width(36)
            .height(4)
            .borderRadius(2)
            .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
            .margin({ top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.MD })

          Row({ space: DesignConstants.SPACING.SM }) {
            LoadingProgress()
              .width(18)
              .height(18)
              .color($r('sys.color.ohos_id_color_emphasize'))
              .opacity(this.isLoading ? 1 : 0)
              .animation({ duration: 200, curve: Curve.EaseInOut })

            Text(this.isLoading ? '正在识别位置...' : (this.locationName || '移动地图选择位置'))
              .fontSize(DesignConstants.FONT.SIZE.LG)
              .fontWeight(this.locationName.length > 0 && !this.isLoading ? FontWeight.Medium : FontWeight.Normal)
              .fontColor(this.locationName.length > 0 && !this.isLoading
                ? $r('sys.color.ohos_id_color_text_primary')
                : $r('sys.color.ohos_id_color_text_secondary'))
              .layoutWeight(1)
              .maxLines(2)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
              .animation({ duration: 200, curve: Curve.EaseInOut })
          }
          .width('100%')
          .constraintSize({ minHeight: 44 })
          .alignItems(VerticalAlign.Center)

          Button('设为浏览锚点')
            .type(ButtonType.Capsule)
            .width('100%')
            .height(48)
            .margin({ top: DesignConstants.SPACING.MD })
            .backgroundColor(this.locationName.length > 0 && !this.isLoading
              ? $r('sys.color.ohos_id_color_emphasize')
              : $r('sys.color.ohos_id_color_component_normal'))
            .fontColor(this.locationName.length > 0 && !this.isLoading
              ? Color.White
              : $r('sys.color.ohos_id_color_text_secondary'))
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontWeight(FontWeight.Medium)
            .enabled(this.locationName.length > 0 && !this.isLoading)
            .opacity(this.locationName.length > 0 && !this.isLoading ? 1.0 : 0.6)
            .animation({ duration: 250, curve: Curve.EaseInOut })
            .onClick(() => this.confirm())
        }
        .width('100%')
        .padding({
          left: DesignConstants.SPACING.XL,
          right: DesignConstants.SPACING.XL,
          bottom: DesignConstants.SPACING.XL
        })
        .backgroundColor($r('sys.color.ohos_id_color_background'))
        .borderRadius({ topLeft: DesignConstants.RADIUS.XL, topRight: DesignConstants.RADIUS.XL })
        .shadow({ radius: 16, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: -4 })
      }
      .width('100%')
      .height('100%')
      .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
      const param = context.pathInfo.param as Record<string, Object>;
      this.initFromParams(param);
    })
  }

  @Builder
  private buildFilterChip(label: string, active: boolean, action: () => void) {
    Text(label)
      .fontSize(DesignConstants.FONT.SIZE.SM)
      .fontWeight(FontWeight.Medium)
      .fontColor(active
        ? Color.White
        : $r('sys.color.ohos_id_color_text_primary'))
      .backgroundColor(active
        ? $r('sys.color.ohos_id_color_emphasize')
        : $r('sys.color.ohos_id_color_background'))
      .borderRadius(DesignConstants.RADIUS.FULL)
      .border({
        width: active ? 0 : 0.5,
        color: $r('sys.color.ohos_id_color_list_separator')
      })
      .padding({
        left: DesignConstants.SPACING.MD,
        right: DesignConstants.SPACING.MD,
        top: DesignConstants.SPACING.XS,
        bottom: DesignConstants.SPACING.XS
      })
      .shadow({ radius: 4, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 1 })
      .onClick(action)
      .animation({ duration: 200, curve: Curve.FastOutSlowIn })
  }
}

@Builder
export function AnchorPickPageBuilder() {
  AnchorPickPage()
}
