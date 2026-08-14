// 附近群组卡片数据类型与映射。
import type { GroupItem } from '@/api/types';
import { formatDistance } from '@/lib/utils';

export interface Group {
	id: string;
	name: string;
	banner: string;
	category: string;
	members: number;
	online: number;
	distance: string;
	privacy: 'public' | 'private';
	joined: boolean;
	desc: string;
	activity: string;
	avatars: string[];
}

/** Map a spec GroupItem (feed/nearby_groups) result to the card shape. */
export function mapGroup(item: GroupItem): Group {
	const g = item.group;
	return {
		id: g.group_id,
		name: g.name,
		banner: g.avatar_url ?? '',
		category: (item.tags?.[0] ?? '') || '同城',
		members: g.members_count,
		online: item.online_members_count ?? 0,
		distance: formatDistance(item.location?.distance),
		// join_method 在 GroupItem 顶层；invite_only/password 视为私密
		privacy: item.join_method === 'invite_only' || item.join_method === 'password' ? 'private' : 'public',
		joined: item.viewer_state !== null,
		desc: item.description ?? '',
		activity: item.next_activity ?? '',
		avatars: []
	};
}
