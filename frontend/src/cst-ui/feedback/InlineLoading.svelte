<script lang="ts">
	// InlineLoading —— 列表/内容区加载态（spinner + 文案），收敛各页手写的
	// 「spinner + 加载xxx…」样板；与 EmptyState/LoadError 同为三态标配组件。
	// 列表容器内自动包 role=listitem（axe aria-required-children，status 语义
	// 保留在内层），裸用（网格/独立内容区）不包裹。
	import { cn } from '@/lib/utils';
	import { isInListContext } from '@/components/list/context';
	let { label = '加载中…', class: cls = '' }: { label?: string; class?: string } = $props();

	const inList = isInListContext();
</script>

{#if inList}
	<div role="listitem">
		{@render loading()}
	</div>
{:else}
	{@render loading()}
{/if}

{#snippet loading()}
	<div class={cn('flex flex-col items-center gap-3 py-20', cls)} role="status" aria-label={label}>
		<div class="spinner size-7"></div>
		<p class="text-text-secondary text-sm">{label}</p>
	</div>
{/snippet}
