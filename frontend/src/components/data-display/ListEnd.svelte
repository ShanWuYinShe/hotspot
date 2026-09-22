<script lang="ts">
	import { cn } from '@/lib/utils';
	import { isInListContext } from '@/components/list/context';

	// ListEnd — 列表收尾提示（「没有更多了」的唯一实现，components/data-display）。
	// 列表/画布内容全部展示完毕后渲染，用于区分「已到底」与「加载中 / 加载失败」——
	// 此前「加载更多」按钮耗尽即消失，用户无法判断是到底还是卡住。
	// 由调用方在前置条件成立时条件渲染（有数据 且 无更多可加载）。
	// 空列表用 EmptyState，加载中用 InlineLoading，失败用 LoadError，三者不互相替代。
	// 列表容器内自动包 role=listitem（axe aria-required-children），裸用不包裹。

	let { label = '没有更多了', class: className = '' }: { label?: string; class?: string } = $props();

	const inList = isInListContext();
</script>

{#if inList}
	<div role="listitem">
		{@render row()}
	</div>
{:else}
	{@render row()}
{/if}

{#snippet row()}
	<p class={cn('py-6 text-center text-xs text-text-secondary opacity-dim', className)}>{label}</p>
{/snippet}
