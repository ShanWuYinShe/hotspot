// 个人中心数据工厂。
// 组件内调用创建独立响应式状态；数据映射函数（mapGroup/mapFriend 等）原样保留。

import {
	getAccount,
	listUserMoments,
	listUserReactedMoments,
	listMyMoments,
	getFavoriteMoments,
	listMyGroups,
	listFriends,
	listFollowing,
	listFollowers
} from '@/api';
import type {
	AccountData,
	FollowerItem,
	FollowingItem,
	FriendItem,
	GroupItem,
	MomentItem
} from '@/api/types';

export interface GalleryMoment {
	id: string;
	img: string;
	title: string;
	likes: number;
	comments: number;
}

export interface ProfileStats {
	moments_count: number;
	friends_count: number;
	followers_count: number;
	following_count: number;
	groups_count: number;
}

/** Map a spec MomentItem to the flat GalleryMoment shape the gallery tiles consume. */
function mapGalleryMoment(m: MomentItem): GalleryMoment {
	return {
		id: m.moment_id,
		img: m.images?.[0]?.url ?? '',
		title: m.description ?? '',
		likes: m.likes_count ?? 0,
		comments: m.comments_count ?? 0
	};
}

export function useAccountData() {
	let avatar = $state('');
	let nickname = $state('');
	let bio = $state('');
	let locationName = $state('');
	let handle = $state('');
	let stats = $state<ProfileStats>({
		moments_count: 0,
		friends_count: 0,
		followers_count: 0,
		following_count: 0,
		groups_count: 0
	});
	let loading = $state(true);

	$effect(() => {
		getAccount()
			.then((res) => {
				if (res.code === 0 && res.data) {
					const d: AccountData = res.data;
					avatar = d.avatar_url ?? '';
					nickname = d.nickname ?? '';
					bio = d.bio ?? '';
					locationName = d.location?.location_name ?? '';
					handle = d.nickname ? `@${d.nickname.replace(/\s/g, '_')}` : '';
					stats = d.stats ?? stats;
				}
			})
			.finally(() => (loading = false));
	});

	return {
		get avatar() {
			return avatar;
		},
		get nickname() {
			return nickname;
		},
		get bio() {
			return bio;
		},
		get locationName() {
			return locationName;
		},
		get handle() {
			return handle;
		},
		get stats() {
			return stats;
		},
		get loading() {
			return loading;
		}
	};
}

export function useGalleryMoments(type: 'my' | 'user' = 'my', userId?: string) {
	let posts = $state<GalleryMoment[]>([]);
	let loading = $state(true);

	$effect(() => {
		const fetcher = type === 'my' ? listMyMoments : () => listUserMoments(userId!);
		fetcher({ pagination: { limit: 20 } })
			.then((res) => {
				if (res.code === 0 && res.data) {
					posts = res.data.items?.map(mapGalleryMoment) ?? [];
				}
			})
			.finally(() => (loading = false));
	});

	return {
		get posts() {
			return posts;
		},
		get loading() {
			return loading;
		}
	};
}

export function useReactedMoments(userId?: string) {
	let posts = $state<GalleryMoment[]>([]);
	let loading = $state(true);

	$effect(() => {
		listUserReactedMoments(userId ?? '', { pagination: { limit: 20 } })
			.then((res) => {
				if (res.code === 0 && res.data) {
					posts = res.data.items?.map(mapGalleryMoment) ?? [];
				}
			})
			.finally(() => (loading = false));
	});

	return {
		get posts() {
			return posts;
		},
		get loading() {
			return loading;
		}
	};
}

export function useFavoriteMoments() {
	let posts = $state<GalleryMoment[]>([]);
	let loading = $state(true);

	$effect(() => {
		getFavoriteMoments({ pagination: { limit: 20 } })
			.then((res) => {
				if (res.code === 0 && res.data) {
					posts = res.data.items?.map(mapGalleryMoment) ?? [];
				}
			})
			.finally(() => (loading = false));
	});

	return {
		get posts() {
			return posts;
		},
		get loading() {
			return loading;
		}
	};
}

const ROLE_LABEL: Record<string, string> = { owner: '群主', admin: '管理员', member: '成员' };

export interface MappedGroup {
	id: string;
	name: string;
	avatar: string;
	banner: string;
	members: number;
	online: number;
	desc: string;
	role: string;
	lastActive: string;
	tags: string[];
}

/**
 * Map a spec GroupItem (`{ group: GroupInfo, description, my_state, ... }`) to the
 * flat shape SectionGroups/MyGroupCard consumes.
 */
function mapGroup(item: GroupItem): MappedGroup {
	const g = item.group;
	const role = item.my_state?.role;
	return {
		id: g.group_id,
		name: g.name,
		avatar: g.avatar_url ?? '',
		// spec GroupItem has no banner; reuse the avatar so the banner area isn't a
		// broken-image icon. A dedicated banner field can be added when the spec does.
		banner: g.avatar_url ?? '',
		members: g.members_count,
		online: item.online_members_count ?? 0,
		desc: item.description ?? '',
		role: role ? (ROLE_LABEL[role] ?? role) : '成员',
		lastActive: item.last_activity_at ?? '',
		tags: item.tags ?? []
	};
}

/** Days elapsed since `iso`, floored to ≥1 so "今天" reads sensibly. */
function daysSince(iso?: string): number {
	if (!iso) return 1;
	const diff = Date.now() - new Date(iso).getTime();
	return Math.max(1, Math.floor(diff / 86400000));
}

export interface MappedFriend {
	id: string;
	name: string;
	avatar: string;
	online: boolean;
	handle: string;
	tag: string;
}

export interface MappedFollowing {
	id: string;
	name: string;
	avatar: string;
	desc: string;
	isMutual: boolean;
}

export interface MappedFollower {
	id: string;
	name: string;
	avatar: string;
	handle: string;
	joinedDays: number;
	isMutual: boolean;
}

/**
 * Map the three spec relationship items to the flat shapes the Section components
 * expect. The spec nests UserInfo under `user:` and uses presence_status / is_mutual.
 */
function mapFriend(item: FriendItem): MappedFriend {
	const u = item.user;
	return {
		id: u.user_id,
		name: u.nickname,
		avatar: u.avatar_url ?? '',
		online: item.presence_status === 'online',
		handle: item.remark ? `@${item.remark}` : `@${u.user_id}`,
		tag: '好友'
	};
}

function mapFollowing(item: FollowingItem): MappedFollowing {
	const u = item.user;
	return {
		id: u.user_id,
		name: u.nickname,
		avatar: u.avatar_url ?? '',
		desc: item.remark ?? '',
		isMutual: Boolean(item.is_mutual)
	};
}

function mapFollower(item: FollowerItem): MappedFollower {
	const u = item.user;
	return {
		id: u.user_id,
		name: u.nickname,
		avatar: u.avatar_url ?? '',
		// spec FollowerItem 无 remark；用 user_id 作 handle 占位
		handle: `@${u.user_id}`,
		joinedDays: daysSince(item.followed_at),
		isMutual: Boolean(item.is_mutual)
	};
}

export function useMyGroups() {
	let groups = $state<MappedGroup[]>([]);
	let loading = $state(true);

	$effect(() => {
		listMyGroups({ pagination: { limit: 20 } })
			.then((res) => {
				if (res.code === 0) groups = (res.data.items ?? []).map(mapGroup);
			})
			.finally(() => (loading = false));
	});

	return {
		get groups() {
			return groups;
		},
		get loading() {
			return loading;
		}
	};
}

/** Relationship lists — friends / following / followers. */
export function useRelationships() {
	let friends = $state<MappedFriend[]>([]);
	let following = $state<MappedFollowing[]>([]);
	let followers = $state<MappedFollower[]>([]);
	let loading = $state(true);

	$effect(() => {
		Promise.all([
			listFriends({ pagination: { limit: 50 } }).catch(() => ({ code: -1, data: { items: [] } })),
			listFollowing({ pagination: { limit: 50 } }).catch(() => ({ code: -1, data: { items: [] } })),
			listFollowers({ pagination: { limit: 50 } }).catch(() => ({ code: -1, data: { items: [] } }))
		])
			.then(([f, fo, fr]) => {
				friends = f.code === 0 ? (f.data.items ?? []).map(mapFriend) : [];
				following = fo.code === 0 ? (fo.data.items ?? []).map(mapFollowing) : [];
				followers = fr.code === 0 ? (fr.data.items ?? []).map(mapFollower) : [];
			})
			.finally(() => (loading = false));
	});

	return {
		get friends() {
			return friends;
		},
		get following() {
			return following;
		},
		get followers() {
			return followers;
		},
		get loading() {
			return loading;
		}
	};
}
