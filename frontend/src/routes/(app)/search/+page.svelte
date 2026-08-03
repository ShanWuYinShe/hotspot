<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import {
		Search as SearchIcon,
		TrendingUp,
		Users,
		FileText,
		Clock,
		ArrowUpRight,
		Hash,
		X
	} from '@lucide/svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import EmbeddedList from '@/components/data-display/EmbeddedList.svelte';
	import { searchUsers, searchMoments } from '@/api';
	import { ROUTES } from '@/lib/routes';

	// 搜索页：最近搜索（localStorage）、热门搜索、推荐话题、全部/用户/动态/话题 Tab。
	// 对齐 Figma Make SearchPage：shadcn 卡片设计语言（rounded-panel border bg-card），
	// 简洁 pill Tab，结果行 border-b + 圆头像 + ArrowUpRight。

	type TabType = 'all' | 'people' | 'posts' | 'topics';

	interface SearchResultUser {
		user_id: string;
		nickname: string;
		avatar_url?: string;
		tags?: string[];
	}
	interface SearchResultMoment {
		moment_id: string;
		description?: string;
		media?: { url?: string }[];
		user?: SearchResultUser;
		tag?: string;
		likes_count?: number;
	}

	const HOT_SEARCHES = ['植物园', '云龙湖', '周末市集', '咖啡馆', '摄影'];
	const TOPICS = ['摄影', '城市探索', '周末去哪', '隐藏咖啡馆', '徐州美食', '本地生活', '户外运动', '街头摄影'];

	function readRecent(): string[] {
		try {
			return JSON.parse(localStorage.getItem('search_recent') ?? '[]');
		} catch {
			return [];
		}
	}

	let query = $state('');
	let tab = $state<TabType>('all');
	let recent = $state<string[]>(readRecent());
	let results = $state<{ users: SearchResultUser[]; moments: SearchResultMoment[]; topics: string[] }>({
		users: [],
		moments: [],
		topics: []
	});
	let loading = $state(false);

	$effect(() => {
		localStorage.setItem('search_recent', JSON.stringify(recent));
	});

	const keyword = $derived(query.trim().toLowerCase());

	async function doSearch(term: string) {
		if (!term) {
			results = { users: [], moments: [], topics: [] };
			return;
		}
		loading = true;
		try {
			const [usersRes, momentsRes] = await Promise.all([
				searchUsers({ keyword: term }).catch(() => ({ code: -1, data: { items: [] } })),
				searchMoments({ keyword: term }).catch(() => ({ code: -1, data: { items: [] } }))
			]);
			const rawUsers = (usersRes.code === 0 ? (usersRes.data as any)?.items : []) ?? [];
			const rawMoments = (momentsRes.code === 0 ? (momentsRes.data as any)?.items : []) ?? [];
			const users = rawUsers.map((u: any) => {
				const inner = u.user ?? u;
				return { user_id: inner.user_id ?? '', nickname: inner.nickname ?? '', avatar_url: inner.avatar_url, tags: inner.tags };
			});
			const moments = rawMoments.map((m: any) => ({
				...m,
				user: m.user ?? m.user_info
			}));
			// 话题从结果动态聚合
			const topicSet = new Set<string>();
			for (const m of moments) {
				if (m.tag) topicSet.add(m.tag);
				if (m.user?.tags) m.user.tags.forEach((t: string) => topicSet.add(t));
			}
			results = { users, moments, topics: Array.from(topicSet) };
		} catch {
			results = { users: [], moments: [], topics: [] };
		} finally {
			loading = false;
		}
	}

	// 300ms 防抖
	$effect(() => {
		const term = keyword;
		const timer = setTimeout(() => void doSearch(term), 300);
		return () => clearTimeout(timer);
	});

	// 对齐 Figma Make：进入搜索页自动聚焦输入框
	let inputEl: HTMLInputElement | undefined;
	onMount(() => {
		inputEl?.focus();
	});

	const remember = (value: string) => {
		recent = [value, ...recent.filter((r) => r !== value)].slice(0, 5);
	};
	const choose = (value: string) => {
		query = value;
		remember(value);
		tab = 'all';
	};

	const hasResults = $derived(results.users.length + results.moments.length + results.topics.length > 0);

	const tabs = $derived<{ id: TabType; label: string }[]>([
		{ id: 'all', label: '全部' },
		{ id: 'people', label: `用户 ${results.users.length || ''}` },
		{ id: 'posts', label: `动态 ${results.moments.length || ''}` },
		{ id: 'topics', label: `话题 ${results.topics.length || ''}` }
	]);
</script>

<StandardPageLayout contentClassName="py-5 sm:py-6">
	{#snippet header()}
		<PageHeader title="搜索" onBack={() => history.back()} sticky={false} />
	{/snippet}
	<!-- 搜索输入卡片（对齐 Figma：rounded-panel border bg-card） -->
	<section
		class="rounded-panel border bg-[var(--neu-bg)] p-3"
		style="border-color:var(--neu-border);"
		aria-label="搜索输入"
	>
		<div class="flex items-center gap-3">
			<SearchIcon class="size-5 shrink-0 text-[var(--neu-accent-primary)]" strokeWidth={1.8} />
			<!-- svelte-ignore a11y_autofocus -->
			<input
				bind:this={inputEl}
				autofocus
				value={query}
				oninput={(e) => (query = e.currentTarget.value)}
				onkeydown={(e) => {
					if (e.key === 'Enter' && query.trim()) remember(query.trim());
				}}
				placeholder="搜索用户、动态、地点或话题"
				class="min-w-0 flex-1 bg-transparent py-2 text-[15px] text-[var(--neu-text)] outline-none placeholder:text-[var(--neu-text-muted)]"
			/>
			{#if query}
				<button
					type="button"
					onclick={() => (query = '')}
					aria-label="清除搜索"
					class="neu-hover-btn flex size-8 items-center justify-center text-[var(--neu-text-muted)] hover:text-[var(--neu-text)]"
				>
					<X class="size-4" />
				</button>
			{/if}
		</div>
	</section>

	{#if !keyword}
		<!-- 无查询发现态（对齐 Figma：双栏 grid + 推荐话题跨列） -->
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
						onclick={() => (recent = [])}
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
							onclick={() => choose(item)}
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
							onclick={() => choose(item)}
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
							onclick={() => choose(topic)}
							class="neu-hover-btn border bg-[var(--neu-bg)] px-3 py-2 text-xs text-[var(--neu-text)] transition-colors hover:bg-[var(--neu-accent-soft)]"
							style="border-color:var(--neu-border);"
						>
							#{topic}
						</button>
					{/each}
				</div>
			</section>
		</div>
	{:else}
		<!-- 有查询结果态（对齐 Figma） -->
		<section class="mt-5" aria-label="搜索结果">
			<!-- Tab 栏：简洁 pill，对齐 Figma（active=primary，inactive=border bg-card） -->
			<div class="no-scrollbar flex gap-2 overflow-x-auto border-b pb-3" style="border-color:var(--neu-border);">
				{#each tabs as t (t.id)}
					<button
						type="button"
						onclick={() => (tab = t.id)}
						aria-pressed={tab === t.id}
						class="shrink-0 px-3 py-1.5 text-xs transition-colors {tab === t.id
							? 'bg-[var(--neu-accent-primary)] text-[var(--text-on-accent)]'
							: 'border bg-[var(--neu-bg)] text-[var(--neu-text-muted)] hover:bg-[var(--neu-accent-soft)] hover:text-[var(--neu-text)]'}"
						style={tab === t.id ? '' : 'border-color:var(--neu-border);'}
					>
						{t.label}
					</button>
				{/each}
			</div>

			{#if loading}
				<div class="flex items-center justify-center py-20">
					<p class="text-sm text-[var(--neu-text-muted)]">搜索中...</p>
				</div>
			{:else if !hasResults}
				<!-- 空态文案对齐 Figma -->
				<div class="flex flex-col items-center py-20 text-center">
					<SearchIcon class="size-8 text-[var(--neu-text-muted)]" strokeWidth={1.4} />
					<p class="mt-4 text-sm text-[var(--neu-text)]">没有找到“{query}”</p>
					<p class="mt-1 text-xs text-[var(--neu-text-muted)]">换个关键词再试试。</p>
				</div>
			{:else}
				<div class="mt-5 grid gap-5 xl:grid-cols-2">
					{#if (tab === 'all' || tab === 'people') && results.users.length > 0}
						<!-- 用户结果组 -->
						<section
							class="overflow-hidden rounded-panel border bg-[var(--neu-bg)]"
							style="border-color:var(--neu-border);"
						>
							<div class="flex items-center gap-2 border-b px-4 py-3" style="border-color:var(--neu-border);">
								<Users class="size-4 text-[var(--neu-accent-primary)]" strokeWidth={1.7} />
								<h2 class="text-sm font-medium text-[var(--neu-text)]">用户</h2>
							</div>
       <EmbeddedList>
							{#each results.users as person (person.user_id)}
								<button
									type="button"
									onclick={() => void goto(ROUTES.user(person.user_id))}
									class="neu-hover-bg flex w-full items-center gap-3 px-4 py-3.5 text-left"
									style="border-color:var(--neu-border);"
								>
									<img
										src={person.avatar_url ?? 'https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80'}
										alt=""
										class="size-9 rounded-full object-cover"
									/>
									<span class="min-w-0 flex-1">
										<span class="block text-sm text-[var(--neu-text)]">{person.nickname}</span>
										<span class="mt-0.5 block text-xs text-[var(--neu-text-muted)]">
											@{person.user_id}{#if person.tags?.length} · {person.tags[0]}{/if}
										</span>
									</span>
									<ArrowUpRight class="size-4 text-[var(--neu-text-muted)]" />
								</button>
							{/each}
       </EmbeddedList>
						</section>
					{/if}

					{#if (tab === 'all' || tab === 'posts') && results.moments.length > 0}
						<!-- 动态结果组（补缩略图，对齐 Figma size-10 image） -->
						<section
							class="overflow-hidden rounded-panel border bg-[var(--neu-bg)]"
							style="border-color:var(--neu-border);"
						>
							<div class="flex items-center gap-2 border-b px-4 py-3" style="border-color:var(--neu-border);">
								<FileText class="size-4 text-[var(--neu-accent-primary)]" strokeWidth={1.7} />
								<h2 class="text-sm font-medium text-[var(--neu-text)]">动态</h2>
							</div>
       <EmbeddedList>
							{#each results.moments as moment (moment.moment_id)}
								<button
									type="button"
									onclick={() => void goto(ROUTES.moment(moment.moment_id))}
									class="neu-hover-bg flex w-full items-center gap-3 px-4 py-3.5 text-left"
									style="border-color:var(--neu-border);"
								>
									{#if moment.media?.[0]?.url}
										<img src={moment.media[0].url} alt="" class="size-10 rounded-tight object-cover" />
									{:else}
										<div
											class="flex size-10 items-center justify-center rounded-tight"
											style="background:var(--neu-accent-soft);"
										>
											<FileText class="size-4 text-[var(--neu-accent-primary)]" strokeWidth={1.5} />
										</div>
									{/if}
									<span class="min-w-0 flex-1">
										<span class="block truncate text-sm text-[var(--neu-text)]">{moment.description ?? '无标题'}</span>
										<span class="mt-0.5 block text-xs text-[var(--neu-text-muted)]">
											{moment.user?.nickname ?? '匿名'}{#if moment.likes_count} · {moment.likes_count} 赞{/if}
										</span>
									</span>
									<ArrowUpRight class="size-4 text-[var(--neu-text-muted)]" />
								</button>
							{/each}
       </EmbeddedList>
						</section>
					{/if}

					{#if (tab === 'all' || tab === 'topics') && results.topics.length > 0}
						<!-- 话题结果组 -->
						<section
							class="overflow-hidden rounded-panel border bg-[var(--neu-bg)]"
							style="border-color:var(--neu-border);"
						>
							<div class="flex items-center gap-2 border-b px-4 py-3" style="border-color:var(--neu-border);">
								<Hash class="size-4 text-[var(--neu-accent-primary)]" strokeWidth={1.7} />
								<h2 class="text-sm font-medium text-[var(--neu-text)]">话题</h2>
							</div>
       <EmbeddedList>
							{#each results.topics as topic (topic)}
								<button
									type="button"
									onclick={() => choose(topic)}
									class="neu-hover-bg flex w-full items-center gap-3 px-4 py-3.5 text-left"
									style="border-color:var(--neu-border);"
								>
									<Hash class="size-4 text-[var(--neu-accent-primary)]" />
									<span class="flex-1 text-sm text-[var(--neu-text)]">#{topic}</span>
									<ArrowUpRight class="size-4 text-[var(--neu-text-muted)]" />
								</button>
							{/each}
       </EmbeddedList>
						</section>
					{/if}
				</div>
			{/if}
		</section>
	{/if}
</StandardPageLayout>
