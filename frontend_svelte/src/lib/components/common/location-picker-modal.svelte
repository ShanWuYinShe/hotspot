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
	import Loader2 from 'lucide-svelte/icons/loader-2';
	import Crosshair from 'lucide-svelte/icons/crosshair';
	import ChevronUp from 'lucide-svelte/icons/chevron-up';
	import ChevronDown from 'lucide-svelte/icons/chevron-down';
	import SlidersHorizontal from 'lucide-svelte/icons/sliders-horizontal';
	import Share2 from 'lucide-svelte/icons/share-2';
	import Search from 'lucide-svelte/icons/search';
	import X from 'lucide-svelte/icons/x';
	import Clock from 'lucide-svelte/icons/clock';
	import Trash2 from 'lucide-svelte/icons/trash-2';
	import CreateLocationShareModal from './create-location-share-modal.svelte';

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

	type MapClickEvent = {
		lnglat?: AMap.LngLat;
	};

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
		latitude = result.latitude;
		longitude = result.longitude;
		locationName = result.name;
		map?.setCenter([result.longitude, result.latitude]);
		showSearch = false;
		searchText = '';
		searchResults = [];
	}

	function selectHistoryItem(item: AnchorHistoryItem) {
		latitude = item.latitude;
		longitude = item.longitude;
		locationName = item.locationName;
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

	function handleConfirm() {
		mapAnchorStore.setAnchor(latitude, longitude, locationName || undefined, searchRadius);
		addToAnchorHistory(latitude, longitude, locationName || '自定义位置');
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
				anchorHistory = getAnchorHistory();
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
		}
	});

	// 组件卸载时清理资源
	onMount(() => {
		return () => {
			map?.destroy();
			map = null;
			if (debounceTimer) {
				clearTimeout(debounceTimer);
				debounceTimer = null;
			}
			if (searchDebounceTimer) {
				clearTimeout(searchDebounceTimer);
				searchDebounceTimer = null;
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

	/* 底部安全区域支持 */
	.pb-safe {
		padding-bottom: max(1rem, env(safe-area-inset-bottom));
	}

	/* 优化移动端触摸反馈 */
	@media (hover: none) and (pointer: coarse) {
		button:active {
			transform: scale(0.98);
		}
	}
</style>

<Dialog.Root bind:open onOpenChange={(value) => { open = value; onOpenChange?.(value); }}>
	<Dialog.Content class="flex h-[90dvh] max-h-[90dvh] flex-col p-0 sm:h-[85vh] sm:max-w-4xl sm:p-6" hideClose>
		<!-- 桌面端标题 -->
		<Dialog.Header class="hidden px-6 pt-6 sm:block">
			<Dialog.Title class="flex items-center gap-2">
				<div class="flex h-10 w-10 items-center justify-center rounded-xl bg-linear-to-br from-primary/20 to-primary/10">
					<MapPin class="h-5 w-5 text-primary" />
				</div>
				<div class="flex-1">
					<div>地图选择锚点</div>
					<div class="text-sm font-normal text-muted-foreground">拖动地图选择浏览位置</div>
				</div>
				<!-- 桌面端搜索按钮 -->
				<Button variant="ghost" size="icon" class="h-9 w-9 rounded-xl" onclick={toggleSearch} aria-label="搜索">
					<Search class="h-4 w-4" />
				</Button>
				<!-- 桌面端关闭按钮 -->
				<Button variant="ghost" size="icon" class="h-9 w-9 rounded-xl" onclick={handleClose} aria-label="关闭">
					<svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
					</svg>
					<span class="sr-only">关闭</span>
				</Button>
			</Dialog.Title>
		</Dialog.Header>

		<!-- 移动端简化标题 -->
		<div class="flex items-center justify-between border-b bg-background px-4 py-3 sm:hidden">
			<div class="flex items-center gap-2">
				<MapPin class="h-5 w-5 text-primary" />
				<span class="font-semibold">选择位置</span>
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
		<div class="relative flex-1 overflow-hidden sm:rounded-2xl sm:border">
			<div class="h-full w-full touch-none" bind:this={mapContainer}></div>

			<!-- 搜索覆盖层 -->
			{#if showSearch}
				<div class="absolute inset-x-0 top-0 z-30 flex flex-col bg-background/95 backdrop-blur-sm shadow-lg">
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
					<div class="text-center">
						<p class="font-medium">地图未配置</p>
						<p class="text-sm text-muted-foreground">请设置 VITE_AMAP_KEY</p>
					</div>
				</div>
			{/if}

			{#if isLoadingMap}
				<div class="absolute inset-0 z-20 flex items-center justify-center bg-background/70 backdrop-blur-sm">
					<Loader2 class="h-8 w-8 animate-spin text-primary" />
				</div>
			{/if}

			<!-- 地图中心标记 -->
			<div class="pointer-events-none absolute left-1/2 top-1/2 z-10 -translate-x-1/2 -translate-y-full">
				<div class="flex flex-col items-center gap-1">
					<div class="flex h-10 w-10 items-center justify-center rounded-full bg-primary shadow-lg">
						<Crosshair class="h-5 w-5 text-primary-foreground" />
					</div>
					<div class="rounded-full bg-background/95 px-2.5 py-0.5 text-xs font-medium shadow-md">
						浏览中心
					</div>
				</div>
			</div>

			<!-- 悬浮定位按钮 (移动端) -->
			<div class="absolute right-3 top-3 z-10 sm:hidden">
				<Button 
					size="icon" 
					class="h-11 w-11 rounded-full shadow-lg" 
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
			</div>

			<!-- 移动端底部抽屉 -->
			<div 
				class="absolute bottom-0 left-0 right-0 z-20 transition-transform duration-300 sm:hidden"
				class:translate-y-0={!isBottomSheetExpanded}
				style="transform: translateY({isBottomSheetExpanded ? '0' : 'calc(100% - 140px)'})"
			>
				<div class="rounded-t-3xl border-t bg-background shadow-2xl">
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
						<div class="mb-3">
							<div class="text-xs font-medium text-muted-foreground">当前选中</div>
							<div class="mt-1 truncate text-lg font-bold">
								{locationName || '拖动地图选择位置'}
							</div>
							<div class="mt-0.5 text-xs text-muted-foreground">
								{latitude.toFixed(6)}, {longitude.toFixed(6)}
							</div>
						</div>

						<!-- 展开后显示的内容 -->
						{#if isBottomSheetExpanded}
							<div class="space-y-4 pb-4">
								<!-- 提示信息 -->
								<div class="rounded-xl bg-muted/50 p-3 text-xs text-muted-foreground">
									💡 双击地图可快速选中位置，确认后才会更新锚点
								</div>

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
							</div>
						{/if}

					<!-- 操作按钮 -->
					<div class="sticky bottom-0 bg-background pb-4 pt-3 space-y-2">
						<Button class="h-12 w-full gap-2 text-base font-semibold" onclick={handleConfirm} disabled={!locationName}>
							<MapPin class="h-5 w-5" />
							设为浏览锚点
						</Button>
						<Button 
							variant="outline" 
							class="h-12 w-full gap-2 text-base font-semibold" 
							onclick={() => showShareModal = true} 
							disabled={!locationName}
						>
							<Share2 class="h-5 w-5" />
							分享此位置
						</Button>
					</div>
					</div>
				</div>
			</div>
		</div>

		<!-- 桌面端信息面板 -->
		<div class="hidden flex-col gap-4 py-4 sm:flex">
			<Card class="border-0 bg-linear-to-r from-primary/5 to-primary/10">
				<CardContent class="flex flex-col gap-4 p-4 sm:flex-row sm:items-center sm:justify-between">
					<div class="min-w-0 flex-1">
						<div class="text-xs font-medium text-muted-foreground">选中位置</div>
						<div class="mt-1 truncate text-lg font-bold">
							{locationName || '拖动地图选择位置'}
						</div>
						<div class="mt-0.5 text-xs text-muted-foreground">
							{latitude.toFixed(6)}, {longitude.toFixed(6)}
						</div>
						<div class="mt-2 text-xs text-muted-foreground">
							💡 双击地图可快速选中位置，确认后才会更新锚点
						</div>
					</div>
					<div class="flex shrink-0 items-center gap-2">
						<Button variant="outline" size="sm" class="gap-2" onclick={handleUseCurrentLocation} disabled={isLocating}>
							{#if isLocating}
								<Loader2 class="h-4 w-4 animate-spin" />
								定位中
							{:else}
								<Navigation class="h-4 w-4" />
								定位到当前
							{/if}
						</Button>
						<Button variant="outline" size="sm" class="gap-2" onclick={() => showShareModal = true} disabled={!locationName}>
							<Share2 class="h-4 w-4" />
							分享
						</Button>
						<Button size="sm" class="gap-2" onclick={handleConfirm} disabled={!locationName}>
							<MapPin class="h-4 w-4" />
							设为锚点
						</Button>
					</div>
				</CardContent>
			</Card>

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
