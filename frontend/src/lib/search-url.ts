// 搜索页 URL 状态（Round 8）：`?query=&tab=&tag=` 的解析/序列化。
// 约束：query 为空时不携带 query/tag（回发现态）；非法 tab 回退 all；
// 输入提交/切 Tab/点 tag 用 history.replaceState 同步（不新增历史项）。
export type SearchTab = 'all' | 'people' | 'posts' | 'groups' | 'topics';

export interface SearchUrlState {
	query: string;
	tab: SearchTab;
	tag: string | null;
}

const TABS: SearchTab[] = ['all', 'people', 'posts', 'groups', 'topics'];
// 兼容外部分享的语义别名：users→people、moments→posts、群组→groups
const TAB_ALIASES: Record<string, SearchTab> = {
	users: 'people',
	people: 'people',
	moments: 'posts',
	posts: 'posts',
	groups: 'groups',
	topics: 'topics',
	all: 'all'
};

export function parseSearchUrl(search: string): SearchUrlState {
	let params: URLSearchParams;
	try {
		params = new URLSearchParams(search.startsWith('?') ? search : `?${search}`);
	} catch {
		return { query: '', tab: 'all', tag: null };
	}
	const query = (params.get('query') ?? '').trim();
	const rawTab = (params.get('tab') ?? '').trim().toLowerCase();
	const tab = TAB_ALIASES[rawTab] ?? 'all';
	const rawTag = (params.get('tag') ?? '').trim();
	// 空 query 不请求、也不保留 tag（tag 语义依赖 query 展示）
	return { query, tab: TABS.includes(tab) ? tab : 'all', tag: query && rawTag ? rawTag : null };
}

/** 生成同步用的相对 URL（pathname 由调用方给）；无 query 时等价于清空参数。 */
export function buildSearchUrl(state: SearchUrlState, pathname = '/search'): string {
	const params = new URLSearchParams();
	if (state.query) {
		params.set('query', state.query);
		if (state.tag) params.set('tag', state.tag);
		if (state.tab !== 'all') params.set('tab', state.tab);
	}
	const qs = params.toString();
	return qs ? `${pathname}?${qs}` : pathname;
}
