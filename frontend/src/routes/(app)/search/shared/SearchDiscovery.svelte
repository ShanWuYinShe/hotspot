<script lang="ts">
	import { Clock, ArrowUpRight, Flame } from '@lucide/svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import TagPill from '@/components/forms/TagPill.svelte';

	// 搜索页「无查询发现态」：最近搜索 + 热门话题。
	// 热门话题来自 moments/search 热门模式的 tags 聚合（页面传入）；无数据时
	// 不渲染该区块（诚实空态原则，不留死区块）。
	// 页面私有组件（colocate），由 search/+page.svelte 渲染。

	let {
		recent,
		hotTopics = [],
		onChoose,
		onChooseTopic,
		onClear
	}: {
		recent: string[];
		/** 热门话题（为空时不渲染区块） */
		hotTopics?: string[];
		onChoose: (value: string) => void;
		/** 热门话题点击：按 tag 语义搜索（moments/search tag 参数），与关键词搜索分流 */
		onChooseTopic: (value: string) => void;
		onClear: () => void;
	} = $props();
</script>

<div class="mt-5">
	<!-- 最近搜索：页面直出（eyebrow 小标 + 行动），无卡片壳——空态/少内容时
	     一张几乎空白的浮起卡会放大页面的空洞感。
	     无记录时整块隐藏（Gemini 审查 r11：干瘪的「暂无记录」文案无信息量），
	     首次搜索后自然出现 -->
	{#if recent.length > 0}
	<section aria-label="最近搜索">
		<div class="flex items-center justify-between">
			<!-- 标题图标与「热门话题」Flame 对称（Gemini 审查 r11） -->
			<h2 class="eyebrow flex items-center gap-1.5"><Clock class="size-3.5 text-accent-default" strokeWidth={1.75} />最近搜索</h2>
			<button
				type="button"
				onclick={onClear}
				class="link-btn -my-1 px-2 py-1 text-xs text-accent-default"
			>
				清除
			</button>
		</div>
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
	</section>
	{/if}

	{#if hotTopics.length > 0}
		<!-- 热门话题：热门动态 tags 聚合，点击即搜 -->
		<section aria-label="热门话题" class="mt-6">
			<div class="flex items-center gap-1.5">
				<Flame class="size-3.5 text-accent-default" strokeWidth={1.75} />
				<h2 class="eyebrow">热门话题</h2>
			</div>
			<div class="mt-3 flex flex-wrap gap-2">
				{#each hotTopics as topic (topic)}
					<TagPill onClick={() => onChooseTopic(topic)}>#{topic}</TagPill>
				{/each}
			</div>
		</section>
	{/if}
</div>
