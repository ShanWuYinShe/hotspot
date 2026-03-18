<script lang="ts">
    import { onMount, tick } from "svelte";
    import { assertSuccess } from "$lib/api/client";
    import {
        mapService,
        groupOverlappingRenderItems,
        isOverlapGroupRenderId,
    } from "$lib/api/services/map";
    import { mapAnchorStore } from "$lib/stores/map-anchor.svelte";
    import { locationStore } from "$lib/stores/location.svelte";
    import { toast } from "$lib/stores/toast.svelte";
    import {
        getAMapVisibleRegion,
        loadAMap,
        loadAMapPlugin,
    } from "$lib/utils/amap";
	import type {
		MapHeatBucket,
		MapHeatCanvas,
		MapHotspotItem,
		MapHotspotsCanvas,
		MapViewport,
		MapCanvasRequest,
		MapCanvasResponse,
	} from "$lib/types/api";
    import {
        getAnchorHistory,
        addToAnchorHistory,
        clearAnchorHistory,
        type AnchorHistoryItem,
    } from "$lib/utils/anchor-history";
    import * as Dialog from "$lib/components/ui/dialog";
    import { Button } from "$lib/components/ui/button";
    import MapPin from "lucide-svelte/icons/map-pin";
    import Plus from "lucide-svelte/icons/plus";
    import Minus from "lucide-svelte/icons/minus";
    import LocateFixed from "lucide-svelte/icons/locate-fixed";
    import Loader2 from "lucide-svelte/icons/loader-2";
    import ChevronUp from "lucide-svelte/icons/chevron-up";
    import ChevronDown from "lucide-svelte/icons/chevron-down";
    import Maximize2 from "lucide-svelte/icons/maximize-2";
    import Minimize2 from "lucide-svelte/icons/minimize-2";
    import { goto } from "$app/navigation";
    import XIcon from "lucide-svelte/icons/x";
    import { browser } from "$app/environment";
    import Share2 from "lucide-svelte/icons/share-2";
    import Search from "lucide-svelte/icons/search";
    import ListFilter from "lucide-svelte/icons/list-filter";
    import X from "lucide-svelte/icons/x";
    import Clock from "lucide-svelte/icons/clock";
    import Trash2 from "lucide-svelte/icons/trash-2";
    import CreateLocationShareModal from "./create-location-share-modal.svelte";
    import AnchorSummaryCard from "./anchor-summary-card.svelte";
    import { useKeyboardFocus } from "$lib/attachments";
    import { configStore } from "$lib/stores/config.svelte";
	import { toGcjLngLat, gcj02ToWgs84 } from "$lib/utils/geo";
    import { mode as themeMode } from "mode-watcher";

    type LocationPickerMode = "anchor" | "location_pick";

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
        mode = "anchor",
        initialLocation = null,
        onConfirm,
        onOpenChange,
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
    let lastRenderedViewportSignature = "";
    let lastMapZoom = 14;
    let renderMarkers: AMap.Marker[] = [];
    let hotspotsClusterer: any = null;
    let selectedMarker: AMap.Marker | null = null;
    let hasInit = $state(false);
    let hasSyncedFromAnchor = $state(false);
    let modalRoot = $state<HTMLDivElement | null>(null);
    const LONG_PRESS_DURATION_MS = 700;
    const GESTURE_HINT_DURATION_MS = 4000;
    const MAP_RENDER_IDLE_MS = 250;
    const GESTURE_HINT_COMBINED = "双击、右键或长按地图可快速定位";
    const GESTURE_HINT_TOUCH = "长按地图可快速定位";
    const GESTURE_HINT_POINTER = "双击或右键地图可快速定位";
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
    let districtSearch: any = null;

    type RenderMarkerKind = "cluster";
    type ClusterPreviewKind = "server_cluster" | "overlap_group";

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
        if (
            typeof window === "undefined" ||
            typeof window.matchMedia !== "function"
        ) {
            return GESTURE_HINT_COMBINED;
        }
        const prefersCoarsePointer =
            window.matchMedia("(pointer: coarse)").matches;
        const prefersHover = window.matchMedia("(hover: hover)").matches;
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
    let placementHintMessage = $state("");
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
    let searchText = $state("");
    let isSearching = $state(false);
    let searchInput = $state<HTMLInputElement | null>(null);
    let searchResults = $state.raw<
        Array<{
            name: string;
            address: string;
            latitude: number;
            longitude: number;
        }>
    >([]);
    let searchDebounceTimer: number | null = null;
    let activeSearchRequestId = 0;
    let placeSearch: AMap.PlaceSearch | null = null;

    // 锚点历史
    let anchorHistory = $state.raw<AnchorHistoryItem[]>([]);

    // 桌面端对话框全屏切换状态
    let isFullscreen = $state(false);

    // 手势提示持久化关闭标记（localStorage）
    const GESTURE_HINT_DISMISS_KEY =
        "hotspot_location_picker_gesture_hint_dismissed";
    let gestureHintDismissed = $state(false);

    const isAnchorMode = $derived(mode === "anchor");
    const modalTitle = $derived(isAnchorMode ? "设置浏览锚点" : "选择位置");
    const confirmButtonText = $derived(
        isAnchorMode ? "设为浏览锚点" : "使用此位置",
    );
    const confirmSuccessText = $derived(
        isAnchorMode ? "浏览锚点已更新" : "位置已选定",
    );
    const selectionDescription = $derived(
        isAnchorMode ? "用于浏览锚点的位置" : "用于当前操作的位置",
    );
    const mapModeLabel = $derived(isAnchorMode ? "浏览锚点" : "位置选择");
    const activeFilterCount = $derived(
        [showUsers, showGroups, showMoments].filter(Boolean).length,
    );

    type MapClickEvent = {
        lnglat?: AMap.LngLat;
    };

    const PLACEMENT_HINT_DEFAULT = "已定位到该位置";

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
                window.localStorage.setItem(GESTURE_HINT_DISMISS_KEY, "1");
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

    function clearRenderMarkers() {
        // 清理聚合器（如有）
        if (hotspotsClusterer) {
            try {
                // 部分版本提供 clearMarkers，部分版本通过 setMap(null) 解绑
                hotspotsClusterer.clearMarkers?.();
                hotspotsClusterer.setMap?.(null);
            } catch {
                // 忽略清理异常，避免影响后续渲染
            }
            hotspotsClusterer = null;
        }

        for (const marker of renderMarkers) {
            marker.setMap(null);
            marker.remove();
        }
        renderMarkers = [];
        hotspotPreviewItems = [];
    }

    function ensureSelectedMarker() {
        if (!map || selectedMarker) return;
        const AMapAny = AMap as any;
        const centerIcon = new AMapAny.Icon({
            size: new AMapAny.Size(40, 50),
            image: "/map/marker_center_pin.svg",
            imageSize: new AMapAny.Size(40, 50),
            imageOffset: new AMapAny.Pixel(0, 0),
        });
        const [gcjLng, gcjLat] = toGcjLngLat({
            latitude,
            longitude,
        });
        selectedMarker = new AMap.Marker({
            position: [gcjLng, gcjLat],
            icon: centerIcon,
            anchor: "bottom-center",
            offset: new AMapAny.Pixel(0, 0),
            clickable: false,
            map,
        });
    }

    function updateSelectedMarker() {
        if (!map) return;
        ensureSelectedMarker();
        if (!selectedMarker) return;
        const [gcjLng, gcjLat] = toGcjLngLat({
            latitude,
            longitude,
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
    }

    function toggleFilterPanel() {
        const nextOpen = !showFilterPanel;
        if (nextOpen) {
            showSearch = false;
            closeClusterPreview();
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
        window.addEventListener("pointerdown", handler, true);
        detachOutsidePointerListener = () => {
            window.removeEventListener("pointerdown", handler, true);
            detachOutsidePointerListener = null;
        };
    }

    function getViewportSignature(viewport: MapViewport) {
        return [
            viewport.bounds.north.toFixed(6),
            viewport.bounds.south.toFixed(6),
            viewport.bounds.east.toFixed(6),
            viewport.bounds.west.toFixed(6),
            viewport.zoom.toFixed(2),
        ].join("|");
    }

    function buildViewport(): MapViewport | null {
        if (!map) return null;
        const visibleRegion = getAMapVisibleRegion(map);
        if (!visibleRegion) return null;
        const northEastWgs = gcj02ToWgs84({
            latitude: visibleRegion.bounds.north,
            longitude: visibleRegion.bounds.east,
        });
        const southWestWgs = gcj02ToWgs84({
            latitude: visibleRegion.bounds.south,
            longitude: visibleRegion.bounds.west,
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
                west,
            },
            zoom: Math.max(1, Math.min(20, visibleRegion.zoom)),
        };
        return viewport;
    }

    function getMarkerIconUrl(_kind: RenderMarkerKind): string {
        // 仅 cluster 热力格网用于地图展示
        return "/map/marker_mixed_cluster.svg";
    }

    function getHotspotMarkerIcon(item: MapHotspotItem): string {
        if (item.object_type === "user") return "/map/marker_user.svg";
        if (item.object_type === "group") return "/map/marker_group.svg";
        if (item.object_type === "moment") return "/map/marker_moment.svg";
        return "/map/marker_user.svg";
    }

    function shouldDisplayItem(_item: RenderItem) {
        // 目前 /map/render 仅返回热力格网统计项，是否显示由筛选开关整体决定
        return showUsers || showGroups || showMoments;
    }

    function resolveMapFilter():
        | { data_type?: "user" | "group" | "moment" | "all" }
        | undefined {
        const noneSelected = !showUsers && !showGroups && !showMoments;
        const allSelected = showUsers && showGroups && showMoments;
        if (noneSelected || allSelected) {
            return undefined;
        }
        if (showUsers && !showGroups && !showMoments) {
            return { data_type: "user" };
        }
        if (!showUsers && showGroups && !showMoments) {
            return { data_type: "group" };
        }
        if (!showUsers && !showGroups && showMoments) {
            return { data_type: "moment" };
        }
        return { data_type: "all" };
    }

    function applyCurrentFilter() {
        // 热力图模式下，本地根据筛选条件对现有聚合结果做一次快速重渲染；
        // 热点模式下，筛选条件会导致重新请求 /map/hotspots，由后端过滤，故无需本地再过滤。
        renderMapItems(
            latestRenderItems.filter((item) => shouldDisplayItem(item)),
        );
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

    function toggleRenderFilter(kind: "user" | "group" | "moment") {
        if (kind === "user") showUsers = !showUsers;
        if (kind === "group") showGroups = !showGroups;
        if (kind === "moment") showMoments = !showMoments;

        lastRenderedViewportSignature = "";
        if (!showUsers && !showGroups && !showMoments) {
            latestRenderItems = [];
            clearRenderMarkers();
            return;
        }

        // 先在前端根据当前筛选条件重新渲染已有 items，确保 UI 立即响应
        applyCurrentFilter();
        void fetchMapRenderData();
    }

    function getMarkerBadgeText(item: RenderItem) {
        return `${item.total_count} 个点`;
    }

    function getItemTypeLabel(_item: RenderItem) {
        return "内容";
    }

    function getItemBadgeClass(_item: RenderItem) {
        return "bg-slate-100 text-slate-700";
    }

    function buildOverlapPreviewDescription(items: RenderItem[]) {
        const total = items.reduce((sum, item) => sum + item.total_count, 0);
        return `${total} 个内容`;
    }

    function openClusterPreview(item: RenderItem) {
        const overlapItems = overlapGroupItems.get(item.render_id) ?? [];
        const isOverlapGroup =
            isOverlapGroupRenderId(item.render_id) && overlapItems.length > 0;
        const currentZoom = map?.getZoom() ?? 14;
        const baseCount =
            item.total_count ||
            overlapItems.reduce((sum, it) => sum + it.total_count, 0);

        clusterPreview = {
            kind: isOverlapGroup ? "overlap_group" : "server_cluster",
            render_id: item.render_id,
            title: `${baseCount} 个内容`,
            description: isOverlapGroup
                ? `${buildOverlapPreviewDescription(overlapItems)}。继续放大后会拆成单独单元。`
                : "当前层级仍在聚合显示，继续放大后会拆成更细的区域。",
            count: baseCount,
            latitude: item.location.latitude,
            longitude: item.location.longitude,
            zoomTarget:
                baseCount > 100
                    ? Math.min(20, currentZoom + 2)
                    : Math.min(20, currentZoom + 3),
            items: overlapItems,
        };
    }

    function zoomToClusterPreview() {
        if (!clusterPreview || !map) {
            return;
        }
        const [gcjLng, gcjLat] = toGcjLngLat({
            latitude: clusterPreview.latitude,
            longitude: clusterPreview.longitude,
        });
        map.setCenter([gcjLng, gcjLat]);
        map.setZoom(clusterPreview.zoomTarget);
        showPlacementHint(
            clusterPreview.kind === "overlap_group"
                ? "已放大到同点内容展开层级"
                : "已放大到更细地图层级",
        );
        closeClusterPreview();
    }

    function focusPreviewEntity(item: RenderItem) {
        closeClusterPreview();
        focusRenderItem(item);
    }

    function createMarkerForItem(item: RenderItem, offset?: [number, number]) {
        if (!map) return null;
        const position = toGcjLngLat({
            latitude: item.location.latitude,
            longitude: item.location.longitude,
        });

        const AMapAny = AMap as any;
        const icon = new AMapAny.Icon({
            size: new AMapAny.Size(40, 50),
            image: getMarkerIconUrl("cluster"),
            imageSize: new AMapAny.Size(40, 50),
            imageOffset: new AMapAny.Pixel(0, 0),
        });

        const marker = new AMap.Marker({
            position,
            icon,
            anchor: "bottom-center",
            offset,
            clickable: true,
            map,
        });

        (
            marker as unknown as {
                on: (type: "click", handler: () => void) => void;
            }
        ).on("click", () => {
            suppressNextMapClick = true;
            focusRenderItemFromMarkerClick(item);
        });

        (
            marker as unknown as {
                setLabel: (opts: {
                    content: string;
                    direction?: string;
                }) => void;
            }
        ).setLabel({
            content: getMarkerBadgeText(item),
            direction: "top",
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
        showPlacementHint(
            `已聚焦 ${item.total_count} 个内容，双击或长按可设为选中位置`,
        );
    }

    function focusRenderItem(item: RenderItem) {
        showSearch = false;
        showFilterPanel = false;
        openClusterPreview(item);
        const [gcjLng, gcjLat] = toGcjLngLat({
            latitude: item.location.latitude,
            longitude: item.location.longitude,
        });
        map?.setCenter([gcjLng, gcjLat]);
        showPlacementHint(`已聚焦 ${item.total_count} 个内容`);
    }

    async function resolveClusterRegion(
        item: RenderItem,
    ): Promise<RegionKey | null> {
        const cacheKey = item.render_id;
        const cached = regionResolveCache.get(cacheKey);
        if (cached) return cached;

        const [gcjLng, gcjLat] = toGcjLngLat({
            latitude: item.location.latitude,
            longitude: item.location.longitude,
        });

        await ensureGeocoder();
        if (!geocoder) return null;

        const key = await new Promise<RegionKey | null>((resolve) => {
            geocoder!.getAddress(
                [gcjLng, gcjLat],
                (status: string, result: any) => {
                    if (status !== "complete" || !result?.regeocode) {
                        resolve(null);
                        return;
                    }
                    const comp = (result.regeocode as any)
                        .addressComponent as any;
                    const rawAdcode: string | undefined =
                        comp?.adcode ?? comp?.citycode;
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
                            centerIsOfficial: false,
                        });
                    }
                    resolve(regionKey);
                },
            );
        });

        if (key) {
            regionResolveCache.set(cacheKey, key);
        }
        return key;
    }

    async function ensureRegionCenter(meta: RegionMeta): Promise<RegionMeta> {
        if (meta.centerIsOfficial) return meta;
        await ensureDistrictSearch();
        if (!districtSearch) return meta;
        const updated = await new Promise<RegionMeta>((resolve) => {
            (districtSearch as any).search(
                meta.adcode,
                (status: string, result: any) => {
                    if (status !== "complete" || !result.districts?.length) {
                        resolve(meta);
                        return;
                    }
                    const d = result.districts[0];
                    const center = d.center;
                    if (!center) {
                        resolve(meta);
                        return;
                    }
                    const next: RegionMeta = {
                        ...meta,
                        center: { lat: center.getLat(), lng: center.getLng() },
                        centerIsOfficial: true,
                    };
                    resolve(next);
                },
            );
        });
        regionMetaCache.set(updated.key, updated);
        return updated;
    }

    async function renderRegionClusters(items: RenderItem[]) {
        if (!map) return;
        const groups = new Map<
            RegionKey,
            { items: RenderItem[]; total: number }
        >();
        for (const item of items) {
            const regionKey = await resolveClusterRegion(item);
            if (!regionKey) continue;
            const group = groups.get(regionKey) ?? { items: [], total: 0 };
            group.items.push(item);
            group.total += item.total_count;
            groups.set(regionKey, group);
        }

        clearRenderMarkers();
        const markers: AMap.Marker[] = [];
        const AMapAny = AMap as any;
        for (const [key, group] of groups) {
            const baseMeta = regionMetaCache.get(key);
            if (!baseMeta) continue;
            const meta = await ensureRegionCenter(baseMeta);
            const position: [number, number] = [
                meta.center.lng,
                meta.center.lat,
            ];
            const icon = new AMapAny.Icon({
                size: new AMapAny.Size(40, 50),
                image: "/map/marker_cluster.svg",
                imageSize: new AMapAny.Size(40, 50),
                imageOffset: new AMapAny.Pixel(0, 0),
            });
            const marker = new AMap.Marker({
                position,
                icon,
                anchor: "bottom-center",
                clickable: true,
                map,
            });
            (marker as any).setLabel?.({
                content: `${meta.name} ${group.total} 个内容`,
                direction: "top",
            });
            (marker as any).on("click", () => {
                // 低缩放级别下，点击行政区聚合 marker 仅展示该区域的聚合信息，不移动相机
                clusterPreview = {
                    kind: "server_cluster",
                    render_id: key,
                    title: `${meta.name} ${group.total} 个内容`,
                    description:
                        "当前层级按城市/区域聚合显示，继续放大后可查看更精细内容。",
                    count: group.total,
                    latitude: group.items[0]?.location.latitude ?? 0,
                    longitude: group.items[0]?.location.longitude ?? 0,
                    zoomTarget:
                        group.total > 100
                            ? Math.min(20, (map?.getZoom() ?? lastMapZoom) + 2)
                            : Math.min(20, (map?.getZoom() ?? lastMapZoom) + 3),
                    items: group.items,
                };
            });
            markers.push(marker);
        }
        renderMarkers = markers;
    }

    async function renderMapItems(items: RenderItem[]) {
        if (!map) return;
        const zoom = map.getZoom() ?? lastMapZoom;
        if (zoom <= ADMIN_REGION_ZOOM_THRESHOLD) {
            await renderRegionClusters(items);
            updateSelectedMarker();
            return;
        }
        clearRenderMarkers();
        const { merged, groups } = groupOverlappingRenderItems(items, zoom);
        overlapGroupItems = groups;
        const currentCluster = clusterPreview;
        if (
            currentCluster &&
            !items.some((item) => item.render_id === currentCluster.render_id)
        ) {
            closeClusterPreview();
        }
        renderMarkers = merged
            .map(({ item }) => createMarkerForItem(item))
            .filter((marker): marker is AMap.Marker => Boolean(marker));
        updateSelectedMarker();
    }

	async function fetchMapRenderData() {
		if (!map || !open || !isMapAvailable) return;
		if (!showUsers && !showGroups && !showMoments) {
			latestRenderItems = [];
			clearRenderMarkers();
			lastRenderedViewportSignature = "";
			return;
		}
		const viewport = buildViewport();
		if (!viewport) return;
		const filter = resolveMapFilter();
		const signature = `${getViewportSignature(viewport)}|${filter?.data_type ?? "all"}`;
		if (
			signature === lastRenderedViewportSignature &&
			renderMarkers.length > 0
		) {
			applyCurrentFilter();
			return;
		}

		const requestId = ++activeRenderRequestId;
		try {
			const request: MapCanvasRequest = {
				viewport,
				filter: filter ? { data_type: filter.data_type } : undefined,
				pagination: { limit: 100 },
			};
			const response = await mapService.canvas(request);
			assertSuccess(response);
			if (requestId !== activeRenderRequestId) {
				return;
			}
			lastRenderedViewportSignature = signature;
			const canvas = (response.data ?? {}) as MapCanvasResponse;

			// 先清空当前渲染状态
			latestRenderItems = [];
			clearRenderMarkers();
			hotspotsClusterer = null;
			hotspotPreviewItems = [];

			if (canvas.hotspots) {
				const data = canvas.hotspots as MapHotspotsCanvas;
				const allItems: MapHotspotItem[] = data.items ?? [];
				const items: MapHotspotItem[] = allItems.filter((item: MapHotspotItem) => {
					if (item.object_type === "user" && !showUsers) return false;
					if (item.object_type === "group" && !showGroups) return false;
					if (item.object_type === "moment" && !showMoments) return false;
					return true;
				});

				if (items.length > 0) {
					await loadAMapPlugin(["AMap.MarkerClusterer"]);
					const AMapAny = AMap as any;
					const markers: AMap.Marker[] = [];
					for (const item of items) {
						const [gcjLng, gcjLat] = toGcjLngLat({
							latitude: item.location.latitude,
							longitude: item.location.longitude,
						});
						const icon = new AMapAny.Icon({
							size: new AMapAny.Size(40, 50),
							image: getHotspotMarkerIcon(item),
							imageSize: new AMapAny.Size(40, 50),
							imageOffset: new AMapAny.Pixel(0, 0),
						});
						const marker = new AMapAny.Marker({
							position: [gcjLng, gcjLat],
							icon,
							anchor: "bottom-center",
							clickable: true,
							map,
						});
						(marker as any).setExtData?.(item);
						marker.on("click", (ev: any) => {
							ev?.stopPropagation?.();
							hotspotPreviewItems = [item];
						});
						markers.push(marker);
					}
					hotspotsClusterer = new AMapAny.MarkerClusterer(map, markers, {
						gridSize: 60,
						minClusterSize: 2,
						maxZoom: 18,
						renderClusterMarker: (context: any) => {
							const count: number = context.count ?? 0;
							const children: AMap.Marker[] = (context.markers ?? context.clusterData ?? []) as AMap.Marker[];
							let hasUser = false;
							let hasGroup = false;
							let hasMoment = false;
							for (const m of children) {
								const data = (m as any).getExtData?.() as MapHotspotItem | undefined;
								if (!data) continue;
								if (data.object_type === "user") hasUser = true;
								if (data.object_type === "group") hasGroup = true;
								if (data.object_type === "moment") hasMoment = true;
							}
							let cls = "map-render-marker-cluster";
							if (hasUser && !hasGroup && !hasMoment) cls = "map-render-marker-user";
							else if (!hasUser && hasGroup && !hasMoment) cls = "map-render-marker-group";
							else if (!hasUser && !hasGroup && hasMoment) cls = "map-render-marker-moment";

							const div = document.createElement("div");
							div.className = `map-render-marker ${cls}`;
							const badge = document.createElement("div");
							badge.className = "map-render-marker-badge";
							badge.textContent = String(count);
							div.appendChild(badge);
							div.onclick = (ev: MouseEvent) => {
								ev.stopPropagation();
								const clusterItems: MapHotspotItem[] = [];
								for (const m of children) {
									const data = (m as any).getExtData?.() as MapHotspotItem | undefined;
									if (data) {
										clusterItems.push(data);
									}
								}
								hotspotPreviewItems = clusterItems;
							};
							context.marker.setContent(div);
						},
					});
					renderMarkers = markers;
				}
			} else if (canvas.heatmap) {
				const data = canvas.heatmap as MapHeatCanvas;
				const buckets: MapHeatBucket[] = data.items ?? [];
				latestRenderItems = buckets.map(
					(bucket: MapHeatBucket, index: number): RenderItem => ({
						...bucket,
						render_id: `${index}_${bucket.location.latitude.toFixed(6)}_${bucket.location.longitude.toFixed(6)}`,
					}),
				);
				applyCurrentFilter();
			}
		} catch {
			if (requestId === activeRenderRequestId) {
				toast.error("地图渲染数据加载失败");
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
        options?: { hintMessage?: string },
    ) {
        cancelReverseGeocodeTimer();
        closeClusterPreview();
        const wgs = gcj02ToWgs84({
            latitude: mapLat,
            longitude: mapLng,
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

    function getLngLatFromTouch(
        event: TouchEvent,
    ): { lat: number; lng: number } | null {
        if (!map || !mapContainer) return null;
        const touch = event.touches[0] ?? event.changedTouches[0];
        if (!touch) return null;
        const rect = mapContainer.getBoundingClientRect();
        const pixel = {
            x: touch.clientX - rect.left,
            y: touch.clientY - rect.top,
        };
        const lnglat = (
            map as unknown as {
                containerToLngLat: (pixel: {
                    x: number;
                    y: number;
                }) => AMap.LngLat;
            }
        ).containerToLngLat(pixel);
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
                hintMessage: isAnchorMode
                    ? "已定位，请点击下方按钮确认"
                    : "已定位到该位置",
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
        mapContainer.addEventListener("touchstart", handleTouchStart, {
            passive: true,
        });
        mapContainer.addEventListener("touchmove", handleTouchMove, {
            passive: true,
        });
        mapContainer.addEventListener("touchend", handleTouchEnd, {
            passive: true,
        });
        detachTouchListeners = () => {
            mapContainer?.removeEventListener("touchstart", handleTouchStart);
            mapContainer?.removeEventListener("touchmove", handleTouchMove);
            mapContainer?.removeEventListener("touchend", handleTouchEnd);
            detachTouchListeners = null;
        };
    }

    function registerContextMenuBlocker() {
        detachContextMenuListener?.();
        if (!mapContainer) return;
        const handler = (event: MouseEvent) => {
            event.preventDefault();
        };
        mapContainer.addEventListener("contextmenu", handler);
        detachContextMenuListener = () => {
            mapContainer?.removeEventListener("contextmenu", handler);
            detachContextMenuListener = null;
        };
    }

    async function ensureGeocoder() {
        if (geocoder) return;
        await loadAMapPlugin(["AMap.Geocoder"]);
        geocoder = new AMap.Geocoder();
    }

    async function ensureDistrictSearch() {
        if (districtSearch) return;
        await loadAMapPlugin(["AMap.DistrictSearch"]);
        const AMapAny = AMap as any;
        districtSearch = new AMapAny.DistrictSearch({
            subdistrict: 0,
            extensions: "base",
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
            closeClusterPreview();
            searchText = "";
            searchResults = [];
            isSearching = false;
            await tick();
            searchInput?.focus();
        }
    }

    async function ensurePlaceSearch() {
        if (placeSearch) return;
        await loadAMapPlugin(["AMap.PlaceSearch"]);
        placeSearch = new AMap.PlaceSearch({
            pageSize: 10,
            extensions: "base",
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

            const results = await new Promise<
                Array<{
                    name: string;
                    address: string;
                    latitude: number;
                    longitude: number;
                }>
            >((resolve) => {
                const centerForSearch = toGcjLngLat({
                    latitude,
                    longitude,
                });
                placeSearch!.searchNearBy(
                    keyword,
                    centerForSearch,
                    configStore.searchRadiusMax, // 使用后端配置的搜索半径上限（米）
                    (status: string, result: AMap.PlaceSearchResult) => {
                        if (
                            status === "complete" &&
                            result.poiList?.pois?.length
                        ) {
                            resolve(
                                result.poiList.pois.map((poi) => {
                                    const wgs = gcj02ToWgs84({
                                        latitude: poi.location.getLat(),
                                        longitude: poi.location.getLng(),
                                    });
                                    return {
                                        name: poi.name,
                                        address: [poi.adname, poi.address]
                                            .filter(Boolean)
                                            .join(" "),
                                        latitude: wgs.latitude,
                                        longitude: wgs.longitude,
                                    };
                                }),
                            );
                        } else {
                            resolve([]);
                        }
                    },
                );
            });

            if (requestId !== activeSearchRequestId) {
                return;
            }

            if (results.length > 0) {
                searchResults = results;
            } else {
                toast.error("未找到该地点，请换个关键词试试");
            }
        } catch {
            if (requestId !== activeSearchRequestId) {
                return;
            }
            toast.error("搜索失败，请稍后重试");
        } finally {
            if (requestId === activeSearchRequestId) {
                isSearching = false;
            }
        }
    }

    function selectSearchResult(result: {
        name: string;
        address: string;
        latitude: number;
        longitude: number;
    }) {
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
            longitude: result.longitude,
        });
        map?.setCenter([gcjLng, gcjLat]);
        showSearch = false;
        searchText = "";
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
            longitude: item.longitude,
        });
        map?.setCenter([gcjLng, gcjLat]);
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
                locationName = "自定义位置";
                return "自定义位置";
            }
            const [gcjLng, gcjLat] = toGcjLngLat({
                latitude: lat,
                longitude: lng,
            });
            return await new Promise<string>((resolve) => {
                currentGeocoder.getAddress(
                    [gcjLng, gcjLat],
                    (status: string, result: AMap.GeocodeResult) => {
                        if (status === "complete" && result.regeocode) {
                            const poiName = result.regeocode.pois?.[0]?.name;
                            const resolved =
                                poiName ||
                                result.regeocode.formattedAddress ||
                                "自定义位置";
                            locationName = resolved;
                            resolve(resolved);
                            return;
                        }
                        locationName = "自定义位置";
                        resolve("自定义位置");
                    },
                );
            });
        } catch {
            locationName = "自定义位置";
            return "自定义位置";
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
            hintMessage: isAnchorMode
                ? "已定位，请点击下方按钮确认"
                : "已定位到该位置",
        });
    }

    async function handleRightClick(event: MapClickEvent) {
        if (!map) return;
        const lnglat = event.lnglat ?? map.getCenter();
        const nextLat = lnglat.getLat();
        const nextLng = lnglat.getLng();
        await moveToCoordinate(nextLat, nextLng, {
            hintMessage: isAnchorMode
                ? "已定位，请点击下方按钮确认"
                : "已定位到该位置",
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
                longitude,
            });

            map = new AMap.Map(mapContainer, {
                center: [gcjLng, gcjLat],
                zoom: 14,
                viewMode: "2D",
                doubleClickZoom: false,
                resizeEnable: true,
            });
            try {
                const anyMap = map as unknown as {
                    setMapStyle?: (style: string) => void;
                };
                if (anyMap?.setMapStyle) {
                    anyMap.setMapStyle(
                        currentMode === "dark"
                            ? "amap://styles/dark"
                            : "amap://styles/normal",
                    );
                }
            } catch {
                // ignore style errors
            }
            ensureSelectedMarker();
            registerTouchListeners();
            registerContextMenuBlocker();

            map.on("movestart", () => {
                cancelReverseGeocodeTimer();
            });
            map.on("moveend", () => {
                handleMoveEnd();
            });
            map.on("zoomstart", () => {
                cancelReverseGeocodeTimer();
            });
            map.on("zoomend", () => {
                handleZoomEnd();
            });
            map.on("click", handleMapSurfaceClick);
            map.on("dblclick", handleDoubleClick);
            map.on("rightclick", handleRightClick);
            await Promise.allSettled([
                reverseGeocode(latitude, longitude),
                fetchMapRenderData(),
            ]);
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
                longitude,
            });
            map?.setZoomAndCenter(15, [gcjLng, gcjLat]);
            updateSelectedMarker();
            const resolvedName = await reverseGeocode(latitude, longitude);
            locationStore.setLocation({
                point: {
                    latitude,
                    longitude,
                },
                location_name: resolvedName,
            });
        } else {
            toast.error("无法获取当前位置");
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
            longitude: anchorLng,
        });
        map?.setZoomAndCenter(14, [gcjLng, gcjLat]);
        updateSelectedMarker();
        if (mapAnchorStore.name) {
            locationName = mapAnchorStore.name;
        } else {
            await reverseGeocode(anchorLat, anchorLng);
        }
        hideGestureHint();
        showPlacementHint("已回到浏览锚点位置");
    }

    function handleConfirm() {
        const normalizedName = locationName.trim() || "自定义位置";
        if (isAnchorMode) {
            mapAnchorStore.setAnchor(
                latitude,
                longitude,
                normalizedName,
                searchRadius,
            );
            addToAnchorHistory(latitude, longitude, normalizedName);
        } else {
            onConfirm?.({
                latitude,
                longitude,
                locationName: normalizedName,
                radius: searchRadius,
            });
        }
        toast.success(confirmSuccessText);
        handleClose();
    }

    function handleRetryMapLoad() {
        clearRenderMarkers();
        clearSelectedMarker();
        closeClusterPreview();
        lastRenderedViewportSignature = "";
        map?.destroy();
        map = null;
        isMapAvailable = true;
        hasInit = false;
        void initMap();
    }

    function handleClose() {
        if (searchDebounceTimer) {
            clearTimeout(searchDebounceTimer);
            searchDebounceTimer = null;
        }
        activeSearchRequestId += 1;
        isSearching = false;
        showSearch = false;
        showFilterPanel = false;
        showShareModal = false;
        closeClusterPreview();
        open = false;
        onOpenChange?.(false);
    }

    $effect(() => {
        if (open) {
            if (!gestureHintVisible) {
                showGestureHint();
            }
            if (!hasSyncedFromAnchor) {
                const fallbackName =
                    mapAnchorStore.name || mapAnchorStore.displayName;
                latitude = initialLocation?.latitude ?? mapAnchorStore.latitude;
                longitude =
                    initialLocation?.longitude ?? mapAnchorStore.longitude;
                locationName =
                    initialLocation?.locationName?.trim() || fallbackName;
                searchRadius = initialLocation?.radius ?? mapAnchorStore.radius;
                anchorHistory = isAnchorMode ? getAnchorHistory() : [];
                showSearch = false;
                showFilterPanel = false;
                searchText = "";
                searchResults = [];
                isSearching = false;

                if (!hasInit) {
                    initMap();
                } else {
                    updateSelectedMarker();
                    const [gcjLng, gcjLat] = toGcjLngLat({
                        latitude,
                        longitude,
                    });
                    map?.setZoomAndCenter(14, [gcjLng, gcjLat]);
                }
                hasSyncedFromAnchor = true;
            }
        } else {
            hasSyncedFromAnchor = false;
            lastRenderedViewportSignature = "";
            closeClusterPreview();
            activeSearchRequestId += 1;
            isSearching = false;
            showSearch = false;
            showFilterPanel = false;
            hideGestureHint();
            placementHintVisible = false;
            // 如果当前 selection 的屏幕区域与位置选择器弹窗区域有明显重叠
            // 则认为选中状态是由本弹窗交互产生的，在关闭前主动清理，避免浏览器
            // 在 DOM 卸载后将选区“迁移”到主内容区域导致大段文字被选中
            if (browser && modalRoot) {
                const selection = window.getSelection();
                if (selection && selection.rangeCount > 0) {
                    const range = selection.getRangeAt(0);
                    const rects = Array.from(range.getClientRects());
                    if (rects.length > 0) {
                        const modalRect = modalRoot.getBoundingClientRect();
                        let intersects = false;
                        for (const rect of rects) {
                            const separated =
                                rect.right < modalRect.left ||
                                rect.left > modalRect.right ||
                                rect.bottom < modalRect.top ||
                                rect.top > modalRect.bottom;
                            if (!separated) {
                                intersects = true;
                                break;
                            }
                        }
                        if (intersects) {
                            selection.removeAllRanges();
                        }
                    }
                }
            }
            // 如果当前选区来自位置选择器自身，在关闭弹窗时清理掉避免浏览器把选区“迁移”到主内容
            if (browser && modalRoot) {
                const selection = window.getSelection();
                if (selection && selection.rangeCount > 0) {
                    const range = selection.getRangeAt(0);
                    const node = range.commonAncestorContainer as Node;
                    if (modalRoot.contains(node)) {
                        selection.removeAllRanges();
                    }
                }
            }
            // 关闭弹窗时销毁地图实例，避免下次打开时复用已失效的 DOM 容器
            clearRenderMarkers();
            clearSelectedMarker();
            map?.destroy();
            map = null;
            isMapAvailable = true;
            hasInit = false;
        }
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
                gestureHintDismissed =
                    window.localStorage.getItem(GESTURE_HINT_DISMISS_KEY) ===
                    "1";
            } catch {
                gestureHintDismissed = false;
            }
        }
        registerOutsidePointerListener();
    });
</script>

<div bind:this={modalRoot}>
    <Dialog.Root
        bind:open
        onOpenChange={(value) => {
            open = value;
            onOpenChange?.(value);
        }}
    >
        <Dialog.Content
            class={`flex h-[90dvh] max-h-[90dvh] flex-col p-0 transition-all duration-200 ease-out sm:flex-row sm:gap-4 select-none ${
                isFullscreen
                    ? "sm:h-screen sm:max-h-screen sm:w-screen sm:max-w-none sm:rounded-none sm:border-0 sm:bg-background sm:p-4"
                    : "sm:h-[88vh] sm:w-[92vw] sm:max-w-6xl lg:max-w-7xl sm:rounded-3xl sm:border sm:border-border/60 sm:bg-card/95 sm:p-6"
            }`}
            hideClose
        >
            <!-- 移动端简化标题 -->
            <div
                class="flex items-center justify-between border-b border-border/60 bg-[linear-gradient(180deg,rgba(255,255,255,0.95),rgba(248,250,252,0.92))] px-4 py-3 backdrop-blur dark:bg-[linear-gradient(180deg,rgba(30,30,36,0.95),rgba(24,24,30,0.92))] sm:hidden"
            >
                <div class="flex min-w-0 items-center gap-3">
                    <div
                        class="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl bg-[linear-gradient(135deg,#0f172a,#2563eb)] shadow-[0_10px_24px_rgba(37,99,235,0.22)]"
                    >
                        <MapPin class="h-5 w-5 text-white" />
                    </div>
                    <div class="min-w-0">
                        <p
                            class="text-[11px] font-semibold uppercase tracking-[0.16em] text-primary/70"
                        >
                            {mapModeLabel}
                        </p>
                        <span class="block truncate font-semibold"
                            >{modalTitle}</span
                        >
                    </div>
                </div>
                <div class="flex items-center gap-2">
                    <Button
                        variant="ghost"
                        size="sm"
                        class="h-9 w-9 rounded-2xl border border-border/60 bg-white/70 p-0 shadow-sm dark:bg-card/80"
                        onclick={toggleSearch}
                    >
                        <Search class="h-4 w-4" />
                        <span class="sr-only">搜索</span>
                    </Button>
                    <Button
                        variant="ghost"
                        size="sm"
                        data-location-filter-trigger="true"
                        class="h-9 w-9 rounded-2xl border border-border/60 bg-white/70 p-0 shadow-sm dark:bg-card/80"
                        onclick={toggleFilterPanel}
                    >
                        <ListFilter class="h-4 w-4" />
                        <span class="sr-only">筛选</span>
                    </Button>
                    <Button
                        variant="ghost"
                        size="sm"
                        class="h-9 w-9 rounded-2xl border border-border/60 bg-white/70 p-0 shadow-sm dark:bg-card/80"
                        onclick={handleClose}
                    >
                        <span class="sr-only">关闭</span>
                        <svg
                            class="h-4 w-4"
                            fill="none"
                            stroke="currentColor"
                            viewBox="0 0 24 24"
                        >
                            <path
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                stroke-width="2"
                                d="M6 18L18 6M6 6l12 12"
                            ></path>
                        </svg>
                    </Button>
                </div>
            </div>
            <!-- 左侧：地图容器 -->
            <div
                class="relative flex-1 overflow-hidden sm:rounded-3xl sm:border sm:border-border/60"
            >
                <div
                    class="h-full w-full touch-none"
                    bind:this={mapContainer}
                ></div>

                <!-- 桌面端地图工具栏：右上角仅保留“筛选”入口，与地图区域靠近 -->
                <div
                    class="pointer-events-none absolute right-4 top-4 z-30 hidden flex-col gap-2 sm:flex"
                >
                    <div class="pointer-events-auto flex flex-col gap-2">
                        <Button
                            data-location-filter-trigger="true"
                            size="icon"
                            variant="ghost"
                            class="h-11 w-11 rounded-[20px] border border-white/60 bg-white/78 text-foreground shadow-[0_14px_28px_rgba(15,23,42,0.14)] backdrop-blur hover:bg-white/90 active:bg-white dark:border-border/60 dark:bg-card/90 dark:shadow-[0_14px_28px_rgba(0,0,0,0.4)] dark:hover:bg-card dark:active:bg-card"
                            onclick={toggleFilterPanel}
                            aria-label="筛选地图内容"
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
                            class="h-11 w-11 rounded-[20px] border border-slate-300/70 bg-white/95 text-foreground shadow-[0_16px_30px_rgba(15,23,42,0.26)] transition hover:border-primary/70 hover:bg-white hover:text-primary dark:border-border/60 dark:bg-card/95 dark:shadow-[0_16px_30px_rgba(0,0,0,0.5)] dark:hover:border-primary/70 dark:hover:bg-card dark:hover:text-primary"
                            onclick={handleUseCurrentLocation}
                            disabled={isLocating}
                            aria-label="定位到当前位置"
                        >
                            {#if isLocating}
                                <Loader2 class="h-4 w-4 animate-spin" />
                            {:else}
                                <LocateFixed
                                    class="h-6 w-6 text-blue-600 dark:text-blue-400"
                                />
                            {/if}
                        </Button>
                        {#if isAnchorMode}
                            <Button
                                size="icon"
                                variant="ghost"
                                class="h-11 w-11 rounded-[20px] border border-slate-300/70 bg-white/95 text-foreground shadow-[0_16px_30px_rgba(15,23,42,0.26)] transition hover:border-primary/70 hover:bg-white hover:text-primary dark:border-border/60 dark:bg-card/95 dark:shadow-[0_16px_30px_rgba(0,0,0,0.5)] dark:hover:border-primary/70 dark:hover:bg-card dark:hover:text-primary"
                                onclick={handleBackToBrowseAnchor}
                                aria-label="回到浏览锚点位置"
                            >
                                <img
                                    src="/map/marker_center_pin.svg"
                                    alt="浏览锚点"
                                    class="h-7 w-7"
                                />
                            </Button>
                        {/if}
                        <Button
                            size="icon"
                            variant="ghost"
                            class="h-11 w-11 rounded-[20px] border border-slate-300/70 bg-white/95 text-foreground shadow-[0_16px_30px_rgba(15,23,42,0.26)] transition hover:border-primary/70 hover:bg-white hover:text-primary dark:border-border/60 dark:bg-card/95 dark:shadow-[0_16px_30px_rgba(0,0,0,0.5)] dark:hover:border-primary/70 dark:hover:bg-card dark:hover:text-primary"
                            onclick={() => {
                                if (!map) return;
                                const currentZoom = map.getZoom?.() ?? 14;
                                map.setZoom(Math.min(20, currentZoom + 1));
                            }}
                            aria-label="放大"
                        >
                            <Plus class="h-5 w-5" />
                        </Button>
                        <Button
                            size="icon"
                            variant="ghost"
                            class="h-11 w-11 rounded-[20px] border border-slate-300/70 bg-white/95 text-foreground shadow-[0_16px_30px_rgba(15,23,42,0.26)] transition hover:border-primary/70 hover:bg-white hover:text-primary dark:border-border/60 dark:bg-card/95 dark:shadow-[0_16px_30px_rgba(0,0,0,0.5)] dark:hover:border-primary/70 dark:hover:bg-card dark:hover:text-primary"
                            onclick={() => {
                                if (!map) return;
                                const currentZoom = map.getZoom?.() ?? 14;
                                map.setZoom(Math.max(1, currentZoom - 1));
                            }}
                            aria-label="缩小"
                        >
                            <Minus class="h-5 w-5" />
                        </Button>
                    </div>
                </div>

                <!-- 移除地图左上角的“地图内容更新中/正在获取当前位置”浮层，避免缩放时闪烁 -->

                {#if showFilterPanel}
                    <div
                        bind:this={filterPanelElement}
                        class="absolute right-3 top-16 z-30 w-40 select-none overflow-hidden rounded-3xl border border-white/60 bg-[linear-gradient(180deg,rgba(255,255,255,0.96),rgba(248,250,252,0.92))] p-2 shadow-[0_24px_60px_rgba(15,23,42,0.18)] backdrop-blur-md dark:border-border/60 dark:bg-[linear-gradient(180deg,rgba(30,30,36,0.96),rgba(24,24,30,0.92))] dark:shadow-[0_24px_60px_rgba(0,0,0,0.5)] sm:right-4 sm:top-16"
                    >
                        <button
                            class={`flex w-full items-center justify-between rounded-2xl px-3 py-2.5 text-left text-sm font-medium transition ${
                                showUsers
                                    ? "border border-blue-500/80 bg-blue-50 text-blue-900 shadow-sm"
                                    : "border border-white/70 bg-white/90 text-slate-400 hover:border-slate-200 hover:bg-slate-50"
                            }`}
                            onclick={() => toggleRenderFilter("user")}
                        >
                            <span class="flex items-center gap-2">
                                <img
                                    src="/map/marker_user.svg"
                                    alt="用户标记"
                                    class="h-8 w-auto"
                                />
                                <span>用户</span>
                            </span>
                            <span
                                class={`h-2.5 w-2.5 rounded-full ${showUsers ? "bg-blue-600" : "bg-slate-300 dark:bg-slate-600"}`}
                            ></span>
                        </button>
                        <button
                            class={`mt-1 flex w-full items-center justify-between rounded-2xl px-3 py-2.5 text-left text-sm font-medium transition ${
                                showGroups
                                    ? "border border-emerald-500/80 bg-emerald-50 text-emerald-900 shadow-sm"
                                    : "border border-white/70 bg-white/90 text-slate-400 hover:border-slate-200 hover:bg-slate-50"
                            }`}
                            onclick={() => toggleRenderFilter("group")}
                        >
                            <span class="flex items-center gap-2">
                                <img
                                    src="/map/marker_group.svg"
                                    alt="群组标记"
                                    class="h-8 w-auto"
                                />
                                <span>群组</span>
                            </span>
                            <span
                                class={`h-2.5 w-2.5 rounded-full ${showGroups ? "bg-emerald-600" : "bg-slate-300 dark:bg-slate-600"}`}
                            ></span>
                        </button>
                        <button
                            class={`mt-1 flex w-full items-center justify-between rounded-2xl px-3 py-2.5 text-left text-sm font-medium transition ${
                                showMoments
                                    ? "border border-orange-500/80 bg-orange-50 text-orange-900 shadow-sm"
                                    : "border border-white/70 bg-white/90 text-slate-400 hover:border-slate-200 hover:bg-slate-50"
                            }`}
                            onclick={() => toggleRenderFilter("moment")}
                        >
                            <span class="flex items-center gap-2">
                                <img
                                    src="/map/marker_moment.svg"
                                    alt="动态标记"
                                    class="h-8 w-auto"
                                />
                                <span>动态</span>
                            </span>
                            <span
                                class={`h-2.5 w-2.5 rounded-full ${showMoments ? "bg-orange-500" : "bg-slate-300 dark:bg-slate-600"}`}
                            ></span>
                        </button>
                        <div
                            class="px-3 pb-1 pt-2 text-[11px] text-muted-foreground"
                        >
                            已启用 {activeFilterCount} 项筛选
                        </div>
                    </div>
                {/if}

                {#if gestureHintVisible || placementHintVisible}
                    <div
                        class="pointer-events-none absolute inset-x-0 top-16 z-30 flex justify-center px-4 sm:top-4"
                    >
                        {#if placementHintVisible}
                            <div
                                class="rounded-2xl border border-border/60 bg-card/95 px-4 py-1.5 text-sm font-medium text-foreground shadow-lg"
                            >
                                {placementHintMessage}
                            </div>
                        {:else if gestureHintVisible}
                            <div
                                class="pointer-events-auto inline-flex items-center gap-3 rounded-2xl border border-dashed border-primary/40 bg-card/90 px-4 py-1.5 text-xs text-muted-foreground shadow"
                            >
                                <span>{gestureHintMessage}</span>
                                <button
                                    type="button"
                                    class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-medium text-primary hover:bg-primary/10"
                                    onclick={dismissGestureHintPermanently}
                                >
                                    <XIcon class="h-3 w-3" />
                                    <span>不再提示</span>
                                </button>
                            </div>
                        {/if}
                    </div>
                {/if}

                <!-- 搜索覆盖层（移动端使用，桌面端由右侧信息栏承载搜索） -->
                {#if showSearch}
                    <div
                        class="absolute inset-x-3 top-3 z-30 overflow-hidden rounded-[28px] border border-white/60 bg-[linear-gradient(180deg,rgba(255,255,255,0.96),rgba(248,250,252,0.92))] shadow-[0_24px_60px_rgba(15,23,42,0.18)] backdrop-blur-md dark:border-border/60 dark:bg-[linear-gradient(180deg,rgba(30,30,36,0.96),rgba(24,24,30,0.92))] dark:shadow-[0_24px_60px_rgba(0,0,0,0.5)] sm:inset-x-auto sm:left-4 sm:right-auto sm:top-4 sm:w-104 sm:hidden"
                    >
                        <div
                            class="border-b border-border/50 bg-[radial-gradient(circle_at_top_left,rgba(59,130,246,0.14),transparent_42%),linear-gradient(180deg,rgba(255,255,255,0.95),rgba(248,250,252,0.88))] px-4 py-3 dark:bg-[radial-gradient(circle_at_top_left,rgba(59,130,246,0.12),transparent_42%),linear-gradient(180deg,rgba(30,30,36,0.95),rgba(24,24,30,0.88))]"
                        >
                            <div class="flex items-start justify-between gap-3">
                                <div>
                                    <div
                                        class="inline-flex items-center gap-1.5 rounded-full border border-primary/15 bg-white/75 px-2.5 py-1 text-[11px] font-semibold uppercase tracking-[0.16em] text-primary/80 dark:bg-card/75"
                                    >
                                        <Search class="h-3.5 w-3.5" />
                                        <span>地点搜索</span>
                                    </div>
                                    <p
                                        class="mt-3 text-base font-bold leading-tight text-foreground"
                                    >
                                        搜索一个地点，或直接回到最近用过的位置
                                    </p>
                                    <p
                                        class="mt-1 text-xs text-muted-foreground"
                                    >
                                        输入地标、商圈、街道名，结果会直接落到地图中心。
                                    </p>
                                </div>
                                <Button
                                    variant="ghost"
                                    size="sm"
                                    class="h-9 w-9 shrink-0 rounded-2xl border border-border/60 bg-white/70 p-0 shadow-sm dark:bg-card/80"
                                    onclick={toggleSearch}
                                >
                                    <X class="h-4 w-4" />
                                    <span class="sr-only">关闭搜索</span>
                                </Button>
                            </div>
                        </div>
                        <!-- 搜索输入框 -->
                        <div class="px-4 py-3">
                            <div
                                class="flex h-12 items-center gap-2 rounded-2xl border border-border/60 bg-white/80 px-3 shadow-sm dark:bg-card/80"
                            >
                                <Search
                                    class="h-4 w-4 shrink-0 text-muted-foreground"
                                />
                                <input
                                    {@attach useKeyboardFocus()}
                                    type="text"
                                    bind:this={searchInput}
                                    bind:value={searchText}
                                    placeholder="搜索地点名称、街道或商圈"
                                    class="flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
                                    oninput={() => {
                                        if (searchDebounceTimer)
                                            clearTimeout(searchDebounceTimer);
                                        if (searchText.trim().length > 0) {
                                            searchDebounceTimer =
                                                window.setTimeout(() => {
                                                    doSearch();
                                                }, 500);
                                        } else {
                                            activeSearchRequestId += 1;
                                            isSearching = false;
                                            searchResults = [];
                                        }
                                    }}
                                    onkeydown={(e) => {
                                        if (e.key === "Enter") {
                                            if (searchDebounceTimer)
                                                clearTimeout(
                                                    searchDebounceTimer,
                                                );
                                            doSearch();
                                        } else if (e.key === "Escape") {
                                            toggleSearch();
                                        }
                                    }}
                                />
                                {#if isSearching}
                                    <Loader2
                                        class="h-4 w-4 shrink-0 animate-spin text-primary"
                                    />
                                {:else if searchText.length > 0}
                                    <button
                                        class="shrink-0 text-muted-foreground hover:text-foreground"
                                        onclick={() => {
                                            if (searchDebounceTimer)
                                                clearTimeout(
                                                    searchDebounceTimer,
                                                );
                                            searchDebounceTimer = null;
                                            activeSearchRequestId += 1;
                                            isSearching = false;
                                            searchText = "";
                                            searchResults = [];
                                            searchInput?.focus();
                                        }}
                                    >
                                        <X class="h-4 w-4" />
                                    </button>
                                {/if}
                            </div>
                        </div>

                        <!-- 搜索结果列表 -->
                        {#if searchResults.length > 0}
                            <div class="max-h-[60vh] overflow-y-auto px-2 pb-2">
                                {#each searchResults as result (`${result.latitude}_${result.longitude}`)}
                                    <button
                                        class="mx-2 mb-2 flex w-[calc(100%-1rem)] items-start gap-3 rounded-2xl border border-border/50 bg-white/72 px-4 py-3 text-left shadow-sm transition-all hover:-translate-y-0.5 hover:border-primary/25 hover:bg-white dark:bg-card/72 dark:hover:bg-card"
                                        onclick={() =>
                                            selectSearchResult(result)}
                                    >
                                        <div
                                            class="flex h-9 w-9 shrink-0 items-center justify-center rounded-2xl bg-primary/10"
                                        >
                                            <MapPin
                                                class="h-4 w-4 text-primary"
                                            />
                                        </div>
                                        <div class="min-w-0 flex-1">
                                            <div
                                                class="truncate text-sm font-medium"
                                            >
                                                {result.name}
                                            </div>
                                            <div
                                                class="mt-1 text-xs text-muted-foreground"
                                            >
                                                {result.address ||
                                                    "未提供详细地址"}
                                            </div>
                                        </div>
                                        <span
                                            class="rounded-full bg-slate-100 px-2 py-1 text-[11px] font-medium text-slate-600 dark:bg-muted dark:text-muted-foreground"
                                            >定位</span
                                        >
                                    </button>
                                {/each}
                            </div>
                        {:else if searchText.trim().length === 0 && anchorHistory.length > 0}
                            <!-- 无输入时展示历史记录 -->
                            <div class="max-h-[60vh] overflow-y-auto px-2 pb-2">
                                <div
                                    class="flex items-center justify-between px-4 py-2"
                                >
                                    <div
                                        class="flex items-center gap-1.5 text-xs font-medium text-muted-foreground"
                                    >
                                        <Clock class="h-3.5 w-3.5" />
                                        <span>最近使用</span>
                                    </div>
                                    <button
                                        class="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
                                        onclick={handleClearHistory}
                                    >
                                        <Trash2 class="h-3 w-3" />
                                        <span>清空</span>
                                    </button>
                                </div>
                                {#each anchorHistory as item (item.timestamp)}
                                    <button
                                        class="mx-2 mb-2 flex w-[calc(100%-1rem)] items-center gap-3 rounded-2xl border border-border/50 bg-white/72 px-4 py-3 text-left shadow-sm transition-all hover:-translate-y-0.5 hover:border-primary/20 hover:bg-white dark:bg-card/72 dark:hover:bg-card"
                                        onclick={() => {
                                            selectHistoryItem(item);
                                            showSearch = false;
                                        }}
                                    >
                                        <div
                                            class="flex h-9 w-9 shrink-0 items-center justify-center rounded-2xl bg-slate-100 dark:bg-muted"
                                        >
                                            <Clock
                                                class="h-4 w-4 text-muted-foreground"
                                            />
                                        </div>
                                        <div class="min-w-0 flex-1">
                                            <div
                                                class="truncate text-sm font-medium"
                                            >
                                                {item.locationName}
                                            </div>
                                            <div
                                                class="mt-1 text-[11px] text-muted-foreground"
                                            >
                                                一键回到这个位置继续浏览
                                            </div>
                                        </div>
                                        <span
                                            class="rounded-full bg-slate-100 px-2 py-1 text-[11px] font-medium text-slate-600 dark:bg-muted dark:text-muted-foreground"
                                            >最近</span
                                        >
                                    </button>
                                {/each}
                            </div>
                        {:else if searchText.trim().length > 0 && !isSearching}
                            <div class="px-4 pb-4 pt-1">
                                <div
                                    class="rounded-2xl border border-dashed border-border/70 bg-white/60 px-4 py-6 text-center shadow-sm dark:bg-card/60"
                                >
                                    <p
                                        class="text-sm font-semibold text-foreground"
                                    >
                                        没有找到匹配地点
                                    </p>
                                    <p
                                        class="mt-1 text-xs text-muted-foreground"
                                    >
                                        换个关键词试试，或者直接拖动地图到目标区域。
                                    </p>
                                </div>
                            </div>
                        {/if}
                    </div>
                {/if}

                {#if !isMapAvailable}
                    <div
                        class="absolute inset-0 z-20 flex items-center justify-center bg-background/80 backdrop-blur-sm"
                    >
                        <div class="text-center space-y-3">
                            <p class="font-medium">地图未配置</p>
                            <p class="text-sm text-muted-foreground">
                                请设置 VITE_AMAP_KEY
                            </p>
                            <Button
                                size="sm"
                                variant="outline"
                                onclick={handleRetryMapLoad}
                            >
                                重试加载
                            </Button>
                        </div>
                    </div>
                {/if}

                {#if isLoadingMap}
                    <div
                        class="absolute inset-0 z-20 flex items-center justify-center bg-background/70 backdrop-blur-sm"
                    >
                        <Loader2 class="h-8 w-8 animate-spin text-primary" />
                    </div>
                {/if}

                <!-- 悬浮定位按钮 (移动端) -->
                <div
                    class="absolute right-3 top-3 z-10 flex flex-col gap-2 sm:hidden"
                >
                    <Button
                        size="icon"
                        variant="ghost"
                        class="h-11 w-11 rounded-2xl border border-slate-300/70 bg-white/95 text-foreground shadow-[0_16px_30px_rgba(15,23,42,0.26)] transition hover:border-primary/70 hover:bg-white hover:text-primary dark:border-border/60 dark:bg-card/95 dark:shadow-[0_16px_30px_rgba(0,0,0,0.5)] dark:hover:border-primary/70 dark:hover:bg-card dark:hover:text-primary"
                        onclick={handleUseCurrentLocation}
                        disabled={isLocating}
                        aria-label="使用当前位置"
                    >
                        {#if isLocating}
                            <Loader2 class="h-5 w-5 animate-spin" />
                        {:else}
                            <LocateFixed
                                class="h-7 w-7 text-blue-600 dark:text-blue-400"
                            />
                        {/if}
                    </Button>
                    {#if isAnchorMode}
                        <Button
                            size="icon"
                            variant="ghost"
                            class="h-11 w-11 rounded-2xl border border-slate-300/70 bg-white/95 text-foreground shadow-[0_16px_30px_rgba(15,23,42,0.26)] transition hover:border-primary/70 hover:bg-white hover:text-primary dark:border-border/60 dark:bg-card/95 dark:shadow-[0_16px_30px_rgba(0,0,0,0.5)] dark:hover:border-primary/70 dark:hover:bg-card dark:hover:text-primary"
                            onclick={handleBackToBrowseAnchor}
                            aria-label="回到浏览锚点位置"
                        >
                            <img
                                src="/map/marker_center_pin.svg"
                                alt="浏览锚点"
                                class="h-7 w-7"
                            />
                        </Button>
                    {/if}
                    <Button
                        size="icon"
                        variant="ghost"
                        class="h-11 w-11 rounded-2xl border border-slate-300/70 bg-white/95 text-foreground shadow-[0_16px_30px_rgba(15,23,42,0.26)] transition hover:border-primary/70 hover:bg-white hover:text-primary dark:border-border/60 dark:bg-card/95 dark:shadow-[0_16px_30px_rgba(0,0,0,0.5)] dark:hover:border-primary/70 dark:hover:bg-card dark:hover:text-primary"
                        onclick={() => {
                            if (!map) return;
                            const currentZoom = map.getZoom?.() ?? 14;
                            map.setZoom(Math.min(20, currentZoom + 1));
                        }}
                        aria-label="放大"
                    >
                        <Plus class="h-5 w-5" />
                    </Button>
                    <Button
                        size="icon"
                        variant="ghost"
                        class="h-11 w-11 rounded-2xl border border-slate-300/70 bg-white/95 text-foreground shadow-[0_16px_30px_rgba(15,23,42,0.26)] transition hover:border-primary/70 hover:bg-white hover:text-primary dark:border-border/60 dark:bg-card/95 dark:shadow-[0_16px_30px_rgba(0,0,0,0.5)] dark:hover:border-primary/70 dark:hover:bg-card dark:hover:text-primary"
                        onclick={() => {
                            if (!map) return;
                            const currentZoom = map.getZoom?.() ?? 14;
                            map.setZoom(Math.max(1, currentZoom - 1));
                        }}
                        aria-label="缩小"
                    >
                        <Minus class="h-5 w-5" />
                    </Button>
                </div>

                {#if clusterPreview}
                    <div
                        class="absolute inset-x-3 bottom-32 z-30 select-none sm:bottom-6 sm:left-4 sm:right-auto sm:w-100"
                    >
                        <div
                            class="overflow-hidden rounded-[28px] border border-white/60 bg-[linear-gradient(180deg,rgba(255,255,255,0.96),rgba(248,250,252,0.92))] shadow-[0_24px_60px_rgba(15,23,42,0.18)] backdrop-blur-md dark:border-border/60 dark:bg-[linear-gradient(180deg,rgba(30,30,36,0.96),rgba(24,24,30,0.92))] dark:shadow-[0_24px_60px_rgba(0,0,0,0.5)]"
                        >
                            <div
                                class="flex items-start justify-between gap-3 border-b border-border/50 px-4 py-3"
                            >
                                <div class="min-w-0">
                                    <p
                                        class="text-[11px] font-semibold uppercase tracking-[0.16em] text-primary/70"
                                    >
                                        {clusterPreview.kind === "overlap_group"
                                            ? "同点内容预览"
                                            : "聚合内容预览"}
                                    </p>
                                    <p
                                        class="truncate text-base font-bold text-foreground"
                                    >
                                        {clusterPreview.title}
                                    </p>
                                    <p
                                        class="mt-1 text-xs text-muted-foreground"
                                    >
                                        {clusterPreview.description}
                                    </p>
                                </div>
                                <Button
                                    variant="ghost"
                                    size="sm"
                                    class="h-9 w-9 shrink-0 rounded-2xl border border-border/60 bg-white/70 p-0 shadow-sm dark:bg-card/80"
                                    onclick={closeClusterPreview}
                                >
                                    <X class="h-4 w-4" />
                                    <span class="sr-only">关闭预览</span>
                                </Button>
                            </div>

                            {#if clusterPreview.items.length > 0}
                                <div
                                    class="max-h-64 space-y-2 overflow-y-auto px-3 py-3"
                                >
                                    {#each clusterPreview.items as previewItem (`preview_${previewItem.location.latitude}_${previewItem.location.longitude}_${previewItem.total_count}`)}
                                        <button
                                            class="flex w-full items-start gap-3 rounded-2xl border border-border/50 bg-white/78 px-3 py-3 text-left shadow-sm transition hover:-translate-y-0.5 hover:border-primary/25 hover:bg-white dark:bg-card/78 dark:hover:bg-card"
                                            onclick={() =>
                                                focusPreviewEntity(previewItem)}
                                        >
                                            <div
                                                class={`rounded-full px-2 py-1 text-[11px] font-semibold ${getItemBadgeClass(previewItem)}`}
                                            >
                                                {getItemTypeLabel(previewItem)}
                                            </div>
                                            <div class="min-w-0 flex-1">
								<p
									class="truncate text-sm font-semibold text-foreground"
								>
									聚合区域
								</p>
                                                <p
                                                    class="mt-1 line-clamp-2 text-xs text-muted-foreground"
                                                >
                                                    {previewItem.total_count} 个内容
                                                </p>
                                            </div>
                                        </button>
                                    {/each}
                                </div>
                            {:else}
                                <div
                                    class="px-4 py-4 text-sm text-muted-foreground"
                                >
                                    当前层级只有聚合摘要，继续放大后才能看到具体对象列表。
                                </div>
                            {/if}

                            <div
                                class="flex gap-2 border-t border-border/50 px-4 py-3"
                            >
                                <Button
                                    variant="outline"
                                    class="h-10 flex-1 rounded-2xl"
                                    onclick={closeClusterPreview}
                                >
                                    稍后再看
                                </Button>
                                <Button
                                    class="h-10 flex-1 rounded-2xl"
                                    onclick={zoomToClusterPreview}
                                >
                                    {clusterPreview.kind === "overlap_group"
                                        ? "放大展开"
                                        : "继续放大"}
                                </Button>
                            </div>
                        </div>
                    </div>
                {/if}

                {#if hotspotPreviewItems.length > 0}
                    <div
                        class="absolute inset-x-3 bottom-32 z-30 select-none sm:bottom-6 sm:left-1/2 sm:right-auto sm:w-[420px] sm:-translate-x-1/2"
                    >
                        <div
                            class="overflow-hidden rounded-[28px] border border-white/60 bg-[linear-gradient(180deg,rgba(255,255,255,0.96),rgba(248,250,252,0.92))] shadow-[0_24px_60px_rgba(15,23,42,0.18)] backdrop-blur-md dark:border-border/60 dark:bg-[linear-gradient(180deg,rgba(30,30,36,0.96),rgba(24,24,30,0.92))] dark:shadow-[0_24px_60px_rgba(0,0,0,0.5)]"
                        >
                            <div
                                class="flex items-start justify-between gap-3 border-b border-border/50 px-4 py-3"
                            >
                                <div class="min-w-0">
                                    <p
                                        class="text-[11px] font-semibold uppercase tracking-[0.16em] text-primary/70"
                                    >
                                        地图对象详情
                                    </p>
                                    <p
                                        class="truncate text-base font-bold text-foreground"
                                    >
                                        {hotspotPreviewItems.length === 1
                                            ? (hotspotPreviewItems[0]?.title ??
                                              "")
                                            : `${hotspotPreviewItems.length} 个内容`}
                                    </p>
                                </div>
                                <Button
                                    variant="ghost"
                                    size="sm"
                                    class="h-9 w-9 shrink-0 rounded-2xl border border-border/60 bg-white/70 p-0 shadow-sm dark:bg-card/80"
                                    onclick={() => (hotspotPreviewItems = [])}
                                >
                                    <X class="h-4 w-4" />
                                    <span class="sr-only">关闭详情</span>
                                </Button>
                            </div>

                            <div
                                class="max-h-72 space-y-2 overflow-y-auto px-3 py-3"
                            >
                                {#each hotspotPreviewItems as item (item.object_type + "_" + item.object_id)}
                                    <button
                                        class="flex w-full items-start gap-3 rounded-2xl border border-border/50 bg-white/78 px-3 py-3 text-left shadow-sm transition hover:-translate-y-0.5 hover:border-primary/25 hover:bg-white dark:bg-card/78 dark:hover:bg-card"
                                        onclick={() => {
                                            hotspotPreviewItems = [];
                                            if (item.object_type === "user") {
                                                goto(
                                                    `/users/${item.object_id}`,
                                                );
                                            } else if (
                                                item.object_type === "group"
                                            ) {
                                                goto(
                                                    `/groups/${item.object_id}`,
                                                );
                                            } else if (
                                                item.object_type === "moment"
                                            ) {
                                                goto(
                                                    `/moments/${item.object_id}`,
                                                );
                                            }
                                        }}
                                    >
                                        <div
                                            class="rounded-full px-2 py-1 text-[11px] font-semibold"
                                            class:text-blue-700={item.object_type ===
                                                "user"}
                                            class:bg-blue-50={item.object_type ===
                                                "user"}
                                            class:text-emerald-700={item.object_type ===
                                                "group"}
                                            class:bg-emerald-50={item.object_type ===
                                                "group"}
                                            class:text-orange-700={item.object_type ===
                                                "moment"}
                                            class:bg-orange-50={item.object_type ===
                                                "moment"}
                                        >
                                            {item.object_type === "user"
                                                ? "用户"
                                                : item.object_type === "group"
                                                  ? "群组"
                                                  : "动态"}
                                        </div>
                                        <div class="min-w-0 flex-1">
                                            <p
                                                class="truncate text-sm font-semibold text-foreground"
                                            >
                                                {item.title ?? ""}
                                            </p>
                                            {#if item.subtitle}
                                                <p
                                                    class="mt-1 line-clamp-2 text-xs text-muted-foreground"
                                                >
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
                    style="transform: translateY({isBottomSheetExpanded
                        ? '0'
                        : 'calc(100% - 116px)'})"
                >
                    <div
                        class="rounded-t-[28px] border border-border/60 bg-card/95 shadow-2xl backdrop-blur-md"
                    >
                        <!-- 展开/收起手柄 -->
                        <button
                            class="flex w-full flex-col items-center gap-1 py-3 active:bg-muted/50"
                            onclick={() =>
                                (isBottomSheetExpanded =
                                    !isBottomSheetExpanded)}
                        >
                            <div
                                class="h-1 w-12 rounded-full bg-muted-foreground/20"
                            ></div>
                            {#if isBottomSheetExpanded}
                                <ChevronDown
                                    class="h-4 w-4 text-muted-foreground"
                                />
                            {:else}
                                <ChevronUp
                                    class="h-4 w-4 text-muted-foreground"
                                />
                            {/if}
                        </button>

                        <!-- 底部抽屉内容 -->
                        <div class="max-h-[60vh] overflow-y-auto px-4 pb-safe">
                            <!-- 位置信息 (精简版 - 始终可见) -->
                            <AnchorSummaryCard
                                label="当前选中"
                                description={selectionDescription}
                                subtle
                                overrideName={locationName ||
                                    "拖动地图选择位置"}
                            />

                            <!-- 操作按钮 -->
                            <div
                                class="sticky bottom-0 space-y-2 bg-[linear-gradient(180deg,rgba(255,255,255,0),rgba(255,255,255,0.92)_18%,rgba(255,255,255,0.98)_100%)] pb-4 pt-3 backdrop-blur-sm dark:bg-[linear-gradient(180deg,rgba(24,24,30,0),rgba(24,24,30,0.92)_18%,rgba(24,24,30,0.98)_100%)]"
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
                                        分享此位置
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
                class="hidden h-full w-[320px] shrink-0 select-none flex-col gap-4 overflow-hidden rounded-3xl border border-border/60 bg-card/95 p-4 shadow-lg sm:flex lg:w-90"
            >
                <div class="flex items-start justify-between gap-3">
                    <div class="min-w-0">
                        <p
                            class="text-[11px] font-semibold uppercase tracking-[0.16em] text-primary/70"
                        >
                            {mapModeLabel}
                        </p>
                        <p class="mt-1 truncate text-base font-bold">
                            {modalTitle}
                        </p>
                    </div>
                    <div class="flex items-center gap-2">
                        <Button
                            variant="ghost"
                            size="icon"
                            class="h-9 w-9 rounded-2xl border border-border/60 bg-background/80 p-0 shadow-sm"
                            onclick={() => (isFullscreen = !isFullscreen)}
                            aria-pressed={isFullscreen}
                            aria-label={isFullscreen ? "退出全屏" : "进入全屏"}
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
                        class="flex h-12 items-center gap-2 rounded-2xl border border-border/60 bg-background/80 px-3 text-sm shadow-sm"
                    >
                        <Search
                            class="h-4 w-4 shrink-0 text-muted-foreground"
                        />
                        <input
                            {@attach useKeyboardFocus()}
                            type="text"
                            bind:this={searchInput}
                            bind:value={searchText}
                            placeholder="搜索地点、地标或坐标"
                            class="flex-1 select-text bg-transparent text-sm outline-none placeholder:text-muted-foreground"
                            oninput={() => {
                                if (searchDebounceTimer)
                                    clearTimeout(searchDebounceTimer);
                                if (searchText.trim().length > 0) {
                                    searchDebounceTimer = window.setTimeout(
                                        () => {
                                            doSearch();
                                        },
                                        500,
                                    );
                                } else {
                                    activeSearchRequestId += 1;
                                    isSearching = false;
                                    searchResults = [];
                                }
                            }}
                            onkeydown={(e) => {
                                if (e.key === "Enter") {
                                    if (searchDebounceTimer)
                                        clearTimeout(searchDebounceTimer);
                                    doSearch();
                                }
                            }}
                        />
                        {#if isSearching}
                            <Loader2
                                class="h-4 w-4 shrink-0 animate-spin text-primary"
                            />
                        {:else if searchText.length > 0}
                            <button
                                class="shrink-0 text-muted-foreground hover:text-foreground"
                                onclick={() => {
                                    if (searchDebounceTimer)
                                        clearTimeout(searchDebounceTimer);
                                    searchDebounceTimer = null;
                                    activeSearchRequestId += 1;
                                    isSearching = false;
                                    searchText = "";
                                    searchResults = [];
                                    searchInput?.focus();
                                }}
                            >
                                <X class="h-4 w-4" />
                            </button>
                        {/if}
                    </div>

                    <div class="max-h-[60vh] space-y-2 overflow-y-auto pr-1">
                        {#if searchResults.length > 0}
                            {#each searchResults as result (`desktop_${result.latitude}_${result.longitude}`)}
                                <button
                                    class="flex w-full items-start gap-3 rounded-2xl border border-border/50 bg-card/80 px-3 py-2 text-left text-xs shadow-sm transition-all hover:-translate-y-0.5 hover:border-primary/25 hover:bg-card"
                                    onclick={() => selectSearchResult(result)}
                                >
                                    <div
                                        class="flex h-8 w-8 shrink-0 items-center justify-center rounded-2xl bg-primary/10"
                                    >
                                        <MapPin class="h-4 w-4 text-primary" />
                                    </div>
                                    <div class="min-w-0 flex-1">
                                        <p class="truncate text-sm font-medium">
                                            {result.name}
                                        </p>
                                        <p
                                            class="mt-0.5 text-[11px] text-muted-foreground"
                                        >
                                            {result.address || "未提供详细地址"}
                                        </p>
                                    </div>
                                </button>
                            {/each}
                        {:else if searchText.trim().length === 0 && anchorHistory.length > 0}
                            <div
                                class="flex items-center justify-between px-1 pb-1"
                            >
                                <div
                                    class="flex items-center gap-1.5 text-[11px] font-medium text-muted-foreground"
                                >
                                    <Clock class="h-3.5 w-3.5" />
                                    <span>最近使用</span>
                                </div>
                                <button
                                    class="flex items-center gap-1 text-[11px] text-muted-foreground hover:text-foreground"
                                    onclick={handleClearHistory}
                                >
                                    <Trash2 class="h-3 w-3" />
                                    <span>清空</span>
                                </button>
                            </div>
                            {#each anchorHistory as item (item.timestamp)}
                                <button
                                    class="flex w-full items-center gap-3 rounded-2xl border border-border/50 bg-card/80 px-3 py-2 text-left text-xs shadow-sm transition-all hover:-translate-y-0.5 hover:border-primary/20 hover:bg-card"
                                    onclick={() => selectHistoryItem(item)}
                                >
                                    <div
                                        class="flex h-8 w-8 shrink-0 items-center justify-center rounded-2xl bg-muted"
                                    >
                                        <Clock
                                            class="h-4 w-4 text-muted-foreground"
                                        />
                                    </div>
                                    <div class="min-w-0 flex-1">
                                        <p class="truncate text-sm font-medium">
                                            {item.locationName}
                                        </p>
                                        <p
                                            class="mt-0.5 text-[11px] text-muted-foreground"
                                        >
                                            一键回到该位置继续浏览
                                        </p>
                                    </div>
                                </button>
                            {/each}
                        {:else if searchText.trim().length > 0 && !isSearching}
                            <div
                                class="rounded-2xl border border-dashed border-border/70 bg-background/60 px-3 py-4 text-[11px] text-muted-foreground"
                            >
                                没有找到匹配地点，换个关键词试试，或者直接拖动地图到目标区域。
                            </div>
                        {/if}
                    </div>
                </div>

                <!-- 当前位置摘要与操作 -->
                <div class="mt-auto space-y-3 border-t border-border/60 pt-3">
                    <div class="space-y-1 text-xs text-muted-foreground">
                        <p class="font-medium text-foreground">当前选中</p>
                        <p class="truncate text-sm font-semibold">
                            {locationName || "拖动地图选择位置"}
                        </p>
                    </div>

                    <div class="space-y-2">
                        <Button
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
                                分享此位置
                            </Button>
                        {/if}
                    </div>
                </div>
            </div>
        </Dialog.Content>
    </Dialog.Root>
</div>

<!-- 位置分享模态框 -->
<CreateLocationShareModal
    bind:open={showShareModal}
    {latitude}
    {longitude}
    {locationName}
/>

<style>
    :global(.map-render-marker) {
        display: flex;
        min-width: 4.4rem;
        max-width: 10rem;
        flex-direction: column;
        align-items: flex-start;
        gap: 0.22rem;
        border: 1px solid transparent;
        border-radius: 1.15rem;
        padding: 0.5rem 0.72rem 0.56rem;
        font-size: 0.72rem;
        font-weight: 700;
        line-height: 1.1;
        box-shadow: 0 14px 30px rgba(15, 23, 42, 0.2);
        cursor: pointer;
        white-space: normal;
        text-align: left;
        backdrop-filter: blur(10px);
        user-select: none;
        -webkit-user-select: none;
        -moz-user-select: none;
    }

    :global(.map-render-marker::before) {
        content: "";
        display: block;
        width: 24px;
        height: 32px;
        background-repeat: no-repeat;
        background-position: left bottom;
        background-size: contain;
    }

    :global(.map-render-marker-badge) {
        display: inline-flex;
        align-items: center;
        border-radius: 999px;
        padding: 0.2rem 0.46rem;
        font-size: 0.58rem;
        font-weight: 800;
        letter-spacing: 0.08em;
        text-transform: uppercase;
    }

    :global(.map-render-marker-title) {
        display: -webkit-box;
        overflow: hidden;
        max-width: 100%;
        font-size: 0.76rem;
        font-weight: 800;
        line-height: 1.15;
        line-clamp: 2;
        -webkit-box-orient: vertical;
        -webkit-line-clamp: 2;
    }

    :global(.map-render-marker-user) {
        background: rgba(255, 255, 255, 0.96);
        color: rgb(15, 23, 42);
        border: 1px solid rgba(148, 163, 184, 0.45);
    }

    :global(.map-render-marker-user .map-render-marker-badge) {
        background: rgba(15, 23, 42, 0.08);
        color: rgba(15, 23, 42, 0.72);
    }

    :global(.map-render-marker-user::before) {
        background-image: url("/map/marker_user_cluster.svg");
    }

    :global(.map-render-marker-group) {
        background: linear-gradient(
            180deg,
            rgba(14, 165, 233, 0.98),
            rgba(2, 132, 199, 0.98)
        );
        color: white;
    }

    :global(.map-render-marker-group .map-render-marker-badge) {
        background: rgba(255, 255, 255, 0.18);
        color: rgba(255, 255, 255, 0.82);
    }

    :global(.map-render-marker-group::before) {
        background-image: url("/map/marker_group_cluster.svg");
    }

    :global(.map-render-marker-moment) {
        background: linear-gradient(
            180deg,
            rgba(249, 115, 22, 0.98),
            rgba(234, 88, 12, 0.98)
        );
        color: white;
    }

    :global(.map-render-marker-moment .map-render-marker-badge) {
        background: rgba(255, 255, 255, 0.18);
        color: rgba(255, 255, 255, 0.82);
    }

    :global(.map-render-marker-moment::before) {
        background-image: url("/map/marker_moment_cluster.svg");
    }

    :global(.map-render-marker-cluster) {
        background: linear-gradient(
            180deg,
            rgba(15, 23, 42, 0.96),
            rgba(30, 41, 59, 0.96)
        );
        color: white;
    }

    :global(.map-render-marker-cluster .map-render-marker-badge) {
        background: rgba(96, 165, 250, 0.2);
        color: rgba(191, 219, 254, 0.95);
    }

    :global(.map-render-marker-cluster::before) {
        background-image: url("/map/marker_mixed_cluster.svg");
    }
</style>
