<script lang="ts">
	import { Compass } from '@lucide/svelte';
	import Button from '@/components/ui/button.svelte';

	// 少数据页面的探索引导（仅当已有内容但 <3 条时出现）：
	// 避免「一张卡 + 整页留白」的失衡，同时给用户下一步动作；不伪造内容、不放大卡片。
	let {
		title,
		description,
		actionLabel,
		onAction
	}: {
		title: string;
		description: string;
		actionLabel: string;
		onAction: () => void;
	} = $props();
</script>

<div
	class="mx-auto mt-4 flex w-full max-w-md flex-wrap items-center gap-3 rounded-lg bg-surface-island px-4 py-3.5 shadow-xs"
>
	<span class="flex size-9 shrink-0 items-center justify-center rounded-full bg-accent-soft text-accent-default">
		<Compass class="size-4" strokeWidth={1.75} aria-hidden="true" />
	</span>
	<div class="min-w-0 flex-1">
		<p class="text-sm font-semibold text-text-primary">{title}</p>
		<p class="mt-0.5 text-xs text-text-secondary">{description}</p>
	</div>
	<Button variant="soft" size="sm" class="hit-slop shrink-0" onclick={onAction}>{actionLabel}</Button>
</div>
