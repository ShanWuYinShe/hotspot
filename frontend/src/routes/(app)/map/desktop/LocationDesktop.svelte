<script lang="ts">
	import { X, ChevronRight, Heart, Crosshair, Plus, Minus } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { ROUTES } from '@/lib/routes';
	import { startDefaultLocation } from '@/lib/start-default-location';
	import { SEARCH_DB, type PinType, type Pin, type LocationResult } from '../shared/location-data';
	import {
		PIN_COLORS, PIN_LABELS, PIN_TONES, fetchMapCanvas, fetchMapCanvasByViewport,
		type MapBounds, type MapHeatBucket
	} from '../shared/location-shared';
	import MapCanvas from '../shared/MapCanvas.svelte';
	import MapFilterMenu from '../shared/MapFilterMenu.svelte';
	import MapPinMarker from '../shared/MapPinMarker.svelte';
	import MapHeatDot from '../shared/MapHeatDot.svelte';
	import PinMedia from '../shared/PinMedia.svelte';
	import LocationDesktopPanel from './LocationDesktopPanel.svelte';

	// 桌面地图探索：左侧栏（统计/搜索/Tab 列表/确认，由 LocationDesktopPanel 承载）
	// + 地图面板（OSM iframe + 图钉 + 筛选 + 详情浮层）。
	// 图钉与统计来自 POST /public/v1/map/canvas（spec: api_spec_map.md）。
	// 筛选菜单由 map/shared/MapFilterMenu 统一承载（关闭行为/动画/重置/至少一项）。

	let searchQuery = $state('');
	let selectedPin = $state<Pin | null>(null);
	let selectedLoc = $state<LocationResult | null>(null);
	let activeFilters = $state<Set<PinType>>(new Set(['user', 'group', 'moment']));
	let activeTab = $state<'nearby' | 'hot' | 'recent'>('nearby');
	// 瓦片缩放级别（13≈全市 / 16≈街区）；数据请求半径仍由 fetchMapCanvas 内部约定
	let mapZoom = $state(14);
	let mapCenter = $state<{ lat: number; lng: number } | null>(null);
	let mapBounds = $state<MapBounds>({ minLat: 0, maxLat: 0, minLng: 0, maxLng: 0 });
	// 数据请求用稳定边界（手势/缩放结束才回写）；图钉/热力跟随用每帧 mapBounds
	let mapStableBounds = $state<MapBounds>({ minLat: 0, maxLat: 0, minLng: 0, maxLng: 0 });
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
	let mapError = $state<string | null>(null);
	let mapReloadKey = $state(0);

	// 视口中心/边界或筛选变化时重新拉取。依赖 stableBounds（手势/缩放结束才回写），
	// 拖拽中的逐帧 bounds 变化只驱动图钉定位，不触发请求。
	$effect(() => {
		void mapReloadKey;
		const center = mapCenter;
		const types = [...activeFilters];
		const bounds = mapStableBounds;
		if (!center) return;
		let cancelled = false;
		mapLoading = true;
		mapError = null;
		// TODO: 固定 delta 仅用于首屏，后续应接入视口 bounds（当前兜底：mapBounds 尚未就绪时用固定 0.045，hasViewport 为真时已用真实视口）
		const hasViewport = bounds.maxLat !== bounds.minLat || bounds.maxLng !== bounds.minLng;
		const request = hasViewport
			? fetchMapCanvasByViewport({ bounds: { north: bounds.maxLat, south: bounds.minLat, east: bounds.maxLng, west: bounds.minLng } }, types)
			: fetchMapCanvas(center, 0.045, types);
		void request
			.then((r) => {
				if (cancelled) return;
				mapPins = r.pins;
				heatBuckets = r.heatBuckets;
				mapCounts = r.counts;
				mapLoading = false;
			})
			.catch((err: unknown) => {
				if (cancelled) return;
				// 无 catch 时请求失败会让 mapLoading 永久卡在 true、地图一片空白
				mapError = err instanceof Error ? err.message : '地图数据加载失败';
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

	const searchResults = $derived(
		searchQuery.trim() ? SEARCH_DB.filter((s) => s.displayName.includes(searchQuery)).slice(0, 6) : []
	);
	const nearbyPins = $derived(mapPins.filter((p) => (p.count ?? 1) === 1));

	// 底图视口地理边界由 MapCanvas 墨卡托换算后双向绑定（图钉/热力定位用）

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
		// 深链直入时无站内历史，back 会离开站点——回退到首页
		if (window.history.length > 1 && document.referrer.startsWith(window.location.origin)) history.back();
		else void goto(ROUTES.home);
	}
</script>

<div class="flex h-full bg-surface-page">
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
	<!-- 地图空白点击由 MapCanvas onMapTap 承担（内部区分拖拽/原地点击，拖拽后不再误清选中）；
	     筛选面板关闭由 MapFilterMenu 的 Popover 外点机制承担；Escape 清选中见 svelte:window -->
	<div class="flex-1 relative overflow-hidden">
		{#if mapCenter}
		<MapCanvas bind:center={mapCenter} bind:zoom={mapZoom} bind:bounds={mapBounds} bind:stableBounds={mapStableBounds} onMapTap={() => (selectedPin = null)} />

		<!-- Current location dot -->
		<div style="position:absolute;left:50%;top:50%;transform:translate(-50%,-50%);pointer-events:none;z-index:5;">
			<div class="relative rounded-full" style="width:18px;height:18px;background:{PIN_COLORS.user};border:var(--cst-border-width-heavy) solid var(--cst-surface-primary);box-shadow:var(--cst-marker-shadow) {PIN_TONES.user.glow};">
				<span class="absolute rounded-full" style="inset:-8px;background:{PIN_TONES.user.bg};border:var(--cst-border-width-hairline) solid {PIN_TONES.user.border};"></span>
			</div>
		</div>

		<!-- Heatmap buckets（大视口 / 高密度时服务端返回热力统计） -->
		{#each heatBuckets as bucket (bucket.cell_token)}
			<MapHeatDot {bucket} bounds={mapBounds} />
		{/each}

		<!-- Pin markers -->
		{#each mapPins as pin (pin.id)}
			<MapPinMarker
				pin={pin}
				bounds={mapBounds}
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
			<div role="status" aria-live="polite" class="absolute top-3 left-1/2 -translate-x-1/2 z-20 rounded-full bg-surface-primary px-3 py-1 text-xs text-text-secondary shadow-raised-sm border border-border-default" >
				加载地图数据…
			</div>
		{:else if mapError}
			<!-- 加载失败：给出原因与重试，避免空白地图被误认为「附近什么都没有」 -->
			<div role="alert" class="absolute top-3 left-1/2 -translate-x-1/2 z-20 flex items-center gap-2 rounded-full bg-surface-primary px-3 py-1.5 text-xs text-status-danger shadow-raised-sm border border-border-default" >
				<span>{mapError}</span>
				<button type="button" class="link-btn focus-ring rounded-sm text-accent-default" onclick={() => mapReloadKey++}>重试</button>
			</div>
		{/if}

		<!-- 缩放控制（右下角；画布拖拽/滚轮亦可缩放）。分段圆角由按钮自身承担，
			不用 overflow-hidden 裁剪（会连带裁掉键盘焦点环），hover 填充与分隔线齐平。
			按钮不用 hover-btn：分段组合控件中点击 0.94 缩放会让高亮块与整卡脱开，
			hover/active 填充由全局 button 规则承载 -->
		<div class="absolute z-30 flex flex-col rounded-lg border border-border-default bg-surface-primary" style="right:14px;bottom:20px;box-shadow:var(--cst-elevation-fab);">
			<button
				type="button"
				class="flex items-center justify-center size-9 rounded-t-lg focus-ring text-text-primary disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-faint"
				style="border-bottom:var(--cst-border-width-hairline) solid var(--cst-border-default);"
				aria-label="放大"
				disabled={mapZoom >= 16}
				onclick={() => (mapZoom = Math.min(16, mapZoom + 1))}
			>
				<Plus class="size-4" strokeWidth={1.75} />
			</button>
			<button
				type="button"
				class="flex items-center justify-center size-9 rounded-b-lg focus-ring text-text-primary disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-faint"
				aria-label="缩小"
				disabled={mapZoom <= 13}
				onclick={() => (mapZoom = Math.max(13, mapZoom - 1))}
			>
				<Minus class="size-4" strokeWidth={1.75} />
			</button>
		</div>

		<!-- 左上筛选菜单（桌面/移动共用组件；含计数徽章、至少一项约束与全部显示重置） -->
		<div class="absolute z-20" style="top:14px;left:14px;">
			<MapFilterMenu bind:activeFilters counts={mapCounts} showLegend align="start" />
		</div>

		<!-- 定位到当前：地图区域右上角，独立于标题栏 -->
		<button
			type="button"
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
								<span class="text-xs" style="padding:1px 6px;border-radius:var(--cst-shape-sm);background:{PIN_COLORS[selectedPin.type]};color:var(--cst-content-on-marker);">
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
							class="hover-btn flex items-center gap-1.5 px-4 py-2 rounded-full focus-ring bg-accent-default text-text-on-accent text-sm shadow-raised-sm hover:shadow-raised-md"
							onclick={() => selectedPin && openPinDetail(selectedPin)}
						>
							查看详情
							<ChevronRight class="size-3.5"  strokeWidth={2} />
						</button>
						<button
							type="button"
							class="hover-btn flex items-center justify-center rounded-full border border-border-default bg-surface-primary size-9 shadow-raised-sm hover:bg-accent-soft focus-ring"
							aria-label="关闭详情"
							onclick={() => (selectedPin = null)}
						>
							<X class="size-3.5 text-text-secondary" strokeWidth={2} />
						</button>
					</div>
				</div>
			</div>
		{/if}
	</div>
</div>

<svelte:window onkeydown={(e) => { if (e.key === 'Escape') selectedPin = null; }} />
