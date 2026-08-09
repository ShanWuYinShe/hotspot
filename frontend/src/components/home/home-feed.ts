import { getTrending, getNearbyUsers, getNearbyGroups } from '@/api';
import { formatDistance } from '@/lib/utils';
import type { MomentItem } from '@/api/types';
import type { NearbyUser } from './nearby-users';
import type { NearbyGroup } from './GroupCarousel.svelte';

// 首页的附近数据拉取：trending + 附近用户/群组。
// 统一 mapping 与失败兜底（任一接口失败返回空集，页面各自处理 loading/错误态）。
// trending 返回的 hot_moments 即 MomentItem[]，直接透传，不再做 Activity 包装
// （保留真实 moment_id 供跳转 /moments/[momentId] 使用）。

export interface HomeFeed {
	moments: MomentItem[];
	users: NearbyUser[];
	groups: NearbyGroup[];
}

export async function fetchHomeFeed(
	referencePoint?: { latitude: number; longitude: number } | null
): Promise<HomeFeed> {
	const searchParams: Record<string, unknown> = { radius: 5000, pagination: { limit: 12 } };
	if (referencePoint) searchParams.reference_point = referencePoint;

	const [trendingRes, nearbyUsersRes, nearbyGroupsRes] = await Promise.all([
		getTrending().catch(() => ({ code: -1, message: '网络异常', data: null })),
		getNearbyUsers(searchParams).catch(() => ({ code: -1, message: '网络异常', data: { items: [] } })),
		getNearbyGroups(searchParams).catch(() => ({ code: -1, message: '网络异常', data: { items: [] } }))
	]);

	const trendingData = trendingRes.code === 0 ? trendingRes.data : null;
	const usersData = nearbyUsersRes.code === 0 ? nearbyUsersRes.data : null;
	const groupsData = nearbyGroupsRes.code === 0 ? nearbyGroupsRes.data : null;

	return {
		moments: trendingData?.hot_moments ?? [],
		users: (usersData?.items ?? []).map((u) => ({
			id: u.user?.user_id ?? '',
			name: u.user?.nickname ?? '',
			distance: formatDistance(u.location?.distance),
			avatar: u.user?.avatar_url ?? '',
			isOnline: u.presence_status === 'online',
			tag: u.tags?.[0] ?? ''
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
