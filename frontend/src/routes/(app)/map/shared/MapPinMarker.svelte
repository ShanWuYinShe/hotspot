<script lang="ts">
	import type { Pin } from './location-data';
	import { getMarkerUrl, PIN_COLORS, type MapBounds } from './location-shared';

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
	class="absolute outline-none"
	style="left:{pos.x}%;top:{pos.y}%;transform:translate(-50%, -100%) scale({isSelected ? 1.32 : 1});z-index:{isSelected ? 30 : 10};transition:transform 0.2s cubic-bezier(0.34,1.56,0.64,1);"
	onclick={onSelect}
>
	{#if (pin.count ?? 1) > 1}
		<div
			class="absolute -top-1 -right-1 flex items-center justify-center rounded-full z-10"
			style="width:16px;height:16px;background:{PIN_COLORS[pin.type]};border:1.5px solid var(--cst-content-on-marker);font-size:9px;color:var(--cst-content-on-marker);font-weight:700;"
		>
			{pin.count}
		</div>
	{/if}
	<img class="w-9 h-12" src={getMarkerUrl(pin.type, pin.count ?? 1)} alt={pin.type}  style="display:block" />
	{#if isSelected}
		<span
			class="absolute rounded-full pointer-events-none"
			style="inset:8px 2px;background:{PIN_COLORS[pin.type]};opacity:.3;filter:blur(7px);"
		></span>
	{/if}
</button>
