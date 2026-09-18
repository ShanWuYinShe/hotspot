/**
 * 关系聚合页（/me/relationships）的 tab 契约——单一事实源。
 *
 * 好友 / 关注 / 粉丝三种关系原先各有一条独立路由（/me/friends 等）与一份页头，
 * 与聚合页的三个 tab 内容完全重复且无入站入口。现收敛为「唯一 canonical 路径 +
 * ?tab= 深链」，旧路径由重定向保留可达性（见 routes/(app)/me/{friends,following,fans}/+page.ts），
 * 所有入口统一经 relationshipsHref() 构造 URL，避免各处手写查询串漂移。
 *
 * blocked（黑名单）无历史路由：数据来自 users/fetch_blocked（命名空间天生私有），
 * 作为聚合页第四个 tab 承载（解除拉黑走 set_block active=false）。
 */

import { ME_SECTION_PATHS } from '@/lib/routes';

export const RELATIONSHIP_TABS = ['friends', 'following', 'fans', 'blocked'] as const;

export type RelationshipTab = (typeof RELATIONSHIP_TABS)[number];

export const DEFAULT_RELATIONSHIP_TAB: RelationshipTab = 'friends';

/** 运行时窄化 ?tab= 查询参数，非法值回落到默认 tab。 */
export function isRelationshipTab(value: string | null | undefined): value is RelationshipTab {
	return typeof value === 'string' && (RELATIONSHIP_TABS as readonly string[]).includes(value);
}

/** 关系聚合页深链（带 tab 查询参数）。 */
export function relationshipsHref(tab: RelationshipTab = DEFAULT_RELATIONSHIP_TAB): string {
	return `${ME_SECTION_PATHS.relationships}?tab=${tab}`;
}
