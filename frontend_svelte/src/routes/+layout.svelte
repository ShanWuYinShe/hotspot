<script lang="ts">
    import "../app.css";
    import { page } from "$app/state";
    import { onMount } from "svelte";
    import { ModeWatcher } from "mode-watcher";
    import { getCurrentLocationLabel, initLocale } from "$lib/i18n";
    import { Toaster } from "$lib/components/ui/sonner";
    import LoginModal from "$lib/components/common/login-modal.svelte";
    import MobileCreateSheet from "$lib/components/common/mobile-create-sheet.svelte";
    import SearchModal from "$lib/components/common/search-modal.svelte";
    import CreateMomentModal from "$lib/components/common/create-moment-modal.svelte";
    import CreateGroupModal from "$lib/components/common/create-group-modal.svelte";
    import PageLoading from "$lib/components/common/page-loading.svelte";
    import { goto } from "$app/navigation";
    import { authStore } from "$lib/stores/auth.svelte";
    import { themeStore } from "$lib/stores/theme.svelte";
    import { normalizeRedirectPath, uiStore } from "$lib/stores/ui.svelte";
    import { locationStore } from "$lib/stores/location.svelte";
    import {
        mapAnchorStore,
        DEFAULT_ANCHOR_NAME,
    } from "$lib/stores/map-anchor.svelte";
    import { loadAMap, loadAMapPlugin } from "$lib/utils/amap";
    import { toGcjLngLat } from "$lib/utils/geo";
    import { usersService } from "$lib/api/services/users";
    import { isSuccess } from "$lib/api/client";

    let { children } = $props();
    let isSyncingLocationToServer = false;
    let lastSyncedLocationSignature = "";
    let lastSyncedUserId: string | null = null;
    let hasBootstrappedLocation = false;
    const LOCATION_MAX_AGE_MS = 1 * 60 * 1000; // 1 分钟视为位置仍然新鲜
    const LOCATION_REFRESH_INTERVAL_MS = 1 * 60 * 1000; // 每 1 分钟刷新一次真实位置
    const MAP_PRELOAD_ROUTES = ["/map", "/groups", "/moments"];
    let locationRefreshTimer: number | null = null;

    function buildLocationSignature(
        userId: string,
        location: {
            point: { latitude: number; longitude: number };
            location_name: string;
        },
    ): string {
        return `${userId}:${location.point.latitude.toFixed(6)}:${location.point.longitude.toFixed(6)}:${location.location_name}`;
    }

    async function syncLocationToServer(
        userId: string,
        location: {
            point: { latitude: number; longitude: number };
            location_name: string;
        },
    ) {
        if (isSyncingLocationToServer) return;
        const signature = buildLocationSignature(userId, location);
        if (userId === lastSyncedUserId && signature === lastSyncedLocationSignature) return;
        isSyncingLocationToServer = true;
        try {
            const response = await usersService.updateLocation({ location });
            if (isSuccess(response)) {
                lastSyncedUserId = userId;
                lastSyncedLocationSignature = signature;
            }
        } catch {
            // 静默失败：避免阻塞主流程；后续登录态或位置变化时会再次触发同步
        } finally {
            isSyncingLocationToServer = false;
        }
    }

    async function reverseGeocode(
        lat: number,
        lng: number,
    ): Promise<string | null> {
        try {
            await loadAMapPlugin(["AMap.Geocoder"]);
            const geocoder = new AMap.Geocoder();
            const [gcjLng, gcjLat] = toGcjLngLat({
                latitude: lat,
                longitude: lng,
            });
            return await new Promise((resolve) => {
                geocoder.getAddress(
                    [gcjLng, gcjLat],
                    (status: string, result: AMap.GeocodeResult) => {
                        if (status === "complete" && result.regeocode) {
                            const poiName = result.regeocode.pois?.[0]?.name;
                            const resolved =
                                poiName ||
                                result.regeocode.formattedAddress ||
                                getCurrentLocationLabel();
                            resolve(resolved);
                            return;
                        }
                        resolve(null);
                    },
                );
            });
        } catch {
            return null;
        }
    }

    function isLocationFresh(): boolean {
        const lastUpdated = locationStore.lastUpdated;
        if (!lastUpdated) return false;
        const timestamp = Date.parse(lastUpdated);
        if (Number.isNaN(timestamp)) return false;
        return Date.now() - timestamp < LOCATION_MAX_AGE_MS;
    }

    function shouldPreloadAMap(pathname: string): boolean {
        return MAP_PRELOAD_ROUTES.some(
            (route) => pathname === route || pathname.startsWith(`${route}/`),
        );
    }

    function syncAnchorFromLocation(location: {
        point: { latitude: number; longitude: number };
        location_name: string;
    }) {
        if (mapAnchorStore.name !== DEFAULT_ANCHOR_NAME) return;
        mapAnchorStore.setAnchor(
            location.point.latitude,
            location.point.longitude,
            location.location_name || '',
        );
    }

    async function initLocation(force: boolean) {
        if (locationStore.permissionDenied) return;

        const existing = locationStore.current;
        if (existing && (force || isLocationFresh())) {
            syncAnchorFromLocation({
                point: {
                    latitude: existing.point.latitude,
                    longitude: existing.point.longitude,
                },
                location_name: existing.location_name || '',
            });
            return;
        }

        const geoPoint = await locationStore.requestLocation();
        if (!geoPoint) return;

        const resolved = await reverseGeocode(
            geoPoint.latitude,
            geoPoint.longitude,
        );
        const location = {
            point: {
                latitude: geoPoint.latitude,
                longitude: geoPoint.longitude,
            },
            location_name:
                resolved || locationStore.current?.location_name || '',
        };
        locationStore.setLocation(location);
        syncAnchorFromLocation(location);
    }

    $effect(() => {
        const isAuthenticated = authStore.isAuthenticated;
        const userId = authStore.currentUserId;

        if (!isAuthenticated || !userId) {
            lastSyncedUserId = null;
            lastSyncedLocationSignature = "";
            return;
        }

        if (userId !== lastSyncedUserId) {
            lastSyncedUserId = null;
            lastSyncedLocationSignature = "";
        }

        const current = locationStore.current;
        if (!current || !hasBootstrappedLocation) return;

        const location = {
            point: {
                latitude: current.point.latitude,
                longitude: current.point.longitude,
            },
            location_name: current.location_name || '',
        };
        void syncLocationToServer(userId, location);
    });

    onMount(() => {
        initLocale();
        themeStore.init();

        const redirectParam = page.url.searchParams.get("redirect");
        const safeRedirect = normalizeRedirectPath(redirectParam);
        if (redirectParam) {
            if (safeRedirect) {
                if (authStore.isAuthenticated) {
                    goto(safeRedirect, { replaceState: true });
                } else {
                    uiStore.saveRedirectUrl(safeRedirect);
                }
            } else {
                uiStore.clearRedirectUrl();
            }
        }

        const pathname = page.url.pathname;
        if (shouldPreloadAMap(pathname)) {
            void loadAMap().catch(() => {
                // 忽略加载失败：缺少 Key 或网络问题时保持降级体验
            });
        }

        void initLocation(true).finally(() => {
            hasBootstrappedLocation = true;
        });
        locationRefreshTimer = window.setInterval(() => {
            void initLocation(false);
        }, LOCATION_REFRESH_INTERVAL_MS);
        return () => {
            if (locationRefreshTimer !== null) {
                window.clearInterval(locationRefreshTimer);
                locationRefreshTimer = null;
            }
        };
    });
</script>

<svelte:head>
    <link rel="icon" type="image/svg+xml" href="/favicon.svg" />
    <link rel="icon" type="image/png" sizes="32x32" href="/favicon-32x32.png" />
    <link rel="icon" type="image/png" sizes="16x16" href="/favicon-16x16.png" />
    <link
        rel="icon"
        type="image/png"
        sizes="192x192"
        href="/android-chrome-192x192.png"
    />
    <link
        rel="icon"
        type="image/png"
        sizes="512x512"
        href="/android-chrome-512x512.png"
    />
    <link rel="apple-touch-icon" sizes="180x180" href="/apple-touch-icon.png" />
    <link rel="shortcut icon" href="/favicon.ico" />
    <title>HotSpot - 发现附近的精彩</title>
</svelte:head>

<ModeWatcher />
<Toaster position="top-center" richColors />
<LoginModal />
<MobileCreateSheet />
<SearchModal open={uiStore.showSearchModal} onOpenChange={(open) => !open && uiStore.closeSearchModal()} />
<CreateMomentModal open={uiStore.showCreateMoment} onOpenChange={(open) => !open && uiStore.closeCreateMoment()} />
<CreateGroupModal open={uiStore.showCreateGroup} onOpenChange={(open) => !open && uiStore.closeCreateGroup()} />
<PageLoading />

{@render children()}
