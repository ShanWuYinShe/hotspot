import { openTheDoor, follow, block, setRemark, netError } from '@/api';
import type { UserPresenceStatus } from '@/api/types';
import { auth, requireAuth } from '@/stores/auth.svelte';
import { showToast } from '@/stores/toast.svelte';
import { platform } from '@/stores/platform.svelte';
import { formatCardTime, formatCount, formatDistance } from '@/lib/utils';
import { computeBspLayoutSequence } from '@/lib/semantic-bsp';
import { copyShareLink } from '@/lib/share';

export interface GalleryMomentItem {
	id: string;
	url: string;
	time: string;
	width?: number | null;
	height?: number | null;
	aspectRatio?: number | null;
}

export interface UserProfileData {
	name: string;
	bio: string;
	location: string;
	avatar: string;
	followers: number;
	following: number;
	moments: number;
	galleryMoments: GalleryMomentItem[];
	tags: string[];
}

export function useUserProfileData(userId: () => string) {
	let user = $state<UserProfileData>({
		name: '',
		bio: '',
		location: '',
		avatar: '',
		followers: 0,
		following: 0,
		moments: 0,
		galleryMoments: [],
		tags: []
	});
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let reloadKey = $state(0);
	let mutualFriends = $state<{ id: string; name: string; sub: string; avatar: string }[]>([]);
	let followed = $state(false);
	let moreOpen = $state(false);
	let blocked = $state(false);
	let presence = $state<UserPresenceStatus | null>(null);
	let remark = $state('');
	let remarkDraft = $state('');
	let savingRemark = $state(false);
	let shareCopied = $state(false);

	let followPending = $state(false);
	let blockPending = false;

	let confirmBlockOpen = $state(false);
	let confirmUnblockOpen = $state(false);
	let reportOpen = $state(false);

	const uid = $derived(userId());
	const device = $derived(platform.platform);
	const isSelf = $derived(!!uid && auth.user?.user_id === uid);

	const handleLine = $derived(remark ? `${remark} · @${uid}` : `@${uid}`);

	const metrics = $derived([
		{ label: '瞬间', value: formatCount(user.moments) },
		{ label: '粉丝', value: formatCount(user.followers) },
		{ label: '关注', value: formatCount(user.following) }
	]);

	$effect(() => {
		void reloadKey;
		let cancelled = false;

		async function fetchProfile() {
			try {
				loading = true;
				loadError = null;
				const res = await openTheDoor(uid, true).catch(() => netError(null));
				if (cancelled) return;
				const data = res.code === 0 ? res.data : null;
				if (res.code !== 0) {
					loadError = res.message ?? '加载失败';
					return;
				}
				if (data) {
					const u = data.user;
					const s = u.stats;
					const r = u.relation;
					user = {
						name: u.user.nickname,
						bio: u.bio ?? '',
						location: [u.location?.location_name, formatDistance(u.location?.distance)].filter(Boolean).join(' · '),
						avatar: u.user.avatar_url ?? '',
						followers: s?.followers_count ?? 0,
						following: s?.following_count ?? 0,
						moments: s?.moments_count ?? 0,
						galleryMoments:
							data.recent_moments?.items
								?.filter((m) => m.images?.[0]?.url)
								.map((m) => {
									const img = m.images![0]!;
									const w = img.width ?? null;
									const h = img.height ?? null;
									return {
										id: m.moment_id,
										url: img.url,
										width: w,
										height: h,
										aspectRatio: w && h && h > 0 ? w / h : null,
										time: formatCardTime(m.created_at ?? '')
									};
								}) ?? [],
						tags: u.tags ?? []
					};
					presence = u.presence_status ?? null;
					if (r?.is_following !== undefined) followed = r.is_following;
					if (r?.is_blocked !== undefined) blocked = r.is_blocked;

					const mutual = u.social_context?.mutual_friends_preview;
					if (mutual) {
						mutualFriends = mutual.map((f) => ({
							id: f.user_id,
							name: f.nickname,
							sub: '',
							avatar: f.avatar_url ?? ''
						}));
					}
				}
			} catch {
				if (!cancelled) loadError = '加载失败，请稍后重试';
			} finally {
				if (!cancelled) loading = false;
			}
		}

		void fetchProfile();
		return () => {
			cancelled = true;
		};
	});

	const galleryBspLayouts = $derived(
		computeBspLayoutSequence(
			user.galleryMoments.map((g) => ({
				aspectRatio: g.aspectRatio,
				width: g.width,
				height: g.height,
				hasImage: Boolean(g.url)
			})),
			{ device }
		)
	);

	const toggleFollow = async () => {
		if (blocked || followPending) return;
		if (!requireAuth({ action: '关注', targetId: uid, run: () => void toggleFollow() })) return;
		const next = !followed;
		const prevFollowed = followed;
		const prevFollowers = user.followers;
		followed = next;
		user.followers = Math.max(0, user.followers + (next ? 1 : -1));
		followPending = true;
		try {
			const res = await follow({ user_id: uid, active: next });
			if (res.code !== 0) throw new Error(res.message || '操作失败');
		} catch (err) {
			followed = prevFollowed;
			user.followers = prevFollowers;
			showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
		} finally {
			followPending = false;
		}
	};

	const toggleBlock = async () => {
		if (blockPending) return;
		if (!requireAuth({ action: '屏蔽', targetId: uid, run: () => void toggleBlock() })) return;
		const next = !blocked;
		const prevBlocked = blocked;
		const prevFollowed = followed;
		const prevFollowers = user.followers;
		blocked = next;
		if (next && followed) {
			user.followers = Math.max(0, user.followers - 1);
		}
		followed = false;
		moreOpen = false;
		blockPending = true;
		try {
			const res = await block({ user_id: uid, active: next });
			if (res.code !== 0) throw new Error(res.message || '操作失败');
		} catch (err) {
			blocked = prevBlocked;
			followed = prevFollowed;
			user.followers = prevFollowers;
			showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
		} finally {
			blockPending = false;
		}
	};

	const saveRemark = async () => {
		if (!followed || savingRemark) return;
		savingRemark = true;
		try {
			const res = await setRemark({ user_id: uid, remark: remarkDraft.trim() });
			if (res.code === 0) {
				remark = remarkDraft.trim();
				moreOpen = false;
				showToast('备注已保存', 'success');
			} else {
				showToast(res.message || '保存失败', 'danger');
			}
		} catch {
			showToast('保存失败，请重试', 'danger');
		} finally {
			savingRemark = false;
		}
	};

	const copyProfileLink = async () => {
		const result = await copyShareLink({ type: 'user', id: uid });
		if (result === 'failed') {
			showToast('复制失败，请手动复制地址栏链接', 'danger');
			return;
		}
		shareCopied = true;
		showToast(result === 'shared' ? '分享链接已复制' : '链接已复制', 'success');
		window.setTimeout(() => (shareCopied = false), 1600);
	};

	const refresh = () => {
		reloadKey++;
	};

	return {
		get user() { return user; },
		get loading() { return loading; },
		get loadError() { return loadError; },
		get mutualFriends() { return mutualFriends; },
		get followed() { return followed; },
		get blocked() { return blocked; },
		get presence() { return presence; },
		get remark() { return remark; },
		get remarkDraft() { return remarkDraft; },
		set remarkDraft(v: string) { remarkDraft = v; },
		get savingRemark() { return savingRemark; },
		get shareCopied() { return shareCopied; },
		get followPending() { return followPending; },
		get isSelf() { return isSelf; },
		get handleLine() { return handleLine; },
		get metrics() { return metrics; },
		get galleryBspLayouts() { return galleryBspLayouts; },
		// 弹窗状态
		get moreOpen() { return moreOpen; },
		set moreOpen(v: boolean) { moreOpen = v; },
		get confirmBlockOpen() { return confirmBlockOpen; },
		set confirmBlockOpen(v: boolean) { confirmBlockOpen = v; },
		get confirmUnblockOpen() { return confirmUnblockOpen; },
		set confirmUnblockOpen(v: boolean) { confirmUnblockOpen = v; },
		get reportOpen() { return reportOpen; },
		set reportOpen(v: boolean) { reportOpen = v; },
		// 动作
		toggleFollow,
		toggleBlock,
		saveRemark,
		copyProfileLink,
		refresh
	};
}

export type UserPageData = ReturnType<typeof useUserProfileData>;
