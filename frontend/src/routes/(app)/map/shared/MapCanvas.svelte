<script lang="ts">
	import type { MapBounds } from './location-shared';
	import { lngToWorldX, latToWorldY, worldXToLng, worldYToLat, wgs84ToGcj02, gcj02ToWgs84, tilesForViewport, viewportToBounds, type TileItem } from './map-projection';

	// 地图画布：自渲染 XYZ 瓦片层（高德瓦片经 nginx /amap-tiles/ 同源代理）
	// + 拖拽平移 + 滚轮缩放。图钉/热力/我的位置由父组件以 bind:bounds 的
	// 视口地理边界自行定位（小跨度下线性百分比与墨卡托完全对齐）。
	//
	// 坐标口径：center/bounds 对外均为 WGS-84；仅瓦片像素空间内部先换算
	// GCJ-02（高德底图口径），拖拽位移同样在 GCJ 空间换算后反演回 WGS-84。
	//
	// center 双向绑定采用「手势结束才回写」策略：拖拽过程中只更新内部渲染
	// 中心，pointerup 才写回 center，避免父级 $effect 逐帧重新拉取数据。

	let {
		center = $bindable(null),
		zoom = $bindable(14),
		bounds = $bindable({ minLat: 0, maxLat: 0, minLng: 0, maxLng: 0 }),
		stableBounds = $bindable({ minLat: 0, maxLat: 0, minLng: 0, maxLng: 0 }),
		minZoom = 13,
		maxZoom = 16,
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

	// 内部渲染中心：拖拽期间独立更新，手势结束才同步回 center prop
	let renderCenter = $state<{ lat: number; lng: number } | null>(null);
	let dragging = $state(false);
	const activeCenter = $derived(renderCenter ?? center);

	// 外部导航（选中地点/图钉/复位）→ 采纳为渲染中心
	$effect(() => {
		if (!dragging && center) renderCenter = center;
	});

	// 瓦片像素空间用 GCJ-02（高德口径）；对外 center/bounds 保持 WGS-84
	const gcjCenter = $derived(activeCenter ? wgs84ToGcj02(activeCenter.lat, activeCenter.lng) : null);

	let tiles = $derived.by(() => {
		if (!gcjCenter || width < 2 || height < 2) return [] as TileItem[];
		return tilesForViewport(gcjCenter, zoom, { width, height });
	});

	// 视口边界回写（父组件供图钉/热力定位）。渲染中心变化即更新——拖拽中
	// 图钉跟随移动，手势结束后 center prop 回写触发父级数据刷新。
	// GCJ 空间求四角后反演回 WGS-84，保证与图钉/API 同口径。
	$effect(() => {
		if (!gcjCenter || width < 2 || height < 2) return;
		const b = viewportToBounds(gcjCenter, zoom, { width, height });
		const sw = gcj02ToWgs84(b.minLat, b.minLng);
		const ne = gcj02ToWgs84(b.maxLat, b.maxLng);
		bounds = { minLat: sw.lat, minLng: sw.lng, maxLat: ne.lat, maxLng: ne.lng };
	});

	// 稳定视口边界（数据请求用）：仅依赖对外 center（手势结束才回写）与 zoom，
	// 拖拽中 renderCenter 的逐帧变化不会触发，父级数据拉取不会被逐帧放大；
	// 图钉/热力逐帧跟随继续消费每帧回写的 bounds。
	$effect(() => {
		if (!center || width < 2 || height < 2) return;
		const g = wgs84ToGcj02(center.lat, center.lng);
		const b = viewportToBounds(g, zoom, { width, height });
		const sw = gcj02ToWgs84(b.minLat, b.minLng);
		const ne = gcj02ToWgs84(b.maxLat, b.maxLng);
		stableBounds = { minLat: sw.lat, minLng: sw.lng, maxLat: ne.lat, maxLng: ne.lng };
	});

	function commitCenter(next: { lat: number; lng: number }) {
		renderCenter = next;
		center = next;
	}

	/* ── 拖拽平移（pointer 统一鼠标/触屏） ── */
	let dragStart: { x: number; y: number; center: { lat: number; lng: number } } | null = null;
	let moved = false;

	function onPointerDown(e: PointerEvent) {
		if (!activeCenter) return;
		// 合成事件（测试/自动化）没有活动 pointer，捕获会抛错——拖拽本身不依赖捕获
		try {
			(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
		} catch {
			// ignore
		}
		dragStart = { x: e.clientX, y: e.clientY, center: activeCenter };
		moved = false;
	}

	function onPointerMove(e: PointerEvent) {
		if (!dragStart) return;
		const dx = e.clientX - dragStart.x;
		const dy = e.clientY - dragStart.y;
		if (!moved && Math.abs(dx) + Math.abs(dy) < 3) return;
		moved = true;
		dragging = true;
		// 像素位移 → 中心经纬度（拖拽方向与地图相反）；GCJ 空间换算后反演回 WGS-84
		const startGcj = wgs84ToGcj02(dragStart.center.lat, dragStart.center.lng);
		const cx = lngToWorldX(startGcj.lng, zoom) - dx;
		const cy = latToWorldY(startGcj.lat, zoom) - dy;
		const g = { lat: worldYToLat(cy, zoom), lng: worldXToLng(cx, zoom) };
		renderCenter = gcj02ToWgs84(g.lat, g.lng);
	}

	function onPointerUp() {
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

	/* ── 滚轮缩放（以视口中心为锚） ── */
	let wheelAcc = 0;
	function onWheel(e: WheelEvent) {
		e.preventDefault();
		wheelAcc += e.deltaY;
		const step = 120;
		if (Math.abs(wheelAcc) < step) return;
		const delta = wheelAcc > 0 ? -1 : 1;
		wheelAcc = 0;
		zoom = Math.max(minZoom, Math.min(maxZoom, zoom + delta));
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
	use:wheelAction
	onpointerdown={onPointerDown}
	onpointermove={onPointerMove}
	onpointerup={onPointerUp}
	role="application"
	aria-label="地图画布"
>
	{#each tiles as t (`${t.z}/${t.x}/${t.y}`)}
		<img
			src="/amap-tiles/{t.z}/{t.x}/{t.y}.png"
			alt=""
			draggable="false"
			loading="eager"
			class="absolute"
			style="left:{t.left}px;top:{t.top}px;width:256px;height:256px;user-select:none;"
		/>
	{/each}

	<span
		class="absolute bottom-1 right-2 text-xxs opacity-subtle pointer-events-none"
		style="color:var(--cst-text-secondary);"
	>
		© 高德地图
	</span>
</div>
