import { KEY_BROWSE_ANCHOR, KEY_CURRENT_LOCATION, KEY_LOCATION_PICK_RESULT } from '../../../core/state/GlobalStateManager';
import { LocationInfo, LocationUtil } from '../../../core/location/LocationUtil';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { AnchorHistoryItem, AnchorHistoryUtil } from '../../../core/utils/AnchorHistoryUtil';
import { NativeMapComponent, NativeMapActions, MapRegion, MapCoordinate, MapMarker, MapCameraPosition } from '../../../components/NativeMapComponent';
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
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';

const DOMAIN_NUMBER: number = 0xFF13;
const TAG: string = 'LocationPickerPage';
const SEARCH_PANEL_HEIGHT: number = 520;
const MAX_HISTORY_ITEMS: number = 5;
// 底部面板三段式吸附位置
const PANEL_SNAP_MIN: number = 200;       // 收起：手柄 + 搜索栏 + 确认按钮
const PANEL_SNAP_HALF: number = 340;      // 半展开：显示部分历史记录
const PANEL_SNAP_MAX: number = 500;       // 全展开：完整内容

interface BoundsDelta {
  latDelta: number;
  lonDelta: number;
}

type LocationPickerMode = 'anchor' | 'location_pick';

interface LocationPickerParamLike {
  mode?: string;
}

interface TopBarProps {
  showUsers: boolean;
  showGroups: boolean;
  showMoments: boolean;
  safeAreaTop: number;
  onBack: () => void;
  onToggleUsers: () => void;
  onToggleGroups: () => void;
  onToggleMoments: () => void;
}

@Builder
function TopBar(props: TopBarProps) {
  Flex({
    direction: FlexDirection.Row,
    justifyContent: FlexAlign.SpaceBetween,
    alignItems: ItemAlign.Center
  }) {
    Row() {
      Text() {
        SymbolSpan($r(AppIcon.BACK))
          .fontSize(22)
          .fontColor([$r('sys.color.ohos_id_color_text_primary')])
      }
    }
    .width(44)
    .height(44)
    .justifyContent(FlexAlign.Center)
    .onClick(props.onBack)

    Row({ space: 8 }) {
      MapFilterChip({
        label: '用户',
        iconSymbol: $r(AppIcon.PERSON),
        accentColor: '#007AFF',
        active: props.showUsers,
        onToggle: props.onToggleUsers
      })
      MapFilterChip({
        label: '群组',
        iconSymbol: $r(AppIcon.GROUP),
        accentColor: '#34C759',
        active: props.showGroups,
        onToggle: props.onToggleGroups
      })
      MapFilterChip({
        label: '动态',
        iconSymbol: $r(AppIcon.PHOTO),
        accentColor: '#FF9500',
        active: props.showMoments,
        onToggle: props.onToggleMoments
      })
    }
    .padding({ left: 8, right: 8, top: 4, bottom: 4 })
    .backgroundColor($r('app.color.bg_secondary'))
    .borderRadius(20)
    .shadow({ radius: 6, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 2 })
  }
  .width('100%')
  .padding({
    left: 12,
    right: 12,
    top: SafeAreaUtils.resolveTopPadding(props.safeAreaTop, DesignConstants.SPACING.XS)
  })
  .backgroundColor(Color.Transparent)
}

interface MapToastProps {
  message: string;
  safeAreaTop: number;
}

@Builder
function MapToast(props: MapToastProps) {
  if (props.message.length === 0) {
    // 不渲染
  } else {
    Text(props.message)
      .fontSize(DesignConstants.FONT.SIZE.SM)
      .fontColor($r('sys.color.ohos_id_color_text_primary'))
      .backgroundColor($r('app.color.bg_secondary'))
      .borderRadius(DesignConstants.RADIUS.MD)
      .padding({ left: 12, right: 12, top: 6, bottom: 6 })
      .shadow({ radius: 6, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 2 })
      .position({ top: SafeAreaUtils.resolveTopPadding(props.safeAreaTop, 100), left: 16 })
      .transition(TransitionEffect.OPACITY.animation({ duration: 250 }))
  }
}

/**
 * 搜索面板（当 showSearchBar 为 true 时显示）
 * 固定高度，不可拖动
 */
@Component
struct SearchPanel {
  @Link searchText: string;
  @Link isSearching: boolean;
  @Link searchResults: LocationInfo[];
  @Link anchorHistory: AnchorHistoryItem[];
  onSearch: () => void = () => {};
  onClose: () => void = () => {};
  onClearHistory: () => void = () => {};
  onSelectResult: (result: LocationInfo) => void = (result) => {};
  onSelectHistory: (item: AnchorHistoryItem) => void = (item) => {};
  onTextChange: (value: string) => void = (value) => {};
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;

  build() {
    Column() {
      // 搜索行 - 使用官方 Search 组件
      Row({ space: 8 }) {
        // 官方 Search 组件（全局内置，无需导入）
        Search({
          value: this.searchText,
          placeholder: '搜索地点、地标或坐标',
          controller: new SearchController()
        })
          .id('locationPickerSearchInput')
          .layoutWeight(1)
          .height(40)
          .backgroundColor($r('app.color.bg_primary'))
          .borderRadius(20)
          .searchIcon({
            size: 18,
            color: $r('sys.color.ohos_id_color_text_primary'),
            src: '' // 使用默认搜索图标
          })
          .cancelButton({
            style: CancelButtonStyle.INPUT // 使用内置清除按钮样式，不指定图标，采用默认图标
          })
          .searchButton('搜索')
          .onChange((value: string) => {
            this.onTextChange(value);
            if (value.trim().length === 0) {
              this.searchResults = [];
            }
          })
          .onSubmit(() => {
            if (this.searchText.trim().length > 0) {
              this.onSearch();
            }
          })

        if (this.isSearching) {
          LoadingProgress()
            .width(18)
            .height(18)
            .color($r('sys.color.ohos_id_color_emphasize'))
        }

        // 取消按钮
        Column() {
          Text('取消')
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
        }
        .height(40)
        .justifyContent(FlexAlign.Center)
        .padding({ left: 8, right: 8 })
        .onClick(this.onClose)
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)
      .padding({ bottom: 12 })

      // 内容区域 - 可滚动
      Scroll() {
        Column() {
          if (this.isSearching) {
            // 加载中状态
            Row() {
              LoadingProgress()
                .width(24)
                .height(24)
                .color($r('sys.color.ohos_id_color_emphasize'))
              Text('正在搜索...')
                .fontSize(DesignConstants.FONT.SIZE.MD)
                .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                .margin({ left: 8 })
            }
            .width('100%')
            .height(80)
            .justifyContent(FlexAlign.Center)
          } else if (this.searchResults.length > 0) {
            // 搜索结果列表
            List({ space: 0 }) {
              ForEach(this.searchResults, (item: LocationInfo, index: number) => {
                ListItem() {
                  Row({ space: DesignConstants.SPACING.MD }) {
                    Column() {
                      Text() {
                        SymbolSpan($r(AppIcon.LOCATION))
                          .fontSize(18)
                          .fontColor([$r('sys.color.ohos_id_color_emphasize')])
                      }
                    }
                    .width(32)
                    .height(32)
                    .justifyContent(FlexAlign.Center)
                    .backgroundColor($r('app.color.bg_primary'))
                    .borderRadius(DesignConstants.RADIUS.MD)

                    Text(item.locationName || '未知地点')
                      .fontSize(DesignConstants.FONT.SIZE.MD)
                      .fontColor($r('sys.color.ohos_id_color_text_primary'))
                      .layoutWeight(1)
                      .maxLines(2)
                      .textOverflow({ overflow: TextOverflow.Ellipsis })
                  }
                  .width('100%')
                  .padding({ left: 4, right: 4, top: 10, bottom: 10 })
                  .onClick(() => { this.onSelectResult(item); })
                }
              }, (item: LocationInfo) => `search_${item.latitude}_${item.longitude}`)
            }
            .width('100%')
            .divider({ strokeWidth: 0.5, color: $r('sys.color.ohos_id_color_list_separator') })
          } else if (this.searchText.trim().length > 0) {
            // 无搜索结果
            Column() {
              Text('未找到该地点')
                .fontSize(DesignConstants.FONT.SIZE.MD)
                .fontColor($r('sys.color.ohos_id_color_text_primary'))
              Text('请换个关键词试试')
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                .margin({ top: 4 })
            }
            .width('100%')
            .height(120)
            .justifyContent(FlexAlign.Center)
            .alignItems(HorizontalAlign.Center)
          } else if (this.anchorHistory.length > 0) {
            // 历史锚点列表（搜索未输入时展示）
            Column({ space: 8 }) {
              Row({ space: 6 }) {
                Text('最近使用')
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .fontWeight(FontWeight.Medium)
                  .fontColor($r('sys.color.ohos_id_color_text_primary'))
                Blank()
                Text('清空')
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                  .onClick(this.onClearHistory)
              }
              .width('100%')

              List({ space: 8 }) {
                ForEach(this.anchorHistory.slice(0, MAX_HISTORY_ITEMS), (item: AnchorHistoryItem, index: number) => {
                  ListItem() {
                    Row({ space: 12 }) {
                      Column() {
                        Text() {
                          SymbolSpan($r(AppIcon.TIME))
                            .fontSize(18)
                            .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
                        }
                      }
                      .width(32)
                      .height(32)
                      .justifyContent(FlexAlign.Center)

                      Text(item.locationName || '未命名位置')
                        .fontSize(DesignConstants.FONT.SIZE.MD)
                        .fontColor($r('sys.color.ohos_id_color_text_primary'))
                        .layoutWeight(1)
                        .maxLines(1)
                        .textOverflow({ overflow: TextOverflow.Ellipsis })
                    }
                    .width('100%')
                    .padding({ top: 8, bottom: 8 })
                    .onClick(() => {
                      this.onSelectHistory(item);
                      this.onClose();
                    })
                  }
                }, (item: AnchorHistoryItem) => `hist_s_${item.latitude}_${item.longitude}`)
              }
              .width('100%')
              .divider({ strokeWidth: 0.5, color: $r('sys.color.ohos_id_color_list_separator') })
            }
            .width('100%')
          } else {
            // 空状态
            Column() {
              Text('暂无搜索历史')
                .fontSize(DesignConstants.FONT.SIZE.MD)
                .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            }
            .width('100%')
            .height(100)
            .justifyContent(FlexAlign.Center)
            .alignItems(HorizontalAlign.Center)
          }
        }
        .width('100%')
        .padding({ top: 4 })
      }
      .scrollable(ScrollDirection.Vertical)
      .scrollBar(BarState.Auto)
      .layoutWeight(1)
      .width('100%')
    }
    .height(SEARCH_PANEL_HEIGHT)
    .padding({ left: 16, right: 16, top: 16, bottom: 12 + this.safeAreaBottom })
    .backgroundColor($r('app.color.bg_secondary'))
    .borderRadius({ topLeft: 24, topRight: 24 })
    .shadow({ radius: 16, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: -4 })
  }
}

/**
 * 主面板 —— 可拖拽的底部面板（Sheet 风格）
 * 三段式吸附：收起 / 半展开 / 全展开，参考 HarmonyOS Petal Maps 交互
 */
@Component
struct MainPanel {
  @Link anchorHistory: AnchorHistoryItem[];
  @Link panelHeight: number;
  onSearchClick: () => void = () => {};
  onClearHistory: () => void = () => {};
  onSelectHistory: (item: AnchorHistoryItem) => void = (item) => {};
  onConfirm: () => void = () => {};
  @Prop confirmLabel: string = '设为浏览锚点';
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;

  private dragStartHeight: number = 0;

  private snapToNearest(): void {
    const snapPoints: number[] = [PANEL_SNAP_MIN, PANEL_SNAP_HALF, PANEL_SNAP_MAX];
    let target = snapPoints[0];
    let minDist = Math.abs(this.panelHeight - snapPoints[0]);
    for (let i = 1; i < snapPoints.length; i++) {
      const dist = Math.abs(this.panelHeight - snapPoints[i]);
      if (dist < minDist) {
        minDist = dist;
        target = snapPoints[i];
      }
    }
    this.getUIContext().animateTo(
      { duration: 250, curve: curves.springMotion(0.6, 0.9) },
      () => { this.panelHeight = target; }
    );
  }

  build() {
    Column() {
      // 拖拽手柄区域（36px 高的触控热区，内含视觉指示条）
      Column() {
        Column()
          .width(36)
          .height(4)
          .borderRadius(2)
          .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
      }
      .width('100%')
      .height(36)
      .justifyContent(FlexAlign.Center)
      .gesture(
        PanGesture({ direction: PanDirection.Vertical })
          .onActionStart(() => {
            this.dragStartHeight = this.panelHeight;
          })
          .onActionUpdate((event: GestureEvent) => {
            let newHeight = this.dragStartHeight - event.offsetY;
            newHeight = Math.max(PANEL_SNAP_MIN, Math.min(PANEL_SNAP_MAX, newHeight));
            this.panelHeight = newHeight;
          })
          .onActionEnd(() => {
            this.snapToNearest();
          })
      )

      // 搜索入口（模拟 Search 输入框外观，点击后展开真正的搜索面板）
      Row({ space: 8 }) {
        Text() {
          SymbolSpan($r(AppIcon.SEARCH))
            .fontSize(16)
            .fontColor([$r('sys.color.ohos_id_color_text_tertiary')])
        }
        Text('搜索地点、地标或坐标')
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor($r('sys.color.ohos_id_color_text_tertiary'))
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
      }
      .width('100%')
      .height(40)
      .padding({ left: 12, right: 12 })
      .alignItems(VerticalAlign.Center)
      .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
      .borderRadius(20)
      .margin({ bottom: 12 })
      .onClick(this.onSearchClick)

      // 可滚动历史记录（面板最低档时隐藏，由 clip(true) 保证拖拽过程平滑裁剪）
      if (this.panelHeight > PANEL_SNAP_MIN) {
        Scroll() {
          Column() {
            // 最近使用标题
            Row() {
              Text('最近使用')
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .fontWeight(FontWeight.Medium)
                .fontColor($r('sys.color.ohos_id_color_text_secondary'))
              Blank()
              if (this.anchorHistory.length > 0) {
                Text('清空')
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor($r('sys.color.ohos_id_color_text_tertiary'))
                  .padding({ left: 8, top: 4, bottom: 4 })
                  .onClick(this.onClearHistory)
              }
            }
            .width('100%')
            .margin({ bottom: 8 })

            if (this.anchorHistory.length > 0) {
              List({ space: 0 }) {
                ForEach(this.anchorHistory.slice(0, MAX_HISTORY_ITEMS), (item: AnchorHistoryItem, index: number) => {
                  ListItem() {
                    Row({ space: 10 }) {
                      Text() {
                        SymbolSpan($r(AppIcon.TIME))
                          .fontSize(16)
                          .fontColor([$r('sys.color.ohos_id_color_text_tertiary')])
                      }

                      Text(item.locationName || '未命名位置')
                        .fontSize(DesignConstants.FONT.SIZE.MD)
                        .fontColor($r('sys.color.ohos_id_color_text_primary'))
                        .layoutWeight(1)
                        .maxLines(1)
                        .textOverflow({ overflow: TextOverflow.Ellipsis })
                    }
                    .width('100%')
                    .padding({ top: 10, bottom: 10 })
                    .onClick(() => { this.onSelectHistory(item); })
                  }
                }, (item: AnchorHistoryItem) => `hist_m_${item.latitude}_${item.longitude}`)
              }
              .width('100%')
              .divider({ strokeWidth: 0.5, color: $r('sys.color.ohos_id_color_list_separator') })
            } else {
              Column() {
                Text() {
                  SymbolSpan($r(AppIcon.TIME))
                    .fontSize(16)
                    .fontColor([$r('sys.color.ohos_id_color_text_tertiary')])
                }
                Text('暂无最近使用的地点')
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor($r('sys.color.ohos_id_color_text_tertiary'))
                  .margin({ top: 4 })
              }
              .width('100%')
              .padding({ top: 16 })
            }
          }
          .width('100%')
        }
        .scrollable(ScrollDirection.Vertical)
        .scrollBar(BarState.Off)
        .align(Alignment.TopStart)
        .layoutWeight(1)
        .width('100%')
      }

      // 确认按钮（固定在底部，始终可见）
      Button(this.confirmLabel)
        .type(ButtonType.Capsule)
        .width('100%')
        .height(44)
        .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
        .fontColor(Color.White)
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .fontWeight(FontWeight.Medium)
        .margin({ top: 8 })
        .onClick(this.onConfirm)
    }
    .height(this.panelHeight)
    .clip(true)
    .padding({ left: 16, right: 16, top: 0, bottom: 12 + this.safeAreaBottom })
    .backgroundColor($r('app.color.bg_secondary'))
    .borderRadius({ topLeft: 24, topRight: 24 })
    .shadow({ radius: 16, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: -4 })
  }
}

@Component
export struct LocationPickerPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;

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
  @State searchText: string = '';
  @State isSearching: boolean = false;
  @State showSearchBar: boolean = false;
  @State searchResults: LocationInfo[] = [];
  @State mapMarkers: MapMarker[] = [];
  @State showUsers: boolean = true;
  @State showGroups: boolean = true;
  @State showMoments: boolean = true;
  @State mapToastMessage: string = '';
  @State anchorHistory: AnchorHistoryItem[] = [];
  @State pickMode: LocationPickerMode = 'anchor';

  // 底部面板高度（仅在主面板时使用，初始为半展开位置）
  @State panelHeight: number = PANEL_SNAP_HALF;

  private renderItemsMap: Map<string, MapRenderItem> = new Map();
  private resolveTimerId: number = 0;
  private renderTimerId: number = 0;
  private isFirstCameraIdle: boolean = true;
  private skipNextResolve: boolean = false;
  private searchDebounceTimerId: number = 0;
  private mapZoom: number = 15;
  private initialised: boolean = false;
  private renderItems: MapRenderItem[] = [];
  private lastViewportSignature: string = '';
  private mapActions: NativeMapActions | null = null;

  async aboutToAppear() {
    SystemUiController.useSurface({
      baseColor: DesignConstants.COLOR.SECONDARY_BG_LIGHT,
      preferDarkContent: true
    });
    this.initialised = false;
    this.anchorHistory = AnchorHistoryUtil.getHistory();
  }

  private initFromParams(param: Record<string, Object> | undefined): void {
    const rawMode = (param as LocationPickerParamLike | undefined)?.mode;
    this.pickMode = rawMode === 'location_pick' ? 'location_pick' : 'anchor';
    this.initLocationAsync().catch((e: Error) => {
      hilog.error(DOMAIN_NUMBER, TAG, `初始化位置失败: ${e?.message ?? e}`);
    });
  }

  private async initLocationAsync(): Promise<void> {
    if (this.initialised) {
      return;
    }

    let target: LocationInfo | null = null;

    // 1. 优先使用浏览锚点（已经是 GCJ02 坐标，直接使用，不再转换）
    const anchor = AppStorage.get<LocationInfo | null>(KEY_BROWSE_ANCHOR);
    if (anchor) {
      target = anchor;
    }

    // 2. 没有锚点时，使用当前定位（WGS84 → 需要转换为 GCJ02）
    if (!target && this.currentLocation) {
      const converted = CoordinateConverter.convertWgs84ToGcj02(
        this.currentLocation.latitude, this.currentLocation.longitude
      );
      target = {
        latitude: converted.latitude,
        longitude: converted.longitude,
        accuracy: this.currentLocation.accuracy,
        altitude: this.currentLocation.altitude,
        speed: this.currentLocation.speed,
        bearing: this.currentLocation.bearing,
        timestamp: this.currentLocation.timestamp,
        locationName: this.currentLocation.locationName
      };
    }

    // 3. 都没有时，获取新定位并转换（getConvertedLocation 内部已做 WGS84→GCJ02）
    if (!target) {
      target = await CoordinateConverter.getConvertedLocation();
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
    if (this.renderTimerId > 0) {
      clearTimeout(this.renderTimerId);
      this.renderTimerId = 0;
    }
    if (this.searchDebounceTimerId > 0) {
      clearTimeout(this.searchDebounceTimerId);
      this.searchDebounceTimerId = 0;
    }
  }

  private getActivePanelHeight(): number {
    return this.showSearchBar ? SEARCH_PANEL_HEIGHT : this.panelHeight;
  }

  /** MapKit camera padding：根据面板/搜索面板高度动态计算 */
  private getMapPaddingBottom(): number {
    return this.getActivePanelHeight() + this.safeAreaBottom + 12;
  }

  private getOverlayOffset(): number {
    return this.getMapPaddingBottom();
  }

  private getControlsBottomOffset(): number {
    return this.getOverlayOffset();
  }

  private getLogoPaddingBottom(): number {
    return this.getOverlayOffset();
  }

  private isLocationPickMode(): boolean {
    return this.pickMode === 'location_pick';
  }

  private getConfirmButtonLabel(): string {
    return this.isLocationPickMode() ? '使用此位置' : '设为浏览锚点';
  }

  /** 定位到当前位置 */
  private locateUser(): void {
    if (!this.currentLocation || !this.mapActions) return;
    const converted = CoordinateConverter.convertWgs84ToGcj02(
      this.currentLocation.latitude, this.currentLocation.longitude
    );
    this.mapActions.moveCamera(converted.latitude, converted.longitude);
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

    // 防抖：快速连续拖动时只在最后一次 idle 后请求
    if (this.renderTimerId > 0) {
      clearTimeout(this.renderTimerId);
    }
    this.renderTimerId = setTimeout(() => {
      this.fetchMapRenderData();
    }, 300);
  }

  private collapsePanel(): void {
    if (this.showSearchBar) {
      this.closeSearchPanel(true);
      return;
    }
    if (this.panelHeight > PANEL_SNAP_MIN) {
      this.getUIContext().animateTo(
        { duration: 250, curve: curves.springMotion(0.6, 0.9) },
        () => { this.panelHeight = PANEL_SNAP_MIN; }
      );
    }
  }

  private onMapDoubleClick(coord: MapCoordinate): void {
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

  private openSearchPanel(): void {
    this.searchText = '';
    this.searchResults = [];
    this.getUIContext().animateTo({ duration: 250, curve: Curve.FastOutSlowIn }, () => {
      this.showSearchBar = true;
    });
    setTimeout(() => {
    this.getUIContext().getFocusController().requestFocus('locationPickerSearchInput');
    }, 300);
  }

  private closeSearchPanel(collapseToMin: boolean = false): void {
    this.getUIContext().animateTo(
      { duration: 300, curve: Curve.FastOutSlowIn },
      () => {
        this.showSearchBar = false;
        this.searchText = '';
        this.searchResults = [];
        this.isSearching = false;
      }
    );

    if (collapseToMin) {
      setTimeout(() => {
        this.getUIContext().animateTo(
          { duration: 300, curve: curves.springMotion(0.6, 0.9) },
          () => { this.panelHeight = PANEL_SNAP_MIN; }
        );
      }, 350);
    }
  }

  private clearAnchorHistory(): void {
    AnchorHistoryUtil.clearHistory();
    this.anchorHistory = [];
  }

  private selectSearchResult(result: LocationInfo): void {
    this.skipNextResolve = true;
    this.setCenter(result.latitude, result.longitude);
    this.locationName = result.locationName || '';
    if (this.mapActions) {
      this.mapActions.moveCamera(result.latitude, result.longitude);
    }
    this.closeSearchPanel();
  }

  private selectHistoryItem(item: AnchorHistoryItem): void {
    this.skipNextResolve = true;
    this.setCenter(item.latitude, item.longitude);
    this.locationName = item.locationName || '历史位置';
    if (this.mapActions) {
      this.mapActions.moveCamera(item.latitude, item.longitude);
    }
    this.resolveLocation(item.latitude, item.longitude);
  }

  private confirm(): void {
    if (this.locationName.length === 0 || this.isLoading) {
      SystemToast.show('请等待位置识别完成', this.getUIContext());
      return;
    }
    const location: LocationInfo = {
      latitude: this.centerLat,
      longitude: this.centerLng,
      locationName: this.locationName
    };
    AnchorHistoryUtil.addToHistory(this.centerLat, this.centerLng, this.locationName);

    if (this.isLocationPickMode()) {
      AppStorage.setOrCreate<LocationInfo | null>(KEY_LOCATION_PICK_RESULT, location);
      SystemToast.showSuccess('已使用当前选中位置', this.getUIContext());
      AppRouter.pop();
      return;
    }

    AppStorage.setOrCreate<LocationInfo | null>(KEY_BROWSE_ANCHOR, location);
    SystemToast.showSuccess('浏览锚点已更新', this.getUIContext());
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
    try {
      const request: MapRenderRequest = { viewport };
      const response = await ApiClient.renderMap(request);
      if (response.code === 0) {
        this.renderItems = response.data?.items ?? [];
        this.mapToastMessage = response.data?.toast_message ?? '';
        this.updateMarkers();
      }
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, `render map failed: ${JSON.stringify(error)}`);
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
        return this.showUsers || this.showGroups || this.showMoments;
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
      Stack({ alignContent: Alignment.Bottom }) {
        // 底层：全屏地图（始终占满所有空间）
        NativeMapComponent({
          mapRegion: this.mapRegion,
          markers: this.mapMarkers,
          showCompass: false,
          showScale: false,
          showZoomControls: false,
          showMyLocationControls: false,
          showCenterPin: true,
          contentPaddingTop: SafeAreaUtils.resolveTopPadding(this.safeAreaTop, 110, 20),
          contentPaddingBottom: this.getMapPaddingBottom(),
          logoPaddingBottom: this.getLogoPaddingBottom(),
          onCameraIdle: (camera: MapCameraPosition) => { this.onCameraIdle(camera); },
          onMapClick: () => { this.collapsePanel(); },
          onMapDoubleClick: (coord: MapCoordinate) => { this.onMapDoubleClick(coord); },
          onMarkerClick: (marker: MapMarker) => { this.onMarkerItemClick(marker); },
          onActionsReady: (actions: NativeMapActions) => { this.mapActions = actions; }
        })
          .width('100%')
          .height('100%')

        // 地图 Toast 提示
        MapToast({ message: this.mapToastMessage, safeAreaTop: this.safeAreaTop })

        // TopBar 浮在地图顶部
        Column() {
          TopBar({
            showUsers: this.showUsers,
            showGroups: this.showGroups,
            showMoments: this.showMoments,
            safeAreaTop: this.safeAreaTop,
            onBack: () => { AppRouter.pop(); },
            onToggleUsers: () => {
              this.showUsers = !this.showUsers;
              this.updateMarkers();
            },
            onToggleGroups: () => {
              this.showGroups = !this.showGroups;
              this.updateMarkers();
            },
            onToggleMoments: () => {
              this.showMoments = !this.showMoments;
              this.updateMarkers();
            }
          })
          Blank()
        }
        .width('100%')
        .height('100%')
        .hitTestBehavior(HitTestMode.Transparent)

        // 底部面板
        if (this.showSearchBar) {
          SearchPanel({
            searchText: this.searchText,
            isSearching: this.isSearching,
            searchResults: this.searchResults,
            anchorHistory: this.anchorHistory,
            onSearch: (): void => { this.doSearch(); },
            onClose: (): void => this.closeSearchPanel(),
            onClearHistory: (): void => this.clearAnchorHistory(),
            onSelectResult: (result): void => this.selectSearchResult(result),
            onSelectHistory: (item): void => this.selectHistoryItem(item),
            onTextChange: (value): void => { this.searchText = value; }
          })
            .transition(TransitionEffect.asymmetric(
              TransitionEffect.move(TransitionEdge.BOTTOM)
                .combine(TransitionEffect.OPACITY)
                .animation({ duration: 300, curve: Curve.FastOutSlowIn }),
              TransitionEffect.OPACITY
                .animation({ duration: 200, curve: Curve.FastOutSlowIn })
            ))
        } else {
          MainPanel({
            anchorHistory: this.anchorHistory,
            panelHeight: this.panelHeight,
            confirmLabel: this.getConfirmButtonLabel(),
            onSearchClick: (): void => this.openSearchPanel(),
            onClearHistory: (): void => this.clearAnchorHistory(),
            onSelectHistory: (item): void => {
              this.selectHistoryItem(item);
            },
            onConfirm: (): void => this.confirm()
          })
            .transition(TransitionEffect.asymmetric(
              TransitionEffect.OPACITY
                .combine(TransitionEffect.translate({ y: 30 }))
                .animation({ duration: 300, delay: 150, curve: Curve.EaseOut }),
              TransitionEffect.OPACITY
                .animation({ duration: 150 })
            ))
        }

        // 自定义地图控件：用 position(Edges) 定位到右下角，仅占按钮实际面积，
        // 既不阻塞地图拖拽，又能独占按钮区域的点击事件
        Column({ space: 8 }) {
          Column() {
            Text() {
              SymbolSpan($r(AppIcon.LOCATION))
                .fontSize(20)
                .fontColor([$r('sys.color.ohos_id_color_text_primary')])
            }
          }
          .width(44)
          .height(44)
          .justifyContent(FlexAlign.Center)
          .backgroundColor($r('app.color.bg_secondary'))
          .borderRadius(22)
          .shadow({ radius: 8, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 2 })
          .onClick(() => { this.locateUser(); })

          Column() {
            Text() {
              SymbolSpan($r(AppIcon.ADD))
                .fontSize(20)
                .fontColor([$r('sys.color.ohos_id_color_text_primary')])
            }
          }
          .width(44)
          .height(44)
          .justifyContent(FlexAlign.Center)
          .backgroundColor($r('app.color.bg_secondary'))
          .borderRadius(22)
          .shadow({ radius: 8, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 2 })
          .onClick(() => { if (this.mapActions) this.mapActions.zoomIn(); })

          Column() {
            Text() {
              SymbolSpan($r(AppIcon.REMOVE))
                .fontSize(20)
                .fontColor([$r('sys.color.ohos_id_color_text_primary')])
            }
          }
          .width(44)
          .height(44)
          .justifyContent(FlexAlign.Center)
          .backgroundColor($r('app.color.bg_secondary'))
          .borderRadius(22)
          .shadow({ radius: 8, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 2 })
          .onClick(() => { if (this.mapActions) this.mapActions.zoomOut(); })
        }
        .position({ right: 12, bottom: this.getControlsBottomOffset() })
      }
    }
    .hideTitleBar(true)
    .onShown(() => { if (this.mapActions) { this.mapActions.show(); } })
    .onHidden(() => { if (this.mapActions) { this.mapActions.hide(); } })
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
      const param = context.pathInfo.param as Record<string, Object>;
      this.initFromParams(param);
    })
  }
}

interface MapFilterChipProps {
  active: boolean;
  label: string;
  iconSymbol?: Resource;
  accentColor: string;
  onToggle: () => void;
}

@Builder
function MapFilterChip(props: MapFilterChipProps) {
  Row({ space: 6 }) {
    if (props.iconSymbol) {
      Text() {
        SymbolSpan(props.iconSymbol)
          .fontSize(14)
          .fontColor([props.active ? Color.White : props.accentColor])
      }
    }
    Text(props.label)
      .fontSize(DesignConstants.FONT.SIZE.SM)
      .fontWeight(FontWeight.Medium)
      .fontColor(props.active ? Color.White : $r('sys.color.ohos_id_color_text_primary'))
  }
  .height(34)
  .padding({ left: 12, right: 14 })
  .backgroundColor(props.active ? props.accentColor : $r('app.color.bg_secondary'))
  .border(props.active ? { width: 0 } : { width: 1.5, color: props.accentColor })
  .borderRadius(17)
  .shadow({ radius: 4, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 1 })
  .onClick(() => props.onToggle())
  .animation({ duration: 200 })
}

@Builder
export function LocationPickerPageBuilder() {
  LocationPickerPage()
}
