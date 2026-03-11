<script lang="ts">
    import { onMount, tick } from "svelte";
    import { assertSuccess } from "$lib/api/client";
    import {
        mapService,
        groupOverlappingRenderItems,
        getOverlapClusterExpandZoom,
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
    import type { MapRenderItem, MapViewport } from "$lib/types/api";
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
    const REVERSE_GEOCODE_IDLE_MS = 800;
    const MIN_LOCATION_DELTA = 0.00005;
    let lastResolvedLat = mapAnchorStore.latitude;
    let lastResolvedLng = mapAnchorStore.longitude;
    let reverseGeocodeTimer: number | null = null;
    let renderMapTimer: number | null = null;
    let skipNextReverseGeocode = false;
    let mapMoveFrame: number | null = null;
    let isLoadingRenderItems = $state(false);
    let activeRenderRequestId = 0;
    let lastRenderedViewportSignature = "";
    let lastToastMessage = "";
    let renderMarkers: AMap.Marker[] = [];
    let selectedMarker: AMap.Marker | null = null;
    let hasInit = $state(false);
    let hasSyncedFromAnchor = $state(false);
    const LONG_PRESS_DURATION_MS = 700;
    const GESTURE_HINT_DURATION_MS = 4000;
    const MAP_RENDER_IDLE_MS = 250;
    const GESTURE_HINT_COMBINED = "双击、右键或长按地图可快速定位";
    const GESTURE_HINT_TOUCH = "长按地图可快速定位";
    const GESTURE_HINT_POINTER = "双击或右键地图可快速定位";

    type RenderMarkerKind = "cluster" | "user" | "group" | "moment";
    type ClusterPreviewKind = "server_cluster" | "overlap_group";

    interface ClusterPreviewState {
        kind: ClusterPreviewKind;
        renderId: string;
        title: string;
        description: string;
        count: number;
        latitude: number;
        longitude: number;
        zoomTarget: number;
        items: MapRenderItem[];
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
    let latestRenderItems = $state.raw<MapRenderItem[]>([]);
    let overlapGroupItems = $state.raw<Map<string, MapRenderItem[]>>(new Map());
    let clusterPreview = $state<ClusterPreviewState | null>(null);

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

    const isAnchorMode = $derived(mode === "anchor");
    const modalTitle = $derived(isAnchorMode ? "设置浏览锚点" : "选择位置");
    const confirmButtonText = $derived(
        isAnchorMode ? "设为浏览锚点" : "使用此位置",
    );
    const confirmSuccessText = $derived(
        isAnchorMode ? "浏览锚点已更新" : "位置已选定",
    );
    const selectionDescription = $derived(
        isAnchorMode
            ? "拖动地图或搜索后，确认才会覆盖锚点"
            : "拖动地图或搜索后，确认用于当前操作",
    );
    const hintAfterConfirm = $derived(
        isAnchorMode
            ? `${gestureHintMessage}，确认后才会更新锚点`
            : `${gestureHintMessage}，确认后将用于当前操作`,
    );
    const mapModeLabel = $derived(isAnchorMode ? "浏览锚点" : "位置选择");
    const mapStatusText = $derived.by(() => {
        if (isLoadingRenderItems) return "地图内容更新中";
        if (isLocating) return "正在获取当前位置";
        return "拖动地图可继续探索";
    });
    const locationReadyText = $derived(
        locationName
            ? "位置已锁定，可直接确认或分享"
            : "还没有精确地点名称，继续拖动或搜索",
    );
    const activeFilterCount = $derived(
        [showUsers, showGroups, showMoments].filter(Boolean).length,
    );

    type MapClickEvent = {
        lnglat?: AMap.LngLat;
    };

    const PLACEMENT_HINT_DEFAULT = "已定位到该位置";

    function showGestureHint(message = getGestureHintMessage()) {
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
        for (const marker of renderMarkers) {
            marker.setMap(null);
            marker.remove();
        }
        renderMarkers = [];
    }

	function createSelectionMarkerContent() {
		const container = document.createElement("div");
		container.className = "map-selection-marker";
		container.setAttribute("aria-hidden", "true");

		const img = document.createElement("img");
		// 与 Harmony 端保持一致：使用独立的中心定位图标
		img.src = "/map/marker_center_pin.svg";
		img.alt = "选择位置";
        img.className = "map-selection-marker-svg";

        container.append(img);
        return container;
    }

    function ensureSelectedMarker() {
        if (!map || selectedMarker) return;
        selectedMarker = new AMap.Marker({
            position: [longitude, latitude],
            content: createSelectionMarkerContent(),
            anchor: "bottom-center",
            offset: [0, 6],
            clickable: false,
            map,
        });
    }

	function updateSelectedMarker() {
		if (!map) return;
		ensureSelectedMarker();
		selectedMarker?.setPosition([longitude, latitude]);
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
        return {
            bounds: visibleRegion.bounds,
            zoom: Math.max(1, Math.min(20, visibleRegion.zoom)),
        };
    }

    function getMarkerClass(kind: RenderMarkerKind) {
        switch (kind) {
            case "cluster":
                return "map-render-marker map-render-marker-cluster";
            case "group":
                return "map-render-marker map-render-marker-group";
            case "moment":
                return "map-render-marker map-render-marker-moment";
            case "user":
            default:
                return "map-render-marker map-render-marker-user";
        }
    }

    function shouldDisplayItem(item: MapRenderItem) {
        switch (item.type) {
            case "user":
                return showUsers;
            case "group":
                return showGroups;
            case "moment":
                return showMoments;
            case "cluster":
                return showUsers || showGroups || showMoments;
            default:
                return true;
        }
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
        renderMapItems(
            latestRenderItems.filter((item) => shouldDisplayItem(item)),
        );
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

    function getMarkerText(item: MapRenderItem) {
        if (item.type === "cluster") {
            return item.cluster?.label?.trim() || `${item.cluster?.count ?? 0}`;
        }
        return item.entity?.title?.trim() || item.type;
    }

    function getMarkerBadgeText(item: MapRenderItem) {
        switch (item.type) {
            case "cluster":
                return `${item.cluster?.count ?? 0} 个点`;
            case "group":
                return "群组";
            case "moment":
                return "动态";
            case "user":
            default:
                return "用户";
        }
    }

    function getItemTypeLabel(item: MapRenderItem) {
        switch (item.type) {
            case "group":
                return "群组";
            case "moment":
                return "动态";
            case "user":
                return "用户";
            case "cluster":
            default:
                return "内容";
        }
    }

    function getItemBadgeClass(item: MapRenderItem) {
        switch (item.type) {
            case "group":
                return "bg-sky-100 text-sky-700";
            case "moment":
                return "bg-orange-100 text-orange-700";
            case "user":
            default:
                return "bg-slate-100 text-slate-700";
        }
    }

    function buildOverlapPreviewDescription(items: MapRenderItem[]) {
        const typeCounts = new Map<string, number>();
        for (const item of items) {
            typeCounts.set(item.type, (typeCounts.get(item.type) ?? 0) + 1);
        }
        return Array.from(typeCounts.entries())
            .map(
                ([type, count]) =>
                    `${count} 个${getItemTypeLabel({ type } as MapRenderItem)}`,
            )
            .join("，");
    }

    function openClusterPreview(item: MapRenderItem) {
        if (item.type !== "cluster") {
            return;
        }
        const overlapItems = overlapGroupItems.get(item.render_id) ?? [];
        const isOverlapGroup =
            isOverlapGroupRenderId(item.render_id) && overlapItems.length > 0;
        const currentZoom = map?.getZoom() ?? 14;
        clusterPreview = {
            kind: isOverlapGroup ? "overlap_group" : "server_cluster",
            renderId: item.render_id,
            title:
                item.cluster?.label?.trim() ||
                `${item.cluster?.count ?? overlapItems.length} 个内容`,
            description: isOverlapGroup
                ? `${buildOverlapPreviewDescription(overlapItems)}。继续放大后会拆成单独 marker。`
                : "当前层级仍在聚合显示，继续放大后会拆成具体对象。",
            count: item.cluster?.count ?? overlapItems.length,
            latitude: item.location.latitude,
            longitude: item.location.longitude,
            zoomTarget: isOverlapGroup
                ? Math.max(currentZoom + 1, getOverlapClusterExpandZoom())
                : (item.cluster?.count ?? 0) > 100
                  ? Math.min(20, currentZoom + 2)
                  : Math.min(20, currentZoom + 3),
            items: overlapItems,
        };
    }

	function zoomToClusterPreview() {
		if (!clusterPreview || !map) {
			return;
		}
		map.setCenter([clusterPreview.longitude, clusterPreview.latitude]);
        map.setZoom(clusterPreview.zoomTarget);
        showPlacementHint(
            clusterPreview.kind === "overlap_group"
                ? "已放大到同点内容展开层级"
                : "已放大到更细地图层级",
        );
        closeClusterPreview();
    }

    function focusPreviewEntity(item: MapRenderItem) {
        closeClusterPreview();
        focusRenderItem(item);
    }

    function createMarkerContent(item: MapRenderItem, onClick: () => void) {
        const element = document.createElement("button");
        element.type = "button";
        element.className = getMarkerClass(item.type);

        const badge = document.createElement("span");
        badge.className = "map-render-marker-badge";
        badge.textContent = getMarkerBadgeText(item);

        const title = document.createElement("span");
        title.className = "map-render-marker-title";
        title.textContent = getMarkerText(item);

        element.append(badge, title);
        element.addEventListener("click", (event) => {
            event.preventDefault();
            event.stopPropagation();
            onClick();
        });
        return element;
    }

    function focusRenderItem(item: MapRenderItem) {
        showSearch = false;
        showFilterPanel = false;
		if (item.type === "cluster") {
			openClusterPreview(item);
			map?.setCenter([item.location.longitude, item.location.latitude]);
            showPlacementHint(
                item.cluster?.label?.trim() ||
                    `已聚焦 ${item.cluster?.count ?? 0} 个内容`,
            );
            return;
        }
        closeClusterPreview();
		const nextLat = item.location.latitude;
		const nextLng = item.location.longitude;
		map?.setCenter([nextLng, nextLat]);
		latitude = nextLat;
		longitude = nextLng;
        skipNextReverseGeocode = true;
        const title = item.entity?.title?.trim();
        if (title) {
            locationName = title;
            lastResolvedLat = nextLat;
            lastResolvedLng = nextLng;
            showPlacementHint(`已定位到 ${title}`);
            return;
        }
        showPlacementHint();
    }

	function renderMapItems(items: MapRenderItem[]) {
		if (!map) return;
        clearRenderMarkers();
        const { merged, groups } = groupOverlappingRenderItems(
            items,
            map.getZoom(),
        );
        overlapGroupItems = groups;
        const currentCluster = clusterPreview;
        if (
            currentCluster &&
            !items.some((item) => item.render_id === currentCluster.renderId)
        ) {
            closeClusterPreview();
        }
		renderMarkers = merged.map(({ item, offset }) => {
			const content = createMarkerContent(item, () =>
				focusRenderItem(item),
			);
			return new AMap.Marker({
				position: [item.location.longitude, item.location.latitude],
				content,
				anchor: "bottom-center",
				offset,
				clickable: true,
				map: map ?? undefined,
			});
		});
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
        isLoadingRenderItems = true;
        try {
            const response = await mapService.render(
                filter ? { viewport, filter } : { viewport },
            );
            assertSuccess(response);
            if (requestId !== activeRenderRequestId) {
                return;
            }
            lastRenderedViewportSignature = signature;
            latestRenderItems = response.data.items ?? [];
            applyCurrentFilter();
            const nextToastMessage = response.data.toast_message?.trim() ?? "";
            if (nextToastMessage && nextToastMessage !== lastToastMessage) {
                toast.info(nextToastMessage);
            }
            lastToastMessage = nextToastMessage;
        } catch {
            if (requestId === activeRenderRequestId) {
                toast.error("地图渲染数据加载失败");
            }
        } finally {
            if (requestId === activeRenderRequestId) {
                isLoadingRenderItems = false;
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
        lat: number,
        lng: number,
        options?: { hintMessage?: string },
    ) {
        cancelReverseGeocodeTimer();
        closeClusterPreview();
        skipNextReverseGeocode = true;
		latitude = lat;
		longitude = lng;
		updateSelectedMarker();
		map?.setCenter([lng, lat]);
        await reverseGeocode(lat, lng);
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
                hintMessage: "已通过长按定位到该位置",
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
				placeSearch!.searchNearBy(
					keyword,
					[longitude, latitude],
					configStore.searchRadiusMax, // 使用后端配置的搜索半径上限（米）
                    (status: string, result: AMap.PlaceSearchResult) => {
                        if (
                            status === "complete" &&
                            result.poiList?.pois?.length
                        ) {
                            resolve(
                                result.poiList.pois.map((poi) => ({
                                    name: poi.name,
                                    address: [poi.adname, poi.address]
                                        .filter(Boolean)
                                        .join(" "),
                                    latitude: poi.location.getLat(),
                                    longitude: poi.location.getLng(),
                                })),
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
        skipNextReverseGeocode = true;
		latitude = result.latitude;
		longitude = result.longitude;
        locationName = result.name;
        lastResolvedLat = result.latitude;
        lastResolvedLng = result.longitude;
		closeClusterPreview();
		updateSelectedMarker();
		map?.setCenter([result.longitude, result.latitude]);
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
        skipNextReverseGeocode = true;
		latitude = item.latitude;
		longitude = item.longitude;
        locationName = item.locationName;
        lastResolvedLat = item.latitude;
        lastResolvedLng = item.longitude;
		closeClusterPreview();
		updateSelectedMarker();
		map?.setCenter([item.longitude, item.latitude]);
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
            return await new Promise<string>((resolve) => {
                currentGeocoder.getAddress(
                    [lng, lat],
                    (status: string, result: AMap.GeocodeResult) => {
                        if (status === "complete" && result.regeocode) {
                            const poiName = result.regeocode.pois?.[0]?.name;
                            const resolved =
                                poiName ||
                                result.regeocode.formattedAddress ||
                                "自定义位置";
                            locationName = resolved;
                            lastResolvedLat = lat;
                            lastResolvedLng = lng;
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

    function hasMeaningfulMovement(lat: number, lng: number) {
        return (
            Math.abs(lat - lastResolvedLat) > MIN_LOCATION_DELTA ||
            Math.abs(lng - lastResolvedLng) > MIN_LOCATION_DELTA
        );
    }

    function scheduleReverseGeocode() {
        cancelReverseGeocodeTimer();
        reverseGeocodeTimer = window.setTimeout(() => {
            reverseGeocodeTimer = null;
            if (!hasMeaningfulMovement(latitude, longitude)) {
                return;
            }
            void reverseGeocode(latitude, longitude);
        }, REVERSE_GEOCODE_IDLE_MS);
    }

	function syncCenterFromMap() {
		if (!map) return;
		const center = map.getCenter();
		if (!center) return;
		latitude = center.getLat();
		longitude = center.getLng();
		updateSelectedMarker();
	}

    function handleLiveMapMove() {
        if (mapMoveFrame !== null) {
            return;
        }
        mapMoveFrame = requestAnimationFrame(() => {
            mapMoveFrame = null;
            syncCenterFromMap();
        });
    }

    function handleMapChanged() {
        syncCenterFromMap();
        closeClusterPreview();
        scheduleMapRender();
        if (skipNextReverseGeocode) {
            skipNextReverseGeocode = false;
            return;
        }
        scheduleReverseGeocode();
    }

    async function handleDoubleClick(event: MapClickEvent) {
        if (!map) return;
        const lnglat = event.lnglat ?? map.getCenter();
        const nextLat = lnglat.getLat();
        const nextLng = lnglat.getLng();
        await moveToCoordinate(nextLat, nextLng, {
            hintMessage: "已通过双击定位到该位置",
        });
    }

    async function handleRightClick(event: MapClickEvent) {
        if (!map) return;
        const lnglat = event.lnglat ?? map.getCenter();
        const nextLat = lnglat.getLat();
        const nextLng = lnglat.getLng();
        await moveToCoordinate(nextLat, nextLng, {
            hintMessage: "已通过右键定位到该位置",
        });
    }

    async function initMap() {
        isLoadingMap = true;
        try {
			await loadAMap();
			await tick();
			if (!mapContainer) return;

			map = new AMap.Map(mapContainer, {
				center: [longitude, latitude],
                zoom: 14,
                viewMode: "2D",
                doubleClickZoom: false,
                resizeEnable: true,
            });
            ensureSelectedMarker();
            registerTouchListeners();
            registerContextMenuBlocker();

            map.on("movestart", () => {
                cancelReverseGeocodeTimer();
            });
            map.on("moveend", () => {
                handleMapChanged();
            });
            map.on("zoomstart", () => {
                cancelReverseGeocodeTimer();
            });
            map.on("zoomend", () => {
                handleMapChanged();
            });
            map.on("mapmove", handleLiveMapMove);
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
			updateSelectedMarker();
			map?.setZoomAndCenter(15, [longitude, latitude]);
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
        skipNextReverseGeocode = true;
		latitude = anchorLat;
		longitude = anchorLng;
		updateSelectedMarker();
		map?.setZoomAndCenter(14, [anchorLng, anchorLat]);
        if (mapAnchorStore.name) {
            locationName = mapAnchorStore.name;
            lastResolvedLat = anchorLat;
            lastResolvedLng = anchorLng;
        } else {
            await reverseGeocode(anchorLat, anchorLng);
        }
        hideGestureHint();
        showPlacementHint("已回到浏览锚点位置");
    }

    function handleConfirm() {
        const normalizedName =
            locationName.trim() ||
            `${latitude.toFixed(6)}, ${longitude.toFixed(6)}`;
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
					map?.setZoomAndCenter(14, [longitude, latitude]);
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
        registerOutsidePointerListener();
    });
</script>

<Dialog.Root
    bind:open
    onOpenChange={(value) => {
        open = value;
        onOpenChange?.(value);
    }}
>
    <Dialog.Content
        class="flex h-[90dvh] max-h-[90dvh] flex-col p-0 sm:h-[85vh] sm:max-w-4xl sm:rounded-3xl sm:border sm:border-border/60 sm:bg-card/95 sm:p-6"
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

        <!-- 地图容器 -->
        <div
            class="relative flex-1 overflow-hidden sm:rounded-3xl sm:border sm:border-border/60"
        >
            <div
                class="h-full w-full touch-none"
                bind:this={mapContainer}
            ></div>

            <!-- 桌面端地图工具栏：右上角为筛选/搜索，右下角为地图操作（当前位置、锚点、放大、缩小） -->
            <div
                class="pointer-events-none absolute right-4 top-4 z-30 hidden flex-col gap-2 sm:flex"
            >
                <div class="pointer-events-auto flex flex-col gap-2">
                    <Button
                        size="icon"
                        variant="ghost"
                        class="h-11 w-11 rounded-[20px] border border-white/60 bg-white/78 text-foreground shadow-[0_14px_28px_rgba(15,23,42,0.14)] backdrop-blur hover:bg-white/90 active:bg-white dark:border-border/60 dark:bg-card/90 dark:shadow-[0_14px_28px_rgba(0,0,0,0.4)] dark:hover:bg-card dark:active:bg-card"
                        onclick={toggleSearch}
                        aria-label="搜索地点"
                    >
                        <Search class="h-4 w-4" />
                    </Button>
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
                            <LocateFixed class="h-6 w-6 text-blue-600 dark:text-blue-400" />
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

            {#if isAnchorMode || isLoadingRenderItems || isLocating}
                <div
                    class="pointer-events-none absolute left-3 top-3 z-30 flex max-w-[min(78vw,22rem)] flex-wrap gap-2 sm:left-4 sm:top-4"
                >
                    <span
                        class="inline-flex items-center gap-2 rounded-full border border-white/60 bg-white/84 px-3 py-1.5 text-[11px] font-semibold uppercase tracking-[0.14em] text-slate-700 shadow-[0_10px_24px_rgba(15,23,42,0.10)] backdrop-blur dark:border-border/60 dark:bg-card/90 dark:text-foreground dark:shadow-[0_10px_24px_rgba(0,0,0,0.3)]"
                    >
                        <MapPin class="h-3.5 w-3.5 text-primary" />
                        {mapModeLabel}
                    </span>
                    <span
                        class="inline-flex items-center gap-2 rounded-full border border-white/60 bg-white/84 px-3 py-1.5 text-xs font-medium text-slate-600 shadow-[0_10px_24px_rgba(15,23,42,0.10)] backdrop-blur dark:border-border/60 dark:bg-card/90 dark:text-muted-foreground dark:shadow-[0_10px_24px_rgba(0,0,0,0.3)]"
                    >
                        {#if isLoadingRenderItems || isLocating}
                            <Loader2
                                class="h-3.5 w-3.5 animate-spin text-primary"
                            />
                        {/if}
                        <span>{mapStatusText}</span>
                    </span>
                </div>
            {/if}

            {#if showFilterPanel}
                <div
                    bind:this={filterPanelElement}
                    class="absolute right-3 top-16 z-30 w-40 overflow-hidden rounded-3xl border border-white/60 bg-[linear-gradient(180deg,rgba(255,255,255,0.96),rgba(248,250,252,0.92))] p-2 shadow-[0_24px_60px_rgba(15,23,42,0.18)] backdrop-blur-md dark:border-border/60 dark:bg-[linear-gradient(180deg,rgba(30,30,36,0.96),rgba(24,24,30,0.92))] dark:shadow-[0_24px_60px_rgba(0,0,0,0.5)] sm:right-4 sm:top-16"
                >
                    <button
                        class={`flex w-full items-center justify-between rounded-2xl px-3 py-2.5 text-left text-sm font-medium transition ${showUsers ? "bg-primary/8 text-foreground" : "text-muted-foreground hover:bg-muted/60"}`}
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
                        class={`mt-1 flex w-full items-center justify-between rounded-2xl px-3 py-2.5 text-left text-sm font-medium transition ${showGroups ? "bg-primary/8 text-foreground" : "text-muted-foreground hover:bg-muted/60"}`}
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
                        class={`mt-1 flex w-full items-center justify-between rounded-2xl px-3 py-2.5 text-left text-sm font-medium transition ${showMoments ? "bg-primary/8 text-foreground" : "text-muted-foreground hover:bg-muted/60"}`}
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
                            class="rounded-2xl border border-dashed border-primary/40 bg-card/90 px-4 py-1 text-xs text-muted-foreground shadow"
                        >
                            {gestureHintMessage}
                        </div>
                    {/if}
                </div>
            {/if}

            <!-- 搜索覆盖层 -->
            {#if showSearch}
                <div
                    class="absolute inset-x-3 top-3 z-30 overflow-hidden rounded-[28px] border border-white/60 bg-[linear-gradient(180deg,rgba(255,255,255,0.96),rgba(248,250,252,0.92))] shadow-[0_24px_60px_rgba(15,23,42,0.18)] backdrop-blur-md dark:border-border/60 dark:bg-[linear-gradient(180deg,rgba(30,30,36,0.96),rgba(24,24,30,0.92))] dark:shadow-[0_24px_60px_rgba(0,0,0,0.5)] sm:inset-x-auto sm:left-4 sm:right-auto sm:top-4 sm:w-[26rem]"
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
                                <p class="mt-1 text-xs text-muted-foreground">
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
                            class="flex items-center gap-2 rounded-2xl border border-border/60 bg-white/80 px-3 py-2.5 shadow-sm dark:bg-card/80"
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
                    </div>

                    <!-- 搜索结果列表 -->
                    {#if searchResults.length > 0}
                        <div class="max-h-80 overflow-y-auto px-2 pb-2">
                            {#each searchResults as result (`${result.latitude}_${result.longitude}`)}
                                <button
                                    class="mx-2 mb-2 flex w-[calc(100%-1rem)] items-start gap-3 rounded-2xl border border-border/50 bg-white/72 px-4 py-3 text-left shadow-sm transition-all hover:-translate-y-0.5 hover:border-primary/25 hover:bg-white dark:bg-card/72 dark:hover:bg-card"
                                    onclick={() => selectSearchResult(result)}
                                >
                                    <div
                                        class="flex h-9 w-9 shrink-0 items-center justify-center rounded-2xl bg-primary/10"
                                    >
                                        <MapPin class="h-4 w-4 text-primary" />
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
                                                `${result.latitude.toFixed(6)}, ${result.longitude.toFixed(6)}`}
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
                        <div class="max-h-80 overflow-y-auto px-2 pb-2">
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
                                <p class="mt-1 text-xs text-muted-foreground">
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
                        <LocateFixed class="h-7 w-7 text-blue-600 dark:text-blue-400" />
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
                    class="absolute inset-x-3 bottom-32 z-30 sm:bottom-6 sm:left-4 sm:right-auto sm:w-[25rem]"
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
                                <p class="mt-1 text-xs text-muted-foreground">
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
                                {#each clusterPreview.items as previewItem (previewItem.render_id)}
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
                                                {previewItem.entity?.title ||
                                                    "未命名内容"}
                                            </p>
                                            {#if previewItem.entity?.subtitle}
                                                <p
                                                    class="mt-1 line-clamp-2 text-xs text-muted-foreground"
                                                >
                                                    {previewItem.entity
                                                        .subtitle}
                                                </p>
                                            {/if}
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

            <!-- 移动端底部抽屉 -->
            <div
                class="absolute bottom-0 left-0 right-0 z-20 transition-transform duration-300 sm:hidden"
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
                            (isBottomSheetExpanded = !isBottomSheetExpanded)}
                    >
                        <div
                            class="h-1 w-12 rounded-full bg-muted-foreground/20"
                        ></div>
                        {#if isBottomSheetExpanded}
                            <ChevronDown
                                class="h-4 w-4 text-muted-foreground"
                            />
                        {:else}
                            <ChevronUp class="h-4 w-4 text-muted-foreground" />
                        {/if}
                    </button>

                    <!-- 底部抽屉内容 -->
                    <div class="max-h-[60vh] overflow-y-auto px-4 pb-safe">
                        <!-- 位置信息 (精简版 - 始终可见) -->
                        <AnchorSummaryCard
                            label="当前选中"
                            description={selectionDescription}
                            subtle
                            overrideName={locationName || "拖动地图选择位置"}
                            overrideLatitude={latitude}
                            overrideLongitude={longitude}
                            showCoordinates
                        />

                        <!-- 展开后显示的内容 -->
                        {#if isBottomSheetExpanded}
                            <div class="space-y-4 pb-4">
                                <!-- 提示信息 -->
                                <div
                                    class="rounded-2xl border border-border/50 bg-background/65 p-3 text-xs text-muted-foreground"
                                >
                                    💡 {hintAfterConfirm}
                                </div>

                                {#if isAnchorMode}
                                    <div
                                        class="rounded-2xl border border-border/60 bg-background/70 p-3 text-xs text-muted-foreground"
                                    >
                                        搜索范围将继续使用当前偏好设置；如果你确实需要修改，建议放到设置页统一调整。
                                    </div>
                                {/if}
                            </div>
                        {/if}

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

                <div
                    class="pointer-events-none absolute inset-x-4 bottom-4 z-30 hidden sm:block"
                >
                    <div
                        class="pointer-events-auto flex items-end gap-3 rounded-[28px] border border-white/60 bg-[linear-gradient(180deg,rgba(255,255,255,0.88),rgba(248,250,252,0.82))] p-3 shadow-[0_24px_60px_rgba(15,23,42,0.16)] backdrop-blur-md dark:border-border/60 dark:bg-[linear-gradient(180deg,rgba(30,30,36,0.88),rgba(24,24,30,0.82))] dark:shadow-[0_24px_60px_rgba(0,0,0,0.5)]"
                    >
                        <div
                            class="min-w-0 flex-1 rounded-[22px] bg-white/78 px-4 py-3 shadow-sm dark:bg-card/78"
                        >
                            <div class="flex items-start gap-3">
                                <div
                                    class="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl bg-primary/10"
                                >
                                    <MapPin class="h-5 w-5 text-primary" />
                                </div>
                                <div class="min-w-0 flex-1">
                                    <p
                                        class="text-[11px] font-semibold uppercase tracking-[0.14em] text-primary/70"
                                    >
                                        当前选中
                                    </p>
                                    <p
                                        class="truncate text-base font-bold leading-snug text-foreground"
                                    >
                                        {locationName || "拖动地图选择位置"}
                                    </p>
                                    <p
                                        class="mt-1 font-mono text-[11px] text-muted-foreground"
                                    >
                                        {latitude.toFixed(6)}, {longitude.toFixed(
                                            6,
                                        )}
                                    </p>
                                    {#if isAnchorMode}
                                        <p
                                            class="mt-2 text-xs text-muted-foreground"
                                        >
                                            搜索范围使用当前偏好设置，主视图优先留给地图浏览。
                                        </p>
                                    {/if}
                                </div>
                            </div>
                        </div>
                        <div
                            class="w-[13.5rem] shrink-0 rounded-[24px] bg-slate-950 px-3 py-3 text-white shadow-[0_18px_36px_rgba(15,23,42,0.24)] dark:bg-slate-900 dark:shadow-[0_18px_36px_rgba(0,0,0,0.5)]"
                        >
                            <p
                                class="text-[11px] font-semibold uppercase tracking-[0.14em] text-white/60"
                            >
                                操作
                            </p>
                            <p class="mt-2 text-sm font-medium text-white/80">
                                {locationReadyText}
                            </p>
                            <div class="mt-3 space-y-2">
                                {#if isAnchorMode}
                                    <Button
                                        variant="outline"
                                        size="sm"
                                        class="h-10 w-full gap-1.5 rounded-2xl border-white/15 bg-white/10 text-white hover:bg-white/15"
                                        onclick={() => (showShareModal = true)}
                                        disabled={!locationName}
                                    >
                                        <Share2 class="h-3.5 w-3.5" />
                                        分享此位置
                                    </Button>
                                {/if}
                                <Button
                                    size="sm"
                                    class="h-10 w-full gap-1.5 rounded-2xl bg-primary text-primary-foreground hover:bg-primary/90"
                                    onclick={handleConfirm}
                                    disabled={!locationName}
                                >
                                    <MapPin class="h-3.5 w-3.5" />
                                    {confirmButtonText}
                                </Button>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </Dialog.Content>
</Dialog.Root>

<!-- 位置分享模态框 -->
<CreateLocationShareModal
    bind:open={showShareModal}
    {latitude}
    {longitude}
    {locationName}
/>

<style>
    :global(.map-selection-marker) {
        display: flex;
        align-items: center;
        justify-content: center;
        pointer-events: none;
    }

    :global(.map-selection-marker-svg) {
        width: 36px;
        height: 48px;
        transform: translateY(-4px);
    }

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
        background-image: url("/map/marker_user.svg");
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
        background-image: url("/map/marker_group.svg");
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
        background-image: url("/map/marker_moment.svg");
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
        background-image: url("/map/marker_cluster.svg");
    }
</style>
