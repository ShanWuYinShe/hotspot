// 附近群组卡片数据类型与映射（迁移自旧版 NearbyGroupsPage）。

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
export function mapGroup(item: any): Group {
	const g = item?.group ?? {};
	return {
		id: String(g.group_id ?? ''),
		name: g.name ?? '',
		banner: g.avatar_url ?? '',
		category: (item?.tags?.[0] ?? '') || '同城',
		members: g.members_count ?? 0,
		// spec GroupItem 无在线/隐私/活动/成员头像字段，卡片展示用默认值
		online: 0,
		distance: item?.location?.distance ?? '',
		privacy: 'public',
		joined: item?.my_state?.is_joined ?? false,
		desc: item?.description ?? '',
		activity: '',
		avatars: []
	};
}
