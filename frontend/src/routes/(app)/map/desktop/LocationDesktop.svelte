<script lang="ts">
	import { X, ChevronRight, SlidersHorizontal, Heart, Check } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { ROUTES } from '@/lib/routes';
	import { startDefaultLocation } from '@/lib/start-default-location';
	import { SEARCH_DB, type PinType, type Pin, type LocationResult } from '../shared/location-data';
	import {
		PIN_COLORS, PIN_LABELS, pinTone, buildMapData, fetchMapCanvas,
		type MapHeatBucket,
		markerUserUrl, markerGroupUrl, markerMomentUrl,
		markerUserClusterUrl, markerGroupClusterUrl, markerMomentClusterUrl
	} from '../shared/location-shared';
	import MapPinMarker from '../shared/MapPinMarker.svelte';
	import MapHeatDot from '../shared/MapHeatDot.svelte';
	import PinMedia from '../shared/PinMedia.svelte';
	import LocationDesktopPanel from './LocationDesktopPanel.svelte';

	// 桌面地图探索：左侧栏（统计/搜索/Tab 列表/确认，由 LocationDesktopPanel 承载）
	// + 地图面板（OSM iframe + 图钉 + 筛选 + 详情浮层）。
	// 图钉与统计来自 POST /api/v1/map/canvas（spec: api_spec_map.md）。

	let searchQuery = $state('');
	let selectedPin = $state<Pin | null>(null);
	let selectedLoc = $state<LocationResult | null>(null);
	let activeFilters = $state<Set<PinType>>(new Set(['user', 'group', 'moment']));
	let activeTab = $state<'nearby' | 'hot' | 'recent'>('nearby');
	let showFilter = $state(true);
	const zoom = 0.045;
	let mapCenter = $state<{ lat: number; lng: number } | null>(null);
	let defaultCenter = $state<{ lat: number; lng: number } | null>(null);

	// 初始/复位中心来自 ClientConfig default_location（MSW mock），不写死坐标。
	$effect(() => startDefaultLocation((loc) => {
						const point = loc?.point ? { lat: loc.point.latitude, lng: loc.point.longitude } : null;
			defaultCenter = point;
			if (!mapCenter) mapCenter = point;
}));

	// map/canvas 数据
	let mapPins = $state<Pin[]>([]);
	let heatBuckets = $state<MapHeatBucket[]>([]);
	let mapCounts = $state<Record<PinType, number>>({ user: 0, group: 0, moment: 0 });
	let mapLoading = $state(true);

	// 视口中心或筛选变化时重新拉取
	$effect(() => {
		const center = mapCenter;
		const types = [...activeFilters];
		if (!center) return;
		let cancelled = false;
		mapLoading = true;
		void fetchMapCanvas(center, zoom, types).then((r) => {
			if (cancelled) return;
			mapPins = r.pins;
			heatBuckets = r.heatBuckets;
			mapCounts = r.counts;
			mapLoading = false;
		});
		return () => {
			cancelled = true;
		};
	});

	// 图钉详情跳转：user/group/moment 分别路由到对应详情页
	function openPinDetail(pin: Pin) {
		if (pin.type === 'user') void goto(ROUTES.user(pin.id));
		else if (pin.type === 'group') void goto(ROUTES.group(pin.id));
		else void goto(ROUTES.moment(pin.id));
	}

	function toggleFilter(type: PinType) {
		const next = new Set(activeFilters);
		next.has(type) ? next.delete(type) : next.add(type);
		activeFilters = next;
	}

	const searchResults = $derived(
		searchQuery.trim() ? SEARCH_DB.filter((s) => s.displayName.includes(searchQuery)).slice(0, 6) : []
	);
	const nearbyPins = $derived(mapPins.filter((p) => (p.count ?? 1) === 1));

	const mapData = $derived(mapCenter ? buildMapData(mapCenter, zoom) : null);

	const userCount = $derived(mapCounts.user);
	const groupCount = $derived(mapCounts.group);
	const momentCount = $derived(mapCounts.moment);

	function handleSelectLoc(loc: LocationResult) {
		selectedLoc = loc;
		searchQuery = loc.displayName;
		mapCenter = { lat: loc.lat, lng: loc.lng };
		selectedPin = null;
	}

	function handleSelectPin(pin: Pin) {
		selectedPin = pin;
		mapCenter = { lat: pin.lat, lng: pin.lng };
	}

	function handleConfirm() {
		if (selectedLoc) {
			try {
				localStorage.setItem('hotspot_last_location', JSON.stringify([selectedLoc.lat, selectedLoc.lng]));
			} catch {}
		}
		history.back();
	}
</script>

<div class="flex h-full bg-surface-primary">
	<!-- ══ LEFT SIDEBAR ══ -->
	<div class="flex flex-col shrink-0 w-85 border-r border-border-default"  >
		<LocationDesktopPanel
			bind:searchQuery
			{searchResults}
			{nearbyPins}
			bind:activeTab
			bind:selectedPin
			bind:selectedLoc
			{mapCounts}
			onSelectPin={handleSelectPin}
			onSelectLoc={handleSelectLoc}
			onReset={() => {
				mapCenter = defaultCenter;
				selectedLoc = null;
				searchQuery = '';
			}}
			onConfirm={handleConfirm}
		/>
	</div>

	<!-- ══ MAP PANEL ══ -->
	<div
		class="flex-1 relative overflow-hidden"
		role="button"
		tabindex="0"
		aria-label="地图（点击空白处取消选中）"
		onclick={(e) => {
			// 详情浮层与筛选下拉内的点击不视为地图空白点击
			if ((e.target as HTMLElement).closest('[data-map-popup]')) return;
			if ((e.target as HTMLElement).closest('[data-map-filter]')) return;
			selectedPin = null;
			showFilter = false;
		}}
		onkeydown={(e) => {
			if (e.target !== e.currentTarget) return;
			if (e.key === 'Enter' || e.key === ' ' || e.key === 'Escape') {
				e.preventDefault();
				selectedPin = null;
			}
		}}
	>
		{#if mapData}
		<iframe src={mapData.url} class="w-full h-full border-0" title="地图" style="pointer-events:none;"></iframe>

		<!-- Current location dot -->
		<div style="position:absolute;left:50%;top:50%;transform:translate(-50%,-50%);pointer-events:none;z-index:5;">
			<div style="width:18px;height:18px;border-radius:50%;background:{PIN_COLORS.user};border:3px solid var(--cst-surface-primary);box-shadow:0 2px 10px {pinTone('user', 55)};position:relative;">
				<span class="absolute rounded-full" style="inset:-8px;background:{pinTone('user', 18)};border:1px solid {pinTone('user', 25)};"></span>
			</div>
		</div>

		<!-- Heatmap buckets（大视口 / 高密度时服务端返回热力统计） -->
		{#each heatBuckets as bucket (bucket.cell_token)}
			<MapHeatDot {bucket} bounds={mapData.bounds} />
		{/each}

		<!-- Pin markers -->
		{#each mapPins as pin (pin.id)}
			<MapPinMarker
				pin={pin}
				bounds={mapData.bounds}
				isSelected={selectedPin?.id === pin.id}
				onSelect={(e) => {
					e.stopPropagation();
					selectedPin = selectedPin?.id === pin.id ? null : pin;
					mapCenter = { lat: pin.lat, lng: pin.lng };
				}}
			/>
		{/each}
		{/if}

		{#if mapLoading}
			<!-- 加载中的轻量指示 -->
			<div class="absolute top-3 left-1/2 -translate-x-1/2 z-20 rounded-full bg-surface-primary px-3 py-1 text-xs text-text-secondary shadow-[0_2px_12px_var(--cst-elevation-shadow-dark)] border border-border-default" >
				加载地图数据…
			</div>
		{/if}

		<!-- 左上筛选浮动按钮 + 下拉（对齐 Figma Make LocationDesktop 筛选面板） -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<div data-map-filter class="absolute z-20" style="top:14px;left:14px;" onclick={(e) => e.stopPropagation()}>
			<button
				type="button"
				class="hover-btn flex items-center gap-2 rounded-full outline-none"
				style="padding:9px 16px;font-size:13px;background:{showFilter ? 'var(--cst-accent-default)' : 'var(--cst-surface-primary)'};color:{showFilter ? 'white' : 'var(--cst-text-primary)'};border:1px solid {showFilter ? 'transparent' : 'var(--cst-border-default)'};box-shadow:{showFilter ? '0 4px 16px var(--cst-glow-accent)' : '0 4px 16px var(--cst-elevation-shadow-dark)'};"
				onclick={() => (showFilter = !showFilter)}
			>
				<SlidersHorizontal class="size-3.25"  strokeWidth={showFilter ? 2.5 : 1.75} />
				筛选
				{#if activeFilters.size < 3}
					<span style="margin-left:2px;font-size:10px;padding:1px 6px;border-radius:999px;background:{showFilter ? 'var(--cst-surface-map-control-active)' : 'var(--cst-accent-soft)'};color:{showFilter ? 'white' : 'var(--cst-accent-default)'};font-weight:500;">{activeFilters.size}</span>
				{/if}
			</button>
			{#if showFilter}
				<div class="mt-2 overflow-hidden rounded-control w-50 shadow-[0_8px_28px_var(--cst-elevation-shadow-dark)] border border-border-default"   style="background:var(--cst-surface-primary)">
					{#each [{ type: 'user', label: '用户', icon: markerUserUrl, clusterIcon: markerUserClusterUrl, color: PIN_COLORS.user, count: userCount }, { type: 'group', label: '群组', icon: markerGroupUrl, clusterIcon: markerGroupClusterUrl, color: PIN_COLORS.group, count: groupCount }, { type: 'moment', label: '瞬间', icon: markerMomentUrl, clusterIcon: markerMomentClusterUrl, color: PIN_COLORS.moment, count: momentCount }] as item, i}
						{#if i > 0}
							<div class="h-0.25"  style="background:var(--cst-border-default)"></div>
						{/if}
						{@const active = activeFilters.has(item.type as PinType)}
						<button
							type="button"
							class="hover-btn flex w-full items-center gap-2 px-3 py-2.5 text-left outline-none"
							style="opacity:{active ? 1 : 0.45};"
							onclick={() => toggleFilter(item.type as PinType)}
						>
							<img class="w-3.75 h-5.25" src={item.icon} alt=""  />
							<img class="w-3.75 h-5.25" src={item.clusterIcon} alt=""  style="opacity:0.65" />
							<span class="flex-1 text-text-primary text-xs" >{item.label}</span>
							<span style="font-size:12px;color:{item.color};font-weight:600;">{item.count}</span>
							<span class="flex items-center justify-center shrink-0" style="width:14px;height:14px;border-radius:4px;background:{active ? item.color : 'transparent'};border:1.5px solid {active ? item.color : 'var(--cst-border-strong)'};transition:all 0.18s;">
								{#if active}
									<Check class="size-2.5"  style="color:var(--cst-content-on-marker)" strokeWidth={3} />
								{/if}
							</span>
						</button>
					{/each}
					<div class="h-0.25"  style="background:var(--cst-border-default)"></div>
					<div class="flex items-center gap-2 px-3 py-2">
						<span style="width:13px;height:13px;border-radius:50%;background:{PIN_COLORS.user};border:2px solid var(--cst-surface-primary);box-shadow:0 0 5px {pinTone('user', 50)};"></span>
						<span class="text-text-secondary text-xs" >我的位置</span>
					</div>
				</div>
			{/if}
		</div>


		<!-- Selected pin popup -->
		{#if selectedPin}
			<div
				data-map-popup
				class="bg-surface-primary"
				style="position:absolute;bottom:20px;left:20px;right:20px;z-index:25;border-radius:20px;padding:16px;box-shadow:0 0 0 1px var(--cst-border-default), 0 16px 48px var(--cst-elevation-shadow-dark);"
			>
				<div class="flex items-center gap-3">
					<PinMedia pin={selectedPin} size={52} markerWidth={26} markerHeight={35} />
					<div class="flex-1 min-w-0">
						<div class="flex items-center gap-2">
							<p class="text-text-primary text-base">{selectedPin.name || selectedPin.title}</p>
							{#if (selectedPin.count ?? 1) > 1}
								<span style="font-size:10px;padding:1px 6px;border-radius:4px;background:{PIN_COLORS[selectedPin.type]};color:var(--cst-content-on-marker);">
									{selectedPin.count}个
								</span>
							{/if}
						</div>
						<div class="flex items-center gap-2 mt-1 flex-wrap">
							<span style="font-size:11px;padding:1px 7px;border-radius:4px;background:{pinTone(selectedPin.type, 15)};color:{PIN_COLORS[selectedPin.type]};">
								{PIN_LABELS[selectedPin.type]}
							</span>
							{#if selectedPin.tag}
								<span class="text-text-secondary text-xs">#{selectedPin.tag}</span>
							{/if}
							{#if selectedPin.distance}
								<span class="text-text-secondary text-xs">{selectedPin.distance}</span>
							{/if}
							{#if selectedPin.members}
								<span class="text-text-secondary text-xs">{selectedPin.members}名成员</span>
							{/if}
							{#if selectedPin.likes}
								<span class="flex items-center gap-1">
									<Heart class="size-2.5"  style="color:var(--cst-status-danger)" strokeWidth={1.5} />
									<span class="text-text-secondary text-xs">{selectedPin.likes}</span>
								</span>
							{/if}
						</div>
					</div>
					<div class="flex gap-2 shrink-0">
						<button
							type="button"
							class="hover-btn flex items-center gap-1.5 px-4 py-2 rounded-full outline-none bg-accent-default text-text-on-accent"
							style="font-size:13px;box-shadow:0 4px 14px var(--cst-glow-accent);"
							onclick={() => selectedPin && openPinDetail(selectedPin)}
						>
							查看详情
							<ChevronRight class="size-3.25"  strokeWidth={2} />
						</button>
						<button
							type="button"
							class="hover-btn outline-none flex items-center justify-center rounded-full bg-surface-primary"
							style="width:36px;height:36px;border:1px solid var(--cst-border-default);box-shadow:var(--shadow-neu-xs);"
							onclick={() => (selectedPin = null)}
						>
							<X class="size-3.5"  style="color:var(--cst-text-secondary)" strokeWidth={2} />
						</button>
					</div>
				</div>
			</div>
		{/if}

		<!-- Attribution -->
		<div class="text-text-secondary text-[9px]"  style="position:absolute;bottom:4px;left:8px;opacity:0.38;pointer-events:none;z-index:1">
			© OpenStreetMap contributors
		</div>
	</div>
</div>
