<script lang="ts">
	// 地图中心准星：明确「拖地图改变的是这个点」，写入浏览锚点时以它为落点。
	// 纯视觉、不可交互（不抢地图拖拽/滚轮）；位于地图画布几何中心。
</script>

<div
	data-map-center-crosshair
	class="pointer-events-none absolute left-1/2 top-1/2 z-[6] -translate-x-1/2 -translate-y-1/2"
	aria-hidden="true"
>
	<svg class="size-6 text-accent-default" viewBox="0 0 24 24" fill="none">
		<path d="M12 2v5M12 17v5M2 12h5M17 12h5" stroke="currentColor" stroke-width="1.25" stroke-linecap="round" />
		<circle cx="12" cy="12" r="2.5" fill="currentColor" stroke="var(--cst-surface-primary)" stroke-width="1.5" />
	</svg>
</div>
