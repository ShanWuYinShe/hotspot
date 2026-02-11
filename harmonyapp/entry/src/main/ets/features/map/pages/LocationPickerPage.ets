/**
 * 位置选择页面（地图选点）
 * - 顶部搜索框：输入地名快速跳转
 * - 拖拽地图移动 → 停止后自动加载附近地点列表
 * - 双击地图 → 将选择器快速移动到该位置
 * - 用户从列表中选择一个地点确认
 */

import { KEY_BROWSE_ANCHOR, KEY_CURRENT_LOCATION, KEY_LOCATION_PICK_RESULT } from '../../../core/state/GlobalStateManager';
import { LocationInfo, LocationUtil } from '../../../core/location/LocationUtil';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { AnchorHistoryItem, AnchorHistoryUtil } from '../../../core/utils/AnchorHistoryUtil';
import { NativeMapComponent, NativeMapActions, MapRegion, MapCoordinate } from '../../../components/NativeMapComponent';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { curves } from '@kit.ArkUI';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { AppRouter } from '../../../core/router/AppRouter';

const DOMAIN_NUMBER: number = 0xFF13;
const TAG: string = 'LocationPickerPage';

@Component
export struct LocationPickerPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;

  @State mapRegion: MapRegion = {
    latitude: 35.154921,
    longitude: 112.68253,
    latitudeDelta: 0.01,
    longitudeDelta: 0.01
  };

  /** 当前解析出的位置名称 */
  @State locationName: string = '';
  /** 当前中心坐标 */
  @State centerLat: number = 0;
  @State centerLng: number = 0;
  @State isLoading: boolean = false;
  @State titleText: string = '地图选点';
  @State confirmText: string = '确认选择';
  @State isPinAnimating: boolean = false;

  /** 搜索相关 */
  @State searchText: string = '';
  @State isSearching: boolean = false;
  @State showSearchBar: boolean = false;
  @State searchResults: LocationInfo[] = [];

  /** 锚点历史（仅 browse_anchor 模式使用） */
  @State anchorHistory: AnchorHistoryItem[] = [];

  private resultKey: string = KEY_LOCATION_PICK_RESULT;
  private mode: 'browse_anchor' | 'pick_location' = 'pick_location';
  private resolveTimerId: number = 0;
  private isFirstCameraIdle: boolean = true;
  /** 编程式 setCenter 后跳过下一次 cameraIdle 的位置解析 */
  private skipNextResolve: boolean = false;
  /** 搜索防抖计时器 */
  private searchDebounceTimerId: number = 0;
  /** 地图控制操作（缩放等） */
  private mapActions: NativeMapActions | null = null;

  async aboutToAppear() {
    // 参数读取与初始化已移至 onReady
  }

  /** 从路由参数初始化（在 onReady 中调用） */
  private initFromParams(param: Record<string, Object> | undefined): void {
    const mode = this.getStringParam(param, 'mode');
    this.mode = mode === 'browse_anchor' ? 'browse_anchor' : 'pick_location';

    const resultKey = this.getStringParam(param, 'resultKey');
    this.resultKey = typeof resultKey === 'string' && resultKey.length > 0 ? resultKey : KEY_LOCATION_PICK_RESULT;

    this.titleText = this.getStringParam(param, 'title') || (this.mode === 'browse_anchor' ? '设置浏览锚点' : '地图选点');
    this.confirmText = this.getStringParam(param, 'confirmText') || (this.mode === 'browse_anchor' ? '设为浏览锚点' : '确认选择');

    AppStorage.setOrCreate<LocationInfo | null>(this.resultKey, null);

    if (this.mode === 'browse_anchor') {
      this.anchorHistory = AnchorHistoryUtil.getHistory();
    }

    this.initLocationAsync();
  }

  private async initLocationAsync(): Promise<void> {
    if (this.mode === 'browse_anchor') {
      const anchor = AppStorage.get<LocationInfo | null>(KEY_BROWSE_ANCHOR);
      if (anchor) {
        this.setCenter(anchor.latitude, anchor.longitude);
        await this.resolveLocation(anchor.latitude, anchor.longitude);
        return;
      }
    }

    if (this.currentLocation) {
      this.setCenter(this.currentLocation.latitude, this.currentLocation.longitude);
      await this.resolveLocation(this.currentLocation.latitude, this.currentLocation.longitude);
      return;
    }

    const location = await LocationUtil.getCurrentLocation();
    if (location) {
      this.setCenter(location.latitude, location.longitude);
      await this.resolveLocation(location.latitude, location.longitude);
    } else {
      hilog.warn(DOMAIN_NUMBER, TAG, '无法获取当前位置，等待位置服务就绪');
    }
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

  private getStringParam(params: Object | undefined, key: string): string | undefined {
    if (!params || typeof params !== 'object') return undefined;
    const dict = params as Record<string, Object>;
    const v = dict[key];
    return typeof v === 'string' ? v as string : undefined;
  }

  private setCenter(lat: number, lng: number) {
    this.mapRegion = { latitude: lat, longitude: lng, latitudeDelta: 0.01, longitudeDelta: 0.01 };
  }

  // ========== 地图事件 ==========

  private onCameraIdle(center: MapCoordinate): void {
    if (this.isFirstCameraIdle) {
      this.isFirstCameraIdle = false;
      return;
    }

    // 编程式 setCenter（搜索选点/历史选点等）后跳过反查，避免覆盖已知名称
    if (this.skipNextResolve) {
      this.skipNextResolve = false;
      return;
    }

    if (this.resolveTimerId > 0) {
      clearTimeout(this.resolveTimerId);
    }

    this.isLoading = true;

    this.resolveTimerId = setTimeout(() => {
      this.resolveLocation(center.latitude, center.longitude);
    }, 400);
  }

  private onMapDoubleClick(coord: MapCoordinate): void {
    // 第一段：图钉弹起（快速上升）
    this.getUIContext().animateTo({
      duration: 200,
      curve: Curve.FastOutSlowIn
    }, () => {
      this.isPinAnimating = true;
    });
    // 第二段：图钉落下 + 回弹（使用弹簧曲线，更自然）
    setTimeout(() => {
      this.getUIContext().animateTo({
        duration: 500,
        curve: curves.springMotion(0.6, 0.8)
      }, () => {
        this.isPinAnimating = false;
      });
    }, 200);
    this.setCenter(coord.latitude, coord.longitude);
  }

  // ========== 位置解析 ==========

  private async resolveLocation(lat: number, lng: number): Promise<void> {
    this.isLoading = true;
    this.centerLat = lat;
    this.centerLng = lng;
    try {
      const addr = await LoadingUtil.withMinimumTime(
        () => LocationUtil.reverseGeocode(lat, lng),
        LoadingUtil.DEFAULT_ACTION_MS
      );
      this.locationName = addr?.fullAddress || '未知位置';
    } catch (e) {
      hilog.warn(DOMAIN_NUMBER, TAG, `解析位置失败: ${JSON.stringify(e)}`);
      // 失败时保留旧名称
      if (this.locationName.length === 0) {
        this.locationName = '未知位置';
      }
    } finally {
      this.isLoading = false;
    }
  }

  // ========== 搜索 ==========

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
        // 收起键盘，让用户看到结果列表
        this.getUIContext().getFocusController().clearFocus();
      } else {
        SystemToast.show('未找到该地点，请换个关键词试试', this.getUIContext());
      }
    } catch (e) {
      hilog.warn(DOMAIN_NUMBER, TAG, `搜索地点失败: ${JSON.stringify(e)}`);
      SystemToast.show('搜索失败，请稍后重试', this.getUIContext());
    } finally {
      this.isSearching = false;
    }
  }

  /** 按与参考点的近似距离排序（仅用于展示排序，非精确距离计算） */
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
    // 跳过下一次 cameraIdle 的反查（名称已知）
    this.skipNextResolve = true;
    // 选中搜索结果 → 移动地图到该位置
    this.setCenter(result.latitude, result.longitude);
    this.locationName = result.locationName || '';
    this.centerLat = result.latitude;
    this.centerLng = result.longitude;
    // 清空搜索状态
    this.getUIContext().animateTo({ duration: 250, curve: Curve.FastOutSlowIn }, () => {
      this.searchResults = [];
      this.showSearchBar = false;
      this.searchText = '';
    });
  }

  // ========== 其他操作 ==========

  /**
   * 移动浏览锚点到当前位置
   * 使用 @StorageLink 缓存的位置（与地图蓝点使用相同的 Location Kit 数据源）
   */
  private relocateToCurrentPosition(): void {
    if (this.currentLocation) {
      this.setCenter(this.currentLocation.latitude, this.currentLocation.longitude);
    } else {
      SystemToast.show('无法获取当前位置', this.getUIContext());
    }
  }

  /** 选择历史记录条目 → 移动地图到该位置 */
  private selectHistory(item: AnchorHistoryItem): void {
    this.skipNextResolve = true;
    this.setCenter(item.latitude, item.longitude);
    this.locationName = item.locationName;
    this.centerLat = item.latitude;
    this.centerLng = item.longitude;
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
    AppStorage.setOrCreate<LocationInfo | null>(this.resultKey, location);
    if (this.mode === 'browse_anchor') {
      AnchorHistoryUtil.addToHistory(this.centerLat, this.centerLng, this.locationName);
    }
    AppRouter.pop();
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
          // 搜索输入框（带入场动画）
          TextInput({ placeholder: '搜索地点名称', text: this.searchText })
            .id('locationPickerSearchInput')
            .layoutWeight(1)
            .height(36)
            .borderRadius(DesignConstants.RADIUS.LG)
            .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .onChange((value: string) => {
              this.searchText = value;
              // 清除上一次防抖计时器
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
              // 主动提交时立即搜索，清除防抖
              if (this.searchDebounceTimerId > 0) {
                clearTimeout(this.searchDebounceTimerId);
                this.searchDebounceTimerId = 0;
              }
              this.doSearch();
            })
            .transition(TransitionEffect.OPACITY.combine(
              TransitionEffect.translate({ x: 40 })
            ).animation({ duration: 250, curve: Curve.FastOutSlowIn }))

          // 搜索 / 取消按钮
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
          // 标题（带入场动画）
          Text(this.titleText)
            .fontSize(DesignConstants.FONT.SIZE.XL)
            .fontWeight(FontWeight.Bold)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
            .layoutWeight(1)
            .textAlign(TextAlign.Center)
            .transition(TransitionEffect.OPACITY.animation({ duration: 250, curve: Curve.FastOutSlowIn }))

          // 搜索按钮
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
            // 延迟请求焦点，确保输入框已渲染
            setTimeout(() => {
              this.getUIContext().getFocusController().requestFocus('locationPickerSearchInput');
            }, 300);
          })
        }
      }
      .width('100%')
      .height(56)
      .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      // ===== 地图区域 + 搜索结果覆盖层 =====
      Stack({ alignContent: Alignment.Center }) {
        NativeMapComponent({
          mapRegion: this.mapRegion,
          markers: [],
          showCompass: true,
          showScale: true,
          showZoomControls: false,
          showMyLocationControls: false,
          onCameraIdle: (center: MapCoordinate) => { this.onCameraIdle(center); },
          onMapDoubleClick: (coord: MapCoordinate) => { this.onMapDoubleClick(coord); },
          onActionsReady: (actions: NativeMapActions) => { this.mapActions = actions; }
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

          // 移动浏览锚点到当前位置按钮
          Column() {
            Image($r('sys.media.ohos_ic_public_location'))
              .width(22)
              .height(22)
              .fillColor($r('sys.color.ohos_id_color_emphasize'))
          }
          .width(44)
          .height(44)
          .justifyContent(FlexAlign.Center)
          .backgroundColor($r('sys.color.ohos_id_color_background'))
          .borderRadius(DesignConstants.RADIUS.FULL)
          .shadow({ radius: 8, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 2 })
          .onClick(() => { this.relocateToCurrentPosition(); })
        }
        .position({ right: 16, bottom: 24 })

        // 搜索覆盖层：历史记录（无输入时） / 搜索结果（有结果时）
        if (this.showSearchBar && this.searchText.trim().length === 0 && this.searchResults.length === 0
          && this.anchorHistory.length > 0 && this.mode === 'browse_anchor') {
          Column({ space: 0 }) {
            // 标题行
            Row() {
              Text('最近使用')
                .fontSize(DesignConstants.FONT.SIZE.MD)
                .fontWeight(FontWeight.Medium)
                .fontColor($r('sys.color.ohos_id_color_text_primary'))
              Blank()
              Text('清空')
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .fontColor($r('sys.color.ohos_id_color_text_tertiary'))
                .onClick(() => {
                  AnchorHistoryUtil.clearHistory();
                  this.anchorHistory = [];
                })
            }
            .width('100%')
            .padding({
              left: DesignConstants.SPACING.LG,
              right: DesignConstants.SPACING.LG,
              top: DesignConstants.SPACING.MD,
              bottom: DesignConstants.SPACING.SM
            })

            // 历史列表
            List({ space: 0 }) {
              ForEach(this.anchorHistory, (item: AnchorHistoryItem, index: number) => {
                ListItem() {
                  Row({ space: DesignConstants.SPACING.MD }) {
                    Column() {
                      Text() {
                        SymbolSpan($r(AppIcon.TIME))
                          .fontSize(18)
                          .fontColor([$r('sys.color.ohos_id_color_text_tertiary')])
                      }
                    }
                    .width(36)
                    .height(36)
                    .justifyContent(FlexAlign.Center)
                    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
                    .borderRadius(DesignConstants.RADIUS.MD)

                    Text(item.locationName)
                      .fontSize(DesignConstants.FONT.SIZE.MD)
                      .fontColor($r('sys.color.ohos_id_color_text_primary'))
                      .maxLines(1)
                      .textOverflow({ overflow: TextOverflow.Ellipsis })
                      .layoutWeight(1)
                  }
                  .width('100%')
                  .padding({
                    left: DesignConstants.SPACING.LG,
                    right: DesignConstants.SPACING.LG,
                    top: DesignConstants.SPACING.MD,
                    bottom: DesignConstants.SPACING.MD
                  })
                  .onClick(() => {
                    this.selectHistory(item);
                    // 选完后收起搜索覆盖层
                    this.getUIContext().animateTo({ duration: 250, curve: Curve.FastOutSlowIn }, () => {
                      this.showSearchBar = false;
                      this.searchText = '';
                      this.searchResults = [];
                    });
                  })
                }
              }, (item: AnchorHistoryItem, index: number) => `history_${index}_${item.timestamp}`)
            }
            .width('100%')
            .divider({ strokeWidth: 0.5, color: $r('sys.color.ohos_id_color_list_separator') })
          }
          .width('100%')
          .constraintSize({ maxHeight: '60%' })
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

        // 搜索结果列表覆盖层
        if (this.searchResults.length > 0) {
          Column() {
            List({ space: 0 }) {
              ForEach(this.searchResults, (item: LocationInfo, index: number) => {
                ListItem() {
                  Row({ space: DesignConstants.SPACING.MD }) {
                    // 定位图标
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

                    // 名称 + 坐标
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
        // 拖拽指示条
        Column()
          .width(36)
          .height(4)
          .borderRadius(2)
          .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
          .margin({ top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.MD })

        // 位置名称（始终渲染，通过 opacity + animation 平滑过渡）
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

        // 确认按钮（backgroundColor / fontColor / opacity 平滑过渡）
        Button(this.confirmText)
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
}

@Builder
export function LocationPickerPageBuilder() {
  LocationPickerPage()
}
