<script lang="ts">
	import type { Pin } from './location-data';
	import { markerSvg, PIN_COLORS, PIN_LABELS, type MapBounds } from './location-shared';
	import { lngToWorldX, latToWorldY, wgs84ToGcj02 } from '@/components/map/map-projection';
	import { mapSourceStore } from '@/stores/mapSource.svelte';

	// 地图图钉。
	// 定位口径：与 MapCanvas 同源——底图是 GCJ-02 瓦片时，图钉 WGS-84 坐标先经
	// wgs84ToGcj02 投到瓦片空间再算视口百分比；OSM 底图恒等直通。此前漏掉这一步，
	// 国内场景下 zoom 越大 WGS 与 GCJ 的偏差被放得越大，图钉与底图地标错位。

	let {
		pin,
		bounds,
		isSelected,
		onSelect
	}: {
		pin: Pin;
		bounds: MapBounds;
		isSelected: boolean;
		onSelect: (event: MouseEvent) => void;
	} = $props();

	// 与 MapCanvas.toTileSpace 同源：底图 GCJ-02 时把 WGS-84 图钉先纠偏到
	// 瓦片空间再投影；OSM 恒等。两个空间混用是「缩放越大 marker 越偏」的根因。
	const toTileSpace = $derived(
		mapSourceStore.source === 'amap'
			? wgs84ToGcj02
			: (lat: number, lng: number) => ({ lat, lng })
	);

	const pos = $derived.by(() => {
		// 与 MapHeatDot 同口径：Web Mercator 世界坐标比例（zoom 因子在比值中
		// 约去）。大跨度视口下线性纬度与瓦片发散；城市小视口差异不可见。
		//
		// 关键：bounds 与 pin 必须投到同一坐标系再算百分比。MapCanvas 对外
		// bounds 是 WGS-84（经 gcj02ToWgs84 反演）；底图瓦片却是 GCJ-02 空间
		// 渲染——若 pin 用 WGS-84 直接投影，国内场景下两者偏差在 zoom 放大下
		// 被放得越来越大（marker 缩放漂移根因）。
		//
		// 修法：把 bounds 与 pin 一起经 toTileSpace 投到瓦片空间（GCJ-02），
		// 同空间做墨卡托投影算百分比，与底图严格对齐。
		const z = 1;
		const sw = toTileSpace(bounds.minLat, bounds.minLng);
		const ne = toTileSpace(bounds.maxLat, bounds.maxLng);
		const tilePin = toTileSpace(pin.lat, pin.lng);
		const xSpan = lngToWorldX(ne.lng, z) - lngToWorldX(sw.lng, z);
		const ySpan = latToWorldY(sw.lat, z) - latToWorldY(ne.lat, z);
		const x = xSpan > 0 ? ((lngToWorldX(tilePin.lng, z) - lngToWorldX(sw.lng, z)) / xSpan) * 100 : 0;
		// y 自视口顶（maxLat，北）向下量：worldY 随纬度减小而增大（南增北减）。
		const y = ySpan > 0 ? ((latToWorldY(tilePin.lat, z) - latToWorldY(ne.lat, z)) / ySpan) * 100 : 0;
		return { x, y };
	});
</script>

<button
	type="button"
	class="pointer-events-auto absolute focus-ring marker-hit"
	style="left:{pos.x}%;top:{pos.y}%;transform:translate(-50%, -100%) scale({isSelected ? 1.32 : 1});z-index:{isSelected ? 30 : 10};transition:transform var(--cst-motion-standard) var(--cst-ease-bounce);background:transparent;"
	aria-label={pin.name || PIN_LABELS[pin.type]}
	onclick={onSelect}
>
	{#if (pin.count ?? 1) > 1}
		<!-- 聚合计数徽章：字重与全站数字强调一致（semibold），10px 下与 bold 无视觉差异 -->
		<div
			class="absolute -top-1 -right-1 flex items-center justify-center rounded-full z-10 text-xxs font-semibold leading-none"
			style="width:16px;height:16px;background:{PIN_COLORS[pin.type]};border:var(--cst-border-width-regular) solid var(--cst-content-on-marker);color:var(--cst-content-on-marker);"
		>
			{pin.count}
		</div>
	{/if}
	
	{@html markerSvg(pin.type, pin.count ?? 1, { w: 36, h: 48 })}
	{#if isSelected}
		<span
			class="absolute rounded-full pointer-events-none opacity-faint"
			style="inset:8px 2px;background:{PIN_COLORS[pin.type]};filter:blur(var(--cst-blur-sm));"
		></span>
	{/if}
</button>

<style>
	/* marker 热区扩展：与 .hit-slop 同机制（::before inset:-6px），但**不能直接用 .hit-slop**——
	   它自带 position:relative，而本项目 styles/theme/utilities.css 在 tailwindcss 之后导入、
	   同 @layer utilities 同权重下后者胜，会把 marker 的 absolute 压成 relative：
	   2026-10-18 实测 17 个 marker 只剩 12 个仍落在覆盖层内、地理投影失效。
	   这里只补伪元素、不碰 position，定位与 translate(-50%,-100%) 锚点均不受影响。 */
	.marker-hit::before {
		content: '';
		position: absolute;
		inset: -6px;
	}
</style>
