<script lang="ts">
	import type { Pin } from './location-data';
	import { markerSvg, PIN_COLORS, PIN_LABELS, type MapBounds } from './location-shared';
	import { lngToWorldX, latToWorldY } from '@/components/map/map-projection';

	// 地图图钉。

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

	const pos = $derived.by(() => {
		// 与 MapHeatDot 同口径：Web Mercator 世界坐标比例（zoom 因子在比值中
		// 约去）。大跨度视口下线性纬度与瓦片发散；城市小视口差异不可见。
		const z = 1;
		const xSpan = lngToWorldX(bounds.maxLng, z) - lngToWorldX(bounds.minLng, z);
		const ySpan = latToWorldY(bounds.minLat, z) - latToWorldY(bounds.maxLat, z);
		const x = xSpan > 0 ? ((lngToWorldX(pin.lng, z) - lngToWorldX(bounds.minLng, z)) / xSpan) * 100 : 0;
		// y 自视口顶（maxLat，北）向下量：worldY 随纬度减小而增大（南增北减）。
		const y = ySpan > 0 ? ((latToWorldY(pin.lat, z) - latToWorldY(bounds.maxLat, z)) / ySpan) * 100 : 0;
		return { x, y };
	});
</script>

<button
	type="button"
	class="absolute focus-ring"
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
