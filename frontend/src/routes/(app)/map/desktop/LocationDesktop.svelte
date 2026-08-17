<script lang="ts">
	import { X, ChevronRight, SlidersHorizontal, Heart, Check, Crosshair } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { ROUTES } from '@/lib/routes';
	import { startDefaultLocation } from '@/lib/start-default-location';
	import { SEARCH_DB, type PinType, type Pin, type LocationResult } from '../shared/location-data';
	import {
		PIN_COLORS, PIN_LABELS, PIN_TONES, markerSvg, buildMapData, fetchMapCanvas,
		type MapHeatBucket
	} from '../shared/location-shared';
	import MapPinMarker from '../shared/MapPinMarker.svelte';
	import MapHeatDot from '../shared/MapHeatDot.svelte';
	import PinMedia from '../shared/PinMedia.svelte';
	import LocationDesktopPanel from './LocationDesktopPanel.svelte';

	// 桌面地图探索：左侧栏（统计/搜索/Tab 列表/确认，由 LocationDesktopPanel 承载）
	// + 地图面板（OSM iframe + 图钉 + 筛选 + 详情浮层）。
	// 图钉与统计来自 POST /public/v1/map/canvas（spec: api_spec_map.md）。

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
			<div class="relative rounded-full" style="width:18px;height:18px;background:{PIN_COLORS.user};border:var(--cst-border-width-heavy) solid var(--cst-surface-primary);box-shadow:var(--cst-marker-shadow) {PIN_TONES.user.glow};">
				<span class="absolute rounded-full" style="inset:-8px;background:{PIN_TONES.user.bg};border:var(--cst-border-width-hairline) solid {PIN_TONES.user.border};"></span>
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
			<div class="absolute top-3 left-1/2 -translate-x-1/2 z-20 rounded-full bg-surface-primary px-3 py-1 text-xs text-text-secondary shadow-raised-sm border border-border-default" >
				加载地图数据…
			</div>
		{/if}

		<!-- 左上筛选浮动按钮 + 下拉（对齐 Figma Make LocationDesktop 筛选面板） -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<div data-map-filter class="absolute z-20" style="top:14px;left:14px;" onclick={(e) => e.stopPropagation()}>
			<button
				type="button"
				class="hover-btn scale-btn flex items-center gap-2 rounded-full focus-ring text-sm"
				style="padding:9px 16px;background:{showFilter ? 'var(--cst-accent-default)' : 'var(--cst-surface-primary)'};color:{showFilter ? 'var(--cst-text-on-accent)' : 'var(--cst-text-primary)'};border:var(--cst-border-width-hairline) solid {showFilter ? 'transparent' : 'var(--cst-border-default)'};box-shadow:{showFilter ? 'var(--shadow-glow-md)' : 'var(--cst-elevation-fab)'};"
				onclick={() => (showFilter = !showFilter)}
			>
				<SlidersHorizontal class="size-3.5"  strokeWidth={showFilter ? 2.5 : 1.75} />
				筛选
				{#if activeFilters.size < 3}
					<span class="text-xxs font-medium rounded-full" style="margin-left:2px;padding:1px 6px;background:{showFilter ? 'var(--cst-surface-map-control-active)' : 'var(--cst-accent-soft)'};color:{showFilter ? 'var(--cst-text-on-accent)' : 'var(--cst-accent-default)'};">{activeFilters.size}</span>
				{/if}
			</button>
			{#if showFilter}
				<div class="mt-2 w-64 overflow-hidden rounded-xl border border-border-default bg-surface-primary shadow-raised-lg">
					<div class="flex items-center justify-between border-b border-border-default px-3.5 py-2.5">
						<span class="text-sm font-medium text-text-primary">显示类型</span>
						<span class="text-xxs text-text-secondary">{activeFilters.size}/3 已选</span>
					</div>
					<div class="p-1.5">
						{#each [{ type: 'user', label: '用户', count: userCount }, { type: 'group', label: '群组', count: groupCount }, { type: 'moment', label: '瞬间', count: momentCount }] as item}
							{@const active = activeFilters.has(item.type as PinType)}
							<button
								type="button"
								class="group flex w-full items-center gap-3 rounded-lg px-2.5 py-2 text-left focus-ring transition-colors {active ? 'bg-accent-soft' : 'hover:bg-interactive-hover'}"
								onclick={() => toggleFilter(item.type as PinType)}
							>
								<span
									class="inline-flex size-8 shrink-0 items-center justify-center rounded-lg"
									style="background:{active ? PIN_TONES[item.type as PinType].bg : 'var(--cst-surface-subtle)'};border:var(--cst-border-width-hairline) solid {active ? PIN_TONES[item.type as PinType].border : 'transparent'};"
								>
									{@html markerSvg(item.type as PinType, 1, { w: 14, h: 19 })}
								</span>
								<span class="min-w-0 flex-1">
									<span class="block text-sm text-text-primary">{item.label}</span>
									<span class="mt-0.5 block text-xxs text-text-secondary">{active ? '正在显示' : '已隐藏'}</span>
								</span>
								{#if item.count}
									<span class="shrink-0 text-xs text-text-secondary">{item.count}</span>
								{/if}
								<span
									class="flex size-5 shrink-0 items-center justify-center rounded-full transition-colors {active ? 'bg-accent-default text-text-on-accent' : 'bg-border-default text-transparent'}"
								>
									<Check class="size-3" strokeWidth={3} />
								</span>
							</button>
						{/each}
					</div>
					<div class="flex items-center gap-2 border-t border-border-default px-3.5 py-2.5">
						<span class="size-3 rounded-full" style="background:{PIN_COLORS.user};border:var(--cst-border-width-strong) solid var(--cst-surface-primary);box-shadow:var(--cst-marker-halo) {PIN_TONES.user.glow};"></span>
						<span class="text-xs text-text-secondary">我的位置</span>
					</div>
				</div>
			{/if}
		</div>

		<!-- 定位到当前：地图区域右上角，独立于标题栏 -->
		<button
			type="button"
			data-map-filter
			class="hover-btn scale-btn absolute z-20 flex items-center justify-center rounded-full focus-ring size-10"
			style="top:14px;right:14px;background:var(--cst-surface-primary);color:var(--cst-text-primary);border:var(--cst-border-width-hairline) solid var(--cst-border-default);box-shadow:var(--cst-elevation-fab);"
			aria-label="定位到当前位置"
			title="定位到当前位置"
			onclick={(e) => {
				e.stopPropagation();
				mapCenter = defaultCenter;
				selectedLoc = null;
				searchQuery = '';
			}}
		>
			<Crosshair class="size-4 text-accent-default" strokeWidth={1.75} />
		</button>


		<!-- Selected pin popup -->
		{#if selectedPin}
			<div
				data-map-popup
				class="bg-surface-primary"
				style="position:absolute;bottom:20px;left:20px;right:20px;z-index:25;border-radius:var(--cst-shape-lg);padding:16px;box-shadow:var(--cst-elevation-modal);"
			>
				<div class="flex items-center gap-3">
					<PinMedia pin={selectedPin} size={52} markerWidth={26} markerHeight={35} />
					<div class="flex-1 min-w-0">
						<div class="flex items-center gap-2">
							<p class="text-text-primary text-sm">{selectedPin.name || selectedPin.title}</p>
							{#if (selectedPin.count ?? 1) > 1}
								<span class="text-xxs" style="padding:1px 6px;border-radius:var(--cst-shape-sm);background:{PIN_COLORS[selectedPin.type]};color:var(--cst-content-on-marker);">
									{selectedPin.count}个
								</span>
							{/if}
						</div>
						<div class="flex items-center gap-2 mt-1 flex-wrap">
							<span class="text-xs" style="padding:1px 7px;border-radius:var(--cst-shape-sm);background:{PIN_TONES[selectedPin.type].bg};color:{PIN_COLORS[selectedPin.type]};">
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
									<Heart class="size-2.5 text-status-danger"   strokeWidth={1.5} />
									<span class="text-text-secondary text-xs">{selectedPin.likes}</span>
								</span>
							{/if}
						</div>
					</div>
					<div class="flex gap-2 shrink-0">
						<button
							type="button"
							class="hover-btn flex items-center gap-1.5 px-4 py-2 rounded-full focus-ring bg-accent-default text-text-on-accent text-sm"
							style="box-shadow:var(--shadow-glow-md);"
							onclick={() => selectedPin && openPinDetail(selectedPin)}
						>
							查看详情
							<ChevronRight class="size-3.5"  strokeWidth={2} />
						</button>
						<button
							type="button"
							class="hover-btn flex items-center justify-center rounded-full border border-border-default bg-surface-primary size-9 shadow-raised-sm focus-ring"
							onclick={() => (selectedPin = null)}
						>
							<X class="size-3.5 text-text-secondary"   strokeWidth={2} />
						</button>
					</div>
				</div>
			</div>
		{/if}

		<!-- Attribution -->
		<div class="text-text-secondary text-xxs opacity-subtle"  style="position:absolute;bottom:4px;left:8px;pointer-events:none;z-index:1">
			© OpenStreetMap contributors
		</div>
	</div>
</div>
