// 个人中心数据工厂（迁移自旧版 me/profile.data.ts 的 hooks）。
// 组件内调用创建独立响应式状态；数据映射函数（mapGroup/mapFriend 等）原样保留。

import {
	getAccount,
	listUserMoments,
	listUserReactedMoments,
	listMyMoments,
	getFavoritePosts,
	listMyGroups,
	listFriends,
	listFollowing,
	listFollowers
} from '@/api';

export interface GalleryPost {
	id: string;
	img: string;
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

export function useAccountData() {
	let avatar = $state('');
	let nickname = $state('');
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
					const d = res.data as any;
					avatar = d.avatar_url ?? '';
					nickname = d.nickname ?? '';
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
		get stats() {
			return stats;
		},
		get loading() {
			return loading;
		}
	};
}

export function useGalleryPosts(type: 'my' | 'user' = 'my', userId?: string) {
	let posts = $state<GalleryPost[]>([]);
	let loading = $state(true);

	$effect(() => {
		const fetcher = type === 'my' ? listMyMoments : () => listUserMoments(userId!);
		fetcher({ pagination: { limit: 20 } })
			.then((res) => {
				if (res.code === 0 && res.data) {
					posts =
						(res.data as any).items?.map((m: any) => ({
							id: m.moment_id,
							img: m.images?.[0]?.url ?? '',
							likes: m.likes_count ?? 0,
							comments: m.comments_count ?? 0
						})) ?? [];
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

export function useReactedPosts(userId?: string) {
	let posts = $state<GalleryPost[]>([]);
	let loading = $state(true);

	$effect(() => {
		listUserReactedMoments(userId ?? '', { pagination: { limit: 20 } })
			.then((res) => {
				if (res.code === 0 && res.data) {
					posts =
						(res.data as any).items?.map((m: any) => ({
							id: m.moment_id,
							img: m.images?.[0]?.url ?? m.images?.[0] ?? '',
							likes: m.likes_count ?? 0,
							comments: m.comments_count ?? 0
						})) ?? [];
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

export function useFavoritePosts() {
	let posts = $state<GalleryPost[]>([]);
	let loading = $state(true);

	$effect(() => {
		getFavoritePosts({ pagination: { limit: 20 } })
			.then((res) => {
				if (res.code === 0 && res.data) {
					posts =
						(res.data as any).items?.map((m: any) => ({
							id: m.moment_id,
							img: m.images?.[0]?.url ?? m.images?.[0] ?? '',
							likes: m.likes_count ?? 0,
							comments: m.comments_count ?? 0
						})) ?? [];
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

/**
 * Map a spec GroupItem (`{ group: GroupInfo, description, my_state, ... }`) to the
 * flat shape SectionGroups/MyGroupCard consumes.
 */
function mapGroup(item: any) {
	const g = item?.group ?? {};
	const role = item?.my_state?.role;
	return {
		id: g.group_id ?? '',
		name: g.name ?? '',
		avatar: g.avatar_url ?? '',
		// spec GroupItem has no banner; reuse the avatar so the banner area isn't a
		// broken-image icon. A dedicated banner field can be added when the spec does.
		banner: g.avatar_url ?? '',
		members: g.members_count ?? 0,
		// spec GroupItem 无在线人数字段
		online: 0,
		desc: item?.description ?? '',
		role: role ? (ROLE_LABEL[role] ?? role) : '成员',
		lastActive: item?.last_activity_at ?? '',
		tags: item?.tags ?? []
	};
}

/** Days elapsed since `iso`, floored to ≥1 so "今天" reads sensibly. */
function daysSince(iso?: string): number {
	if (!iso) return 1;
	const diff = Date.now() - new Date(iso).getTime();
	return Math.max(1, Math.floor(diff / 86400000));
}

/**
 * Map the three spec relationship items to the flat shapes the Section components
 * expect. The spec nests UserInfo under `user:` and uses presence_status / is_mutual.
 */
function mapFriend(item: any) {
	const u = item?.user ?? {};
	return {
		id: u.user_id ?? '',
		name: u.nickname ?? '',
		avatar: u.avatar_url ?? '',
		online: item?.presence_status === 'online',
		handle: item?.remark ? `@${item.remark}` : `@${u.user_id ?? ''}`,
		mutual: true, // friends list is bidirectional by definition
		tag: '好友'
	};
}

function mapFollowing(item: any) {
	const u = item?.user ?? {};
	return {
		id: u.user_id ?? '',
		name: u.nickname ?? '',
		avatar: u.avatar_url ?? '',
		desc: item?.remark ?? item?.bio ?? '',
		posts: item?.moments_count ?? 0,
		latestImg: item?.latest_moment_image ?? '',
		isMutual: Boolean(item?.is_mutual)
	};
}

function mapFollower(item: any) {
	const u = item?.user ?? {};
	return {
		id: u.user_id ?? '',
		name: u.nickname ?? '',
		avatar: u.avatar_url ?? '',
		handle: item?.remark ? `@${item.remark}` : `@${u.user_id ?? ''}`,
		joinedDays: daysSince(item?.followed_at),
		followBack: Boolean(item?.is_mutual),
		isMutual: Boolean(item?.is_mutual)
	};
}

export function useMyGroups() {
	let groups = $state<any[]>([]);
	let loading = $state(true);

	$effect(() => {
		listMyGroups({ pagination: { limit: 20 } })
			.then((res) => {
				if (res.code === 0) groups = ((res.data as any).items ?? []).map(mapGroup);
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
	let friends = $state<any[]>([]);
	let following = $state<any[]>([]);
	let followers = $state<any[]>([]);
	let loading = $state(true);

	$effect(() => {
		Promise.all([
			listFriends({ pagination: { limit: 50 } }).catch(() => ({ code: -1, data: { items: [] } })),
			listFollowing({ pagination: { limit: 50 } }).catch(() => ({ code: -1, data: { items: [] } })),
			listFollowers({ pagination: { limit: 50 } }).catch(() => ({ code: -1, data: { items: [] } }))
		])
			.then(([f, fo, fr]) => {
				friends = f.code === 0 ? ((f.data as any).items ?? []).map(mapFriend) : [];
				following = fo.code === 0 ? ((fo.data as any).items ?? []).map(mapFollowing) : [];
				followers = fr.code === 0 ? ((fr.data as any).items ?? []).map(mapFollower) : [];
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
