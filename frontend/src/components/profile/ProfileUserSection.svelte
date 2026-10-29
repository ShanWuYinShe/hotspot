<script lang="ts" generics="T extends { id: string }">
	import type { Component, Snippet } from 'svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
import NeedsAuthNotice from '@/components/auth/NeedsAuthNotice.svelte';
import SkeletonList from '@/components/data-display/SkeletonList.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import Badge from '@/components/data-display/Badge.svelte';
	import MetricRow, { type MetricItem } from '@/components/data-display/MetricRow.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import type { SectionFilter } from '@/components/profile/section-filter';

	// ProfileUserSection — 个人主页用户列表分区骨架（好友/关注/粉丝三 Section 共用）。
	// 统一：可选 header（标题+计数+副标题+右上角）、搜索框、统计行、筛选 pill 栏、
	// 连接列表行（embedded 与非 embedded 同语言）、空态。
	// 行渲染由 children snippet 注入（item 透传），数据源与动作差异留在各 Section。
	// 纯内容组件：滚动与 gutter 由外层 StandardPageLayout/ContentColumn 承担，不自建容器。

	let {
		title,
		subtitle,
		count,
		countSuffix = '',
		headerNote,
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
		error = null,
		needsAuth = false,
		onRetry,
		items,
		children
	}: {
		title: string;
		subtitle?: string;
		count: number;
		countSuffix?: string;
		headerNote?: string;
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
		/** 加载失败信息：非空时列表区渲染 LoadError（重试走 onRetry），不落空态 */
		error?: string | null;
		/** 未登录（10100）：显示登录引导而非错误重试 */
		needsAuth?: boolean;
		onRetry?: () => void;
		items: T[];
		children: Snippet<[T]>;
	} = $props();
</script>

<div>
	{#if !embedded}
		<!-- 分区标题：全站唯一实现 SectionHead（标题 + 计数 + 副标题）；
		     页级硬线已清零（Round 3），靠留白分组。
		     右侧自定义位（如粉丝「待回关」徽章）与标题基线对齐。 -->
		<div class="flex items-start justify-between gap-3 pt-2 pb-3">
			<SectionHead
				class="mb-0 min-w-0 flex-1"
				{title}
				{subtitle}
				count={needsAuth ? undefined : `${count}${countSuffix}`}
			/>
			{#if headerNote}
				<Badge size="md" tone="accent" class="shrink-0">{headerNote}</Badge>
			{:else if headerExtra}
				<div class="shrink-0">{@render headerExtra()}</div>
			{/if}
		</div>
	{/if}

		<!-- Search 已上提至聚合页头部行（tabs 左 + 搜索右限宽，固定头部高度，消除
		     切 tab 时搜索行独占一行的纵向跳动，Gemini 审查 r12-r18） -->
	{#if stats && !needsAuth}
		<!-- 统计行：MetricRow strip（数字做节奏锚点，发丝竖线分隔） -->
		<div class="py-3">
			<MetricRow items={stats} variant="strip" size="sm" />
		</div>
	{/if}

	{#if filters && !embedded && !needsAuth}
		<!-- Filter tabs — 分类/筛选一律 TabBar（与 nearby/people|groups、search 一致），
		     TagPill 只用于表单内标签与多选 chips；embedded（聚合页）时页面级 tab 已存在，不再双层嵌套 -->
		<div class="pt-3 pb-3">
			<TabBar
				items={filters.map((f) => ({ id: f.id, label: f.label, icon: f.icon }))}
				value={filterValue ?? ''}
				onChange={(id) => onFilter?.(id)}
			/>
		</div>
	{/if}

	<!-- 列表区：embedded（聚合页）与独立分区页统一为连接列表行——同一语义的简单
	     关系行不再拆成一张张独立卡（卡片碎片化），与消息/通知的行语言一致。
	     embedded 时页面级 tab 已存在，分区内不重复渲染 header/filter；
	     尾部留白由页面 contentClassName 统一承担，分区内不叠 pb。 -->
	<div>
		{#if loading}
			<!-- 首屏骨架屏：关系列表行（前导 44 + 两行文本 + 行尾动作 chip），壳与真实
			     ListContainer 一致（mx-auto max-w-3xl），骨架→内容不位移 -->
			<div class="overflow-hidden rounded-lg bg-surface-island shadow-xs">
				<SkeletonList rows={5} leading={44} trailing label="加载中…" />
			</div>
		{:else if needsAuth}
			<NeedsAuthNotice spotlight title="登录后查看" description="登录后即可查看你的好友与关注关系" />
		{:else if error}
			<LoadError message={error} onRetry={onRetry} />
		{:else if items.length === 0}
			<EmptyState spotlight iconTone="accent" title={emptyTitle} description={emptyDescription} class="py-16">
				{#snippet icon()}
					<emptyIcon size={28} strokeWidth={1.5}></emptyIcon>
				{/snippet}
			</EmptyState>
		{:else}
			<!-- 桌面全宽行会让行尾动作离名字过远（扫描距离），列表限宽到阅读列；
			     但必须**居中**：限宽 + 左对齐会在宽屏上把 300+px 空白全堆在右侧，
			     观感上就是"内容没适配剩余空间"（2026-10-21 截图复核 /me/relationships）。 -->
			<ListContainer bordered class="overflow-hidden">
				{#each items as item (item.id)}
					{@render children(item)}
				{/each}
			</ListContainer>
		{/if}
	</div>
</div>