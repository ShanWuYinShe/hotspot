<script lang="ts">
	import { platform } from '@/stores/platform.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import { searchUsers, searchMoments, searchSuggest } from '@/api';
	import SearchDiscovery from './shared/SearchDiscovery.svelte';
	import SearchResults, { type TabType, type SearchResultsData } from './shared/SearchResults.svelte';

	// 搜索页：最近搜索（localStorage）、热门搜索、推荐话题、全部/用户/动态/话题 Tab。
	// 对齐 Figma Make SearchPage：卡片使用 surface-primary + border 的 token 类，
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
	let searchSeq = 0;
	let recent = $state<string[]>(readRecent());
	let discoveryHot = $state<string[]>([]);
	let discoveryTopics = $state<string[]>([]);
	let results = $state<SearchResultsData>({
		users: [],
		moments: [],
		topics: []
	});
	let loading = $state(false);
	let searchError = $state<string | null>(null);

	const device = $derived(platform.platform);

	$effect(() => {
		localStorage.setItem('search_recent', JSON.stringify(recent));
	});

	// 发现态的热门搜索/推荐话题来自 search/suggest（spec SearchSuggestRequest prefix 1-50 必填），
	// 空查询不调 suggest（spec 空串会 10000），沿用本地 recent/空面板；有输入时仅取 tag 类型联想，
	// 避免混合类型下 `payload.name` 对 user/moment 字段丢失（user→nickname、moment→description_preview）。
	$effect(() => {
		const prefix = keyword;
		if (!prefix) {
			discoveryHot = [];
			discoveryTopics = [];
			return;
		}
		let cancelled = false;
		const timer = setTimeout(() => {
			void searchSuggest({ prefix, reference_point: null, types: ['tag'], radius: null }).then((res) => {
				if (cancelled) return;
				const suggest = res as unknown as { code: number; data?: { items?: Array<{ payload?: { name?: string } }> } | null };
				const items = suggest.code === 0 ? suggest.data?.items ?? [] : [];
				const names = items.map((i) => i.payload?.name ?? '').filter(Boolean) as string[];
				discoveryHot = names;
				discoveryTopics = names;
			});
		}, 300);
		return () => {
			cancelled = true;
			clearTimeout(timer);
		};
	});

	const keyword = $derived(query.trim().toLowerCase());

	async function doSearch(term: string) {
		const seq = ++searchSeq;
		searchError = null;
		if (!term) {
			results = { users: [], moments: [], topics: [] };
			return;
		}
		loading = true;
		try {
			const [usersRes, momentsRes] = await Promise.all([
				searchUsers({ keyword: term }).catch(() => ({ code: -1, message: '网络异常', data: { items: [] } })),
				searchMoments({ keyword: term }).catch(() => ({ code: -1, message: '网络异常', data: { items: [] } }))
			]);
			const rawUsers = usersRes.code === 0 ? usersRes.data?.items : undefined;
			const rawMoments = momentsRes.code === 0 ? momentsRes.data?.items : undefined;
			if (usersRes.code !== 0 || momentsRes.code !== 0) {
				if (seq === searchSeq) {
					searchError = usersRes.message ?? momentsRes.message ?? '搜索失败，请稍后重试';
					loading = false;
				}
				return;
			}
			const users = (rawUsers ?? []).map((u) => {
				// spec: users/search 返回 UserItem（user 嵌套）；兼容扁平结构兜底，避免搜索崩溃
				const inner = u.user ?? (u as unknown as { user_id?: string; nickname?: string; avatar_url?: string });
				return {
					user_id: inner.user_id ?? '',
					nickname: inner.nickname ?? '',
					avatar_url: inner.avatar_url ?? '',
					tags: (u as { tags?: string[] }).tags ?? []
				};
			});
			const moments = rawMoments ?? [];
			// 话题从结果动态聚合
			const topicSet = new Set<string>();
			for (const m of moments) {
				m.tags?.forEach((t) => topicSet.add(t));
			}
			if (seq !== searchSeq) return; // 旧请求晚返回时丢弃，避免覆盖新关键词结果
			results = { users, moments, topics: Array.from(topicSet) };
		} catch {
			if (seq !== searchSeq) return;
			searchError = '搜索失败，请稍后重试';
			results = { users: [], moments: [], topics: [] };
		} finally {
			if (seq === searchSeq) loading = false;
		}
	}

	// 300ms 防抖
	$effect(() => {
		const term = keyword;
		const timer = setTimeout(() => void doSearch(term), 300);
		return () => clearTimeout(timer);
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
<svelte:head>
	<title>搜索 · HotSpot</title>
</svelte:head>


<StandardPageLayout contentClassName="animate-fade-in pt-4 pb-6">
	{#snippet header()}
		<PageHeader title="搜索" onBack={() => history.back()} titleSize={device === 'desktop' ? 20 : device === 'tablet' ? 24 : 22}>
			{#snippet below()}
				<!-- 搜索框随 sticky header 吸顶（对齐 nearby/moments 样板 below 结构） -->
				<div class="mt-3 mb-2.5">
					<SearchInput
						value={query}
						onChange={(v) => (query = v)}
						placeholder="搜索用户、动态、地点或话题"
						autofocus
						onEnter={(v) => v.trim() && remember(v.trim())}
						class="px-4 py-2.5 rounded-md"
					/>
				</div>
			{/snippet}
		</PageHeader>
	{/snippet}

	{#if searchError}
		<LoadError message={searchError} onRetry={() => void doSearch(keyword)} />
	{:else if !keyword}
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
