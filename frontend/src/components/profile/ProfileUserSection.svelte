<script lang="ts" generics="T extends { id: string }">
	import type { Component, Snippet } from 'svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import MetricRow, { type MetricItem } from '@/components/data-display/MetricRow.svelte';
	import TagPill from '@/components/forms/TagPill.svelte';
	import type { SectionFilter } from '@/components/profile/section-filter';

	// ProfileUserSection — 个人主页用户列表分区骨架（好友/关注/粉丝三 Section 共用）。
	// 统一：可选 header（标题+计数+副标题+右上角）、搜索框、统计行、筛选 pill 栏、
	// 双模列表容器（embedded 卡片网格 / 非 embedded 连接列表行）、空态。
	// 行渲染由 children snippet 注入（item 透传），数据源与动作差异留在各 Section。
	// 纯内容组件：滚动与 gutter 由外层 StandardPageLayout/ContentColumn 承担，不自建容器。

	let {
		title,
		subtitle,
		count,
		countSuffix = '',
		headerNote,
		search,
		stats,
		filters,
		filterValue,
		onFilter,
		emptyTitle,
		emptyDescription,
		emptyIcon,
		headerExtra,
		embedded = false,
		loading = false,
		items,
		children
	}: {
		title: string;
		subtitle?: string;
		count: number;
		countSuffix?: string;
		headerNote?: string;
		search?: { value: string; placeholder: string; onChange: (v: string) => void };
		stats?: MetricItem[];
		filters?: SectionFilter[];
		filterValue?: string;
		onFilter?: (id: string) => void;
		emptyTitle: string;
		emptyDescription: string;
		emptyIcon: Component;
		headerExtra?: Snippet;
		embedded?: boolean;
		/** 数据加载中：列表区显示统一 spinner（加载完成前不渲染空态，避免误报） */
		loading?: boolean;
		items: T[];
		children: Snippet<[T]>;
	} = $props();
</script>

<div>
	{#if !embedded}
		<!-- Header -->
		<header class="flex items-center justify-between py-4 border-b border-border-default" >
			<div>
				<div class="flex items-center gap-2.5">
					<span class="text-text-primary text-sm">{title}</span>
					<span class="rounded-full bg-accent-default px-2 py-0.5 text-xs text-text-on-accent">
						{count}{countSuffix}
					</span>
				</div>
				{#if subtitle}
					<p class="mt-1 text-text-secondary text-xs">{subtitle}</p>
				{/if}
			</div>
			{#if headerNote}
				<span class="rounded-full bg-accent-soft px-3 py-1 text-xs text-accent-default border border-accent-border" >
					{headerNote}
				</span>
			{:else if headerExtra}
				{@render headerExtra()}
			{/if}
		</header>
	{/if}

	{#if search}
		<!-- Search -->
		<div class="pt-4 pb-2">
			<SearchInput value={search.value} onChange={search.onChange} placeholder={search.placeholder} class="px-3 py-2" />
		</div>
	{/if}

	{#if stats}
		<!-- Stats row — flat joined cells -->
		<div class="py-3">
			<MetricRow items={stats} />
		</div>
	{/if}

	{#if filters}
		<!-- Filter pills — ghost chips -->
		<div class="flex gap-1.5 pt-3 pb-3">
			{#each filters as pill}
				{@const active = filterValue === pill.id}
				<TagPill
					active={active}
					icon={pill.icon}
					onClick={() => onFilter?.(pill.id)}
				>
					{pill.label}
				</TagPill>
			{/each}
		</div>
	{/if}

	{#if embedded}
		<!-- Card grid（relationships 聚合页 embedded 模式） -->
		<div class="grid w-full grid-cols-1 gap-4 pb-8 md:grid-cols-2 xl:grid-cols-3">
			{#if loading}
				<div class="col-span-full flex flex-col items-center gap-3 py-16">
					<div class="spinner size-7"></div>
					<p class="text-text-secondary text-sm">加载中…</p>
				</div>
			{:else if items.length === 0}
				<div class="col-span-full">
					<EmptyState title={emptyTitle} description={emptyDescription}>
						{#snippet icon()}
							<emptyIcon size={18}></emptyIcon>
						{/snippet}
					</EmptyState>
				</div>
			{:else}
				{#each items as item (item.id)}
					{@render children(item)}
				{/each}
			{/if}
		</div>
	{:else}
		<!-- Bordered list（独立分区页） -->
		<div class="pb-6">
			{#if loading}
				<div class="flex flex-col items-center gap-3 py-16">
					<div class="spinner size-7"></div>
					<p class="text-text-secondary text-sm">加载中…</p>
				</div>
			{:else if items.length === 0}
				<EmptyState title={emptyTitle} description={emptyDescription}>
					{#snippet icon()}
						<emptyIcon size={18}></emptyIcon>
					{/snippet}
				</EmptyState>
			{:else}
				<ListContainer bordered>
					{#each items as item (item.id)}
						{@render children(item)}
					{/each}
				</ListContainer>
			{/if}
		</div>
	{/if}
</div>
