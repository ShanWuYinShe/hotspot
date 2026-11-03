<script lang="ts">
	import { Navigation } from '@lucide/svelte';
	import { lngToWorldX, latToWorldY, wgs84ToGcj02 } from '@/components/map/map-projection';
	import { mapSourceStore } from '@/stores/mapSource.svelte';
	import type { MapPoint } from '@/lib/map-auto-locate';
	import { anchorDisplayName } from './browse-anchor';
	import type { MapBounds } from './location-shared';

	// 已保存浏览锚点的地图标记：与实体图钉（用户/群组/瞬间）和 GPS 蓝点明确区分——
	// 锚点代表「附近页距离基准」，不是地图中心也不是我的位置。
	// 投影口径与 MapPinMarker / MapMyLocationDot 完全同源（GCJ-02 纠偏后按 Mercator 比例）。
	let {
		anchor,
		bounds,
		onSelect
	}: {
		anchor: MapPoint;
		bounds: MapBounds;
		onSelect: () => void;
	} = $props();

	const toTileSpace = $derived(
		mapSourceStore.source === 'amap' ? wgs84ToGcj02 : (lat: number, lng: number) => ({ lat, lng })
	);

	const pos = $derived.by(() => {
		const z = 1;
		const sw = toTileSpace(bounds.minLat, bounds.minLng);
		const ne = toTileSpace(bounds.maxLat, bounds.maxLng);
		const tile = toTileSpace(anchor.lat, anchor.lng);
		const xSpan = lngToWorldX(ne.lng, z) - lngToWorldX(sw.lng, z);
		const ySpan = latToWorldY(sw.lat, z) - latToWorldY(ne.lat, z);
		const x = xSpan > 0 ? ((lngToWorldX(tile.lng, z) - lngToWorldX(sw.lng, z)) / xSpan) * 100 : 0;
		const y = ySpan > 0 ? ((latToWorldY(tile.lat, z) - latToWorldY(ne.lat, z)) / ySpan) * 100 : 0;
		return { x, y };
	});
</script>

<button
	type="button"
	class="pointer-events-auto absolute z-[16] flex size-11 flex-col items-center justify-end rounded-full focus-ring"
	style="left:{pos.x}%;top:{pos.y}%;transform:translate(-50%,-100%);"
	aria-label={`浏览锚点：${anchorDisplayName(anchor)}，点击回到此处`}
	title={`浏览锚点：${anchorDisplayName(anchor)}`}
	onclick={(e) => {
		e.stopPropagation();
		onSelect();
	}}
>
	<span class="flex max-w-40 items-center gap-1 rounded-full bg-accent-default px-2 py-0.5 text-xxs font-semibold leading-none text-text-on-accent shadow-raised-sm">
		<Navigation class="size-2.5 shrink-0" strokeWidth={2.25} aria-hidden="true" />
		<span class="truncate">锚点 · {anchorDisplayName(anchor)}</span>
	</span>
	<span class="mx-auto mt-0.5 block size-1.5 rotate-45 bg-accent-default" aria-hidden="true"></span>
</button>
