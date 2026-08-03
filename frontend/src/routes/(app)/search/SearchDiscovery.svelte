<script lang="ts">
	import { TrendingUp, Clock, Hash, ArrowUpRight } from '@lucide/svelte';
	import EmbeddedList from '@/components/data-display/EmbeddedList.svelte';

	// 搜索页「无查询发现态」：最近搜索 + 热门搜索 + 推荐话题。
	// 页面私有组件（colocate），由 search/+page.svelte 渲染。

	const HOT_SEARCHES = ['植物园', '云龙湖', '周末市集', '咖啡馆', '摄影'];
	const TOPICS = ['摄影', '城市探索', '周末去哪', '隐藏咖啡馆', '徐州美食', '本地生活', '户外运动', '街头摄影'];

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

<div class="mt-6 grid gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(320px,0.7fr)]">
	<!-- 最近搜索 -->
	<section
		class="rounded-panel border bg-[var(--neu-bg)]"
		style="border-color:var(--neu-border);"
		aria-label="最近搜索"
	>
		<div class="flex items-center justify-between border-b px-5 py-4" style="border-color:var(--neu-border);">
			<div>
				<h2 class="text-sm font-medium text-[var(--neu-text)]">最近搜索</h2>
				<p class="mt-1 text-xs text-[var(--neu-text-muted)]">继续上一次的探索</p>
			</div>
			<button
				type="button"
				onclick={onClear}
				class="neu-hover-btn text-xs text-[var(--neu-accent-primary)] hover:underline"
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
						class="neu-hover-bg flex w-full items-center gap-3 px-5 py-4 text-left"
					>
						<Clock class="size-4 text-[var(--neu-text-muted)]" strokeWidth={1.7} />
						<span class="flex-1 text-sm text-[var(--neu-text)]">{item}</span>
						<ArrowUpRight class="size-4 text-[var(--neu-text-muted)]" strokeWidth={1.7} />
					</button>
				{/each}
			</EmbeddedList>
		{:else}
			<p class="px-5 py-8 text-sm text-[var(--neu-text-muted)]">还没有搜索记录。</p>
		{/if}
	</section>

	<!-- 正在关注（热门搜索） -->
	<section
		class="rounded-panel border bg-[var(--neu-bg)] p-5"
		style="border-color:var(--neu-border);"
		aria-label="热门搜索"
	>
		<div class="flex items-center gap-2">
			<TrendingUp class="size-4 text-[var(--neu-accent-primary)]" strokeWidth={1.8} />
			<h2 class="text-sm font-medium text-[var(--neu-text)]">正在关注</h2>
		</div>
		<div class="mt-4 space-y-1">
			{#each HOT_SEARCHES as item, index (item)}
				<button
					type="button"
					onclick={() => onChoose(item)}
					class="neu-hover-bg flex w-full items-center gap-3 py-2 text-left hover:text-[var(--neu-accent-primary)]"
				>
					<span class="w-4 text-xs text-[var(--neu-accent-primary)]">{index + 1}</span>
					<span class="flex-1 text-sm text-[var(--neu-text)]">{item}</span>
				</button>
			{/each}
		</div>
	</section>

	<!-- 推荐话题（跨两列） -->
	<section
		class="rounded-panel border bg-[var(--neu-bg)] p-5 lg:col-span-2"
		style="border-color:var(--neu-border);"
		aria-label="推荐话题"
	>
		<div class="flex items-center gap-2">
			<Hash class="size-4 text-[var(--neu-accent-primary)]" strokeWidth={1.8} />
			<h2 class="text-sm font-medium text-[var(--neu-text)]">推荐话题</h2>
		</div>
		<div class="mt-4 flex flex-wrap gap-2">
			{#each TOPICS as topic (topic)}
				<button
					type="button"
					onclick={() => onChoose(topic)}
					class="neu-hover-btn border bg-[var(--neu-bg)] px-3 py-2 text-xs text-[var(--neu-text)] transition-colors hover:bg-[var(--neu-accent-soft)]"
					style="border-color:var(--neu-border);"
				>
					#{topic}
				</button>
			{/each}
		</div>
	</section>
</div>
