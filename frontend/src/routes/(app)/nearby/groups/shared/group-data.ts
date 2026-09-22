// 附近群组卡片数据类型与映射。
import type { GroupItem } from '@/api/types';
import { formatDistance } from '@/lib/utils';

export interface Group {
	id: string;
	name: string;
	banner: string;
	/** 展示用首标签类目；筛选走全量 tags（首标签之外同样可命中） */
	category: string;
	tags: string[];
	members: number;
	online: number;
	distance: string;
	privacy: 'public' | 'private';
	joined: boolean;
	desc: string;
	activity: string;
	avatars: string[];
}

/** 将 groups/search（radius + sort_by='distance'）返回的 GroupItem 映射为卡片数据（feed 域已废除，附近群组发现由 search 承担）。 */
export function mapGroup(item: GroupItem): Group {
	const g = item.group;
	return {
		id: g.group_id,
		name: g.name,
		banner: g.avatar_url ?? '',
		category: (item.tags?.[0] ?? '') || '同城',
		tags: item.tags ?? [],
		members: g.members_count,
		online: item.online_members_count ?? 0,
		distance: formatDistance(item.location?.distance),
		// join_method 在 GroupItem 顶层；invite_only/password 视为私密
		privacy: item.join_method === 'invite_only' || item.join_method === 'password' ? 'private' : 'public',
		joined: item.viewer_state !== null,
		desc: item.description ?? '',
		activity: item.next_activity_at ?? '',
		avatars: []
	};
}
