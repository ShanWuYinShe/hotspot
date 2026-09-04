<script lang="ts">
	import { Clock, ArrowUpRight } from '@lucide/svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';

	// 搜索页「无查询发现态」：最近搜索。
	// 热门搜索/推荐话题无服务端数据源（search/suggest 需要非空 prefix），
	// 不再渲染恒空区块；输入联想由 SearchResults 的 suggestions 承载。
	// 页面私有组件（colocate），由 search/+page.svelte 渲染。

	let {
		recent,
		onChoose,
		onClear
	}: {
		recent: string[];
		onChoose: (value: string) => void;
		onClear: () => void;
	} = $props();
</script>

<div class="mt-6">
	<!-- 最近搜索 -->
	<section
		class="rounded-lg border border-border-default bg-surface-primary"
		aria-label="最近搜索"
	>
		<div class="flex items-center justify-between border-b border-border-default px-5 py-4">
			<div>
				<h2 class="text-sm font-medium text-text-primary">最近搜索</h2>
				<p class="mt-1 text-xs text-text-secondary">继续上一次的探索</p>
			</div>
			{#if recent.length > 0}
				<button
					type="button"
					onclick={onClear}
					class="link-btn -my-1 px-2 py-1 text-xs text-accent-default"
				>
					清除
				</button>
			{/if}
		</div>
		{#if recent.length > 0}
			<ListContainer>
				{#each recent as item (item)}
					<ListItem
						onClick={() => onChoose(item)}
						ariaLabel={`重新搜索 ${item}`}
						class="flex items-center gap-3 px-5 py-4"
					>
						<Clock class="size-4 text-text-secondary" strokeWidth={1.75} />
						<span class="flex-1 text-sm text-text-primary">{item}</span>
						<ArrowUpRight class="size-4 text-text-secondary" strokeWidth={1.75} />
					</ListItem>
				{/each}
			</ListContainer>
		{:else}
			<EmptyState bare title="还没有搜索记录" description="输入关键词搜索用户、动态与地点。" />
		{/if}
	</section>
</div>
