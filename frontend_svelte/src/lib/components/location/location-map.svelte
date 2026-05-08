<script lang="ts">
    import { onMount, untrack } from "svelte";
    import { assertSuccess } from "$lib/api/client";
    import {
        mapService,
        groupOverlappingRenderItems,
    } from "$lib/api/services/map";
    import { mapAnchorStore } from "$lib/stores/map-anchor.svelte";
    import {
        getAMapVisibleRegion,
        loadAMap,
        loadAMapPlugin,
    } from "$lib/utils/amap";
    import type {
        MapCanvasRequest,
        MapCanvasResponse,
        MapHeatBucket,
        MapHotspotItem,
        MapHotspotsCanvas,
        MapHeatCanvas,
        MapViewport,
    } from "$lib/types/api";
    import { toGcjLngLat, gcj02ToWgs84 } from "$lib/utils/geo";
    import { mode as themeMode } from "mode-watcher";
    import Crosshair from "lucide-svelte/icons/crosshair";
    import Plus from "lucide-svelte/icons/plus";
    import Minus from "lucide-svelte/icons/minus";
    import Compass from "lucide-svelte/icons/compass";
    import MapPin from "lucide-svelte/icons/map-pin";
    import X from "lucide-svelte/icons/x";
    import Users from "lucide-svelte/icons/users";
    import Heart from "lucide-svelte/icons/heart";
    import ChevronRight from "lucide-svelte/icons/chevron-right";

    // ---------------------------------------------------------------------------
    // Props
    // ---------------------------------------------------------------------------

    interface Props {
        onviewportchange?: (viewport: MapViewport) => void;
        onpinselect?: (item: MapHotspotItem | null) => void;
        onmapclick?: () => void;
        selectedLocation?: { latitude: number; longitude: number } | null;
        showControls?: boolean;
        isMobile?: boolean;
        bottomOffset?: number;
    }

    let {
        onviewportchange,
        onpinselect,
        onmapclick,
        selectedLocation = null,
        showControls = true,
        isMobile = false,
        bottomOffset = 0,
    }: Props = $props();

    // ---------------------------------------------------------------------------
    // Constants
    // ---------------------------------------------------------------------------

    const MAP_RENDER_IDLE_MS = 250;
    const LONG_PRESS_DURATION_MS = 700;
    const DEFAULT_MARKER_ANCHOR: AMap.MarkerAnchor = "bottom-center";

    // ---------------------------------------------------------------------------
    // State
    // ---------------------------------------------------------------------------

    let mapContainer = $state<HTMLDivElement | null>(null);
    let map: AMap.Map | null = null;
    let renderMapTimer: number | null = null;
    let activeRenderRequestId = 0;
    let lastRenderedViewportSignature = "";
    let renderMarkers: AMap.Marker[] = [];
    let centerPinMarker: AMap.Marker | null = null;
    let currentTheme = $state(themeMode.current);
    let selectedPin = $state<MapHotspotItem | null>(null);

    interface AMapMarkerCluster {
        clearMarkers?(): void;
        setMap?(map: AMap.Map | null): void;
    }

    interface HotspotMarkerClusterContext {
        count?: number;
        markers?: AMap.Marker[];
        clusterData?: AMap.Marker[];
        marker: AMap.Marker;
    }

    interface AMapMarkerClusterConstructor {
        new (
            map: AMap.Map,
            markers: AMap.Marker[],
            options: {
                gridSize?: number;
                minClusterSize?: number;
                maxZoom?: number;
                renderClusterMarker?: (
                    context: HotspotMarkerClusterContext,
                ) => void;
            },
        ): AMapMarkerCluster;
    }

    interface AMapMarkerClusterFactory {
        MarkerClusterer?: AMapMarkerClusterConstructor;
    }

    let hotspotsClusterer: AMapMarkerCluster | null = null;

    // Touch long-press state
    let longPressTimer: number | null = null;
    let lastTouchLngLat: { lat: number; lng: number } | null = null;
    let touchMovedDuringLongPress = false;
    let detachTouchListeners: (() => void) | null = null;

    type RenderItem = MapHeatBucket & { render_id: string };

    // ---------------------------------------------------------------------------
    // Helpers
    // ---------------------------------------------------------------------------

    function createMarkerIcon(image: string): AMap.Icon {
        return new AMap.Icon({
            size: new AMap.Size(40, 50),
            image,
            imageSize: new AMap.Size(40, 50),
            imageOffset: new AMap.Pixel(0, 0),
        });
    }

    function getMarkerIconUrl(): string {
        return "/map/marker_mixed_cluster.svg";
    }

    function getHotspotMarkerIcon(item: MapHotspotItem): string {
        if (item.object_type === "user") return "/map/marker_user.svg";
        if (item.object_type === "group") return "/map/marker_group.svg";
        if (item.object_type === "moment") return "/map/marker_moment.svg";
        return "/map/marker_user.svg";
    }

    function clampLatitude(lat: number): number {
        if (!Number.isFinite(lat)) return lat;
        return Math.max(-85, Math.min(85, lat));
    }

    function normalizeLongitude(lng: number): number {
        if (!Number.isFinite(lng)) return lng;
        const normalized = ((((lng + 180) % 360) + 360) % 360) - 180;
        return normalized;
    }

    function getViewportSignature(viewport: MapViewport): string {
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

        return {
            bounds: { north, south, east, west },
            zoom: Math.max(1, Math.min(20, visibleRegion.zoom)),
        };
    }

    // ---------------------------------------------------------------------------
    // Marker management
    // ---------------------------------------------------------------------------

    function disposeRenderMarkers(
        markers: AMap.Marker[],
        clusterer: AMapMarkerCluster | null,
    ) {
        if (clusterer) {
            try {
                clusterer.clearMarkers?.();
                clusterer.setMap?.(null);
            } catch {
                // ignore cleanup errors
            }
        }
        for (const marker of markers) {
            marker.setMap(null);
            marker.remove();
        }
    }

    function clearRenderMarkers() {
        disposeRenderMarkers(renderMarkers, hotspotsClusterer);
        renderMarkers = [];
        hotspotsClusterer = null;
    }

    function replaceRenderMarkers(
        nextMarkers: AMap.Marker[],
        options?: { clusterer?: AMapMarkerCluster | null },
    ) {
        const previousMarkers = renderMarkers;
        const previousClusterer = hotspotsClusterer;
        const nextClusterer = options?.clusterer ?? null;

        renderMarkers = nextMarkers;
        hotspotsClusterer = nextClusterer;

        if (!nextClusterer) {
            for (const marker of nextMarkers) {
                marker.setMap(map);
            }
        }

        disposeRenderMarkers(previousMarkers, previousClusterer);
    }

    // ---------------------------------------------------------------------------
    // Center pin marker (selectedLocation prop)
    // ---------------------------------------------------------------------------

    function ensureCenterPinMarker() {
        if (!map || centerPinMarker) return;
        const anchor = selectedLocation ?? {
            latitude: mapAnchorStore.latitude,
            longitude: mapAnchorStore.longitude,
        };
        const [gcjLng, gcjLat] = toGcjLngLat(anchor);
        centerPinMarker = new AMap.Marker({
            position: [gcjLng, gcjLat],
            icon: createMarkerIcon("/map/marker_center_pin.svg"),
            anchor: DEFAULT_MARKER_ANCHOR,
            offset: new AMap.Pixel(0, 0),
            clickable: false,
            map,
        });
    }

    function updateCenterPinMarker() {
        if (!map) return;
        ensureCenterPinMarker();
        if (!centerPinMarker) return;
        const anchor = selectedLocation ?? {
            latitude: mapAnchorStore.latitude,
            longitude: mapAnchorStore.longitude,
        };
        const [gcjLng, gcjLat] = toGcjLngLat(anchor);
        centerPinMarker.setPosition([gcjLng, gcjLat]);
    }

    function clearCenterPinMarker() {
        centerPinMarker?.setMap(null);
        centerPinMarker?.remove();
        centerPinMarker = null;
    }

    // ---------------------------------------------------------------------------
    // Viewport change notification
    // ---------------------------------------------------------------------------

    function notifyViewportChange() {
        const viewport = buildViewport();
        if (viewport) {
            onviewportchange?.(viewport);
        }
    }

    // ---------------------------------------------------------------------------
    // Map data fetching & rendering
    // ---------------------------------------------------------------------------

    function createHeatMarker(
        item: RenderItem,
        attachToMap = true,
    ): AMap.Marker | null {
        if (!map) return null;
        const position = toGcjLngLat({
            latitude: item.location.latitude,
            longitude: item.location.longitude,
        });
        const marker = new AMap.Marker({
            position,
            icon: createMarkerIcon(getMarkerIconUrl()),
            anchor: DEFAULT_MARKER_ANCHOR,
            clickable: true,
            map: attachToMap ? map : undefined,
        });
        marker.setLabel?.({
            content: `${item.total_count} 个点`,
            direction: "top",
        });
        return marker;
    }

    async function fetchAndRenderMapData() {
        if (!map) return;

        const viewport = buildViewport();
        if (!viewport) return;

        const signature = getViewportSignature(viewport);
        if (signature === lastRenderedViewportSignature && renderMarkers.length > 0) {
            return;
        }

        const requestId = ++activeRenderRequestId;

        try {
            const request: MapCanvasRequest = {
                viewport,
                pagination: { limit: 100 },
            };
            const response = await mapService.canvas(request);
            assertSuccess(response);

            if (requestId !== activeRenderRequestId) return;

            lastRenderedViewportSignature = signature;
            const canvas = (response.data ?? {}) as MapCanvasResponse;

            if (canvas.hotspots) {
                const data = canvas.hotspots as MapHotspotsCanvas;
                const items: MapHotspotItem[] = data.items ?? [];

                if (items.length > 0) {
                    await loadAMapPlugin(["AMap.MarkerClusterer"]);
                    const markers: AMap.Marker[] = [];

                    for (const item of items) {
                        const [gcjLng, gcjLat] = toGcjLngLat({
                            latitude: item.location.latitude,
                            longitude: item.location.longitude,
                        });
                        const marker = new AMap.Marker({
                            position: [gcjLng, gcjLat],
                            icon: createMarkerIcon(getHotspotMarkerIcon(item)),
                            anchor: DEFAULT_MARKER_ANCHOR,
                            clickable: true,
                        });
                        marker.setExtData?.(item);
                        marker.on("click", (event) => {
                            event.stopPropagation?.();
                            selectPin(item);
                        });
                        markers.push(marker);
                    }

                    const factory = AMap as unknown as AMapMarkerClusterFactory;
                    const MarkerClusterer = factory.MarkerClusterer;
                    if (!MarkerClusterer) {
                        clearRenderMarkers();
                        return;
                    }

                    const nextClusterer = new MarkerClusterer(map, markers, {
                        gridSize: 60,
                        minClusterSize: 2,
                        maxZoom: 18,
                        renderClusterMarker: (context) => {
                            const count = context.count ?? 0;
                            const children = context.markers ?? context.clusterData ?? [];
                            let hasUser = false;
                            let hasGroup = false;
                            let hasMoment = false;

                            for (const m of children) {
                                const data = m.getExtData?.() as MapHotspotItem | undefined;
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

                            div.onclick = (e: MouseEvent) => {
                                e.stopPropagation();
                                // Click on cluster zooms in
                                map?.setZoom((map?.getZoom() ?? 14) + 2);
                            };

                            context.marker.setContent?.(div);
                        },
                    });

                    replaceRenderMarkers(markers, { clusterer: nextClusterer });
                } else {
                    clearRenderMarkers();
                }
            } else if (canvas.heatmap) {
                const data = canvas.heatmap as MapHeatCanvas;
                const buckets: MapHeatBucket[] = data.items ?? [];
                const renderItems: RenderItem[] = buckets.map(
                    (bucket, index): RenderItem => ({
                        ...bucket,
                        render_id: `${index}_${bucket.location.latitude.toFixed(6)}_${bucket.location.longitude.toFixed(6)}`,
                    }),
                );

                const zoom = map.getZoom() ?? 14;
                const { merged } = groupOverlappingRenderItems(renderItems, zoom);
                const nextMarkers = merged
                    .map(({ item }) => createHeatMarker(item, false))
                    .filter((m): m is AMap.Marker => Boolean(m));

                replaceRenderMarkers(nextMarkers);
            }

            updateCenterPinMarker();
            notifyViewportChange();
        } catch {
            // Silently handle map data fetch errors
        }
    }

    function scheduleMapRender() {
        if (renderMapTimer) {
            clearTimeout(renderMapTimer);
        }
        renderMapTimer = window.setTimeout(() => {
            renderMapTimer = null;
            void fetchAndRenderMapData();
        }, MAP_RENDER_IDLE_MS);
    }

    // ---------------------------------------------------------------------------
    // Pin selection
    // ---------------------------------------------------------------------------

    function selectPin(item: MapHotspotItem) {
        selectedPin = item;
        onpinselect?.(item);
    }

    function clearPinSelection() {
        selectedPin = null;
        onpinselect?.(null);
    }

    // ---------------------------------------------------------------------------
    // Touch long-press
    // ---------------------------------------------------------------------------

    function getLngLatFromTouch(event: TouchEvent): { lat: number; lng: number } | null {
        if (!map || !mapContainer) return null;
        const touch = event.touches[0] ?? event.changedTouches[0];
        if (!touch) return null;
        const rect = mapContainer.getBoundingClientRect();
        const pixel = {
            x: touch.clientX - rect.left,
            y: touch.clientY - rect.top,
        };
        const lnglat = map.containerToLngLat(pixel);
        if (!lnglat) return null;
        return { lat: lnglat.getLat(), lng: lnglat.getLng() };
    }

    function handleTouchStart(event: TouchEvent) {
        const lnglat = getLngLatFromTouch(event);
        if (!lnglat) return;
        lastTouchLngLat = lnglat;
        touchMovedDuringLongPress = false;
        if (longPressTimer) clearTimeout(longPressTimer);
        longPressTimer = window.setTimeout(() => {
            if (touchMovedDuringLongPress || !lastTouchLngLat) return;
            // Long press sets position — handled by parent via onmapclick
            onmapclick?.();
        }, LONG_PRESS_DURATION_MS);
    }

    function handleTouchMove() {
        touchMovedDuringLongPress = true;
        if (longPressTimer) {
            clearTimeout(longPressTimer);
            longPressTimer = null;
        }
    }

    function handleTouchEnd() {
        if (longPressTimer) {
            clearTimeout(longPressTimer);
            longPressTimer = null;
        }
        lastTouchLngLat = null;
    }

    function registerTouchListeners() {
        detachTouchListeners?.();
        if (!mapContainer) return;
        mapContainer.addEventListener("touchstart", handleTouchStart, { passive: true });
        mapContainer.addEventListener("touchmove", handleTouchMove, { passive: true });
        mapContainer.addEventListener("touchend", handleTouchEnd, { passive: true });
        detachTouchListeners = () => {
            mapContainer?.removeEventListener("touchstart", handleTouchStart);
            mapContainer?.removeEventListener("touchmove", handleTouchMove);
            mapContainer?.removeEventListener("touchend", handleTouchEnd);
            detachTouchListeners = null;
        };
    }

    // ---------------------------------------------------------------------------
    // Floating controls
    // ---------------------------------------------------------------------------

    function handleGpsRecenter() {
        const anchor = mapAnchorStore;
        const [gcjLng, gcjLat] = toGcjLngLat({
            latitude: anchor.latitude,
            longitude: anchor.longitude,
        });
        map?.setZoomAndCenter(14, [gcjLng, gcjLat]);
    }

    function handleZoomIn() {
        const zoom = map?.getZoom() ?? 14;
        map?.setZoom(Math.min(20, zoom + 1));
    }

    function handleZoomOut() {
        const zoom = map?.getZoom() ?? 14;
        map?.setZoom(Math.max(3, zoom - 1));
    }

    function handleResetCompass() {
        // Reset to north-up orientation
        map?.setZoomAndCenter(
            map?.getZoom() ?? 14,
            map?.getCenter()
                ? [map.getCenter().getLng(), map.getCenter().getLat()]
                : toGcjLngLat({
                      latitude: mapAnchorStore.latitude,
                      longitude: mapAnchorStore.longitude,
                  }),
        );
    }

    // ---------------------------------------------------------------------------
    // Pin popup helpers (desktop)
    // ---------------------------------------------------------------------------

    const pinTypeLabel = $derived(
        selectedPin?.object_type === "user"
            ? "用户"
            : selectedPin?.object_type === "group"
              ? "群组"
              : "瞬间",
    );

    const pinTypeColor = $derived(
        selectedPin?.object_type === "user"
            ? "var(--neu-accent-primary)"
            : selectedPin?.object_type === "group"
              ? "var(--neu-accent-secondary, #6366f1)"
              : "var(--neu-accent-tertiary, #f59e0b)",
    );

    function getPinIcon() {
        if (!selectedPin) return MapPin;
        if (selectedPin.object_type === "user") return MapPin;
        if (selectedPin.object_type === "group") return Users;
        return Heart;
    }

    function closePinPopup() {
        clearPinSelection();
    }

    // ---------------------------------------------------------------------------
    // Theme
    // ---------------------------------------------------------------------------

    $effect(() => {
        const mode = themeMode.current;
        if (mode === currentTheme) return;
        currentTheme = mode;
        if (map) {
            try {
                map.setMapStyle?.(
                    mode === "dark"
                        ? "amap://styles/dark"
                        : "amap://styles/normal",
                );
            } catch {
                // ignore style errors
            }
        }
    });

    // ---------------------------------------------------------------------------
    // Watch selectedLocation changes
    // ---------------------------------------------------------------------------

    $effect(() => {
        // React to selectedLocation changes
        const loc = selectedLocation;
        if (!map) return;
        untrack(() => {
            if (loc) {
                const [gcjLng, gcjLat] = toGcjLngLat(loc);
                map?.setCenter([gcjLng, gcjLat]);
            }
            updateCenterPinMarker();
        });
    });

    // ---------------------------------------------------------------------------
    // Map initialization
    // ---------------------------------------------------------------------------

    async function initMap() {
        try {
            await loadAMap();
            if (!mapContainer) return;

            const anchor = selectedLocation ?? {
                latitude: mapAnchorStore.latitude,
                longitude: mapAnchorStore.longitude,
            };
            const [gcjLng, gcjLat] = toGcjLngLat(anchor);

            map = new AMap.Map(mapContainer, {
                center: [gcjLng, gcjLat],
                zoom: 14,
                viewMode: "2D",
                doubleClickZoom: false,
                resizeEnable: true,
                touchZoom: true,
                dragEnable: true,
            });

            try {
                map.setMapStyle?.(
                    currentTheme === "dark"
                        ? "amap://styles/dark"
                        : "amap://styles/normal",
                );
            } catch {
                // ignore style errors
            }

            ensureCenterPinMarker();
            registerTouchListeners();

            map.on("moveend", () => {
                scheduleMapRender();
                notifyViewportChange();
            });

            map.on("zoomend", () => {
                scheduleMapRender();
                notifyViewportChange();
            });

            map.on("click", () => {
                clearPinSelection();
                onmapclick?.();
            });

            // Initial data fetch
            void fetchAndRenderMapData();
            notifyViewportChange();
        } catch {
            if (map) {
                map.destroy();
                map = null;
            }
        }
    }

    // ---------------------------------------------------------------------------
    // Lifecycle
    // ---------------------------------------------------------------------------

    onMount(() => {
        void initMap();

        return () => {
            // Cleanup on destroy
            if (renderMapTimer) {
                clearTimeout(renderMapTimer);
                renderMapTimer = null;
            }
            if (longPressTimer) {
                clearTimeout(longPressTimer);
                longPressTimer = null;
            }
            clearRenderMarkers();
            clearCenterPinMarker();
            detachTouchListeners?.();
            if (map) {
                map.destroy();
                map = null;
            }
        };
    });
</script>

<!-- Map container (full size) -->
<div class="relative h-full w-full overflow-hidden">
    <div
        class="h-full w-full touch-pan-x touch-pan-y"
        bind:this={mapContainer}
    ></div>

    <!-- Pulsing dot at map center (current position indicator) -->
    <div class="pointer-events-none absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-10">
        <div class="location-map-pulse-dot"></div>
    </div>

    <!-- Floating controls -->
    {#if showControls}
        <div
            class="pointer-events-auto absolute right-3 z-20 flex flex-col gap-2"
            style={isMobile
                ? `bottom: ${Math.max(16, bottomOffset + 16)}px;`
                : 'top: 16px;'}
        >
            <!-- GPS re-center -->
            <button
                class="location-map-control-btn"
                onclick={handleGpsRecenter}
                aria-label="重新定位到锚点"
            >
                <Crosshair class="h-5 w-5" style="color: var(--neu-text);" />
            </button>

            <!-- Zoom in -->
            <button
                class="location-map-control-btn"
                onclick={handleZoomIn}
                aria-label="放大"
            >
                <Plus class="h-5 w-5" style="color: var(--neu-text);" />
            </button>

            <!-- Zoom out -->
            <button
                class="location-map-control-btn"
                onclick={handleZoomOut}
                aria-label="缩小"
            >
                <Minus class="h-5 w-5" style="color: var(--neu-text);" />
            </button>

            <!-- Compass (desktop only) -->
            {#if !isMobile}
                <button
                    class="location-map-control-btn"
                    onclick={handleResetCompass}
                    aria-label="重置指南针"
                >
                    <Compass class="h-5 w-5" style="color: var(--neu-text);" />
                </button>
            {/if}
        </div>
    {/if}

    <!-- Selected pin popup (desktop only) -->
    {#if selectedPin && !isMobile}
        <div class="location-map-pin-popup">
            <div class="location-map-pin-popup-header">
                <div class="flex items-center gap-2 min-w-0">
                    <span
                        class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-medium"
                        style="background: {pinTypeColor}20; color: {pinTypeColor};"
                    >
                        <svelte:component this={getPinIcon()} class="h-3 w-3" />
                        {pinTypeLabel}
                    </span>
                    <span class="truncate text-sm font-semibold" style="color: var(--neu-text);">
                        {selectedPin.title}
                    </span>
                </div>
                <button
                    class="shrink-0 rounded-full p-1 transition-colors hover:bg-black/10"
                    onclick={closePinPopup}
                    aria-label="关闭"
                >
                    <X class="h-4 w-4" style="color: var(--neu-text-muted);" />
                </button>
            </div>

            <div class="location-map-pin-popup-body">
                {#if selectedPin.subtitle}
                    <p class="text-xs" style="color: var(--neu-text-muted);">
                        {selectedPin.subtitle}
                    </p>
                {/if}
                {#if selectedPin.distance_meters != null}
                    <p class="text-xs" style="color: var(--neu-text-muted);">
                        {selectedPin.distance_meters < 1000
                            ? `${Math.round(selectedPin.distance_meters)} 米`
                            : `${(selectedPin.distance_meters / 1000).toFixed(1)} 公里`}
                    </p>
                {/if}
            </div>

            <button class="location-map-pin-popup-action" onclick={() => onpinselect?.(selectedPin)}>
                查看详情
                <ChevronRight class="h-4 w-4" />
            </button>
        </div>
    {/if}
</div>

<style>
    /* Pulsing dot animation */
    .location-map-pulse-dot {
        width: 16px;
        height: 16px;
        border-radius: 50%;
        background: var(--neu-accent-primary, #3b82f6);
        opacity: 0.7;
        animation: location-map-pulse 2s ease-in-out infinite;
    }

    @keyframes location-map-pulse {
        0% {
            transform: scale(1);
            opacity: 0.7;
        }
        50% {
            transform: scale(1.8);
            opacity: 0.2;
        }
        100% {
            transform: scale(1);
            opacity: 0.7;
        }
    }

    /* Floating control button */
    .location-map-control-btn {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 40px;
        height: 40px;
        border-radius: 14px;
        background: var(--neu-bg);
        border: 1px solid var(--neu-border);
        box-shadow: 0 4px 16px var(--neu-shadow-dark);
        cursor: pointer;
        transition: transform 150ms ease;
    }

    .location-map-control-btn:active {
        transform: scale(0.88);
    }

    /* Selected pin popup card (desktop) */
    .location-map-pin-popup {
        position: absolute;
        bottom: 20px;
        left: 50%;
        transform: translateX(-50%);
        z-index: 30;
        width: min(380px, calc(100% - 32px));
        border-radius: 20px;
        background: var(--neu-bg);
        border: 1px solid var(--neu-border);
        box-shadow: 0 8px 32px var(--neu-shadow-dark);
        padding: 16px;
        animation: location-map-slide-up 250ms ease-out;
    }

    @keyframes location-map-slide-up {
        from {
            transform: translateX(-50%) translateY(20px);
            opacity: 0;
        }
        to {
            transform: translateX(-50%) translateY(0);
            opacity: 1;
        }
    }

    .location-map-pin-popup-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 8px;
    }

    .location-map-pin-popup-body {
        margin-top: 10px;
        display: flex;
        flex-direction: column;
        gap: 2px;
    }

    .location-map-pin-popup-action {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 4px;
        margin-top: 12px;
        width: 100%;
        padding: 8px 0;
        border-radius: 12px;
        background: var(--neu-accent-primary, #3b82f6);
        color: #fff;
        font-size: 13px;
        font-weight: 600;
        border: none;
        cursor: pointer;
        transition: transform 150ms ease, opacity 150ms ease;
    }

    .location-map-pin-popup-action:active {
        transform: scale(0.96);
    }
</style>
