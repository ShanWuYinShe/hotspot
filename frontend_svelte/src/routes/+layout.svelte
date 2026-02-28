<script lang="ts">
	import '../app.css';
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import { ModeWatcher } from 'mode-watcher';
	import { Toaster } from '$lib/components/ui/sonner';
	import LoginModal from '$lib/components/common/login-modal.svelte';
	import PageLoading from '$lib/components/common/page-loading.svelte';
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/auth.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { locationStore } from '$lib/stores/location.svelte';
	import { mapAnchorStore, DEFAULT_ANCHOR_NAME } from '$lib/stores/map-anchor.svelte';
	import { loadAMapPlugin } from '$lib/utils/amap';
	import { usersService } from '$lib/api/services/users';
	import { isSuccess } from '$lib/api/client';

	let { children } = $props();
	let isSyncingLocationToServer = false;
	let lastSyncedLocationSignature = '';

	function buildLocationSignature(
		userId: string,
		location: { point: { latitude: number; longitude: number }; location_name: string }
	): string {
		return `${userId}:${location.point.latitude.toFixed(6)}:${location.point.longitude.toFixed(6)}:${location.location_name}`;
	}

	async function syncLocationToServer(
		userId: string,
		location: { point: { latitude: number; longitude: number }; location_name: string }
	) {
		if (isSyncingLocationToServer) return;
		const signature = buildLocationSignature(userId, location);
		if (signature === lastSyncedLocationSignature) return;
		isSyncingLocationToServer = true;
		try {
			const response = await usersService.updateLocation({ location });
			if (isSuccess(response)) {
				lastSyncedLocationSignature = signature;
			}
		} catch {
			// 静默失败：避免阻塞主流程；后续登录态或位置变化时会再次触发同步
		} finally {
			isSyncingLocationToServer = false;
		}
	}

	async function reverseGeocode(lat: number, lng: number): Promise<string | null> {
		try {
			await loadAMapPlugin(['AMap.Geocoder']);
			const geocoder = new AMap.Geocoder();
			return await new Promise((resolve) => {
				geocoder.getAddress([lng, lat], (status: string, result: AMap.GeocodeResult) => {
					if (status === 'complete' && result.regeocode) {
						const poiName = result.regeocode.pois?.[0]?.name;
						const resolved = poiName || result.regeocode.formattedAddress || '当前位置';
						resolve(resolved);
						return;
					}
					resolve(null);
				});
			});
		} catch {
			return null;
		}
	}

	async function initLocation() {
		if (locationStore.permissionDenied) return;
		const existing = locationStore.current;
		const existingName = existing?.location_name;
		if (existingName && existingName !== '当前位置') return;

		const geoPoint = existing
			? { latitude: existing.point.latitude, longitude: existing.point.longitude }
			: await locationStore.requestLocation();
		if (!geoPoint) return;

		const resolved = await reverseGeocode(geoPoint.latitude, geoPoint.longitude);
		const name = resolved || locationStore.current?.location_name || '当前位置';
		locationStore.setLocation({
			point: { latitude: geoPoint.latitude, longitude: geoPoint.longitude },
			location_name: name
		});

		if (mapAnchorStore.name === DEFAULT_ANCHOR_NAME) {
			mapAnchorStore.setAnchor(geoPoint.latitude, geoPoint.longitude, name);
		}
	}

	$effect(() => {
		const isAuthenticated = authStore.isAuthenticated;
		const userId = authStore.currentUserId;
		const current = locationStore.current;
		if (!isAuthenticated || !userId || !current) return;
		const location = {
			point: {
				latitude: current.point.latitude,
				longitude: current.point.longitude
			},
			location_name: current.location_name || '当前位置'
		};
		void syncLocationToServer(userId, location);
	});

	onMount(() => {
		// 处理 URL 中的 redirect 参数（例如外部链接 /?redirect=/profile）
		const redirectParam = page.url.searchParams.get('redirect');
		if (redirectParam) {
			if (authStore.isAuthenticated) {
				// 已登录：直接跳转到目标页面
				goto(redirectParam, { replaceState: true });
			} else {
				// 未登录：保存目标 URL，登录后自动跳转
				uiStore.saveRedirectUrl(redirectParam);
			}
		}

		initLocation();
		// 注：WebSocket 在 hooks.client.ts 中初始化，无需在组件中管理
	});
</script>

<svelte:head>
	<link rel="icon" type="image/svg+xml" href="/favicon.svg" />
	<link rel="icon" type="image/png" sizes="32x32" href="/favicon-32x32.png" />
	<link rel="icon" type="image/png" sizes="16x16" href="/favicon-16x16.png" />
	<link rel="icon" type="image/png" sizes="192x192" href="/android-chrome-192x192.png" />
	<link rel="icon" type="image/png" sizes="512x512" href="/android-chrome-512x512.png" />
	<link rel="apple-touch-icon" sizes="180x180" href="/apple-touch-icon.png" />
	<link rel="shortcut icon" href="/favicon.ico" />
	<title>HotSpot - 发现附近的精彩</title>
</svelte:head>

<ModeWatcher />
<Toaster position="top-center" richColors />
<LoginModal />
<PageLoading />

{@render children()}
