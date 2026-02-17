/**
 * 位置选择页面（通用选点）
 * - 供 CreateMomentPage、CreateGroupPage 等页面跳转使用
 * - 选取一个位置后将结果写入 KEY_LOCATION_PICK_RESULT，然后返回
 * - 默认定位到当前浏览锚点位置，但不会修改浏览锚点
 */

import { KEY_BROWSE_ANCHOR, KEY_CURRENT_LOCATION, KEY_LOCATION_PICK_RESULT } from '../../../core/state/GlobalStateManager';
import { LocationInfo, LocationUtil } from '../../../core/location/LocationUtil';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { NativeMapComponent, MapRegion, MapCoordinate, MapCameraPosition } from '../../../components/NativeMapComponent';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { curves } from '@kit.ArkUI';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { AppRouter } from '../../../core/router/AppRouter';
import { CoordinateConverter } from '../../../core/geo/CoordinateConverter';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { AnchorHistoryItem, AnchorHistoryUtil } from '../../../core/utils/AnchorHistoryUtil';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';

const DOMAIN_NUMBER: number = 0xFF14;
const TAG: string = 'LocationPickerPage';

@Component
export struct LocationPickerPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;

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
  @State anchorHistory: AnchorHistoryItem[] = [];

  private resolveTimerId: number = 0;
  private isFirstCameraIdle: boolean = true;
  private skipNextResolve: boolean = false;
  private searchDebounceTimerId: number = 0;
  private initialised: boolean = false;

  async aboutToAppear() {
    SystemUiController.useSurface({
      baseColor: DesignConstants.COLOR.PRIMARY_BG_LIGHT,
      preferDarkContent: true
    });
    this.initialised = false;
    this.anchorHistory = AnchorHistoryUtil.getHistory();
  }

  private initFromParams(_param: Record<string, Object> | undefined): void {
    this.initLocationAsync();
  }

  /**
   * 初始位置优先级：浏览锚点（已是 GCJ02）> 当前位置（WGS84 需转换）> geoLocationManager
   * 注意：只读取浏览锚点作为默认位置，不会修改它。
   */
  private async initLocationAsync(): Promise<void> {
    if (this.initialised) {
      return;
    }

    // 优先使用浏览锚点位置（已是 GCJ02，由 AnchorPickPage 写入）
    const browseAnchor = AppStorage.get<LocationInfo | null>(KEY_BROWSE_ANCHOR) ?? null;
    if (browseAnchor) {
      this.setCenter(browseAnchor.latitude, browseAnchor.longitude);
      await this.resolveLocation(browseAnchor.latitude, browseAnchor.longitude);
      this.initialised = true;
      return;
    }

    // 其次使用当前位置（WGS84 → 需要转 GCJ02）
    if (this.currentLocation) {
      const converted = CoordinateConverter.convertWgs84ToGcj02(
        this.currentLocation.latitude, this.currentLocation.longitude
      );
      this.setCenter(converted.latitude, converted.longitude);
      await this.resolveLocation(converted.latitude, converted.longitude);
      this.initialised = true;
      return;
    }

    // 兜底：通过 geoLocationManager 获取 + 转换
    const location = await CoordinateConverter.getConvertedLocation();
    if (location) {
      this.setCenter(location.latitude, location.longitude);
      await this.resolveLocation(location.latitude, location.longitude);
    } else {
      hilog.warn(DOMAIN_NUMBER, TAG, '无法获取当前位置');
    }
    this.initialised = true;
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

  private setCenter(lat: number, lng: number) {
    this.centerLat = lat;
    this.centerLng = lng;
    this.mapRegion = { latitude: lat, longitude: lng, latitudeDelta: 0.01, longitudeDelta: 0.01 };
  }

  private onCameraIdle(camera: MapCameraPosition): void {
    if (this.isFirstCameraIdle) {
      this.isFirstCameraIdle = false;
      return;
    }

    this.centerLat = camera.latitude;
    this.centerLng = camera.longitude;

    if (this.skipNextResolve) {
      this.skipNextResolve = false;
      return;
    }

    if (this.resolveTimerId > 0) {
      clearTimeout(this.resolveTimerId);
    }
    this.isLoading = true;
    this.resolveTimerId = setTimeout(() => {
      this.resolveLocation(camera.latitude, camera.longitude);
    }, 400);
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
      const addr = await LoadingUtil.withMinimumTime(
        () => LocationUtil.reverseGeocode(lat, lng),
        LoadingUtil.DEFAULT_ACTION_MS
      );
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
      const results = await LoadingUtil.withMinimumTime(
        () => LocationUtil.geocodeMultiple(keyword, 10),
        LoadingUtil.DEFAULT_ACTION_MS
      );
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
    return results.sort((a: LocationInfo, b: LocationInfo): number => {
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

  private selectHistoryItem(item: AnchorHistoryItem): void {
    this.skipNextResolve = true;
    this.setCenter(item.latitude, item.longitude);
    this.locationName = item.locationName || '历史位置';
    this.resolveLocation(item.latitude, item.longitude);
  }

  /**
   * 确认选点：将结果写入 KEY_LOCATION_PICK_RESULT，不修改浏览锚点。
   */
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
    AppStorage.setOrCreate<LocationInfo | null>(KEY_LOCATION_PICK_RESULT, location);
    AppRouter.pop();
  }

  build() {
    NavDestination() {
      Stack() {
        // 1. 地图（全铺）
        NativeMapComponent({
          mapRegion: this.mapRegion,
          markers: [],
          showCompass: true,
          showScale: true,
          showZoomControls: true,
          showMyLocationControls: true,
          contentPaddingTop: (this.safeAreaTop > 0 ? this.safeAreaTop : 20) + 80,
          contentPaddingBottom: 200 + this.safeAreaBottom,
          onCameraIdle: (camera: MapCameraPosition) => { this.onCameraIdle(camera); },
          onMapDoubleClick: (coord: MapCoordinate) => { this.onMapDoubleClick(coord); }
        })
        .width('100%')
        .height('100%')

        // 2. 中心指示器
        Column() {
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
          Column()
            .width(3)
            .height(14)
            .borderRadius(1.5)
            .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
            .margin({ top: 6 })
        }
        .offset({ y: this.isPinAnimating ? -30 : -18 })
        .scale({ x: this.isPinAnimating ? 1.12 : 1, y: this.isPinAnimating ? 1.12 : 1 })
        .animation({ duration: 500, curve: curves.springMotion(0.6, 0.8) })
        .hitTestBehavior(HitTestMode.Transparent)

        // 3. 全屏覆盖层：顶部浮层 + Blank + 底部面板
        Column() {
          // 顶部导航 + 搜索
          Column({ space: 12 }) {
            Row({ space: 12 }) {
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
              .backgroundColor($r('app.color.bg_secondary'))
              .borderRadius(22)
              .shadow({ radius: 6, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 2 })
              .onClick(() => { AppRouter.pop(); })

              TextInput({ placeholder: '搜索地点名称', text: this.searchText })
                .id('locationPickerSearchInput')
                .layoutWeight(1)
                .height(44)
                .borderRadius(22)
                .backgroundColor($r('app.color.bg_secondary'))
                .padding({ left: 16, right: 16 })
                .shadow({ radius: 6, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 2 })
                .fontSize(DesignConstants.FONT.SIZE.MD)
                .onChange((value: string) => {
                  this.searchText = value;
                  if (this.searchDebounceTimerId > 0) clearTimeout(this.searchDebounceTimerId);
                  if (value.trim().length > 0) {
                    this.searchDebounceTimerId = setTimeout(() => this.doSearch(), 500);
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

              if (this.isSearching) {
                LoadingProgress()
                  .width(24)
                  .height(24)
                  .color($r('sys.color.ohos_id_color_emphasize'))
                  .alignSelf(ItemAlign.Center)
              }
            }
            .width('100%')

            if (this.searchResults.length > 0) {
              Column() {
                List({ space: 0 }) {
                  ForEach(this.searchResults, (item: LocationInfo, index: number) => {
                    ListItem() {
                      Column({ space: 4 }) {
                        Text(item.locationName || '未知地点')
                          .fontSize(DesignConstants.FONT.SIZE.MD)
                          .fontColor($r('sys.color.ohos_id_color_text_primary'))
                          .maxLines(2)
                          .textOverflow({ overflow: TextOverflow.Ellipsis })
                        Text(`${item.latitude.toFixed(6)}, ${item.longitude.toFixed(6)}`)
                          .fontSize(DesignConstants.FONT.SIZE.SM)
                          .fontColor($r('sys.color.ohos_id_color_text_tertiary'))
                      }
                      .width('100%')
                      .padding({ left: 16, right: 16, top: 12, bottom: 12 })
                      .onClick(() => this.selectSearchResult(item))
                    }
                  }, (item: LocationInfo, index: number) => `search_${index}_${item.latitude}_${item.longitude}`)
                }
                .divider({ strokeWidth: 0.5, color: $r('sys.color.ohos_id_color_list_separator') })
              }
              .backgroundColor($r('app.color.bg_secondary'))
              .borderRadius(24)
              .shadow({ radius: 12, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 4 })
              .constraintSize({ maxHeight: 300 })
            }
          }
          .width('100%')
          .padding({ left: 16, right: 16 })
          .margin({ top: SafeAreaUtils.resolveTopPadding(this.safeAreaTop, DesignConstants.SPACING.XL) })

          Blank()

          // 底部信息面板（圆角露出地图，Map Kit 内置控件通过 contentPadding 推到可见区域）
          Column({ space: 16 }) {
            Column({ space: 4 }) {
              Row({ space: 8 }) {
                LoadingProgress()
                  .width(16)
                  .height(16)
                  .color($r('sys.color.ohos_id_color_emphasize'))
                  .opacity(this.isLoading ? 1 : 0)
                Text(this.isLoading ? '正在识别位置...' : (this.locationName || '移动地图选择位置'))
                  .fontSize(DesignConstants.FONT.SIZE.LG)
                  .fontWeight(FontWeight.Medium)
                  .fontColor($r('sys.color.ohos_id_color_text_primary'))
                  .maxLines(2)
                  .textOverflow({ overflow: TextOverflow.Ellipsis })
                  .layoutWeight(1)
              }
              if (this.locationName.length > 0 && !this.isLoading) {
                Text(`${this.centerLat.toFixed(6)}, ${this.centerLng.toFixed(6)}`)
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor($r('sys.color.ohos_id_color_text_tertiary'))
              }
            }

            if (this.anchorHistory.length > 0) {
              Column({ space: 8 }) {
                Text('历史锚点')
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                Scroll() {
                  Row({ space: 12 }) {
                    ForEach(this.anchorHistory, (item: AnchorHistoryItem, index: number) => {
                      Column({ space: 4 }) {
                        Text(item.locationName || '未命名位置')
                          .fontSize(DesignConstants.FONT.SIZE.SM)
                          .fontColor($r('sys.color.ohos_id_color_text_primary'))
                          .maxLines(1)
                          .textOverflow({ overflow: TextOverflow.Ellipsis })
                        Text(`${item.latitude.toFixed(4)}, ${item.longitude.toFixed(4)}`)
                          .fontSize(DesignConstants.FONT.SIZE.XS)
                          .fontColor($r('sys.color.ohos_id_color_text_tertiary'))
                      }
                      .padding({ left: 12, right: 12, top: 8, bottom: 8 })
                      .backgroundColor($r('app.color.bg_primary'))
                      .borderRadius(DesignConstants.RADIUS.LG)
                      .shadow({ radius: 4, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 1 })
                      .onClick(() => this.selectHistoryItem(item))
                    }, (item: AnchorHistoryItem, index: number) => `picker_history_${index}`)
                  }
                }
                .scrollable(ScrollDirection.Horizontal)
                .scrollBar(BarState.Off)
              }
            }

            Button('确认位置')
              .type(ButtonType.Capsule)
              .width('100%')
              .height(48)
              .backgroundColor(this.locationName.length > 0 && !this.isLoading
                ? $r('sys.color.ohos_id_color_emphasize')
                : $r('sys.color.ohos_id_color_component_normal'))
              .fontColor(this.locationName.length > 0 && !this.isLoading ? Color.White : $r('sys.color.ohos_id_color_text_secondary'))
              .fontSize(DesignConstants.FONT.SIZE.LG)
              .fontWeight(FontWeight.Medium)
              .enabled(this.locationName.length > 0 && !this.isLoading)
              .opacity(this.locationName.length > 0 && !this.isLoading ? 1 : 0.6)
              .onClick(() => this.confirm())
          }
          .width('100%')
          .padding({ left: 24, right: 24, bottom: 16 + this.safeAreaBottom, top: 16 })
          .backgroundColor($r('sys.color.ohos_id_color_background'))
          .borderRadius({ topLeft: 24, topRight: 24 })
          .shadow({ radius: 16, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: -4 })
        }
        .width('100%')
        .height('100%')
        .hitTestBehavior(HitTestMode.Transparent)
      }
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
      const param = context.pathInfo.param as Record<string, Object>;
      this.initFromParams(param);
    })
  }
}

@Builder
export function LocationPickerPageBuilder() {
  LocationPickerPage()
}
