<script lang="ts">
	import { X, ChevronRight, Heart, Crosshair } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { ROUTES } from '@/lib/routes';
	import { startDefaultLocation } from '@/lib/start-default-location';
	import { createLocateAction } from '@/lib/locate-action.svelte';
	import { autoLocateOnce, clearLocateDeclined, readLastLocation, writeLastLocation } from '@/lib/map-auto-locate';
	import { reportLocation } from '@/lib/location-report';
	import { auth } from '@/stores/auth.svelte';
	import { formatReverseAddress, reverseGeocode, searchPlaces, type GeoPlace } from '@/lib/geo-service';
	import { createDebouncedSearch } from '@/lib/debounced-search.svelte';
	import { mapSourceStore } from '@/stores/mapSource.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import { type PinType, type Pin, type LocationResult } from '../shared/location-data';
	import {
		PIN_COLORS, PIN_LABELS, PIN_TONES, fetchMapCanvas, fetchMapCanvasByViewport, canvasToListPins,
		type MapBounds, type MapHeatBucket
	} from '../shared/location-shared';
	import { MAP_MAX_ZOOM } from '../shared/map-projection';
	import MapCanvas from '../shared/MapCanvas.svelte';
	import MapFilterMenu from '../shared/MapFilterMenu.svelte';
	import MapPinMarker from '../shared/MapPinMarker.svelte';
	import MapHeatDot from '../shared/MapHeatDot.svelte';
	import MapSourceSwitch from '../shared/MapSourceSwitch.svelte';
	import MapZoomControls from '../shared/MapZoomControls.svelte';
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
	// 瓦片缩放级别（13≈全市 / 16≈街区）；数据请求半径仍由 fetchMapCanvas 内部约定
	let mapZoom = $state(14);
	// 首屏初值取上次确认的浏览锚点（localStorage 同步可读），不再干等 client_config；
	// 配置回来或自动定位成功后再校正（见下方 $effect）。
	let mapCenter = $state<{ lat: number; lng: number } | null>(readLastLocation());
	// 中心来源（本地锚点/服务端默认位置）是否已判定完毕——用于结束首屏 loading
	let centerResolved = $state(false);
	// 用户是否已手动选点（选地点/点图钉/点手动定位）：自动定位不再抢镜头
	let userPicked = $state(false);
	let mapBounds = $state<MapBounds>({ minLat: 0, maxLat: 0, minLng: 0, maxLng: 0 });
	// 数据请求用稳定边界（手势/缩放结束才回写）；图钉/热力跟随用每帧 mapBounds
	let mapStableBounds = $state<MapBounds>({ minLat: 0, maxLat: 0, minLng: 0, maxLng: 0 });
	let defaultCenter = $state<{ lat: number; lng: number } | null>(null);
	// 服务端默认位置名（ClientConfig default_location.location_name）：逆地理失败时的兜底。
	let defaultName = $state('');
	// 上次确认的浏览锚点（含名）：中心与其同格且逆地理缺席时，用锚点名兜底。
	let anchorPoint = $state<{ lat: number; lng: number; name?: string } | null>(readLastLocation());
	// 逆地理结果（key=四位小数网格；失败记 null 免重复请求，随 centerNamenull 兜底链消化）。
	let reverseEntry = $state<{ key: string; name: string | null } | null>(null);
	const reverseCache = new Map<string, string | null>();

	// 服务端默认位置（ClientConfig default_location）——本地无锚点时的兜底与复位目标。
	// centerResolved 回调落地后置真：无锚点且配置也拿不到时结束首屏 loading，不空转。
	// centerReloadKey：让「重试」能重新拉一次配置：loadDefaultLocation 失败不缓存，
	// 重调即重新请求；否则无锚点 + 配置失败时重试按钮是死的。
	let centerReloadKey = $state(0);
	$effect(() => {
		void centerReloadKey;
		return startDefaultLocation((loc) => {
			const point = loc?.point ? { lat: loc.point.latitude, lng: loc.point.longitude } : null;
			defaultCenter = point;
			defaultName = loc?.location_name ?? '';
			if (!mapCenter) mapCenter = point;
			centerResolved = true;
		});
	});

	// 进入地图自动定位一次（Geolocation 单次快照：调一次给一次，不会自动更新；
	// 权限按域名持久记忆，用户允许过之后调用不再弹窗）。成功即采纳为最新中心——
	// 首屏本地锚点只是快照，会被新鲜 GPS 覆盖；用户已手动选点则让位。
	// 拒绝/不支持静默回退默认位置、不 toast 打扰（手动定位按钮不受影响）。
	$effect(() => {
		void (async () => {
			const fix = await autoLocateOnce();
			if (!fix) return;
			// 真实位置上报（users/update_location）：与是否采纳为中心无关，
			// 上报的是「用户在哪」；失败静默，不打断交互。
			void reportLocation({ lat: fix.lat, lng: fix.lng }, auth.isAuthenticated);
			if (userPicked) return;
			mapCenter = { lat: fix.lat, lng: fix.lng };
		})();
	});

	// 视口中心地名：选了地点/图钉直接用其名；否则逆地理编码实际中心。
	// 防抖 500ms + 网格缓存：Nominatim 经自家 /osm-service 代理免费调用（高德源优先高德 key，
	// 未配 key 自动降级 OSM），高频拖拽不刷服务。失败/未决时由下面的 locationName 兜底链消化。
	$effect(() => {
		const center = mapCenter;
		const source = mapSourceStore.source;
		if (selectedLoc || selectedPin || !center) return;
		const key = `${center.lat.toFixed(4)},${center.lng.toFixed(4)}`;
		const hit = reverseCache.get(key);
		if (hit !== undefined) {
			reverseEntry = { key, name: hit };
			return;
		}
		let cancelled = false;
		const timer = setTimeout(() => {
			void reverseGeocode(source, center.lat, center.lng)
				.then((raw) => {
					if (cancelled) return;
					const name = formatReverseAddress(raw);
					reverseCache.set(key, name);
					reverseEntry = { key, name };
				})
				.catch(() => {
					if (cancelled) return;
					reverseCache.set(key, null);
					if (reverseEntry?.key === key) reverseEntry = { key, name: null };
				});
		}, 500);
		return () => {
			cancelled = true;
			clearTimeout(timer);
		};
	});

	// 左侧栏显示的地名（顶栏副标题 +「当前位置」卡片）：永远描述实际视口中心，
	// 而非服务端静态默认名。优先级：选中地点 > 选中图钉 > 同格逆地理 > 同格锚点名 > 服务端默认名。
	const locationName = $derived(
		selectedLoc?.displayName ??
			(selectedPin?.name || selectedPin?.title || undefined) ??
			(mapCenter && reverseEntry && reverseEntry.key === `${mapCenter.lat.toFixed(4)},${mapCenter.lng.toFixed(4)}`
				? reverseEntry.name
				: undefined) ??
			(mapCenter &&
			anchorPoint?.name &&
			anchorPoint.lat.toFixed(4) === mapCenter.lat.toFixed(4) &&
			anchorPoint.lng.toFixed(4) === mapCenter.lng.toFixed(4)
				? anchorPoint.name
				: undefined) ??
			defaultName
	);

	// map/canvas 数据
	let mapPins = $state<Pin[]>([]);
	let heatBuckets = $state<MapHeatBucket[]>([]);
	let mapCounts = $state<Record<PinType, number>>({ user: 0, group: 0, moment: 0 });
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
			// 中心未定不请求；判定完毕仍无中心（本地无锚点且配置拿不到）就结束 loading 并
			// 给可重试提示，否则「加载地图数据…」永久空转、用户只见空白。
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

	// 从聚合桶"放大看明细"时的目标级别：街区级，稳定低于服务端 50km 热力阈值
	// （桌面视口 zoom14 跨度约 11.6km，留足余量）。
	const AGGREGATE_DETAIL_ZOOM = 14;

	// 图钉详情跳转：user/group/moment 分别路由到对应详情页。
	// 聚合桶不是真实实体（id 形如 "heat:<cell_token>"，无对应记录），
	// 跳详情会得到 404；改为把地图中心移到该格，让用户放大后看真实明细。
	function openPinDetail(pin: Pin) {
		if (pin.isAggregate) {
			// 聚合桶是 S2 level-10 网格（格边长约 25km），"+3 级"不足以保证视口跨度
			// 降到 50km 阈值以下（实测桌面视口 zoom14 才约 11.6km）。这里直接对齐到
			// 能稳定拿到明细的街区级，跨过阈值后服务端切回 hotspots，列表随之变为实体。
			mapCenter = { lat: pin.lat, lng: pin.lng };
			mapZoom = Math.max(mapZoom, AGGREGATE_DETAIL_ZOOM);
			selectedPin = null;
			return;
		}
		if (pin.type === 'user') void goto(ROUTES.user(pin.id));
		else if (pin.type === 'group') void goto(ROUTES.group(pin.id));
		else void goto(ROUTES.moment(pin.id));
	}

	// ── 真实地理编码搜索（高德/OSM 随地图源，经同源代理）────────────
	// 输入防抖 300ms 后查询；服务不可用/无结果时诚实展示空态，
	// 不回退本地演示数据（原 SEARCH_DB 兜底带虚构打卡人数，已移除）。
	let remoteResults = $state<GeoPlace[] | null>(null);
	// 搜索进行中：上游（OSM 代理）偶发数秒级慢响应，无指示会让用户以为搜索失效
	let searching = $state(false);
	const searchResults = $derived(searchQuery.trim() ? (remoteResults ?? []) : []);
	// 「附近」列表数据源：hotspots 用实体条目（count>1 的是地图聚类视觉，不重复出现在列表），
	// heatmap 视口下 pins 为空，改用聚合桶投影——否则缩放到城市级时列表永远显示"附近暂无内容"，
	// 与地图上明明有热力点自相矛盾（用户坐标并未变化，只是服务端切换了返回模式）。
	const nearbyPins = $derived(
		mapPins.length > 0
			? mapPins.filter((p) => (p.count ?? 1) === 1)
			: canvasToListPins({ mode: 'heatmap', pins: [], heatBuckets, counts: mapCounts })
	);

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

	// ── 定位到我（浏览器 GPS，失败回退默认位置并提示）──────────────
	// 成功即持久化为浏览锚点：下次进地图先读到本次定位，不会掉回默认位置；
	// 权限既已授予，顺手清除自动定位拒绝标记，下次进入恢复自动定位。
	const locate = createLocateAction({
		onSuccess: (fix) => {
			userPicked = true;
			void reportLocation({ lat: fix.lat, lng: fix.lng }, auth.isAuthenticated);
			mapCenter = { lat: fix.lat, lng: fix.lng };
			writeLastLocation({ lat: fix.lat, lng: fix.lng });
			anchorPoint = { lat: fix.lat, lng: fix.lng };
			clearLocateDeclined();
			selectedLoc = null;
			searchQuery = '';
			showToast(`已定位（精度约 ${Math.round(fix.accuracy)} 米）`, 'success');
		},
		onError: (err) => {
			showToast(err.message, 'danger');
			if (defaultCenter) mapCenter = defaultCenter;
		}
	});

	// 底图视口地理边界由 MapCanvas 墨卡托换算后双向绑定（图钉/热力定位用）

	function handleSelectLoc(loc: LocationResult) {
		userPicked = true;
		selectedLoc = loc;
		searchQuery = loc.displayName;
		mapCenter = { lat: loc.lat, lng: loc.lng };
		selectedPin = null;
	}

	function handleSelectPin(pin: Pin) {
		userPicked = true;
		selectedPin = pin;
		mapCenter = { lat: pin.lat, lng: pin.lng };
	}

	function handleConfirm() {
		// 只写好浏览锚点、不关页：用户自己决定何时离开（返回在左侧面板顶栏）。
		// 锚点语义 = 「以当前视口中心为准」：未选地点时用 mapCenter 兜底。
		const point = selectedLoc ?? mapCenter;
		if (!point) {
			showToast('暂时无法获取位置信息', 'danger');
			return;
		}
		// 标记已手动确认：迟到的自动定位不再覆盖刚写好的锚点。
		userPicked = true;
		// 选中地点时连带名称一起持久化：附近页/侧栏显示的位置名须与锚点一致。
		writeLastLocation({ ...point, name: selectedLoc?.displayName });
		anchorPoint = { lat: point.lat, lng: point.lng, name: selectedLoc?.displayName ?? undefined };
		showToast('浏览锚点已设置', 'success');
	}
</script>

<div class="flex h-full bg-surface-page">
	<!-- ══ LEFT SIDEBAR ══ -->
	<div class="flex flex-col shrink-0 w-85 border-r border-border-default"  >
		<LocationDesktopPanel
			bind:searchQuery
			{searchResults}
			{searching}
			{nearbyPins}
			bind:selectedPin
			bind:selectedLoc
			{mapCounts}
			locationName={locationName}
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
					userPicked = true;
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
				<button type="button" class="link-btn focus-ring rounded-sm text-accent-default" onclick={retryLoad}>重试</button>
			</div>
		{/if}

		<!-- 缩放控制（右下角；画布拖拽/滚轮亦可缩放），分段控件视觉契约见 MapZoomControls -->
		<MapZoomControls
			bind:zoom={mapZoom}
			class="absolute z-30 border border-border-default bg-surface-primary"
			style="right:14px;bottom:20px;box-shadow:var(--cst-elevation-fab);"
		/>

		<!-- 左上筛选菜单（桌面/移动共用组件；含计数徽章、至少一项约束与全部显示重置） -->
		<div class="absolute z-20" style="top:14px;left:14px;">
			<MapFilterMenu bind:activeFilters counts={mapCounts} showLegend align="start" />
		</div>

		<!-- 右上角控制组：定位到当前位置（浏览器 GPS，失败回退默认位置）+ 地图源切换。
		     地图源菜单 align="end"：按钮贴右边缘，菜单向左展开防视口裁剪
		     （此前默认 align="start" 向右展开，200px 菜单被右边缘挡住一部分）。 -->
		<div class="absolute z-20 flex items-center gap-2" style="top:14px;right:14px;">
			<MapSourceSwitch align="end" />
			<button
				type="button"
				class="hover-btn scale-btn flex items-center justify-center rounded-full focus-ring size-10 disabled:pointer-events-none disabled:opacity-faint"
				style="background:var(--cst-surface-primary);color:var(--cst-text-primary);border:var(--cst-border-width-hairline) solid var(--cst-border-default);box-shadow:var(--cst-elevation-fab);"
				aria-label={locate.locating ? '正在定位…' : '定位到当前位置'}
				title="定位到当前位置"
				disabled={locate.locating}
				onclick={(e) => {
					e.stopPropagation();
					void locate.locate();
				}}
			>
				{#if locate.locating}
					<span class="spinner size-4" role="status" aria-hidden="true"></span>
				{:else}
					<Crosshair class="size-4 text-accent-default" strokeWidth={1.75} />
				{/if}
			</button>
		</div>


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
							<!-- 聚合桶无详情页：不能给"查看详情"（点了没反应/404），改为放大看明细 -->
							{selectedPin?.isAggregate ? '放大地图查看' : '查看详情'}
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
