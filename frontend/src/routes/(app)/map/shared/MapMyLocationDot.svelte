<script lang="ts">
	import { lngToWorldX, latToWorldY, wgs84ToGcj02 } from '@/components/map/map-projection';
	import { mapSourceStore } from '@/stores/mapSource.svelte';
	import type { MapBounds } from './location-shared';

	// 我的位置蓝点。
	// 语义红线：蓝点代表「我的地理位置」（浏览器 GPS fix），不是地图中心。
	// 定位口径与 MapPinMarker 完全同源——底图是 GCJ-02 瓦片时先经 wgs84ToGcj02
	// 投到瓦片空间再算视口百分比；OSM 底图恒等直通。此前用 left:50%;top:50%
	// 钉视口中心模拟，缩放锚点偏离中心或拖拽后即相对底图漂移（详见
	// .agents/skills/hotspot-frontend-map/SKILL.md）。

	let {
		lat,
		lng,
		bounds,
		pulse
	}: {
		lat: number;
		lng: number;
		bounds: MapBounds;
		/** 移动端脉冲光环（halo scale/opacity 循环），桌面静态 */
		pulse: boolean;
	} = $props();

	// 与 MapPinMarker 同源：底图 GCJ-02 时把 WGS-84 坐标先纠偏到瓦片空间再投影。
	const toTileSpace = $derived(
		mapSourceStore.source === 'amap'
			? wgs84ToGcj02
			: (lat: number, lng: number) => ({ lat, lng })
	);

	const pos = $derived.by(() => {
		const z = 1;
		const sw = toTileSpace(bounds.minLat, bounds.minLng);
		const ne = toTileSpace(bounds.maxLat, bounds.maxLng);
		const tileMe = toTileSpace(lat, lng);
		const xSpan = lngToWorldX(ne.lng, z) - lngToWorldX(sw.lng, z);
		const ySpan = latToWorldY(sw.lat, z) - latToWorldY(ne.lat, z);
		const x = xSpan > 0 ? ((lngToWorldX(tileMe.lng, z) - lngToWorldX(sw.lng, z)) / xSpan) * 100 : 0;
		const y = ySpan > 0 ? ((latToWorldY(tileMe.lat, z) - latToWorldY(ne.lat, z)) / ySpan) * 100 : 0;
		return { x, y };
	});
</script>

<div
	class="pointer-events-none absolute z-[5]"
	style="left:{pos.x}%;top:{pos.y}%;transform:translate(-50%,-50%);"
>
	<div
		class="relative rounded-full"
		style="width:18px;height:18px;background:var(--cst-marker-user);border:var(--cst-border-width-heavy) solid var(--cst-surface-primary);box-shadow:var(--cst-marker-shadow) var(--cst-marker-user-glow);"
	>
		<span
			class="{pulse ? 'map-pulse-halo' : ''} absolute rounded-full"
			style="inset:{pulse ? '-7px' : '-8px'};background:var(--cst-marker-user-bg);border:var(--cst-border-width-hairline) solid var(--cst-marker-user-border);"
		></span>
	</div>
</div>
