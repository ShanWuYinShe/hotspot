// 首页双段（关注 / 发现）的状态判定 —— 冷启动矩阵的唯一实现处，纯函数便于用例锁定。
//
// 依据（spec）：moments/fetch_following 依赖查看者关注关系且**无匿名版**；
// moments/search 热门模式承接发现页首屏、匿名可用。故：
//   - 未登录            → 默认「发现」（关注流对游客不可用）
//   - 已登录但零社交关系 → 默认「发现」（让首屏一定有内容；「关注」段仍可手动切换并给关注引导）
//   - 已登录且有关注     → 默认「关注」（关系轴优先）
// 判定用 UserInfo.following_count（已有字段，零额外请求、零额外接口）。

export type HomeFeedSegment = 'following' | 'discover';

export function resolveHomeSegment(
	isLoggedIn: boolean,
	followingCount: number | null | undefined
): HomeFeedSegment {
	if (!isLoggedIn) return 'discover';
	return (followingCount ?? 0) > 0 ? 'following' : 'discover';
}

export interface FollowingEmptyState {
	/** no-relations：还没关注任何人；no-activity：有关注，但关注的人没发新动态 */
	kind: 'no-relations' | 'no-activity';
	title: string;
	description: string;
	actionLabel: string;
}

/**
 * 「关注」段空态文案。
 * 关键点：接口两种情况都返回空数组，**只有 following_count 能把它们分开**——
 * 对零关系用户说「你关注的人还没发动态」是错的（他根本没关注人）。
 */
export function resolveFollowingEmpty(
	followingCount: number | null | undefined
): FollowingEmptyState {
	if ((followingCount ?? 0) <= 0) {
		return {
			kind: 'no-relations',
			title: '还没有关注任何人',
			description: '关注几个附近的人，他们的瞬间会出现在这里。',
			actionLabel: '去附近看看'
		};
	}
	return {
		kind: 'no-activity',
		title: '你关注的人还没发新瞬间',
		description: '过一会儿再来看看，或者先逛逛发现里的热门内容。',
		actionLabel: '去发现看看'
	};
}
