<script lang="ts">
	import { ChevronRight } from '@lucide/svelte';

	// 首页分区标题。

	let { title, action, onAction }: { title: string; action?: string; onAction?: () => void } = $props();
</script>

<div class="flex items-end justify-between mb-3">
	<span class="eyebrow">{title}</span>
	{#if action && onAction}
		<button
			type="button"
			class="hover-btn flex items-center gap-0.5 focus-ring surface-ghost rounded-full px-2 py-0.5 text-accent-default"
			onclick={onAction}
		>
			<span class="text-xs">{action}</span>
			<ChevronRight class="size-3"  strokeWidth={1.5} />
		</button>
	{/if}
</div>
