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
, netError } from '@/api';
import type {
	AccountData,
	RelatedUserItem,
	GroupItem,
	MomentItem
} from '@/api/types';

export interface GalleryMoment {
	id: string;
	/** 软删除墓碑（后端下发 is_deleted，契约要求客户端展示占位而非空白卡）。 */
	is_deleted: boolean;
	img: string;
	title: string;
	likes: number;
	comments: number;
	created_at: string;
	user?: string;
	avatar?: string;
	location?: string;
	tag?: string;
	liked?: boolean;
	saved?: boolean;
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
	const mine = (m.reactions || []).find((r) => r.reacted_by_me);
	return {
		id: m.moment_id,
		is_deleted: m.is_deleted ?? false,
		img: m.images?.[0]?.url ?? '',
		title: m.description ?? '',
		likes: (m.reactions ?? []).reduce((sum, r) => sum + r.count, 0),
		comments: m.comments_count ?? 0,
		created_at: m.created_at ?? '',
		user: m.user?.nickname ?? '',
		avatar: m.user?.avatar_url ?? '',
		location: m.location?.location_name ?? '',
		tag: m.tags?.[0] ?? '',
		liked: Boolean(mine),
		saved: Boolean(m.is_favorited)
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
	let error = $state<string | null>(null);
	// 未登录（10100）是正常业务态而非错误：重试无意义，UI 应引导登录
	let needsAuth = $state(false);
	let reloadKey = $state(0);

	$effect(() => {
		void reloadKey;
		loading = true;
		error = null;
		needsAuth = false;
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
				} else if (res.code === 10100) {
					needsAuth = true;
				} else {
					error = res.message ?? '加载失败';
				}
			})
			.catch(() => (error = '网络异常，请稍后重试'))
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
		},
		get error() {
			return error;
		},
		get needsAuth() {
			return needsAuth;
		},
		retry() {
			reloadKey++;
		}
	};
}

export function useGalleryMoments(type: 'my' | 'user' = 'my', userId?: string) {
	let posts = $state<GalleryMoment[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let needsAuth = $state(false);
	let reloadKey = $state(0);

	$effect(() => {
		void reloadKey;
		loading = true;
		error = null;
		needsAuth = false;
		const fetcher =
			type === 'my' ? listMyMoments : (params: { pagination?: { limit?: number; cursor?: string } }) =>
					listUserMoments(userId!, params.pagination);
		fetcher({ pagination: { limit: 20 } })
			.then((res) => {
				if (res.code === 0 && res.data) {
					posts = res.data.items?.map(mapGalleryMoment) ?? [];
				} else {
					if (res.code === 10100) { needsAuth = true; } else { error = res.message ?? '加载失败'; }
				}
			})
			.catch(() => (error = '网络异常，请稍后重试'))
			.finally(() => (loading = false));
	});

	return {
		get posts() {
			return posts;
		},
		get loading() {
			return loading;
		},
		get error() {
			return error;
		},
		get needsAuth() {
			return needsAuth;
		},
		retry() {
			reloadKey++;
		}
	};
}

export function useReactedMoments(userId?: string) {
	let posts = $state<GalleryMoment[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let needsAuth = $state(false);
	let reloadKey = $state(0);

	$effect(() => {
		void reloadKey;
		loading = true;
		error = null;
		needsAuth = false;
		listUserReactedMoments(userId ?? '', { limit: 20 })
			.then((res) => {
				if (res.code === 0 && res.data) {
					posts = res.data.items?.map(mapGalleryMoment) ?? [];
				} else {
					if (res.code === 10100) { needsAuth = true; } else { error = res.message ?? '加载失败'; }
				}
			})
			.catch(() => (error = '网络异常，请稍后重试'))
			.finally(() => (loading = false));
	});

	return {
		get posts() {
			return posts;
		},
		get loading() {
			return loading;
		},
		get error() {
			return error;
		},
		get needsAuth() {
			return needsAuth;
		},
		retry() {
			reloadKey++;
		}
	};
}

export function useFavoriteMoments() {
	let posts = $state<GalleryMoment[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let needsAuth = $state(false);
	let reloadKey = $state(0);

	$effect(() => {
		void reloadKey;
		loading = true;
		error = null;
		needsAuth = false;
		getFavoriteMoments({ pagination: { limit: 20 } })
			.then((res) => {
				if (res.code === 0 && res.data) {
					posts = res.data.items?.map(mapGalleryMoment) ?? [];
				} else {
					if (res.code === 10100) { needsAuth = true; } else { error = res.message ?? '加载失败'; }
				}
			})
			.catch(() => (error = '网络异常，请稍后重试'))
			.finally(() => (loading = false));
	});

	return {
		get posts() {
			return posts;
		},
		get loading() {
			return loading;
		},
		get error() {
			return error;
		},
		get needsAuth() {
			return needsAuth;
		},
		retry() {
			reloadKey++;
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
 * Map a spec GroupItem (`{ group: GroupInfo, description, viewer_state, ... }`) to the
 * flat shape SectionGroups/MyGroupCard consumes.
 */
function mapGroup(item: GroupItem): MappedGroup {
	const g = item.group;
	const role = item.viewer_state?.role;
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
 * expect. The spec nests UserInfo under `user:` and uses presence_status;
 * mutual follows are computed from `user.relation` (is_following && is_followed_by).
 */
// spec 三列表统一为 RelatedUserItem（user: UserItem 承载 presence 隐私字段；
// remark 仅查询自己时返回，followers 列表在互关场景同样返回）。
function mapFriend(item: RelatedUserItem): MappedFriend {
	const u = item.user;
	return {
		id: u.user.user_id,
		name: u.user.nickname,
		avatar: u.user.avatar_url ?? '',
		online: u.presence_status === 'online',
		handle: item.remark ? `@${item.remark}` : `@${u.user.user_id}`,
		tag: '好友'
	};
}

// spec: RelatedUserItem 无 is_mutual 冗余字段——互关由 user.relation 计算
function isMutualRelation(item: RelatedUserItem): boolean {
	const relation = item.user.relation;
	return relation ? relation.is_following && relation.is_followed_by : false;
}

function mapFollowing(item: RelatedUserItem): MappedFollowing {
	const u = item.user;
	return {
		id: u.user.user_id,
		name: u.user.nickname,
		avatar: u.user.avatar_url ?? '',
		desc: item.remark ?? '',
		isMutual: isMutualRelation(item)
	};
}

function mapFollower(item: RelatedUserItem): MappedFollower {
	const u = item.user;
	return {
		id: u.user.user_id,
		name: u.user.nickname,
		avatar: u.user.avatar_url ?? '',
		handle: item.remark ? `@${item.remark}` : `@${u.user.user_id}`,
		joinedDays: daysSince(item.followed_at),
		isMutual: isMutualRelation(item)
	};
}

export function useMyGroups() {
	let groups = $state<MappedGroup[]>([]);
	let loading = $state(true);

	let error = $state<string | null>(null);
	let needsAuth = $state(false);
	let reloadKey = $state(0);

	$effect(() => {
		void reloadKey;
		loading = true;
		error = null;
		needsAuth = false;
		listMyGroups({ pagination: { limit: 20 } })
			.then((res) => {
				if (res.code === 0) groups = (res.data.items ?? []).map(mapGroup);
				else if (res.code === 10100) { needsAuth = true; } else { error = res.message ?? '加载失败'; }
			})
			.catch(() => (error = '网络异常，请稍后重试'))
			.finally(() => (loading = false));
	});

	return {
		get groups() {
			return groups;
		},
		get loading() {
			return loading;
		},
		get error() {
			return error;
		},
		get needsAuth() {
			return needsAuth;
		},
		retry() {
			reloadKey++;
		}
	};
}

/** Relationship lists — friends / following / followers. */
export function useRelationships() {
	let friends = $state<MappedFriend[]>([]);
	let following = $state<MappedFollowing[]>([]);
	let followers = $state<MappedFollower[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let needsAuth = $state(false);
	let reloadKey = $state(0);

	$effect(() => {
		void reloadKey;
		loading = true;
		error = null;
		needsAuth = false;
		// spec: users/fetch_friends|following|followers 的 user_id 必传（拉自己的列表也要显式传自己的 user_id）。
		getAccount()
			.then((accountRes) => {
				if (accountRes.code !== 0 || !accountRes.data) {
					if (accountRes.code === 10100) needsAuth = true;
					else error = accountRes.message ?? '加载失败';
					loading = false;
					return;
				}
				const userId = accountRes.data.user_id;
				return Promise.all([
					listFriends({ user_id: userId, pagination: { limit: 50 } }).catch(() => netError({ items: [] })),
					listFollowing({ user_id: userId, pagination: { limit: 50 } }).catch(() => netError({ items: [] })),
					listFollowers({ user_id: userId, pagination: { limit: 50 } }).catch(() => netError({ items: [] }))
				]);
			})
			.then((results) => {
				if (!results) return;
				const [f, fo, fr] = results;
				friends = f.code === 0 ? (f.data.items ?? []).map(mapFriend) : [];
				following = fo.code === 0 ? (fo.data.items ?? []).map(mapFollowing) : [];
				followers = fr.code === 0 ? (fr.data.items ?? []).map(mapFollower) : [];
			})
			.catch(() => (error = '网络异常，请稍后重试'))
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
		},
		get error() {
			return error;
		},
		get needsAuth() {
			return needsAuth;
		},
		retry() {
			reloadKey++;
		}
	};
}
