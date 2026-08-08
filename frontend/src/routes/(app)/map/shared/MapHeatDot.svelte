<script lang="ts">
	import type { MapHeatBucket, MapBounds } from './location-shared';

	// 热力点 — 按 S2 网格桶的聚合位置渲染半透明圆点，大小随 total_count 增长。

	let { bucket, bounds }: { bucket: MapHeatBucket; bounds: MapBounds } = $props();

	const style = $derived.by(() => {
		const x = ((bucket.location.longitude - bounds.minLng) / (bounds.maxLng - bounds.minLng)) * 100;
		const y = ((bounds.maxLat - bucket.location.latitude) / (bounds.maxLat - bounds.minLat)) * 100;
		const size = Math.min(30, 8 + bucket.total_count * 1.6);
		const opacity = Math.min(0.55, 0.18 + bucket.total_count * 0.05);
		return `left:${x}%;top:${y}%;width:${size}px;height:${size}px;opacity:${opacity};`;
	});
</script>

<div
	class="absolute rounded-full pointer-events-none"
	style="transform:translate(-50%,-50%);background:var(--cst-accent-default);box-shadow:var(--cst-marker-halo) var(--cst-accent-default);{style}"
></div>
