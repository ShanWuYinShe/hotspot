<script lang="ts">
	import type { MapBounds } from './types';
	import { lngToWorldX, latToWorldY, worldXToLng, worldYToLat, wgs84ToGcj02, gcj02ToWgs84, tilesForViewport, viewportToBounds, MAP_MIN_ZOOM, MAP_MAX_ZOOM, type TileItem } from './map-projection';
	import { mapSourceStore } from '@/stores/mapSource.svelte';

	// 地图画布：自渲染 XYZ 瓦片层（高德瓦片浏览器直连 GCJ-02 · OSM 走 /osm-tiles/
	// WGS-84，均经 nginx 同源代理）+ 拖拽平移 + 滚轮缩放。图钉/热力/我的位置
	// 由父组件以 bind:bounds 的视口地理边界自行定位（小跨度下线性百分比与
	// 墨卡托完全对齐）。
	//
	// 坐标口径：center/bounds 对外均为 WGS-84；瓦片像素空间按当前地图源选择
	// 纠偏策略——高德底图先换算 GCJ-02，OSM 底图恒等直通；拖拽位移同样在
	// 瓦片空间换算后反演回 WGS-84。
	//
	// center 双向绑定采用「手势结束才回写」策略：拖拽过程中只更新内部渲染
	// 中心，pointerup 才写回 center，避免父级 $effect 逐帧重新拉取数据。

	let {
		center = $bindable(null),
		zoom = $bindable(14),
		bounds = $bindable({ minLat: 0, maxLat: 0, minLng: 0, maxLng: 0 }),
		stableBounds = $bindable({ minLat: 0, maxLat: 0, minLng: 0, maxLng: 0 }),
		minZoom = MAP_MIN_ZOOM,
		maxZoom = MAP_MAX_ZOOM,
		onMapTap
	}: {
		center: { lat: number; lng: number } | null;
		zoom?: number;
		bounds?: MapBounds;
		stableBounds?: MapBounds;
		minZoom?: number;
		maxZoom?: number;
		onMapTap?: () => void;
	} = $props();

	let width = $state(0);
	let height = $state(0);
	let containerEl: HTMLElement | null = null;

	// 内部渲染中心：拖拽期间独立更新，手势结束才同步回 center prop
	let renderCenter = $state<{ lat: number; lng: number } | null>(null);
	let dragging = $state(false);
	const activeCenter = $derived(renderCenter ?? center);

	// 外部导航（选中地点/图钉/复位）→ 采纳为渲染中心
	$effect(() => {
		if (!dragging && center) renderCenter = center;
	});

	// 瓦片像素空间纠偏策略：高德底图为 GCJ-02（WGS-84 → GCJ 换算），OSM 底图
	// 为 WGS-84（恒等直通）。组件层其余代码不感知具体源，只调用这对函数。
	let tileSource = $derived(mapSourceStore.source);
	const toTileSpace = $derived(tileSource === 'amap' ? wgs84ToGcj02 : (lat: number, lng: number) => ({ lat, lng }));
	const fromTileSpace = $derived(tileSource === 'amap' ? gcj02ToWgs84 : (lat: number, lng: number) => ({ lat, lng }));

	// 瓦片像素空间用底图口径；对外 center/bounds 保持 WGS-84
	const tileCenter = $derived(activeCenter ? toTileSpace(activeCenter.lat, activeCenter.lng) : null);

	let tiles = $derived.by(() => {
		if (!tileCenter || width < 2 || height < 2) return [] as TileItem[];
		return tilesForViewport(tileCenter, zoom, { width, height });
	});

	// 视口边界回写（父组件供图钉/热力定位）。渲染中心变化即更新——拖拽中
	// 图钉跟随移动，手势结束后 center prop 回写触发父级数据刷新。
	// 瓦片空间求四角后反演回 WGS-84，保证与图钉/API 同口径。
	$effect(() => {
		if (!tileCenter || width < 2 || height < 2) return;
		const b = viewportToBounds(tileCenter, zoom, { width, height });
		const sw = fromTileSpace(b.minLat, b.minLng);
		const ne = fromTileSpace(b.maxLat, b.maxLng);
		bounds = { minLat: sw.lat, minLng: sw.lng, maxLat: ne.lat, maxLng: ne.lng };
	});

	// 稳定视口边界（数据请求用）：仅依赖对外 center（手势结束才回写）与 zoom，
	// 拖拽中 renderCenter 的逐帧变化不会触发，父级数据拉取不会被逐帧放大；
	// 图钉/热力逐帧跟随继续消费每帧回写的 bounds。
	$effect(() => {
		if (!center || width < 2 || height < 2) return;
		const g = toTileSpace(center.lat, center.lng);
		const b = viewportToBounds(g, zoom, { width, height });
		const sw = fromTileSpace(b.minLat, b.minLng);
		const ne = fromTileSpace(b.maxLat, b.maxLng);
		stableBounds = { minLat: sw.lat, minLng: sw.lng, maxLat: ne.lat, maxLng: ne.lng };
	});

	function commitCenter(next: { lat: number; lng: number }) {
		renderCenter = next;
		center = next;
	}

	/* ── 拖拽平移 + 双指捏合缩放（pointer 统一鼠标/触屏） ── */
	let dragStart: { x: number; y: number; center: { lat: number; lng: number } } | null = null;
	let moved = false;
	// 双指捏合（移动端地图核心手势）：跟踪活动指针，两指距离比驱动逐级缩放
	const activePointers = new Map<number, { x: number; y: number }>();
	let pinchBase = 0;
	const pinchDist = () => {
		const pts = Array.from(activePointers.values());
		return Math.hypot(pts[0].x - pts[1].x, pts[0].y - pts[1].y);
	};

	function onPointerDown(e: PointerEvent) {
		if (!activeCenter) return;
		// 合成事件（测试/自动化）没有活动 pointer，捕获会抛错——拖拽本身不依赖捕获
		try {
			(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
		} catch {
			// ignore
		}
		activePointers.set(e.pointerId, { x: e.clientX, y: e.clientY });
		if (activePointers.size === 2) {
			// 第二指落下：进入捏合，取消单指拖拽语义
			dragStart = null;
			dragging = false;
			pinchBase = pinchDist();
			return;
		}
		if (activePointers.size > 2) return;
		dragStart = { x: e.clientX, y: e.clientY, center: activeCenter };
		moved = false;
	}

	function onPointerMove(e: PointerEvent) {
		if (activePointers.has(e.pointerId)) {
			activePointers.set(e.pointerId, { x: e.clientX, y: e.clientY });
		}
		if (activePointers.size >= 2) {
			// 捏合中：距离比驱动整数级缩放（跟手 ±1 级步进，触达边界钳制）。
			// 跳级后必须以当前指距重置基准，否则 ratio 永远基于最初距离——
			// 跨过一级后每帧继续 +1，轻捏直接飙到 maxZoom（缩放不跟手的根因）。
			if (pinchBase > 0) {
				const ratio = pinchDist() / pinchBase;
				const stepLevel = Math.round(Math.log2(ratio));
				if (stepLevel !== 0) {
					const next = Math.max(minZoom, Math.min(maxZoom, zoom + stepLevel));
					if (next !== zoom) {
						// 以双指中点为锚（与滚轮鼠标锚同一套换算）：捏合放大的位置不漂移
						if (activeCenter && containerEl) {
							const pts = Array.from(activePointers.values());
							const rect = containerEl.getBoundingClientRect();
							const ax = (pts[0].x + pts[1].x) / 2 - rect.left;
							const ay = (pts[0].y + pts[1].y) / 2 - rect.top;
							const g = toTileSpace(activeCenter.lat, activeCenter.lng);
							const worldAx = lngToWorldX(g.lng, zoom) - width / 2 + ax;
							const worldAy = latToWorldY(g.lat, zoom) - height / 2 + ay;
							const k = Math.pow(2, next - zoom);
							const g1 = {
								lat: worldYToLat(worldAy * k + height / 2 - ay, next),
								lng: worldXToLng(worldAx * k + width / 2 - ax, next)
							};
							renderCenter = fromTileSpace(g1.lat, g1.lng);
						}
						zoom = next;
						pinchBase = pinchDist();
					}
				}
			}
			return;
		}
		if (!dragStart) return;
		const dx = e.clientX - dragStart.x;
		const dy = e.clientY - dragStart.y;
		if (!moved && Math.abs(dx) + Math.abs(dy) < 3) return;
		moved = true;
		dragging = true;
		// 像素位移 → 中心经纬度（拖拽方向与地图相反）；瓦片空间换算后反演回 WGS-84
		const startTile = toTileSpace(dragStart.center.lat, dragStart.center.lng);
		const cx = lngToWorldX(startTile.lng, zoom) - dx;
		const cy = latToWorldY(startTile.lat, zoom) - dy;
		const g = { lat: worldYToLat(cy, zoom), lng: worldXToLng(cx, zoom) };
		renderCenter = fromTileSpace(g.lat, g.lng);
	}

	function onPointerUp(e?: PointerEvent) {
		if (e && activePointers.has(e.pointerId)) activePointers.delete(e.pointerId);
		if (activePointers.size >= 1) {
			// 捏合结束但仍有手指残留：不进入点击/拖拽判定，等待全部抬起
			pinchBase = 0;
			return;
		}
		pinchBase = 0;
		if (!dragStart) return;
		const next = renderCenter;
		dragStart = null;
		dragging = false;
		if (moved && next) {
			commitCenter(next);
		} else if (!moved) {
			// 原地点击空白：取消选中（由父级处理）
			onMapTap?.();
		}
	}

	/* ── 滚轮缩放（以鼠标位置为锚） ── */
	let wheelAcc = 0;
	function onWheel(e: WheelEvent) {
		e.preventDefault();
		wheelAcc += e.deltaY;
		const step = 120;
		if (Math.abs(wheelAcc) < step) return;
		const delta = wheelAcc > 0 ? -1 : 1;
		wheelAcc = 0;
		const next = Math.max(minZoom, Math.min(maxZoom, zoom + delta));
		if (next === zoom || !activeCenter || !containerEl) return;
		// 以鼠标位置为锚（标准地图行为）：缩放前后锚点指向同一地理坐标，
		// 否则以视口中心为锚会让「想放大的位置」随每级跳变漂移（不跟手）。
		const rect = containerEl.getBoundingClientRect();
		const ax = e.clientX - rect.left;
		const ay = e.clientY - rect.top;
		const g = toTileSpace(activeCenter.lat, activeCenter.lng);
		const worldCx = lngToWorldX(g.lng, zoom);
		const worldCy = latToWorldY(g.lat, zoom);
		// 视口左上角世界坐标 → 锚点世界坐标；跳级后锚点像素坐标不变，
		// 反推新中心 = 新锚世界坐标 − (视口半径 − 锚点视口偏移)。
		const k = Math.pow(2, next - zoom);
		const worldAx = worldCx - width / 2 + ax;
		const worldAy = worldCy - height / 2 + ay;
		const cx = worldAx * k + width / 2 - ax;
		const cy = worldAy * k + height / 2 - ay;
		const g1 = { lat: worldYToLat(cy, next), lng: worldXToLng(cx, next) };
		commitCenter(fromTileSpace(g1.lat, g1.lng));
		zoom = next;
	}

	// Svelte 5 的 onwheel 事件属性注册为 passive 监听，preventDefault 不生效
	//（页面会跟着滚）——滚轮缩放必须阻止默认滚动，改用非被动原生监听。
	function wheelAction(node: HTMLElement) {
		node.addEventListener('wheel', onWheel, { passive: false });
		return {
			destroy() {
				node.removeEventListener('wheel', onWheel);
			}
		};
	}
</script>

<svelte:window onpointerup={() => dragStart && onPointerUp()} />

<div
	class="absolute inset-0 overflow-hidden select-none"
	style="touch-action:none;cursor:{dragging ? 'grabbing' : 'grab'};background:var(--cst-surface-page);"
	bind:clientWidth={width}
	bind:clientHeight={height}
	bind:this={containerEl}
	use:wheelAction
	onpointerdown={onPointerDown}
	onpointermove={onPointerMove}
	onpointerup={(e) => onPointerUp(e)}
	onpointercancel={(e) => onPointerUp(e)}
	role="application"
	aria-label="地图画布"
>
	{#each tiles as t (`${tileSource}/${t.z}/${t.x}/${t.y}`)}
		<img
			src={tileSource === 'amap'
				? // 高德栅格瓦片浏览器直连（免 key；<img> 不受同源策略限制）。
				  // webrd01-04 子域按坐标哈希轮换，避开单域 6 连接的浏览器并发上限。
				  `https://webrd0${((t.x + t.y) % 4) + 1}.is.autonavi.com/appmaptile?lang=zh_cn&size=1&scale=1&style=7&x=${t.x}&y=${t.y}&z=${t.z}`
				: `/osm-tiles/${t.z}/${t.x}/${t.y}.png`}
			alt=""
			draggable="false"
			loading="eager"
			class="map-tile absolute"
			style="left:{t.left}px;top:{t.top}px;width:256px;height:256px;user-select:none;"
			onload={(e) => e.currentTarget.classList.add('loaded')}
		/>
	{/each}

	<span
		class="absolute bottom-1 right-2 text-xxs opacity-subtle pointer-events-none"
		style="color:var(--cst-text-secondary);"
	>
		{tileSource === 'amap' ? '© 高德地图' : '© OpenStreetMap contributors'}
	</span>
</div>

<style>
	.map-tile {
		/* 跳级时新瓦片网络加载完成前不可见：加载完成淡入，避免整层闪变生硬。
		   已缓存瓦片 onload 同步触发，无感。 */
		opacity: 0;
	}
	/* loaded 由 onload 在运行时追加，需 :global 避免 Svelte 误判未用 */
	.map-tile:global(.loaded) {
		opacity: 1;
		transition: opacity var(--cst-motion-instant) var(--cst-ease-standard);
	}
	/* 深色主题下底图反色适配（标准 invert+hue-rotate 手法），marker/控件不受影响 */
	:global(.dark) .map-tile {
		filter: invert(1) hue-rotate(180deg) brightness(0.92) contrast(0.9) saturate(0.7);
	}
</style>
