import { createDebouncedSearch } from '@/lib/debounced-search.svelte';
import { searchUsers, searchMoments, searchGroups, searchSuggest, netError } from '@/api';
import { formatDistance } from '@/lib/utils';
import type { MomentItem } from '@/api/types';
import { buildSearchUrl, parseSearchUrl } from '@/lib/search-url';
import { isSearchNavKey, nextSearchNavIndex } from '@/lib/search-nav';
import { resolveDiscoveryErrorView } from '@/lib/search-discovery-error';
import type { DiscoveryUser, DiscoveryGroup } from './SearchDiscovery.svelte';
import type { TabType, SearchResultsData } from './SearchResults.svelte';

function readRecent(): string[] {
	try {
		return JSON.parse(localStorage.getItem('search_recent') ?? '[]');
	} catch {
		return [];
	}
}

export function useSearchPageData() {
	const initialUrlState =
		typeof window !== 'undefined'
			? parseSearchUrl(window.location.search)
			: { query: '', tab: 'all' as TabType, tag: null };

	let query = $state(initialUrlState.query);
	let searchTag = $state<string | null>(initialUrlState.tag);
	let tab = $state<TabType>(initialUrlState.tab as TabType);
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

	let hotTopics = $state<string[]>([]);
	let hotMoments = $state<MomentItem[]>([]);
	let hotUsers = $state<DiscoveryUser[]>([]);
	let hotGroups = $state<DiscoveryGroup[]>([]);

	let momentsLoading = $state(true);
	let usersLoading = $state(true);
	let groupsLoading = $state(true);

	let momentsFailed = $state(false);
	let usersFailed = $state(false);
	let groupsFailed = $state(false);
	let discoveryReload = $state(0);

	const keyword = $derived(query.trim().toLowerCase());

	// 持久化最近搜索
	$effect(() => {
		try {
			localStorage.setItem('search_recent', JSON.stringify(recent));
		} catch {
			// Safari 无痕模式或配额超限时不崩溃
		}
	});

	function commitSearchUrl() {
		if (typeof window === 'undefined' || applyingFromUrl) return;
		const next = buildSearchUrl({ query: query.trim(), tab, tag: searchTag });
		const current = window.location.pathname + window.location.search;
		if (next !== current) window.history.replaceState(window.history.state, '', next);
	}

	function clearSearch() {
		query = '';
		searchTag = null;
		commitSearchUrl();
		const input = document.querySelector<HTMLInputElement>('[data-search-root] header input');
		if (!input) return;
		input.value = '';
		input.dispatchEvent(new Event('input', { bubbles: true }));
		input.focus();
	}

	// 初始归一化
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

	// popstate
	$effect(() => {
		if (typeof window === 'undefined') return;
		const onPopState = () => {
			const state = parseSearchUrl(window.location.search);
			applyingFromUrl = true;
			query = state.query;
			searchTag = state.tag;
			tab = state.tab as TabType;
			applyingFromUrl = false;
		};
		window.addEventListener('popstate', onPopState);
		return () => window.removeEventListener('popstate', onPopState);
	});

	// 输入联想
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

	// 发现态热门推荐
	$effect(() => {
		void keyword;
		void discoveryReload;
		if (keyword) return;
		const controller = new AbortController();

		hotTopics = [];
		hotMoments = [];
		hotUsers = [];
		hotGroups = [];
		momentsLoading = true;
		usersLoading = true;
		groupsLoading = true;
		momentsFailed = false;
		usersFailed = false;
		groupsFailed = false;

		void searchMoments({ sort_by: 'popular', pagination: { limit: 12 } }, { signal: controller.signal })
			.then((res) => {
				if (controller.signal.aborted) return;
				if (res.code !== 0 || !res.data) {
					momentsFailed = true;
					return;
				}
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
			.catch(() => {
				if (!controller.signal.aborted) momentsFailed = true;
			})
			.finally(() => {
				if (!controller.signal.aborted) momentsLoading = false;
			});

		void searchUsers({ radius: 5000, sort_by: 'distance', pagination: { limit: 3 } }, { signal: controller.signal })
			.then((res) => {
				if (controller.signal.aborted) return;
				if (res.code !== 0 || !res.data) {
					usersFailed = true;
					return;
				}
				hotUsers = (res.data.items ?? []).slice(0, 3).map((u) => ({
					id: u.user?.user_id ?? '',
					name: u.user?.nickname ?? '',
					avatar: u.user?.avatar_url ?? '',
					tag: u.tags?.[0] ?? '',
					distance: formatDistance(u.location?.distance),
					isOnline: u.presence_status === 'online'
				}));
			})
			.catch(() => {
				if (!controller.signal.aborted) usersFailed = true;
			})
			.finally(() => {
				if (!controller.signal.aborted) usersLoading = false;
			});

		void searchGroups({ radius: 5000, sort_by: 'popular', pagination: { limit: 3 } }, { signal: controller.signal })
			.then((res) => {
				if (controller.signal.aborted) return;
				if (res.code !== 0 || !res.data) {
					groupsFailed = true;
					return;
				}
				hotGroups = (res.data.items ?? []).slice(0, 3).map((g) => ({
					id: g.group?.group_id ?? '',
					name: g.group?.name ?? '',
					avatar: g.group?.avatar_url ?? '',
					category: g.tags?.[0] ?? '',
					memberCount: g.group?.members_count ?? 0
				}));
			})
			.catch(() => {
				if (!controller.signal.aborted) groupsFailed = true;
			})
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

			const topicSet = new Set<string>();
			for (const m of moments) {
				m.tags?.forEach((t) => topicSet.add(t));
			}
			if (seq !== searchSeq) return;
			results = { users, moments, topics: Array.from(topicSet), groups };
		} catch {
			if (seq !== searchSeq) return;
			searchError = '搜索失败，请稍后重试';
			results = { users: [], moments: [], topics: [], groups: [] };
		} finally {
			if (seq === searchSeq) loading = false;
		}
	}

	const SEARCH_KEY_SEP = '\u0000';
	createDebouncedSearch({
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

	const chooseTopic = (value: string) => {
		searchTag = value;
		query = value;
		remember(value);
		tab = 'all';
		commitSearchUrl();
	};

	const retryDiscovery = () => {
		discoveryReload += 1;
	};

	const resultsTotal = $derived(results.users.length + results.moments.length + results.topics.length + results.groups.length);
	const hasResults = $derived(resultsTotal > 0);

	const islandMeta = $derived.by(() => {
		const term = query.trim();
		if (!term) return '';
		if (searchError) return `${term} · 搜索失败`;
		if (loading) return `${term} · 搜索中…`;
		if (!hasResults) return `${term} · 无匹配结果`;
		return `${term} · ${resultsTotal} 个结果`;
	});

	const tabs = $derived<{ id: TabType; label: string; count?: number }[]>([
		{ id: 'all', label: '全部' },
		{ id: 'people', label: '用户', count: loading ? undefined : results.users.length },
		{ id: 'posts', label: '动态', count: loading ? undefined : results.moments.length },
		{ id: 'groups', label: '群组', count: loading ? undefined : results.groups.length },
		{ id: 'topics', label: '话题', count: loading ? undefined : results.topics.length }
	]);

	const discoveryAllFailed = $derived(momentsFailed && usersFailed && groupsFailed);
	const discoveryErrorView = $derived(
		resolveDiscoveryErrorView({ moments: momentsFailed, users: usersFailed, groups: groupsFailed }, recent.length > 0)
	);

	function searchNavCandidates(): HTMLElement[] {
		const root = document.querySelector('[data-search-root]');
		if (!root) return [];
		return Array.from(
			root.querySelectorAll<HTMLElement>('a[href], button:not([disabled]), [data-cover]')
		).filter(
			(el) =>
				!el.closest('header') &&
				!el.closest('[role="tablist"], [data-tab-bar], .input-surface') &&
				el.offsetParent !== null &&
				!el.hasAttribute('aria-hidden')
		);
	}

	function handleSearchNavKeydown(e: KeyboardEvent) {
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
		if (!isSearchNavKey(e.key)) return;
		const active = document.activeElement as HTMLElement | null;
		const root = document.querySelector('[data-search-root]');
		if (!root || !active || !root.contains(active)) return;
		if (document.querySelector('[role="dialog"]')) return;
		const nodes = searchNavCandidates();
		const next = nextSearchNavIndex(nodes.indexOf(active), nodes.length, e.key);
		if (next === null) return;
		e.preventDefault();
		nodes[next]?.focus();
	}

	return {
		get query() { return query; },
		set query(v: string) { query = v; },
		get searchTag() { return searchTag; },
		set searchTag(v: string | null) { searchTag = v; },
		get tab() { return tab; },
		set tab(v: TabType) { tab = v; },
		get recent() { return recent; },
		set recent(v: string[]) { recent = v; },
		get discoveryTopics() { return discoveryTopics; },
		get results() { return results; },
		get loading() { return loading; },
		get searchError() { return searchError; },
		get hotTopics() { return hotTopics; },
		get hotMoments() { return hotMoments; },
		get hotUsers() { return hotUsers; },
		get hotGroups() { return hotGroups; },
		get momentsLoading() { return momentsLoading; },
		get usersLoading() { return usersLoading; },
		get groupsLoading() { return groupsLoading; },
		get discoveryAllFailed() { return discoveryAllFailed; },
		get discoveryErrorView() { return discoveryErrorView; },
		get keyword() { return keyword; },
		get resultsTotal() { return resultsTotal; },
		get hasResults() { return hasResults; },
		get islandMeta() { return islandMeta; },
		get tabs() { return tabs; },
		clearSearch,
		commitSearchUrl,
		remember,
		choose,
		chooseTopic,
		retryDiscovery,
		doSearch,
		handleSearchNavKeydown
	};
}

export type SearchPageData = ReturnType<typeof useSearchPageData>;
