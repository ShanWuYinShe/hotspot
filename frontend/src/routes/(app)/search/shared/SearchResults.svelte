<script lang="ts">
	import { goto } from '$app/navigation';
	import { Search as SearchIcon, Users, FileText, Hash, ArrowUpRight } from '@lucide/svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import TagPill from '@/components/forms/TagPill.svelte';
	import { ROUTES } from '@/lib/routes';
	import TabBar from '@/components/navigation/TabBar.svelte';

	// 搜索页「有查询结果态」：Tab 栏 + 用户/动态/话题结果组。
	// 页面私有组件（colocate），由 search/+page.svelte 渲染。

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
		onTabChange: (id: TabType) => void;
		onChoose: (value: string) => void;
	} = $props();
</script>

<section class="mt-5" aria-label="搜索结果">
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

	{#if loading}
		<InlineLoading label="搜索中…" />
	{:else if !hasResults}
		<!-- 空态统一 EmptyState（对齐全局空态契约） -->
		<EmptyState iconSize={56} title={`没有找到“${query}”`} description="换个关键词再试试。">
			{#snippet icon()}
				<SearchIcon class="size-5.5 text-text-secondary" strokeWidth={1.5} />
			{/snippet}
		</EmptyState>
	{:else}
		<div class="mt-5 grid gap-5 xl:grid-cols-2">
			{#if (tab === 'all' || tab === 'people') && results.users.length > 0}
				<!-- 用户结果组 -->
				<section
					class="overflow-hidden rounded-lg border border-border-default bg-surface-primary"
				>
					<div class="flex items-center gap-2 border-b border-border-default px-4 py-3">
						<Users class="size-4 text-accent-default" strokeWidth={1.75} />
						<h2 class="text-sm font-medium text-text-primary">用户</h2>
					</div>
					<ListContainer>
						{#each results.users as person (person.user_id)}
							<ListItem
								onClick={() => void goto(ROUTES.user(person.user_id))}
								ariaLabel={`查看 ${person.nickname} 的资料`}
								class="flex items-center gap-3 px-4 py-3.5"
							>
								<Avatar src={person.avatar_url ?? ''} name={person.nickname} size={36} />
								<span class="min-w-0 flex-1">
									<span class="block text-sm text-text-primary">{person.nickname}</span>
									<span class="mt-0.5 block text-xs text-text-secondary">
										@{person.user_id}{#if person.tags?.length} · {person.tags[0]}{/if}
									</span>
								</span>
								<ArrowUpRight class="size-4 text-text-secondary" />
							</ListItem>
						{/each}
					</ListContainer>
				</section>
			{/if}

			{#if (tab === 'all' || tab === 'posts') && results.moments.length > 0}
				<!-- 动态结果组（补缩略图，对齐 Figma size-10 image） -->
				<section
					class="overflow-hidden rounded-lg border border-border-default bg-surface-primary"
				>
					<div class="flex items-center gap-2 border-b border-border-default px-4 py-3">
						<FileText class="size-4 text-accent-default" strokeWidth={1.75} />
						<h2 class="text-sm font-medium text-text-primary">动态</h2>
					</div>
					<ListContainer>
						{#each results.moments as moment (moment.moment_id)}
							<ListItem
								onClick={() => void goto(ROUTES.moment(moment.moment_id))}
								ariaLabel={moment.description ?? '无标题'}
								class="flex items-center gap-3 px-4 py-3.5"
							>
								{#if moment.images?.[0]?.url}
		<img src={moment.images[0].url} alt="" loading="lazy" class="size-10 rounded-sm object-cover" />
								{:else}
									<div
										class="flex size-10 items-center justify-center rounded-sm bg-accent-soft"
										
									>
										<FileText class="size-4 text-accent-default" strokeWidth={1.5} />
									</div>
								{/if}
								<span class="min-w-0 flex-1">
									<span class="block truncate text-sm text-text-primary">{moment.description ?? '无标题'}</span>
									<span class="mt-0.5 block text-xs text-text-secondary">
										{moment.user?.nickname ?? '匿名'}{#if (moment.reactions ?? []).reduce((sum, r) => sum + r.count, 0)} · {(moment.reactions ?? []).reduce((sum, r) => sum + r.count, 0)} 赞{/if}
									</span>
								</span>
								<ArrowUpRight class="size-4 text-text-secondary" />
							</ListItem>
						{/each}
					</ListContainer>
				</section>
			{/if}

			{#if (tab === 'all' || tab === 'groups') && results.groups.length > 0}
				<!-- 群组结果组（复用用户组同款行语言：Avatar 圆角区分群组 + ArrowUpRight） -->
				<section
					class="overflow-hidden rounded-lg border border-border-default bg-surface-primary"
				>
					<div class="flex items-center gap-2 border-b border-border-default px-4 py-3">
						<Users class="size-4 text-accent-default" strokeWidth={1.75} />
						<h2 class="text-sm font-medium text-text-primary">群组</h2>
					</div>
					<ListContainer>
						{#each results.groups as group (group.group_id)}
							<ListItem
								onClick={() => void goto(ROUTES.group(group.group_id))}
								ariaLabel={`查看群组 ${group.name}`}
								class="flex items-center gap-3 px-4 py-3.5"
							>
								<Avatar src={group.avatar_url ?? ''} name={group.name} size={36} shape="rounded" />
								<span class="min-w-0 flex-1">
									<span class="block text-sm text-text-primary">{group.name}</span>
									<span class="mt-0.5 block text-xs text-text-secondary">
										{group.members_count ?? 0} 位成员
									</span>
								</span>
								<ArrowUpRight class="size-4 text-text-secondary" />
							</ListItem>
						{/each}
					</ListContainer>
				</section>
			{/if}

			{#if (tab === 'all' || tab === 'topics') && results.topics.length > 0}
				<!-- 话题结果组 -->
				<section
					class="overflow-hidden rounded-lg border border-border-default bg-surface-primary"
				>
					<div class="flex items-center gap-2 border-b border-border-default px-4 py-3">
						<Hash class="size-4 text-accent-default" strokeWidth={1.75} />
						<h2 class="text-sm font-medium text-text-primary">话题</h2>
					</div>
					<ListContainer>
						{#each results.topics as topic (topic)}
							<ListItem
								onClick={() => onChoose(topic)}
								ariaLabel={`搜索话题 ${topic}`}
								class="flex items-center gap-3 px-4 py-3.5"
							>
								<Hash class="size-4 text-accent-default" />
								<span class="flex-1 text-sm text-text-primary">#{topic}</span>
								<ArrowUpRight class="size-4 text-text-secondary" />
							</ListItem>
						{/each}
					</ListContainer>
				</section>
			{/if}
		</div>
	{/if}
</section>
