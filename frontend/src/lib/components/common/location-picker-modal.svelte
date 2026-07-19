<script lang="ts">
  import { onMount, tick, untrack } from 'svelte';
  import { t } from '$lib/i18n';
  import { assertSuccess } from '$lib/api/client';
  import {
    mapService,
    groupOverlappingRenderItems,
    isOverlapGroupRenderId,
    OVERLAP_CLUSTER_MAX_ZOOM
  } from '$lib/api/services/map';
  import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
  import { locationStore } from '$lib/stores/location.svelte';
  import { toast } from '$lib/stores/toast.svelte';
  import { getAMapVisibleRegion, loadAMap, loadAMapPlugin } from '$lib/utils/amap';
  import type {
    MapHeatBucket,
    MapHeatCanvas,
    MapHotspotItem,
    MapHotspotsCanvas,
    MapViewport,
    MapCanvasRequest,
    MapCanvasResponse,
    HotspotObjectType
  } from '$lib/types/api';
  import {
    getAnchorHistory,
    addToAnchorHistory,
    clearAnchorHistory,
    type AnchorHistoryItem
  } from '$lib/utils/anchor-history';
  import * as Dialog from '$lib/components/ui/dialog';
  import { Button } from '$lib/components/ui/button';
  import MapPin from 'lucide-svelte/icons/map-pin';
  import Plus from 'lucide-svelte/icons/plus';
  import Minus from 'lucide-svelte/icons/minus';
  import LocateFixed from 'lucide-svelte/icons/locate-fixed';
  import Spinner from '$lib/components/common/spinner.svelte';
  import ChevronUp from 'lucide-svelte/icons/chevron-up';
  import ChevronDown from 'lucide-svelte/icons/chevron-down';
  import Maximize2 from 'lucide-svelte/icons/maximize-2';
  import Minimize2 from 'lucide-svelte/icons/minimize-2';
  import { goto } from '$app/navigation';
  import XIcon from 'lucide-svelte/icons/x';
  import { browser } from '$app/environment';
  import Share2 from 'lucide-svelte/icons/share-2';
  import Search from 'lucide-svelte/icons/search';
  import ListFilter from 'lucide-svelte/icons/list-filter';
  import X from 'lucide-svelte/icons/x';
  import Clock from 'lucide-svelte/icons/clock';
  import Trash2 from 'lucide-svelte/icons/trash-2';
  import CreateLocationShareModal from './create-location-share-modal.svelte';
  import AnchorSummaryCard from './anchor-summary-card.svelte';
  import { useKeyboardFocus } from '$lib/attachments';
  import { configStore } from '$lib/stores/config.svelte';
  import { toGcjLngLat, gcj02ToWgs84 } from '$lib/utils/geo';
  import { mode as themeMode } from 'mode-watcher';

  type LocationPickerMode = 'anchor' | 'location_pick';

  export interface LocationPickerResult {
    latitude: number;
    longitude: number;
    locationName: string;
    radius: number;
  }

  interface LocationPickerInitialLocation {
    latitude: number;
    longitude: number;
    locationName?: string | null;
    radius?: number | null;
  }

  interface Props {
    open: boolean;
    mode?: LocationPickerMode;
    initialLocation?: LocationPickerInitialLocation | null;
    onConfirm?: (value: LocationPickerResult) => void;
    onOpenChange?: (open: boolean) => void;
  }

  let {
    open = $bindable(),
    mode = 'anchor',
    initialLocation = null,
    onConfirm,
    onOpenChange
  }: Props = $props();

  let mapContainer = $state<HTMLDivElement | null>(null);
  let map: AMap.Map | null = null;
  let geocoder: AMap.Geocoder | null = null;
  let isLoadingMap = $state(false);
  let isMapAvailable = $state(true);
  let isLocating = $state(false);
  let locationName = $state(mapAnchorStore.name);
  let latitude = $state(mapAnchorStore.latitude);
  let longitude = $state(mapAnchorStore.longitude);
  let reverseGeocodeTimer: number | null = null;
  let renderMapTimer: number | null = null;
  let mapMoveFrame: number | null = null;
  let activeRenderRequestId = 0;
  let lastRenderedViewportSignature = '';
  let lastHotspotsSignature = '';
  let lastMapZoom = 14;
  let renderMarkers: AMap.Marker[] = [];

  let selectedMarker: AMap.Marker | null = null;
  let hasInit = $state(false);
  let hasSyncedFromAnchor = $state(false);
  let modalRoot = $state<HTMLDivElement | null>(null);
  const LONG_PRESS_DURATION_MS = 700;
  const GESTURE_HINT_DURATION_MS = 4000;
  const MAP_RENDER_IDLE_MS = 250;
  const GESTURE_HINT_COMBINED = $derived(t('locationPicker.gestureHintCombined'));
  const GESTURE_HINT_TOUCH = $derived(t('locationPicker.gestureHintTouch'));
  const GESTURE_HINT_POINTER = $derived(t('locationPicker.gestureHintPointer'));
  const currentMode = $derived(themeMode.current);
  let suppressNextMapClick = false;

  // 低缩放级别下的行政区聚合展示阈值
  const ADMIN_REGION_ZOOM_THRESHOLD = 10;
  type RegionKey = string;
  interface RegionMeta {
    key: RegionKey;
    name: string;
    adcode: string;
    center: { lat: number; lng: number };
    centerIsOfficial?: boolean;
  }
  const regionMetaCache = new Map<RegionKey, RegionMeta>();
  const regionResolveCache = new Map<string, RegionKey>();
  const regionResolvePromiseCache = new Map<string, Promise<RegionKey | null>>();
  const regionCenterPromiseCache = new Map<RegionKey, Promise<RegionMeta>>();
  let districtSearch: AMap.DistrictSearch | null = null;

  type ClusterPreviewKind = 'server_cluster' | 'overlap_group';

  type RenderItem = MapHeatBucket & { render_id: string };

  interface ClusterPreviewState {
    kind: ClusterPreviewKind;
    render_id: string;
    title: string;
    description: string;
    count: number;
    latitude: number;
    longitude: number;
    zoomTarget: number;
    items: RenderItem[];
  }

  function getGestureHintMessage(): string {
    if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
      return GESTURE_HINT_COMBINED;
    }
    const prefersCoarsePointer = window.matchMedia('(pointer: coarse)').matches;
    const prefersHover = window.matchMedia('( hover)').matches;
    if (prefersCoarsePointer && !prefersHover) {
      return GESTURE_HINT_TOUCH;
    }
    if (!prefersCoarsePointer && prefersHover) {
      return GESTURE_HINT_POINTER;
    }
    return GESTURE_HINT_COMBINED;
  }

  let gestureHintVisible = $state(false);
  let gestureHintMessage = $state(getGestureHintMessage());
  let placementHintVisible = $state(false);
  let placementHintMessage = $state('');
  let placementHintTimer: number | null = null;
  let gestureHintTimer: number | null = null;
  let longPressTimer: number | null = null;
  let lastTouchLngLat: { lat: number; lng: number } | null = null;
  let touchMovedDuringLongPress = false;
  let detachTouchListeners: (() => void) | null = null;
  let detachContextMenuListener: (() => void) | null = null;
  let detachOutsidePointerListener: (() => void) | null = null;

  let searchRadius = $state(mapAnchorStore.radius);
  let isBottomSheetExpanded = $state(false);
  let showShareModal = $state(false);
  let showFilterPanel = $state(false);
  let filterPanelElement = $state<HTMLDivElement | null>(null);
  let showUsers = $state(true);
  let showGroups = $state(true);
  let showMoments = $state(true);
  let latestRenderItems = $state.raw<RenderItem[]>([]);
  let overlapGroupItems = $state.raw<Map<string, RenderItem[]>>(new Map());
  let clusterPreview = $state<ClusterPreviewState | null>(null);
  let hotspotPreviewItems = $state.raw<MapHotspotItem[]>([]);

  // 搜索相关状态
  let showSearch = $state(false);
  let searchText = $state('');
  let isSearching = $state(false);
  let searchInput = $state<HTMLInputElement | null>(null);
  let searchResults = $state.raw<SearchResultItem[]>([]);
  let searchDebounceTimer: number | null = null;
  let activeSearchRequestId = 0;
  let placeSearch: AMap.PlaceSearch | null = null;

  // 锚点历史
  let anchorHistory = $state.raw<AnchorHistoryItem[]>([]);

  // 桌面端对话框全屏切换状态
  let isFullscreen = $state(false);

  // 手势提示持久化关闭标记（localStorage）
  const GESTURE_HINT_DISMISS_KEY = 'hotspot_location_picker_gesture_hint_dismissed';
  let gestureHintDismissed = $state(false);

  const isAnchorMode = $derived(mode === 'anchor');
  const modalTitle = $derived(isAnchorMode ? t('locationPicker.modalTitleAnchor') : t('locationPicker.modalTitleLocation'));
  const confirmButtonText = $derived(isAnchorMode ? t('locationPicker.confirmButtonAnchor') : t('locationPicker.confirmButtonLocation'));
  const confirmSuccessText = $derived(isAnchorMode ? t('locationPicker.confirmSuccessAnchor') : t('locationPicker.confirmSuccessLocation'));
  const selectionDescription = $derived(isAnchorMode ? t('locationPicker.selectionDescriptionAnchor') : t('locationPicker.selectionDescriptionLocation'));
  const mapModeLabel = $derived(isAnchorMode ? t('locationPicker.mapModeAnchor') : t('locationPicker.mapModeLocation'));
  const activeFilterCount = $derived([showUsers, showGroups, showMoments].filter(Boolean).length);

  interface SearchResultItem {
    name: string;
    address: string;
    latitude: number;
    longitude: number;
  }

  type MapClickEvent = AMap.MapClickEvent;

  const DEFAULT_MARKER_ANCHOR: AMap.MarkerAnchor = 'bottom-center';
  const PLACEMENT_HINT_DEFAULT = $derived(t('locationPicker.placementHintDefault'));

  // 同坐标重叠 marker 的处理阈值：
  // - 个数 ≤ FAN_OUT_LIMIT 时，在高缩放下像素展开为独立 marker；
  // - 超过则不再尝试展开（铺不开也读不清），统一合并为「数量气泡」聚合点。
  const FAN_OUT_LIMIT = 6;

  // 展开半径按 marker 真实宽度取值，确保相邻 marker 边缘不互相压叠。
  function buildFanOutOffset(index: number, total: number): [number, number] {
    if (total === 2) {
      return index === 0 ? [0, -18] : [0, 18];
    }
    if (total === 3) {
      if (index === 0) return [0, -20];
      if (index === 1) return [0, 0];
      return [0, 20];
    }
    const ringRadius = 30;
    const angle = (Math.PI * 2 * index) / total - Math.PI / 2;
    return [Math.round(Math.cos(angle) * ringRadius), Math.round(Math.sin(angle) * ringRadius)];
  }

  function showGestureHint(message = getGestureHintMessage()) {
    if (gestureHintDismissed) return;
    gestureHintMessage = message;
    gestureHintVisible = true;
    if (gestureHintTimer) {
      clearTimeout(gestureHintTimer);
    }
    gestureHintTimer = window.setTimeout(() => {
      gestureHintVisible = false;
      gestureHintTimer = null;
    }, GESTURE_HINT_DURATION_MS);
  }

  function hideGestureHint() {
    gestureHintVisible = false;
    if (gestureHintTimer) {
      clearTimeout(gestureHintTimer);
      gestureHintTimer = null;
    }
  }

  function dismissGestureHintPermanently() {
    gestureHintVisible = false;
    gestureHintDismissed = true;
    if (browser) {
      try {
        window.localStorage.setItem(GESTURE_HINT_DISMISS_KEY, '1');
      } catch {
        // ignore
      }
    }
  }

  function showPlacementHint(message = PLACEMENT_HINT_DEFAULT) {
    placementHintMessage = message;
    placementHintVisible = true;
    if (placementHintTimer) {
      clearTimeout(placementHintTimer);
    }
    placementHintTimer = window.setTimeout(() => {
      placementHintVisible = false;
      placementHintTimer = null;
    }, 2000);
  }

  function createHotspotMarkerContent(svgUrl: string): string {
    return `<div class="map-icon-marker"><img src="${svgUrl}" alt="" /></div>`;
  }

  function disposeRenderMarkers(markers: AMap.Marker[]) {
    for (const marker of markers) {
      marker.setMap(null);
      marker.remove();
    }
  }

  function clearRenderMarkers() {
    disposeRenderMarkers(renderMarkers);
    renderMarkers = [];
    hotspotPreviewItems = [];
    lastHotspotsSignature = '';
  }

  function replaceRenderMarkers(nextMarkers: AMap.Marker[], clearHotspotPreview = true) {
    const previousMarkers = renderMarkers;

    renderMarkers = nextMarkers;
    if (clearHotspotPreview) {
      hotspotPreviewItems = [];
    }

    for (const marker of nextMarkers) {
      marker.setMap(map);
    }

    disposeRenderMarkers(previousMarkers);
  }

  function ensureSelectedMarker() {
    if (!map || selectedMarker) return;
    const [gcjLng, gcjLat] = toGcjLngLat({
      latitude,
      longitude
    });
    selectedMarker = new AMap.Marker({
      position: [gcjLng, gcjLat],
      content: createHotspotMarkerContent('/map/marker_center_pin.svg'),
      anchor: DEFAULT_MARKER_ANCHOR,
      offset: new AMap.Pixel(0, 0),
      clickable: false,
      map
    });
  }

  function updateSelectedMarker() {
    if (!map) return;
    ensureSelectedMarker();
    if (!selectedMarker) return;
    const [gcjLng, gcjLat] = toGcjLngLat({
      latitude,
      longitude
    });
    selectedMarker.setPosition([gcjLng, gcjLat]);
  }

  function clearSelectedMarker() {
    selectedMarker?.setMap(null);
    selectedMarker?.remove();
    selectedMarker = null;
  }

  function closeClusterPreview() {
    clusterPreview = null;
    if (hotspotPreviewItems.length === 0 && !showSearch && !showFilterPanel) {
      isBottomSheetExpanded = false;
    }
  }

  function openHotspotPreview(items: MapHotspotItem[]) {
    showSearch = false;
    showFilterPanel = false;
    closeClusterPreview();
    hotspotPreviewItems = items;
    isBottomSheetExpanded = true;
  }

  function toggleFilterPanel() {
    const nextOpen = !showFilterPanel;
    if (nextOpen) {
      showSearch = false;
      hotspotPreviewItems = [];
      closeClusterPreview();
      isBottomSheetExpanded = true;
    }
    showFilterPanel = nextOpen;
  }

  function handleMapSurfaceClick() {
    if (suppressNextMapClick) {
      suppressNextMapClick = false;
      return;
    }
    showSearch = false;
    showFilterPanel = false;
    hotspotPreviewItems = [];
    closeClusterPreview();
  }

  function isClickInsideFilterControls(event: PointerEvent) {
    const target = event.target;
    if (!(target instanceof Node)) {
      return false;
    }
    if (filterPanelElement?.contains(target)) {
      return true;
    }
    if (!(target instanceof HTMLElement)) {
      return false;
    }
    return Boolean(target.closest('[data-location-filter-trigger="true"]'));
  }

  function registerOutsidePointerListener() {
    detachOutsidePointerListener?.();
    const handler = (event: PointerEvent) => {
      if (!open || !showFilterPanel) {
        return;
      }
      if (isClickInsideFilterControls(event)) {
        return;
      }
      showFilterPanel = false;
    };
    window.addEventListener('pointerdown', handler, true);
    detachOutsidePointerListener = () => {
      window.removeEventListener('pointerdown', handler, true);
      detachOutsidePointerListener = null;
    };
  }

  function getViewportSignature(viewport: MapViewport) {
    return [
      viewport.bounds.north.toFixed(6),
      viewport.bounds.south.toFixed(6),
      viewport.bounds.east.toFixed(6),
      viewport.bounds.west.toFixed(6),
      viewport.zoom.toFixed(2)
    ].join('|');
  }

  function buildViewport(): MapViewport | null {
    if (!map) return null;
    const visibleRegion = getAMapVisibleRegion(map);
    if (!visibleRegion) return null;
    const northEastWgs = gcj02ToWgs84({
      latitude: visibleRegion.bounds.north,
      longitude: visibleRegion.bounds.east
    });
    const southWestWgs = gcj02ToWgs84({
      latitude: visibleRegion.bounds.south,
      longitude: visibleRegion.bounds.west
    });
    const north = clampLatitude(northEastWgs.latitude);
    const south = clampLatitude(southWestWgs.latitude);
    let east = normalizeLongitude(northEastWgs.longitude);
    let west = normalizeLongitude(southWestWgs.longitude);
    if (Number.isFinite(east) && Number.isFinite(west) && east < west) {
      const tmp = east;
      east = west;
      west = tmp;
    }
    const viewport: MapViewport = {
      bounds: {
        north,
        south,
        east,
        west
      },
      zoom: Math.max(1, Math.min(20, visibleRegion.zoom))
    };
    return viewport;
  }

  // 热力格网聚合单元统一用混合类型图标展示
  const CLUSTER_MARKER_ICON = '/map/marker_mixed_cluster.svg';

  function getHotspotMarkerIcon(item: MapHotspotItem): string {
    if (item.object_type === 'user') return '/map/marker_user.svg';
    if (item.object_type === 'group') return '/map/marker_group.svg';
    if (item.object_type === 'moment') return '/map/marker_moment.svg';
    return '/map/marker_user.svg';
  }

  function shouldDisplayItem(_item: RenderItem) {
    // 目前 /map/render 仅返回热力格网统计项，是否显示由筛选开关整体决定
    return showUsers || showGroups || showMoments;
  }

  function resolveMapFilter(): { data_type?: 'user' | 'group' | 'moment' | 'all' } | undefined {
    const noneSelected = !showUsers && !showGroups && !showMoments;
    const allSelected = showUsers && showGroups && showMoments;
    if (noneSelected || allSelected) {
      return undefined;
    }
    if (showUsers && !showGroups && !showMoments) {
      return { data_type: 'user' };
    }
    if (!showUsers && showGroups && !showMoments) {
      return { data_type: 'group' };
    }
    if (!showUsers && !showGroups && showMoments) {
      return { data_type: 'moment' };
    }
    return { data_type: 'all' };
  }

  function applyCurrentFilter() {
    // 热力图模式下，本地根据筛选条件对现有聚合结果做一次快速重渲染；
    // 热点模式下，筛选条件会导致重新请求 /map/hotspots，由后端过滤，故无需本地再过滤。
    renderMapItems(latestRenderItems.filter((item) => shouldDisplayItem(item)));
  }

  function clampLatitude(lat: number): number {
    if (!Number.isFinite(lat)) return lat;
    // 避免极区附近数值异常，约束到常见 Web 地图用的可视范围
    return Math.max(-85, Math.min(85, lat));
  }

  function normalizeLongitude(lng: number): number {
    if (!Number.isFinite(lng)) return lng;
    // 归一化到 [-180, 180]，避免出现 229 之类超过 180 的经度
    const normalized = ((((lng + 180) % 360) + 360) % 360) - 180;
    return normalized;
  }

  function toggleRenderFilter(kind: 'user' | 'group' | 'moment') {
    if (kind === 'user') showUsers = !showUsers;
    if (kind === 'group') showGroups = !showGroups;
    if (kind === 'moment') showMoments = !showMoments;

    lastRenderedViewportSignature = '';
    lastHotspotsSignature = '';
    if (!showUsers && !showGroups && !showMoments) {
      latestRenderItems = [];
      overlapGroupItems = new Map();
      closeClusterPreview();
      clearRenderMarkers();
      return;
    }

    // 先在前端根据当前筛选条件重新渲染已有 items，确保 UI 立即响应
    applyCurrentFilter();
    void fetchMapRenderData();
  }

  function getMarkerBadgeText(item: RenderItem) {
    return t('locationPicker.itemsCount', { count: item.total_count });
  }

  /** 按 breakdown 推断该聚合项的主导类型，无 breakdown 时返回 null */
  function getDominantType(item: RenderItem): HotspotObjectType | null {
    const b = item.breakdown;
    if (!b) return null;
    const entries: Array<[HotspotObjectType, number]> = [
      ['user', b.user ?? 0],
      ['group', b.group ?? 0],
      ['moment', b.moment ?? 0]
    ];
    entries.sort((a, z) => z[1] - a[1]);
    return entries[0]![1] > 0 ? entries[0]![0] : null;
  }

  function getItemTypeLabel(item: RenderItem) {
    switch (getDominantType(item)) {
      case 'user':
        return t('locationPicker.itemLabelUser');
      case 'group':
        return t('locationPicker.itemLabelGroup');
      case 'moment':
        return t('locationPicker.itemLabelMoment');
      default:
        return t('locationPicker.itemLabelDefault');
    }
  }

  function getItemBadgeClass(item: RenderItem) {
    switch (getDominantType(item)) {
      case 'user':
        return 'badge-user';
      case 'group':
        return 'badge-group';
      case 'moment':
        return 'badge-moment';
      default:
        return '';
    }
  }

  function buildOverlapPreviewDescription(items: RenderItem[]) {
    const total = items.reduce((sum, item) => sum + item.total_count, 0);
    return t('locationPicker.contentCount', { count: total });
  }

  function openClusterPreview(item: RenderItem) {
    const overlapItems = overlapGroupItems.get(item.render_id) ?? [];
    const isOverlapGroup = isOverlapGroupRenderId(item.render_id) && overlapItems.length > 0;
    const currentZoom = map?.getZoom() ?? 14;
    const baseCount = item.total_count || overlapItems.reduce((sum, it) => sum + it.total_count, 0);

    hotspotPreviewItems = [];
    isBottomSheetExpanded = true;
    clusterPreview = {
      kind: isOverlapGroup ? 'overlap_group' : 'server_cluster',
      render_id: item.render_id,
      title: t('locationPicker.contentCount', { count: baseCount }),
      description: isOverlapGroup
        ? `${buildOverlapPreviewDescription(overlapItems)}。${t('locationPicker.overlapDescSuffix')}`
        : t('locationPicker.clusterDesc'),
      count: baseCount,
      latitude: item.location.latitude,
      longitude: item.location.longitude,
      zoomTarget: baseCount > 100 ? Math.min(20, currentZoom + 2) : Math.min(20, currentZoom + 3),
      items: overlapItems
    };
  }

  function zoomToClusterPreview() {
    if (!clusterPreview || !map) {
      return;
    }
    const [gcjLng, gcjLat] = toGcjLngLat({
      latitude: clusterPreview.latitude,
      longitude: clusterPreview.longitude
    });
    map.setCenter([gcjLng, gcjLat]);
    map.setZoom(clusterPreview.zoomTarget);
    showPlacementHint(
      clusterPreview.kind === 'overlap_group' ? t('locationPicker.placementHintOverlapExpanded') : t('locationPicker.placementHintClusterExpanded')
    );
    closeClusterPreview();
  }

  function focusPreviewEntity(item: RenderItem) {
    closeClusterPreview();
    focusRenderItem(item);
  }

  function createMarkerForItem(
    item: RenderItem,
    offset?: AMap.Pixel | [number, number],
    attachToMap = true
  ) {
    if (!map) return null;
    const position = toGcjLngLat({
      latitude: item.location.latitude,
      longitude: item.location.longitude
    });

    const marker = new AMap.Marker({
      position,
      content: createHotspotMarkerContent(CLUSTER_MARKER_ICON),
      anchor: DEFAULT_MARKER_ANCHOR,
      offset,
      clickable: true,
      map: attachToMap ? map : undefined
    });

    marker.on('click', () => {
      suppressNextMapClick = true;
      focusRenderItemFromMarkerClick(item);
    });

    marker.setLabel?.({
      content: getMarkerBadgeText(item),
      direction: 'top'
    });

    return marker;
  }

  // 地图上的 marker 点击只用于“浏览”聚合信息，不改变位置选择器的选中坐标；
  // 真正修改选中位置仍然只通过双击/右键/长按/搜索结果/历史记录/拖动结束等显式操作完成。
  function focusRenderItemFromMarkerClick(item: RenderItem) {
    showSearch = false;
    showFilterPanel = false;
    // 仅展示聚合预览信息，不移动相机
    openClusterPreview(item);
    showPlacementHint(t('locationPicker.placementHintFocusedSelectable', { count: item.total_count }));
  }

  function focusRenderItem(item: RenderItem) {
    showSearch = false;
    showFilterPanel = false;
    openClusterPreview(item);
    const [gcjLng, gcjLat] = toGcjLngLat({
      latitude: item.location.latitude,
      longitude: item.location.longitude
    });
    map?.setCenter([gcjLng, gcjLat]);
    showPlacementHint(t('locationPicker.placementHintFocused', { count: item.total_count }));
  }

  async function resolveClusterRegion(item: RenderItem): Promise<RegionKey | null> {
    const cacheKey = item.render_id;
    const cached = regionResolveCache.get(cacheKey);
    if (cached) return cached;

    const pending = regionResolvePromiseCache.get(cacheKey);
    if (pending) return pending;

    const promise = (async () => {
      const [gcjLng, gcjLat] = toGcjLngLat({
        latitude: item.location.latitude,
        longitude: item.location.longitude
      });

      await ensureGeocoder();
      if (!geocoder) return null;

      const key = await new Promise<RegionKey | null>((resolve) => {
        geocoder!.getAddress([gcjLng, gcjLat], (status: string, result: AMap.GeocodeResult) => {
          if (status !== 'complete' || !result.regeocode) {
            resolve(null);
            return;
          }
          const comp = result.regeocode.addressComponent;
          const rawAdcode: string | undefined = comp?.adcode ?? comp?.citycode;
          const rawCity: string | string[] | undefined = comp?.city;
          const province: string | undefined = comp?.province;
          const district: string | undefined = comp?.district;
          const city = Array.isArray(rawCity) ? rawCity[0] : rawCity;
          const adcode = rawAdcode;
          const name = city || province || district;
          if (!adcode || !name) {
            resolve(null);
            return;
          }
          const regionKey: RegionKey = `city:${adcode}`;
          if (!regionMetaCache.has(regionKey)) {
            regionMetaCache.set(regionKey, {
              key: regionKey,
              name,
              adcode,
              center: { lat: gcjLat, lng: gcjLng },
              centerIsOfficial: false
            });
          }
          resolve(regionKey);
        });
      });

      if (key) {
        regionResolveCache.set(cacheKey, key);
      }
      return key;
    })();

    regionResolvePromiseCache.set(cacheKey, promise);
    try {
      return await promise;
    } finally {
      regionResolvePromiseCache.delete(cacheKey);
    }
  }

  async function ensureRegionCenter(meta: RegionMeta): Promise<RegionMeta> {
    if (meta.centerIsOfficial) return meta;

    const pending = regionCenterPromiseCache.get(meta.key);
    if (pending) return pending;

    const promise = (async () => {
      await ensureDistrictSearch();
      const search = districtSearch;
      if (!search) return meta;
      const updated = await new Promise<RegionMeta>((resolve) => {
        search.search(meta.adcode, (status: string, result: AMap.DistrictSearchResult) => {
          if (status !== 'complete' || !result.districts?.length) {
            resolve(meta);
            return;
          }
          const district = result.districts[0];
          const center = district?.center;
          if (!center) {
            resolve(meta);
            return;
          }
          const next: RegionMeta = {
            ...meta,
            center: { lat: center.getLat(), lng: center.getLng() },
            centerIsOfficial: true
          };
          resolve(next);
        });
      });
      regionMetaCache.set(updated.key, updated);
      return updated;
    })();

    regionCenterPromiseCache.set(meta.key, promise);
    try {
      return await promise;
    } finally {
      regionCenterPromiseCache.delete(meta.key);
    }
  }

  async function renderRegionClusters(items: RenderItem[]) {
    if (!map) return;
    const regionEntries = await Promise.all(
      items.map(async (item) => ({
        item,
        regionKey: await resolveClusterRegion(item)
      }))
    );

    const groups = new Map<RegionKey, { items: RenderItem[]; total: number }>();
    for (const entry of regionEntries) {
      if (!entry.regionKey) continue;
      const group = groups.get(entry.regionKey) ?? { items: [], total: 0 };
      group.items.push(entry.item);
      group.total += entry.item.total_count;
      groups.set(entry.regionKey, group);
    }

    const metaEntries = await Promise.all(
      Array.from(groups.entries()).map(async ([key, group]) => {
        const baseMeta = regionMetaCache.get(key);
        if (!baseMeta) {
          return null;
        }
        const meta = await ensureRegionCenter(baseMeta);
        return { key, group, meta };
      })
    );

    const markers: AMap.Marker[] = [];
    for (const entry of metaEntries) {
      if (!entry) continue;
      const { key, group, meta } = entry;
      const position: [number, number] = [meta.center.lng, meta.center.lat];
      const marker = new AMap.Marker({
        position,
        content: createHotspotMarkerContent('/map/marker_mixed_cluster.svg'),
        anchor: DEFAULT_MARKER_ANCHOR,
        clickable: true
      });
      marker.setLabel?.({
        content: `${meta.name} ${t('locationPicker.contentCount', { count: group.total })}`,
        direction: 'top'
      });
      marker.on('click', () => {
        clusterPreview = {
          kind: 'server_cluster',
          render_id: key,
          title: `${meta.name} ${t('locationPicker.contentCount', { count: group.total })}`,
          description: t('locationPicker.regionClusterDesc'),
          count: group.total,
          latitude: group.items[0]?.location.latitude ?? 0,
          longitude: group.items[0]?.location.longitude ?? 0,
          zoomTarget:
            group.total > 100
              ? Math.min(20, (map?.getZoom() ?? lastMapZoom) + 2)
              : Math.min(20, (map?.getZoom() ?? lastMapZoom) + 3),
          items: group.items
        };
      });
      markers.push(marker);
    }
    replaceRenderMarkers(markers);
  }

  async function renderMapItems(items: RenderItem[]) {
    if (!map) return;
    const zoom = map.getZoom() ?? lastMapZoom;
    if (zoom <= ADMIN_REGION_ZOOM_THRESHOLD) {
      await renderRegionClusters(items);
      updateSelectedMarker();
      return;
    }
    const { merged, groups } = groupOverlappingRenderItems(items, zoom);
    overlapGroupItems = groups;
    const currentCluster = clusterPreview;
    if (currentCluster && !items.some((item) => item.render_id === currentCluster.render_id)) {
      closeClusterPreview();
    }
    const nextMarkers = merged
      .map(({ item }) => createMarkerForItem(item, undefined, false))
      .filter((marker): marker is AMap.Marker => Boolean(marker));
    replaceRenderMarkers(nextMarkers);
    updateSelectedMarker();
  }

  async function fetchMapRenderData() {
    if (!map || !open || !isMapAvailable) return;
    if (!showUsers && !showGroups && !showMoments) {
      latestRenderItems = [];
      overlapGroupItems = new Map();
      closeClusterPreview();
      clearRenderMarkers();
      lastRenderedViewportSignature = '';
      return;
    }
    const viewport = buildViewport();
    if (!viewport) return;
    const filter = resolveMapFilter();
    const signature = `${getViewportSignature(viewport)}|${filter?.data_type ?? 'all'}`;
    if (signature === lastRenderedViewportSignature && renderMarkers.length > 0) {
      if (latestRenderItems.length > 0) {
        applyCurrentFilter();
      }
      return;
    }

    const requestId = ++activeRenderRequestId;
    try {
      const request: MapCanvasRequest = {
        viewport,
        filter: filter ? { data_type: filter.data_type } : undefined,
        pagination: { limit: 100 }
      };
      const response = await mapService.canvas(request);
      assertSuccess(response);
      if (requestId !== activeRenderRequestId) {
        return;
      }
      lastRenderedViewportSignature = signature;
      const canvas = (response.data ?? {}) as MapCanvasResponse;

      if (canvas.hotspots) {
        latestRenderItems = [];
        overlapGroupItems = new Map();

        const data = canvas.hotspots as MapHotspotsCanvas;
        const allItems: MapHotspotItem[] = data.items ?? [];
        const items: MapHotspotItem[] = allItems.filter((item: MapHotspotItem) => {
          if (item.object_type === 'user' && !showUsers) return false;
          if (item.object_type === 'group' && !showGroups) return false;
          if (item.object_type === 'moment' && !showMoments) return false;
          return true;
        });

        // 数据+zoom 签名——zoom 变化也会影响聚类结果（手动聚类不依赖 MarkerClusterer）。
        const currentZoom = map.getZoom() ?? lastMapZoom;
        const currentHotspotsSignature = `${currentZoom.toFixed(1)}::${items
          .map((i) => `${i.object_type}|${i.object_id}|${i.location.latitude.toFixed(4)}|${i.location.longitude.toFixed(4)}`)
          .join('::')}`;
        if (currentHotspotsSignature === lastHotspotsSignature && renderMarkers.length > 0) {
          return;
        }
        lastHotspotsSignature = currentHotspotsSignature;

        if (items.length > 0) {
          // 按 ~11m 精度分组
          const coordPrecision = 10000;
          const coordGroups = new Map<string, MapHotspotItem[]>();
          for (const item of items) {
            const key = `${Math.floor(item.location.latitude * coordPrecision)}_${Math.floor(item.location.longitude * coordPrecision)}`;
            const group = coordGroups.get(key);
            if (group) { group.push(item); } else { coordGroups.set(key, [item]); }
          }

          const markers: AMap.Marker[] = [];
          for (const [, groupItems] of coordGroups) {
            const representative = groupItems[0];
            if (!representative) continue;
            const [gcjLng, gcjLat] = toGcjLngLat(representative.location);

            if (groupItems.length === 1) {
              // 单个 item：直接用类型 SVG 图标
              const item = representative;
              const marker = new AMap.Marker({
                position: [gcjLng, gcjLat],
                content: createHotspotMarkerContent(getHotspotMarkerIcon(item)),
                anchor: DEFAULT_MARKER_ANCHOR,
                clickable: true
              });
              marker.setExtData?.(item);
              marker.on('click', () => openHotspotPreview([item]));
              markers.push(marker);
            } else if (groupItems.length <= FAN_OUT_LIMIT && currentZoom > OVERLAP_CLUSTER_MAX_ZOOM) {
              // 高缩放 + 少量同点：像素展开为独立 marker（半径已按 marker 尺寸放大，不互相压叠）
              groupItems.forEach((item, index) => {
                const offsetVal = buildFanOutOffset(index, groupItems.length);
                const marker = new AMap.Marker({
                  position: [gcjLng, gcjLat],
                  content: createHotspotMarkerContent(getHotspotMarkerIcon(item)),
                  anchor: DEFAULT_MARKER_ANCHOR,
                  offset: new AMap.Pixel(offsetVal[0], offsetVal[1]),
                  clickable: true
                });
                marker.setExtData?.(item);
                marker.on('click', () => openHotspotPreview([item]));
                markers.push(marker);
              });
            } else {
              // 较多同点 / 低缩放：合并为「数量气泡」聚合点（固定 3 头像 SVG 与数量脱节，改为大号数字气泡）
              const typeCount = { user: 0, group: 0, moment: 0 };
              for (const gi of groupItems) {
                if (gi.object_type === 'user') typeCount.user++;
                else if (gi.object_type === 'group') typeCount.group++;
                else if (gi.object_type === 'moment') typeCount.moment++;
              }
              // 主导类型：数量最多的；多种则视为 mixed
              const entries = Object.entries(typeCount) as Array<[string, number]>;
              entries.sort((a, b) => b[1] - a[1]);
              const [dominant, dominantCount] = entries[0]!;
              const hasMixed = entries.filter(([, c]) => c > 0).length > 1;
              const variant = dominantCount > 0 && !hasMixed ? dominant : 'mixed';
              const clusterItems = [...groupItems];

              const div = document.createElement('div');
              div.className = `map-cluster-bubble map-cluster-bubble--${variant}`;
              const num = document.createElement('span');
              num.className = 'map-cluster-bubble__num';
              num.textContent = String(groupItems.length);
              div.appendChild(num);

              const marker = new AMap.Marker({
                position: [gcjLng, gcjLat],
                content: div.outerHTML,
                anchor: DEFAULT_MARKER_ANCHOR,
                clickable: true
              });
              marker.on('click', () => openHotspotPreview(clusterItems));
              markers.push(marker);
            }
          }

          replaceRenderMarkers(markers);
        } else {
          clearRenderMarkers();
        }
      } else if (canvas.heatmap) {
        const data = canvas.heatmap as MapHeatCanvas;
        const buckets: MapHeatBucket[] = data.items ?? [];
        latestRenderItems = buckets.map(
          (bucket: MapHeatBucket, index: number): RenderItem => ({
            ...bucket,
            render_id: `${index}_${bucket.location.latitude.toFixed(6)}_${bucket.location.longitude.toFixed(6)}`
          })
        );
        applyCurrentFilter();
      }
    } catch {
      if (requestId === activeRenderRequestId) {
        toast.error(t('locationPicker.renderLoadFailed'));
      }
    }
  }

  function scheduleMapRender() {
    if (renderMapTimer) {
      clearTimeout(renderMapTimer);
    }
    renderMapTimer = window.setTimeout(() => {
      renderMapTimer = null;
      void fetchMapRenderData();
    }, MAP_RENDER_IDLE_MS);
  }

  async function moveToCoordinate(
    mapLat: number,
    mapLng: number,
    options?: { hintMessage?: string }
  ) {
    cancelReverseGeocodeTimer();
    closeClusterPreview();
    const wgs = gcj02ToWgs84({
      latitude: mapLat,
      longitude: mapLng
    });
    latitude = wgs.latitude;
    longitude = wgs.longitude;
    if (map) {
      map.setCenter([mapLng, mapLat]);
    }
    updateSelectedMarker();
    await reverseGeocode(latitude, longitude);
    hideGestureHint();
    showPlacementHint(options?.hintMessage ?? PLACEMENT_HINT_DEFAULT);
  }

  function getLngLatFromTouch(event: TouchEvent): { lat: number; lng: number } | null {
    if (!map || !mapContainer) return null;
    const touch = event.touches[0] ?? event.changedTouches[0];
    if (!touch) return null;
    const rect = mapContainer.getBoundingClientRect();
    const pixel = {
      x: touch.clientX - rect.left,
      y: touch.clientY - rect.top
    };
    const lnglat = map.containerToLngLat(pixel);
    if (!lnglat) return null;
    return { lat: lnglat.getLat(), lng: lnglat.getLng() };
  }

  const handleTouchStart = (event: TouchEvent) => {
    const lnglat = getLngLatFromTouch(event);
    if (!lnglat) return;
    lastTouchLngLat = lnglat;
    touchMovedDuringLongPress = false;
    if (longPressTimer) {
      clearTimeout(longPressTimer);
    }
    longPressTimer = window.setTimeout(() => {
      if (touchMovedDuringLongPress || !lastTouchLngLat) return;
      void moveToCoordinate(lastTouchLngLat.lat, lastTouchLngLat.lng, {
        hintMessage: isAnchorMode ? t('locationPicker.placementHintLocatedConfirm') : t('locationPicker.placementHintDefault')
      });
    }, LONG_PRESS_DURATION_MS);
  };

  const handleTouchMove = () => {
    touchMovedDuringLongPress = true;
    if (longPressTimer) {
      clearTimeout(longPressTimer);
      longPressTimer = null;
    }
  };

  const handleTouchEnd = () => {
    if (longPressTimer) {
      clearTimeout(longPressTimer);
      longPressTimer = null;
    }
    lastTouchLngLat = null;
  };

  function registerTouchListeners() {
    detachTouchListeners?.();
    if (!mapContainer) return;
    mapContainer.addEventListener('touchstart', handleTouchStart, {
      passive: true
    });
    mapContainer.addEventListener('touchmove', handleTouchMove, {
      passive: true
    });
    mapContainer.addEventListener('touchend', handleTouchEnd, {
      passive: true
    });
    detachTouchListeners = () => {
      mapContainer?.removeEventListener('touchstart', handleTouchStart);
      mapContainer?.removeEventListener('touchmove', handleTouchMove);
      mapContainer?.removeEventListener('touchend', handleTouchEnd);
      detachTouchListeners = null;
    };
  }

  function registerContextMenuBlocker() {
    detachContextMenuListener?.();
    if (!mapContainer) return;
    const handler = (event: MouseEvent) => {
      event.preventDefault();
    };
    mapContainer.addEventListener('contextmenu', handler);
    detachContextMenuListener = () => {
      mapContainer?.removeEventListener('contextmenu', handler);
      detachContextMenuListener = null;
    };
  }

  async function ensureGeocoder() {
    if (geocoder) return;
    await loadAMapPlugin(['AMap.Geocoder']);
    geocoder = new AMap.Geocoder();
  }

  async function ensureDistrictSearch() {
    if (districtSearch) return;
    await loadAMapPlugin(['AMap.DistrictSearch']);
    districtSearch = new AMap.DistrictSearch({
      subdistrict: 0,
      extensions: 'base'
    });
  }

  // ========== 搜索功能 ==========

  async function toggleSearch() {
    if (searchDebounceTimer) {
      clearTimeout(searchDebounceTimer);
      searchDebounceTimer = null;
    }
    activeSearchRequestId += 1;
    showSearch = !showSearch;
    if (showSearch) {
      showFilterPanel = false;
      hotspotPreviewItems = [];
      closeClusterPreview();
      isBottomSheetExpanded = true;
      searchText = '';
      searchResults = [];
      isSearching = false;
      await tick();
      searchInput?.focus();
    }
  }

  async function ensurePlaceSearch() {
    if (placeSearch) return;
    await loadAMapPlugin(['AMap.PlaceSearch']);
    placeSearch = new AMap.PlaceSearch({
      pageSize: 10,
      extensions: 'base'
    });
  }

  async function doSearch() {
    const keyword = searchText.trim();
    if (keyword.length === 0) return;

    const requestId = ++activeSearchRequestId;
    isSearching = true;
    searchResults = [];
    try {
      await ensurePlaceSearch();
      if (!placeSearch) return;

      const results = await new Promise<SearchResultItem[]>((resolve) => {
        const centerForSearch = toGcjLngLat({
          latitude,
          longitude
        });
        placeSearch!.searchNearBy(
          keyword,
          centerForSearch,
          configStore.searchRadiusMax, // 使用后端配置的搜索半径上限（米）
          (status: string, result: AMap.PlaceSearchResult) => {
            if (status === 'complete' && result.poiList?.pois?.length) {
              resolve(
                result.poiList.pois.map((poi) => {
                  const wgs = gcj02ToWgs84({
                    latitude: poi.location.getLat(),
                    longitude: poi.location.getLng()
                  });
                  return {
                    name: poi.name,
                    address: [poi.adname, poi.address].filter(Boolean).join(' '),
                    latitude: wgs.latitude,
                    longitude: wgs.longitude
                  };
                })
              );
            } else {
              resolve([]);
            }
          }
        );
      });

      if (requestId !== activeSearchRequestId) {
        return;
      }

      if (results.length > 0) {
        searchResults = results;
      } else {
        toast.error(t('locationPicker.searchNotFound'));
      }
    } catch {
      if (requestId !== activeSearchRequestId) {
        return;
      }
      toast.error(t('locationPicker.searchFailed'));
    } finally {
      if (requestId === activeSearchRequestId) {
        isSearching = false;
      }
    }
  }

  function selectSearchResult(result: SearchResultItem) {
    if (searchDebounceTimer) {
      clearTimeout(searchDebounceTimer);
      searchDebounceTimer = null;
    }
    activeSearchRequestId += 1;
    isSearching = false;
    cancelReverseGeocodeTimer();
    latitude = result.latitude;
    longitude = result.longitude;
    locationName = result.name;
    closeClusterPreview();
    updateSelectedMarker();
    const [gcjLng, gcjLat] = toGcjLngLat({
      latitude: result.latitude,
      longitude: result.longitude
    });
    map?.setCenter([gcjLng, gcjLat]);
    showSearch = false;
    isBottomSheetExpanded = false;
    showPlacementHint(t('locationPicker.placementHintSearchResult'));
    searchText = '';
    searchResults = [];
  }

  function selectHistoryItem(item: AnchorHistoryItem) {
    if (searchDebounceTimer) {
      clearTimeout(searchDebounceTimer);
      searchDebounceTimer = null;
    }
    activeSearchRequestId += 1;
    isSearching = false;
    cancelReverseGeocodeTimer();
    latitude = item.latitude;
    longitude = item.longitude;
    locationName = item.locationName;
    closeClusterPreview();
    updateSelectedMarker();
    const [gcjLng, gcjLat] = toGcjLngLat({
      latitude: item.latitude,
      longitude: item.longitude
    });
    map?.setCenter([gcjLng, gcjLat]);
    isBottomSheetExpanded = false;
    showPlacementHint(t('locationPicker.placementHintHistory'));
  }

  function handleClearHistory() {
    clearAnchorHistory();
    anchorHistory = [];
  }

  async function reverseGeocode(lat: number, lng: number): Promise<string> {
    try {
      await ensureGeocoder();
      const currentGeocoder = geocoder;
      if (!currentGeocoder) {
        locationName = t('locationPicker.customLocation');
        return locationName;
      }
      const [gcjLng, gcjLat] = toGcjLngLat({
        latitude: lat,
        longitude: lng
      });
      return await new Promise<string>((resolve) => {
        currentGeocoder.getAddress(
          [gcjLng, gcjLat],
          (status: string, result: AMap.GeocodeResult) => {
            if (status === 'complete' && result.regeocode) {
              const poiName = result.regeocode.pois?.[0]?.name;
              const resolved = poiName || result.regeocode.formattedAddress || t('locationPicker.customLocation');
              locationName = resolved;
              resolve(resolved);
              return;
            }
            locationName = t('locationPicker.customLocation');
            resolve(locationName);
          }
        );
      });
    } catch {
      locationName = t('locationPicker.customLocation');
      return locationName;
    }
  }
  function cancelReverseGeocodeTimer() {
    if (reverseGeocodeTimer) {
      clearTimeout(reverseGeocodeTimer);
      reverseGeocodeTimer = null;
    }
  }

  // 选中位置的解析在显式选点（双击/长按/搜索/历史/回到锚点等）时直接调用
  // reverseGeocode()，拖动/缩放地图不再自动触发位置解析。

  function handleMoveEnd() {
    if (!map) return;
    // 拖动结束仅刷新地图渲染，不修改选中位置，也不触发逆地理
    closeClusterPreview();
    scheduleMapRender();
  }

  function handleZoomEnd() {
    if (!map) return;
    const currentZoom = map.getZoom() ?? lastMapZoom;
    lastMapZoom = currentZoom;
    closeClusterPreview();
    scheduleMapRender();
    // 纯缩放不改选中位置，也不触发逆地理
  }

  async function handleDoubleClick(event: MapClickEvent) {
    if (!map) return;
    const lnglat = event.lnglat ?? map.getCenter();
    const nextLat = lnglat.getLat();
    const nextLng = lnglat.getLng();
    await moveToCoordinate(nextLat, nextLng, {
      hintMessage: isAnchorMode ? t('locationPicker.placementHintLocatedConfirm') : t('locationPicker.placementHintDefault')
    });
  }

  async function handleRightClick(event: MapClickEvent) {
    if (!map) return;
    const lnglat = event.lnglat ?? map.getCenter();
    const nextLat = lnglat.getLat();
    const nextLng = lnglat.getLng();
    await moveToCoordinate(nextLat, nextLng, {
      hintMessage: isAnchorMode ? t('locationPicker.placementHintLocatedConfirm') : t('locationPicker.placementHintDefault')
    });
  }

  async function initMap() {
    isLoadingMap = true;
    try {
      await loadAMap();
      await tick();
      if (!mapContainer) return;

      const [gcjLng, gcjLat] = toGcjLngLat({
        latitude,
        longitude
      });

      map = new AMap.Map(mapContainer, {
        center: [gcjLng, gcjLat],
        zoom: 14,
        viewMode: '2D',
        doubleClickZoom: false,
        resizeEnable: true
      });
      try {
        map.setMapStyle?.(currentMode === 'dark' ? 'amap://styles/dark' : 'amap://styles/normal');
      } catch {
        // ignore style errors
      }
      ensureSelectedMarker();
      registerTouchListeners();
      registerContextMenuBlocker();

      map.on('movestart', () => {
        cancelReverseGeocodeTimer();
      });
      map.on('moveend', () => {
        handleMoveEnd();
      });
      map.on('zoomstart', () => {
        cancelReverseGeocodeTimer();
      });
      map.on('zoomend', () => {
        handleZoomEnd();
      });
      map.on('click', handleMapSurfaceClick);
      map.on('dblclick', handleDoubleClick);
      map.on('rightclick', handleRightClick);
      await Promise.allSettled([reverseGeocode(latitude, longitude), fetchMapRenderData()]);
      hasInit = true;
    } catch {
      // 地图实例可能已创建但后续操作（如逆地理编码）失败，需释放资源
      clearSelectedMarker();
      if (map) {
        map.destroy();
        map = null;
      }
      isMapAvailable = false;
    } finally {
      isLoadingMap = false;
    }
  }

  async function handleUseCurrentLocation() {
    isLocating = true;
    closeClusterPreview();
    const geoPoint = await locationStore.requestLocation();
    if (geoPoint) {
      latitude = geoPoint.latitude;
      longitude = geoPoint.longitude;
      const [gcjLng, gcjLat] = toGcjLngLat({
        latitude,
        longitude
      });
      map?.setZoomAndCenter(15, [gcjLng, gcjLat]);
      updateSelectedMarker();
      const resolvedName = await reverseGeocode(latitude, longitude);
      locationStore.setLocation({
        point: {
          latitude,
          longitude
        },
        location_name: resolvedName
      });
    } else {
      toast.error(t('locationPicker.locationUnavailable'));
    }
    isLocating = false;
  }

  async function handleBackToBrowseAnchor() {
    const anchorLat = mapAnchorStore.latitude;
    const anchorLng = mapAnchorStore.longitude;
    closeClusterPreview();
    cancelReverseGeocodeTimer();
    latitude = anchorLat;
    longitude = anchorLng;
    const [gcjLng, gcjLat] = toGcjLngLat({
      latitude: anchorLat,
      longitude: anchorLng
    });
    map?.setZoomAndCenter(14, [gcjLng, gcjLat]);
    updateSelectedMarker();
    if (mapAnchorStore.name) {
      locationName = mapAnchorStore.name;
    } else {
      await reverseGeocode(anchorLat, anchorLng);
    }
    hideGestureHint();
    showPlacementHint(t('locationPicker.placementHintAnchor'));
  }

  function handleConfirm() {
    const normalizedName = locationName.trim() || t('locationPicker.customLocation');
    if (isAnchorMode) {
      mapAnchorStore.setAnchor(latitude, longitude, normalizedName, searchRadius);
      addToAnchorHistory(latitude, longitude, normalizedName);
    } else {
      onConfirm?.({
        latitude,
        longitude,
        locationName: normalizedName,
        radius: searchRadius
      });
    }
    toast.success(confirmSuccessText);
    handleClose();
  }

  function handleRetryMapLoad() {
    clearRenderMarkers();
    clearSelectedMarker();
    closeClusterPreview();
    overlapGroupItems = new Map();
    latestRenderItems = [];
    lastRenderedViewportSignature = '';
    lastHotspotsSignature = '';
    map?.destroy();
    map = null;
    isMapAvailable = true;
    hasInit = false;
    void initMap();
  }

  function setOpenState(nextOpen: boolean) {
    if (open === nextOpen) {
      return;
    }
    open = nextOpen;
    onOpenChange?.(nextOpen);
  }

  function handleDialogOpenChange(nextOpen: boolean) {
    open = nextOpen;
    onOpenChange?.(nextOpen);
  }

  function resetTransientUiState() {
    if (searchDebounceTimer) {
      clearTimeout(searchDebounceTimer);
      searchDebounceTimer = null;
    }
    activeSearchRequestId += 1;
    isSearching = false;
    showSearch = false;
    showFilterPanel = false;
    showShareModal = false;
    searchText = '';
    searchResults = [];
  }

  function resetPreviewUiState() {
    clusterPreview = null;
    hotspotPreviewItems = [];
    isBottomSheetExpanded = false;
  }

  function clearModalSelection() {
    if (!browser || !modalRoot) {
      return;
    }
    const selection = window.getSelection();
    if (!selection || selection.rangeCount === 0) {
      return;
    }
    const range = selection.getRangeAt(0);
    const commonAncestor = range.commonAncestorContainer;
    let shouldClear = modalRoot.contains(commonAncestor);

    if (!shouldClear) {
      const rects = Array.from(range.getClientRects());
      if (rects.length === 0) {
        return;
      }
      const modalRect = modalRoot.getBoundingClientRect();
      shouldClear = rects.some((rect) => {
        const separated =
          rect.right < modalRect.left ||
          rect.left > modalRect.right ||
          rect.bottom < modalRect.top ||
          rect.top > modalRect.bottom;
        return !separated;
      });
    }

    if (shouldClear) {
      selection.removeAllRanges();
    }
  }

  function destroyMapSession() {
    clearRenderMarkers();
    overlapGroupItems = new Map();
    latestRenderItems = [];
    clearSelectedMarker();
    map?.destroy();
    map = null;
    isMapAvailable = true;
    hasInit = false;
  }

  function syncStateFromCurrentSource() {
    if (hasSyncedFromAnchor) {
      return;
    }
    const fallbackName = mapAnchorStore.name || mapAnchorStore.displayName;
    latitude = initialLocation?.latitude ?? mapAnchorStore.latitude;
    longitude = initialLocation?.longitude ?? mapAnchorStore.longitude;
    locationName = initialLocation?.locationName?.trim() || fallbackName;
    searchRadius = initialLocation?.radius ?? mapAnchorStore.radius;
    anchorHistory = isAnchorMode ? getAnchorHistory() : [];
    resetTransientUiState();
    resetPreviewUiState();

    if (!hasInit) {
      void initMap();
    } else {
      updateSelectedMarker();
      const [gcjLng, gcjLat] = toGcjLngLat({
        latitude,
        longitude
      });
      map?.setZoomAndCenter(14, [gcjLng, gcjLat]);
    }
    hasSyncedFromAnchor = true;
  }

  function cleanupClosedSession() {
    hasSyncedFromAnchor = false;
    lastRenderedViewportSignature = '';
    lastHotspotsSignature = '';
    resetTransientUiState();
    resetPreviewUiState();
    hideGestureHint();
    placementHintVisible = false;
    cancelReverseGeocodeTimer();
    if (renderMapTimer) {
      clearTimeout(renderMapTimer);
      renderMapTimer = null;
    }
    if (placementHintTimer) {
      clearTimeout(placementHintTimer);
      placementHintTimer = null;
    }
    if (longPressTimer) {
      clearTimeout(longPressTimer);
      longPressTimer = null;
    }
    if (mapMoveFrame !== null) {
      cancelAnimationFrame(mapMoveFrame);
      mapMoveFrame = null;
    }
    lastTouchLngLat = null;
    detachTouchListeners?.();
    detachContextMenuListener?.();
    detachOutsidePointerListener?.();
    clearModalSelection();
    destroyMapSession();
  }

  function handleClose() {
    setOpenState(false);
  }

  $effect(() => {
    if (open) {
      untrack(() => {
        registerOutsidePointerListener();
        if (!gestureHintVisible) {
          showGestureHint();
        }
        syncStateFromCurrentSource();
      });
      return;
    }

    untrack(() => {
      cleanupClosedSession();
    });
  });

  // 组件卸载时清理资源
  onMount(() => {
    return () => {
      clearRenderMarkers();
      clearSelectedMarker();
      map?.destroy();
      map = null;
      detachTouchListeners?.();
      detachContextMenuListener?.();
      detachOutsidePointerListener?.();
      if (reverseGeocodeTimer) {
        clearTimeout(reverseGeocodeTimer);
        reverseGeocodeTimer = null;
      }
      if (mapMoveFrame !== null) {
        cancelAnimationFrame(mapMoveFrame);
        mapMoveFrame = null;
      }
      if (searchDebounceTimer) {
        clearTimeout(searchDebounceTimer);
        searchDebounceTimer = null;
      }
      if (renderMapTimer) {
        clearTimeout(renderMapTimer);
        renderMapTimer = null;
      }
      if (placementHintTimer) {
        clearTimeout(placementHintTimer);
        placementHintTimer = null;
      }
      if (gestureHintTimer) {
        clearTimeout(gestureHintTimer);
        gestureHintTimer = null;
      }
      if (longPressTimer) {
        clearTimeout(longPressTimer);
        longPressTimer = null;
      }
    };
  });

  onMount(() => {
    if (browser) {
      try {
        gestureHintDismissed = window.localStorage.getItem(GESTURE_HINT_DISMISS_KEY) === '1';
      } catch {
        gestureHintDismissed = false;
      }
    }
    registerOutsidePointerListener();
  });
</script>

<div bind:this={modalRoot}>
  <Dialog.Root bind:open onOpenChange={handleDialogOpenChange}>
    <Dialog.Content
      class={`location-dialog flex flex-col p-0 transition-all duration-200 ease-out select-none sm:flex-row ${isFullscreen ? 'is-fullscreen' : ''}`}
      style="background: var(--surface-bg);"
      hideClose
    >
      <!-- 移动端轻量工具栏 -->
      <div
        class="border-b px-3 pb-3 pt-[max(0.75rem,env(safe-area-inset-top))] backdrop-blur-sm sm:hidden"
        style="background: var(--surface-bg); border-color: var(--border-theme);"
      >
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <p class="text-[11px] font-medium uppercase tracking-[0.16em]">
              {mapModeLabel}
            </p>
            <p class="mt-1 truncate text-base font-semibold">
              {modalTitle}
            </p>
            <p class="mt-1 text-xs">{t('locationPicker.dragHint')}</p>
          </div>
          <Button
            variant="ghost"
            size="sm"
            class="h-9 w-9 shrink-0 rounded-full p-0"
            onclick={handleClose}
          >
            <span class="sr-only">{t('locationPicker.close')}</span>
            <X class="h-4 w-4" />
          </Button>
        </div>
        <div class="mt-3 flex items-center gap-2">
          <Button
            variant={showSearch ? 'default' : 'outline'}
            size="sm"
            class="h-9 flex-1 rounded-full"
            onclick={toggleSearch}
          >
            <Search class="h-4 w-4" />
            {t('locationPicker.searchPlace')}
          </Button>
          <Button
            data-location-filter-trigger="true"
            variant={showFilterPanel ? 'default' : 'outline'}
            size="sm"
            class="h-9 rounded-full px-3"
            onclick={toggleFilterPanel}
          >
            <ListFilter class="h-4 w-4" />
            {t('locationPicker.filterWithCount', { count: activeFilterCount })}
          </Button>
        </div>
      </div>
      <!-- 左侧：地图容器 -->
      <div class="relative flex-1 overflow-hidden sm:order-2">
        <div class="h-full w-full touch-pan-x touch-pan-y" bind:this={mapContainer}></div>

        <!-- 桌面端地图工具栏：右上角仅保留“筛选”入口，与地图区域靠近 -->
        <div class="pointer-events-none absolute right-4 top-4 z-30 hidden flex-col gap-2 sm:flex">
          <div class="pointer-events-auto flex flex-col gap-2">
            <Button
              data-location-filter-trigger="true"
              size="icon"
              variant="ghost"
              class="h-11 w-11 rounded-[var(--radius-sm)] backdrop-blur"
              style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
              onclick={toggleFilterPanel}
              aria-label={t('locationPicker.filterMapContentAria')}
            >
              <ListFilter class="h-4 w-4" />
            </Button>
          </div>
        </div>

        <div
          class="pointer-events-none absolute right-4 bottom-6 z-30 hidden flex-col gap-2 sm:flex"
        >
          <div class="pointer-events-auto flex flex-col gap-2">
            <Button
              size="icon"
              variant="ghost"
              class="h-11 w-11 rounded-[var(--radius-sm)] transition"
              style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
              onclick={handleUseCurrentLocation}
              disabled={isLocating}
              aria-label={t('locationPicker.locateCurrent')}
            >
              {#if isLocating}
                <Spinner size="md" />
              {:else}
                <LocateFixed class="h-6 w-6" style="color: var(--brand);" />
              {/if}
            </Button>
            {#if isAnchorMode}
              <Button
                size="icon"
                variant="ghost"
                class="h-11 w-11 rounded-[var(--radius-sm)] transition"
                style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
                onclick={handleBackToBrowseAnchor}
                aria-label={t('locationPicker.backToAnchorAria')}
              >
                <img src="/map/marker_center_pin.svg" alt={t('locationPicker.anchorImageAlt')} class="h-7 w-7" />
              </Button>
            {/if}
            <Button
              size="icon"
              variant="ghost"
              class="h-11 w-11 rounded-[var(--radius-sm)] transition"
              style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
              onclick={() => {
                if (!map) return;
                const currentZoom = map.getZoom?.() ?? 14;
                map.setZoom(Math.min(20, currentZoom + 1));
              }}
              aria-label={t('locationPicker.zoomInAria')}
            >
              <Plus class="h-5 w-5" />
            </Button>
            <Button
              size="icon"
              variant="ghost"
              class="h-11 w-11 rounded-[var(--radius-sm)] transition"
              style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
              onclick={() => {
                if (!map) return;
                const currentZoom = map.getZoom?.() ?? 14;
                map.setZoom(Math.max(1, currentZoom - 1));
              }}
              aria-label={t('locationPicker.zoomOutAria')}
            >
              <Minus class="h-5 w-5" />
            </Button>
          </div>
        </div>

        <!-- 移除地图左上角的“地图内容更新中/正在获取当前位置”浮层，避免缩放时闪烁 -->

        {#if showFilterPanel}
          <div
            bind:this={filterPanelElement}
            class="absolute right-4 top-16 z-30 hidden w-40 select-none overflow-hidden rounded-[var(--radius-lg)] p-2 backdrop-blur-md sm:block"
            style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
          >
            <button
              class={`flex w-full items-center justify-between rounded-[var(--radius-md)] px-3 py-2.5 text-left text-sm font-medium transition`}
              style={showUsers
                ? 'background: var(--surface-bg); border: 1px solid var(--brand); color: var(--text-primary);'
                : 'background: var(--surface-bg); border: 1px solid var(--border-theme); color: var(--text-secondary);'}
              onclick={() => toggleRenderFilter('user')}
            >
              <span class="flex items-center gap-2">
                <img src="/map/marker_user.svg" alt={t('locationPicker.filterUserAlt')} class="h-8 w-auto" />
                <span>{t('locationPicker.filterUser')}</span>
              </span>
              <span
                class="h-2.5 w-2.5 rounded-full"
                style={showUsers ? 'background: var(--brand);' : 'background: var(--border-theme);'}
              ></span>
            </button>
            <button
              class={`mt-1 flex w-full items-center justify-between rounded-[var(--radius-md)] px-3 py-2.5 text-left text-sm font-medium transition`}
              style={showGroups
                ? 'background: var(--surface-bg); border: 1px solid var(--brand); color: var(--text-primary);'
                : 'background: var(--surface-bg); border: 1px solid var(--border-theme); color: var(--text-secondary);'}
              onclick={() => toggleRenderFilter('group')}
            >
              <span class="flex items-center gap-2">
                <img src="/map/marker_group.svg" alt={t('locationPicker.filterGroupAlt')} class="h-8 w-auto" />
                <span>{t('locationPicker.filterGroup')}</span>
              </span>
              <span
                class="h-2.5 w-2.5 rounded-full"
                style={showGroups
                  ? 'background: var(--brand);'
                  : 'background: var(--border-theme);'}
              ></span>
            </button>
            <button
              class={`mt-1 flex w-full items-center justify-between rounded-[var(--radius-md)] px-3 py-2.5 text-left text-sm font-medium transition`}
              style={showMoments
                ? 'background: var(--surface-bg); border: 1px solid var(--brand); color: var(--text-primary);'
                : 'background: var(--surface-bg); border: 1px solid var(--border-theme); color: var(--text-secondary);'}
              onclick={() => toggleRenderFilter('moment')}
            >
              <span class="flex items-center gap-2">
                <img src="/map/marker_moment.svg" alt={t('locationPicker.filterMomentAlt')} class="h-8 w-auto" />
                <span>{t('locationPicker.filterMoment')}</span>
              </span>
              <span
                class="h-2.5 w-2.5 rounded-full"
                style={showMoments
                  ? 'background: var(--brand);'
                  : 'background: var(--border-theme);'}
              ></span>
            </button>
            <div class="px-3 pb-1 pt-2 text-[11px]">
              {t('locationPicker.filtersEnabled', { count: activeFilterCount })}
            </div>
          </div>
        {/if}

        {#if gestureHintVisible || placementHintVisible}
          <div
            class="pointer-events-none absolute inset-x-0 top-16 z-30 flex justify-center px-4 sm:top-4"
          >
            {#if placementHintVisible}
              <div
                class="rounded-[var(--radius-md)] px-4 py-1.5 text-sm font-medium backdrop-blur-sm"
                style="background: var(--surface-bg); border: 1px solid var(--border-theme); color: var(--text-primary);"
              >
                {placementHintMessage}
              </div>
            {:else if gestureHintVisible}
              <div
                class="pointer-events-auto inline-flex items-center gap-3 rounded-[var(--radius-md)] border border-dashed px-4 py-1.5 text-xs backdrop-blur-sm"
                style="border-color: var(--text-secondary); background: var(--surface-bg); color: var(--text-secondary);"
              >
                <span>{gestureHintMessage}</span>
                <button
                  type="button"
                  class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-medium"
                  style="color: var(--text-secondary);"
                  onclick={dismissGestureHintPermanently}
                >
                  <XIcon class="h-3 w-3" />
                  <span>{t('locationPicker.noMoreHints')}</span>
                </button>
              </div>
            {/if}
          </div>
        {/if}

        {#if !isMapAvailable}
          <div class="absolute inset-0 z-20 flex items-center justify-center backdrop-blur-sm">
            <div class="text-center space-y-3">
              <p class="font-medium">{t('locationPicker.mapNotConfigured')}</p>
              <p class="text-sm">{t('locationPicker.mapKeyHint')}</p>
              <Button size="sm" variant="outline" onclick={handleRetryMapLoad}>{t('locationPicker.retryLoad')}</Button>
            </div>
          </div>
        {/if}

        {#if isLoadingMap}
          <div class="absolute inset-0 z-20 flex items-center justify-center backdrop-blur-sm">
            <Spinner size="xl" />
          </div>
        {/if}

        <!-- 悬浮定位按钮 (移动端) — 置于拇指可达的底部，收起态底栏(116px)之上 -->
        <div class="absolute right-3 bottom-[132px] z-10 flex flex-col gap-2 sm:hidden">
          <Button
            size="icon"
            variant="ghost"
            class="h-12 w-12 rounded-full backdrop-blur-sm transition active:scale-95"
            style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
            onclick={handleUseCurrentLocation}
            disabled={isLocating}
            aria-label={t('locationPicker.useCurrentLocationAria')}
          >
            {#if isLocating}
              <Spinner size="lg" />
            {:else}
              <LocateFixed class="h-7 w-7" style="color: var(--brand);" />
            {/if}
          </Button>
          {#if isAnchorMode}
            <Button
              size="icon"
              variant="ghost"
              class="h-12 w-12 rounded-full backdrop-blur-sm transition active:scale-95"
              style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
              onclick={handleBackToBrowseAnchor}
              aria-label={t('locationPicker.backToAnchorAria')}
            >
              <img src="/map/marker_center_pin.svg" alt={t('locationPicker.anchorImageAlt')} class="h-7 w-7" />
            </Button>
          {/if}
        </div>

        {#if clusterPreview}
          <div class="absolute bottom-6 left-4 right-auto z-30 hidden w-100 select-none sm:block">
            <div
              class="overflow-hidden rounded-[var(--radius-button)] backdrop-blur-md"
              style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
            >
              <div
                class="flex items-start justify-between gap-3 border-b px-4 py-3"
                style="border-color: var(--border-theme);"
              >
                <div class="min-w-0">
                  <p
                    class="text-[11px] font-semibold uppercase tracking-[0.16em]"
                    style="color: var(--text-secondary);"
                  >
                    {clusterPreview.kind === 'overlap_group' ? t('locationPicker.overlapPreviewTitle') : t('locationPicker.clusterPreviewTitle')}
                  </p>
                  <p class="truncate text-base font-bold">
                    {clusterPreview.title}
                  </p>
                  <p class="mt-1 text-xs">
                    {clusterPreview.description}
                  </p>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  class="h-9 w-9 shrink-0 rounded-[var(--radius-md)] p-0"
                  style="background: var(--surface-bg);"
                  onclick={closeClusterPreview}
                >
                  <X class="h-4 w-4" />
                  <span class="sr-only">{t('locationPicker.closePreview')}</span>
                </Button>
              </div>

              {#if clusterPreview.items.length > 0}
                <div class="max-h-64 space-y-2 overflow-y-auto px-3 py-3">
                  {#each clusterPreview.items as previewItem (`preview_${previewItem.location.latitude}_${previewItem.location.longitude}_${previewItem.total_count}`)}
                    <button
                      class="flex w-full items-start gap-3 rounded-[var(--radius-md)] px-3 py-3 text-left transition hover:-translate-y-0.5"
                      style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
                      onclick={() => focusPreviewEntity(previewItem)}
                    >
                      <div
                        class={`rounded-full px-2 py-1 text-[11px] font-semibold ${getItemBadgeClass(previewItem)}`}
                      >
                        {getItemTypeLabel(previewItem)}
                      </div>
                      <div class="min-w-0 flex-1">
                        <p class="truncate text-sm font-semibold">{t('locationPicker.clusterRegion')}</p>
                        <p class="mt-1 line-clamp-2 text-xs">
                          {t('locationPicker.contentCount', { count: previewItem.total_count })}
                        </p>
                      </div>
                    </button>
                  {/each}
                </div>
              {:else}
                <div class="px-4 py-4 text-sm">
                  {t('locationPicker.clusterSummaryOnly')}
                </div>
              {/if}

              <div class="flex gap-2 border-t px-4 py-3" style="border-color: var(--border-theme);">
                <Button
                  variant="outline"
                  class="h-10 flex-1 rounded-[var(--radius-md)]"
                  onclick={closeClusterPreview}
                >
                  {t('locationPicker.reviewLater')}
                </Button>
                <Button
                  class="h-10 flex-1 rounded-[var(--radius-md)]"
                  onclick={zoomToClusterPreview}
                >
                  {clusterPreview.kind === 'overlap_group' ? t('locationPicker.zoomToExpandOverlap') : t('locationPicker.zoomToExpandCluster')}
                </Button>
              </div>
            </div>
          </div>
        {/if}

        {#if hotspotPreviewItems.length > 0}
          <div
            class="absolute bottom-6 left-1/2 right-auto z-30 hidden w-[420px] -translate-x-1/2 select-none sm:block"
          >
            <div
              class="overflow-hidden rounded-[var(--radius-button)] backdrop-blur-md"
              style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
            >
              <div
                class="flex items-start justify-between gap-3 border-b px-4 py-3"
                style="border-color: var(--border-theme);"
              >
                <div class="min-w-0">
                  <p
                    class="text-[11px] font-semibold uppercase tracking-[0.16em]"
                    style="color: var(--text-secondary);"
                  >
                    {t('locationPicker.mapObjectDetails')}
                  </p>
                  <p class="truncate text-base font-bold">
                    {hotspotPreviewItems.length === 1
                      ? (hotspotPreviewItems[0]?.title ?? '')
                      : t('locationPicker.contentCount', { count: hotspotPreviewItems.length })}
                  </p>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  class="h-9 w-9 shrink-0 rounded-[var(--radius-md)] p-0"
                  style="background: var(--surface-bg);"
                  onclick={() => {
                    hotspotPreviewItems = [];
                    isBottomSheetExpanded = false;
                  }}
                >
                  <X class="h-4 w-4" />
                  <span class="sr-only">{t('locationPicker.closeDetails')}</span>
                </Button>
              </div>

              <div class="max-h-72 space-y-2 overflow-y-auto px-3 py-3">
                {#each hotspotPreviewItems as item (item.object_type + '_' + item.object_id)}
                  <button
                    class="flex w-full items-start gap-3 rounded-[var(--radius-md)] px-3 py-3 text-left transition hover:-translate-y-0.5"
                    style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
                    onclick={() => {
                      hotspotPreviewItems = [];
                      if (item.object_type === 'user') {
                        goto(`/users/${item.object_id}`);
                      } else if (item.object_type === 'group') {
                        goto(`/groups/${item.object_id}`);
                      } else if (item.object_type === 'moment') {
                        goto(`/moments/${item.object_id}`);
                      }
                    }}
                  >
                    <div
                      class="rounded-full px-2 py-1 text-[11px] font-semibold"
                      style="background: var(--surface-bg); color: var(--brand); border: 1px solid var(--border-theme);"
                    >
                      {item.object_type === 'user'
                        ? t('locationPicker.itemLabelUser')
                        : item.object_type === 'group'
                          ? t('locationPicker.itemLabelGroup')
                          : t('locationPicker.itemLabelMoment')}
                    </div>
                    <div class="min-w-0 flex-1">
                      <p class="truncate text-sm font-semibold">
                        {item.title ?? ''}
                      </p>
                      {#if item.subtitle}
                        <p class="mt-1 line-clamp-2 text-xs">
                          {item.subtitle}
                        </p>
                      {/if}
                    </div>
                  </button>
                {/each}
              </div>
            </div>
          </div>
        {/if}

        <!-- 移动端底部抽屉 -->
        <div
          class="absolute bottom-0 left-0 right-0 z-20 transition-transform duration-250 ease-out sm:hidden"
          class:translate-y-0={!isBottomSheetExpanded}
          style="transform: translateY({isBottomSheetExpanded ? '0' : 'calc(100% - 116px)'})"
        >
          <div
            class="rounded-t-[28px] backdrop-blur-md"
            style="background: var(--surface-bg); border-top: 1px solid var(--border-theme); box-shadow: 0 -4px 10px var(--shadow-dark);"
          >
            <!-- 展开/收起手柄 -->
            <button
              class="flex w-full flex-col items-center gap-1 px-4 py-3"
              onclick={() => (isBottomSheetExpanded = !isBottomSheetExpanded)}
              aria-expanded={isBottomSheetExpanded}
              aria-label={isBottomSheetExpanded ? t('locationPicker.collapsePanelAria') : t('locationPicker.expandPanelAria')}
            >
              <div class="h-1 w-12 rounded-full" style="background: var(--border-theme);"></div>
              <div
                class="flex items-center gap-2 text-[11px] font-semibold tracking-[0.08em] /90 uppercase"
              >
                <span>{isBottomSheetExpanded ? t('locationPicker.collapsePanel') : t('locationPicker.expandPanel')}</span>
                {#if isBottomSheetExpanded}
                  <ChevronDown class="h-4 w-4" />
                {:else}
                  <ChevronUp class="h-4 w-4" />
                {/if}
              </div>
            </button>

            <!-- 底部抽屉内容 -->
            <div class="max-h-[70dvh] overflow-y-auto px-4 pb-safe">
              <div class="space-y-3 pb-4">
                <AnchorSummaryCard
                  label={t('locationPicker.currentSelection')}
                  description={selectionDescription}
                  subtle
                  overrideName={locationName || t('locationPicker.dragToSelect')}
                />

                {#if showSearch}
                  <div
                    class="overflow-hidden rounded-[var(--radius-button)] backdrop-blur-md"
                    style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
                  >
                    <div class="border-b px-4 py-3" style="border-color: var(--border-theme);">
                      <div class="flex items-start justify-between gap-3">
                        <div>
                          <p
                            class="text-[11px] font-semibold uppercase tracking-[0.16em]"
                            style="color: var(--text-secondary);"
                          >
                            {t('locationPicker.placeSearchTitle')}
                          </p>
                          <p class="mt-2 text-sm font-semibold">
                            {t('locationPicker.placeSearchDesc')}
                          </p>
                        </div>
                        <Button
                          variant="ghost"
                          size="sm"
                          class="h-9 w-9 shrink-0 rounded-[var(--radius-md)] p-0"
                          onclick={toggleSearch}
                        >
                          <X class="h-4 w-4" />
                          <span class="sr-only">{t('locationPicker.closeSearch')}</span>
                        </Button>
                      </div>
                    </div>
                    <div class="space-y-3 px-4 py-3">
                      <div
                        class="input-surface flex h-12 items-center gap-3 rounded-[var(--radius-md)] px-3"
                        style="background: var(--surface-bg);"
                      >
                        <Search
                          class="h-4 w-4 shrink-0"
                          color="var(--text-secondary)"
                          strokeWidth={1.5}
                        />
                        <input
                          {@attach useKeyboardFocus()}
                          type="text"
                          bind:this={searchInput}
                          bind:value={searchText}
                          placeholder={t('locationPicker.searchPlaceholderMobile')}
                          class="flex-1 select-text bg-transparent text-sm outline-none"
                          oninput={() => {
                            if (searchDebounceTimer) clearTimeout(searchDebounceTimer);
                            if (searchText.trim().length > 0) {
                              searchDebounceTimer = window.setTimeout(() => {
                                doSearch();
                              }, 500);
                            } else {
                              activeSearchRequestId += 1;
                              isSearching = false;
                              searchResults = [];
                            }
                          }}
                          onkeydown={(e) => {
                            if (e.key === 'Enter') {
                              if (searchDebounceTimer) clearTimeout(searchDebounceTimer);
                              doSearch();
                            } else if (e.key === 'Escape') {
                              toggleSearch();
                            }
                          }}
                        />
                        {#if isSearching}
                          <Spinner size="md" class="shrink-0" />
                        {:else if searchText.length > 0}
                          <button
                            class="shrink-0"
                            onclick={() => {
                              if (searchDebounceTimer) clearTimeout(searchDebounceTimer);
                              searchDebounceTimer = null;
                              activeSearchRequestId += 1;
                              isSearching = false;
                              searchText = '';
                              searchResults = [];
                              searchInput?.focus();
                            }}
                          >
                            <X class="h-4 w-4" />
                          </button>
                        {/if}
                      </div>

                      {#if searchResults.length > 0}
                        <div class="max-h-[38dvh] space-y-2 overflow-y-auto">
                          {#each searchResults as result (`sheet_${result.latitude}_${result.longitude}`)}
                            <button
                              class="flex w-full items-start gap-3 rounded-[var(--radius-md)] px-4 py-3 text-left transition-all hover:-translate-y-0.5"
                              onclick={() => selectSearchResult(result)}
                            >
                              <div
                                class="flex h-9 w-9 shrink-0 items-center justify-center rounded-[var(--radius-md)]"
                              >
                                <MapPin class="h-4 w-4" />
                              </div>
                              <div class="min-w-0 flex-1">
                                <div class="truncate text-sm font-medium">
                                  {result.name}
                                </div>
                                <div class="mt-1 text-xs">
                                  {result.address || t('locationPicker.noAddress')}
                                </div>
                              </div>
                            </button>
                          {/each}
                        </div>
                      {:else if searchText.trim().length === 0 && anchorHistory.length > 0}
                        <div class="space-y-2">
                          <div class="flex items-center justify-between px-1">
                            <div class="flex items-center gap-1.5 text-xs font-medium">
                              <Clock class="h-3.5 w-3.5" />
                              <span>{t('locationPicker.recentlyUsed')}</span>
                            </div>
                            <button
                              class="flex items-center gap-1 text-xs"
                              onclick={handleClearHistory}
                            >
                              <Trash2 class="h-3 w-3" />
                              <span>{t('locationPicker.clearAll')}</span>
                            </button>
                          </div>
                          <div class="max-h-[32dvh] space-y-2 overflow-y-auto">
                            {#each anchorHistory as item (item.timestamp)}
                              <button
                                class="flex w-full items-center gap-3 rounded-[var(--radius-md)] px-4 py-3 text-left transition-all hover:-translate-y-0.5"
                                onclick={() => {
                                  selectHistoryItem(item);
                                  showSearch = false;
                                }}
                              >
                                <div
                                  class="flex h-9 w-9 shrink-0 items-center justify-center rounded-[var(--radius-md)]"
                                >
                                  <Clock class="h-4 w-4" />
                                </div>
                                <div class="min-w-0 flex-1">
                                  <div class="truncate text-sm font-medium">
                                    {item.locationName}
                                  </div>
                                  <div class="mt-1 text-[11px]">{t('locationPicker.historyItemHint')}</div>
                                </div>
                              </button>
                            {/each}
                          </div>
                        </div>
                      {:else if searchText.trim().length > 0 && !isSearching}
                        <div
                          class="rounded-[var(--radius-md)] border border-dashed px-4 py-6 text-center"
                        >
                          <p class="text-sm font-semibold">{t('locationPicker.noMatchTitle')}</p>
                          <p class="mt-1 text-xs">{t('locationPicker.noMatchDesc')}</p>
                        </div>
                      {/if}
                    </div>
                  </div>
                {/if}

                {#if showFilterPanel}
                  <div
                    class="overflow-hidden rounded-[var(--radius-button)] p-3 backdrop-blur-md"
                    style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
                  >
                    <div class="mb-2 flex items-center justify-between px-1">
                      <div>
                        <p
                          class="text-[11px] font-semibold uppercase tracking-[0.16em]"
                          style="color: var(--text-secondary);"
                        >
                          {t('locationPicker.mapFilterTitle')}
                        </p>
                        <p class="mt-1 text-xs">
                          {t('locationPicker.filtersEnabled', { count: activeFilterCount })}
                        </p>
                      </div>
                      <Button
                        variant="ghost"
                        size="sm"
                        class="h-9 w-9 rounded-[var(--radius-md)] p-0"
                        onclick={toggleFilterPanel}
                      >
                        <X class="h-4 w-4" />
                        <span class="sr-only">{t('locationPicker.closeFilter')}</span>
                      </Button>
                    </div>
                    <div class="space-y-2">
                      <button
                        class={`flex w-full items-center justify-between rounded-[var(--radius-md)] px-3 py-3 text-left text-sm font-medium transition`}
                        style={showUsers
                          ? 'background: var(--surface-bg); border: 1px solid var(--brand); color: var(--text-primary);'
                          : 'background: var(--surface-bg); border: 1px solid var(--border-theme); color: var(--text-secondary);'}
                        onclick={() => toggleRenderFilter('user')}
                      >
                        <span class="flex items-center gap-2">
                          <img src="/map/marker_user.svg" alt={t('locationPicker.filterUserAlt')} class="h-8 w-auto" />
                          <span>{t('locationPicker.filterUser')}</span>
                        </span>
                        <span
                          class="h-2.5 w-2.5 rounded-full"
                          style={showUsers
                            ? 'background: var(--brand);'
                            : 'background: var(--border-theme);'}
                        ></span>
                      </button>
                      <button
                        class={`flex w-full items-center justify-between rounded-[var(--radius-md)] px-3 py-3 text-left text-sm font-medium transition`}
                        style={showGroups
                          ? 'background: var(--surface-bg); border: 1px solid var(--brand); color: var(--text-primary);'
                          : 'background: var(--surface-bg); border: 1px solid var(--border-theme); color: var(--text-secondary);'}
                        onclick={() => toggleRenderFilter('group')}
                      >
                        <span class="flex items-center gap-2">
                          <img src="/map/marker_group.svg" alt={t('locationPicker.filterGroupAlt')} class="h-8 w-auto" />
                          <span>{t('locationPicker.filterGroup')}</span>
                        </span>
                        <span
                          class="h-2.5 w-2.5 rounded-full"
                          style={showGroups
                            ? 'background: var(--brand);'
                            : 'background: var(--border-theme);'}
                        ></span>
                      </button>
                      <button
                        class={`flex w-full items-center justify-between rounded-[var(--radius-md)] px-3 py-3 text-left text-sm font-medium transition`}
                        style={showMoments
                          ? 'background: var(--surface-bg); border: 1px solid var(--brand); color: var(--text-primary);'
                          : 'background: var(--surface-bg); border: 1px solid var(--border-theme); color: var(--text-secondary);'}
                        onclick={() => toggleRenderFilter('moment')}
                      >
                        <span class="flex items-center gap-2">
                          <img src="/map/marker_moment.svg" alt={t('locationPicker.filterMomentAlt')} class="h-8 w-auto" />
                          <span>{t('locationPicker.filterMoment')}</span>
                        </span>
                        <span
                          class="h-2.5 w-2.5 rounded-full"
                          style={showMoments
                            ? 'background: var(--brand);'
                            : 'background: var(--border-theme);'}
                        ></span>
                      </button>
                    </div>
                  </div>
                {/if}

                {#if clusterPreview}
                  <div
                    class="overflow-hidden rounded-[var(--radius-button)] backdrop-blur-md"
                    style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
                  >
                    <div
                      class="flex items-start justify-between gap-3 border-b px-4 py-3"
                      style="border-color: var(--border-theme);"
                    >
                      <div class="min-w-0">
                        <p
                          class="text-[11px] font-semibold uppercase tracking-[0.16em]"
                          style="color: var(--text-secondary);"
                        >
                          {clusterPreview.kind === 'overlap_group'
                            ? t('locationPicker.overlapPreviewTitle')
                            : t('locationPicker.clusterPreviewTitle')}
                        </p>
                        <p class="truncate text-base font-bold">
                          {clusterPreview.title}
                        </p>
                        <p class="mt-1 text-xs">
                          {clusterPreview.description}
                        </p>
                      </div>
                      <Button
                        variant="ghost"
                        size="sm"
                        class="h-9 w-9 shrink-0 rounded-[var(--radius-md)] p-0"
                        onclick={closeClusterPreview}
                      >
                        <X class="h-4 w-4" />
                        <span class="sr-only">{t('locationPicker.closePreview')}</span>
                      </Button>
                    </div>
                    {#if clusterPreview.items.length > 0}
                      <div class="max-h-[32dvh] space-y-2 overflow-y-auto px-3 py-3">
                        {#each clusterPreview.items as previewItem (`sheet_preview_${previewItem.location.latitude}_${previewItem.location.longitude}_${previewItem.total_count}`)}
                          <button
                            class="flex w-full items-start gap-3 rounded-[var(--radius-md)] px-3 py-3 text-left transition hover:-translate-y-0.5"
                            onclick={() => focusPreviewEntity(previewItem)}
                          >
                            <div
                              class={`rounded-full px-2 py-1 text-[11px] font-semibold ${getItemBadgeClass(previewItem)}`}
                            >
                              {getItemTypeLabel(previewItem)}
                            </div>
                            <div class="min-w-0 flex-1">
                              <p class="truncate text-sm font-semibold">{t('locationPicker.clusterRegion')}</p>
                              <p class="mt-1 line-clamp-2 text-xs">
                                {t('locationPicker.contentCount', { count: previewItem.total_count })}
                              </p>
                            </div>
                          </button>
                        {/each}
                      </div>
                    {:else}
                      <div class="px-4 py-4 text-sm">
                        {t('locationPicker.clusterSummaryOnly')}
                      </div>
                    {/if}
                    <div
                      class="flex gap-2 border-t px-4 py-3"
                      style="border-color: var(--border-theme);"
                    >
                      <Button
                        variant="outline"
                        class="h-10 flex-1 rounded-[var(--radius-md)]"
                        onclick={closeClusterPreview}
                      >
                        {t('locationPicker.reviewLater')}
                      </Button>
                      <Button
                        class="h-10 flex-1 rounded-[var(--radius-md)]"
                        onclick={zoomToClusterPreview}
                      >
                        {clusterPreview.kind === 'overlap_group' ? t('locationPicker.zoomToExpandOverlap') : t('locationPicker.zoomToExpandCluster')}
                      </Button>
                    </div>
                  </div>
                {/if}

                {#if hotspotPreviewItems.length > 0}
                  <div
                    class="overflow-hidden rounded-[var(--radius-button)] backdrop-blur-md"
                    style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
                  >
                    <div
                      class="flex items-start justify-between gap-3 border-b px-4 py-3"
                      style="border-color: var(--border-theme);"
                    >
                      <div class="min-w-0">
                        <p
                          class="text-[11px] font-semibold uppercase tracking-[0.16em]"
                          style="color: var(--text-secondary);"
                        >
                          {t('locationPicker.mapObjectDetails')}
                        </p>
                        <p class="truncate text-base font-bold">
                          {hotspotPreviewItems.length === 1
                            ? (hotspotPreviewItems[0]?.title ?? '')
                            : t('locationPicker.contentCount', { count: hotspotPreviewItems.length })}
                        </p>
                      </div>
                      <Button
                        variant="ghost"
                        size="sm"
                        class="h-9 w-9 shrink-0 rounded-[var(--radius-md)] p-0"
                        onclick={() => {
                          hotspotPreviewItems = [];
                          isBottomSheetExpanded = false;
                        }}
                      >
                        <X class="h-4 w-4" />
                        <span class="sr-only">{t('locationPicker.closeDetails')}</span>
                      </Button>
                    </div>
                    <div class="max-h-[32dvh] space-y-2 overflow-y-auto px-3 py-3">
                      {#each hotspotPreviewItems as item (item.object_type + '_' + item.object_id)}
                        <button
                          class="flex w-full items-start gap-3 rounded-[var(--radius-md)] px-3 py-3 text-left transition hover:-translate-y-0.5"
                          onclick={() => {
                            hotspotPreviewItems = [];
                            if (item.object_type === 'user') {
                              goto(`/users/${item.object_id}`);
                            } else if (item.object_type === 'group') {
                              goto(`/groups/${item.object_id}`);
                            } else if (item.object_type === 'moment') {
                              goto(`/moments/${item.object_id}`);
                            }
                          }}
                        >
                          <div
                            class="rounded-full px-2 py-1 text-[11px] font-semibold"
                            style="background: var(--surface-bg); color: var(--brand); border: 1px solid var(--border-theme);"
                          >
                            {item.object_type === 'user'
                              ? t('locationPicker.itemLabelUser')
                              : item.object_type === 'group'
                                ? t('locationPicker.itemLabelGroup')
                                : t('locationPicker.itemLabelMoment')}
                          </div>
                          <div class="min-w-0 flex-1">
                            <p class="truncate text-sm font-semibold">
                              {item.title ?? ''}
                            </p>
                            {#if item.subtitle}
                              <p class="mt-1 line-clamp-2 text-xs">
                                {item.subtitle}
                              </p>
                            {/if}
                          </div>
                        </button>
                      {/each}
                    </div>
                  </div>
                {/if}
              </div>

              <div
                class="sticky bottom-0 space-y-2 border-t pb-4 pt-3 backdrop-blur-sm"
                style="background: var(--surface-bg); border-color: var(--border-theme);"
              >
                <Button
                  class="h-12 w-full gap-2 text-base font-semibold"
                  onclick={handleConfirm}
                  disabled={!locationName}
                >
                  <MapPin class="h-5 w-5" />
                  {confirmButtonText}
                </Button>
                {#if isAnchorMode}
                  <Button
                    variant="outline"
                    class="h-12 w-full gap-2 text-base font-semibold"
                    onclick={() => (showShareModal = true)}
                    disabled={!locationName}
                  >
                    <Share2 class="h-5 w-5" />
                    {t('locationPicker.shareLocation')}
                  </Button>
                {/if}
              </div>
            </div>
          </div>

          <!-- 原桌面底部操作卡片在引入右侧信息栏后不再展示，避免重复操作区域 -->
        </div>
      </div>

      <!-- 右侧：桌面信息栏（标题 / 搜索 / 历史 / 确认） -->
      <div
        class="hidden h-full w-[340px] shrink-0 select-none flex-col gap-5 overflow-hidden p-5 sm:flex sm:order-1"
        style="background: var(--surface-bg); border-right: 1px solid var(--border-theme);"
      >
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <p class="text-[11px]" style="color: var(--text-secondary);">
              {mapModeLabel} <span style="margin: 0 4px;">›</span>
            </p>
            <p class="text-h2 mt-0.5 truncate text-base font-bold">
              {modalTitle}
            </p>
          </div>
          <div class="flex items-center gap-2">
            <Button
              variant="ghost"
              size="icon"
              class="h-9 w-9 rounded-[var(--radius-md)] p-0"
              onclick={() => (isFullscreen = !isFullscreen)}
              aria-pressed={isFullscreen}
              aria-label={isFullscreen ? t('locationPicker.exitFullscreenAria') : t('locationPicker.enterFullscreenAria')}
            >
              {#if isFullscreen}
                <Minimize2 class="h-4 w-4" />
              {:else}
                <Maximize2 class="h-4 w-4" />
              {/if}
            </Button>
          </div>
        </div>

        <!-- 去掉右侧顶部的“地图内容更新中”等瞬时提示，避免缩放时闪烁扰动 -->

        <!-- 搜索与结果（桌面端常驻在信息栏中） -->
        <div class="space-y-2">
          <div
            class="input-surface flex h-12 items-center gap-3 rounded-[var(--radius-md)] px-3 text-sm"
            style="background: var(--surface-bg);"
          >
            <Search class="h-4 w-4 shrink-0" color="var(--text-secondary)" strokeWidth={1.5} />
            <input
              {@attach useKeyboardFocus()}
              type="text"
              bind:this={searchInput}
              bind:value={searchText}
              placeholder={t('locationPicker.searchPlaceholderDesktop')}
              class="flex-1 select-text bg-transparent text-sm outline-none"
              oninput={() => {
                if (searchDebounceTimer) clearTimeout(searchDebounceTimer);
                if (searchText.trim().length > 0) {
                  searchDebounceTimer = window.setTimeout(() => {
                    doSearch();
                  }, 500);
                } else {
                  activeSearchRequestId += 1;
                  isSearching = false;
                  searchResults = [];
                }
              }}
              onkeydown={(e) => {
                if (e.key === 'Enter') {
                  if (searchDebounceTimer) clearTimeout(searchDebounceTimer);
                  doSearch();
                }
              }}
            />
            {#if isSearching}
              <Spinner size="md" class="shrink-0" />
            {:else if searchText.length > 0}
              <button
                class="shrink-0"
                onclick={() => {
                  if (searchDebounceTimer) clearTimeout(searchDebounceTimer);
                  searchDebounceTimer = null;
                  activeSearchRequestId += 1;
                  isSearching = false;
                  searchText = '';
                  searchResults = [];
                  searchInput?.focus();
                }}
              >
                <X class="h-4 w-4" />
              </button>
            {/if}
          </div>

          <div class="max-h-[60dvh] space-y-2 overflow-y-auto pr-1">
            {#if searchResults.length > 0}
              {#each searchResults as result (`desktop_${result.latitude}_${result.longitude}`)}
                <button
                  class="flex w-full items-start gap-3 rounded-[var(--radius-md)] px-3 py-2 text-left text-xs transition-all hover:-translate-y-0.5"
                  onclick={() => selectSearchResult(result)}
                >
                  <div
                    class="flex h-8 w-8 shrink-0 items-center justify-center rounded-[var(--radius-md)]"
                  >
                    <MapPin class="h-4 w-4" />
                  </div>
                  <div class="min-w-0 flex-1">
                    <p class="truncate text-sm font-medium">
                      {result.name}
                    </p>
                    <p class="mt-0.5 text-[11px]">
                      {result.address || t('locationPicker.noAddress')}
                    </p>
                  </div>
                </button>
              {/each}
            {:else if searchText.trim().length === 0 && anchorHistory.length > 0}
              <div class="flex items-center justify-between px-1 pb-1">
                <div class="flex items-center gap-1.5 text-[11px] font-medium">
                  <Clock class="h-3.5 w-3.5" />
                  <span>{t('locationPicker.recentlyUsed')}</span>
                </div>
                <button class="flex items-center gap-1 text-[11px]" onclick={handleClearHistory}>
                  <Trash2 class="h-3 w-3" />
                  <span>{t('locationPicker.clearAll')}</span>
                </button>
              </div>
              {#each anchorHistory as item (item.timestamp)}
                <button
                  class="flex w-full items-center gap-3 rounded-[var(--radius-md)] px-3 py-2 text-left text-xs transition-all hover:-translate-y-0.5"
                  onclick={() => selectHistoryItem(item)}
                >
                  <div
                    class="flex h-8 w-8 shrink-0 items-center justify-center rounded-[var(--radius-md)]"
                  >
                    <Clock class="h-4 w-4" />
                  </div>
                  <div class="min-w-0 flex-1">
                    <p class="truncate text-sm font-medium">
                      {item.locationName}
                    </p>
                    <p class="mt-0.5 text-[11px]">{t('locationPicker.historyItemHintShort')}</p>
                  </div>
                </button>
              {/each}
            {:else if searchText.trim().length > 0 && !isSearching}
              <div class="rounded-[var(--radius-md)] border border-dashed px-3 py-4 text-[11px]">
                {t('locationPicker.noMatchDescShort')}
              </div>
            {/if}
          </div>
        </div>

        <!-- 当前位置摘要与操作 -->
        <div class="mt-auto space-y-3 border-t pt-3" style="border-color: var(--border-theme);">
          <div>
            <p class="text-body-lg truncate font-semibold" style="color: var(--text-primary)">
              {locationName || t('locationPicker.dragToSelect')}
            </p>
          </div>

          <div class="space-y-2">
            <Button
              variant="outline"
              class="h-11 w-full gap-2 text-sm font-semibold"
              onclick={handleConfirm}
              disabled={!locationName}
            >
              <MapPin class="h-4 w-4" />
              {confirmButtonText}
            </Button>
            {#if isAnchorMode}
              <Button
                variant="outline"
                class="h-11 w-full gap-2 text-sm font-semibold"
                onclick={() => (showShareModal = true)}
                disabled={!locationName}
              >
                <Share2 class="h-4 w-4" />
                {t('locationPicker.shareLocation')}
              </Button>
            {/if}
          </div>
        </div>
      </div>
    </Dialog.Content>
  </Dialog.Root>
</div>

<!-- 位置分享模态框 -->
<CreateLocationShareModal bind:open={showShareModal} {latitude} {longitude} {locationName} />

<style>
  /* Location dialog: mobile fullscreen, desktop centered large modal */
  @media (max-width: 639px) {
    :global(.dialog-responsive-position.location-dialog) {
      left: 0;
      right: 0;
      top: 0;
      bottom: 0;
      transform: none;
      width: 100vw;
      height: 100dvh;
      max-width: 100%;
      max-height: 100dvh;
      border-radius: 0;
      border: none;
      box-shadow: none;
    }
  }
  @media (min-width: 640px) {
    :global(.dialog-responsive-position.location-dialog) {
      width: 94vw;
      height: 90vh;
      max-width: 1400px;
      max-height: 90vh;
      overflow: hidden;
      border: 1px solid var(--border-theme);
    }
    :global(.dialog-responsive-position.location-dialog.is-fullscreen) {
      width: 100vw;
      height: 100dvh;
      max-width: 100vw;
      max-height: 100dvh;
      border-radius: 0;
      border: none;
      box-shadow: none;
    }
  }

  :global(.map-icon-marker) {
    width: 36px;
    height: 48px;
    display: flex;
    align-items: flex-end;
    justify-content: center;
    background: transparent;
  }

  :global(.map-icon-marker img) {
    display: block;
    width: 36px;
    height: 48px;
    background: transparent;
  }

  /* 同坐标重叠聚合点：数量气泡（替代旧的固定 3 头像 SVG）。
     锚点为 bottom-center，气泡底部三角指向坐标点。 */
  :global(.map-cluster-bubble) {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    min-width: 30px;
    height: 30px;
    padding: 0 8px;
    border-radius: var(--radius-pill);
    font-size: 14px;
    font-weight: 700;
    line-height: 1;
    cursor: pointer;
    color: var(--text-on-overlay);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.35);
    border: 2px solid var(--surface-bg);
    transition: transform 0.15s ease;
    will-change: transform;
  }

  :global(.map-cluster-bubble:hover) {
    transform: scale(1.12);
    z-index: 2;
  }

  /* 指向坐标点的三角指针 */
  :global(.map-cluster-bubble::after) {
    content: '';
    position: absolute;
    bottom: -6px;
    left: 50%;
    width: 0;
    height: 0;
    transform: translateX(-50%);
    border-left: 6px solid transparent;
    border-right: 6px solid transparent;
    border-top: 7px solid var(--surface-bg);
  }

  :global(.map-cluster-bubble--user) {
    background: var(--map-user);
  }
  :global(.map-cluster-bubble--user::after) {
    border-top-color: var(--map-user);
  }

  :global(.map-cluster-bubble--group) {
    background: var(--map-group);
  }
  :global(.map-cluster-bubble--group::after) {
    border-top-color: var(--map-group);
  }

  :global(.map-cluster-bubble--moment) {
    background: var(--map-post);
  }
  :global(.map-cluster-bubble--moment::after) {
    border-top-color: var(--map-post);
  }

  :global(.map-cluster-bubble--mixed) {
    background: var(--map-mixed);
  }
  :global(.map-cluster-bubble--mixed::after) {
    border-top-color: var(--map-mixed);
  }

  /* 聚合预览列表的类型标签，颜色随主题族变化（--map-* token） */
  :global(.badge-user) {
    background: var(--map-user);
    color: var(--text-on-overlay);
  }

  :global(.badge-group) {
    background: var(--map-group);
    color: var(--text-on-overlay);
  }

  :global(.badge-moment) {
    background: var(--map-post);
    color: var(--text-on-overlay);
  }
</style>
