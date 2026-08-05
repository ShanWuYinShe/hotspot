<script lang="ts">
	import { TrendingUp, Clock, Hash, ArrowUpRight } from '@lucide/svelte';
	import EmbeddedList from '@/components/data-display/EmbeddedList.svelte';

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
		class="rounded-panel border bg-surface-primary"
		style="border-color:var(--border-default);"
		aria-label="最近搜索"
	>
		<div class="flex items-center justify-between border-b px-5 py-4" style="border-color:var(--border-default);">
			<div>
				<h2 class="text-sm font-medium text-text-primary">最近搜索</h2>
				<p class="mt-1 text-xs text-text-secondary">继续上一次的探索</p>
			</div>
			<button
				type="button"
				onclick={onClear}
				class="hover-btn text-xs text-accent-default hover:underline"
			>
				清除
			</button>
		</div>
		{#if recent.length > 0}
			<EmbeddedList>
				{#each recent as item (item)}
					<button
						type="button"
						onclick={() => onChoose(item)}
						class="hover-bg flex w-full items-center gap-3 px-5 py-4 text-left"
					>
						<Clock class="size-4 text-text-secondary" strokeWidth={1.7} />
						<span class="flex-1 text-sm text-text-primary">{item}</span>
						<ArrowUpRight class="size-4 text-text-secondary" strokeWidth={1.7} />
					</button>
				{/each}
			</EmbeddedList>
		{:else}
			<p class="px-5 py-8 text-sm text-text-secondary">还没有搜索记录。</p>
		{/if}
	</section>

	<!-- 正在关注（热门搜索） -->
	<section
		class="rounded-panel border bg-surface-primary p-5"
		style="border-color:var(--border-default);"
		aria-label="热门搜索"
	>
		<div class="flex items-center gap-2">
			<TrendingUp class="size-4 text-accent-default" strokeWidth={1.8} />
			<h2 class="text-sm font-medium text-text-primary">正在关注</h2>
		</div>
		<div class="mt-4 space-y-1">
			{#each hot as item, index (item)}
				<button
					type="button"
					onclick={() => onChoose(item)}
					class="hover-bg flex w-full items-center gap-3 py-2 text-left hover:text-accent-default"
				>
					<span class="w-4 text-xs text-accent-default">{index + 1}</span>
					<span class="flex-1 text-sm text-text-primary">{item}</span>
				</button>
			{/each}
		</div>
	</section>

	<!-- 推荐话题（跨两列） -->
	<section
		class="rounded-panel border bg-surface-primary p-5 lg:col-span-2"
		style="border-color:var(--border-default);"
		aria-label="推荐话题"
	>
		<div class="flex items-center gap-2">
			<Hash class="size-4 text-accent-default" strokeWidth={1.8} />
			<h2 class="text-sm font-medium text-text-primary">推荐话题</h2>
		</div>
		<div class="mt-4 flex flex-wrap gap-2">
			{#each topics as topic (topic)}
				<button
					type="button"
					onclick={() => onChoose(topic)}
					class="hover-btn border bg-surface-primary px-3 py-2 text-xs text-text-primary transition-colors hover:bg-accent-soft"
					style="border-color:var(--border-default);"
				>
					#{topic}
				</button>
			{/each}
		</div>
	</section>
</div>
