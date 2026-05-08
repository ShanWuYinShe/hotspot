<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { platformStore } from '$lib/stores/platform.svelte';
	import { loadAMap, loadAMapPlugin } from '$lib/utils/amap';
	import { getAnchorHistory, addToAnchorHistory } from '$lib/utils/anchor-history';
	import type { MapHotspotItem, MapViewport } from '$lib/types/api';
	import { feedService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import LocationMap from './location-map.svelte';
	import MobileBottomSheet from './mobile-bottom-sheet.svelte';
	import DesktopSidebar from './desktop-sidebar.svelte';
	import ArrowLeft from 'lucide-svelte/icons/arrow-left';
	import SlidersHorizontal from 'lucide-svelte/icons/sliders-horizontal';

	interface SearchResultItem {
		name: string;
		address: string;
		latitude: number;
		longitude: number;
	}

	interface HotLocation {
		name: string;
		desc: string;
		emoji: string;
		users?: number;
	}

	const returnTarget = $derived(page.url.searchParams.get('from') || '/');

	let selectedLocation = $state<{ latitude: number; longitude: number; name: string } | null>(null);
	let selectedPin = $state<MapHotspotItem | null>(null);
	let activeTab = $state<'recent' | 'hot' | 'nearby'>('nearby');
	let searchText = $state('');
	let searchResults = $state.raw<SearchResultItem[]>([]);
	let anchorHistory = $state.raw<ReturnType<typeof getAnchorHistory>>([]);
	let hotLocations = $state.raw<HotLocation[]>([]);
	let nearbyPins = $state.raw<MapHotspotItem[]>([]);
	let showFilter = $state(false);
	let filterState = $state({ showUsers: true, showGroups: true, showMoments: true });
	let totalCounts = $state({ users: 0, groups: 0, moments: 0 });
	let bottomSheetHeight = $state(318);
	let placeSearch: AMap.PlaceSearch | null = null;

	function handleBack() {
		goto(returnTarget);
	}

	function handleConfirm() {
		const loc = selectedLocation ?? {
			latitude: mapAnchorStore.latitude,
			longitude: mapAnchorStore.longitude,
			name: mapAnchorStore.name || '当前位置',
		};
		mapAnchorStore.setAnchor(loc.latitude, loc.longitude, loc.name);
		addToAnchorHistory(loc.latitude, loc.longitude, loc.name);
		goto(returnTarget);
	}

	function handleSelectLocation(loc: { latitude: number; longitude: number; name: string }) {
		selectedLocation = loc;
		searchText = loc.name;
		selectedPin = null;
	}

	function handleViewportChange(_viewport: MapViewport) {
		// Map data is fetched internally by location-map
	}

	function handlePinSelect(item: MapHotspotItem | null) {
		selectedPin = item;
	}

	function handleMapClick() {
		selectedPin = null;
		showFilter = false;
	}

	function handleTabChange(tab: 'recent' | 'hot' | 'nearby') {
		activeTab = tab;
	}

	async function handleSearch(text: string) {
		searchText = text;
		if (!text.trim()) {
			searchResults = [];
			return;
		}
		try {
			await loadAMap();
			await loadAMapPlugin(['AMap.PlaceSearch']);
			if (!placeSearch) {
				placeSearch = new AMap.PlaceSearch({
					city: '全国',
					pageSize: 5,
				});
			}
			placeSearch.search(text, (status: string, result: any) => {
				if (status === 'complete' && result.poiList?.pois) {
					searchResults = result.poiList.pois
						.filter((poi: any) => poi.location)
						.map((poi: any) => ({
							name: poi.name || '',
							address: poi.address || '',
							latitude: poi.location.lat,
							longitude: poi.location.lng,
						}));
				} else {
					searchResults = [];
				}
			});
		} catch {
			searchResults = [];
		}
	}

	async function loadHotLocations() {
		try {
			const response = await feedService.getTrending();
			if (isSuccess(response) && response.data) {
				hotLocations = (response.data.trending_tags ?? []).slice(0, 8).map((tag) => ({
					name: tag.name,
					desc: `${(tag as any).count || 0} 条相关内容`,
					emoji: '🔥',
					users: (tag as any).count,
				}));
			}
		} catch { /* ignore */ }
	}

	function handleFilterChange(key: 'showUsers' | 'showGroups' | 'showMoments') {
		filterState[key] = !filterState[key];
		filterState = { ...filterState };
	}

	function handleRecenter() {
		selectedLocation = null;
		searchText = '';
	}

	function handleBottomSheetHeight(height: number) {
		bottomSheetHeight = height;
	}

	onMount(() => {
		anchorHistory = getAnchorHistory();
		loadHotLocations();
	});
</script>

{#if !platformStore.isDesktop}
	<!-- Mobile / Tablet Layout -->
	<div class="relative h-full w-full overflow-hidden" style="background: var(--neu-bg);">
		<!-- Map (full screen) -->
		<LocationMap
			onviewportchange={handleViewportChange}
			onpinselect={handlePinSelect}
			onmapclick={handleMapClick}
			selectedLocation={selectedLocation}
			isMobile={true}
			bottomOffset={bottomSheetHeight + 10}
		/>

		<!-- Top bar -->
		<div class="absolute left-0 right-0 top-0 z-50 p-3">
			<div class="flex items-center justify-between">
				<button
					onclick={handleBack}
					class="flex items-center justify-center rounded-full outline-none neu-circle-hover transition-transform duration-150 active:scale-[0.88]"
					style="width: 42px; height: 42px; background: var(--neu-bg); border: 1px solid var(--neu-border); box-shadow: 0 2px 12px var(--neu-shadow-dark);"
					aria-label="返回"
				>
					<ArrowLeft size={17} color="var(--neu-text)" strokeWidth={2} />
				</button>

				<button
					onclick={() => { showFilter = !showFilter; }}
					class="flex items-center gap-2 rounded-full outline-none transition-transform duration-150 active:scale-[0.93]"
					style="padding: 9px 16px; font-size: 14px; background: {showFilter ? 'var(--neu-accent-primary)' : 'var(--neu-bg)'}; color: {showFilter ? 'white' : 'var(--neu-text)'}; border: {showFilter ? '1px solid transparent' : '1px solid var(--neu-border)'}; box-shadow: {showFilter ? '0 4px 16px var(--neu-glow)' : '0 2px 12px var(--neu-shadow-dark)'};"
				>
					<SlidersHorizontal size={14} strokeWidth={showFilter ? 2.5 : 1.75} />
					筛选
				</button>
			</div>

			<!-- Filter legend dropdown -->
			{#if showFilter}
				<div
					class="absolute right-3 top-[62px] w-[136px] overflow-hidden rounded-2xl"
					style="background: var(--neu-bg); border: 1px solid var(--neu-border); box-shadow: 0 8px 28px var(--neu-shadow-dark); z-index: 60;"
				>
					{#each [
						{ key: 'showUsers' as const, label: '用户', icon: '/map/marker_user.svg', color: '#2563EB' },
						{ key: 'showGroups' as const, label: '群组', icon: '/map/marker_group.svg', color: '#059669' },
						{ key: 'showMoments' as const, label: '动态', icon: '/map/marker_moment.svg', color: '#EA580C' },
					] as filter, i}
						{#if i > 0}
							<div style="height: 1px; background: var(--neu-border);"></div>
						{/if}
						<button
							onclick={() => handleFilterChange(filter.key)}
							class="flex w-full items-center gap-2 px-3 py-3 outline-none transition-transform duration-150 active:scale-[0.95]"
							style="opacity: {filterState[filter.key] ? 1 : 0.38};"
						>
							<img src={filter.icon} alt={filter.label} style="width: 18px; height: 24px; flex-shrink: 0;" />
							<span style="flex: 1; font-size: 13px; color: var(--neu-text); text-align: left;">{filter.label}</span>
							<div
								style="width: 13px; height: 13px; border-radius: 50%; flex-shrink: 0; background: {filterState[filter.key] ? filter.color : 'transparent'}; border: 2px solid {filterState[filter.key] ? filter.color : 'var(--neu-border)'}; box-shadow: {filterState[filter.key] ? `0 0 6px ${filter.color}55` : 'none'}; transition: all 0.18s;"
							></div>
						</button>
					{/each}
				</div>
			{/if}
		</div>

		<!-- Bottom sheet -->
		<MobileBottomSheet
			{selectedLocation}
			{selectedPin}
			{activeTab}
			{searchText}
			{searchResults}
			{anchorHistory}
			{nearbyPins}
			{hotLocations}
			onconfirm={handleConfirm}
			onsearch={handleSearch}
			onselectlocation={handleSelectLocation}
			ontabchange={handleTabChange}
			onpindismiss={() => { selectedPin = null; }}
			onbottomsheetheight={handleBottomSheetHeight}
		/>
	</div>
{:else}
	<!-- Desktop Layout -->
	<div class="flex h-full" style="background: var(--neu-bg);">
		<DesktopSidebar
			{selectedLocation}
			{selectedPin}
			{activeTab}
			{searchText}
			{searchResults}
			{anchorHistory}
			{nearbyPins}
			{hotLocations}
			{filterState}
			{totalCounts}
			onback={handleBack}
			onconfirm={handleConfirm}
			onsearch={handleSearch}
			onselectlocation={handleSelectLocation}
			ontabchange={handleTabChange}
			onpinselect={handlePinSelect}
			onfilterchange={(f) => { filterState = f; }}
			onrecenter={handleRecenter}
		/>

		<div class="flex-1 relative overflow-hidden">
			<LocationMap
				onviewportchange={handleViewportChange}
				onpinselect={handlePinSelect}
				onmapclick={handleMapClick}
				selectedLocation={selectedLocation}
				isMobile={false}
			/>
		</div>
	</div>
{/if}
