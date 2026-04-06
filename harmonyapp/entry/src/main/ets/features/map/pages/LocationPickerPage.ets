import { KEY_BROWSE_ANCHOR, KEY_CURRENT_LOCATION, KEY_LOCATION_PICK_RESULT } from '../../../core/state/GlobalStateManager';
import { LocationInfo, LocationUtil } from '../../../core/location/LocationUtil';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { AnchorHistoryItem, AnchorHistoryUtil } from '../../../core/utils/AnchorHistoryUtil';
import { NativeMapComponent, NativeMapActions, MapRegion, MapCoordinate, MapMarker, MapCameraPosition, MapVisibleRegion } from '../../../components/NativeMapComponent';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { curves } from '@kit.ArkUI';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { ApiClient } from '../../../core/api/ApiClient';
import { Bounds, MapCanvasFilter, MapCanvasRequest, MapCanvasResponse, MapHeatBucket, MapHotspotItem, MapViewport, PaginationParams } from '../../../core/types/api';
import { AppRouter, GroupDetailRouteParam, ProfileRouteParam, MomentDetailRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { CoordinateBounds, CoordinateConverter } from '../../../core/geo/CoordinateConverter';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';

const DOMAIN_NUMBER: number = 0xFF13;
const TAG: string = 'LocationPickerPage';
const SEARCH_PANEL_HEIGHT: number = 440;
const MAX_HISTORY_ITEMS: number = 5;
// 底部面板三段式吸附位置（内容高度，不含 safeAreaBottom）
const PANEL_SNAP_MIN: number = 120;       // 收起：手柄 + 搜索栏 + 确认按钮
const PANEL_SNAP_HALF: number = 224;      // 半展开：显示部分历史记录
const PANEL_SNAP_MAX: number = 392;       // 全展开：完整内容
const DEFAULT_GESTURE_HINT: string = '长按地图即可快速选点';
const DOUBLE_CLICK_PLACEMENT_HINT: string = '已为你定位到该位置';
const LONG_PRESS_PLACEMENT_HINT: string = '已为你定位到该位置';
const ANCHOR_PLACEMENT_HINT: string = '已回到浏览锚点位置';
const GENERIC_PLACEMENT_HINT: string = '已定位到该位置';
const MARKER_COLOR_PICKER: string = DesignConstants.MAP_MARKER.PICKER;
const MARKER_COLOR_USER: string = DesignConstants.MAP_MARKER.USER;
const MARKER_COLOR_GROUP: string = DesignConstants.MAP_MARKER.GROUP;
const MARKER_COLOR_MOMENT: string = DesignConstants.MAP_MARKER.MOMENT;

type MapRenderItemType = 'heat_bucket' | 'user' | 'group' | 'moment';

interface MapRenderItemLocation {
	latitude: number;
	longitude: number;
}

interface MapRenderEntity {
	payload_id: string;
	title: string;
	subtitle?: string;
	type: 'user' | 'group' | 'moment';
	distance_meters?: number;
}

interface MapRenderHeatBucket {
  count: number;
  breakdown?: string;
}

interface MapRenderItem {
	render_id: string;
	location: MapRenderItemLocation;
	type: MapRenderItemType;
	total_count: number;
	entity?: MapRenderEntity;
  heatBucket?: MapRenderHeatBucket;
}

interface ConvertedRenderItem {
	item: MapRenderItem;
	gcj02Lat: number;
	gcj02Lng: number;
}

interface BoundsDelta {
  latDelta: number;
  lonDelta: number;
}

type LocationPickerMode = 'anchor' | 'location_pick';

interface LocationPickerParamLike {
  mode?: string;
}

interface TopBarProps {
  filterOpen: boolean;
  safeAreaTop: number;
  resolvedThemeId?: string;
  onBack: () => void;
  onToggleFilter: () => void;
}

function resolveLocationPickerPalette(themeId?: string) {
  return AppThemeManager.getPaletteById(themeId ?? 'default_clean_light');
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
    .borderRadius(DesignConstants.RADIUS.FULL)
    .backgroundColor(resolveLocationPickerPalette(props.resolvedThemeId).surfaceSubtle)
    .onClick(props.onBack)

    Row({ space: 6 }) {
      Text() {
        SymbolSpan($r(AppIcon.FILTER))
          .fontSize(16)
          .fontColor([$r('sys.color.ohos_id_color_text_primary')])
      }
      Text('筛选')
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontWeight(FontWeight.Medium)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
    }
    .height(40)
    .padding({
      left: DesignConstants.SPACING.MD,
      right: DesignConstants.SPACING.MD
    })
    .backgroundColor(props.filterOpen
      ? resolveLocationPickerPalette(props.resolvedThemeId).surfaceSubtle
      : resolveLocationPickerPalette(props.resolvedThemeId).surfaceBackground)
    .borderRadius(DesignConstants.RADIUS.FULL)
    .shadow(DesignConstants.SHADOW.SMALL)
    .border({ width: 1, color: resolveLocationPickerPalette(props.resolvedThemeId).borderColor })
    .onClick(props.onToggleFilter)
  }
  .width('100%')
  .padding({
    left: DesignConstants.SPACING.MD,
    right: DesignConstants.SPACING.MD,
    top: SafeAreaUtils.resolveTopPadding(props.safeAreaTop, DesignConstants.SPACING.XS)
  })
  .onClick(() => {})
}

interface FilterMenuItemProps {
  markerSvg: string;
  label: string;
  accentColor: string;
  active: boolean;
  resolvedThemeId?: string;
  onToggle: () => void;
}

@Builder
function FilterMenuItem(props: FilterMenuItemProps) {
  Row({ space: DesignConstants.SPACING.MD }) {
    Image($rawfile(props.markerSvg))
      .width(24)
      .height(32)
      .objectFit(ImageFit.Contain)
    Text(props.label)
      .fontSize(DesignConstants.FONT.SIZE.MD)
      .fontWeight(props.active ? FontWeight.Medium : FontWeight.Normal)
      .fontColor(props.active
        ? resolveLocationPickerPalette(props.resolvedThemeId).textPrimary
        : resolveLocationPickerPalette(props.resolvedThemeId).textSecondary)
      .layoutWeight(1)
    Row()
      .width(14)
      .height(14)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(props.active ? props.accentColor : Color.Transparent)
      .border({ width: 1.5, color: props.accentColor })
  }
  .width('100%')
  .height(44)
  .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD })
  .justifyContent(FlexAlign.Center)
  .alignItems(VerticalAlign.Center)
  .backgroundColor(props.active
    ? resolveLocationPickerPalette(props.resolvedThemeId).surfaceSubtle
    : Color.Transparent)
  .borderRadius(DesignConstants.RADIUS.LG)
  .opacity(props.active ? 1.0 : DesignConstants.OPACITY.SUBTLE)
  .onClick(() => props.onToggle())
}

interface FilterPanelProps {
  showUsers: boolean;
  showGroups: boolean;
  showMoments: boolean;
  resolvedThemeId?: string;
  onToggleUsers: () => void;
  onToggleGroups: () => void;
  onToggleMoments: () => void;
}

@Builder
function FilterDropdownPanel(props: FilterPanelProps) {
  Column({ space: DesignConstants.SPACING.XS }) {
    FilterMenuItem({
      markerSvg: 'marker_user.svg',
      label: '用户',
      accentColor: MARKER_COLOR_USER,
      active: props.showUsers,
      resolvedThemeId: props.resolvedThemeId,
      onToggle: props.onToggleUsers
    })
    FilterMenuItem({
      markerSvg: 'marker_group.svg',
      label: '群组',
      accentColor: MARKER_COLOR_GROUP,
      active: props.showGroups,
      resolvedThemeId: props.resolvedThemeId,
      onToggle: props.onToggleGroups
    })
    FilterMenuItem({
      markerSvg: 'marker_moment.svg',
      label: '动态',
      accentColor: MARKER_COLOR_MOMENT,
      active: props.showMoments,
      resolvedThemeId: props.resolvedThemeId,
      onToggle: props.onToggleMoments
    })
  }
  .width(180)
  .padding(DesignConstants.SPACING.SM)
  .backgroundColor(resolveLocationPickerPalette(props.resolvedThemeId).surfaceBackground)
  .borderRadius(DesignConstants.SHAPE.CARD)
  .shadow(DesignConstants.SHADOW.MEDIUM)
  .border({ width: 1, color: resolveLocationPickerPalette(props.resolvedThemeId).borderColor })
  .onClick(() => {})
}

interface MapToastProps {
  message: string;
  safeAreaTop: number;
  resolvedThemeId?: string;
}

@Builder
function MapToast(props: MapToastProps) {
  if (props.message.length === 0) {
    // 不渲染
  } else {
    Text(props.message)
      .fontSize(DesignConstants.FONT.SIZE.SM)
      .fontColor(resolveLocationPickerPalette(props.resolvedThemeId).textPrimary)
      .backgroundColor(resolveLocationPickerPalette(props.resolvedThemeId).surfaceElevated)
      .borderRadius(DesignConstants.RADIUS.XL)
      .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: 8, bottom: 8 })
      .shadow(DesignConstants.SHADOW.SMALL)
      .position({ top: SafeAreaUtils.resolveTopPadding(props.safeAreaTop, 76), left: 20, right: 20 })
      .onClick(() => {})
      .transition(
        TransitionEffect.OPACITY.animation({ duration: 280, curve: Curve.FastOutSlowIn })
          .combine(TransitionEffect.move(TransitionEdge.TOP).animation({ duration: 280, curve: Curve.FastOutSlowIn }))
      )
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
  controller: SearchController = new SearchController();
  onSearch: () => void = () => {};
  onClose: () => void = () => {};
  onClearHistory: () => void = () => {};
  onSelectResult: (result: LocationInfo) => void = (result) => {};
  onSelectHistory: (item: AnchorHistoryItem) => void = (item) => {};
  onTextChange: (value: string) => void = (value) => {};
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Column() {
      // 搜索行 - 使用官方 Search 组件
      Row({ space: DesignConstants.SPACING.SM }) {
        // 官方 Search 组件（全局内置，无需导入）
        Search({
          value: this.searchText,
          placeholder: '搜索地点、地标或坐标',
          controller: this.controller
        })
          .id('locationPickerSearchInput')
          .layoutWeight(1)
          .height(40)
          .backgroundColor(this.getPalette().surfaceSubtle)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .shadow(DesignConstants.SHADOW.INSET)
          .searchIcon({
            size: 18,
            color: this.getPalette().textPrimary
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
        .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM })
        .onClick(this.onClose)
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)
      .padding({ bottom: DesignConstants.SPACING.MD })

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
                .margin({ left: DesignConstants.SPACING.SM })
            }
            .width('100%')
            .height(80)
            .justifyContent(FlexAlign.Center)
          } else if (this.searchResults.length > 0) {
            // 搜索结果列表
            List({ space: DesignConstants.SPACING.SM }) {
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
                    .backgroundColor(this.getPalette().surfaceSubtle)
                    .borderRadius(DesignConstants.RADIUS.MD)

                    Text(item.locationName || '未知地点')
                      .fontSize(DesignConstants.FONT.SIZE.MD)
                      .fontColor($r('sys.color.ohos_id_color_text_primary'))
                      .layoutWeight(1)
                      .maxLines(2)
                      .textOverflow({ overflow: TextOverflow.Ellipsis })
                  }
                  .width('100%')
                  .padding({
                    left: DesignConstants.SPACING.MD,
                    right: DesignConstants.SPACING.MD,
                    top: DesignConstants.SPACING.MD,
                    bottom: DesignConstants.SPACING.MD
                  })
                  .backgroundColor(this.getPalette().surfaceBackground)
                  .borderRadius(DesignConstants.RADIUS.XXL)
                  .shadow({
                    radius: DesignConstants.SHADOW.MEDIUM.radius,
                    color: DesignConstants.SHADOW.MEDIUM.color,
                    offsetX: DesignConstants.SHADOW.MEDIUM.offsetX,
                    offsetY: DesignConstants.SHADOW.MEDIUM.offsetY
                  })
                  .border({ width: 1, color: this.getPalette().borderColor })
                  .onClick(() => { this.onSelectResult(item); })
                }
              }, (item: LocationInfo) => `search_${item.latitude}_${item.longitude}`)
            }
            .width('100%')
            .divider({ strokeWidth: 0, color: $r('sys.color.ohos_id_color_list_separator') })
          } else if (this.searchText.trim().length > 0) {
            // 无搜索结果
            Column() {
              Text('未找到该地点')
                .fontSize(DesignConstants.FONT.SIZE.MD)
                .fontColor($r('sys.color.ohos_id_color_text_primary'))
              Text('请换个关键词试试')
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                .margin({ top: DesignConstants.SPACING.XS })
            }
            .width('100%')
            .height(120)
            .justifyContent(FlexAlign.Center)
            .alignItems(HorizontalAlign.Center)
          } else if (this.anchorHistory.length > 0) {
            // 历史锚点列表（搜索未输入时展示）
            Column({ space: DesignConstants.SPACING.SM }) {
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

              List({ space: DesignConstants.SPACING.SM }) {
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
                      .backgroundColor(this.getPalette().surfaceSubtle)
                      .borderRadius(DesignConstants.RADIUS.MD)

                      Text(item.locationName || '未命名位置')
                        .fontSize(DesignConstants.FONT.SIZE.MD)
                        .fontColor($r('sys.color.ohos_id_color_text_primary'))
                        .layoutWeight(1)
                        .maxLines(1)
                        .textOverflow({ overflow: TextOverflow.Ellipsis })
                    }
                    .width('100%')
                    .padding({
                      left: DesignConstants.SPACING.MD,
                      right: DesignConstants.SPACING.MD,
                      top: DesignConstants.SPACING.MD,
                      bottom: DesignConstants.SPACING.MD
                    })
                    .backgroundColor(this.getPalette().surfaceBackground)
                    .borderRadius(DesignConstants.RADIUS.XXL)
                    .shadow({
                      radius: DesignConstants.SHADOW.MEDIUM.radius,
                      color: DesignConstants.SHADOW.MEDIUM.color,
                      offsetX: DesignConstants.SHADOW.MEDIUM.offsetX,
                      offsetY: DesignConstants.SHADOW.MEDIUM.offsetY
                    })
                    .border({ width: 1, color: this.getPalette().borderColor })
                    .onClick(() => {
                      this.onSelectHistory(item);
                      this.onClose();
                    })
                  }
                }, (item: AnchorHistoryItem) => `hist_s_${item.latitude}_${item.longitude}`)
              }
              .width('100%')
              .divider({ strokeWidth: 0, color: $r('sys.color.ohos_id_color_list_separator') })
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
    .height(SEARCH_PANEL_HEIGHT + this.safeAreaBottom)
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.LG,
      bottom: this.safeAreaBottom
    })
    .backgroundColor(this.getPalette().surfaceBackground)
    .border({ width: { top: 1 }, color: $r('app.color.divider') })
    .borderRadius({ topLeft: 24, topRight: 24 })
    .onClick(() => {})
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
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';

  private dragStartHeight: number = 0;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

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
          .borderRadius(DesignConstants.RADIUS.SM)
          .backgroundColor(this.getPalette().surfaceSubtle)
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
      Row({ space: DesignConstants.SPACING.SM }) {
        Text() {
          SymbolSpan($r(AppIcon.SEARCH))
            .fontSize(DesignConstants.FONT.SIZE.LG)
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
      .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD })
      .alignItems(VerticalAlign.Center)
      .backgroundColor(this.getPalette().surfaceSubtle)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .shadow(DesignConstants.SHADOW.INSET)
      .margin({ bottom: DesignConstants.SPACING.MD })
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
                  .padding({ left: DesignConstants.SPACING.SM, top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
                  .onClick(this.onClearHistory)
              }
            }
            .width('100%')
            .margin({ bottom: DesignConstants.SPACING.SM })

            if (this.anchorHistory.length > 0) {
              List({ space: DesignConstants.SPACING.SM }) {
                ForEach(this.anchorHistory.slice(0, MAX_HISTORY_ITEMS), (item: AnchorHistoryItem, index: number) => {
                  ListItem() {
                    Row({ space: DesignConstants.SPACING.MD }) {
                      Column() {
                        Text() {
                          SymbolSpan($r(AppIcon.TIME))
                            .fontSize(DesignConstants.FONT.SIZE.LG)
                            .fontColor([$r('sys.color.ohos_id_color_text_tertiary')])
                        }
                      }
                      .width(32)
                      .height(32)
                      .justifyContent(FlexAlign.Center)
                      .backgroundColor(this.getPalette().surfaceSubtle)
                      .borderRadius(DesignConstants.RADIUS.MD)

                      Text(item.locationName || '未命名位置')
                        .fontSize(DesignConstants.FONT.SIZE.MD)
                        .fontColor($r('sys.color.ohos_id_color_text_primary'))
                        .layoutWeight(1)
                        .maxLines(1)
                        .textOverflow({ overflow: TextOverflow.Ellipsis })
                    }
                    .width('100%')
                    .padding({
                      left: DesignConstants.SPACING.MD,
                      right: DesignConstants.SPACING.MD,
                      top: DesignConstants.SPACING.MD,
                      bottom: DesignConstants.SPACING.MD
                    })
                    .backgroundColor(this.getPalette().surfaceBackground)
                    .borderRadius(DesignConstants.RADIUS.XXL)
                    .shadow({
                      radius: DesignConstants.SHADOW.MEDIUM.radius,
                      color: DesignConstants.SHADOW.MEDIUM.color,
                      offsetX: DesignConstants.SHADOW.MEDIUM.offsetX,
                      offsetY: DesignConstants.SHADOW.MEDIUM.offsetY
                    })
                     .border({ width: 1, color: this.getPalette().borderColor })
                    .onClick(() => { this.onSelectHistory(item); })
                  }
                }, (item: AnchorHistoryItem) => `hist_m_${item.latitude}_${item.longitude}`)
              }
              .width('100%')
              .divider({ strokeWidth: 0, color: $r('sys.color.ohos_id_color_list_separator') })
            } else {
              Column() {
                Text() {
                  SymbolSpan($r(AppIcon.TIME))
                    .fontSize(DesignConstants.FONT.SIZE.LG)
                    .fontColor([$r('sys.color.ohos_id_color_text_tertiary')])
                }
                Text('暂无最近使用的地点')
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor($r('sys.color.ohos_id_color_text_tertiary'))
                  .margin({ top: DesignConstants.SPACING.XS })
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

      Button(this.confirmLabel)
        .type(ButtonType.Capsule)
        .width('100%')
        .height(48)
        .backgroundColor(this.getPalette().accentBackground)
        .fontColor(this.getPalette().accentText)
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .fontWeight(FontWeight.Medium)
        .shadow(DesignConstants.SHADOW.SMALL)
        .border({ width: 1, color: this.getPalette().borderColor })
        .margin({ top: DesignConstants.SPACING.SM })
        .onClick(this.onConfirm)
    }
    .height(this.panelHeight + this.safeAreaBottom)
    .clip(true)
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: 0,
      bottom: this.safeAreaBottom
    })
    .backgroundColor(this.getPalette().surfaceBackground)
    .border({ width: { top: 1 }, color: $r('app.color.divider') })
    .borderRadius({ topLeft: 24, topRight: 24 })
    .onClick(() => {})
  }
}

@Preview
@Component
export struct LocationPickerPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';

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
  private visibleRegion: MapVisibleRegion | null = null;
  @State showUsers: boolean = true;
  @State showGroups: boolean = true;
  @State showMoments: boolean = true;
	@State showFilterPanel: boolean = false;
	@State mapToastMessage: string = '';
	@State overlappingPanelItems: MapRenderItem[] = [];
  private mapToastTimerId: number = 0;
  @State anchorHistory: AnchorHistoryItem[] = [];
  @State pickMode: LocationPickerMode = 'anchor';

  // 底部面板高度（仅在主面板时使用，初始为收起位置）
  @State panelHeight: number = PANEL_SNAP_MIN;

	private renderItemsMap: Map<string, MapRenderItem> = new Map();
  private resolveTimerId: number = 0;
  private renderTimerId: number = 0;
  private renderRequestId: number = 0;
  private resolveRequestId: number = 0;
  private isFirstCameraIdle: boolean = true;
  private skipNextResolve: boolean = false;
  private searchDebounceTimerId: number = 0;
  private mapZoom: number = 15;
  private initialised: boolean = false;
  private mapReady: boolean = false;
	private renderItems: MapRenderItem[] = [];
  private lastViewportSignature: string = '';
  private mapActions: NativeMapActions | null = null;
  private hasShownGestureHint: boolean = false;
  private searchController: SearchController = new SearchController();
  private lastCameraLat: number = NaN;
  private lastCameraLng: number = NaN;

  async aboutToAppear() {
    SystemUiController.syncWithCurrentTheme();
    this.initialised = false;
    this.mapReady = false;
    this.visibleRegion = null;
    this.lastViewportSignature = '';
    this.anchorHistory = AnchorHistoryUtil.getHistory();
  }

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
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

    // 1. 优先使用浏览锚点（WGS84 坐标，需要转换为 GCJ02 供地图使用）
    const anchor = AppStorage.get<LocationInfo | null>(KEY_BROWSE_ANCHOR);
    if (anchor) {
      const converted = CoordinateConverter.convertWgs84ToGcj02(
        anchor.latitude, anchor.longitude
      );
      target = {
        latitude: converted.latitude,
        longitude: converted.longitude,
        locationName: anchor.locationName
      };
    }

    // 2. 没有锚点时，使用当前定位（WGS84 → 转换为 GCJ02）
    if (!target && this.currentLocation) {
      const converted = CoordinateConverter.convertWgs84ToGcj02(
        this.currentLocation.latitude, this.currentLocation.longitude
      );
      target = {
        latitude: converted.latitude,
        longitude: converted.longitude,
        locationName: this.currentLocation.locationName
      };
    }

    // 3. 都没有时，获取新定位并转换（getConvertedLocation 内部已做 WGS84→GCJ02）
    if (!target) {
      target = await CoordinateConverter.getConvertedLocation();
    }

    if (target) {
      this.setCenter(target.latitude, target.longitude, true);
      // 初始化时立即在地图上显示选中图钉
      this.updateMarkers();
      await this.resolveLocation(target.latitude, target.longitude);
    } else {
      hilog.warn(DOMAIN_NUMBER, TAG, '无法获取当前位置');
    }

    this.initialised = true;
    this.showInitialGestureHint();
    await this.fetchMapRenderData('init_location');
  }

  private showInitialGestureHint(): void {
    if (this.hasShownGestureHint) {
      return;
    }
    this.hasShownGestureHint = true;
    this.showMapToast(DEFAULT_GESTURE_HINT, 3500);
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
    if (this.mapToastTimerId > 0) {
      clearTimeout(this.mapToastTimerId);
      this.mapToastTimerId = 0;
    }
  }

  private getActivePanelHeight(): number {
    const contentHeight = this.showSearchBar ? SEARCH_PANEL_HEIGHT : this.panelHeight;
    return contentHeight + this.safeAreaBottom;
  }

  /** MapKit camera padding：根据面板/搜索面板高度动态计算 */
  private getMapPaddingBottom(): number {
    return this.getActivePanelHeight() + 8;
  }

  private getOverlayOffset(): number {
    return this.getMapPaddingBottom();
  }

  private getControlsBottomOffset(): number {
    return this.getOverlayOffset() + 12;
  }

  private getLogoPaddingBottom(): number {
    return this.getOverlayOffset() + 8;
  }

  private isLocationPickMode(): boolean {
    return this.pickMode === 'location_pick';
  }

  private getConfirmButtonLabel(): string {
    return this.isLocationPickMode() ? '使用此位置' : '设为浏览锚点';
  }

  private applyMapRegion(lat: number, lng: number): void {
    this.mapRegion = { latitude: lat, longitude: lng, latitudeDelta: 0.01, longitudeDelta: 0.01 };
  }

  private moveCameraSmoothly(lat: number, lng: number): void {
    if (this.mapActions) {
      this.mapActions.moveCamera(lat, lng);
      return;
    }
    // 地图控制器尚未就绪时，通过更新 region 兜底
    this.applyMapRegion(lat, lng);
  }

  /** 定位到当前位置 */
  private locateUser(): void {
    if (!this.currentLocation) return;
    const converted = CoordinateConverter.convertWgs84ToGcj02(
      this.currentLocation.latitude, this.currentLocation.longitude
    );
    this.setCenter(converted.latitude, converted.longitude);
    this.updateMarkers();
    this.moveCameraSmoothly(converted.latitude, converted.longitude);
    // Harmony 端对齐 Web：显式动作才更新选中位置并解析名称
    this.resolveLocation(converted.latitude, converted.longitude).catch((e: Error) => {
      hilog.warn(DOMAIN_NUMBER, TAG, `locateUser 解析位置失败: ${e?.message ?? e}`);
    });
  }

  /** 回到当前浏览锚点位置（与“回到当前位置”区分） */
  private async locateBrowseAnchor(): Promise<void> {
    const anchor = AppStorage.get<LocationInfo | null>(KEY_BROWSE_ANCHOR);
    if (!anchor) {
      SystemToast.show('暂无浏览锚点，请先设置锚点', this.getUIContext());
      return;
    }
    const gcj02 = CoordinateConverter.convertWgs84ToGcj02(anchor.latitude, anchor.longitude);
    await this.focusOnCoordinate(gcj02.latitude, gcj02.longitude, ANCHOR_PLACEMENT_HINT);
  }

  private setCenter(lat: number, lng: number, syncRegion: boolean = false): void {
    this.centerLat = lat;
    this.centerLng = lng;
    if (syncRegion) {
      this.applyMapRegion(lat, lng);
    }
  }

  private async focusOnCoordinate(lat: number, lng: number, hintMessage: string = GENERIC_PLACEMENT_HINT): Promise<void> {
    this.skipNextResolve = true;
    this.setCenter(lat, lng);
    this.updateMarkers();
    this.moveCameraSmoothly(lat, lng);
    await this.resolveLocation(lat, lng);
    this.showMapToast(hintMessage);
  }

  private onCameraIdle(camera: MapCameraPosition): void {
    if (this.isFirstCameraIdle) {
      this.isFirstCameraIdle = false;
    }

    const prevZoom = this.mapZoom;
    const currentZoom = camera.zoom !== undefined ? camera.zoom : prevZoom;
    this.mapZoom = currentZoom;

		// 仅用于判断是否需要触发地图热力渲染，拖动/缩放都不会修改选中位置 centerLat/centerLng
    this.lastCameraLat = camera.latitude;
    this.lastCameraLng = camera.longitude;

    if (this.skipNextResolve) {
      // 下次相机空闲仅用于消费 skip 标记，不再自动解析位置
      this.skipNextResolve = false;
    }

		if (this.initialised && this.mapReady) {
			if (this.renderTimerId > 0) {
				clearTimeout(this.renderTimerId);
			}
			const timerId: number = setTimeout(() => {
				this.fetchMapRenderData('camera_idle');
			}, 150) as number;
			this.renderTimerId = timerId;
		}

  }

  private handleMapReady(): void {
    this.mapReady = true;
    if (this.initialised) {
      this.fetchMapRenderData('map_ready');
    }
  }

  private collapsePanel(): void {
    this.overlappingPanelItems = [];
    if (this.showFilterPanel) {
      this.showFilterPanel = false;
    }
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

  private handleBackNavigation(): boolean {
    if (this.overlappingPanelItems.length > 0) {
      this.overlappingPanelItems = [];
      return true;
    }
    if (this.showFilterPanel) {
      this.showFilterPanel = false;
      return true;
    }
    if (this.showSearchBar) {
      this.closeSearchPanel(true);
      return true;
    }
    AppRouter.pop();
    return true;
  }

  private handleVisibleRegionChange(region: MapVisibleRegion): void {
    this.visibleRegion = region;
    if (!this.initialised || !this.mapReady) {
      return;
    }
    if (this.renderTimerId > 0) {
      clearTimeout(this.renderTimerId);
    }
    const timerId: number = setTimeout(() => {
      this.fetchMapRenderData('visible_region_change');
    }, 150) as number;
    this.renderTimerId = timerId;
  }

  private async onMapDoubleClick(coord: MapCoordinate): Promise<void> {
    await this.focusOnCoordinate(coord.latitude, coord.longitude, DOUBLE_CLICK_PLACEMENT_HINT);
  }

  private async onMapLongPress(coord: MapCoordinate): Promise<void> {
    await this.focusOnCoordinate(coord.latitude, coord.longitude, LONG_PRESS_PLACEMENT_HINT);
  }

  private async resolveLocation(lat: number, lng: number): Promise<void> {
    const requestId = ++this.resolveRequestId;
    this.isLoading = true;
    try {
      const wgs84 = CoordinateConverter.convertGcj02ToWgs84(lat, lng);
      const addr = await LocationUtil.reverseGeocode(wgs84.latitude, wgs84.longitude);
      if (requestId !== this.resolveRequestId) {
        return;
      }
      this.locationName = addr?.fullAddress || '未知位置';
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, `解析位置失败: ${JSON.stringify(error)}`);
      if (requestId === this.resolveRequestId && this.locationName.length === 0) {
        this.locationName = '未知位置';
      }
    } finally {
      if (requestId === this.resolveRequestId) {
        this.isLoading = false;
      }
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
        this.searchResults = results;
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

  private openSearchPanel(): void {
    this.showFilterPanel = false;
    this.overlappingPanelItems = [];
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
    this.overlappingPanelItems = [];
    this.skipNextResolve = true;
    const gcj02 = CoordinateConverter.convertWgs84ToGcj02(
      result.latitude, result.longitude
    );
    this.setCenter(gcj02.latitude, gcj02.longitude);
    this.locationName = result.locationName || '';
    this.moveCameraSmoothly(gcj02.latitude, gcj02.longitude);
    this.closeSearchPanel();
  }

  private selectHistoryItem(item: AnchorHistoryItem): void {
    this.overlappingPanelItems = [];
    this.skipNextResolve = true;
    const gcj02 = CoordinateConverter.convertWgs84ToGcj02(
      item.latitude, item.longitude
    );
    this.setCenter(gcj02.latitude, gcj02.longitude);
    this.updateMarkers();
    this.moveCameraSmoothly(gcj02.latitude, gcj02.longitude);
    // 用历史项已有的名称，避免 GCJ02→WGS84→reverseGeocode 的冗余转换
    this.locationName = item.locationName || '历史位置';
  }

  private confirm(): void {
    if (this.locationName.length === 0) {
      SystemToast.show('请等待位置识别完成', this.getUIContext());
      return;
    }
    const wgs84 = CoordinateConverter.convertGcj02ToWgs84(this.centerLat, this.centerLng);
    const location: LocationInfo = {
      latitude: wgs84.latitude,
      longitude: wgs84.longitude,
      locationName: this.locationName
    };
    AnchorHistoryUtil.addToHistory(wgs84.latitude, wgs84.longitude, this.locationName);

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

	private async fetchMapRenderData(trigger: string = 'unknown'): Promise<void> {
		if (!this.initialised || !this.mapReady) {
			return;
		}
		const viewport = this.buildViewport();
		if (!viewport) {
			return;
		}
		const filter = this.resolveMapFilter();
		const filterSignature = filter?.data_type ?? 'all';
		const boundsForSpan = new CoordinateBounds(
			viewport.bounds.north,
			viewport.bounds.south,
			viewport.bounds.east,
			viewport.bounds.west
		);
		const zoom = Math.max(1, Math.min(20, this.mapZoom));
		const spanM: number = CoordinateConverter.calculateViewportSpanMeters(boundsForSpan);
		const shouldUseHotspots: boolean = spanM <= 3000 && zoom >= 13;
		const mode: string = shouldUseHotspots ? 'hotspots' : 'heatmap';
		const signature = `${viewport.bounds.north.toFixed(6)}|${viewport.bounds.south.toFixed(6)}|${viewport.bounds.east.toFixed(6)}|${viewport.bounds.west.toFixed(6)}|${zoom.toFixed(2)}|${filterSignature}|${mode}`;
		if (signature === this.lastViewportSignature) {
			hilog.info(DOMAIN_NUMBER, TAG, `skip map/${mode} duplicate trigger=${trigger} signature=${signature}`);
			return;
		}
		const requestId = ++this.renderRequestId;
		try {
			if (shouldUseHotspots) {
					hilog.info(DOMAIN_NUMBER, TAG, `start map/hotspots trigger=${trigger} requestId=${requestId} signature=${signature}`);
					const pagination: PaginationParams = { limit: 100 };
					const requestFilter: MapCanvasFilter | undefined = filter ? {
						data_type: filter.data_type,
						time_range: filter.time_range
					} : undefined;
					const request: MapCanvasRequest = {
						viewport: viewport,
						filter: requestFilter,
						pagination: pagination
					};
					const response = await ApiClient.getMapCanvas(request);
				if (requestId !== this.renderRequestId) {
					hilog.info(DOMAIN_NUMBER, TAG, `ignore stale map/hotspots response trigger=${trigger} requestId=${requestId}`);
					return;
				}
					if (response.code === 0) {
						this.lastViewportSignature = signature;
						const canvas: MapCanvasResponse | undefined = response.data as MapCanvasResponse | undefined;
						const allItems: MapHotspotItem[] = canvas?.hotspots?.items ?? [];
					const nextItems: MapRenderItem[] = [];
					for (let index = 0; index < allItems.length; index++) {
						const item = allItems[index];
						if (item.object_type === 'user' && !this.showUsers) {
							continue;
						}
						if (item.object_type === 'group' && !this.showGroups) {
							continue;
						}
						if (item.object_type === 'moment' && !this.showMoments) {
							continue;
						}
						const renderId = `${index}_${item.object_type}_${item.object_id}`;
						const renderItem: MapRenderItem = {
							render_id: renderId,
							location: {
								latitude: item.location.latitude,
								longitude: item.location.longitude
							},
							type: item.object_type,
							total_count: 1,
              entity: {
                payload_id: item.object_id,
                title: item.title,
                subtitle: item.subtitle,
                type: item.object_type,
                distance_meters: item.distance_meters
              }
						};
						nextItems.push(renderItem);
					}
					this.renderItems = nextItems;
					hilog.info(DOMAIN_NUMBER, TAG, `finish map/hotspots trigger=${trigger} requestId=${requestId} items=${this.renderItems.length}`);
					this.mapToastMessage = '';
					this.updateMarkers();
				} else {
					hilog.warn(DOMAIN_NUMBER, TAG, `map/hotspots business error trigger=${trigger} requestId=${requestId} code=${response.code} message=${response.message ?? ''}`);
				}
			} else {
					hilog.info(DOMAIN_NUMBER, TAG, `start map/heatmap trigger=${trigger} requestId=${requestId} signature=${signature}`);
					let request: MapCanvasRequest;
					if (filter) {
						request = { viewport: viewport, filter: filter };
					} else {
						request = { viewport: viewport };
					}
					const response = await ApiClient.getMapCanvas(request);
				if (requestId !== this.renderRequestId) {
					hilog.info(DOMAIN_NUMBER, TAG, `ignore stale map/heatmap response trigger=${trigger} requestId=${requestId}`);
					return;
				}
					if (response.code === 0) {
						this.lastViewportSignature = signature;
						const canvas: MapCanvasResponse | undefined = response.data as MapCanvasResponse | undefined;
						const buckets: MapHeatBucket[] = canvas?.heatmap?.items ?? [];
					const nextItems: MapRenderItem[] = [];
					for (let index = 0; index < buckets.length; index++) {
						const bucket = buckets[index];
						const renderId = `${index}_${bucket.location.latitude.toFixed(6)}_${bucket.location.longitude.toFixed(6)}`;
						const renderItem: MapRenderItem = {
							render_id: renderId,
							location: {
								latitude: bucket.location.latitude,
								longitude: bucket.location.longitude
							},
							type: 'heat_bucket',
							total_count: bucket.total_count,
              heatBucket: {
                count: bucket.total_count,
                breakdown: this.formatHeatBreakdown(bucket)
              }
						};
						nextItems.push(renderItem);
					}
						this.renderItems = nextItems;
						hilog.info(DOMAIN_NUMBER, TAG, `finish map/heatmap trigger=${trigger} requestId=${requestId} items=${this.renderItems.length}`);
						this.mapToastMessage = '';
						this.updateMarkers();
				} else {
					hilog.warn(DOMAIN_NUMBER, TAG, `map/heatmap business error trigger=${trigger} requestId=${requestId} code=${response.code} message=${response.message ?? ''}`);
				}
			}
		} catch (error) {
			hilog.warn(DOMAIN_NUMBER, TAG, `render map data failed trigger=${trigger} requestId=${requestId}: ${JSON.stringify(error)}`);
		}
	}

  private formatHeatBreakdown(bucket: MapHeatBucket): string | undefined {
    const parts: string[] = [];
    if (bucket.breakdown?.user) {
      parts.push(`${bucket.breakdown.user} 个用户`);
    }
    if (bucket.breakdown?.group) {
      parts.push(`${bucket.breakdown.group} 个群组`);
    }
    if (bucket.breakdown?.moment) {
      parts.push(`${bucket.breakdown.moment} 条动态`);
    }
    return parts.length > 0 ? parts.join(' · ') : undefined;
  }

  private showMapToast(message: string, duration: number = 2000): void {
    if (message.length === 0) {
      this.mapToastMessage = '';
      return;
    }
    this.mapToastMessage = message;
    if (this.mapToastTimerId > 0) {
      clearTimeout(this.mapToastTimerId);
    }
    const timerId: number = setTimeout(() => {
      this.mapToastMessage = '';
      this.mapToastTimerId = 0;
    }, duration) as number;
    this.mapToastTimerId = timerId;
  }

  private buildViewport(): MapViewport {
    const zoom = Math.max(1, Math.min(20, this.mapZoom));

    // 优先使用真实可视区域（如果 MapKit 支持 getVisibleRegion）
    if (this.visibleRegion) {
      const corners: MapCoordinate[] = [
        this.visibleRegion.northeast,
        this.visibleRegion.northwest,
        this.visibleRegion.southeast,
        this.visibleRegion.southwest
      ];
      const convertedCorners = corners.map((corner: MapCoordinate) =>
        CoordinateConverter.convertGcj02ToWgs84(corner.latitude, corner.longitude)
      );
      const latitudes = convertedCorners.map((point) => point.latitude);
      const longitudes = convertedCorners.map((point) => point.longitude);
      const north = Math.min(90, Math.max(...latitudes));
      const south = Math.max(-90, Math.min(...latitudes));
      const east = Math.min(180, Math.max(...longitudes));
      const west = Math.max(-180, Math.min(...longitudes));

      // 视口中心由后端根据 bounds 自行计算，这里不下发额外字段
      const bounds: Bounds = {
        north: north,
        south: south,
        east: east,
        west: west
      };
      const viewport: MapViewport = {
        bounds: bounds,
      };
      return viewport;
    }

    // 兜底：部分设备/版本不支持 getVisibleRegion 时，退回到“中心点 + zoom 推算”的近似方案
    const gcj02Lat = this.centerLat || this.mapRegion.latitude;
    const gcj02Lng = this.centerLng || this.mapRegion.longitude;
    const wgs84 = CoordinateConverter.convertGcj02ToWgs84(gcj02Lat, gcj02Lng);
    const delta: BoundsDelta = this.computeBoundsDelta(zoom);
    const bounds: Bounds = {
      north: Math.min(90, wgs84.latitude + delta.latDelta / 2),
      south: Math.max(-90, wgs84.latitude - delta.latDelta / 2),
      east: Math.min(180, wgs84.longitude + delta.lonDelta / 2),
      west: Math.max(-180, wgs84.longitude - delta.lonDelta / 2)
    };
    const viewport: MapViewport = {
      bounds: bounds,
    };
    return viewport;
  }

  private computeBoundsDelta(zoom: number): BoundsDelta {
    const latDelta = 180 / Math.pow(2, zoom);
    const lonDelta = 360 / Math.pow(2, zoom);
    const delta: BoundsDelta = { latDelta: latDelta, lonDelta: lonDelta };
    return delta;
  }

  private updateMarkers(): void {
    this.renderItemsMap.clear();

    const nextMarkers: MapMarker[] = [];
    for (const item of this.renderItems) {
      this.renderItemsMap.set(item.render_id, item);
      if (!this.shouldDisplayItem(item)) {
        continue;
      }
      const gcj02 = CoordinateConverter.convertWgs84ToGcj02(
        item.location.latitude,
        item.location.longitude
      );
      nextMarkers.push(this.buildSingleMarker({
        item: item,
        gcj02Lat: gcj02.latitude,
        gcj02Lng: gcj02.longitude
      }));
    }

    if (this.centerLat !== 0 || this.centerLng !== 0) {
      nextMarkers.push({
        id: 'center_pin_selection',
        latitude: this.centerLat,
        longitude: this.centerLng,
        title: '',
        snippet: '',
        type: 'user',
        iconKey: 'center_pin',
        clickable: false
      });
    }
    this.mapMarkers = nextMarkers;
  }

  private buildSingleMarker(c: ConvertedRenderItem): MapMarker {
    return {
      id: c.item.render_id,
      latitude: c.gcj02Lat,
      longitude: c.gcj02Lng,
      title: this.getMarkerTitle(c.item),
      snippet: this.getMarkerSnippet(c.item),
      type: c.item.type,
      iconKey: this.getMarkerIconKey(c.item),
      clusterable: c.item.type === 'user' || c.item.type === 'group' || c.item.type === 'moment'
    };
  }

  private shouldDisplayItem(item: MapRenderItem): boolean {
    switch (item.type) {
      case 'user':
        return this.showUsers;
      case 'group':
        return this.showGroups;
      case 'moment':
        return this.showMoments;
      case 'heat_bucket':
        return this.showUsers || this.showGroups || this.showMoments;
      default:
        return true;
    }
  }

  private getMarkerIconKey(item: MapRenderItem): 'user' | 'group' | 'moment' | 'heat_bucket' {
		if (item.type === 'user' || item.type === 'group' || item.type === 'moment') {
			return item.type;
		}
		return 'heat_bucket';
  }

  private getMarkerTitle(item: MapRenderItem): string {
    if (item.entity) {
      return item.entity.title;
    }
    if (item.type === 'heat_bucket') {
      return `${item.total_count} 个热点`;
    }
    return '';
  }

  private getMarkerSnippet(item: MapRenderItem): string {
    if (item.entity?.subtitle && item.entity.subtitle.length > 0) {
      return item.entity.subtitle;
    }
    if (item.type === 'heat_bucket') {
      return item.heatBucket?.breakdown ?? `共 ${item.total_count} 项`;
    }
    if (item.entity?.distance_meters !== undefined) {
      return `距你 ${LocationUtil.formatDistance(item.entity.distance_meters)}`;
    }
    return '';
  }

  /**
   * 根据当前勾选的类型决定向后端传递的 data_type 过滤条件
   * - 全选或全不选：不下发 filter，让服务端返回所有类型
   * - 仅单选某一类：下发对应 data_type，减少无关数据
   */
  private resolveMapFilter(): MapCanvasFilter | undefined {
    const showUsers = this.showUsers;
    const showGroups = this.showGroups;
    const showMoments = this.showMoments;

    const noneSelected = !showUsers && !showGroups && !showMoments;
    const allSelected = showUsers && showGroups && showMoments;
    if (noneSelected || allSelected) {
      return undefined;
    }

    let dataType: 'user' | 'group' | 'moment' | 'all' = 'all';
    if (showUsers && !showGroups && !showMoments) {
      dataType = 'user';
    } else if (!showUsers && showGroups && !showMoments) {
      dataType = 'group';
    } else if (!showUsers && !showGroups && showMoments) {
      dataType = 'moment';
    } else {
      dataType = 'all';
    }

    const filter: MapCanvasFilter = { data_type: dataType };
    return filter;
  }

  private applyFilterChange(toggle: () => void): void {
    toggle();
    this.lastViewportSignature = '';
    if (this.renderTimerId > 0) {
      clearTimeout(this.renderTimerId);
    }
    if (this.initialised && this.mapReady) {
      this.fetchMapRenderData('filter_change');
    } else {
      this.updateMarkers();
    }
  }

  private onMarkerItemClick(marker: MapMarker): void {
    this.showFilterPanel = false;
    if (this.showSearchBar) {
      this.closeSearchPanel();
    }

    const item = this.renderItemsMap.get(marker.id);
    if (!item || !item.entity) {
      return;
    }
    this.showOverlappingPanel([item]);
  }

  private onClusterClick(markers: MapMarker[]): void {
    this.showFilterPanel = false;
    if (this.showSearchBar) {
      this.closeSearchPanel();
    }
    const items: MapRenderItem[] = [];
    for (const marker of markers) {
      const item = this.renderItemsMap.get(marker.id);
      if (item && item.entity) {
        items.push(item);
      }
    }
    this.showOverlappingPanel(items);
  }

  private showOverlappingPanel(items: MapRenderItem[]): void {
    const entityItems = items.filter((item: MapRenderItem) => item.entity !== undefined);
    if (entityItems.length === 0) return;
    this.showFilterPanel = false;
    this.overlappingPanelItems = entityItems;
  }

  private navigateToItem(item: MapRenderItem): void {
    if (!item.entity) return;
    this.overlappingPanelItems = [];
    this.showFilterPanel = false;
    if (item.type === 'group') {
      const param = new GroupDetailRouteParam();
      param.groupId = item.entity.payload_id;
      AppRouter.push(PageNames.GROUP_DETAIL, param as Object);
    } else if (item.type === 'user') {
      const param = new ProfileRouteParam();
      param.userId = item.entity.payload_id;
      AppRouter.push(PageNames.PROFILE, param as Object);
    } else if (item.type === 'moment') {
      const param = new MomentDetailRouteParam();
      param.momentId = item.entity.payload_id;
      AppRouter.push(PageNames.MOMENT_DETAIL, param as Object);
    }
  }

  private getItemTypeLabel(type: string): string {
    switch (type) {
      case 'user': return '用户';
      case 'group': return '群组';
      case 'moment': return '动态';
      default: return '内容';
    }
  }

  private getTypeColor(type: string): string {
    switch (type) {
      case 'user': return MARKER_COLOR_USER;
      case 'group': return MARKER_COLOR_GROUP;
      case 'moment': return MARKER_COLOR_MOMENT;
      default: return DesignConstants.MAP_MARKER.DEFAULT;
    }
  }

  private getTypeMarkerSvg(type: string): string {
    switch (type) {
      case 'user': return 'marker_user.svg';
      case 'group': return 'marker_group.svg';
      case 'moment': return 'marker_moment.svg';
      default: return 'marker_user.svg';
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
          showCenterPin: false,
          contentPaddingTop: SafeAreaUtils.resolveTopPadding(this.safeAreaTop, 110, 20),
          contentPaddingBottom: this.getMapPaddingBottom(),
          logoPaddingBottom: this.getLogoPaddingBottom(),
          onCameraIdle: (camera: MapCameraPosition) => { this.onCameraIdle(camera); },
          onMapClick: (_coord: MapCoordinate) => { this.collapsePanel(); },
          onMapDoubleClick: (coord: MapCoordinate) => { this.onMapDoubleClick(coord); },
          onMapLongPress: (coord: MapCoordinate) => { this.onMapLongPress(coord); },
          onMarkerClick: (marker: MapMarker) => { this.onMarkerItemClick(marker); },
          onClusterClick: (markers: MapMarker[]) => { this.onClusterClick(markers); },
          onActionsReady: (actions: NativeMapActions) => { this.mapActions = actions; },
          onMapReady: () => { this.handleMapReady(); },
          onVisibleRegionChange: (region: MapVisibleRegion) => { this.handleVisibleRegionChange(region); }
        })
          .width('100%')
          .height('100%')

        // 地图 Toast 提示
        MapToast({ message: this.mapToastMessage, safeAreaTop: this.safeAreaTop, resolvedThemeId: this.resolvedThemeId })

        // TopBar + 筛选下拉面板
        Column() {
          TopBar({
            filterOpen: this.showFilterPanel,
            safeAreaTop: this.safeAreaTop,
            resolvedThemeId: this.resolvedThemeId,
            onBack: () => { this.handleBackNavigation(); },
            onToggleFilter: () => {
              this.overlappingPanelItems = [];
              if (!this.showFilterPanel) {
                this.closeSearchPanel();
              }
              this.showFilterPanel = !this.showFilterPanel;
            },
          })
          if (this.showFilterPanel) {
            Row() {
              FilterDropdownPanel({
                showUsers: this.showUsers,
                showGroups: this.showGroups,
                showMoments: this.showMoments,
                resolvedThemeId: this.resolvedThemeId,
                onToggleUsers: () => {
                  this.applyFilterChange(() => { this.showUsers = !this.showUsers; });
                },
                onToggleGroups: () => {
                  this.applyFilterChange(() => { this.showGroups = !this.showGroups; });
                },
                onToggleMoments: () => {
                  this.applyFilterChange(() => { this.showMoments = !this.showMoments; });
                }
              })
            }
            .width('100%')
            .justifyContent(FlexAlign.End)
            .padding({ right: DesignConstants.SPACING.MD, top: DesignConstants.SPACING.XS })
          }
        }
        .width('100%')
        .position({ top: 0, left: 0 })
        .zIndex(30)

        // 底部面板
        if (this.showSearchBar) {
          SearchPanel({
            searchText: this.searchText,
            isSearching: this.isSearching,
            searchResults: this.searchResults,
            anchorHistory: this.anchorHistory,
            controller: this.searchController,
            onSearch: (): void => { this.doSearch(); },
            onClose: (): void => { this.closeSearchPanel(); },
            onClearHistory: (): void => { this.clearAnchorHistory(); },
            onSelectResult: (result: LocationInfo): void => { this.selectSearchResult(result); },
            onSelectHistory: (item: AnchorHistoryItem): void => { this.selectHistoryItem(item); },
            onTextChange: (value: string): void => { this.searchText = value; }
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
            onSearchClick: (): void => { this.openSearchPanel(); },
            onClearHistory: (): void => { this.clearAnchorHistory(); },
            onSelectHistory: (item: AnchorHistoryItem): void => {
              this.selectHistoryItem(item);
            },
            onConfirm: (): void => { this.confirm(); }
          })
            .transition(TransitionEffect.asymmetric(
              TransitionEffect.OPACITY
                .combine(TransitionEffect.translate({ y: 30 }))
                .animation({ duration: 300, delay: 150, curve: Curve.EaseOut }),
              TransitionEffect.OPACITY
                .animation({ duration: 150 })
            ))
        }

        // 重叠 marker 选择面板
        if (this.overlappingPanelItems.length > 0) {
          Column() {
            // 遮罩层：点击关闭
            Column()
              .width('100%')
              .layoutWeight(1)
              .onClick(() => { this.overlappingPanelItems = []; })

            Column() {
              Row() {
                Text(`该位置有 ${this.overlappingPanelItems.length} 个内容`)
                  .fontSize(DesignConstants.FONT.SIZE.LG)
                  .fontWeight(FontWeight.Medium)
                  .fontColor($r('sys.color.ohos_id_color_text_primary'))
                  .layoutWeight(1)
                Text('关闭')
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                  .onClick(() => { this.overlappingPanelItems = []; })
              }
              .width('100%')
              .padding({
                left: DesignConstants.SPACING.LG,
                right: DesignConstants.SPACING.LG,
                top: DesignConstants.SPACING.LG,
                bottom: DesignConstants.SPACING.SM
              })

              List() {
                ForEach(this.overlappingPanelItems, (item: MapRenderItem) => {
                  ListItem() {
                    Row({ space: DesignConstants.SPACING.MD }) {
                      Image($rawfile(this.getTypeMarkerSvg(item.type)))
                        .width(28)
                        .height(36)
                        .objectFit(ImageFit.Contain)
                      Column({ space: 2 }) {
                        Text(item.entity ? item.entity.title : '')
                          .fontSize(DesignConstants.FONT.SIZE.MD)
                          .fontWeight(FontWeight.Medium)
                          .fontColor($r('sys.color.ohos_id_color_text_primary'))
                          .maxLines(1)
                          .textOverflow({ overflow: TextOverflow.Ellipsis })
                        if (item.entity && item.entity.subtitle) {
                          Text(item.entity.subtitle)
                            .fontSize(DesignConstants.FONT.SIZE.SM)
                            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                            .maxLines(1)
                            .textOverflow({ overflow: TextOverflow.Ellipsis })
                        }
                      }
                      .layoutWeight(1)
                      .alignItems(HorizontalAlign.Start)

                      Text(this.getItemTypeLabel(item.type))
                        .fontSize(DesignConstants.FONT.SIZE.XS)
                        .fontColor(this.getTypeColor(item.type))
                        .padding({ left: 8, right: 8, top: 4, bottom: 4 })
                        .borderRadius(DesignConstants.RADIUS.SM)
                        .backgroundColor(this.getCurrentThemePalette().surfaceSubtle)
                    }
                    .width('100%')
                    .padding({
                      left: DesignConstants.SPACING.LG,
                      right: DesignConstants.SPACING.LG,
                      top: DesignConstants.SPACING.SM,
                      bottom: DesignConstants.SPACING.SM
                    })
                    .onClick(() => {
                      this.overlappingPanelItems = [];
                      this.navigateToItem(item);
                    })
                  }
                }, (item: MapRenderItem) => item.render_id)
              }
              .width('100%')
              .divider({ strokeWidth: 0.5, color: $r('app.color.divider'), startMargin: 60 })
              .constraintSize({ maxHeight: 320 })
              .padding({ bottom: this.safeAreaBottom + DesignConstants.SPACING.MD })
            }
            .width('100%')
            .backgroundColor(this.getCurrentThemePalette().surfaceBackground)
            .borderRadius({ topLeft: 20, topRight: 20 })
            .shadow({
              radius: DesignConstants.SHADOW.LARGE.radius,
              color: DesignConstants.SHADOW.LARGE.color,
              offsetX: DesignConstants.SHADOW.LARGE.offsetX,
              offsetY: DesignConstants.SHADOW.LARGE.offsetY
            })
          }
          .width('100%')
          .height('100%')
          .zIndex(50)
          .transition(
            TransitionEffect.OPACITY.animation({ duration: 200, curve: Curve.FastOutSlowIn })
              .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 250, curve: Curve.FastOutSlowIn }))
          )
        }

        // 自定义地图控件：用 position(Edges) 定位到右下角，仅占按钮实际面积，
        // 既不阻塞地图拖拽，又能独占按钮区域的点击事件
        Column({ space: DesignConstants.SPACING.MD }) {
          Column() {
            Text() {
              SymbolSpan($r(AppIcon.LOCATION))
                .fontSize(DesignConstants.FONT.SIZE.XL)
                .fontColor([MARKER_COLOR_USER])
            }
          }
          .width(48)
          .height(48)
          .justifyContent(FlexAlign.Center)
          .backgroundColor(this.getCurrentThemePalette().surfaceBackground)
          .borderRadius(DesignConstants.RADIUS.XXL)
          .shadow({
            radius: DesignConstants.SHADOW.MEDIUM.radius,
            color: DesignConstants.SHADOW.MEDIUM.color,
            offsetX: DesignConstants.SHADOW.MEDIUM.offsetX,
            offsetY: DesignConstants.SHADOW.MEDIUM.offsetY
          })
          .border({ width: 1, color: this.getCurrentThemePalette().borderColor })
          .onClick(() => { this.locateUser(); })

          Column() {
            Image($rawfile('marker_center_pin.svg'))
              .width(20)
              .height(28)
              .objectFit(ImageFit.Contain)
          }
          .width(48)
          .height(48)
          .justifyContent(FlexAlign.Center)
          .backgroundColor(this.getCurrentThemePalette().surfaceBackground)
          .borderRadius(DesignConstants.RADIUS.XXL)
          .shadow({
            radius: DesignConstants.SHADOW.MEDIUM.radius,
            color: DesignConstants.SHADOW.MEDIUM.color,
            offsetX: DesignConstants.SHADOW.MEDIUM.offsetX,
            offsetY: DesignConstants.SHADOW.MEDIUM.offsetY
          })
          .border({ width: 1, color: this.getCurrentThemePalette().borderColor })
          .onClick(() => { this.locateBrowseAnchor(); })

          Column() {
            Text() {
              SymbolSpan($r(AppIcon.ADD))
                .fontSize(DesignConstants.FONT.SIZE.XL)
                .fontColor([$r('sys.color.ohos_id_color_text_primary')])
            }
          }
          .width(48)
          .height(48)
          .justifyContent(FlexAlign.Center)
          .backgroundColor(this.getCurrentThemePalette().surfaceBackground)
          .borderRadius(DesignConstants.RADIUS.XXL)
          .shadow({
            radius: DesignConstants.SHADOW.MEDIUM.radius,
            color: DesignConstants.SHADOW.MEDIUM.color,
            offsetX: DesignConstants.SHADOW.MEDIUM.offsetX,
            offsetY: DesignConstants.SHADOW.MEDIUM.offsetY
          })
          .border({ width: 1, color: this.getCurrentThemePalette().borderColor })
          .onClick(() => { if (this.mapActions) this.mapActions.zoomIn(); })

          Column() {
            Text() {
              SymbolSpan($r(AppIcon.REMOVE))
                .fontSize(DesignConstants.FONT.SIZE.XL)
                .fontColor([$r('sys.color.ohos_id_color_text_primary')])
            }
          }
          .width(48)
          .height(48)
          .justifyContent(FlexAlign.Center)
           .backgroundColor(this.getCurrentThemePalette().surfaceBackground)
          .borderRadius(DesignConstants.RADIUS.XXL)
          .shadow({
            radius: DesignConstants.SHADOW.MEDIUM.radius,
            color: DesignConstants.SHADOW.MEDIUM.color,
            offsetX: DesignConstants.SHADOW.MEDIUM.offsetX,
            offsetY: DesignConstants.SHADOW.MEDIUM.offsetY
          })
           .border({ width: 1, color: this.getCurrentThemePalette().borderColor })
          .onClick(() => { if (this.mapActions) this.mapActions.zoomOut(); })
        }
        .position({ right: DesignConstants.SPACING.MD, bottom: this.getControlsBottomOffset() })
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
    .hideTitleBar(true)
    .onBackPressed((): boolean => this.handleBackNavigation())
    .onShown(() => { if (this.mapActions) { this.mapActions.show(); } })
    .onHidden(() => { if (this.mapActions) { this.mapActions.hide(); } })
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
