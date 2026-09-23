<script lang="ts">
	import { Search, MapPin, X, Crosshair, Check, ArrowLeft } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { ROUTES } from '@/lib/routes';
	import { startDefaultLocation } from '@/lib/start-default-location';
	import { createLocateAction } from '@/lib/locate-action.svelte';
	import { autoLocateOnce, clearLocateDeclined, readLastLocation, writeLastLocation } from '@/lib/map-auto-locate';
	import { reportLocation } from '@/lib/location-report';
	import { auth } from '@/stores/auth.svelte';
	import { searchPlaces, type GeoPlace } from '@/lib/geo-service';
	import { createDebouncedSearch } from '@/lib/debounced-search.svelte';
	import { mapSourceStore } from '@/stores/mapSource.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import { type PinType, type Pin, type LocationResult } from '../shared/location-data';
	import { SNAP_MIN, SNAP_MID, SNAP_MAX, SNAP_HEIGHTS, PIN_COLORS, PIN_TONES, floatBtnStyle, fetchMapCanvas, fetchMapCanvasByViewport, type MapBounds, type MapHeatBucket, type SnapState } from '../shared/location-shared';
	import MapCanvas from '../shared/MapCanvas.svelte';
	import MapFilterMenu from '../shared/MapFilterMenu.svelte';
	import MapPinMarker from '../shared/MapPinMarker.svelte';
	import MapHeatDot from '../shared/MapHeatDot.svelte';
	import MapSourceSwitch from '../shared/MapSourceSwitch.svelte';
	import MapZoomControls from '../shared/MapZoomControls.svelte';
	import LocationSheetList from './LocationSheetList.svelte';
	import { smartBack } from '@/lib/smart-back';

	// 移动端地图探索：全屏底图（高德/OSM 可切换）+ 可拖拽吸附底部 Sheet（148/318/540 三档）。
	// motion spring 吸附改 CSS transition。
	// 筛选菜单由 map/shared/MapFilterMenu 统一承载（桌面/移动一致）。

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
	let activeFilters = $state<Set<PinType>>(new Set(['user', 'group', 'moment']));
	let searchQuery = $state('');
	let selectedPin = $state<Pin | null>(null);
	let selectedLoc = $state<LocationResult | null>(null);
	// 瓦片缩放级别；数据请求半径由 fetchMapCanvas 内部约定
	let mapZoom = $state(14);
	let mapBounds = $state<MapBounds>({ minLat: 0, maxLat: 0, minLng: 0, maxLng: 0 });
	// 数据请求用稳定边界（手势/缩放结束才回写）；图钉/热力跟随用每帧 mapBounds
	let mapStableBounds = $state<MapBounds>({ minLat: 0, maxLat: 0, minLng: 0, maxLng: 0 });
	// 首屏初值取上次确认的浏览锚点（localStorage 同步可读），不等 client_config；
	// 配置回来或自动定位成功后再校正。
	let mapCenter = $state<{ lat: number; lng: number } | null>(readLastLocation());
	// 中心来源（本地锚点/服务端默认位置）是否判定完毕——用于结束首屏 loading
	let centerResolved = $state(false);
	// 用户是否已手动选点：自动定位结果不再抢镜头
	let userPicked = $state(false);
	let defaultCenter = $state<{ lat: number; lng: number } | null>(null);
	// 最近使用：搜索选点后记录（会话内状态，原演示播种已按产品决定移除）
	let recentItems = $state<LocationResult[]>([]);

	// 服务端默认位置（ClientConfig default_location）——本地无锚点时的兜底与复位目标。
	// centerReloadKey：让「重试」能重新拉一次配置：loadDefaultLocation 失败不缓存，
	// 重调即重新请求；否则无锚点 + 配置失败时重试按钮是死的。
	let centerReloadKey = $state(0);
	$effect(() => {
		void centerReloadKey;
		return startDefaultLocation((loc) => {
			const point = loc?.point ? { lat: loc.point.latitude, lng: loc.point.longitude } : null;
			defaultCenter = point;
			if (!mapCenter) mapCenter = point;
			centerResolved = true;
		});
	});

	// 进入地图自动定位一次（Geolocation 单次快照，不会自动更新；权限按域名持久记忆，
	// 允许过之后调用不再弹窗）。成功即采纳为最新中心，覆盖首屏锚点快照；用户已手动
	// 选点则让位。拒绝/不支持静默回退默认位置、不 toast 打扰。
	$effect(() => {
		void (async () => {
			const fix = await autoLocateOnce();
			if (!fix) return;
			// 真实位置上报；失败静默，不打断交互。
			void reportLocation({ lat: fix.lat, lng: fix.lng }, auth.isAuthenticated);
			if (userPicked) return;
			mapCenter = { lat: fix.lat, lng: fix.lng };
		})();
	});

	// map/canvas 数据
	let mapPins = $state<Pin[]>([]);
	let heatBuckets = $state<MapHeatBucket[]>([]);
	let mapLoading = $state(true);
	let mapError = $state<string | null>(null);
	let mapReloadKey = $state(0);

	// 重试：既重拉地图数据，也重拉中心配置——中心拿不到时只 bump mapReloadKey
	// 会原地踏步（effect 因 !center 直接返回），重试按钮形同死的。
	function retryLoad() {
		centerReloadKey++;
		mapReloadKey++;
	}

	// 视口中心/边界或筛选变化时重新拉取。依赖 stableBounds（手势/缩放结束才回写），
	// 拖拽中的逐帧 bounds 变化只驱动图钉定位，不触发请求。
	$effect(() => {
		void mapReloadKey;
		const center = mapCenter;
		const types = [...activeFilters];
		const bounds = mapStableBounds;
		if (!center) {
			// 中心未定不请求；判定完毕仍无中心就结束 loading 并给可重试提示，避免永久空转。
			if (centerResolved) {
				mapLoading = false;
				mapError = '暂时无法获取位置信息';
			}
			return;
		}
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

	// 底图视口地理边界由 MapCanvas 墨卡托换算后双向绑定（图钉/热力定位用）
	// ── 真实地理编码搜索（高德/OSM 随地图源，经同源代理）────────────
	// 输入防抖 300ms 后查询；服务不可用/无结果时诚实展示空态，
	// 不回退本地演示数据（原 SEARCH_DB 兜底带虚构打卡人数，已移除）。
	let remoteResults = $state<GeoPlace[] | null>(null);
	// 搜索进行中（OSM 代理偶发数秒慢，需指示）
	let searching = $state(false);
	createDebouncedSearch({
		query: () => searchQuery.trim(),
		deps: () => mapSourceStore.source,
		fetch: (query, signal) => searchPlaces(mapSourceStore.source, query, mapCenter, { signal }),
		apply: (places) => (remoteResults = places),
		clear: () => (remoteResults = null),
		// 请求失败按空结果处理
		onError: () => (remoteResults = null),
		onPending: (p) => (searching = p)
	});
	const searchResults = $derived(searchQuery.trim() ? (remoteResults ?? []) : []);

	// ── 定位到我（浏览器 GPS，失败回退默认位置并提示）──────────────
	// 成功即持久化为浏览锚点：下次进地图先读到本次定位，不会掉回默认位置；
	// 权限既已授予，顺手清除自动定位拒绝标记，下次进入恢复自动定位。
	const locate = createLocateAction({
		onSuccess: (fix) => {
			userPicked = true;
			void reportLocation({ lat: fix.lat, lng: fix.lng }, auth.isAuthenticated);
			mapCenter = { lat: fix.lat, lng: fix.lng };
			writeLastLocation({ lat: fix.lat, lng: fix.lng });
			clearLocateDeclined();
			selectedLoc = null;
			searchQuery = '';
			snapTo('mid');
			showToast(`已定位（精度约 ${Math.round(fix.accuracy)} 米）`, 'success');
		},
		onError: (err) => {
			showToast(err.message, 'danger');
			if (defaultCenter) mapCenter = defaultCenter;
		}
	});

	// 地图原地点击（MapCanvas 内部区分拖拽，拖动不再误触收起）：清选中并收起 sheet
	function handleMapTap() {
		selectedPin = null;
		snapTo('min');
	}

	function handleSelectLoc(loc: LocationResult) {
		userPicked = true;
		selectedLoc = loc;
		searchQuery = loc.displayName;
		mapCenter = { lat: loc.lat, lng: loc.lng };
		selectedPin = null;
		// 记录到「最近使用」：会话内状态，按名称去重、最新在前，最多 8 条
		recentItems = [loc, ...recentItems.filter((r) => r.displayName !== loc.displayName)].slice(0, 8);
		if (snap === 'max') snapTo('mid');
	}

	function handleConfirm() {
		// 只写好浏览锚点、不关页：用户自己决定何时离开（返回按钮仍在顶栏）。
		// 未选地点时以视口中心兜底（与桌面端一致）。
		const point = selectedLoc ?? mapCenter;
		if (!point) {
			showToast('暂时无法获取位置信息', 'danger');
			return;
		}
		// 标记已手动确认：迟到的自动定位不再覆盖刚写好的锚点。
		userPicked = true;
		writeLastLocation({ ...point, name: selectedLoc?.displayName });
		showToast('浏览锚点已设置', 'success');
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

<div class="bg-surface-page" style="position:relative;width:100%;height:100%;overflow:hidden;">
	<h1 class="sr-only">地图</h1>
	<!-- ══ MAP (full screen, tappable) ══ -->
	<!-- 原地点击空白由 MapCanvas onMapTap 承担（拖拽不触发）；键盘 Escape 等价收起见 svelte:window -->
	<div style="position:absolute;inset:0;z-index:0;">
		{#if mapCenter}
		<MapCanvas bind:center={mapCenter} bind:zoom={mapZoom} bind:bounds={mapBounds} bind:stableBounds={mapStableBounds} onMapTap={handleMapTap} />

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
					userPicked = true;
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
		{:else if mapError}
			<!-- 加载失败：给出原因与重试，避免空白地图被误认为「附近什么都没有」 -->
			<div class="absolute top-16 left-1/2 -translate-x-1/2 z-20 flex items-center gap-2 rounded-full bg-surface-primary px-3 py-1.5 text-xs text-status-danger shadow-raised-sm border border-border-default" >
				<span>{mapError}</span>
				<button type="button" class="link-btn focus-ring rounded-sm text-accent-default" onclick={retryLoad}>重试</button>
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
					aria-label="返回"
					onclick={() => smartBack()}
				>
					<ArrowLeft class="size-4.5 text-text-primary" strokeWidth={2} />
				</button>

				<!-- Filter menu（桌面/移动共用组件：徽章/重置/至少一项/外点关闭统一） -->
				<MapFilterMenu bind:activeFilters align="end" />
			</div>
		</div>

	<!-- ══ RIGHT CONTROLS — bottom tracks sheet ══ -->
	<div style="position:absolute;right:12px;bottom:{rightCtrlBottom}px;z-index:35;display:flex;flex-direction:column;gap:8px;align-items:center;transition:bottom var(--cst-motion-spring) var(--cst-ease-bounce);">
		<!-- Zoom in/out（画布拖拽/双指外滚轮亦可缩放），分段控件视觉契约见 MapZoomControls -->
		<MapZoomControls
			bind:zoom={mapZoom}
			style="border:var(--cst-border-width-hairline) solid var(--cst-border-default);{floatBtnStyle}"
		/>

		<!-- 回到我的位置（浏览器 GPS，失败回退默认位置）+ 地图源切换 -->
		<button
			type="button"
			aria-label={locate.locating ? '正在定位…' : '回到我的位置'}
			title="回到我的位置"
			class="hover-btn flex items-center justify-center rounded-full focus-ring circle-hover size-10.5 disabled:pointer-events-none disabled:opacity-faint"
			style="{floatBtnStyle}"
			disabled={locate.locating}
			onclick={() => void locate.locate()}
		>
			{#if locate.locating}
				<span class="spinner size-4.5" role="status" aria-hidden="true"></span>
			{:else}
				<Crosshair class="size-4.5 text-accent-default" strokeWidth={1.75} />
			{/if}
		</button>
		<!-- 右侧控制列贴着底部 Sheet：菜单向上展开，否则会压住 Sheet 内容 -->
		<MapSourceSwitch align="end" side="top" />
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
			style="flex-shrink:0;display:flex;flex-direction:column;align-items:center;padding-top:10px;padding-bottom:6px;cursor:grab;touch-action:none;user-select:none;"
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

		<!-- Scrollable content — 收起(min)时隐藏而非卸载（保留滚动位置/输入状态） -->
		<div
			style="flex:1;min-height:0;overflow-y:auto;{snap === 'min' ? 'display:none;' : ''}"
			class="no-scrollbar pb-3"
			aria-hidden={snap === 'min'}
		>
			<LocationSheetList
				bind:searchQuery
				{searchResults}
				{searching}
				bind:selectedPin
				bind:selectedLoc
				bind:recentItems
				onSelectLoc={handleSelectLoc}
				onOpenPinDetail={openPinDetail}
				onClearPin={() => (selectedPin = null)}
			/>
		</div>

		<!-- Confirm button — always at bottom of sheet -->
		<div class="shrink-0 px-4 pt-2 pb-4.5">
			<button
				type="button"
				class="hover-btn w-full flex items-center justify-center gap-2 rounded-full focus-ring bg-accent-default text-text-on-accent text-sm h-12.5 px-4 shadow-raised-md hover:shadow-raised-lg"
				onclick={handleConfirm}
			>
				<MapPin class="size-4 shrink-0" strokeWidth={2} />
				<span class="min-w-0 truncate">{selectedLoc ? `确认：${selectedLoc.displayName}` : '设为浏览锚点'}</span>
			</button>
		</div>
	</div>
</div>

<svelte:window onkeydown={(e) => { if (e.key === 'Escape') handleMapTap(); }} />
