<script lang="ts">
	import { Search, MapPin, X, SlidersHorizontal, Crosshair, Check, ArrowLeft } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { ROUTES } from '@/lib/routes';
	import * as Popover from '@/components/ui/popover';
	import { loadDefaultLocation } from '@/lib/default-location';
	import { RECENT_LOCATIONS, HOT_LOCATIONS, SEARCH_DB, type PinType, type Pin, type LocationResult } from '../shared/location-data';
	import { SNAP_MIN, SNAP_MID, SNAP_MAX, SNAP_HEIGHTS, PIN_COLORS, pinTone, floatBtnStyle, buildMapData, fetchMapCanvas, type MapHeatBucket, markerUserUrl, markerGroupUrl, markerMomentUrl, type SnapState } from '../shared/location-shared';
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
	const zoom = 0.045;
	let mapCenter = $state<{ lat: number; lng: number } | null>(null);
	let defaultCenter = $state<{ lat: number; lng: number } | null>(null);
	let recentItems = $state(RECENT_LOCATIONS);
	let activeTab = $state<'recent' | 'hot'>('recent');

	// 初始/复位中心来自 ClientConfig default_location（MSW mock），不写死坐标。
	$effect(() => {
		let cancelled = false;
		void loadDefaultLocation().then((loc) => {
			if (cancelled) return;
			const point = loc?.point ? { lat: loc.point.latitude, lng: loc.point.longitude } : null;
			defaultCenter = point;
			if (!mapCenter) mapCenter = point;
		});
		return () => {
			cancelled = true;
		};
	});

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
		`position:absolute;bottom:0;left:0;right:0;height:${sheetH}px;z-index:20;border-radius:22px 22px 0 0;border:1px solid var(--border-default);border-bottom:none;box-shadow:0 -6px 28px var(--elevation-shadow-dark);display:flex;flex-direction:column;overflow:hidden;${dragging ? '' : 'transition:height 0.3s cubic-bezier(0.34,1.56,0.64,1);'}`
	);
	const rightCtrlBottom = $derived(sheetH + 10);
</script>

<div class="bg-surface-primary" style="position:relative;width:100%;height:100%;overflow:hidden;">
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
			<div style="width:18px;height:18px;border-radius:50%;background:{PIN_COLORS.user};border:3px solid var(--surface-primary);box-shadow:0 2px 10px {pinTone('user', 55)};position:relative;">
				<span class="map-pulse-halo absolute rounded-full" style="inset:-7px;background:{pinTone('user', 20)};border:1px solid {pinTone('user', 30)};"></span>
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
			<div class="absolute top-16 left-1/2 -translate-x-1/2 z-20 rounded-full bg-surface-primary px-3 py-1 text-xs text-text-secondary shadow-[0_2px_12px_var(--elevation-shadow-dark)] border border-border-default" >
				加载地图数据…
			</div>
		{/if}

		<!-- Attribution -->
		<div class="text-text-secondary text-[9px]"  style="position:absolute;bottom:4px;left:6px;opacity:0.4;pointer-events:none;z-index:1">
			© OpenStreetMap contributors
		</div>
	</div>

	<!-- ══ TOP BAR (fixed, floating over map — 对齐 Figma Make：无标题，仅返回 + 筛选) ══ -->
	<div style="position:absolute;top:0;left:0;right:0;z-index:50;padding:12px 14px 0;">
		<div style="display:flex;align-items:center;justify-content:space-between;">
			<!-- Back -->
			<button
				type="button"
				class="hover-btn flex items-center justify-center rounded-full outline-none circle-hover"
				style="width:42px;height:42px;{floatBtnStyle}"
				onclick={() => history.back()}
			>
				<ArrowLeft class="size-4.25"  style="color:var(--text-primary)" strokeWidth={2} />
			</button>

			<!-- Filter toggle -->
			<Popover.Root bind:open={showFilter}>
				<div bind:this={filterAnchorEl}>
					<button
						type="button"
						class="hover-btn flex items-center gap-2 rounded-full outline-none {showFilter ? 'bg-accent-default text-text-on-accent' : 'bg-surface-primary text-text-primary'}"
						style="padding:9px 16px;font-size:14px;border:{showFilter ? '1px solid transparent' : '1px solid var(--border-default)'};box-shadow:{showFilter ? '0 4px 16px var(--glow-accent)' : '0 2px 12px var(--elevation-shadow-dark), 0 1px 3px var(--elevation-shadow-dark)'};"
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
						class="w-34 overflow-hidden rounded-control p-0"
					>
						{#each [
							['user', '用户', markerUserUrl, PIN_COLORS.user],
							['group', '群组', markerGroupUrl, PIN_COLORS.group],
							['moment', '瞬间', markerMomentUrl, PIN_COLORS.moment]
						] as [type, label, icon, color], i}
							{@const active = activeFilters.has(type as PinType)}
							{#if i > 0}
								<div class="bg-border-default h-0.25" ></div>
							{/if}
							<button
								type="button"
								class="hover-btn w-full flex items-center gap-2 px-3 py-3 outline-none"
								style="opacity:{active ? 1 : 0.38};"
								onclick={() => toggleFilter(type as PinType)}
							>
								<img class="w-4.5 h-6 shrink-0" src={icon} alt={label}  />
								<span class="text-text-primary text-base"  style="flex:1;text-align:left">{label}</span>
								<div style="width:13px;height:13px;border-radius:50%;shrink:0;background:{active ? color : 'transparent'};border:2px solid {active ? color : 'var(--border-default)'};box-shadow:{active ? `0 0 6px ${color}55` : 'none'};transition:all 0.18s;"></div>
							</button>
						{/each}
					</Popover.Content>
				</Popover.Root>
		</div>
	</div>

	<!-- ══ RIGHT CONTROLS — bottom tracks sheet ══ -->
	<div style="position:absolute;right:12px;bottom:{rightCtrlBottom}px;z-index:35;display:flex;flex-direction:column;gap:8px;align-items:center;transition:bottom 0.3s cubic-bezier(0.34,1.56,0.64,1);">
		<!-- GPS re-center -->
		<button
			type="button"
			class="hover-btn flex items-center justify-center rounded-full outline-none circle-hover"
			style="width:42px;height:42px;{floatBtnStyle}"
			onclick={() => {
				mapCenter = defaultCenter;
				selectedLoc = null;
				searchQuery = '';
			}}
		>
			<Crosshair class="size-4.5"  style="color:var(--accent-default)" strokeWidth={1.75} />
		</button>

		<!-- My location dot（对齐 Figma Make：用户位置圆点 + 白描边 + 主题色 glow） -->
		<button
			type="button"
			aria-label="我的位置"
			class="hover-btn flex items-center justify-center rounded-full outline-none circle-hover"
			style="width:42px;height:42px;{floatBtnStyle}"
			onclick={() => {
				mapCenter = defaultCenter;
				selectedLoc = null;
				searchQuery = '';
			}}
		>
			<div
				style="width:14px;height:14px;border-radius:50%;background:var(--marker-user);border:2.5px solid var(--surface-primary);box-shadow:0 0 8px color-mix(in srgb, var(--marker-user) 65%, transparent);"
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
		<div class="shrink-0"  style="padding:0 14px 8px">
			<div class="flex items-center gap-2.5 px-3.5 rounded-[15px] input-surface h-11" >
				<Search class="size-3.75 shrink-0"  style="color:var(--text-secondary)" strokeWidth={1.5} />
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
					class="flex-1 bg-transparent outline-none text-text-primary text-sm"
				/>
				{#if searchQuery}
					<button
						type="button"
						class="hover-btn outline-none"
						onclick={() => {
							searchQuery = '';
							selectedLoc = null;
						}}
					>
						<X class="size-3.5"  style="color:var(--text-secondary)" strokeWidth={2} />
					</button>
				{:else if selectedLoc}
					<Check class="size-3.5 shrink-0"  style="color:var(--accent-default)" strokeWidth={2.5} />
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
		<div class="shrink-0"  style="padding:8px 14px 18px">
			<button
				type="button"
				class="hover-btn w-full flex items-center justify-center gap-2 rounded-full outline-none bg-accent-default text-text-on-accent"
				style="height:50px;font-size:15px;box-shadow:0 6px 20px var(--glow-accent);"
				onclick={handleConfirm}
			>
				<MapPin class="size-3.75"  strokeWidth={2} />
				{selectedLoc ? `确认：${selectedLoc.displayName.slice(0, 11)}` : '设为浏览锚点'}
			</button>
		</div>
	</div>
</div>
