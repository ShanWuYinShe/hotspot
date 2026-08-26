<script lang="ts">
	import { Search, MapPin, X, SlidersHorizontal, Crosshair, Check, ArrowLeft, Plus, Minus } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { ROUTES } from '@/lib/routes';
	import * as Popover from '@/components/ui/popover';
	import { startDefaultLocation } from '@/lib/start-default-location';
	import { RECENT_LOCATIONS, HOT_LOCATIONS, SEARCH_DB, type PinType, type Pin, type LocationResult } from '../shared/location-data';
	import { SNAP_MIN, SNAP_MID, SNAP_MAX, SNAP_HEIGHTS, PIN_COLORS, PIN_TONES, markerSvg, floatBtnStyle, fetchMapCanvas, type MapBounds, type MapHeatBucket, type SnapState } from '../shared/location-shared';
	import MapCanvas from '../shared/MapCanvas.svelte';
	import MapPinMarker from '../shared/MapPinMarker.svelte';
	import MapHeatDot from '../shared/MapHeatDot.svelte';
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
	// 瓦片缩放级别；数据请求半径由 fetchMapCanvas 内部约定
	let mapZoom = $state(14);
	let mapBounds = $state<MapBounds>({ minLat: 0, maxLat: 0, minLng: 0, maxLng: 0 });
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
		void fetchMapCanvas(center, 0.045, types).then((r) => {
			if (cancelled) return;
			mapPins = r.pins;
			heatBuckets = r.heatBuckets;
			mapLoading = false;
		});
		return () => {
			cancelled = true;
		};
	});

	// 底图视口地理边界由 MapCanvas 墨卡托换算后双向绑定（图钉/热力定位用）
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
		`position:absolute;bottom:0;left:0;right:0;height:${sheetH}px;z-index:20;border-radius:var(--radius-sheet) var(--radius-sheet) 0 0;border:var(--cst-border-width-hairline) solid var(--cst-border-default);border-bottom:none;box-shadow:var(--cst-elevation-sheet);display:flex;flex-direction:column;overflow:hidden;${dragging ? '' : 'transition:height var(--cst-motion-spring) var(--cst-ease-bounce);'}`
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
		{#if mapCenter}
		<MapCanvas bind:center={mapCenter} bind:zoom={mapZoom} bind:bounds={mapBounds} />

		<!-- Current location pulsing dot（对齐 Figma Make：halo scale 1→1.8 / opacity 0.55→0 循环） -->
		<div style="position:absolute;left:50%;top:46%;transform:translate(-50%,-50%);z-index:5;pointer-events:none;">
			<div class="relative rounded-full" style="width:18px;height:18px;background:{PIN_COLORS.user};border:var(--cst-border-width-heavy) solid var(--cst-surface-primary);box-shadow:var(--cst-marker-shadow) {PIN_TONES.user.glow};">
				<span class="map-pulse-halo absolute rounded-full" style="inset:-7px;background:{PIN_TONES.user.bg};border:var(--cst-border-width-hairline) solid {PIN_TONES.user.border};"></span>
			</div>
		</div>

		<!-- Heatmap buckets（大视口 / 高密度时服务端返回热力统计） -->
		{#each heatBuckets as bucket (bucket.cell_token)}
			<MapHeatDot {bucket} bounds={mapBounds} />
		{/each}

		<!-- Marker pins -->
		{#each mapPins as pin (pin.id)}
			<MapPinMarker
				pin={pin}
				bounds={mapBounds}
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
						style="padding:9px 16px;border:{showFilter ? 'var(--cst-border-width-hairline) solid transparent' : 'var(--cst-border-width-hairline) solid var(--cst-border-default)'};box-shadow:{showFilter ? 'var(--shadow-glow-md)' : 'var(--cst-elevation-fab)'};"
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
						class="w-64 overflow-hidden rounded-xl p-0"
					>
						<div class="flex items-center justify-between border-b border-border-default px-3.5 py-2.5">
							<span class="text-sm font-medium text-text-primary">显示类型</span>
							<span class="text-xxs text-text-secondary">{activeFilters.size}/3 已选</span>
						</div>
						<div class="flex flex-col gap-1 p-1.5">
							{#each [
								['user', '用户'],
								['group', '群组'],
								['moment', '瞬间']
							] as [type, label]}
								{@const active = activeFilters.has(type as PinType)}
								<button
									type="button"
									class="group flex w-full items-center gap-3 rounded-lg px-2.5 py-2 text-left focus-ring transition-colors {active ? 'bg-accent-soft' : 'hover:bg-interactive-hover'}"
									onclick={() => toggleFilter(type as PinType)}
								>
									<span
										class="inline-flex size-8 shrink-0 items-center justify-center rounded-lg"
										style="background:{active ? PIN_TONES[type as PinType].bg : 'var(--cst-surface-subtle)'};border:var(--cst-border-width-hairline) solid {active ? PIN_TONES[type as PinType].border : 'transparent'};"
									>
										{@html markerSvg(type as PinType, 1, { w: 14, h: 19 })}
									</span>
									<span class="min-w-0 flex-1">
										<span class="block text-sm text-text-primary">{label}</span>
										<span class="mt-0.5 block text-xxs text-text-secondary">{active ? '正在显示' : '已隐藏'}</span>
									</span>
									<span
										class="flex size-5 shrink-0 items-center justify-center rounded-full transition-colors {active ? 'bg-accent-default text-text-on-accent' : 'bg-border-default text-transparent'}"
									>
										<Check class="size-3" strokeWidth={3} />
									</span>
								</button>
							{/each}
						</div>
					</Popover.Content>
				</Popover.Root>
		</div>
	</div>

	<!-- ══ RIGHT CONTROLS — bottom tracks sheet ══ -->
	<div style="position:absolute;right:12px;bottom:{rightCtrlBottom}px;z-index:35;display:flex;flex-direction:column;gap:8px;align-items:center;transition:bottom var(--cst-motion-spring) var(--cst-ease-bounce);">
		<!-- Zoom in/out（画布拖拽/双指外滚轮亦可缩放）。分段圆角由按钮自身承担，
			不用 overflow-hidden 裁剪（会连带裁掉键盘焦点环）。按钮不用 hover-btn：
			分段组合控件中点击 0.94 缩放会让高亮块与整卡脱开 -->
		<div class="flex flex-col rounded-lg" style="border:var(--cst-border-width-hairline) solid var(--cst-border-default);{floatBtnStyle}">
			<button
				type="button"
				class="flex items-center justify-center size-9 rounded-t-lg focus-ring text-text-primary"
				style="border-bottom:var(--cst-border-width-hairline) solid var(--cst-border-default);"
				aria-label="放大"
				disabled={mapZoom >= 16}
				onclick={() => (mapZoom = Math.min(16, mapZoom + 1))}
			>
				<Plus class="size-4" strokeWidth={1.75} />
			</button>
			<button
				type="button"
				class="flex items-center justify-center size-9 rounded-b-lg focus-ring text-text-primary"
				aria-label="缩小"
				disabled={mapZoom <= 13}
				onclick={() => (mapZoom = Math.max(13, mapZoom - 1))}
			>
				<Minus class="size-4" strokeWidth={1.75} />
			</button>
		</div>

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
				class="rounded-full"
				style="width:14px;height:14px;background:var(--cst-marker-user);border:var(--cst-border-width-heavy) solid var(--cst-surface-primary);box-shadow:var(--cst-marker-halo) var(--cst-marker-user-glow);"
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
			<div class="bg-border-default w-9.5 h-1 rounded-full" ></div>
		</div>

		<!-- Search bar -->
		<div class="shrink-0 px-4 pb-2">
			<div class="flex items-center gap-2.5 px-3.5 rounded-md input-surface h-11" >
				<Search class="size-3.5 shrink-0 text-text-secondary"   strokeWidth={1.5} />
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
					class="flex-1 bg-transparent text-text-primary text-sm"
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
			<div style="flex:1;min-height:0;overflow-y:auto;" class="no-scrollbar pb-3">
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
		<div class="shrink-0 px-4 pt-2 pb-4.5">
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
