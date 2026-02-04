<template>
    <div
        ref="rootEl"
        class="relative w-full h-full rounded-3xl overflow-hidden ring-1 ring-neutral-200 dark:ring-neutral-800 shadow-sm group"
        @click.capture="handleContainerClick"
    >
        <!-- 地图容器 -->
        <div
            ref="mapContainer"
            class="w-full h-full bg-neutral-100 dark:bg-neutral-900 transition-colors"
            :data-map-interactive="isInteractive ? 'true' : 'false'"
        />

        <!-- 浏览锚点：中心覆盖 -->
        <div
            v-if="mapReady && showAnchor"
            class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-full z-20 pointer-events-none"
        >
            <div class="flex flex-col items-center animate-fade-in-up">
                <div class="relative">
                    <div class="absolute inset-0 bg-primary-500/20 rounded-full animate-ping" />
                    <UBadge
                        color="primary"
                        variant="solid"
                        size="lg"
                        class="lsyb-map-anchor-badge"
                    >
                        <UIcon name="i-lucide-crosshair" class="size-5" />
                    </UBadge>
                </div>
                <div class="mt-2">
                    <UCard variant="subtle" class="lsyb-map-anchor-card">
                        <template #body>
                            <div
                                class="px-2 py-1 text-xs font-medium lsyb-map-anchor-label"
                            >
                                浏览中心
                            </div>
                        </template>
                    </UCard>
                </div>
            </div>
        </div>

        <!-- 交互提示遮罩 -->
        <div
            v-if="mapReady && !isInteractive"
            class="absolute inset-0 z-10 bg-black/0 hover:bg-black/5 transition-colors flex items-center justify-center pointer-events-none group-hover:pointer-events-auto"
        >
            <div class="lsyb-map-interactive-hint-enter">
                <UCard variant="subtle" class="lsyb-map-interactive-hint-card">
                    <div class="flex items-center gap-2 px-3 py-2">
                        <UIcon
                            name="i-lucide-mouse-pointer-click"
                            class="text-primary-500"
                        />
                        <span class="text-sm font-medium text-neutral-900 dark:text-white"
                            >点击地图启用交互</span
                        >
                    </div>
                </UCard>
            </div>
        </div>

        <!-- 加载中 -->
        <div
            v-if="loading"
            class="absolute inset-0 z-50 flex items-center justify-center bg-white/80 dark:bg-neutral-900/80 backdrop-blur-sm"
        >
            <div class="text-center space-y-3">
                <UIcon
                    name="i-lucide-loader-2"
                    class="size-8 text-primary-500 animate-spin mx-auto"
                />
                <div class="text-neutral-500">
                    正在加载地图资源...
                </div>
            </div>
        </div>

        <!-- 浮动控制栏 -->
        <div
            v-if="mapReady"
            class="absolute top-4 right-4 flex flex-col gap-2 z-30"
        >
            <div class="lsyb-map-zoom-controls">
                <UButton
                    icon="i-lucide-plus"
                    variant="ghost"
                    color="neutral"
                    size="sm"
                    class="lsyb-map-zoom-btn lsyb-map-zoom-btn--top"
                    @click="zoomIn"
                />
                <UButton
                    icon="i-lucide-minus"
                    variant="ghost"
                    color="neutral"
                    size="sm"
                    class="lsyb-map-zoom-btn lsyb-map-zoom-btn--bottom"
                    @click="zoomOut"
                />
            </div>

            <UButton
                icon="i-lucide-map-pin"
                size="md"
                color="white"
                variant="solid"
                :loading="locating"
            />
        </div>

        <!-- 选中标记信息 -->
        <Transition
            enter-active-class="transition ease-out duration-200"
            enter-from-class="opacity-0 translate-y-2"
            enter-to-class="opacity-100 translate-y-0"
            leave-active-class="transition ease-in duration-150"
            leave-from-class="opacity-100 translate-y-0"
            leave-to-class="opacity-0 translate-y-2"
        >
            <div
                v-if="selectedMarker"
                class="absolute bottom-6 left-1/2 -translate-x-1/2 w-[calc(100%-2rem)] max-w-sm z-30"
            >
                <UCard variant="solid">
                    <div class="flex items-start gap-3 p-1">
                        <UAvatar
                            :src="selectedMarker.avatar"
                            :alt="selectedMarker.title"
                            size="md"
                            class="shrink-0"
                        />
                        <div class="flex-1 min-w-0">
                            <div class="flex items-center justify-between">
                                <h3 class="font-semibold text-neutral-900 dark:text-white text-sm truncate">
                                    {{ selectedMarker.title }}
                                </h3>
                                <UButton
                                    icon="i-lucide-x"
                                    variant="ghost"
                                    color="neutral"
                                    size="xs"
                                    class="-mr-2 -mt-2"
                                    @click="selectedMarker = null"
                                />
                            </div>
                            <p
                                class="mt-0.5 line-clamp-2 text-neutral-500 dark:text-neutral-400"
                            >
                                {{ selectedMarker.description }}
                            </p>

                            <div
                                v-if="selectedMarker.distance"
                                class="mt-2 flex items-center gap-2"
                            >
                                <UBadge
                                    size="xs"
                                    color="primary"
                                    variant="subtle"
                                    icon="i-lucide-navigation"
                                >
                                    {{
                                        formatDistance(selectedMarker.distance)
                                    }}
                                </UBadge>
                            </div>
                        </div>
                    </div>
                </UCard>
            </div>
        </Transition>

        <!-- 未配置提示 -->
        <div
            v-if="!isConfigured"
            class="absolute inset-0 z-10 flex items-center justify-center bg-neutral-50/50 backdrop-blur-sm pointer-events-none"
        >
            <UEmpty
                icon="i-lucide-map-off"
                title="地图未配置"
                description="请检查 NUXT_PUBLIC_AMAP_KEY 配置"
                variant="naked"
            />
        </div>
    </div>
</template>

<script setup lang="ts">
import { useGeolocation } from "~/composables/useGeolocation";
import { waitForAMap } from "~/plugins/01.amap.client";

export interface MapMarker {
    id: string;
    type: "user" | "group" | "moment" | "location" | "cluster";
    title: string;
    description: string;
    latitude: number;
    longitude: number;
    avatar?: string;
    distance?: number;
    data?: Record<string, unknown>;
}

const props = withDefaults(
    defineProps<{
        markers: MapMarker[];
        center: { latitude: number; longitude: number };
        zoom: number;
        interactive?: boolean;
        showAnchor?: boolean;
    }>(),
    {
        markers: () => [],
        center: () => ({ latitude: 39.9042, longitude: 116.4074 }),
        zoom: 13,
        interactive: false,
        showAnchor: true,
    },
);

const emit = defineEmits<{
    markerClick: [marker: MapMarker];
    mapClick: [coords: { latitude: number; longitude: number }];
    viewportChange: [
        payload: {
            center: { latitude: number; longitude: number };
            zoom: number;
            bounds: {
                north: number;
                south: number;
                east: number;
                west: number;
            };
        },
    ];
}>();

const config = useRuntimeConfig();
const feedback = useFeedbackStore();
const colorMode = useColorMode();
const geolocation = useGeolocation();
const { formatDistance } = useFormatters();

const mapContainer = ref<HTMLElement>();
const rootEl = ref<HTMLElement>();
const mapInstance = ref<AMap.Map | null>(null);
const markerInstances = ref<Map<string, AMap.Marker>>(new Map());
const loading = ref(true);
const locating = ref(false);
const mapReady = ref(false);
const selectedMarker = ref<MapMarker | null>(null);

const isConfigured = computed(() => !!config.public.amapKey);
const isInteractive = ref<boolean>(props.interactive);

const setInteractiveStatus = (enabled: boolean) => {
    const map = mapInstance.value as unknown as {
        setStatus?: (s: Record<string, boolean>) => void;
    } | null;
    map?.setStatus?.({
        scrollWheel: enabled,
        dragEnable: enabled,
        zoomEnable: enabled,
        doubleClickZoom: enabled,
        keyboardEnable: enabled,
        touchZoom: enabled,
    });
};

const initMap = async () => {
    if (!mapContainer.value || !isConfigured.value) {
        loading.value = false;
        return;
    }

    try {
        await waitForAMap();

        mapInstance.value = new AMap.Map(mapContainer.value, {
            center: [props.center.longitude, props.center.latitude],
            zoom: props.zoom,
            viewMode: "2D",
            mapStyle:
                colorMode.value === "dark"
                    ? "amap://styles/dark"
                    : "amap://styles/normal",
        });

        setInteractiveStatus(isInteractive.value);

        mapInstance.value.on("click", (e: unknown) => {
            const event = e as { lnglat: AMap.LngLat };
            emit("mapClick", {
                latitude: event.lnglat.getLat(),
                longitude: event.lnglat.getLng(),
            });
        });

        const emitViewport = () => {
            if (!mapInstance.value) return;
            const center = mapInstance.value.getCenter();
            const zoom = mapInstance.value.getZoom();
            const b = (
                mapInstance.value as unknown as {
                    getBounds: () => {
                        getNorthEast: () => AMap.LngLat;
                        getSouthWest: () => AMap.LngLat;
                    };
                }
            ).getBounds();
            emit("viewportChange", {
                center: {
                    latitude: center.getLat(),
                    longitude: center.getLng(),
                },
                zoom,
                bounds: {
                    north: b.getNorthEast().getLat(),
                    east: b.getNorthEast().getLng(),
                    south: b.getSouthWest().getLat(),
                    west: b.getSouthWest().getLng(),
                },
            });
        };
        mapInstance.value.on("moveend", emitViewport);
        mapInstance.value.on("zoomend", emitViewport);

        mapReady.value = true;
        updateMarkers();
    } catch (error) {
        console.error("[Map] Init failed:", error);
        feedback.error("地图加载失败");
    } finally {
        loading.value = false;
    }
};

const updateMarkers = () => {
    if (!mapInstance.value) return;
    markerInstances.value.forEach((m) => mapInstance.value?.remove(m));
    markerInstances.value.clear();

    props.markers.forEach((m) => {
        if (m.type === 'cluster') {
            addClusterMarker(m);
        } else {
            addMarker(m);
        }
    });
};

const addClusterMarker = (data: MapMarker) => {
    // 尝试从 data.data 中获取 count，或者从 title 中解析数字
    const clusterData = data.data as { count?: number } | undefined;
    const count = clusterData?.count || parseInt(data.title.replace(/\D/g, '')) || 0;
    
    const marker = new AMap.Marker({
        position: [data.longitude, data.latitude],
        title: data.title,
        extData: data,
        content: `<div class="inline-flex items-center justify-center select-none min-w-8 h-8 px-2 rounded-full text-xs font-bold bg-neutral-900/90 dark:bg-white/90 text-white dark:text-neutral-900 ring-2 ring-white dark:ring-neutral-900 shadow-lg transition-transform hover:scale-110 cursor-pointer">${count}</div>`,
        zIndex: 100,
    } as unknown as AMap.MarkerOptions);
    
    marker.on("click", () => {
        if (mapInstance.value) {
            // 点击聚合点：放大视图
            mapInstance.value.setZoomAndCenter(mapInstance.value.getZoom() + 2, [data.longitude, data.latitude]);
        }
    });
    
    mapInstance.value?.add(marker);
    markerInstances.value.set(data.id, marker);
};

const addMarker = (data: MapMarker) => {
    const marker = new AMap.Marker({
        position: [data.longitude, data.latitude],
        title: data.title,
        extData: data,
    });
    marker.on("click", () => {
        selectedMarker.value = data;
        emit("markerClick", data);
    });
    mapInstance.value?.add(marker);
    markerInstances.value.set(data.id, marker);
};

const zoomIn = () =>
    mapInstance.value?.setZoom(mapInstance.value.getZoom() + 1);
const zoomOut = () =>
    mapInstance.value?.setZoom(mapInstance.value.getZoom() - 1);

const locateMe = async () => {
    if (!mapInstance.value) return;
    locating.value = true;
    try {
        const pos = await geolocation.getPosition({ silent: false });
        if (pos) {
            mapInstance.value.setCenter([
                pos.coords.longitude,
                pos.coords.latitude,
            ]);
            feedback.success("定位成功");
        }
    } catch {
        feedback.error("定位失败");
    } finally {
        locating.value = false;
    }
};

watch(() => props.markers, updateMarkers, { deep: true });
watch(
    () => [props.center.latitude, props.center.longitude] as const,
    ([lat, lng]) => mapInstance.value?.setCenter([lng, lat]),
);
watch(
    () => props.zoom,
    (z) => mapInstance.value?.setZoom(z),
);
watch(
    () => colorMode.value,
    (mode) => {
        const map = mapInstance.value as unknown as {
            setMapStyle?: (style: string) => void;
        } | null;
        map?.setMapStyle?.(
            mode === "dark" ? "amap://styles/dark" : "amap://styles/normal",
        );
    },
);

onMounted(initMap);
onUnmounted(() => {
    document.removeEventListener("pointerdown", onOutsidePointerDown, true);
    mapInstance.value?.destroy();
    mapInstance.value = null;
});

const handleContainerClick = () => {
    if (!mapReady.value || isInteractive.value) return;
    isInteractive.value = true;
};

const onOutsidePointerDown = (e: PointerEvent) => {
    if (!isInteractive.value || !rootEl.value) return;
    if (rootEl.value.contains(e.target as Node)) return;
    isInteractive.value = false;
    selectedMarker.value = null;
};

watch(
    isInteractive,
    (enabled) => {
        if (!mapReady.value) return;
        setInteractiveStatus(enabled);
        document.removeEventListener("pointerdown", onOutsidePointerDown, true);
        if (enabled)
            document.addEventListener(
                "pointerdown",
                onOutsidePointerDown,
                true,
            );
    },
    { immediate: true },
);
</script>
