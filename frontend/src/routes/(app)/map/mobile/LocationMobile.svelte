<script lang="ts">
	import { Search, MapPin, X, SlidersHorizontal, Crosshair, ArrowLeft } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { ROUTES } from '@/lib/routes';
	import * as Popover from '@/components/ui/popover';
	import { startDefaultLocation } from '@/lib/start-default-location';
	import { RECENT_LOCATIONS, HOT_LOCATIONS, SEARCH_DB, type PinType, type Pin, type LocationResult } from '../shared/location-data';
	import { SNAP_MIN, SNAP_MID, SNAP_MAX, SNAP_HEIGHTS, PIN_COLORS, PIN_TONES, markerSvg, floatBtnStyle, buildMapData, fetchMapCanvas, type MapHeatBucket, type SnapState } from '../shared/location-shared';
	import MapPinMarker from '../shared/MapPinMarker.svelte';
	import MapHeatDot from '../shared/MapHeatDot.svelte';
	import TagPill from '@/components/forms/TagPill.svelte';
	import LocationSheetList from './LocationSheetList.svelte';

	// 移动端地图探索：全屏 OSM + 可拖拽吸附底部 Sheet（148/318/540 三档）。
	// motion spring 吸附改 CSS transition。

	/* ── Sheet drag ── */
	let sheetH = $state(SNAP_MID);
	let dragging = $state(false);
	let snap = $state<SnapState>('mid');
	let dragState = $state({ active: false, startY: 0, startH: SNAP_MID, velY: 0, prevY: 0, prevT: 0 });

	function snapTo(target: SnapState) {
		snap = target;
		sheetH = SNAP_HEIGHTS[target];
	}

	function resolveSnap(h: number, vel: number): SnapState {
		const proj = h + vel * 90;
		if (proj < (SNAP_MIN + SNAP_MID) / 2) return 'min';
		if (proj < (SNAP_MID + SNAP_MAX) / 2) return 'mid';
		return 'max';
	}

	function onHandlePointerDown(e: PointerEvent) {
		e.preventDefault();
		const now = Date.now();
		dragging = true;
		dragState = { active: true, startY: e.clientY, startH: sheetH, velY: 0, prevY: e.clientY, prevT: now };

		function onMove(ev: PointerEvent) {
			if (!dragState.active) return;
			const t = Date.now();
			const dt = t - dragState.prevT || 1;
			dragState.velY = (dragState.prevY - ev.clientY) / dt;
			dragState.prevY = ev.clientY;
			dragState.prevT = t;
			const delta = dragState.startY - ev.clientY;
			sheetH = Math.max(SNAP_MIN - 10, Math.min(SNAP_MAX + 10, dragState.startH + delta));
		}

		function onUp() {
			dragState.active = false;
			dragging = false;
			window.removeEventListener('pointermove', onMove);
			window.removeEventListener('pointerup', onUp);
			snapTo(resolveSnap(sheetH, dragState.velY));
		}

		window.addEventListener('pointermove', onMove);
		window.addEventListener('pointerup', onUp);
	}

	/** 键盘调整面板高度（拖拽的键盘等价操作）。 */
	function onHandleKeydown(e: KeyboardEvent) {
		if (e.target !== e.currentTarget) return;
		const steps: SnapState[] = ['min', 'mid', 'max'];
		const idx = steps.indexOf(snap);
		if (e.key === 'ArrowUp') {
			e.preventDefault();
			snapTo(steps[Math.min(idx + 1, steps.length - 1)]);
		} else if (e.key === 'ArrowDown') {
			e.preventDefault();
			snapTo(steps[Math.max(idx - 1, 0)]);
		}
	}

	/* ── Content state ── */
	let showFilter = $state(false);
	let filterAnchorEl = $state<HTMLElement | null>(null);
	let activeFilters = $state<Set<PinType>>(new Set(['user', 'group', 'moment']));
	let searchQuery = $state('');
	let selectedPin = $state<Pin | null>(null);
	let selectedLoc = $state<LocationResult | null>(null);
	const zoom = 0.045;
	let mapCenter = $state<{ lat: number; lng: number } | null>(null);
	let defaultCenter = $state<{ lat: number; lng: number } | null>(null);
	let recentItems = $state(RECENT_LOCATIONS);
	let activeTab = $state<'recent' | 'hot'>('recent');

	// 初始/复位中心来自 ClientConfig default_location（MSW mock），不写死坐标。
	$effect(() => startDefaultLocation((loc) => {
						const point = loc?.point ? { lat: loc.point.latitude, lng: loc.point.longitude } : null;
			defaultCenter = point;
			if (!mapCenter) mapCenter = point;
}));

	// map/canvas 数据
	let mapPins = $state<Pin[]>([]);
	let heatBuckets = $state<MapHeatBucket[]>([]);
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
			mapLoading = false;
		});
		return () => {
			cancelled = true;
		};
	});

	const mapData = $derived(mapCenter ? buildMapData(mapCenter, zoom) : null);
	const searchResults = $derived(searchQuery.trim() ? SEARCH_DB.filter((s) => s.displayName.includes(searchQuery)).slice(0, 5) : []);

	function handleMapTap() {
		selectedPin = null;
		showFilter = false;
		snapTo('min');
	}

	function handleSelectLoc(loc: LocationResult) {
		selectedLoc = loc;
		searchQuery = loc.displayName;
		mapCenter = { lat: loc.lat, lng: loc.lng };
		selectedPin = null;
		if (snap === 'max') snapTo('mid');
	}

	function handleConfirm() {
		const loc = selectedLoc ?? { displayName: '当前位置', lat: mapCenter?.lat ?? 0, lng: mapCenter?.lng ?? 0 };
		try {
			localStorage.setItem('hotspot_last_location', JSON.stringify([loc.lat, loc.lng]));
		} catch {}
		history.back();
	}

	function toggleFilter(type: PinType) {
		const next = new Set(activeFilters);
		next.has(type) ? next.delete(type) : next.add(type);
		activeFilters = next;
	}

	// 图钉详情跳转：user/group/moment 分别路由到对应详情页
	function openPinDetail(pin: Pin) {
		if (pin.type === 'user') void goto(ROUTES.user(pin.id));
		else if (pin.type === 'group') void goto(ROUTES.group(pin.id));
		else void goto(ROUTES.moment(pin.id));
	}

	const sheetStyle = $derived(
		`position:absolute;bottom:0;left:0;right:0;height:${sheetH}px;z-index:20;border-radius:var(--radius-sheet) var(--radius-sheet) 0 0;border:1px solid var(--cst-border-default);border-bottom:none;box-shadow:var(--cst-elevation-sheet);display:flex;flex-direction:column;overflow:hidden;${dragging ? '' : 'transition:height 0.3s var(--cst-ease-bounce);'}`
	);
	const rightCtrlBottom = $derived(sheetH + 10);
</script>

<div class="bg-surface-primary" style="position:relative;width:100%;height:100%;overflow:hidden;">
	<h1 class="sr-only">地图</h1>
	<!-- ══ MAP (full screen, tappable) ══ -->
	<div
		style="position:absolute;inset:0;z-index:0;"
		role="button"
		tabindex="0"
		aria-label="地图（点击空白处收起面板）"
		onclick={handleMapTap}
		onkeydown={(e) => {
			if (e.target !== e.currentTarget) return;
			if (e.key === 'Enter' || e.key === ' ' || e.key === 'Escape') {
				e.preventDefault();
				handleMapTap();
			}
		}}
	>
		{#if mapData}
		<iframe src={mapData.url} title="地图" style="width:100%;height:100%;border:0;pointer-events:none;"></iframe>

		<!-- Current location pulsing dot（对齐 Figma Make：halo scale 1→1.8 / opacity 0.55→0 循环） -->
		<div style="position:absolute;left:50%;top:46%;transform:translate(-50%,-50%);z-index:5;pointer-events:none;">
			<div style="width:18px;height:18px;border-radius:50%;background:{PIN_COLORS.user};border:3px solid var(--cst-surface-primary);box-shadow:var(--cst-marker-shadow) {PIN_TONES.user.glow};position:relative;">
				<span class="map-pulse-halo absolute rounded-full" style="inset:-7px;background:{PIN_TONES.user.bg};border:1px solid {PIN_TONES.user.border};"></span>
			</div>
		</div>

		<!-- Heatmap buckets（大视口 / 高密度时服务端返回热力统计） -->
		{#each heatBuckets as bucket (bucket.cell_token)}
			<MapHeatDot {bucket} bounds={mapData.bounds} />
		{/each}

		<!-- Marker pins -->
		{#each mapPins as pin (pin.id)}
			<MapPinMarker
				pin={pin}
				bounds={mapData.bounds}
				isSelected={selectedPin?.id === pin.id}
				onSelect={(e) => {
					e.stopPropagation();
					selectedPin = selectedPin?.id === pin.id ? null : pin;
					if (snap === 'min') snapTo('mid');
				}}
			/>
		{/each}
		{/if}

		{#if mapLoading}
			<div class="absolute top-16 left-1/2 -translate-x-1/2 z-20 rounded-full bg-surface-primary px-3 py-1 text-xs text-text-secondary shadow-raised-sm border border-border-default" >
				加载地图数据…
			</div>
		{/if}

		<!-- Attribution -->
		<div class="text-text-secondary text-xxs opacity-40"  style="position:absolute;bottom:4px;left:6px;pointer-events:none;z-index:1">
			© OpenStreetMap contributors
		</div>
	</div>

	<!-- ══ TOP BAR (fixed, floating over map — 对齐 Figma Make：无标题，仅返回 + 筛选) ══ -->
		<div style="position:absolute;top:0;left:0;right:0;z-index:50;padding:12px 14px 0;">
			<div style="display:flex;align-items:center;justify-content:space-between;">
				<!-- Back -->
				<button
					type="button"
					class="hover-btn flex items-center justify-center rounded-full focus-ring circle-hover size-10.5"
					style="{floatBtnStyle}"
					onclick={() => history.back()}
				>
					<ArrowLeft class="size-4.5 text-text-primary"   strokeWidth={2} />
				</button>

			<!-- Filter toggle -->
			<Popover.Root bind:open={showFilter}>
				<div bind:this={filterAnchorEl}>
					<button
						type="button"
						class="hover-btn flex items-center gap-2 rounded-full focus-ring text-md {showFilter ? 'bg-accent-default text-text-on-accent' : 'bg-surface-primary text-text-primary'}"
						style="padding:9px 16px;border:{showFilter ? '1px solid transparent' : '1px solid var(--cst-border-default)'};box-shadow:{showFilter ? 'var(--shadow-glow-md)' : 'var(--cst-elevation-fab)'};"
						onclick={() => (showFilter = !showFilter)}
					>
						<SlidersHorizontal class="size-3.5"  strokeWidth={showFilter ? 2.5 : 1.75} />
						筛选
					</button>
				</div>

					<!-- Filter legend panel -->
					<Popover.Content
						customAnchor={filterAnchorEl}
						side="bottom"
						align="end"
						sideOffset={8}
						class="w-56 overflow-hidden rounded-md p-2"
					>
						<div class="flex flex-wrap gap-1.5">
							{#each [
								['user', '用户'],
								['group', '群组'],
								['moment', '瞬间']
							] as [type, label]}
								{@const active = activeFilters.has(type as PinType)}
								<TagPill active={active} onClick={() => toggleFilter(type as PinType)}>
									{#snippet iconSnippet()}
										<span class="inline-flex shrink-0">{@html markerSvg(type as PinType, 1, { w: 14, h: 19 })}</span>
									{/snippet}
									{label}
								</TagPill>
							{/each}
						</div>
					</Popover.Content>
				</Popover.Root>
		</div>
	</div>

	<!-- ══ RIGHT CONTROLS — bottom tracks sheet ══ -->
	<div style="position:absolute;right:12px;bottom:{rightCtrlBottom}px;z-index:35;display:flex;flex-direction:column;gap:8px;align-items:center;transition:bottom 0.3s var(--cst-ease-bounce);">
		<!-- GPS re-center -->
		<button
			type="button"
			class="hover-btn flex items-center justify-center rounded-full focus-ring circle-hover size-10.5"
			style="{floatBtnStyle}"
			onclick={() => {
				mapCenter = defaultCenter;
				selectedLoc = null;
				searchQuery = '';
			}}
		>
			<Crosshair class="size-4.5 text-accent-default"   strokeWidth={1.75} />
		</button>

		<!-- My location dot（对齐 Figma Make：用户位置圆点 + 白描边 + 主题色 glow） -->
		<button
			type="button"
			aria-label="我的位置"
			class="hover-btn flex items-center justify-center rounded-full focus-ring circle-hover size-10.5"
			style="{floatBtnStyle}"
			onclick={() => {
				mapCenter = defaultCenter;
				selectedLoc = null;
				searchQuery = '';
			}}
		>
			<div
				style="width:14px;height:14px;border-radius:50%;background:var(--cst-marker-user);border:2.5px solid var(--cst-surface-primary);box-shadow:var(--cst-marker-halo) var(--cst-marker-user-glow);"
			></div>
		</button>
	</div>

	<!-- ══ BOTTOM SHEET (draggable) ══ -->
	<div class="bg-surface-primary" style={sheetStyle}>
		<!-- Drag handle -->
		<div
			role="slider"
			aria-label="面板高度"
			aria-orientation="vertical"
			aria-valuemin={SNAP_MIN}
			aria-valuemax={SNAP_MAX}
			aria-valuenow={Math.min(SNAP_MAX, Math.max(SNAP_MIN, sheetH))}
			tabindex="0"
			style="shrink:0;display:flex;flex-direction:column;align-items:center;padding-top:10px;padding-bottom:6px;cursor:grab;touch-action:none;user-select:none;"
			onpointerdown={onHandlePointerDown}
			onkeydown={onHandleKeydown}
		>
			<div class="bg-border-default w-9.5 h-1 rounded-[2px]" ></div>
		</div>

		<!-- Search bar -->
		<div class="shrink-0 px-3.5 pb-2">
			<div class="flex items-center gap-2.5 px-3.5 rounded-md input-surface h-11" >
				<Search class="size-4 shrink-0 text-text-secondary"   strokeWidth={1.5} />
				<input
					bind:value={searchQuery}
					oninput={(e) => {
						searchQuery = e.currentTarget.value;
						selectedLoc = null;
						if (snap === 'min') snapTo('mid');
					}}
					onfocus={() => {
						if (snap === 'min') snapTo('mid');
					}}
					placeholder="搜索地点、地标或坐标"
					class="flex-1 bg-transparent focus-ring text-text-primary text-sm"
				/>
				{#if searchQuery}
					<button
						type="button"
						class="hover-btn focus-ring"
						onclick={() => {
							searchQuery = '';
							selectedLoc = null;
						}}
					>
						<X class="size-3.5 text-text-secondary"   strokeWidth={2} />
					</button>
				{:else if selectedLoc}
					<Check class="size-3.5 shrink-0 text-accent-default"   strokeWidth={2.5} />
				{/if}
			</div>
		</div>

		<!-- Scrollable content — ONLY shown when snap is not "min" -->
		{#if snap !== 'min'}
			<div style="flex:1;min-height:0;overflow-y:auto;" class="no-scrollbar">
				<LocationSheetList
					bind:searchQuery
					{searchResults}
					bind:selectedPin
					bind:selectedLoc
					bind:activeTab
					bind:recentItems
					onSelectLoc={handleSelectLoc}
					onOpenPinDetail={openPinDetail}
					onClearPin={() => (selectedPin = null)}
				/>
			</div>
		{/if}

		<!-- Confirm button — always at bottom of sheet -->
		<div class="shrink-0 px-3.5 pt-2 pb-4.5">
			<button
				type="button"
				class="hover-btn w-full flex items-center justify-center gap-2 rounded-full focus-ring bg-accent-default text-text-on-accent text-md h-12.5"
				style="box-shadow:var(--shadow-glow-lg);"
				onclick={handleConfirm}
			>
				<MapPin class="size-4"  strokeWidth={2} />
				{selectedLoc ? `确认：${selectedLoc.displayName.slice(0, 11)}` : '设为浏览锚点'}
			</button>
		</div>
	</div>
</div>
