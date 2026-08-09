<script lang="ts">
	import type { Pin } from './location-data';
	import { markerSvg, PIN_COLORS, type MapBounds } from './location-shared';

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
		const x = ((pin.lng - bounds.minLng) / (bounds.maxLng - bounds.minLng)) * 100;
		const y = ((bounds.maxLat - pin.lat) / (bounds.maxLat - bounds.minLat)) * 100;
		return { x, y };
	});
</script>

<button
	type="button"
	class="absolute focus-ring"
	style="left:{pos.x}%;top:{pos.y}%;transform:translate(-50%, -100%) scale({isSelected ? 1.32 : 1});z-index:{isSelected ? 30 : 10};transition:transform 0.2s var(--cst-ease-bounce);"
	onclick={onSelect}
>
	{#if (pin.count ?? 1) > 1}
		<div
			class="absolute -top-1 -right-1 flex items-center justify-center rounded-full z-10 text-xxs font-bold"
			style="width:16px;height:16px;background:{PIN_COLORS[pin.type]};border:1.5px solid var(--cst-content-on-marker);color:var(--cst-content-on-marker);"
		>
			{pin.count}
		</div>
	{/if}
	
	{@html markerSvg(pin.type, pin.count ?? 1, { w: 36, h: 48 })}
	{#if isSelected}
		<span
			class="absolute rounded-full pointer-events-none opacity-30"
			style="inset:8px 2px;background:{PIN_COLORS[pin.type]};filter:blur(var(--cst-blur-sm));"
		></span>
	{/if}
</button>
