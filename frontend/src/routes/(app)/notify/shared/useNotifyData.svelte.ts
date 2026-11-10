import { untrack } from 'svelte';
import {
	listNotifies,
	getUnreadCount,
	markRead,
	listMyInvites,
	acceptGroupInvite,
	rejectGroupInvite,
	listFollowing,
	follow,
	netError
} from '@/api';
import { setNotifyUnread, unread as globalUnread } from '@/stores/unread.svelte';
import { auth } from '@/stores/auth.svelte';
import { WS_NOTIFY_EVENT } from '@/lib/ws/client';
import { formatMomentTime } from '@/lib/utils';
import { showToast } from '@/stores/toast.svelte';
import type { GroupInviteItem, NotifyItem } from '@/api/types';
import {
	normalizeNotifType,
	resolveNotifHref,
	type NotifyViewItem
} from './notify-data';

export const FILTER_TABS = [
	{ id: 'all', label: '全部' },
	{ id: 'reaction', label: '点赞' },
	{ id: 'follow', label: '关注' },
	{ id: 'comment', label: '评论' },
	{ id: 'nearby', label: '附近' }
];

const NOTIFY_PAGE_SIZE = 30;

export function useNotifyData() {
	let notifies = $state<NotifyViewItem[]>([]);
	let invites = $state<GroupInviteItem[]>([]);
	let invitesLoaded = false;
	let filter = $state('all');
	let nextCursor = $state<string | null>(null);
	let loadingMore = $state(false);

	let needsAuth = $state(false);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let reloadKey = $state(0);

	let followedBack = $state<Set<string>>(new Set());
	let invitePending = $state(new Set<string>());

	// WS 实时通知推送监听
	$effect(() => {
		const onNotify = () => {
			reloadKey++;
		};
		window.addEventListener(WS_NOTIFY_EVENT, onNotify);
		return () => window.removeEventListener(WS_NOTIFY_EVENT, onNotify);
	});

	function toViewItems(raw: NotifyItem[], offset: number): NotifyViewItem[] {
		const sorted = raw.slice().sort((a, b) =>
			String(b.created_at ?? '').localeCompare(String(a.created_at ?? ''))
		);
		return sorted.map((n, i) => {
			const actor = n.payload?.actor ?? n.payload?.applicant ?? { user_id: '', nickname: '', avatar_url: null };
			const groupName = n.payload?.group_name ?? '群组';
			let text = '';
			if (n.type === 'moment_commented') text = n.payload?.comment_preview ?? '评论了你的动态';
			else if (n.type === 'comment_reply') text = `回复了你的评论：${n.payload?.reply_content_preview ?? ''}`;
			else if (n.type === 'new_follower') text = '关注了你';
			else if (n.type === 'mutual_follow') text = '和你互相关注了';
			else if (n.type === 'group_invite_received') text = `邀请你加入 ${groupName}`;
			else if (n.type === 'group_application_received') text = `申请加入「${groupName}」，去群管理处理`;
			else if (n.type === 'group_application_accepted') text = `你加入「${groupName}」的申请已通过`;
			else if (n.type === 'group_application_rejected') text = `${n.payload?.rejected_by?.nickname ?? '管理员'} 拒绝了你的「${groupName}」入群申请`;
			else if (n.type === 'reaction_added') text = `做出了反应 ${n.payload?.emoji ?? ''}`;
			else if (n.type === 'group_mentioned') text = `在「${groupName}」群聊中 @ 了你${n.payload?.content_preview ? `：${n.payload.content_preview}` : ''}`;
			else if (n.type === 'report_resolved') text = `你举报的内容已处理${n.payload?.note ? `：${n.payload.note}` : ''}`;
			else if (n.type === 'report_dismissed') text = `你的举报未通过审核${n.payload?.note ? `：${n.payload.note}` : ''}`;
			else if (n.type === 'nearby_new_moment') {
				const dist = typeof n.payload?.distance === 'number' ? n.payload.distance : null;
				const distText = dist === null ? '' : dist <= 1000 ? `（${dist}m）` : `（${(dist / 1000).toFixed(1)}km）`;
				text = `在附近发布了新瞬间${distText}`;
			}
			else if (n.type === 'system_announcement') text = n.payload?.title ?? '系统公告';
			else text = n.payload?.message ?? '';

			const actorId = String(actor.user_id ?? '');
			const href = resolveNotifHref(n.type ?? '', n.payload ?? {}, actorId);
			return {
				id: n.notify_id ?? String(offset + i + 1),
				type: normalizeNotifType(n.type ?? 'reaction_added'),
				read: n.is_read ?? false,
				time: formatMomentTime(n.created_at),
				user: { id: actor.user_id ?? '', name: actor.nickname ?? '', avatar: actor.avatar_url ?? '' },
				text,
				momentImage: n.payload?.moment_summary?.cover_url ?? undefined,
				inviteId: n.type === 'group_invite_received' ? (n.payload as { invite_id?: string })?.invite_id : undefined,
				href
			};
		});
	}

	function stripInviteRowActions(items: NotifyViewItem[]): NotifyViewItem[] {
		if (!invitesLoaded) return items;
		return items.map((n) => (n.inviteId ? { ...n, inviteId: undefined } : n));
	}

	// 拉取通知与未读
	$effect(() => {
		void reloadKey;
		let cancelled = false;

		async function fetchNotifs() {
			try {
				needsAuth = false;
				if (untrack(() => !notifies.length)) loading = true;
				loadError = null;

				const [notifRes, unreadRes, invitesRes, followingRes] = await Promise.all([
					listNotifies({ pagination: { limit: NOTIFY_PAGE_SIZE } }).catch(() => netError({ items: [], next_cursor: null })),
					getUnreadCount().catch(() => netError({ total: 0 })),
					listMyInvites().catch(() => netError({ items: [] })),
					listFollowing({ user_id: auth.user?.user_id ?? '', pagination: { limit: 100 } }).catch(() => netError({ items: [], next_cursor: null }))
				]);
				if (cancelled) return;

				const notifData = notifRes.code === 0 ? notifRes.data : null;
				if (notifRes.code === 10100) {
					needsAuth = true;
					return;
				}
				if (notifRes.code !== 0) {
					if (notifies.length) showToast(notifRes.message || '通知刷新失败', 'danger');
					else loadError = notifRes.message ?? '加载失败';
					return;
				}

				nextCursor = notifData?.next_cursor ?? null;

				if (invitesRes.code === 0) {
					invites = invitesRes.data?.items ?? [];
					invitesLoaded = true;
				}

				if (followingRes.code === 0) {
					const ids = (followingRes.data?.items ?? [])
						.map((u) => u.user?.user?.user_id ?? '')
						.filter(Boolean);
					followedBack = new Set([...followedBack, ...ids]);
				}

				if (notifData) {
					notifies = stripInviteRowActions(toViewItems(notifData.items || [], 0));
				}

				if (unreadRes.code === 0 && unreadRes.data) {
					setNotifyUnread(unreadRes.data.total);
				}
			} catch {
				if (!cancelled) {
					if (notifies.length) showToast('通知刷新失败，请稍后重试', 'danger');
					else loadError = '加载失败，请稍后重试';
				}
			} finally {
				if (!cancelled) loading = false;
			}
		}

		void fetchNotifs();
		return () => {
			cancelled = true;
		};
	});

	async function loadMoreNotifs() {
		if (nextCursor === null || loadingMore) return;
		loadingMore = true;
		try {
			const res = await listNotifies({ pagination: { limit: NOTIFY_PAGE_SIZE, cursor: nextCursor } }).catch(() => netError({ items: [], next_cursor: null }));
			if (res.code === 0 && res.data) {
				notifies = stripInviteRowActions([...notifies, ...toViewItems(res.data.items || [], notifies.length)]);
				nextCursor = res.data.next_cursor ?? null;
			}
		} finally {
			loadingMore = false;
		}
	}

	const followBack = async (n: NotifyViewItem) => {
		if (!n.user.id || followedBack.has(n.user.id)) return;
		followedBack = new Set([...followedBack, n.user.id]);
		try {
			const res = await follow({ user_id: n.user.id, active: true });
			if (res.code !== 0) throw new Error(res.message || '操作失败');
		} catch (err) {
			followedBack = new Set([...followedBack].filter((id) => id !== n.user.id));
			showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
		}
	};

	const markAllRead = () => {
		const snapshot = notifies;
		notifies = notifies.map((n) => ({ ...n, read: true }));
		setNotifyUnread(0);
		void markRead({ scope: 'all' }).catch(() => {
			notifies = snapshot;
			void getUnreadCount().then((r) => {
				if (r.code === 0 && r.data) setNotifyUnread(r.data.total);
			});
			showToast('全部已读失败，请重试', 'danger');
		});
	};

	const markSingleRead = (id: string) => {
		const wasUnread = notifies.find((n) => n.id === id)?.read === false;
		notifies = notifies.map((n) => (n.id === id ? { ...n, read: true } : n));
		if (wasUnread) setNotifyUnread(Math.max(0, globalUnread.notify - 1));
		void markRead({ scope: 'ids', notify_ids: [id] }).catch(() => {});
	};

	const remove = (id: string) => {
		notifies = notifies.filter((n) => n.id !== id);
	};

	const removeInvite = (inviteId: string) => {
		invites = invites.filter((i) => i.invite_id !== inviteId);
		notifies = notifies.filter((n) => n.inviteId !== inviteId);
	};

	const handleAcceptInvite = async (inviteId: string) => {
		if (invitePending.has(inviteId)) return;
		invitePending = new Set(invitePending).add(inviteId);
		try {
			const res = await acceptGroupInvite({ invite_id: inviteId });
			if (res.code === 0) {
				showToast('已加入群组', 'success');
				removeInvite(inviteId);
			} else {
				showToast(res.message || '操作失败', 'danger');
			}
		} catch {
			showToast('操作失败，请重试', 'danger');
		} finally {
			const rest = new Set(invitePending);
			rest.delete(inviteId);
			invitePending = rest;
		}
	};

	const handleRejectInvite = async (inviteId: string) => {
		if (invitePending.has(inviteId)) return;
		invitePending = new Set(invitePending).add(inviteId);
		try {
			const res = await rejectGroupInvite({ invite_id: inviteId });
			if (res.code === 0) {
				showToast('已忽略该邀请', 'info');
				removeInvite(inviteId);
			} else {
				showToast(res.message || '操作失败', 'danger');
			}
		} catch {
			showToast('操作失败，请重试', 'danger');
		} finally {
			const rest = new Set(invitePending);
			rest.delete(inviteId);
			invitePending = rest;
		}
	};

	const refresh = () => {
		reloadKey++;
	};

	// 派生计算
	const unread = $derived(notifies.filter((n) => !n.read).length);
	const meId = $derived(auth.user?.user_id ?? '');
	const filteredNotifies = $derived(
		filter === 'all' ? notifies : notifies.filter((n) => n.type === filter)
	);

	const reactionCount = $derived(notifies.filter((n) => n.type === 'reaction').length);
	const followCount = $derived(notifies.filter((n) => n.type === 'follow').length);
	const commentCount = $derived(notifies.filter((n) => n.type === 'comment').length);
	const nearbyCount = $derived(notifies.filter((n) => n.type === 'nearby').length);

	const recentFollowers = $derived.by(() => {
		const seen = new Set<string>();
		const list: NotifyViewItem[] = [];
		for (const n of notifies) {
			if (n.type === 'follow' && n.user.id && !seen.has(n.user.id)) {
				seen.add(n.user.id);
				list.push(n);
				if (list.length >= 5) break;
			}
		}
		return list;
	});

	return {
		get notifies() { return notifies; },
		get filteredNotifies() { return filteredNotifies; },
		get invites() { return invites; },
		get filter() { return filter; },
		set filter(v: string) { filter = v; },
		get nextCursor() { return nextCursor; },
		get loadingMore() { return loadingMore; },
		get needsAuth() { return needsAuth; },
		get loading() { return loading; },
		get loadError() { return loadError; },
		get unread() { return unread; },
		get meId() { return meId; },
		get followedBack() { return followedBack; },
		get invitePending() { return invitePending; },
		get reactionCount() { return reactionCount; },
		get followCount() { return followCount; },
		get commentCount() { return commentCount; },
		get nearbyCount() { return nearbyCount; },
		get recentFollowers() { return recentFollowers; },
		loadMoreNotifs,
		followBack,
		markAllRead,
		markSingleRead,
		remove,
		handleAcceptInvite,
		handleRejectInvite,
		refresh
	};
}

export type NotifyData = ReturnType<typeof useNotifyData>;
