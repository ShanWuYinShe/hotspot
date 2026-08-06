<script lang="ts">
	import type { Snippet } from 'svelte';
	import { ChevronRight } from '@lucide/svelte';

	// 详情栏通用小节包装。

	let {
		title,
		action,
		onAction,
		children
	}: {
		title: string;
		action?: string;
		onAction?: () => void;
		children: Snippet;
	} = $props();
</script>

<div class="surface-flat rounded-relaxed p-4">
	<div class="flex items-end justify-between mb-3">
		<span class="eyebrow">{title}</span>
		{#if action && onAction}
		<button
			type="button"
			class="hover-btn flex items-center gap-0.5 outline-none"
			style="color:var(--accent-default);"
			onclick={onAction}
		>
				<span class="text-xs" >{action}</span>
				<ChevronRight class="size-3"  strokeWidth={1.5} />
			</button>
		{/if}
	</div>
	{@render children()}
</div>
