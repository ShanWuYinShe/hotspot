/**
 * 首页双段（关注 / 发现）的状态判定 —— 与 web `frontend/src/components/home/home-feed-mode.ts`
 * 同口径（含冷启动矩阵），纯函数便于对齐与复查。
 *
 * 依据（spec）：`moments/fetch_following` 依赖查看者关注关系且**无匿名版**；
 * `moments/search` 热门模式承接发现页首屏、匿名可用。故：
 *   - 未登录            → 默认「发现」
 *   - 已登录但零社交关系 → 默认「发现」（让首屏一定有内容；「关注」段仍可切换并给关注引导）
 *   - 已登录且有关注     → 默认「关注」
 * 关注数由 `users/fetch_following`（limit:1 足够分辨零与非零）探测；-1 表示尚未探测。
 */
export type HomeFeedSegment = 'following' | 'discover';

export const SEGMENT_UNKNOWN_COUNT: number = -1;

export function resolveHomeSegment(isLoggedIn: boolean, followingCount: number): HomeFeedSegment {
  if (!isLoggedIn) {
    return 'discover';
  }
  return followingCount > 0 ? 'following' : 'discover';
}

/**
 * 「关注」段空态文案。
 * 关键点：接口在「零关系」与「有关注但没人更新」两种情况下都返回空数组，
 * **只有关注计数能把它们分开**——对零关系用户说「你关注的人还没发动态」是错的。
 */
export interface FollowingEmptyCopy {
  /** true = 还没关注任何人（引导去关注）；false = 有关注但没更新（引导去发现） */
  noRelations: boolean;
  title: string;
  message: string;
  actionLabel: string;
}

export function resolveFollowingEmpty(followingCount: number): FollowingEmptyCopy {
  if (followingCount <= 0) {
    return {
      noRelations: true,
      title: '还没有关注任何人',
      message: '关注几个附近的人，他们的动态会出现在这里。',
      actionLabel: '去附近看看'
    };
  }
  return {
    noRelations: false,
    title: '你关注的人还没发新动态',
    message: '过一会儿再来看看，或者先逛逛发现里的热门内容。',
    actionLabel: '去发现看看'
  };
}
