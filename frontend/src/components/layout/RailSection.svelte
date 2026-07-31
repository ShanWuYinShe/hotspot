<script lang="ts">
	import type { Snippet } from 'svelte';
	import { ChevronRight } from '@lucide/svelte';

	// 详情栏通用小节包装（迁移自 React DetailRail.tsx 内 RailSection）。

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

<div class="neu-surface-flat rounded-[18px] p-4">
	<div class="flex items-end justify-between mb-3">
		<span class="neu-eyebrow">{title}</span>
		{#if action && onAction}
			<button
				type="button"
				class="neu-hover-action flex items-center gap-0.5 outline-none"
				style="color:var(--neu-accent-primary);"
				onclick={onAction}
			>
				<span style="font-size:11px;">{action}</span>
				<ChevronRight style="width:12px;height:12px;" strokeWidth={1.5} />
			</button>
		{/if}
	</div>
	{@render children()}
</div>
