<script lang="ts">
	import { onMount } from 'svelte';
	import { Search as SearchIcon, X } from '@lucide/svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import { searchUsers, searchMoments, searchSuggest } from '@/api';
	import SearchDiscovery from './SearchDiscovery.svelte';
	import SearchResults, { type TabType, type SearchResultsData } from './SearchResults.svelte';

	// 搜索页：最近搜索（localStorage）、热门搜索、推荐话题、全部/用户/动态/话题 Tab。
	// 对齐 Figma Make SearchPage：shadcn 卡片设计语言（rounded-panel border bg-card），
	// 简洁 pill Tab，结果行 border-b + 圆头像 + ArrowUpRight。
	// 发现态与结果态分别由 SearchDiscovery / SearchResults 私有组件承载。

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
	let discoveryHot = $state<string[]>([]);
	let discoveryTopics = $state<string[]>([]);
	let results = $state<SearchResultsData>({
		users: [],
		moments: [],
		topics: []
	});
	let loading = $state(false);

	$effect(() => {
		localStorage.setItem('search_recent', JSON.stringify(recent));
	});

	// 发现态的热门搜索/推荐话题来自 search/suggest（MSW mock，后端契约一致）。
	$effect(() => {
		let cancelled = false;
		void searchSuggest().then((res) => {
			if (cancelled) return;
			const suggest = res as { code: number; data?: { items?: Array<{ payload?: { name?: string } }> } | null };
			const items = suggest.code === 0 ? suggest.data?.items ?? [] : [];
			const names = items.map((i) => i.payload?.name ?? '').filter(Boolean);
			discoveryHot = names;
			discoveryTopics = names;
		});
		return () => {
			cancelled = true;
		};
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
			const rawUsers = usersRes.code === 0 ? usersRes.data?.items : undefined;
			const rawMoments = momentsRes.code === 0 ? momentsRes.data?.items : undefined;
			const users = (rawUsers ?? []).map((u) => {
				const inner = u.user;
				return { user_id: inner.user_id, nickname: inner.nickname, avatar_url: inner.avatar_url, tags: u.tags };
			});
			const moments = rawMoments ?? [];
			// 话题从结果动态聚合
			const topicSet = new Set<string>();
			for (const m of moments) {
				m.tags?.forEach((t) => topicSet.add(t));
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
		class="rounded-panel border bg-surface-primary p-3"
		style="border-color:var(--border-default);"
		aria-label="搜索输入"
	>
		<div class="flex items-center gap-3">
			<SearchIcon class="size-5 shrink-0 text-accent-default" strokeWidth={1.8} />
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
				class="min-w-0 flex-1 bg-transparent py-2 text-[15px] text-text-primary outline-none placeholder:text-text-secondary"
			/>
			{#if query}
				<button
					type="button"
					onclick={() => (query = '')}
					aria-label="清除搜索"
					class="hover-btn flex size-8 items-center justify-center text-text-secondary hover:text-text-primary"
				>
					<X class="size-4" />
				</button>
			{/if}
		</div>
	</section>

	{#if !keyword}
		<!-- 无查询发现态（对齐 Figma：双栏 grid + 推荐话题跨列） -->
		<SearchDiscovery recent={recent} hot={discoveryHot} topics={discoveryTopics} onChoose={choose} onClear={() => (recent = [])} />
	{:else}
		<!-- 有查询结果态（对齐 Figma） -->
		<SearchResults
			{query}
			{tab}
			{tabs}
			{loading}
			{hasResults}
			{results}
			onTabChange={(id) => (tab = id)}
			onChoose={choose}
		/>
	{/if}
</StandardPageLayout>
