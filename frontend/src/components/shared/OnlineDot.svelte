<script lang="ts">
	// OnlineDot — 在线状态圆点（绝对定位在头像包装内）。

	let {
		size = 'md',
		className = ''
	}: {
		size?: 'sm' | 'md' | 'lg';
		className?: string;
	} = $props();

	const sizeMap = { sm: 8, md: 10, lg: 12 } as const;
	const px = $derived(sizeMap[size]);
</script>

<span
	class={className}
	style="width:{px}px;height:{px}px;border-radius:9999px;background:var(--semantic-online);border:2px solid var(--neu-bg);position:absolute;bottom:0;right:0;"
></span>
