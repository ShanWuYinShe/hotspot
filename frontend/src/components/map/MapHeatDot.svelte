<script lang="ts">
	import type { MapHeatBucket } from '@/api/types';
	import type { MapBounds } from './types';
	import { lngToWorldX, latToWorldY, wgs84ToGcj02 } from './map-projection';
	import { mapSourceStore } from '@/stores/mapSource.svelte';

	// 热力聚合点 — 每个 S2 网格桶渲染一个「数字徽标」：显示的数目 = 该网格内
	// 对象总数（bucket.total_count），一眼可知这里聚合了多少内容。
	//
	// 为何是「中性卡面 + accent 描边」而不是实心 accent 圆：
	// 地图上实心蓝已被两个语义占用——图钉 --cst-marker-user 与 accent 同色
	// （都是 #2B5EB2），"当前位置点"也是实心蓝。而热力桶聚合的是全部类型
	// （user/group/moment），既不对应任何单一类型色，也不该与「我在这」的位置点
	// 混为一谈（用户反馈：两者同色分不清）。
	// 改为白卡面 + accent 描边的数字徽标：既保留品牌色的醒目度，又与「实心蓝点」
	// 在形体上彻底区分（空心带数字 vs 实心点）。

	let {
		bucket,
		bounds,
		// 点击下钻：与桌面左栏聚合行、详情浮层「放大地图查看」同口径
		//（center + 放大到街区级）。移动端 sheet 没有聚合列表，这是唯一的下钻入口。
		onSelect
	}: {
		bucket: MapHeatBucket;
		bounds: MapBounds;
		onSelect?: (bucket: MapHeatBucket) => void;
	} = $props();

	// 超过 999 收敛为 999+：四位数会撑破圆点，且该量级差异在缩放视图里已无意义
	const label = $derived(bucket.total_count > 999 ? '999+' : String(bucket.total_count));

	// 分类明细：与列表聚合行同一口径，用作无障碍名称。
	const breakdownText = $derived(
		[
			bucket.breakdown?.user ? `${bucket.breakdown.user} 位用户` : '',
			bucket.breakdown?.group ? `${bucket.breakdown.group} 个群组` : '',
			bucket.breakdown?.moment ? `${bucket.breakdown.moment} 条动态` : ''
		]
			.filter(Boolean)
			.join(' · ')
	);
	const ariaLabel = $derived(
		breakdownText
			? `此处聚合 ${bucket.total_count} 项：${breakdownText}`
			: `此处聚合 ${bucket.total_count} 项`
	);

	// 与 MapCanvas.toTileSpace 同源：底图 GCJ-02 时把 WGS-84 桶心与视口 bounds
	// 一起纠偏到瓦片空间再算百分比；OSM 恒等直通。bounds 与桶心必须同空间，
	// 否则国内场景下 zoom 越大偏移越明显（同 MapPinMarker 的根因与修法）。
	const toTileSpace = $derived(
		mapSourceStore.source === 'amap'
			? wgs84ToGcj02
			: (lat: number, lng: number) => ({ lat, lng })
	);

	const style = $derived.by(() => {
		// 定位用 Web Mercator 世界坐标的比例（zoom 因子在比值中约去）：
		// 大跨度视口（全国级）线性纬度与墨卡托发散明显，统一走投影才是
		// 与瓦片对齐的正解；小跨度（城市级）两种算法差异不可见。
		const z = 1;
		const sw = toTileSpace(bounds.minLat, bounds.minLng);
		const ne = toTileSpace(bounds.maxLat, bounds.maxLng);
		const tileLoc = toTileSpace(bucket.location.latitude, bucket.location.longitude);
		const xSpan = lngToWorldX(ne.lng, z) - lngToWorldX(sw.lng, z);
		const ySpan = latToWorldY(sw.lat, z) - latToWorldY(ne.lat, z);
		const x = xSpan > 0 ? ((lngToWorldX(tileLoc.lng, z) - lngToWorldX(sw.lng, z)) / xSpan) * 100 : 0;
		// y 自视口顶（maxLat，北）向下量：worldY 随纬度减小而增大（南增北减）。
		const y = ySpan > 0 ? ((latToWorldY(tileLoc.lat, z) - latToWorldY(ne.lat, z)) / ySpan) * 100 : 0;
		// 直径按数字位数分档（1-2 位 / 3 位 / 4 位）：保证数字完整显示、不贴边
		const size = label.length <= 2 ? 28 : label.length === 3 ? 34 : 40;
		// 层叠见 class 上的 z-10：当前位置点在视口正中心（z-index:5），
		// 地图中心常与数据密集区重合，不抬层会出现「热力数字被定位点整个盖住」。
		return `left:${x}%;top:${y}%;width:${size}px;height:${size}px;`;
	});
</script>

<button
	type="button"
	aria-label={ariaLabel}
	class="pointer-events-auto absolute z-10 inline-flex items-center justify-center rounded-full p-0 text-xs font-semibold leading-none shadow-raised-md focus-ring"
	style="transform:translate(-50%,-50%);background:var(--cst-surface-primary);color:var(--cst-text-primary);border:var(--cst-border-width-strong) solid var(--cst-accent-default);{style}"
	onclick={(e) => {
		e.stopPropagation();
		onSelect?.(bucket);
	}}
>{label}</button>
