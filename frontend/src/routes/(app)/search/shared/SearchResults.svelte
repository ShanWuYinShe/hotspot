<script lang="ts">
	import type { Snippet } from 'svelte';
	import { goto } from '$app/navigation';
	import { Search as SearchIcon, Users, FileText, Hash, ChevronRight } from '@lucide/svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import Skeleton from '@/components/data-display/Skeleton.svelte';
	import SkeletonList from '@/components/data-display/SkeletonList.svelte';
	import TagPill from '@/components/forms/TagPill.svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import { ROUTES } from '@/lib/routes';
	import { fadeImage, presetImageUrl, presetSrcSet } from '@/lib/images';
	import { contentFade } from '@/lib/motion';

	// 搜索页「有查询结果态」：Tab 栏 + 用户/动态/话题结果组。
	// 页面私有组件（colocate），由 search/+page.svelte 渲染。
	// 结果组卡头统一 SectionHead（icon + 标题 + 数量）；空态走新版 EmptyState（图标砖 +
	// spotlight + 建议 chips 的 extra），不再是一行灰字。

	export type TabType = 'all' | 'people' | 'posts' | 'groups' | 'topics';

	export interface SearchResultUser {
		user_id: string;
		nickname: string;
		// spec UserInfo.avatar_url：恒定 nullable
		avatar_url: string | null;
		tags?: string[];
	}
	export interface SearchResultMoment {
		moment_id: string;
		description?: string;
		images?: { url?: string }[];
		user?: SearchResultUser;
		tag?: string;
		reactions?: { count: number }[];
	}

	export interface SearchResultGroup {
		group_id: string;
		name: string;
		// spec GroupInfo.avatar_url：恒定 nullable
		avatar_url: string | null;
		members_count?: number;
	}

	export interface SearchResultsData {
		users: SearchResultUser[];
		moments: SearchResultMoment[];
		topics: string[];
		groups: SearchResultGroup[];
	}

	let {
		query,
		tab,
		tabs,
		loading,
		hasResults,
		results,
		suggestions = [],
		hotTopics = [],
		onChooseTopic,
		onTabChange,
		onChoose
	}: {
		query: string;
		tab: TabType;
		tabs: { id: TabType; label: string; count?: number }[];
		loading: boolean;
		hasResults: boolean;
		results: SearchResultsData;
		/** 输入联想（tag suggest）作为结果页顶部快速入口 */
		suggestions?: string[];
		/** 发现态热门话题（空态建议 chips 用；可为空） */
		hotTopics?: string[];
		onChooseTopic?: (value: string) => void;
		onTabChange: (id: TabType) => void;
		onChoose: (value: string) => void;
	} = $props();

	// 单 tab 为空时同样给空态（此前仅全局无结果才显示，切到空 tab 一片空白）。
	const tabEmpty = $derived(
		tab === 'people' ? results.users.length === 0
		: tab === 'posts' ? results.moments.length === 0
		: tab === 'groups' ? results.groups.length === 0
		: tab === 'topics' ? results.topics.length === 0
		: !hasResults
	);

	// 建议 chips：优先输入联想（与当前关键词同源），否则用发现态热门话题兜底。
	const chipTopics = $derived(
		(suggestions.length > 0 ? suggestions : hotTopics)
			.filter((t) => t && t.toLowerCase() !== query.trim().toLowerCase())
			.slice(0, 6)
	);
	const pickTopic = (topic: string) => (onChooseTopic ?? onChoose)(topic);
</script>

<!-- 纵向节奏：页顶留白只由页面 contentClassName pt-4 提供；过滤区（TabBar/相关话题）→
     首个结果组 16；结果组之间 24（grid 纵向 gap-y-6） -->
<!-- 不加 max-w 居中：正文与 PageHeader 共用 ContentColumn gutter，避免切换发现态/结果态时整页水平位移 -->
<section class="w-full" aria-label="搜索结果">
	<!-- 分类 Tab 统一走 TabBar（胶囊）；count 结构化透传，不做 label 反解析 -->
	<TabBar
		items={tabs}
		value={tab}
		onChange={(id) => onTabChange(id as TabType)}
	/>

	{#if suggestions.length > 0}
		<!-- 输入联想（相关话题）：一键细化关键词 -->
		<div class="mt-4 flex flex-wrap items-center gap-2" aria-label="相关话题">
			<span class="text-xs text-text-secondary">相关话题</span>
			{#each suggestions.slice(0, 8) as topic (topic)}
				<TagPill onClick={() => onChoose(topic)}>#{topic}</TagPill>
			{/each}
		</div>
	{/if}

	<div use:contentFade={loading}>
	{#if loading}
		<!-- 首屏骨架屏：搜索结果行（前导 36 = Avatar md 档，与真实用户结果行同构） -->
		<div class="mt-4 flex flex-col gap-3">
			<div class="flex items-center gap-2 px-1">
				<Skeleton class="h-4 w-12 rounded-sm" />
				<Skeleton class="h-3 w-8 rounded-sm" />
			</div>
			<SkeletonList rows={4} leading={36} label="搜索中…" rowClass="px-4 py-3.5" />
		</div>
	{:else if !hasResults || tabEmpty}
		<!-- 空态统一 EmptyState：页面级 spotlight + accent 图标软底 + 建议 chips -->
		<EmptyState
			spotlight
			iconTone="accent"
			title={`没有找到“${query}”`}
			description={tabEmpty && hasResults ? '当前分类下没有结果，切换分类或换个关键词试试。' : '换个关键词再试试，或从下面的话题开始探索。'}
			class="py-16"
		>
			{#snippet icon()}
				<SearchIcon class="size-7" strokeWidth={1.5} />
			{/snippet}
			{#snippet extra()}
				{#if chipTopics.length > 0}
					<div class="flex max-w-md flex-wrap items-center justify-center gap-2">
						{#each chipTopics as topic (topic)}
							<TagPill onClick={() => pickTopic(topic)}>#{topic}</TagPill>
						{/each}
					</div>
				{/if}
			{/snippet}
		</EmptyState>
	{:else}
		{#snippet resultGroup(title: string, Icon: typeof Users, count: number, rows: Snippet)}
			<section class="flex flex-col">
				<SectionHead title={title} icon={Icon} count={count} class="mb-2" />
				{@render rows()}
			</section>
		{/snippet}
		<div class="mt-4 grid gap-y-6 gap-x-5 lg:grid-cols-2">
			{#if (tab === 'all' || tab === 'people') && results.users.length > 0}
				<!-- 用户结果组 -->
				{#snippet userRows()}
					<ListContainer bordered>
						{#each results.users as person (person.user_id)}
							<ListItem
								onClick={() => void goto(ROUTES.user(person.user_id))}
								ariaLabel={`查看 ${person.nickname} 的资料`}
								class="flex items-center gap-3 px-4 py-3.5"
							>
								<Avatar src={person.avatar_url ?? ''} name={person.nickname} size="md" />
								<span class="min-w-0 flex-1">
									<span class="block text-md text-text-primary">{person.nickname}</span>
									<span class="mt-0.5 block text-xs text-text-secondary">
										@{person.user_id}{#if person.tags?.length} · {person.tags[0]}{/if}
									</span>
								</span>
								<ChevronRight class="size-4 text-text-secondary" />
							</ListItem>
						{/each}
					</ListContainer>
				{/snippet}
				{@render resultGroup('用户', Users, results.users.length, userRows)}
			{/if}

			{#if (tab === 'all' || tab === 'posts') && results.moments.length > 0}
				<!-- 动态结果组（补缩略图，对齐 Figma size-10 image） -->
				{#snippet momentRows()}
					<ListContainer bordered>
						{#each results.moments as moment (moment.moment_id)}
							<ListItem
								onClick={() => void goto(ROUTES.moment(moment.moment_id))}
								ariaLabel={moment.description ?? '无标题'}
								class="flex items-center gap-3 px-4 py-3.5"
							>
								{#if moment.images?.[0]?.url}
									<span class="media-placeholder flex size-10 shrink-0 items-center justify-center overflow-hidden rounded-sm">
										<img
											src={presetImageUrl(moment.images[0].url, 'thumb', 200)}
											srcset={presetSrcSet(moment.images[0].url, 'thumb')}
											sizes="40px"
											alt=""
											loading="lazy"
											decoding="async"
											class="size-10 object-cover"
											use:fadeImage
										/>
									</span>
								{:else}
									<div class="flex size-10 items-center justify-center rounded-sm bg-accent-soft">
										<FileText class="size-4 text-accent-default" strokeWidth={1.5} />
									</div>
								{/if}
								<span class="min-w-0 flex-1">
									<span class="block truncate text-md text-text-primary">{moment.description ?? '无标题'}</span>
									<span class="mt-0.5 block text-xs tabular-nums text-text-secondary">
										{moment.user?.nickname ?? '匿名'}{#if (moment.reactions ?? []).reduce((sum, r) => sum + r.count, 0)} · {(moment.reactions ?? []).reduce((sum, r) => sum + r.count, 0)} 赞{/if}
									</span>
								</span>
								<ChevronRight class="size-4 text-text-secondary" />
							</ListItem>
						{/each}
					</ListContainer>
				{/snippet}
				{@render resultGroup('动态', FileText, results.moments.length, momentRows)}
			{/if}

			{#if (tab === 'all' || tab === 'groups') && results.groups.length > 0}
				<!-- 群组结果组（复用用户组同款行语言：Avatar 圆角区分群组 + ChevronRight） -->
				{#snippet groupRows()}
					<ListContainer bordered>
						{#each results.groups as group (group.group_id)}
							<ListItem
								onClick={() => void goto(ROUTES.group(group.group_id))}
								ariaLabel={`查看群组 ${group.name}`}
								class="flex items-center gap-3 px-4 py-3.5"
							>
								<Avatar src={group.avatar_url ?? ''} name={group.name} size="md" shape="rounded" />
								<span class="min-w-0 flex-1">
									<span class="block text-md text-text-primary">{group.name}</span>
									<span class="mt-0.5 block text-xs tabular-nums text-text-secondary">
										{group.members_count ?? 0} 位成员
									</span>
								</span>
								<ChevronRight class="size-4 text-text-secondary" />
							</ListItem>
						{/each}
					</ListContainer>
				{/snippet}
				{@render resultGroup('群组', Users, results.groups.length, groupRows)}
			{/if}

			{#if (tab === 'all' || tab === 'topics') && results.topics.length > 0}
				<!-- 话题结果组 -->
				{#snippet topicRows()}
					<ListContainer bordered>
						{#each results.topics as topic (topic)}
							<ListItem
								onClick={() => onChoose(topic)}
								ariaLabel={`搜索话题 ${topic}`}
								class="flex items-center gap-3 px-4 py-3.5"
							>
								<Hash class="size-4 text-accent-default" />
								<span class="flex-1 text-md text-text-primary">#{topic}</span>
								<ChevronRight class="size-4 text-text-secondary" />
							</ListItem>
						{/each}
					</ListContainer>
				{/snippet}
				{@render resultGroup('话题', Hash, results.topics.length, topicRows)}
			{/if}
		</div>
	{/if}
	</div>
</section>