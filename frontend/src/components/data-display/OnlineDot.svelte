<script lang="ts">
	// OnlineDot — 在线状态圆点（绝对定位在头像包装内）。

	let {
		size = 'md',
		class: className = '',
		color
	}: {
		size?: 'sm' | 'md' | 'lg';
		class?: string;
		/** 覆盖默认在线色（如深色玻璃背景上改用亮色）。 */
		color?: string;
	} = $props();

	const sizeMap = { sm: 8, md: 10, lg: 12 } as const;
	const px = $derived(sizeMap[size]);
</script>

<span
	class={className}
	style="width:{px}px;height:{px}px;border-radius:9999px;background:{color ?? 'var(--cst-status-online)'};border:2px solid var(--cst-surface-primary);position:absolute;bottom:0;right:0;"
></span>
