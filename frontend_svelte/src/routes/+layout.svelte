<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { ModeWatcher } from 'mode-watcher';
	import { Toaster } from '$lib/components/ui/sonner';
	import LoginModal from '$lib/components/common/login-modal.svelte';
	import { locationStore } from '$lib/stores/location.svelte';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { loadAMapPlugin } from '$lib/utils/amap';

	let { children } = $props();

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
			? { latitude: existing.latitude, longitude: existing.longitude }
			: await locationStore.requestLocation();
		if (!geoPoint) return;

		const resolved = await reverseGeocode(geoPoint.latitude, geoPoint.longitude);
		const name = resolved || locationStore.current?.location_name || '当前位置';
		locationStore.setLocation({
			latitude: geoPoint.latitude,
			longitude: geoPoint.longitude,
			location_name: name
		});

		if (mapAnchorStore.name === '南天门') {
			mapAnchorStore.setAnchor(geoPoint.latitude, geoPoint.longitude, name);
		}
	}

	onMount(() => {
		initLocation();
	});
</script>

<svelte:head>
	<link rel="icon" type="image/png" sizes="32x32" href="/logo_32.png" />
	<link rel="icon" type="image/png" sizes="16x16" href="/logo_16.png" />
	<link rel="apple-touch-icon" sizes="180x180" href="/logo_180.png" />
	<title>HotSpot - 发现附近的精彩</title>
</svelte:head>

<ModeWatcher />
<Toaster position="top-center" richColors />
<LoginModal />

{@render children()}
