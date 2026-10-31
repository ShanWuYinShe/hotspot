<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import LoadMoreButton from '@/components/list/LoadMoreButton.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import { contentFade } from '@/lib/motion';

	// AdminListView —— 平台管理列表通用外壳原语：
	// - 统一处理列表的加载态（InlineLoading）、错误态（LoadError + 重试）、空态（EmptyState）；
	// - 自动管理滚动渐变与加载更多分页（LoadMoreButton / ListEnd）；
	// - 保证 9 个后台管理子模块呈现 100% 统一的交互行为与视觉反馈。

	let {
		loading,
		error,
		count,
		hasMore,
		loadingMore,
		onReload,
		onLoadMore,
		loadingLabel = '加载数据中…',
		emptyTitle = '暂无匹配数据',
		emptyDescription = '换个搜索词或调整筛选条件后重试。',
		emptyIcon: EmptyIcon,
		emptyAction,
		children
	}: {
		loading: boolean;
		error: string | null;
		count: number;
		hasMore: boolean;
		loadingMore: boolean;
		onReload: () => void;
		onLoadMore: () => void;
		loadingLabel?: string;
		emptyTitle?: string;
		emptyDescription?: string;
		emptyIcon?: Component;
		emptyAction?: Snippet;
		children: Snippet;
	} = $props();
</script>

<div use:contentFade={loading}>
	{#if loading}
		<InlineLoading label={loadingLabel} />
	{:else if error}
		<LoadError message={error} onRetry={onReload} />
	{:else if count === 0}
		<EmptyState title={emptyTitle} description={emptyDescription} iconTone="neutral">
			{#snippet icon()}
				{#if EmptyIcon}
					<EmptyIcon class="size-7" strokeWidth={1.5} />
				{/if}
			{/snippet}
			{#snippet action()}
				{#if emptyAction}
					{@render emptyAction()}
				{/if}
			{/snippet}
		</EmptyState>
	{:else}
		{@render children()}
		{#if hasMore}
			<LoadMoreButton loading={loadingMore} onLoad={onLoadMore} />
		{:else}
			<ListEnd />
		{/if}
	{/if}
</div>
