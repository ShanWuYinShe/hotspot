<script lang="ts">
	import { goto } from '$app/navigation';
	import { Search as SearchIcon, Users, FileText, Hash, ArrowUpRight } from '@lucide/svelte';
	import EmbeddedList from '@/components/data-display/EmbeddedList.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import { ROUTES } from '@/lib/routes';

	// 搜索页「有查询结果态」：Tab 栏 + 用户/动态/话题结果组。
	// 页面私有组件（colocate），由 search/+page.svelte 渲染。

	export type TabType = 'all' | 'people' | 'posts' | 'topics';

	export interface SearchResultUser {
		user_id: string;
		nickname: string;
		avatar_url?: string;
		tags?: string[];
	}
	export interface SearchResultMoment {
		moment_id: string;
		description?: string;
		media?: { url?: string }[];
		user?: SearchResultUser;
		tag?: string;
		likes_count?: number;
	}

	export interface SearchResultsData {
		users: SearchResultUser[];
		moments: SearchResultMoment[];
		topics: string[];
	}

	let {
		query,
		tab,
		tabs,
		loading,
		hasResults,
		results,
		onTabChange,
		onChoose
	}: {
		query: string;
		tab: TabType;
		tabs: { id: TabType; label: string }[];
		loading: boolean;
		hasResults: boolean;
		results: SearchResultsData;
		onTabChange: (id: TabType) => void;
		onChoose: (value: string) => void;
	} = $props();
</script>

<section class="mt-5" aria-label="搜索结果">
	<!-- Tab 栏：简洁 pill，对齐 Figma（active=primary，inactive=border bg-card） -->
	<div class="no-scrollbar flex gap-2 overflow-x-auto border-b pb-3" style="border-color:var(--border-default);">
		{#each tabs as t (t.id)}
			<button
				type="button"
				onclick={() => onTabChange(t.id)}
				aria-pressed={tab === t.id}
				class="shrink-0 px-3 py-1.5 text-xs transition-colors {tab === t.id
					? 'bg-accent-default text-text-on-accent'
					: 'border bg-surface-primary text-text-secondary hover:bg-accent-soft hover:text-text-primary'}"
				style={tab === t.id ? '' : 'border-color:var(--border-default);'}
			>
				{t.label}
			</button>
		{/each}
	</div>

	{#if loading}
		<div class="flex items-center justify-center py-20">
			<p class="text-sm text-text-secondary">搜索中...</p>
		</div>
	{:else if !hasResults}
		<!-- 空态文案对齐 Figma -->
		<div class="flex flex-col items-center py-20 text-center">
			<SearchIcon class="size-8 text-text-secondary" strokeWidth={1.4} />
			<p class="mt-4 text-sm text-text-primary">没有找到“{query}”</p>
			<p class="mt-1 text-xs text-text-secondary">换个关键词再试试。</p>
		</div>
	{:else}
		<div class="mt-5 grid gap-5 xl:grid-cols-2">
			{#if (tab === 'all' || tab === 'people') && results.users.length > 0}
				<!-- 用户结果组 -->
				<section
					class="overflow-hidden rounded-panel border bg-surface-primary"
					style="border-color:var(--border-default);"
				>
					<div class="flex items-center gap-2 border-b px-4 py-3" style="border-color:var(--border-default);">
						<Users class="size-4 text-accent-default" strokeWidth={1.7} />
						<h2 class="text-sm font-medium text-text-primary">用户</h2>
					</div>
					<EmbeddedList>
						{#each results.users as person (person.user_id)}
							<button
								type="button"
								onclick={() => void goto(ROUTES.user(person.user_id))}
								class="hover-bg flex w-full items-center gap-3 px-4 py-3.5 text-left"
								style="border-color:var(--border-default);"
							>
								<Avatar src={person.avatar_url ?? ''} name={person.nickname} size={36} />
								<span class="min-w-0 flex-1">
									<span class="block text-sm text-text-primary">{person.nickname}</span>
									<span class="mt-0.5 block text-xs text-text-secondary">
										@{person.user_id}{#if person.tags?.length} · {person.tags[0]}{/if}
									</span>
								</span>
								<ArrowUpRight class="size-4 text-text-secondary" />
							</button>
						{/each}
					</EmbeddedList>
				</section>
			{/if}

			{#if (tab === 'all' || tab === 'posts') && results.moments.length > 0}
				<!-- 动态结果组（补缩略图，对齐 Figma size-10 image） -->
				<section
					class="overflow-hidden rounded-panel border bg-surface-primary"
					style="border-color:var(--border-default);"
				>
					<div class="flex items-center gap-2 border-b px-4 py-3" style="border-color:var(--border-default);">
						<FileText class="size-4 text-accent-default" strokeWidth={1.7} />
						<h2 class="text-sm font-medium text-text-primary">动态</h2>
					</div>
					<EmbeddedList>
						{#each results.moments as moment (moment.moment_id)}
							<button
								type="button"
								onclick={() => void goto(ROUTES.moment(moment.moment_id))}
								class="hover-bg flex w-full items-center gap-3 px-4 py-3.5 text-left"
								style="border-color:var(--border-default);"
							>
								{#if moment.media?.[0]?.url}
									<img src={moment.media[0].url} alt="" class="size-10 rounded-tight object-cover" />
								{:else}
									<div
										class="flex size-10 items-center justify-center rounded-tight"
										style="background:var(--accent-soft);"
									>
										<FileText class="size-4 text-accent-default" strokeWidth={1.5} />
									</div>
								{/if}
								<span class="min-w-0 flex-1">
									<span class="block truncate text-sm text-text-primary">{moment.description ?? '无标题'}</span>
									<span class="mt-0.5 block text-xs text-text-secondary">
										{moment.user?.nickname ?? '匿名'}{#if moment.likes_count} · {moment.likes_count} 赞{/if}
									</span>
								</span>
								<ArrowUpRight class="size-4 text-text-secondary" />
							</button>
						{/each}
					</EmbeddedList>
				</section>
			{/if}

			{#if (tab === 'all' || tab === 'topics') && results.topics.length > 0}
				<!-- 话题结果组 -->
				<section
					class="overflow-hidden rounded-panel border bg-surface-primary"
					style="border-color:var(--border-default);"
				>
					<div class="flex items-center gap-2 border-b px-4 py-3" style="border-color:var(--border-default);">
						<Hash class="size-4 text-accent-default" strokeWidth={1.7} />
						<h2 class="text-sm font-medium text-text-primary">话题</h2>
					</div>
					<EmbeddedList>
						{#each results.topics as topic (topic)}
							<button
								type="button"
								onclick={() => onChoose(topic)}
								class="hover-bg flex w-full items-center gap-3 px-4 py-3.5 text-left"
								style="border-color:var(--border-default);"
							>
								<Hash class="size-4 text-accent-default" />
								<span class="flex-1 text-sm text-text-primary">#{topic}</span>
								<ArrowUpRight class="size-4 text-text-secondary" />
							</button>
						{/each}
					</EmbeddedList>
				</section>
			{/if}
		</div>
	{/if}
</section>
