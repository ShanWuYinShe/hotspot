<script lang="ts">
	import { platform } from '@/stores/platform.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import { searchUsers, searchMoments, searchGroups, searchSuggest , netError } from '@/api';
	import SearchDiscovery, { type DiscoveryUser, type DiscoveryGroup } from './shared/SearchDiscovery.svelte';
	import SearchResults, { type TabType, type SearchResultsData } from './shared/SearchResults.svelte';
	import { smartBack } from '@/lib/smart-back';
	import { createDebouncedSearch } from '@/lib/debounced-search.svelte';
	import { goto } from '$app/navigation';
	import { ROUTES } from '@/lib/routes';
	import { formatDistance } from '@/lib/utils';
	import type { MomentItem } from '@/api/types';
	import { buildSearchUrl, parseSearchUrl } from '@/lib/search-url';

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

	// URL 状态（Round 8）：`?query=&tab=&tag=` 初始化恢复；输入提交/切 Tab/点 tag 用
	// history.replaceState 同步（不新增历史项）；popstate 前进/后退恢复。
	const initialUrlState =
		typeof window !== 'undefined'
			? parseSearchUrl(window.location.search)
			: { query: '', tab: 'all' as TabType, tag: null };
	let query = $state(initialUrlState.query);
	/** 非空 = tag 语义搜索（moments/search tag 参数）；输入/最近搜索路径复位为 null */
	let searchTag = $state<string | null>(initialUrlState.tag);
	let tab = $state<TabType>(initialUrlState.tab as TabType);
	/** popstate/URL 应用期间跳过 replaceState，避免循环 */
	let applyingFromUrl = false;
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
	// 同一热门响应的动态条目复用（此前只取 tags 即丢弃）：取前 4 条进「热门动态」区块
	let hotMoments = $state<MomentItem[]>([]);
	// 人物推荐：users/search 距离排序前 3（补发现态下半屏空白，P1-16）；字段映射
	// 与 home-feed 的 NearbyUser 同口径（tag/距离/在线态）。失败静默、无数据不渲染。
	let hotUsers = $state<DiscoveryUser[]>([]);
	// 推荐群组：groups/search 热门排序前 3，丰富发现态并平衡桌面端左右栏高度
	let hotGroups = $state<DiscoveryGroup[]>([]);
	// 发现态分区加载标志（R17 CLS 整改）：三个请求各自落定前，对应分区先渲染与最终内容
	// 等高的骨架，骨架常驻最终位置 → 数据到达只替换块内内容，不再把已渲染分区整段下移。
	// 归并为三态（moments 响应同时供「热门话题」+「热门动态」）；失败也在 finally 落定，
	// 不出现永久骨架；数据为空时分区按既有「诚实空态」不渲染。
	let momentsLoading = $state(true);
	let usersLoading = $state(true);
	let groupsLoading = $state(true);

	const device = $derived(platform.platform);

	// 全局 ⌘K / Ctrl+K 快捷键（监听在 (app)/+layout.svelte）：搜索页在输入框尾部提示该入口。
	const searchShortcut = $derived(
		typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform) ? '⌘K' : 'Ctrl K'
	);

	$effect(() => {
		try {
			localStorage.setItem('search_recent', JSON.stringify(recent));
		} catch {
			// Safari 无痕模式或配额超限时不崩溃
		}
	});

	const keyword = $derived(query.trim().toLowerCase());

	// URL 写入仅发生在「明确搜索动作」：提交(Enter)/选最近·热门/点 tag/切 Tab/Esc·清空。
	// 输入防抖与联想不写 URL（避免历史栏/地址栏随输入抖动）；replaceState 不新增历史项。
	function commitSearchUrl() {
		if (typeof window === 'undefined' || applyingFromUrl) return;
		const next = buildSearchUrl({ query: query.trim(), tab, tag: searchTag });
		const current = window.location.pathname + window.location.search;
		if (next !== current) window.history.replaceState(window.history.state, '', next);
	}

	// 初始归一化：非法 tab/冗余参数只修正地址栏一次（不新增历史、不触发搜索）
	if (typeof window !== 'undefined') {
		const canonical = buildSearchUrl({
			query: initialUrlState.query,
			tab: initialUrlState.tab as TabType,
			tag: initialUrlState.tag
		});
		if (canonical !== window.location.pathname + window.location.search) {
			window.history.replaceState(window.history.state, '', canonical);
		}
	}

	// 浏览器前进/后退（popstate）恢复搜索态；刷新/分享由上面 initialUrlState 承担。
	$effect(() => {
		if (typeof window === 'undefined') return;
		const onPopState = () => {
			const state = parseSearchUrl(window.location.search);
			applyingFromUrl = true;
			query = state.query;
			searchTag = state.tag;
			tab = state.tab as TabType;
			applyingFromUrl = false;
			// 不在此直接 doSearch：防抖搜索键含 query/tag，恢复后由统一防抖路径触发一次，
			// 避免 popstate 与防抖各发一轮请求（深链/后退均只触发一组搜索）。
		};
		window.addEventListener('popstate', onPopState);
		return () => window.removeEventListener('popstate', onPopState);
	});

	// 输入联想（search/suggest，spec prefix 1-50 必填、types=['tag']）：
	// 关键词非空时防抖取 tag 联想，作为结果页顶部的「相关话题」快速入口。
	// 空查询不调 suggest（spec 空串会 10000），发现态只保留最近搜索。
	createDebouncedSearch({
		query: () => keyword,
		fetch: (prefix, signal) => searchSuggest({ prefix, reference_point: null, types: ['tag'], radius: null }, { signal }),
		apply: (res) => {
			const suggest = res as unknown as { code: number; data?: { items?: Array<{ payload?: { name?: string } }> } | null };
			const items = suggest.code === 0 ? suggest.data?.items ?? [] : [];
			const names = items.map((i) => i.payload?.name ?? '').filter(Boolean) as string[];
			discoveryTopics = names;
		},
		clear: () => (discoveryTopics = [])
	});

	// 发现态热门话题：只拉一次（无查询时展示，有查询时不打扰结果态）
	$effect(() => {
		// 有查询时不拉发现态推荐（结果态不渲染这些区块；也避免深链触发两组 search 请求）；
		// 清空查询后自动重新拉取。
		void keyword;
		if (keyword) return;
		const controller = new AbortController();
		// 重新进入发现态（清空查询）时回到骨架态并清空旧数据：避免旧内容与新请求交错，
		// 也保证「清空查询」不会闪一帧上一轮的结果。
		hotTopics = [];
		hotMoments = [];
		hotUsers = [];
		hotGroups = [];
		momentsLoading = true;
		usersLoading = true;
		groupsLoading = true;
		void searchMoments({ sort_by: 'popular', pagination: { limit: 12 } }, { signal: controller.signal })
			.then((res) => {
				if (controller.signal.aborted || res.code !== 0 || !res.data) return;
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
				hotMoments = (res.data.items ?? []).slice(0, 4);
			})
			.catch(() => {})
			.finally(() => {
				if (!controller.signal.aborted) momentsLoading = false;
			});
		// 人物推荐：距离排序（不带 reference_point 走服务端默认位置，与首页无锚点
		// 回退同口径）；限 3 条（服务端对 limit 的裁剪口径不一，客户端显式
		// 切片兜底）；仅发现态消费，失败静默。
		void searchUsers({ radius: 5000, sort_by: 'distance', pagination: { limit: 3 } }, { signal: controller.signal })
			.then((res) => {
				if (controller.signal.aborted || res.code !== 0 || !res.data) return;
				hotUsers = (res.data.items ?? []).slice(0, 3).map((u) => ({
					id: u.user?.user_id ?? '',
					name: u.user?.nickname ?? '',
					avatar: u.user?.avatar_url ?? '',
					tag: u.tags?.[0] ?? '',
					distance: formatDistance(u.location?.distance),
					isOnline: u.presence_status === 'online'
				}));
			})
			.catch(() => {})
			.finally(() => {
				if (!controller.signal.aborted) usersLoading = false;
			});
		// 推荐群组：热门排序限 3 条，与人物推荐呼应
		void searchGroups({ sort_by: 'popular', pagination: { limit: 3 } }, { signal: controller.signal })
			.then((res) => {
				if (controller.signal.aborted || res.code !== 0 || !res.data) return;
				hotGroups = (res.data.items ?? []).slice(0, 3).map((g) => ({
					id: g.group?.group_id ?? '',
					name: g.group?.name ?? '',
					avatar: g.group?.avatar_url ?? '',
					category: g.tags?.[0] ?? '',
					memberCount: g.group?.members_count ?? 0
				}));
			})
			.catch(() => {})
			.finally(() => {
				if (!controller.signal.aborted) groupsLoading = false;
			});
		return () => {
			controller.abort();
		};
	});

	async function doSearch(term: string, signal?: AbortSignal) {
		const seq = ++searchSeq;
		searchError = null;
		if (!term) {
			results = { users: [], moments: [], topics: [], groups: [] };
			return;
		}
		loading = true;
		try {
			const [usersRes, momentsRes, groupsRes] = await Promise.all([
				searchUsers({ keyword: term }, { signal }).catch(() => netError({ items: [] })),
				searchMoments(searchTag ? { tag: searchTag } : { keyword: term }, { signal }).catch(() => netError({ items: [] })),
				searchGroups({ keyword: term }, { signal }).catch(() => netError({ items: [] }))
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

	// 结果搜索同样统一 300ms 防抖（SEARCH_DEBOUNCE_MS）；请求竞态由 doSearch 内部 seq 承载。
	// 行为微调：清空关键词由「防抖到期后清空结果」改为立即清空（输入框已空时旧结果不应滞留）。
	const SEARCH_KEY_SEP = '\u0000';
	createDebouncedSearch({
		// 键含 tag：同一关键词下 tag 语义切换（深链/后退恢复）也会重新搜索一次
		query: () => keyword + SEARCH_KEY_SEP + (searchTag ?? ''),
		fetch: (key, signal) => {
			const term = key.split(SEARCH_KEY_SEP)[0] ?? '';
			return doSearch(term, signal).then(() => undefined);
		},
		apply: () => {},
		clear: () => void doSearch('')
	});

	const remember = (value: string) => {
		recent = [value, ...recent.filter((r) => r !== value)].slice(0, 5);
	};
	const choose = (value: string) => {
		searchTag = null;
		query = value;
		remember(value);
		tab = 'all';
		commitSearchUrl();
	};

	// 热门话题点击：按 tag 语义搜索（moments/search 的 tag 参数精确匹配
	// moment_tags），而非把 tag 名当 description 关键词——只存在于标签上的
	// 话题（正文不含该词）用关键词搜必然 0 结果。手动输入/最近搜索仍走
	// 关键词路径（searchTag 在输入 onChange 与 choose 中复位）。
	const chooseTopic = (value: string) => {
		searchTag = value;
		query = value;
		remember(value);
		tab = 'all';
		commitSearchUrl();
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

	// 搜索键盘导航（Round 6，轻量不引入 combobox）：↑↓ 在输入框 / 结果行 / 最近·热门条目间移动焦点，
	// 焦点落在真实链接/按钮上（Enter 原生打开，Esc 由 SearchInput 清空）；Tab 顺序不受影响。
	function searchNavCandidates(): HTMLElement[] {
		const root = document.querySelector('[data-search-root]');
		if (!root) return [];
		return Array.from(
			root.querySelectorAll<HTMLElement>('a[href], button:not([disabled]), [data-cover]')
		).filter(
			(el) =>
				!el.closest('[role="tablist"], [data-tab-bar], .input-surface') &&
				el.offsetParent !== null &&
				!el.hasAttribute('aria-hidden')
		);
	}
	function handleSearchNavKeydown(e: KeyboardEvent) {
		// Esc：清空输入并回到输入框（搜索页范围内；弹层打开时交给弹层处理）
		if (e.key === 'Escape') {
			const root = document.querySelector('[data-search-root]');
			const active = document.activeElement as HTMLElement | null;
			if (!root || !active || !root.contains(active) || document.querySelector('[role="dialog"]')) return;
			const input = root.querySelector('input') as HTMLInputElement | null;
			if (!input) return;
			if (query || input.value) {
				e.preventDefault();
				query = '';
				searchTag = null;
				input.value = '';
				input.dispatchEvent(new Event('input', { bubbles: true }));
				commitSearchUrl();
			}
			input.focus();
			return;
		}
		if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp') return;
		const active = document.activeElement as HTMLElement | null;
		const root = document.querySelector('[data-search-root]');
		if (!root || !active || !root.contains(active)) return;
		// 弹层（图片查看/筛选等）打开时不劫持方向键
		if (document.querySelector('[role="dialog"]')) return;
		const nodes = searchNavCandidates();
		if (nodes.length === 0) return;
		e.preventDefault();
		const idx = nodes.indexOf(active);
		let next: number;
		if (idx < 0) next = e.key === 'ArrowDown' ? 0 : nodes.length - 1;
		else next = Math.min(nodes.length - 1, Math.max(0, idx + (e.key === 'ArrowDown' ? 1 : -1)));
		nodes[next]?.focus();
	}
</script>
<svelte:head>
	<title>搜索 · HotSpot</title>
</svelte:head>

<svelte:window onkeydown={handleSearchNavKeydown} />


<div class="flex flex-col flex-1 min-h-full" data-search-root>
<StandardPageLayout contentClassName="animate-fade-in pt-4 pb-6">
	{#snippet header()}
		<PageHeader title="搜索" glass onBack={() => smartBack()} titleSize={device === 'desktop' ? 'xl' : '2xl'}>
			{#snippet below()}
				<!-- 搜索框随 sticky header 吸顶（对齐 nearby/moments 样板 below 结构）。
				     below 容器仍全宽、与标题同轴；仅搜索输入本身在桌面限宽 max-w-3xl(768)——
				     满宽 1116px 会被读成「页头条带」而不是输入框（2026-10-29 审计 P0-1），
				     限宽后形成「标题 → 搜索入口 → 内容」三级节奏，移动端保持满宽。 -->
				<div class="mt-3 mb-2.5 w-full">
					<!-- 占位词与全站术语对齐:瞬间(非动态)并补群组维度(Gemini 审查 r11) -->
					<SearchInput
						value={query}
						onChange={(v) => {
							searchTag = null;
							query = v;
							if (!v.trim()) commitSearchUrl();
						}}
						placeholder="搜索用户、瞬间、群组、地点或话题"
						ariaLabel="搜索用户、瞬间、群组、地点或话题"
						autofocus
						class={device === 'desktop' ? 'max-w-3xl' : ''}
						onEnter={(v) => {
							if (!v.trim()) return;
							remember(v.trim());
							commitSearchUrl();
						}}
					>
						{#snippet trailing()}
							{#if device === 'desktop'}
								<kbd
									class="flex shrink-0 items-center rounded-sm border border-border-default bg-surface-subtle px-1.5 py-0.5 text-xs font-medium text-text-secondary"
									aria-hidden="true"
								>{searchShortcut}</kbd>
							{/if}
						{/snippet}
					</SearchInput>
				</div>
			{/snippet}
		</PageHeader>
	{/snippet}

	{#if searchError}
		<LoadError message={searchError} onRetry={() => void doSearch(keyword)} />
	{:else if !keyword}
		<!-- 无查询发现态：最近搜索 + 热门话题 + 人物推荐 + 热门动态（无数据时各区块内部不渲染） -->
		<SearchDiscovery
			recent={recent}
			hotTopics={hotTopics}
			hotMoments={hotMoments}
			hotUsers={hotUsers}
			hotGroups={hotGroups}
			topicsLoading={momentsLoading}
			{momentsLoading}
			{usersLoading}
			{groupsLoading}
			onChoose={choose}
			onChooseTopic={chooseTopic}
			onOpenMoment={(id) => void goto(ROUTES.moment(id))}
			onOpenUser={(id) => void goto(ROUTES.user(id))}
			onOpenGroup={(id) => void goto(ROUTES.group(id))}
			onSeeAllUsers={() => void goto(ROUTES.nearbyPeople)}
			onSeeAllGroups={() => void goto(ROUTES.nearbyGroups)}
			onClear={() => (recent = [])}
		/>
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
			hotTopics={hotTopics}
			onChooseTopic={chooseTopic}
			onTabChange={(id) => {
				tab = id;
				commitSearchUrl();
			}}
			onChoose={choose}
		/>
	{/if}
</StandardPageLayout>
</div>
