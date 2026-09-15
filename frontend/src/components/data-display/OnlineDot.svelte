<script lang="ts">
	import { cn } from '@/lib/utils';

	// OnlineDot — 在线状态圆点（绝对定位在头像包装内）。
	// className 经 tailwind-merge 合并：大头像角标可覆写内边距（如 bottom-1 right-1）。

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
	data-online-dot
	class={cn('absolute bottom-0 right-0 rounded-full border-strong border-surface-primary', className)}
	style="width:{px}px;height:{px}px;background:{color ?? 'var(--cst-status-online)'};"
></span>
