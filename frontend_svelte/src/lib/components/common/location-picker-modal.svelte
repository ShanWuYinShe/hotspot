<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { locationStore } from '$lib/stores/location.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { loadAMap, loadAMapPlugin } from '$lib/utils/amap';
	import {
		getAnchorHistory,
		addToAnchorHistory,
		clearAnchorHistory,
		type AnchorHistoryItem
	} from '$lib/utils/anchor-history';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent } from '$lib/components/ui/card';
	import MapPin from 'lucide-svelte/icons/map-pin';
	import Navigation from 'lucide-svelte/icons/navigation';
	import PersonStanding from 'lucide-svelte/icons/person-standing';
	import Loader2 from 'lucide-svelte/icons/loader-2';
	import ChevronUp from 'lucide-svelte/icons/chevron-up';
	import ChevronDown from 'lucide-svelte/icons/chevron-down';
	import SlidersHorizontal from 'lucide-svelte/icons/sliders-horizontal';
	import Share2 from 'lucide-svelte/icons/share-2';
	import Search from 'lucide-svelte/icons/search';
	import X from 'lucide-svelte/icons/x';
	import Clock from 'lucide-svelte/icons/clock';
	import Trash2 from 'lucide-svelte/icons/trash-2';
	import CreateLocationShareModal from './create-location-share-modal.svelte';
	import AnchorSummaryCard from './anchor-summary-card.svelte';

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
	let isMapInteracting = $state(false);
	let locationName = $state(mapAnchorStore.name);
	let latitude = $state(mapAnchorStore.latitude);
	let longitude = $state(mapAnchorStore.longitude);
	const REVERSE_GEOCODE_IDLE_MS = 800;
	const MIN_LOCATION_DELTA = 0.00005;
	let lastResolvedLat = mapAnchorStore.latitude;
	let lastResolvedLng = mapAnchorStore.longitude;
	let reverseGeocodeTimer: ReturnType<typeof setTimeout> | null = null;
	let skipNextReverseGeocode = false;
	let mapMoveFrame: number | null = null;
	let hasInit = $state(false);
	let hasSyncedFromAnchor = $state(false);
	const LONG_PRESS_DURATION_MS = 700;
	const GESTURE_HINT_DURATION_MS = 4000;
	const GESTURE_HINT_COMBINED = '双击、右键或长按地图可快速定位';
	const GESTURE_HINT_TOUCH = '长按地图可快速定位';
	const GESTURE_HINT_POINTER = '双击或右键地图可快速定位';

	function getGestureHintMessage(): string {
		if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
			return GESTURE_HINT_COMBINED;
		}
		const prefersCoarsePointer = window.matchMedia('(pointer: coarse)').matches;
		const prefersHover = window.matchMedia('(hover: hover)').matches;
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
	let placementHintTimer: ReturnType<typeof setTimeout> | null = null;
	let gestureHintTimer: ReturnType<typeof setTimeout> | null = null;
	let longPressTimer: ReturnType<typeof setTimeout> | null = null;
	let lastTouchLngLat: { lat: number; lng: number } | null = null;
	let touchMovedDuringLongPress = false;
	let detachTouchListeners: (() => void) | null = null;
	let detachContextMenuListener: (() => void) | null = null;

	let searchRadius = $state(mapAnchorStore.radius);
	let isBottomSheetExpanded = $state(false);
	let showShareModal = $state(false);

	// 搜索相关状态
	let showSearch = $state(false);
	let searchText = $state('');
	let isSearching = $state(false);
	let searchInput = $state<HTMLInputElement | null>(null);
	let searchResults = $state.raw<Array<{ name: string; address: string; latitude: number; longitude: number }>>([]);
	let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null;
	let placeSearch: AMap.PlaceSearch | null = null;

	// 锚点历史
	let anchorHistory = $state.raw<AnchorHistoryItem[]>([]);

	const isAnchorMode = $derived(mode === 'anchor');
	const modalTitle = $derived(isAnchorMode ? '设置浏览锚点' : '选择位置');
	const confirmButtonText = $derived(isAnchorMode ? '设为浏览锚点' : '使用此位置');
	const confirmSuccessText = $derived(isAnchorMode ? '浏览锚点已更新' : '位置已选定');
	const selectionDescription = $derived(
		isAnchorMode ? '拖动地图或搜索后，确认才会覆盖锚点' : '拖动地图或搜索后，确认用于当前操作'
	);
	const hintAfterConfirm = $derived(
		isAnchorMode ? `${gestureHintMessage}，确认后才会更新锚点` : `${gestureHintMessage}，确认后将用于当前操作`
	);

	type MapClickEvent = {
		lnglat?: AMap.LngLat;
	};

	const PLACEMENT_HINT_DEFAULT = '已定位到该位置';

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

	async function moveToCoordinate(lat: number, lng: number, options?: { hintMessage?: string }) {
		cancelReverseGeocodeTimer();
		skipNextReverseGeocode = true;
		latitude = lat;
		longitude = lng;
		map?.setCenter([lng, lat]);
		await reverseGeocode(lat, lng);
		hideGestureHint();
		showPlacementHint(options?.hintMessage ?? PLACEMENT_HINT_DEFAULT);
	}

	function getLngLatFromTouch(event: TouchEvent): { lat: number; lng: number } | null {
		if (!map || !mapContainer) return null;
		const touch = event.touches[0] ?? event.changedTouches[0];
		if (!touch) return null;
		const rect = mapContainer.getBoundingClientRect();
		const pixel = { x: touch.clientX - rect.left, y: touch.clientY - rect.top };
		const lnglat = (map as unknown as { containerToLngLat: (pixel: { x: number; y: number }) => AMap.LngLat }).containerToLngLat(pixel);
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
			void moveToCoordinate(lastTouchLngLat.lat, lastTouchLngLat.lng, { hintMessage: '已通过长按定位到该位置' });
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
		mapContainer.addEventListener('touchstart', handleTouchStart, { passive: true });
		mapContainer.addEventListener('touchmove', handleTouchMove, { passive: true });
		mapContainer.addEventListener('touchend', handleTouchEnd, { passive: true });
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

	// ========== 搜索功能 ==========

	async function toggleSearch() {
		showSearch = !showSearch;
		if (showSearch) {
			searchText = '';
			searchResults = [];
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

		isSearching = true;
		searchResults = [];
		try {
			await ensurePlaceSearch();
			if (!placeSearch) return;

			const results = await new Promise<
				Array<{ name: string; address: string; latitude: number; longitude: number }>
			>((resolve) => {
				placeSearch!.searchNearBy(
					keyword,
					[longitude, latitude],
					50000, // 50km 搜索半径
					(status: string, result: AMap.PlaceSearchResult) => {
						if (status === 'complete' && result.poiList?.pois?.length) {
							resolve(
								result.poiList.pois.map((poi) => ({
									name: poi.name,
									address: [poi.adname, poi.address].filter(Boolean).join(' '),
									latitude: poi.location.getLat(),
									longitude: poi.location.getLng()
								}))
							);
						} else {
							resolve([]);
						}
					}
				);
			});

			if (results.length > 0) {
				searchResults = results;
			} else {
				toast.error('未找到该地点，请换个关键词试试');
			}
		} catch {
			toast.error('搜索失败，请稍后重试');
		} finally {
			isSearching = false;
		}
	}

	function selectSearchResult(result: { name: string; address: string; latitude: number; longitude: number }) {
		cancelReverseGeocodeTimer();
		skipNextReverseGeocode = true;
		latitude = result.latitude;
		longitude = result.longitude;
		locationName = result.name;
		lastResolvedLat = result.latitude;
		lastResolvedLng = result.longitude;
		map?.setCenter([result.longitude, result.latitude]);
		showSearch = false;
		searchText = '';
		searchResults = [];
	}

	function selectHistoryItem(item: AnchorHistoryItem) {
		cancelReverseGeocodeTimer();
		skipNextReverseGeocode = true;
		latitude = item.latitude;
		longitude = item.longitude;
		locationName = item.locationName;
		lastResolvedLat = item.latitude;
		lastResolvedLng = item.longitude;
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
				locationName = '自定义位置';
				return '自定义位置';
			}
			return await new Promise<string>((resolve) => {
				currentGeocoder.getAddress([lng, lat], (status: string, result: AMap.GeocodeResult) => {
					if (status === 'complete' && result.regeocode) {
						const poiName = result.regeocode.pois?.[0]?.name;
						const resolved = poiName || result.regeocode.formattedAddress || '自定义位置';
						locationName = resolved;
						lastResolvedLat = lat;
						lastResolvedLng = lng;
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
	function cancelReverseGeocodeTimer() {
		if (reverseGeocodeTimer) {
			clearTimeout(reverseGeocodeTimer);
			reverseGeocodeTimer = null;
		}
	}

	function hasMeaningfulMovement(lat: number, lng: number) {
		return Math.abs(lat - lastResolvedLat) > MIN_LOCATION_DELTA || Math.abs(lng - lastResolvedLng) > MIN_LOCATION_DELTA;
	}

	function scheduleReverseGeocode() {
		cancelReverseGeocodeTimer();
		reverseGeocodeTimer = setTimeout(() => {
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
		await moveToCoordinate(nextLat, nextLng, { hintMessage: '已通过双击定位到该位置' });
	}

	async function handleRightClick(event: MapClickEvent) {
		if (!map) return;
		const lnglat = event.lnglat ?? map.getCenter();
		const nextLat = lnglat.getLat();
		const nextLng = lnglat.getLng();
		await moveToCoordinate(nextLat, nextLng, { hintMessage: '已通过右键定位到该位置' });
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
		registerTouchListeners();
		registerContextMenuBlocker();

		map.on('movestart', () => {
			isMapInteracting = true;
			cancelReverseGeocodeTimer();
		});
		map.on('moveend', () => {
			isMapInteracting = false;
			handleMapChanged();
		});
		map.on('zoomstart', () => {
			isMapInteracting = true;
			cancelReverseGeocodeTimer();
		});
		map.on('zoomend', () => {
			isMapInteracting = false;
			handleMapChanged();
		});
		map.on('mapmove', handleLiveMapMove);
		map.on('dblclick', handleDoubleClick);
		map.on('rightclick', handleRightClick);
			await reverseGeocode(latitude, longitude);
			hasInit = true;
	} catch {
		// 地图实例可能已创建但后续操作（如逆地理编码）失败，需释放资源
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
		const geoPoint = await locationStore.requestLocation();
		if (geoPoint) {
			latitude = geoPoint.latitude;
			longitude = geoPoint.longitude;
			map?.setZoomAndCenter(15, [longitude, latitude]);
			const resolvedName = await reverseGeocode(latitude, longitude);
			locationStore.setLocation({
				point: {
					latitude,
					longitude
				},
				location_name: resolvedName
			});
		} else {
			toast.error('无法获取当前位置');
		}
		isLocating = false;
	}

	async function handleBackToBrowseAnchor() {
		const anchorLat = mapAnchorStore.latitude;
		const anchorLng = mapAnchorStore.longitude;
		cancelReverseGeocodeTimer();
		skipNextReverseGeocode = true;
		latitude = anchorLat;
		longitude = anchorLng;
		map?.setZoomAndCenter(14, [anchorLng, anchorLat]);
		if (mapAnchorStore.name) {
			locationName = mapAnchorStore.name;
			lastResolvedLat = anchorLat;
			lastResolvedLng = anchorLng;
		} else {
			await reverseGeocode(anchorLat, anchorLng);
		}
		hideGestureHint();
		showPlacementHint('已回到浏览锚点位置');
	}

	function handleConfirm() {
		const normalizedName = locationName.trim() || `${latitude.toFixed(6)}, ${longitude.toFixed(6)}`;
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
		map?.destroy();
		map = null;
		isMapAvailable = true;
		hasInit = false;
		void initMap();
	}

	function handleClose() {
		open = false;
		onOpenChange?.(false);
	}

	$effect(() => {
		if (open) {
			if (!gestureHintVisible) {
				showGestureHint();
			}
			if (!hasSyncedFromAnchor) {
				const fallbackName = mapAnchorStore.name || mapAnchorStore.displayName;
				latitude = initialLocation?.latitude ?? mapAnchorStore.latitude;
				longitude = initialLocation?.longitude ?? mapAnchorStore.longitude;
				locationName = initialLocation?.locationName?.trim() || fallbackName;
				searchRadius = initialLocation?.radius ?? mapAnchorStore.radius;
				anchorHistory = isAnchorMode ? getAnchorHistory() : [];
				showSearch = false;
				searchText = '';
				searchResults = [];

				if (!hasInit) {
					initMap();
				} else {
					map?.setZoomAndCenter(14, [longitude, latitude]);
				}
				hasSyncedFromAnchor = true;
			}
		} else {
			hasSyncedFromAnchor = false;
			hideGestureHint();
			placementHintVisible = false;
		}
	});

	// 组件卸载时清理资源
	onMount(() => {
		return () => {
			map?.destroy();
			map = null;
			detachTouchListeners?.();
			detachContextMenuListener?.();
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

</script>

<style>
	/* 优化滑动条样式 */
	input[type='range'] {
		-webkit-appearance: none;
		appearance: none;
	}

	input[type='range']::-webkit-slider-thumb {
		-webkit-appearance: none;
		appearance: none;
		width: 20px;
		height: 20px;
		border-radius: 50%;
		background: hsl(var(--primary));
		cursor: pointer;
		box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
		transition: all 0.2s;
	}

	input[type='range']::-webkit-slider-thumb:hover {
		transform: scale(1.1);
		box-shadow: 0 4px 8px rgba(0, 0, 0, 0.15);
	}

	input[type='range']::-webkit-slider-thumb:active {
		transform: scale(1.2);
	}

	input[type='range']::-moz-range-thumb {
		width: 20px;
		height: 20px;
		border-radius: 50%;
		background: hsl(var(--primary));
		cursor: pointer;
		border: none;
		box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
		transition: all 0.2s;
	}

	input[type='range']::-moz-range-thumb:hover {
		transform: scale(1.1);
		box-shadow: 0 4px 8px rgba(0, 0, 0, 0.15);
	}

	input[type='range']::-moz-range-thumb:active {
		transform: scale(1.2);
	}

	.center-pin {
		position: relative;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.2rem;
		pointer-events: none;
		transition: transform 150ms ease;
	}

	.center-pin-icon {
		width: 40px;
		height: 43px;
		display: flex;
		align-items: center;
		justify-content: center;
		filter: drop-shadow(0 2px 4px rgba(10, 28, 103, 0.22));
	}

	.center-pin-svg {
		width: 100%;
		height: 100%;
	}

	.center-pin.dragging {
		transform: translateY(-5px);
	}
</style>

<Dialog.Root bind:open onOpenChange={(value) => { open = value; onOpenChange?.(value); }}>
	<Dialog.Content class="flex h-[90dvh] max-h-[90dvh] flex-col p-0 sm:h-[85vh] sm:max-w-4xl sm:rounded-3xl sm:border sm:border-border/60 sm:bg-card/95 sm:p-6" hideClose>

		<!-- 移动端简化标题 -->
		<div class="flex items-center justify-between border-b border-border/60 bg-card/95 px-4 py-2 backdrop-blur sm:hidden">
			<div class="flex items-center gap-2">
				<MapPin class="h-5 w-5 text-primary" />
				<span class="font-semibold">{modalTitle}</span>
			</div>
			<div class="flex items-center gap-1">
				<Button variant="ghost" size="sm" class="h-8 w-8 p-0" onclick={toggleSearch}>
					<Search class="h-4 w-4" />
					<span class="sr-only">搜索</span>
				</Button>
				<Button variant="ghost" size="sm" class="h-8 w-8 p-0" onclick={handleClose}>
					<span class="sr-only">关闭</span>
					<svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
					</svg>
				</Button>
			</div>
		</div>

		<!-- 地图容器 -->
		<div class="relative flex-1 overflow-hidden sm:rounded-3xl sm:border sm:border-border/60">
			<div class="h-full w-full touch-none" bind:this={mapContainer}></div>

		<!-- 桌面端地图工具栏 -->
		<div class="pointer-events-none absolute right-4 top-4 z-30 hidden flex-col gap-2 sm:flex">
			<div class="pointer-events-auto flex flex-col gap-2">
				<Button
					size="icon"
					variant="outline"
					class="h-10 w-10 rounded-2xl border-border/60 bg-card/90 shadow-md backdrop-blur"
					onclick={toggleSearch}
					aria-label="搜索地点"
				>
					<Search class="h-4 w-4" />
				</Button>
				<Button
					size="icon"
					variant="outline"
					class="h-10 w-10 rounded-2xl border-border/60 bg-card/90 shadow-md backdrop-blur"
					onclick={handleUseCurrentLocation}
					disabled={isLocating}
					aria-label="定位到当前位置"
				>
					{#if isLocating}
						<Loader2 class="h-4 w-4 animate-spin" />
					{:else}
						<Navigation class="h-4 w-4" />
					{/if}
				</Button>
				{#if isAnchorMode}
					<Button
						size="icon"
						variant="outline"
						class="h-10 w-10 rounded-2xl border-border/60 bg-card/90 shadow-md backdrop-blur"
						onclick={handleBackToBrowseAnchor}
						aria-label="回到浏览锚点位置"
					>
						<PersonStanding class="h-4 w-4" />
					</Button>
				{/if}
				<Button
					size="icon"
					variant="outline"
					class="h-10 w-10 rounded-2xl border-border/60 bg-card/90 shadow-md backdrop-blur"
					onclick={handleClose}
					aria-label="关闭"
				>
					<X class="h-4 w-4" />
				</Button>
			</div>
		</div>

			{#if gestureHintVisible || placementHintVisible}
			<div class="pointer-events-none absolute inset-x-0 top-4 z-30 flex justify-center px-4">
				{#if placementHintVisible}
					<div class="rounded-2xl border border-border/60 bg-card/95 px-4 py-1.5 text-sm font-medium text-foreground shadow-lg">{placementHintMessage}</div>
				{:else if gestureHintVisible}
					<div class="rounded-2xl border border-dashed border-primary/40 bg-card/90 px-4 py-1 text-xs text-muted-foreground shadow">{gestureHintMessage}</div>
				{/if}
			</div>
		{/if}

		<div class="pointer-events-none absolute inset-0 z-10 flex items-center justify-center">
				<div class="center-pin" class:dragging={isMapInteracting} aria-hidden="true">
					<div class="center-pin-icon">
						<svg class="center-pin-svg" viewBox="0 0 40 43" aria-hidden="true" focusable="false">
							<defs>
								<linearGradient id="picker-ring-gradient" x1="6" y1="3" x2="33" y2="30" gradientUnits="userSpaceOnUse">
									<stop offset="0" stop-color="#0A1C67" />
									<stop offset="1" stop-color="#0C2A7B" />
								</linearGradient>
								<linearGradient id="picker-core-gradient" x1="10" y1="8" x2="29" y2="26" gradientUnits="userSpaceOnUse">
									<stop offset="0" stop-color="#63BCFF" />
									<stop offset="1" stop-color="#3E90FF" />
								</linearGradient>
								<linearGradient id="picker-stem-gradient" x1="17.6" y1="24.5" x2="22.4" y2="42" gradientUnits="userSpaceOnUse">
									<stop offset="0" stop-color="#58B2FF" />
									<stop offset="1" stop-color="#3A8AFF" />
								</linearGradient>
							</defs>
							<circle cx="20" cy="16.5" r="14" fill="url(#picker-ring-gradient)" />
							<circle cx="20" cy="16.5" r="9.6" fill="url(#picker-core-gradient)" />
							<rect x="17.6" y="24.5" width="4.8" height="17.5" rx="2.4" fill="url(#picker-stem-gradient)" />
						</svg>
					</div>
				</div>
			</div>

			<!-- 搜索覆盖层 -->
			{#if showSearch}
				<div class="absolute inset-x-0 top-0 z-30 flex flex-col border-b border-border/60 bg-card/95 backdrop-blur-sm shadow-lg">
					<!-- 搜索输入框 -->
					<div class="flex items-center gap-2 border-b px-3 py-2">
						<Search class="h-4 w-4 shrink-0 text-muted-foreground" />
					<input
						type="text"
						bind:this={searchInput}
						bind:value={searchText}
						placeholder="搜索地点名称"
						class="flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
						oninput={() => {
							if (searchDebounceTimer) clearTimeout(searchDebounceTimer);
							if (searchText.trim().length > 0) {
								searchDebounceTimer = setTimeout(() => { doSearch(); }, 500);
							} else {
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
							<Loader2 class="h-4 w-4 shrink-0 animate-spin text-primary" />
						{:else if searchText.length > 0}
							<button class="shrink-0 text-muted-foreground hover:text-foreground" onclick={() => { if (searchDebounceTimer) clearTimeout(searchDebounceTimer); searchText = ''; searchResults = []; searchInput?.focus(); }}>
								<X class="h-4 w-4" />
							</button>
						{/if}
						<Button variant="ghost" size="sm" class="shrink-0 text-xs" onclick={toggleSearch}>
							取消
						</Button>
					</div>

				<!-- 搜索结果列表 -->
				{#if searchResults.length > 0}
					<div class="max-h-64 overflow-y-auto">
						{#each searchResults as result, i (`${result.latitude}_${result.longitude}`)}
							<button
								class="flex w-full items-center gap-3 px-4 py-3 text-left transition-colors hover:bg-muted/50"
								onclick={() => selectSearchResult(result)}
							>
								<MapPin class="h-4 w-4 shrink-0 text-primary" />
								<div class="min-w-0 flex-1">
									<div class="truncate text-sm font-medium">{result.name}</div>
									<div class="text-xs text-muted-foreground">
										{result.address || `${result.latitude.toFixed(6)}, ${result.longitude.toFixed(6)}`}
									</div>
								</div>
							</button>
							{#if i < searchResults.length - 1}
								<div class="mx-4 border-b"></div>
							{/if}
						{/each}
					</div>
				{:else if searchText.trim().length === 0 && anchorHistory.length > 0}
					<!-- 无输入时展示历史记录 -->
					<div class="max-h-64 overflow-y-auto">
						<div class="flex items-center justify-between px-4 py-2">
							<div class="flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
								<Clock class="h-3.5 w-3.5" />
								<span>最近使用</span>
							</div>
							<button class="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground" onclick={handleClearHistory}>
								<Trash2 class="h-3 w-3" />
								<span>清空</span>
							</button>
						</div>
						{#each anchorHistory as item, i (item.timestamp)}
							<button
								class="flex w-full items-center gap-3 px-4 py-2.5 text-left transition-colors hover:bg-muted/50"
								onclick={() => { selectHistoryItem(item); showSearch = false; }}
							>
								<Clock class="h-4 w-4 shrink-0 text-muted-foreground" />
								<div class="min-w-0 flex-1 truncate text-sm">{item.locationName}</div>
							</button>
							{#if i < anchorHistory.length - 1}
								<div class="mx-4 border-b"></div>
							{/if}
						{/each}
					</div>
				{/if}
				</div>
			{/if}

			{#if !isMapAvailable}
				<div class="absolute inset-0 z-20 flex items-center justify-center bg-background/80 backdrop-blur-sm">
					<div class="text-center space-y-3">
						<p class="font-medium">地图未配置</p>
						<p class="text-sm text-muted-foreground">请设置 VITE_AMAP_KEY</p>
						<Button size="sm" variant="outline" onclick={handleRetryMapLoad}>
							重试加载
						</Button>
					</div>
				</div>
			{/if}

			{#if isLoadingMap}
				<div class="absolute inset-0 z-20 flex items-center justify-center bg-background/70 backdrop-blur-sm">
					<Loader2 class="h-8 w-8 animate-spin text-primary" />
				</div>
			{/if}

			<!-- 悬浮定位按钮 (移动端) -->
			<div class="absolute right-3 top-3 z-10 flex flex-col gap-2 sm:hidden">
				<Button 
					size="icon" 
					class="h-11 w-11 rounded-2xl shadow-lg" 
					onclick={handleUseCurrentLocation} 
					disabled={isLocating}
					aria-label="使用当前位置"
				>
					{#if isLocating}
						<Loader2 class="h-5 w-5 animate-spin" />
					{:else}
						<Navigation class="h-5 w-5" />
					{/if}
				</Button>
				{#if isAnchorMode}
					<Button
						size="icon"
						variant="secondary"
						class="h-11 w-11 rounded-2xl border border-border/60 bg-card/90 shadow-lg"
						onclick={handleBackToBrowseAnchor}
						aria-label="回到浏览锚点位置"
					>
						<PersonStanding class="h-5 w-5" />
					</Button>
				{/if}
			</div>

			<!-- 移动端底部抽屉 -->
			<div 
				class="absolute bottom-0 left-0 right-0 z-20 transition-transform duration-300 sm:hidden"
				class:translate-y-0={!isBottomSheetExpanded}
				style="transform: translateY({isBottomSheetExpanded ? '0' : 'calc(100% - 140px)'})"
			>
				<div class="rounded-t-[28px] border border-border/60 bg-card/95 shadow-2xl backdrop-blur-md">
					<!-- 展开/收起手柄 -->
					<button
						class="flex w-full flex-col items-center gap-1 py-3 active:bg-muted/50"
						onclick={() => isBottomSheetExpanded = !isBottomSheetExpanded}
					>
						<div class="h-1 w-12 rounded-full bg-muted-foreground/20"></div>
						{#if isBottomSheetExpanded}
							<ChevronDown class="h-4 w-4 text-muted-foreground" />
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
						overrideName={locationName || '拖动地图选择位置'}
						overrideLatitude={latitude}
						overrideLongitude={longitude}
						showCoordinates
					/>

					<!-- 展开后显示的内容 -->
						{#if isBottomSheetExpanded}
							<div class="space-y-4 pb-4">
								<!-- 提示信息 -->
								<div class="rounded-2xl border border-border/50 bg-background/65 p-3 text-xs text-muted-foreground">
									💡 {hintAfterConfirm}
								</div>

								{#if isAnchorMode}
									<!-- 搜索范围 -->
									<Card class="border-muted">
										<CardContent class="flex flex-col gap-3 p-4">
											<div class="flex items-center justify-between">
												<div class="flex items-center gap-2 text-sm font-medium">
													<SlidersHorizontal class="h-4 w-4 text-primary" />
													<span>搜索范围</span>
												</div>
												<span class="text-sm font-bold text-primary">{searchRadius} km</span>
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
								{/if}
							</div>
						{/if}

					<!-- 操作按钮 -->
					<div class="sticky bottom-0 bg-background pb-4 pt-3 space-y-2">
						<Button class="h-12 w-full gap-2 text-base font-semibold" onclick={handleConfirm} disabled={!locationName}>
							<MapPin class="h-5 w-5" />
							{confirmButtonText}
						</Button>
						{#if isAnchorMode}
							<Button
								variant="outline"
								class="h-12 w-full gap-2 text-base font-semibold"
								onclick={() => showShareModal = true}
								disabled={!locationName}
							>
								<Share2 class="h-5 w-5" />
								分享此位置
							</Button>
						{/if}
					</div>
					</div>
				</div>
			</div>
		</div>

		<!-- 桌面端信息面板 -->
		<div class="hidden gap-4 rounded-3xl border border-border/60 bg-card/80 p-4 shadow-sm backdrop-blur-md sm:flex">
			<!-- 左侧：位置信息 + 搜索范围 -->
			<div class="flex min-w-0 flex-1 flex-col gap-3">
				<div class="flex items-start gap-3">
					<div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-primary/10">
						<MapPin class="h-5 w-5 text-primary" />
					</div>
					<div class="min-w-0 flex-1">
						<p class="text-xs font-medium text-muted-foreground">选中位置</p>
						<p class="truncate text-base font-bold leading-snug">
							{locationName || '拖动地图选择位置'}
						</p>
						<p class="mt-0.5 font-mono text-xs text-muted-foreground">
							{latitude.toFixed(6)}, {longitude.toFixed(6)}
						</p>
					</div>
				</div>
				{#if isAnchorMode}
					<!-- 搜索范围 -->
					<div class="flex items-center gap-3">
						<div class="flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
							<SlidersHorizontal class="h-3.5 w-3.5" />
							<span>搜索范围</span>
						</div>
						<input
							type="range"
							min="1"
							max="50"
							step="1"
							class="h-1.5 flex-1 cursor-pointer appearance-none rounded-full bg-muted accent-primary"
							bind:value={searchRadius}
						/>
						<span class="min-w-[4ch] text-right text-sm font-bold text-primary">{searchRadius} km</span>
					</div>
				{/if}
			</div>
			<!-- 右侧：操作按钮 -->
			<div class="flex shrink-0 items-end gap-2">
				{#if isAnchorMode}
					<Button variant="outline" size="sm" class="h-10 gap-1.5 rounded-2xl border-border/60 bg-background/70" onclick={() => showShareModal = true} disabled={!locationName}>
						<Share2 class="h-3.5 w-3.5" />
						分享
					</Button>
				{/if}
				<Button size="sm" class="h-10 gap-1.5 rounded-2xl" onclick={handleConfirm} disabled={!locationName}>
					<MapPin class="h-3.5 w-3.5" />
					{confirmButtonText}
				</Button>
			</div>
		</div>
	</Dialog.Content>
</Dialog.Root>

<!-- 位置分享模态框 -->
<CreateLocationShareModal 
	bind:open={showShareModal}
	latitude={latitude}
	longitude={longitude}
	locationName={locationName}
/>
