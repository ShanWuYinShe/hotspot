import { KEY_BROWSE_ANCHOR, KEY_CURRENT_LOCATION, KEY_LOCATION_PICK_RESULT } from '../../../core/state/GlobalStateManager';
import { LocationInfo, LocationUtil } from '../../../core/location/LocationUtil';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { AnchorHistoryItem, AnchorHistoryUtil } from '../../../core/utils/AnchorHistoryUtil';
import { NativeMapComponent, NativeMapActions, MapRegion, MapCoordinate, MapMarker, MapCameraPosition, MapVisibleRegion } from '../../../components/NativeMapComponent';
import { SystemToast } from '../../../components/SystemToast';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { EmptyState } from '../../../components/EmptyState';
import { LocationResultRow } from '../components/LocationResultRow';
import { Pressable } from '../../../components/Pressable';
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
import { LucideIcon } from '../../../components/LucideIcon';
export const DOMAIN_NUMBER: number = 0xFF13;
export const TAG: string = 'LocationPickerPage';
export const SEARCH_PANEL_HEIGHT: number = 440;
export const MAX_HISTORY_ITEMS: number = 5;
// 底部面板三段式吸附位置（内容高度，不含 safeAreaBottom）
export const PANEL_SNAP_MIN: number = 120;       // 收起：手柄 + 搜索栏 + 确认按钮
export const PANEL_SNAP_HALF: number = 224;      // 半展开：显示部分历史记录
export const PANEL_SNAP_MAX: number = 392;       // 全展开：完整内容
export const DEFAULT_GESTURE_HINT: string = '长按地图即可快速选点';
export const DOUBLE_CLICK_PLACEMENT_HINT: string = '已为你定位到该位置';
export const LONG_PRESS_PLACEMENT_HINT: string = '已为你定位到该位置';
export const ANCHOR_PLACEMENT_HINT: string = '已回到浏览锚点位置';
export const GENERIC_PLACEMENT_HINT: string = '已定位到该位置';

type MapRenderItemType = 'heat_bucket' | 'user' | 'group' | 'moment';

export interface MapRenderItemLocation {
	latitude: number;
	longitude: number;
}

export interface MapRenderEntity {
	payload_id: string;
	title: string;
	subtitle?: string;
	type: 'user' | 'group' | 'moment';
	distance_meters?: number;
}

export interface MapRenderHeatBucket {
  count: number;
  breakdown?: string;
}

export interface MapRenderItem {
	render_id: string;
	location: MapRenderItemLocation;
	type: MapRenderItemType;
	total_count: number;
	entity?: MapRenderEntity;
  heatBucket?: MapRenderHeatBucket;
}

export interface ConvertedRenderItem {
	item: MapRenderItem;
	gcj02Lat: number;
	gcj02Lng: number;
}

export interface BoundsDelta {
  latDelta: number;
  lonDelta: number;
}

export type LocationPickerMode = 'anchor' | 'location_pick';

export interface LocationPickerParamLike {
  mode?: string;
}

export interface TopBarProps {
  filterOpen: boolean;
  safeAreaTop: number;
  resolvedThemeId?: string;
  onBack: () => void;
  onToggleFilter: () => void;
}

export function resolveLocationPickerPalette(themeId?: string) {
  return AppThemeManager.getPaletteById(themeId ?? 'vibrant_light');
}

@Builder
export function TopBar(props: TopBarProps) {
  Flex({
    direction: FlexDirection.Row,
    justifyContent: FlexAlign.SpaceBetween,
    alignItems: ItemAlign.Center
  }) {
    Row() {
      LucideIcon({
          icon: AppIcon.BACK,
          iconSize: DesignConstants.FONT.SIZE.XL,
          color: resolveLocationPickerPalette(props.resolvedThemeId).textPrimary
        })
    }
    .width(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL_PLUS)
    .height(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL_PLUS)
    .justifyContent(FlexAlign.Center)
    .borderRadius(DesignConstants.RADIUS.FULL)
    .backgroundColor(resolveLocationPickerPalette(props.resolvedThemeId).surfaceSubtle)
    // 按压缩放（顶层 @Builder 无组件状态，用 scale 类工具不可行；最少补 opacity 按压反馈）
    .onClick(props.onBack)

    Row({ space: DesignConstants.SPACING.SM2 }) {
      LucideIcon({
          icon: AppIcon.FILTER,
          iconSize: DesignConstants.FONT.SIZE.LG,
          color: resolveLocationPickerPalette(props.resolvedThemeId).textPrimary
        })
      Text('筛选')
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
        .fontColor(resolveLocationPickerPalette(props.resolvedThemeId).textPrimary)
    }
    .height(DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT_MD)
    .padding({
      left: DesignConstants.SPACING.MD,
      right: DesignConstants.SPACING.MD
    })
    .backgroundColor(props.filterOpen
      ? resolveLocationPickerPalette(props.resolvedThemeId).surfaceSubtle
      : resolveLocationPickerPalette(props.resolvedThemeId).surfaceBackground)
    .borderRadius(DesignConstants.RADIUS.FULL)
    .shadow(DesignConstants.SHADOW.SMALL)
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: resolveLocationPickerPalette(props.resolvedThemeId).borderColor })
    .onClick(props.onToggleFilter)
  }
  .width('100%')
  .padding({
    left: DesignConstants.SPACING.MD,
    right: DesignConstants.SPACING.MD,
    top: SafeAreaUtils.resolveTopPadding(props.safeAreaTop, DesignConstants.SPACING.XS)
  })
  .hitTestBehavior(HitTestMode.Default)
}

export interface FilterMenuItemProps {
  markerSvg: string;
  label: string;
  accentColor: string;
  active: boolean;
  resolvedThemeId?: string;
  onToggle: () => void;
}

@Builder
export function FilterMenuItem(props: FilterMenuItemProps) {
  // 整行按压反馈收敛到 Pressable（顶层 @Builder 无组件状态可用）；
  // 按压差异：选中行按压不缩放（选中底色已表达状态），未选中按压缩放（「将选中」预览）
  Pressable({
    onPress: () => props.onToggle(),
    pressScale: props.active ? 1.0 : DesignConstants.INTERACTION.SCALE_PRESSED_INPUT_ROW
  }) {
    Row({ space: DesignConstants.SPACING.MD }) {
      Image($rawfile(props.markerSvg))
        .width(DesignConstants.COMPONENT.ICON_SIZE_MD)
        .height(DesignConstants.COMPONENT.AVATAR_SIZE.XS)
        .objectFit(ImageFit.Contain)
      Text(props.label)
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .fontWeight(props.active ? DesignConstants.FONT.WEIGHT.MEDIUM : DesignConstants.FONT.WEIGHT.REGULAR)
        .fontColor(props.active
          ? resolveLocationPickerPalette(props.resolvedThemeId).textPrimary
          : resolveLocationPickerPalette(props.resolvedThemeId).textSecondary)
        .layoutWeight(1)
      Row()
        .width(DesignConstants.SPACING.MD3)
        .height(DesignConstants.SPACING.MD3)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .backgroundColor(props.active ? props.accentColor : DesignConstants.COLOR.TRANSPARENT)
        .border({ width: DesignConstants.BORDER_WIDTH.REGULAR, color: props.accentColor })
    }
    .width('100%')
    .height(DesignConstants.COMPONENT.SHEET_ACTION_HEIGHT)
    .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD })
    .justifyContent(FlexAlign.Center)
    .alignItems(VerticalAlign.Center)
    .backgroundColor(props.active
      ? resolveLocationPickerPalette(props.resolvedThemeId).surfaceSubtle
      : DesignConstants.COLOR.TRANSPARENT)
    .borderRadius(DesignConstants.RADIUS.LG)
    .opacity(props.active ? 1.0 : DesignConstants.OPACITY.SUBTLE)
  }
}

export interface FilterPanelProps {
  showUsers: boolean;
  showGroups: boolean;
  showMoments: boolean;
  resolvedThemeId?: string;
  onToggleUsers: () => void;
  onToggleGroups: () => void;
  onToggleMoments: () => void;
}

@Builder
export function FilterDropdownPanel(props: FilterPanelProps) {
  Column({ space: DesignConstants.SPACING.XS }) {
    Row() {
      Text('显示类型')
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
        .fontColor(resolveLocationPickerPalette(props.resolvedThemeId).textPrimary)
      Blank()
      Text(`${(props.showUsers ? 1 : 0) + (props.showGroups ? 1 : 0) + (props.showMoments ? 1 : 0)}/3`)
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .fontColor(resolveLocationPickerPalette(props.resolvedThemeId).textSecondary)
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM, top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })

    FilterMenuItem({
      markerSvg: 'marker_user.svg',
      label: '用户',
      accentColor: resolveLocationPickerPalette(props.resolvedThemeId).mapUser,
      active: props.showUsers,
      resolvedThemeId: props.resolvedThemeId,
      onToggle: props.onToggleUsers
    })
    FilterMenuItem({
      markerSvg: 'marker_group.svg',
      label: '群组',
      accentColor: resolveLocationPickerPalette(props.resolvedThemeId).mapGroup,
      active: props.showGroups,
      resolvedThemeId: props.resolvedThemeId,
      onToggle: props.onToggleGroups
    })
    FilterMenuItem({
      markerSvg: 'marker_moment.svg',
      label: '动态',
      accentColor: resolveLocationPickerPalette(props.resolvedThemeId).mapMoment,
      active: props.showMoments,
      resolvedThemeId: props.resolvedThemeId,
      onToggle: props.onToggleMoments
    })
  }
  .width(DesignConstants.LAYOUT.FILTER_PANEL_WIDTH)
  .padding(DesignConstants.SPACING.SM)
  .backgroundColor(resolveLocationPickerPalette(props.resolvedThemeId).surfaceBackground)
  .borderRadius(DesignConstants.SHAPE.CARD)
  .shadow(DesignConstants.SHADOW.MEDIUM)
  .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: resolveLocationPickerPalette(props.resolvedThemeId).borderColor })
  .hitTestBehavior(HitTestMode.Default)
}

export interface MapToastProps {
  message: string;
  safeAreaTop: number;
  resolvedThemeId?: string;
}

@Builder
export function MapToast(props: MapToastProps) {
  if (props.message.length === 0) {
    // 不渲染
  } else {
    Text(props.message)
      .fontSize(DesignConstants.FONT.SIZE.SM)
      .fontColor(resolveLocationPickerPalette(props.resolvedThemeId).textPrimary)
      .backgroundColor(resolveLocationPickerPalette(props.resolvedThemeId).surfaceElevated)
      .borderRadius(DesignConstants.RADIUS.XL)
      .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
      .shadow(DesignConstants.SHADOW.SMALL)
      .position({ top: SafeAreaUtils.resolveTopPadding(props.safeAreaTop, 76), left: 20, right: 20 })
      .hitTestBehavior(HitTestMode.Default)
      .transition(
        TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_280, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
          .combine(TransitionEffect.move(TransitionEdge.TOP).animation({ duration: DesignConstants.ANIMATION.DURATION_280, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
      )
  }
}

/**
 * 搜索面板（当 showSearchBar 为 true 时显示）
 * 固定高度，不可拖动
 */
@Component
export struct SearchPanel {
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
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @State cancelPressed: boolean = false;
  @State clearPressed: boolean = false;
  @State pressedRowKey: string = '';

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
          .height(DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT_MD)
          .backgroundColor(resolveLocationPickerPalette(this.resolvedThemeId).surfaceSubtle)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .shadow(DesignConstants.SHADOW.INSET)
          .searchIcon({
            size: 18,
            color: resolveLocationPickerPalette(this.resolvedThemeId).textPrimary
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
            .width(DesignConstants.COMPONENT.LOADING_ICON_SIZE)
            .height(DesignConstants.COMPONENT.LOADING_ICON_SIZE)
            .color(resolveLocationPickerPalette(this.resolvedThemeId).accentBackground)
        }

        // 取消按钮
        Column() {
          Text('取消')
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor(resolveLocationPickerPalette(this.resolvedThemeId).textPrimary)
        }
        .height(DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT_MD)
        .justifyContent(FlexAlign.Center)
        .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM })
        .scale(this.cancelPressed
          ? { x: DesignConstants.INTERACTION.SCALE_PRESSED_BUTTON, y: DesignConstants.INTERACTION.SCALE_PRESSED_BUTTON }
          : { x: 1, y: 1 })
        .animation({
          curve: curves.springMotion(
            DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
            DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
          )
        })
        .onTouch((event: TouchEvent) => {
          if (event.type === TouchType.Down) {
            this.cancelPressed = true;
          } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
            this.cancelPressed = false;
          }
        })
        .onClick(this.onClose)
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)
      .padding({ bottom: DesignConstants.SPACING.MD })

      // 内容区域 - 可滚动
      Scroll() {
        Column() {
          if (this.isSearching) {
            // 加载态统一 SystemLoadingView（与其余页面一致）
            SystemLoadingView({ message: '正在搜索...' })
          } else if (this.searchResults.length > 0) {
            // 搜索结果列表
            List({ space: DesignConstants.SPACING.SM }) {
              ForEach(this.searchResults, (item: LocationInfo, index: number) => {
                ListItem() {
                  LocationResultRow({
                    item: item,
                    resolvedThemeId: this.resolvedThemeId,
                    onSelect: (result) => { this.onSelectResult(result); }
                  })
                }
              }, (item: LocationInfo) => `search_${item.latitude}_${item.longitude}`)
            }
            .width('100%')
            .divider({ strokeWidth: 0, color: resolveLocationPickerPalette(this.resolvedThemeId).borderColor })
          } else if (this.searchText.trim().length > 0) {
            // 空态统一 EmptyState（bare 形态，适配面板内嵌场景）
            EmptyState({ title: '未找到该地点', message: '请换个关键词试试', bare: true })
          } else if (this.anchorHistory.length > 0) {
            // 历史锚点列表（搜索未输入时展示）
            Column({ space: DesignConstants.SPACING.SM }) {
              Row({ space: DesignConstants.SPACING.SM2 }) {
                Text('最近使用')
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                  .fontColor(resolveLocationPickerPalette(this.resolvedThemeId).textPrimary)
                Blank()
                Text('清空')
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor(resolveLocationPickerPalette(this.resolvedThemeId).textSecondary)
                  .scale(this.clearPressed
                    ? { x: DesignConstants.INTERACTION.SCALE_PRESSED_INPUT_ROW, y: DesignConstants.INTERACTION.SCALE_PRESSED_INPUT_ROW }
                    : { x: 1, y: 1 })
                  .onTouch((event: TouchEvent) => {
                    if (event.type === TouchType.Down) {
                      this.clearPressed = true;
                    } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
                      this.clearPressed = false;
                    }
                  })
                  .onClick(this.onClearHistory)
              }
              .width('100%')

              List({ space: DesignConstants.SPACING.SM }) {
                ForEach(this.anchorHistory.slice(0, MAX_HISTORY_ITEMS), (item: AnchorHistoryItem, index: number) => {
                  ListItem() {
                    Row({ space: DesignConstants.SPACING.MD }) {
                      Column() {
                        LucideIcon({
                            icon: AppIcon.TIME,
                            iconSize: DesignConstants.FONT.SIZE.LG_PLUS,
                            color: resolveLocationPickerPalette(this.resolvedThemeId).textSecondary
                          })
                      }
                      .width(DesignConstants.COMPONENT.AVATAR_SIZE.XS)
                      .height(DesignConstants.COMPONENT.AVATAR_SIZE.XS)
                      .justifyContent(FlexAlign.Center)
                      .backgroundColor(resolveLocationPickerPalette(this.resolvedThemeId).surfaceSubtle)
                      .borderRadius(DesignConstants.RADIUS.MD)

                      Text(item.locationName || '未命名位置')
                        .fontSize(DesignConstants.FONT.SIZE.MD)
                        .fontColor(resolveLocationPickerPalette(this.resolvedThemeId).textPrimary)
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
                    // 面板内行去三件套（surface 叠 surface + 边框 + 阴影），改 tint 行 + 按压反馈
                    .backgroundColor(resolveLocationPickerPalette(this.resolvedThemeId).surfaceSubtle)
                    .borderRadius(DesignConstants.RADIUS.LG)
                    .scale(this.pressedRowKey === `hist_s_${item.latitude}_${item.longitude}` ?
                      { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD } :
                      { x: 1, y: 1 })
                    .animation({ curve: curves.springMotion(
                      DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
                      DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
                    ) })
                    .onTouch((event: TouchEvent) => {
                      if (event.type === TouchType.Down) {
                        this.pressedRowKey = `hist_s_${item.latitude}_${item.longitude}`;
                      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
                        this.pressedRowKey = '';
                      }
                    })
                    .onClick(() => {
                      this.onSelectHistory(item);
                      this.onClose();
                    })
                  }
                }, (item: AnchorHistoryItem) => `hist_s_${item.latitude}_${item.longitude}`)
              }
              .width('100%')
              .divider({ strokeWidth: 0, color: resolveLocationPickerPalette(this.resolvedThemeId).borderColor })
            }
            .width('100%')
          } else {
            // 空状态统一 EmptyState
            EmptyState({ title: '暂无搜索历史', bare: true })
          }
        }
        .width('100%')
        .padding({ top: DesignConstants.SPACING.XS })
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
    .backgroundColor(resolveLocationPickerPalette(this.resolvedThemeId).surfaceBackground)
    .border({ width: { top: DesignConstants.BORDER_WIDTH.HAIRLINE }, color: resolveLocationPickerPalette(this.resolvedThemeId).borderColor })
    .borderRadius({ topLeft: DesignConstants.SHAPE.SHEET_TOP_RADIUS, topRight: DesignConstants.SHAPE.SHEET_TOP_RADIUS })
    .hitTestBehavior(HitTestMode.Default)
  }
}

/**
 * 主面板 —— 可拖拽的底部面板（Sheet 风格）
 * 三段式吸附：收起 / 半展开 / 全展开，参考 HarmonyOS Petal Maps 交互
 */
@Component
export struct MainPanel {
  @Link anchorHistory: AnchorHistoryItem[];
  @Link panelHeight: number;
  onSearchClick: () => void = () => {};
  onClearHistory: () => void = () => {};
  onSelectHistory: (item: AnchorHistoryItem) => void = (item) => {};
  onConfirm: () => void = () => {};
  @Prop confirmLabel: string = '设为浏览锚点';
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;

  /** 底部手势区兜底：safeAreaBottom 上报为 0 时按 XL(20vp) 处理 */
  private resolveBottomInset(): number {
    return Math.max(this.safeAreaBottom, DesignConstants.SPACING.XL);
  }
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @State clearPressed: boolean = false;
  @State searchEntryPressed: boolean = false;
  @State pressedRowKey: string = '';

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
      { duration: DesignConstants.ANIMATION.DURATION_250, curve: curves.springMotion(0.6, 0.9) },
      () => { this.panelHeight = target; }
    );
  }

  build() {
    Column() {
      // 拖拽手柄区域（36px 高的触控热区，内含视觉指示条）
      Column() {
        // 拖拽手柄视觉条 4vp：物理细件例外（对齐前端拖拽把手例外），不做间距档位
        Column()
          .width(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
          .height(4)
          .borderRadius(DesignConstants.RADIUS.SM)
          .backgroundColor(resolveLocationPickerPalette(this.resolvedThemeId).surfaceSubtle)
      }
      .width('100%')
      .height(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
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
        LucideIcon({
            icon: AppIcon.SEARCH,
            iconSize: DesignConstants.FONT.SIZE.LG,
            color: resolveLocationPickerPalette(this.resolvedThemeId).textTertiary
          })
        Text('搜索地点、地标或坐标')
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor(resolveLocationPickerPalette(this.resolvedThemeId).textTertiary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
      }
      .width('100%')
      .height(DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT_MD)
      .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD })
      .alignItems(VerticalAlign.Center)
      .backgroundColor(resolveLocationPickerPalette(this.resolvedThemeId).surfaceSubtle)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .shadow(DesignConstants.SHADOW.INSET)
      .margin({ bottom: DesignConstants.SPACING.MD })
      // 按压反馈（对齐 tap-97 语义）
      .scale(this.searchEntryPressed ?
        { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD } :
        { x: 1, y: 1 })
      .animation({ curve: curves.springMotion(
        DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
        DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
      ) })
      .onTouch((event: TouchEvent) => {
        if (event.type === TouchType.Down) {
          this.searchEntryPressed = true;
        } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
          this.searchEntryPressed = false;
        }
      })
      .onClick(this.onSearchClick)

      // 可滚动历史记录（面板最低档时隐藏，由 clip(true) 保证拖拽过程平滑裁剪）
      if (this.panelHeight > PANEL_SNAP_MIN) {
        Scroll() {
          Column() {
            // 最近使用标题
            Row() {
              Text('最近使用')
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                .fontColor(resolveLocationPickerPalette(this.resolvedThemeId).textSecondary)
              Blank()
              if (this.anchorHistory.length > 0) {
                Text('清空')
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor(resolveLocationPickerPalette(this.resolvedThemeId).textTertiary)
                  .padding({ left: DesignConstants.SPACING.SM, top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
                  .scale(this.clearPressed
                    ? { x: DesignConstants.INTERACTION.SCALE_PRESSED_INPUT_ROW, y: DesignConstants.INTERACTION.SCALE_PRESSED_INPUT_ROW }
                    : { x: 1, y: 1 })
                  .onTouch((event: TouchEvent) => {
                    if (event.type === TouchType.Down) {
                      this.clearPressed = true;
                    } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
                      this.clearPressed = false;
                    }
                  })
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
                        LucideIcon({
                            icon: AppIcon.TIME,
                            iconSize: DesignConstants.FONT.SIZE.LG,
                            color: resolveLocationPickerPalette(this.resolvedThemeId).textTertiary
                          })
                      }
                      .width(DesignConstants.COMPONENT.AVATAR_SIZE.XS)
                      .height(DesignConstants.COMPONENT.AVATAR_SIZE.XS)
                      .justifyContent(FlexAlign.Center)
                      .backgroundColor(resolveLocationPickerPalette(this.resolvedThemeId).surfaceSubtle)
                      .borderRadius(DesignConstants.RADIUS.MD)

                      Text(item.locationName || '未命名位置')
                        .fontSize(DesignConstants.FONT.SIZE.MD)
                        .fontColor(resolveLocationPickerPalette(this.resolvedThemeId).textPrimary)
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
                    // 面板内行去三件套（surface 叠 surface + 边框 + 阴影），改 tint 行 + 按压反馈
                    .backgroundColor(resolveLocationPickerPalette(this.resolvedThemeId).surfaceSubtle)
                    .borderRadius(DesignConstants.RADIUS.LG)
                    .scale(this.pressedRowKey === `hist_m_${item.latitude}_${item.longitude}` ?
                      { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD } :
                      { x: 1, y: 1 })
                    .animation({ curve: curves.springMotion(
                      DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
                      DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
                    ) })
                    .onTouch((event: TouchEvent) => {
                      if (event.type === TouchType.Down) {
                        this.pressedRowKey = `hist_m_${item.latitude}_${item.longitude}`;
                      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
                        this.pressedRowKey = '';
                      }
                    })
                    .onClick(() => { this.onSelectHistory(item); })
                  }
                }, (item: AnchorHistoryItem) => `hist_m_${item.latitude}_${item.longitude}`)
              }
              .width('100%')
              .divider({ strokeWidth: 0, color: resolveLocationPickerPalette(this.resolvedThemeId).borderColor })
            } else {
              EmptyState({ icon: AppIcon.TIME, title: '暂无最近使用的地点', bare: true })
            }
          }
          .width('100%')
          // 滚动尾部呼吸间距（避免末行贴面板底缘，对齐 MomentsFeed 桌面分支语义）
          .padding({ bottom: DesignConstants.SPACING.LG })
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
        .height(DesignConstants.COMPONENT.BUTTON_HEIGHT_LG)
        .backgroundColor(resolveLocationPickerPalette(this.resolvedThemeId).accentBackground)
        .fontColor(resolveLocationPickerPalette(this.resolvedThemeId).accentText)
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
        .shadow(DesignConstants.SHADOW.SMALL)
        .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: resolveLocationPickerPalette(this.resolvedThemeId).borderColor })
        .margin({ top: DesignConstants.SPACING.SM })
        .onClick(this.onConfirm)
    }
    .height(this.panelHeight + this.resolveBottomInset())
    .clip(true)
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: 0,
      // 兜底 navIndicator 高度：部分设备/模拟器 safeAreaBottom 上报 0，按钮贴底被手势条遮挡
      bottom: this.resolveBottomInset()
    })
    .backgroundColor(resolveLocationPickerPalette(this.resolvedThemeId).surfaceBackground)
    .border({ width: { top: DesignConstants.BORDER_WIDTH.HAIRLINE }, color: resolveLocationPickerPalette(this.resolvedThemeId).borderColor })
    .borderRadius({ topLeft: DesignConstants.SHAPE.SHEET_TOP_RADIUS, topRight: DesignConstants.SHAPE.SHEET_TOP_RADIUS })
    .hitTestBehavior(HitTestMode.Default)
  }
}

