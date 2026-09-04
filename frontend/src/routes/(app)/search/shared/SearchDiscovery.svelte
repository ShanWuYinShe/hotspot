<script lang="ts">
	import { Clock, ArrowUpRight } from '@lucide/svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';

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

<div class="mt-5">
	<!-- 最近搜索：页面直出（eyebrow 小标 + 行动），无卡片壳——空态/少内容时
	     一张几乎空白的浮起卡会放大页面的空洞感 -->
	<section aria-label="最近搜索">
		<div class="flex items-center justify-between">
			<h2 class="eyebrow">最近搜索</h2>
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
			<ListContainer bordered class="mt-2.5">
				{#each recent as item (item)}
					<ListItem
						onClick={() => onChoose(item)}
						ariaLabel={`重新搜索 ${item}`}
						class="flex items-center gap-3 px-4 py-3"
					>
						<Clock class="size-4 text-text-secondary" strokeWidth={1.75} />
						<span class="flex-1 text-sm text-text-primary">{item}</span>
						<ArrowUpRight class="size-4 text-text-secondary" strokeWidth={1.75} />
					</ListItem>
				{/each}
			</ListContainer>
		{:else}
			<!-- 紧凑空态：一行引导文案，不占视觉重量 -->
			<p class="mt-3 text-sm text-text-secondary opacity-soft">输入关键词，搜索用户、动态与地点。</p>
		{/if}
	</section>
</div>
