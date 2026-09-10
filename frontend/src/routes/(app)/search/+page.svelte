<script lang="ts">
	import { platform } from '@/stores/platform.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import { searchUsers, searchMoments, searchGroups, searchSuggest , netError } from '@/api';
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
	let discoveryTopics = $state<string[]>([]);
	let results = $state<SearchResultsData>({
		users: [],
		moments: [],
		topics: [],
		groups: []
	});
	let loading = $state(false);
	let searchError = $state<string | null>(null);
	// 发现态热门话题：moments/search 热门模式取热门动态 tags 聚合（与 home-feed 同源）；
	// 无数据时不渲染区块（诚实空态原则），失败静默。
	let hotTopics = $state<string[]>([]);

	const device = $derived(platform.platform);

	$effect(() => {
		localStorage.setItem('search_recent', JSON.stringify(recent));
	});

	const keyword = $derived(query.trim().toLowerCase());

	// 输入联想（search/suggest，spec prefix 1-50 必填、types=['tag']）：
	// 关键词非空时防抖取 tag 联想，作为结果页顶部的「相关话题」快速入口。
	// 空查询不调 suggest（spec 空串会 10000），发现态只保留最近搜索。
	$effect(() => {
		const prefix = keyword;
		if (!prefix) {
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
				discoveryTopics = names;
			});
		}, 300);
		return () => {
			cancelled = true;
			clearTimeout(timer);
		};
	});

	// 发现态热门话题：只拉一次（无查询时展示，有查询时不打扰结果态）
	$effect(() => {
		let cancelled = false;
		void searchMoments({ sort_by: 'popular', pagination: { limit: 12 } })
			.then((res) => {
				if (cancelled || res.code !== 0 || !res.data) return;
				const seen = new Set<string>();
				const list: string[] = [];
				for (const m of res.data.items ?? []) {
					for (const t of m.tags ?? []) {
						if (t && !seen.has(t)) {
							seen.add(t);
							list.push(t);
						}
					}
				}
				hotTopics = list.slice(0, 10);
			})
			.catch(() => {});
		return () => {
			cancelled = true;
		};
	});

	async function doSearch(term: string) {
		const seq = ++searchSeq;
		searchError = null;
		if (!term) {
			results = { users: [], moments: [], topics: [], groups: [] };
			return;
		}
		loading = true;
		try {
			const [usersRes, momentsRes, groupsRes] = await Promise.all([
				searchUsers({ keyword: term }).catch(() => netError({ items: [] })),
				searchMoments({ keyword: term }).catch(() => netError({ items: [] })),
				searchGroups({ keyword: term }).catch(() => netError({ items: [] }))
			]);
			const rawUsers = usersRes.code === 0 ? usersRes.data?.items : undefined;
			const rawMoments = momentsRes.code === 0 ? momentsRes.data?.items : undefined;
			const rawGroups = groupsRes.code === 0 ? groupsRes.data?.items : undefined;
			if (usersRes.code !== 0 || momentsRes.code !== 0 || groupsRes.code !== 0) {
				if (seq === searchSeq) {
					searchError = usersRes.message ?? momentsRes.message ?? groupsRes.message ?? '搜索失败，请稍后重试';
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
			const groups = (rawGroups ?? []).map((g) => ({
				group_id: g.group?.group_id ?? '',
				name: g.group?.name ?? '',
				avatar_url: g.group?.avatar_url ?? '',
				members_count: g.group?.members_count ?? 0
			}));
			// 话题从结果动态聚合
			const topicSet = new Set<string>();
			for (const m of moments) {
				m.tags?.forEach((t) => topicSet.add(t));
			}
			if (seq !== searchSeq) return; // 旧请求晚返回时丢弃，避免覆盖新关键词结果
			results = { users, moments, topics: Array.from(topicSet), groups };
		} catch {
			if (seq !== searchSeq) return;
			searchError = '搜索失败，请稍后重试';
			results = { users: [], moments: [], topics: [], groups: [] };
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

	const hasResults = $derived(results.users.length + results.moments.length + results.topics.length + results.groups.length > 0);

	// 计数在「搜索中」时不显示（undefined），搜索完成后即使是 0 也显示——
	// 否则 0 结果与加载中都是「无计数」，两者不可区分（历史审查遗留项）。
	const tabs = $derived<{ id: TabType; label: string; count?: number }[]>([
		{ id: 'all', label: '全部' },
		{ id: 'people', label: '用户', count: loading ? undefined : results.users.length },
		{ id: 'posts', label: '动态', count: loading ? undefined : results.moments.length },
		{ id: 'groups', label: '群组', count: loading ? undefined : results.groups.length },
		{ id: 'topics', label: '话题', count: loading ? undefined : results.topics.length }
	]);
</script>
<svelte:head>
	<title>搜索 · HotSpot</title>
</svelte:head>


<StandardPageLayout contentClassName="animate-fade-in pt-4 pb-6">
	{#snippet header()}
		<PageHeader title="搜索" onBack={() => history.back()} titleSize={device === 'desktop' ? 'xl' : '2xl'}>
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
		<!-- 无查询发现态：最近搜索 + 热门话题（无数据时 SearchDiscovery 内部不渲染该区块） -->
		<SearchDiscovery recent={recent} hotTopics={hotTopics} onChoose={choose} onClear={() => (recent = [])} />
	{:else}
		<!-- 有查询结果态（对齐 Figma）；输入联想（相关话题）作为顶部快速入口 -->
		<SearchResults
			{query}
			{tab}
			{tabs}
			{loading}
			{hasResults}
			{results}
			suggestions={discoveryTopics.filter((t) => t.toLowerCase() !== keyword)}
			onTabChange={(id) => (tab = id)}
			onChoose={choose}
		/>
	{/if}
</StandardPageLayout>
