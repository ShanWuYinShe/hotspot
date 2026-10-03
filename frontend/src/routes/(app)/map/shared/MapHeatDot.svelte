<script lang="ts">
	import type { MapHeatBucket, MapBounds } from './location-shared';

	// 热力聚合点 — 每个 S2 网格桶渲染一个「实色圆 + 数目」的标记。
	// 数目 = 该网格内对象总数（bucket.total_count），一眼可知这里聚合了多少内容。
	// 此前是 0.4~0.8 透明度的纯蓝点、无任何文字，看不出它是什么（用户反馈）；
	// 现复用图钉聚合数字的徽标语言：实色底 + 反色数字 + 描边。

	let { bucket, bounds }: { bucket: MapHeatBucket; bounds: MapBounds } = $props();

	// 超过 999 收敛为 999+：四位数会撑破圆点，且该量级差异在缩放视图里已无意义
	const label = $derived(bucket.total_count > 999 ? '999+' : String(bucket.total_count));

	// 分类明细：与列表聚合行同一口径，用作无障碍名称。
	// 不做悬浮提示——本元素 pointer-events:none，点击需穿透到地图画布。
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

	const style = $derived.by(() => {
		const x = ((bucket.location.longitude - bounds.minLng) / (bounds.maxLng - bounds.minLng)) * 100;
		const y = ((bounds.maxLat - bucket.location.latitude) / (bounds.maxLat - bounds.minLat)) * 100;
		// 直径按数字位数分档（1-2 位 / 3 位 / 4 位）：保证数字完整显示、不贴边
		const size = label.length <= 2 ? 28 : label.length === 3 ? 34 : 40;
		// 层叠见 class 上的 z-10：当前位置点在视口正中心（z-index:5），
		// 地图中心常与数据密集区重合，不抬层会出现"热力数字被定位点整个盖住"。
		return `left:${x}%;top:${y}%;width:${size}px;height:${size}px;`;
	});
</script>

<div
	role="img"
	aria-label={ariaLabel}
	class="absolute z-10 inline-flex items-center justify-center rounded-full pointer-events-none text-xs font-semibold leading-none"
	style="transform:translate(-50%,-50%);background:var(--cst-accent-default);color:var(--cst-text-on-accent);border:var(--cst-border-width-regular) solid var(--cst-content-on-marker);box-shadow:var(--cst-marker-shadow) var(--cst-marker-user-glow);{style}"
>{label}</div>
