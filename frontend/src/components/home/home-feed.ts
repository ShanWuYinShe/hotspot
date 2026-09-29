import { fetchNearbyMoments, searchUsers, searchGroups , netError } from '@/api';
import { formatDistance } from '@/lib/utils';
import type { MomentItem } from '@/api/types';
import type { NearbyUser } from './nearby-users';
import type { NearbyGroup } from './GroupCarousel.svelte';

// 首页的附近数据拉取：附近瞬间 + 附近用户/群组（三者统一按 5km 半径过滤）。
// 请求异常统一转 netError；全部接口失败时抛 FeedError 让页面走 LoadError
// （不再伪装成「附近还没有内容」空态——与 LoadError 契约一致），部分失败
// 保留成功侧数据。
// 瞬间走 moments/fetch_nearby（与附近专区同源；此前走全局 sort_by='popular'，
// 首页标榜 5km 而内容全球，导致与附近专区计数对不上——线上实测）。
// MomentItem 直接透传，不再做 Activity 包装（保留真实 moment_id 供跳转
// /moments/[momentId] 使用）。

export interface HomeFeed {
	moments: MomentItem[];
	users: NearbyUser[];
	groups: NearbyGroup[];
}

/** 三个数据源全部失败时抛出（页面 catch 后渲染 LoadError）。 */
export class FeedError extends Error {}

export async function fetchHomeFeed(
	referencePoint?: { latitude: number; longitude: number } | null
): Promise<HomeFeed> {
	const point = referencePoint ? { reference_point: referencePoint } : {};

	const [hotRes, nearbyUsersRes, nearbyGroupsRes] = await Promise.all([
		fetchNearbyMoments({ radius: 5000, ...point, pagination: { limit: 12 } }).catch(() => netError(null)),
		searchUsers({ radius: 5000, sort_by: 'distance', ...point, pagination: { limit: 12 } }).catch(() => netError({ items: [] })),
		searchGroups({ radius: 5000, sort_by: 'distance', ...point, pagination: { limit: 12 } }).catch(() => netError({ items: [] }))
	]);

	const hotData = hotRes.code === 0 ? hotRes.data : null;
	const usersData = nearbyUsersRes.code === 0 ? nearbyUsersRes.data : null;
	const groupsData = nearbyGroupsRes.code === 0 ? nearbyGroupsRes.data : null;

	if (!hotData && !usersData && !groupsData) {
		throw new FeedError(hotRes.message || nearbyUsersRes.message || '加载失败');
	}

	return {
		moments: hotData?.items ?? [],
		users: (usersData?.items ?? []).map((u) => ({
			id: u.user?.user_id ?? '',
			name: u.user?.nickname ?? '',
			distance: formatDistance(u.location?.distance),
			avatar: u.user?.avatar_url ?? '',
			isOnline: u.presence_status === 'online',
			tag: u.tags?.[0] ?? '',
			isFollowing: u.relation?.is_following ?? false
		})),
		groups: (groupsData?.items ?? []).map((g) => ({
			id: g.group?.group_id ?? '',
			name: g.group?.name ?? '',
			members: g.group?.members_count ?? 0,
			banner: g.group?.avatar_url ?? '',
			distance: formatDistance(g.location?.distance),
			privacy: g.join_method === 'invite_only' || g.join_method === 'password' ? 'private' : 'public'
		}))
	};
}
