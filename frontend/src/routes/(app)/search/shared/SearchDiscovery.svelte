<script lang="ts">
	import { TrendingUp, Clock, Hash, ArrowUpRight } from '@lucide/svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import TagPill from '@/components/forms/TagPill.svelte';

	// 搜索页「无查询发现态」：最近搜索 + 热门搜索 + 推荐话题。
	// 页面私有组件（colocate），由 search/+page.svelte 渲染。

	let {
		recent,
		hot,
		topics,
		onChoose,
		onClear
	}: {
		recent: string[];
		hot: string[];
		topics: string[];
		onChoose: (value: string) => void;
		onClear: () => void;
	} = $props();
</script>

<div class="mt-6 grid gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(320px,0.7fr)]">
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
			<button
				type="button"
				onclick={onClear}
				class="link-btn text-xs text-accent-default"
			>
				清除
			</button>
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
			<EmptyState bare title="还没有搜索记录" />
		{/if}
	</section>

	<!-- 正在关注 -->
	<section
		class="rounded-lg border border-border-default bg-surface-primary p-5"
		aria-label="正在关注"
	>
		<div class="flex items-center gap-2">
			<TrendingUp class="size-4 text-accent-default" strokeWidth={1.75} />
			<h2 class="text-sm font-medium text-text-primary">正在关注</h2>
		</div>
		<div class="mt-4">
			{#if hot.length > 0}
				<ListContainer>
					{#each hot as item, index (item)}
						<ListItem
							onClick={() => onChoose(item)}
							ariaLabel={`搜索 ${item}`}
							class="flex items-center gap-3 py-2"
						>
							<span class="w-4 text-xs text-accent-default">{index + 1}</span>
							<span class="flex-1 text-sm text-text-primary">{item}</span>
						</ListItem>
					{/each}
				</ListContainer>
			{:else}
				<EmptyState bare title="暂无热门搜索" />
			{/if}
		</div>
	</section>

	<!-- 推荐话题（跨两列） -->
	<section
		class="rounded-lg border border-border-default bg-surface-primary p-5 lg:col-span-2"
		aria-label="推荐话题"
	>
		<div class="flex items-center gap-2">
			<Hash class="size-4 text-accent-default" strokeWidth={1.75} />
			<h2 class="text-sm font-medium text-text-primary">推荐话题</h2>
		</div>
		<div class="mt-4 flex flex-wrap gap-2">
			{#if topics.length > 0}
				{#each topics as topic (topic)}
					<TagPill onClick={() => onChoose(topic)}>#{topic}</TagPill>
				{/each}
			{:else}
				<EmptyState bare title="暂无推荐话题" />
			{/if}
		</div>
	</section>
</div>
