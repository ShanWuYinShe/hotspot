<script lang="ts">
	import { goto } from '$app/navigation';
	import { Search as SearchIcon, TrendingUp, Users, FileText, Clock } from '@lucide/svelte';
	import { platformStore } from '@/stores/platform.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import { searchUsers, searchMoments } from '@/api';
	import { ROUTES } from '@/lib/routes';

	// 搜索页：300ms 防抖、最近搜索（localStorage）、全部/用户/动态/标签 Tab。
	// 迁移自旧版 SearchPage。

	type TabType = 'all' | 'people' | 'posts' | 'tags';

	interface SearchResultUser {
		user_id: string;
		nickname: string;
		avatar_url?: string;
		tags?: string[];
	}
	interface SearchResultMoment {
		moment_id: string;
		description?: string;
		user?: SearchResultUser;
		tag?: string;
		likes_count?: number;
	}

	const HOT_SEARCHES: string[] = [];

	const isDesktop = $derived(platformStore.platform === 'desktop');
	const shellStyle = $derived(`max-width:${isDesktop ? 1040 : 640}px;margin-left:auto;margin-right:auto;width:100%;`);

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
	let results = $state<{ users: SearchResultUser[]; moments: SearchResultMoment[]; tags: string[] }>({ users: [], moments: [], tags: [] });
	let loading = $state(false);

	$effect(() => {
		localStorage.setItem('search_recent', JSON.stringify(recent));
	});

	const q = $derived(query.toLowerCase().trim());

	async function doSearch(term: string) {
		if (!term) {
			results = { users: [], moments: [], tags: [] };
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
				return { user_id: inner.user_id ?? '', nickname: inner.nickname ?? '', avatar_url: inner.avatar_url };
			});
			const moments = rawMoments.map((m: any) => ({
				...m,
				user: m.user ?? m.user_info
			}));
			results = { users, moments, tags: [] };
		} catch {
			results = { users: [], moments: [], tags: [] };
		} finally {
			loading = false;
		}
	}

	// 300ms 防抖
	$effect(() => {
		const term = q;
		const timer = setTimeout(() => void doSearch(term), 300);
		return () => clearTimeout(timer);
	});

	const addRecent = (s: string) => {
		recent = [s, ...recent.filter((r) => r !== s)].slice(0, 6);
	};

	const handleSearch = (s: string) => {
		query = s;
		addRecent(s);
	};

	const hasResults = $derived(results.users.length > 0 || results.moments.length > 0 || results.tags.length > 0);

	const RESULT_TABS = $derived<{ id: TabType; label: string }[]>([
		{ id: 'all', label: '全部' },
		{ id: 'people', label: `用户 ${results.users.length ? `(${results.users.length})` : ''}` },
		{ id: 'posts', label: `动态 ${results.moments.length ? `(${results.moments.length})` : ''}` },
		{ id: 'tags', label: `标签 ${results.tags.length ? `(${results.tags.length})` : ''}` }
	]);
</script>

<div class="flex flex-col h-full bg-[var(--neu-bg)]">
	<div class="flex-shrink-0">
		<PageHeader title="搜索" sticky onBack={() => history.back()} />
		<div class="px-5 py-2">
			<SearchInput value={query} onChange={(v) => (query = v)} placeholder="搜索用户、地点、话题…" />
		</div>
	</div>

	<!-- Scrollable body -->
	<div class="flex-1 overflow-y-auto no-scrollbar">
		{#if !q}
			<!-- No query: show discovery -->
			<div class="px-5 pt-4 pb-6" style={shellStyle}>
				{#if recent.length > 0}
					<div class="mb-5">
						<div class="flex items-center justify-between mb-2.5">
							<div class="flex items-center gap-1.5">
								<Clock style="width:13px;height:13px;color:var(--neu-text-muted);" strokeWidth={1.5} />
								<p class="uppercase text-[var(--neu-text-muted)] text-[10px] tracking-[0.12em]">最近搜索</p>
							</div>
							<button type="button" class="neu-hover-btn outline-none text-[var(--neu-accent-primary)] text-xs" onclick={() => (recent = [])}>
								清空
							</button>
						</div>
						<div class="flex flex-wrap gap-2">
							{#each recent as r (r)}
								<button
									type="button"
									class="neu-hover-btn flex items-center gap-1.5 px-3.5 py-1.5 rounded-full outline-none neu-surface-ghost text-[var(--neu-text-muted)] text-xs"
									onclick={() => handleSearch(r)}
								>
									<Clock style="width:10px;height:10px;" strokeWidth={1.5} />
									{r}
								</button>
							{/each}
						</div>
					</div>
				{/if}

				<div class="mb-5">
					<div class="flex items-center gap-1.5 mb-2.5">
						<TrendingUp style="width:13px;height:13px;color:var(--neu-accent-primary);" strokeWidth={2} />
						<p class="uppercase text-[var(--neu-text-muted)] text-[10px] tracking-[0.12em]">热门搜索</p>
					</div>
					{#each HOT_SEARCHES as h, i (h)}
						<button
							type="button"
							class="neu-hover-bg flex items-center gap-3 w-full py-2.5 outline-none"
							style="border-bottom:{i < HOT_SEARCHES.length - 1 ? '1px solid var(--neu-border)' : 'none'};"
							onclick={() => handleSearch(h)}
						>
							<span class="text-[var(--neu-accent-primary)] text-sm font-bold w-5 text-center">{i + 1}</span>
							<span class="text-[var(--neu-text)] text-sm">{h}</span>
							{#if i < 3}
								<span class="ml-auto px-2 py-0.5 rounded-full text-[10px] text-[var(--semantic-hot)]" style="background:color-mix(in srgb, var(--semantic-hot) 12%, transparent);">热</span>
							{/if}
						</button>
					{/each}
				</div>
			</div>
		{:else}
			<!-- Has query: show results -->
			<div style={shellStyle}>
				<div class="flex overflow-x-auto no-scrollbar px-5 pt-3 pb-0 gap-2">
					{#each RESULT_TABS as t}
						<button
							type="button"
							class="neu-hover-btn flex-shrink-0 px-3.5 py-1.5 rounded-full outline-none text-xs {tab === t.id ? 'bg-[var(--neu-accent-primary)] text-white' : 'bg-[var(--neu-bg)] text-[var(--neu-text-muted)]'}"
							style={tab === t.id
								? 'box-shadow:0 3px 10px var(--neu-glow);'
								: 'border:1px solid var(--neu-border);box-shadow:3px 3px 6px var(--neu-shadow-dark), -3px -3px 6px var(--neu-shadow-light);'}
							onclick={() => (tab = t.id)}
						>
							{t.label}
						</button>
					{/each}
				</div>

				<div class="px-5 pt-4 pb-6 flex flex-col gap-4">
					{#if loading}
						<div class="flex items-center justify-center py-12">
							<p class="text-[var(--neu-text-muted)] text-[13px]">搜索中...</p>
						</div>
					{/if}

					{#if !loading && !hasResults}
						<EmptyState title="未找到相关内容" description="尝试其他关键词">
							{#snippet icon()}
								<SearchIcon size={20} color="var(--neu-text-muted)" strokeWidth={1.25} />
							{/snippet}
						</EmptyState>
					{/if}

					{#if (tab === 'all' || tab === 'people') && results.users.length > 0}
						<div>
							{#if tab === 'all'}
								<p class="uppercase mb-2.5 text-[var(--neu-text-muted)] text-[10px] tracking-[0.14em]">用户</p>
							{/if}
							<div class="flex flex-col gap-2.5">
								{#each results.users as user (user.user_id)}
									<button
										type="button"
										class="neu-hover-card app-list-row app-list-row--rich neu-row-hover cursor-pointer rounded-[20px] px-4 w-full text-left neu-surface-flat"
										onclick={() => void goto(ROUTES.user(user.user_id))}
									>
										<div class="rounded-full overflow-hidden flex-shrink-0" style="width:42px;height:42px;">
											<img src={user.avatar_url ?? 'https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80'} alt={user.nickname} class="w-full h-full object-cover" />
										</div>
										<div class="flex-1">
											<p class="text-[var(--neu-text)] text-sm">{user.nickname}</p>
											{#if user.tags && user.tags.length > 0}
												<span class="px-2 py-0.5 rounded-full bg-[var(--neu-accent-soft)] text-[var(--neu-accent-primary)] text-[10px]">#{user.tags[0]}</span>
											{/if}
										</div>
										<Users style="width:15px;height:15px;color:var(--neu-text-muted);" strokeWidth={1.5} />
									</button>
								{/each}
							</div>
						</div>
					{/if}

					{#if (tab === 'all' || tab === 'posts') && results.moments.length > 0}
						<div>
							{#if tab === 'all'}
								<p class="uppercase mb-2.5 text-[var(--neu-text-muted)] text-[10px] tracking-[0.14em]">动态</p>
							{/if}
							<div class="flex flex-col gap-2.5">
								{#each results.moments as moment (moment.moment_id)}
									<button
										type="button"
										class="neu-hover-card app-list-row app-list-row--rich neu-row-hover cursor-pointer rounded-[20px] px-4 w-full text-left neu-surface-flat"
										onclick={() => void goto(ROUTES.moment(moment.moment_id))}
									>
										<div class="rounded-[12px] overflow-hidden flex-shrink-0 bg-[var(--neu-accent-soft)]" style="width:42px;height:42px;"></div>
										<div class="flex-1 min-w-0">
											<p class="truncate text-[var(--neu-text)] text-sm">{moment.description ?? '无标题'}</p>
											<div class="flex items-center gap-1.5 mt-0.5">
												<span class="text-[var(--neu-text-muted)] text-xs">{moment.user?.nickname ?? ''}</span>
												{#if moment.likes_count !== undefined}
													<span class="text-[var(--neu-text-muted)] text-[10px] opacity-50">·</span>
													<span class="text-[var(--neu-accent-primary)] text-[11px]">❤️ {moment.likes_count}</span>
												{/if}
											</div>
										</div>
										<FileText style="width:15px;height:15px;color:var(--neu-text-muted);" strokeWidth={1.5} />
									</button>
								{/each}
							</div>
						</div>
					{/if}

					{#if (tab === 'all' || tab === 'tags') && results.tags.length > 0}
						<div>
							{#if tab === 'all'}
								<p class="uppercase mb-2.5 text-[var(--neu-text-muted)] text-[10px] tracking-[0.14em]">标签</p>
							{/if}
							<div class="flex flex-wrap gap-2">
								{#each results.tags as t (t)}
									<button type="button" class="neu-hover-btn px-4 py-2 rounded-full outline-none neu-theme-chip text-[13px]">
										#{t}
									</button>
								{/each}
							</div>
						</div>
					{/if}
				</div>
			</div>
		{/if}
	</div>
</div>
