<script lang="ts">
	import { onDestroy, tick } from 'svelte';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { locationStore } from '$lib/stores/location.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { loadAMap, loadAMapPlugin } from '$lib/utils/amap';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent } from '$lib/components/ui/card';
	import MapPin from 'lucide-svelte/icons/map-pin';
	import Navigation from 'lucide-svelte/icons/navigation';
	import Loader2 from 'lucide-svelte/icons/loader-2';
	import Crosshair from 'lucide-svelte/icons/crosshair';

	import SlidersHorizontal from 'lucide-svelte/icons/sliders-horizontal';

	interface Props {
		open: boolean;
		onOpenChange?: (open: boolean) => void;
	}

	let { open = $bindable(), onOpenChange }: Props = $props();

	let mapContainer = $state<HTMLDivElement | null>(null);
	let map: AMap.Map | null = null;
	let geocoder: AMap.Geocoder | null = null;
	let isLoadingMap = $state(false);
	let isMapAvailable = $state(true);
	let isLocating = $state(false);
	let locationName = $state(mapAnchorStore.name);
	let latitude = $state(mapAnchorStore.latitude);
	let longitude = $state(mapAnchorStore.longitude);
	let hasInit = $state(false);
	let hasSyncedFromAnchor = $state(false);
	let debounceTimer: ReturnType<typeof setTimeout> | null = null;
	let searchRadius = $state(mapAnchorStore.radius);

	type MapClickEvent = {
		lnglat?: AMap.LngLat;
	};

	async function ensureGeocoder() {
		if (geocoder) return;
		await loadAMapPlugin(['AMap.Geocoder']);
		geocoder = new AMap.Geocoder();
	}

	async function reverseGeocode(lat: number, lng: number): Promise<string> {
		try {
			await ensureGeocoder();
			const currentGeocoder = geocoder;
			if (!currentGeocoder) {
				locationName = '自定义位置';
				return '自定义位置';
			}
			return await new Promise<string>((resolve) => {
				currentGeocoder.getAddress([lng, lat], (status: string, result: AMap.GeocodeResult) => {
					if (status === 'complete' && result.regeocode) {
						const poiName = result.regeocode.pois?.[0]?.name;
						const resolved = poiName || result.regeocode.formattedAddress || '自定义位置';
						locationName = resolved;
						resolve(resolved);
						return;
					}
					locationName = '自定义位置';
					resolve('自定义位置');
				});
			});
		} catch {
			locationName = '自定义位置';
			return '自定义位置';
		}
	}

	function handleMapChanged() {
		if (!map) return;
		const center = map.getCenter();
		latitude = center.getLat();
		longitude = center.getLng();

		if (debounceTimer) clearTimeout(debounceTimer);
		debounceTimer = setTimeout(() => {
			void reverseGeocode(latitude, longitude);
		}, 300);
	}

async function handleDoubleClick(event: MapClickEvent) {
	if (!map) return;
	const lnglat = event.lnglat ?? map.getCenter();
	const nextLat = lnglat.getLat();
	const nextLng = lnglat.getLng();
	latitude = nextLat;
	longitude = nextLng;
	map.setCenter([nextLng, nextLat]);
	await reverseGeocode(nextLat, nextLng);
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
				viewMode: '2D',
				doubleClickZoom: false,
				resizeEnable: true
			});

			map.on('moveend', handleMapChanged);
			map.on('zoomend', handleMapChanged);
		map.on('dblclick', handleDoubleClick);
			await reverseGeocode(latitude, longitude);
			hasInit = true;
		} catch {
			isMapAvailable = false;
		} finally {
			isLoadingMap = false;
		}
	}

	async function handleUseCurrentLocation() {
		isLocating = true;
		const geoPoint = await locationStore.requestLocation();
		if (geoPoint) {
			latitude = geoPoint.latitude;
			longitude = geoPoint.longitude;
			map?.setZoomAndCenter(15, [longitude, latitude]);
			const resolvedName = await reverseGeocode(latitude, longitude);
			locationStore.setLocation({
				latitude,
				longitude,
				location_name: resolvedName
			});
		} else {
			toast.error('无法获取当前位置');
		}
		isLocating = false;
	}

	function handleConfirm() {
		mapAnchorStore.setAnchor(latitude, longitude, locationName || undefined, searchRadius);
		toast.success(`浏览锚点已更新`);
		handleClose();
	}

	function handleClose() {
		open = false;
		onOpenChange?.(false);
	}

	$effect(() => {
		if (open) {
				if (!hasSyncedFromAnchor) {
				latitude = mapAnchorStore.latitude;
				longitude = mapAnchorStore.longitude;
				locationName = mapAnchorStore.name;
				searchRadius = mapAnchorStore.radius;

				if (!hasInit) {
					initMap();
				} else {
					map?.setZoomAndCenter(14, [longitude, latitude]);
				}
				hasSyncedFromAnchor = true;
			}
		} else {
			hasSyncedFromAnchor = false;
		}
	});

	onDestroy(() => {
		map?.destroy();
		map = null;
	});
</script>

<Dialog.Root bind:open onOpenChange={(value) => { open = value; onOpenChange?.(value); }}>
	<Dialog.Content class="flex h-[85vh] flex-col sm:max-w-4xl">
		<Dialog.Header>
			<Dialog.Title class="flex items-center gap-2">
				<div class="flex h-10 w-10 items-center justify-center rounded-xl bg-linear-to-br from-primary/20 to-primary/10">
					<MapPin class="h-5 w-5 text-primary" />
				</div>
				<div>
					<div>地图选择锚点</div>
					<div class="text-sm font-normal text-muted-foreground">拖动地图选择浏览位置</div>
				</div>
			</Dialog.Title>
		</Dialog.Header>

		<div class="flex flex-1 flex-col min-h-0 gap-4 py-4">
			<div class="relative flex-1 overflow-hidden rounded-2xl border bg-muted">
				<div class="h-full w-full touch-none" bind:this={mapContainer}></div>

				{#if !isMapAvailable}
					<div class="absolute inset-0 z-20 flex items-center justify-center bg-background/80 backdrop-blur-sm">
						<div class="text-center">
							<p class="font-medium">地图未配置</p>
							<p class="text-sm text-muted-foreground">请设置 VITE_AMAP_KEY</p>
						</div>
					</div>
				{/if}

				{#if isLoadingMap}
					<div class="absolute inset-0 z-20 flex items-center justify-center bg-background/70">
						<Loader2 class="h-6 w-6 animate-spin text-muted-foreground" />
					</div>
				{/if}

				<div class="pointer-events-none absolute left-1/2 top-1/2 z-10 -translate-x-1/2 -translate-y-full">
					<div class="flex flex-col items-center gap-2">
						<div class="flex h-10 w-10 items-center justify-center rounded-full bg-primary/10 text-primary">
							<Crosshair class="h-5 w-5" />
						</div>
						<div class="rounded-full bg-background/90 px-3 py-1 text-xs text-muted-foreground shadow">
							浏览中心
						</div>
					</div>
				</div>
			</div>

			<Card class="border-0 bg-linear-to-r from-primary/5 to-primary/10">
				<CardContent class="flex flex-col gap-4 p-4 sm:flex-row sm:items-center sm:justify-between">
					<div class="min-w-0">
						<div class="text-xs text-muted-foreground">选中位置</div>
						<div class="truncate text-lg font-semibold">
							{locationName || '拖动地图选择位置'}
						</div>
					<div class="text-xs text-muted-foreground">
							{latitude.toFixed(6)}, {longitude.toFixed(6)}
						</div>
					<div class="mt-1 text-xs text-muted-foreground">
						双击地图可快速选中位置，确认后才会更新锚点
					</div>
					</div>
					<div class="flex items-center gap-2">
						<Button variant="outline" class="gap-2" onclick={handleUseCurrentLocation} disabled={isLocating}>
							{#if isLocating}
								<Loader2 class="h-4 w-4 animate-spin" />
								定位中
							{:else}
								<Navigation class="h-4 w-4" />
								定位到当前
							{/if}
						</Button>
						<Button class="gap-2" onclick={handleConfirm} disabled={!locationName}>
							设为锚点
						</Button>
					</div>
				</CardContent>
			</Card>

			<Card class="border-0 bg-card/50">
				<CardContent class="flex flex-col gap-3 p-4">
					<div class="flex items-center justify-between">
						<div class="flex items-center gap-2 text-sm font-medium">
							<SlidersHorizontal class="h-4 w-4 text-primary" />
							<span>搜索范围</span>
						</div>
						<span class="text-sm font-medium text-primary">{searchRadius} km</span>
					</div>
					<input
						type="range"
						min="1"
						max="50"
						step="1"
						class="h-2 w-full cursor-pointer appearance-none rounded-full bg-muted accent-primary"
						bind:value={searchRadius}
					/>
					<div class="flex justify-between text-xs text-muted-foreground">
						<span>1km</span>
						<span>50km</span>
					</div>
				</CardContent>
			</Card>
		</div>
	</Dialog.Content>
</Dialog.Root>
