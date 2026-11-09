<script lang="ts">
	import { onDestroy } from 'svelte';
	import { Crosshair, Check, ArrowLeft } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { ROUTES } from '@/lib/routes';
	import { startDefaultLocation } from '@/lib/start-default-location';
	import { createLocateAction } from '@/lib/locate-action.svelte';
	import {
		autoLocateOnce,
		clearLastLocation,
		clearLocateDeclined,
		LOCATION_ANCHOR_UPDATED_EVENT,
		readLastLocation,
		writeLastLocation
	} from '@/lib/map-auto-locate';
	import { reportLocation } from '@/lib/location-report';
	import { auth } from '@/stores/auth.svelte';
	import { formatReverseAddress, reverseGeocode, searchPlaces, type GeoPlace } from '@/lib/geo-service';
	import { createDebouncedSearch } from '@/lib/debounced-search.svelte';
	import { mapSourceStore } from '@/stores/mapSource.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import { type PinType, type Pin, type LocationResult } from '../shared/location-data';
	import { SNAP_MIN, SNAP_MID, SNAP_MAX, SNAP_HEIGHTS, floatBtnStyle, aggregateDrillZoom, type MapBounds, type MapHeatBucket, type SnapState } from '../shared/location-shared';
	import { createMapCanvasState } from '../shared/create-map-canvas-state.svelte';
	import { locationGridKey, resolveLocationName } from '../shared/location-name';
	import MapCanvas from '@/components/map/MapCanvas.svelte';
	import MapCenterCrosshair from '../shared/MapCenterCrosshair.svelte';
	import BrowseAnchorCard from '../shared/BrowseAnchorCard.svelte';
	import AnchorConfirmButton from '../shared/AnchorConfirmButton.svelte';
	import { createAnchorFeedback } from '../shared/anchor-feedback.svelte';
	import { anchorActionLabel, anchorPurposeHint, resolveAnchorName } from '../shared/browse-anchor';
	import MapFilterMenu from '../shared/MapFilterMenu.svelte';
	import MapOverlays from '../shared/MapOverlays.svelte';
	import MapSourceSwitch from '../shared/MapSourceSwitch.svelte';
	import MapZoomControls from '../shared/MapZoomControls.svelte';
	import LocationSheetList from './LocationSheetList.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import { smartBack } from '@/lib/smart-back';
	import { platform } from '@/stores/platform.svelte';

	// 移动端地图探索：全屏底图（高德/OSM 可切换）+ 可拖拽吸附底部 Sheet（148/318/540 三档）。
	// motion spring 吸附改 CSS transition。
	// 筛选菜单由 map/shared/MapFilterMenu 统一承载（桌面/移动一致）。

	/* ── Sheet drag ── */
	let sheetH = $state(SNAP_MID);
	let dragging = $state(false);
	let snap = $state<SnapState>('mid');
	let dragState = $state({ active: false, startY: 0, startH: SNAP_MID, velY: 0, prevY: 0, prevT: 0 });
	// 列表底部净空用的吸附高度快照 & 贴底确认 dock 的实测高度：
	// 只在吸附/实测变化时更新，拖拽中不逐帧改 padding（避免布局逐帧动画）。
	let padSheetH = $state(SNAP_MID);
	let dockH = $state(0);

	function snapTo(target: SnapState) {
		snap = target;
		sheetH = SNAP_HEIGHTS[target];
		padSheetH = SNAP_HEIGHTS[target];
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
	// 捏合手势中的连续缩放变换（MapCanvas 回写），施加于 marker 覆盖层保持与瓦片同步
	let mapZoom = $state(14);
	let mapBounds = $state<MapBounds>({ minLat: 0, maxLat: 0, minLng: 0, maxLng: 0 });
	// 数据请求用稳定边界（手势/缩放结束才回写）；图钉/热力跟随用每帧 mapBounds
	let mapStableBounds = $state<MapBounds>({ minLat: 0, maxLat: 0, minLng: 0, maxLng: 0 });
	// 首屏初值取上次确认的浏览锚点（localStorage 同步可读），不等 client_config；
	// 配置回来或自动定位成功后再校正。
	let mapCenter = $state<{ lat: number; lng: number } | null>(readLastLocation());
	// 上次确认的浏览锚点（含名）：与地图中心分开维护——中心可自由拖动/选点，
	// 锚点只在底部 CTA 显式确认时改写（旧逻辑「定位到我」会静默改写锚点，语义错误）。
	let anchorPoint = $state<{ lat: number; lng: number; name?: string } | null>(readLastLocation());
	// 确认按钮的反馈状态机（与桌面端同一套）：pending → success → 恢复主按钮。
	const anchorFeedback = createAnchorFeedback();
	onDestroy(() => anchorFeedback.destroy());
	// 我的位置（浏览器 GPS fix，WGS-84）：蓝点按它做地理投影，不随地图中心走
	// （语义红线：蓝点=我的位置，不是地图中心；见 MapMyLocationDot）。
	let myLocation = $state<{ lat: number; lng: number } | null>(null);
	// 中心来源（本地锚点/服务端默认位置）是否判定完毕——用于结束首屏 loading
	let centerResolved = $state(false);
	// 用户是否已手动选点：自动定位结果不再抢镜头
	let userPicked = $state(false);
	let defaultCenter = $state<{ lat: number; lng: number } | null>(null);
	// 服务端默认位置名（ClientConfig default_location.location_name）：逆地理失败时的兜底。
	let defaultName = $state('');
	// 逆地理结果（key=四位小数网格；失败记 null 免重复请求，随兜底链消化）。
	let reverseEntry = $state<{ key: string; name: string | null } | null>(null);
	const reverseCache = new Map<string, string | null>();
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
			defaultName = loc?.location_name ?? '';
			if (!mapCenter) mapCenter = point;
			centerResolved = true;
		});
	});

	// 锚点外部变更同步（附近页「改用默认位置」等会清 localStorage 并派发事件）：
	// 地图页常驻时不重挂载，必须监听事件重读，否则锚点卡片/地图标记停留在挂载时旧值。
	$effect(() => {
		const syncAnchor = () => {
			anchorPoint = readLastLocation();
		};
		window.addEventListener(LOCATION_ANCHOR_UPDATED_EVENT, syncAnchor);
		return () => window.removeEventListener(LOCATION_ANCHOR_UPDATED_EVENT, syncAnchor);
	});

	// 进入地图自动定位一次（Geolocation 单次快照，不会自动更新；权限按域名持久记忆，
	// 允许过之后调用不再弹窗）。有浏览锚点时以锚点为首屏视野（只补蓝点）；
	// 无锚点才采纳 GPS 为中心；用户已手动选点则让位。拒绝/不支持静默回退默认位置、不 toast 打扰。
	$effect(() => {
		void (async () => {
			const fix = await autoLocateOnce();
			if (!fix) return;
			// 真实位置上报；失败静默，不打断交互。
			void reportLocation({ lat: fix.lat, lng: fix.lng }, auth.isAuthenticated);
			myLocation = { lat: fix.lat, lng: fix.lng };
			if (userPicked || anchorPoint) return;
			mapCenter = { lat: fix.lat, lng: fix.lng };
		})();
	});

	// 视口中心地名：选了地点/图钉直接用其名；否则逆地理编码实际中心（防抖 500ms + 网格缓存）。
	// 与桌面端同一兜底链，移动端锚点卡片/锚点命名才有可读名称。
	$effect(() => {
		const center = mapCenter;
		const source = mapSourceStore.source;
		if (selectedLoc || selectedPin || !center) return;
		const key = locationGridKey(center);
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

	// 中心地名兜底链与桌面端一致：选中地点 > 图钉 > 同格逆地理 > 锚点名 > 我的位置 > 默认位置 > 未命名区域。
	const locationName = $derived(
		resolveLocationName({
			center: mapCenter,
			selectedName: selectedLoc?.displayName,
			pinName: selectedPin?.name || selectedPin?.title,
			reverseName: mapCenter && reverseEntry?.key === locationGridKey(mapCenter) ? reverseEntry.name : null,
			anchor: anchorPoint,
			myLocation,
			defaultLocation: defaultCenter ? { ...defaultCenter, name: defaultName } : null
		})
	);

	// map/canvas 数据（桌面/移动共用工厂）：图钉/热力/计数 + 加载/错误/重试
	const mapData = createMapCanvasState({
		getCenter: () => mapCenter,
		getFilters: () => activeFilters,
		getStableBounds: () => mapStableBounds,
		isCenterResolved: () => centerResolved,
		onRetry: () => (centerReloadKey++)
	});

		let mapGestureTransform = $state('');
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
			myLocation = { lat: fix.lat, lng: fix.lng };
			// 定位只移动视野/蓝点，不静默改写浏览锚点：锚点必须由用户显式点底部 CTA 确认。
			clearLocateDeclined();
			selectedLoc = null;
			searchQuery = '';
			snapTo('mid');
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

	// 底部 CTA：始终是主按钮（不再出现灰色 disabled）；中心≈锚点时点击 = 重复确认。
	const ctaLabel = $derived(anchorActionLabel(selectedLoc?.displayName ?? undefined));

	function handleConfirm() {
		// 反馈收进按钮本体（pending → success → 恢复主按钮），不再弹成功 toast。
		anchorFeedback.run(() => {
			// 只写好浏览锚点、不关页：用户自己决定何时离开（返回按钮仍在顶栏）。
			// 落点语义：选中地点优先，否则以地图中心（准星）为准（与桌面端一致）。
			const point = selectedLoc ?? mapCenter;
			if (!point) return false;
			// 标记已手动确认：迟到的自动定位不再覆盖刚写好的锚点。
			userPicked = true;
			// 名称一并持久化：附近页/侧栏的位置名须与锚点一致；未命名区域/我的位置不落库。
			const name = resolveAnchorName(selectedLoc?.displayName, locationName);
			writeLastLocation({ ...point, name });
			anchorPoint = { lat: point.lat, lng: point.lng, name };
			return true;
		});
	}

	/** 恢复默认浏览位置：清除本机锚点，地图回到服务端默认位置。 */
	function handleResetAnchor() {
		anchorFeedback.reset();
		clearLastLocation();
		anchorPoint = null;
		selectedLoc = null;
		searchQuery = '';
		userPicked = false;
		if (defaultCenter) mapCenter = defaultCenter;
		// 恢复默认的反馈由卡片/地图标记承担，不再弹 toast（与「设置锚点」同一原则）。
	}

	/** 点击地图上的锚点标记：视野回到锚点。 */
	function handleSelectAnchor() {
		if (!anchorPoint) return;
		userPicked = true;
		mapCenter = { lat: anchorPoint.lat, lng: anchorPoint.lng };
		selectedLoc = null;
		selectedPin = null;
		snapTo('mid');
	}

	// 图钉详情跳转：user/group/moment 分别路由到对应详情页
	function openPinDetail(pin: Pin) {
		if (pin.type === 'user') void goto(ROUTES.user(pin.id));
		else if (pin.type === 'group') void goto(ROUTES.group(pin.id));
		else void goto(ROUTES.moment(pin.id));
	}

	// 聚合桶下钻：中心移到该格并放大到「整格入画」的级别——跨过 50km 阈值后
	// 服务端切回 hotspots，地图随之变为真实实体明细。移动端 sheet 没有聚合列表，
	// 点热力徽标是热力桶唯一的下钻入口。
	function handleHeatSelect(bucket: MapHeatBucket) {
		userPicked = true;
		mapCenter = { lat: bucket.location.latitude, lng: bucket.location.longitude };
		mapZoom = aggregateDrillZoom(mapZoom, mapBounds);
		selectedPin = null;
	}

	const isLandscape = $derived(platform.orientation === 'landscape');

	// Sheet 外壳：无描边、只靠 sheet/modal 投影与地图分层（控件边界允许，但不叠描边+投影）。
	// 横屏是悬浮侧卡（投影 modal）；竖屏是底部抽屉（投影 sheet），动效改为合成器位移：
	// - 面板高度固定为最大档 SNAP_MAX（只在挂载时确定一次），展开/收起只改
	//   transform: translateY —— 不再逐帧改 height 触发重排/重绘（P0-3）；
	// - 面板相对 root 顶部对齐，可见部分 = 顶部 sheetH，超出视口的下半段由贴底 dock 覆盖；
	// - 确认 CTA 由独立 dock 常驻视口底部（面板位移时 dock 不动），语义与观感不变。
	// 拖拽/手势仍直接写 sheetH（transform 每帧插值走合成器），松手回弹到吸附档。
	const sheetRootStyle = $derived.by(() => {
		if (isLandscape) return 'position:absolute;inset:0;z-index:20;pointer-events:none;';
		return `position:absolute;bottom:0;left:0;right:0;max-width:640px;margin:0 auto;height:${SNAP_MAX}px;z-index:20;pointer-events:none;`;
	});

	const sheetStyle = $derived.by(() => {
		if (isLandscape) {
			return 'position:absolute;top:60px;bottom:14px;left:14px;width:340px;max-width:calc(50vw - 28px);z-index:20;border-radius:var(--radius-sheet);box-shadow:var(--cst-elevation-modal);display:flex;flex-direction:column;overflow:hidden;pointer-events:auto;';
		}
		const shift = SNAP_MAX - sheetH;
		return `position:absolute;inset:0;border-radius:var(--radius-sheet) var(--radius-sheet) 0 0;box-shadow:var(--cst-elevation-sheet);display:flex;flex-direction:column;overflow:hidden;pointer-events:auto;transform:translateY(${shift}px);${dragging ? '' : 'transition:transform var(--cst-motion-spring) var(--cst-ease-bounce);'}`;
	});

	// 列表滚动容器：底部留出「面板下移量 + dock 高度」净空，滚到底时最后一项正好停在
	// dock 上沿（dock 覆盖区域不承载可读内容）。下移量取吸附快照，拖拽中不逐帧改 padding。
	const scrollStyle = $derived.by(() => {
		if (isLandscape) return 'flex:1;min-height:0;overflow-y:auto;padding-bottom:12px;';
		const pad = Math.max(0, SNAP_MAX - padSheetH) + dockH + 12;
		return `flex:1;min-height:0;overflow-y:auto;padding-bottom:${pad}px;${snap === 'min' ? 'display:none;' : ''}`;
	});

	// 右列控件：竖屏用与面板同一条 transform 位移（贴着抽屉上沿），不再动画 bottom。
	const rightCtrlStyle = $derived.by(() => {
		if (isLandscape) {
			return 'position:absolute;right:12px;bottom:20px;z-index:35;display:flex;flex-direction:column;gap:8px;align-items:center;';
		}
		const shift = SNAP_MAX - sheetH;
		return `position:absolute;right:12px;bottom:${SNAP_MAX + 10}px;z-index:35;display:flex;flex-direction:column;gap:8px;align-items:center;transform:translateY(${shift}px);${dragging ? '' : 'transition:transform var(--cst-motion-spring) var(--cst-ease-bounce);'}`;
	});
</script>

<!-- 底部确认条（横屏留在面板内 / 竖屏由贴底 dock 承载，两处同一份标记） -->
{#snippet confirmBar()}
	<AnchorConfirmButton state={anchorFeedback.state} label={ctaLabel} onclick={handleConfirm} />
	{#if !anchorPoint}
		<!-- 收起(min)时隐藏说明：dock 高度直接决定面板可见高度，min 档留给把手+搜索栏 -->
		<p
			class="mt-1.5 text-center text-xxs text-text-secondary {!isLandscape && snap === 'min'
				? 'hidden'
				: ''}"
		>
			{anchorPurposeHint(false)}
		</p>
	{/if}
{/snippet}

<div class="bg-surface-page" style="position:relative;width:100%;height:100%;overflow:hidden;">
	<h1 class="sr-only">地图</h1>
	<!-- ══ MAP (full screen, tappable) ══ -->
	<!-- 原地点击空白由 MapCanvas onMapTap 承担（拖拽不触发）；键盘 Escape 等价收起见 svelte:window -->
	<div style="position:absolute;inset:0;z-index:0;">
		{#if mapCenter}
			<MapCanvas bind:center={mapCenter} bind:zoom={mapZoom} bind:bounds={mapBounds} bind:stableBounds={mapStableBounds} bind:gestureTransform={mapGestureTransform} onMapTap={handleMapTap} />
		{/if}

		<!-- 地图中心准星：拖地图改变的是它；设为浏览锚点时以它为落点 -->
		<MapCenterCrosshair />

		<!-- 数据覆盖层（当前位置点/热力桶/图钉/加载错误条）：桌面/移动共用组件 -->
		<!-- 捏合手势中与瓦片层做同款连续缩放变换，保证图钉/热力跟手不脱节 -->
		<!-- pointer-events-none:同桌面端——覆盖层放行真实指针事件给底层画布 -->
<div class="pointer-events-none absolute inset-0" style={mapGestureTransform}>
		<MapOverlays
			pins={mapData.pins}
			bounds={mapBounds}
			heatBuckets={mapData.heatBuckets}
			selectedPinId={selectedPin?.id ?? null}
			loading={mapData.loading}
			error={mapData.error}
			myLocation={myLocation}
			dotPulse={true}
			chipTopClass="top-16"
			onSelectPin={(pin, e) => {
				e.stopPropagation();
				userPicked = true;
				selectedPin = selectedPin?.id === pin.id ? null : pin;
				if (snap === 'min') snapTo('mid');
			}}
			onHeatSelect={handleHeatSelect}
			onRetry={mapData.retryLoad}
			anchor={anchorPoint}
			onSelectAnchor={handleSelectAnchor}
		/>
		</div>
	</div>

	<!-- ══ TOP BAR (fixed, floating over map — 对齐 Figma Make：无标题，仅返回 + 筛选) ══ -->
		<div style="position:absolute;top:0;left:0;right:0;z-index:50;padding:12px 14px 0;">
			<div style="display:flex;align-items:center;justify-content:space-between;">
				<!-- Back -->
				<div class="pointer-events-auto flex items-center rounded-full border border-border-default/60 bg-surface-primary/85 text-text-primary p-0.5 shadow-raised-sm backdrop-blur-md shrink-0">
					<ActionIconButton
						icon={ArrowLeft}
						label="返回"
						onClick={() => smartBack()}
						size={36}
						variant="ghost"
						tone="current"
						class="shrink-0"
					/>
				</div>

				<!-- Filter menu（桌面/移动共用组件：徽章/重置/至少一项/外点关闭统一） -->
				<MapFilterMenu bind:activeFilters align="end" />
			</div>
		</div>

	<!-- ══ RIGHT CONTROLS — bottom tracks sheet ══ -->
	<div style={rightCtrlStyle}>
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
			class="hover-btn hit-slop flex items-center justify-center rounded-full focus-ring circle-hover size-10.5 disabled:pointer-events-none disabled:opacity-faint"
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

	<!-- ══ BOTTOM SHEET (draggable in portrait, floating side panel in landscape) ══ -->
	<!-- 竖屏：固定高度面板 + transform 位移（合成器），确认 CTA 由贴底 dock 承载。
	     横屏：root 铺满地图区（pointer-events:none），侧卡仍是原来的绝对定位面板。 -->
	<div style={sheetRootStyle}>
	<div class="bg-surface-primary" style={sheetStyle}>
		{#if !isLandscape}
		<!-- Drag handle -->
		<div
			role="slider"
			aria-label="面板高度"
			aria-orientation="vertical"
			aria-valuemin={SNAP_MIN}
			aria-valuemax={SNAP_MAX}
			aria-valuenow={Math.min(SNAP_MAX, Math.max(SNAP_MIN, sheetH))}
			tabindex="0"
			style="flex-shrink:0;display:flex;flex-direction:column;align-items:center;padding-top:8px;padding-bottom:6px;cursor:{dragging ? 'grabbing' : 'grab'};touch-action:none;user-select:none;"
			onpointerdown={onHandlePointerDown}
			onkeydown={onHandleKeydown}
		>
			<!-- 拖拽把手：加宽加深一档，让「可拖动」在玻璃底上第一眼可辨；热区由整条 24px 高容器承担 -->
			<div class="h-1.5 w-11 rounded-full bg-border-strong opacity-soft transition-colors duration-rapid"></div>
		</div>
		{/if}

		<!-- Search bar -->
		<div class="shrink-0 px-4 {isLandscape ? 'pt-3 pb-2' : 'pb-2'}">
			<SearchInput
				value={searchQuery}
				onChange={(v) => {
					searchQuery = v;
					selectedLoc = null;
					if (snap === 'min') snapTo('mid');
				}}
				onFocus={() => {
					if (snap === 'min') snapTo('mid');
				}}
				placeholder="搜索地点、地标或坐标"
				ariaLabel="搜索地点、地标或坐标"
			>
				{#snippet trailing()}
					{#if selectedLoc}
						<Check class="size-3.5 shrink-0 text-accent-default" strokeWidth={2.5} aria-hidden="true" />
					{/if}
				{/snippet}
			</SearchInput>
		</div>

		<!-- Scrollable content — 收起(min)时隐藏而非卸载（保留滚动位置/输入状态；横屏恒定展示） -->
		<div style={scrollStyle} class="no-scrollbar" aria-hidden={!isLandscape && snap === 'min'}>
			<!-- 搜索建议/选中地点/最近使用贴着搜索框（主操作路径）；浏览锚点卡片下沉到列表之后，
			     避免它在 mid 档把「最近使用」空态顶到首屏之外（射程内仍可滚动查看）。 -->
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
			<!-- 浏览锚点卡片：与桌面端同组件，明确「已保存锚点 / 地图中心」并支持恢复默认 -->
			<div class="px-4 pt-1 pb-3">
				<BrowseAnchorCard
					anchor={anchorPoint}
					center={mapCenter}
					centerName={locationName}
					{defaultName}
					onReset={handleResetAnchor}
				/>
			</div>
		</div>

		{#if isLandscape}
			<!-- 横屏侧卡：确认条留在面板内（面板不做位移，无需 dock） -->
			<div
				class="shrink-0 px-4 pt-2"
				style="padding-bottom:max(18px, env(safe-area-inset-bottom, 18px));"
			>
				{@render confirmBar()}
			</div>
		{/if}
	</div>

	{#if !isLandscape}
		<!-- 贴底确认 dock：不随面板 transform 移动，收起到任意档位 CTA 都常驻视口底部；
		     dock 高度实测后参与列表底部净空（scrollStyle），滚到底时最后一项停在其上沿。 -->
		<div
			class="pointer-events-auto absolute inset-x-0 bottom-0 z-[1] bg-surface-primary px-4 pt-2"
			style="padding-bottom:max(18px, env(safe-area-inset-bottom, 18px));"
			bind:clientHeight={dockH}
		>
			{@render confirmBar()}
		</div>
	{/if}
	</div>
</div>

<svelte:window onkeydown={(e) => { if (e.key === 'Escape') handleMapTap(); }} />