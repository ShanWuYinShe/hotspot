// 附近群组卡片数据类型与映射（迁移自 React NearbyGroupsPage.tsx）。

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

/** Map a searchGroups result item to the card shape. */
export function mapGroup(item: any): Group {
	const g = item?.group ?? item ?? {};
	return {
		id: String(g.group_id ?? g.id ?? ''),
		name: g.name ?? '',
		banner: g.avatar_url ?? g.banner ?? g.cover_url ?? '',
		category: (item?.tags?.[0] ?? g.category ?? '') || '同城',
		members: g.members_count ?? g.members ?? 0,
		online: g.online_count ?? g.online ?? 0,
		distance: item?.distance ?? g.distance ?? '',
		privacy: g.privacy_level === 'private' ? 'private' : 'public',
		joined: item?.my_state?.is_joined ?? g.is_joined ?? false,
		desc: item?.description ?? g.description ?? g.bio ?? '',
		activity: item?.next_activity ?? g.activity ?? '',
		avatars: []
	};
}
