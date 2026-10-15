<script lang="ts">
	import { ChevronRight } from '@lucide/svelte';

	// 首页分区标题。

	let { title, action, onAction, subtitle }: { title: string; action?: string; onAction?: () => void; subtitle?: string } = $props();
</script>

<div class="flex items-center justify-between mb-3">
	<!-- 分区标题用真实标题层级（15px semibold），eyebrow 仅用于表单/分组 label -->
	<div>
		<h2 class="text-md font-semibold text-text-primary">{title}</h2>
		{#if subtitle}
			<p class="mt-1 text-xs text-text-secondary">{subtitle}</p>
		{/if}
	</div>
	{#if action && onAction}
		<button
			type="button"
			class="link-btn hit-slop flex items-center gap-0.5 focus-ring -my-1.5 px-1 py-1.5 text-xs text-accent-default"
			onclick={onAction}
		>
			<span>{action}</span>
			<ChevronRight class="size-3"  strokeWidth={1.5} />
		</button>
	{/if}
</div>
