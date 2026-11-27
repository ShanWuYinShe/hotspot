import { untrack } from 'svelte';
import { page } from '$app/state';
import {
	listNotifies,
	listSystemAnnouncements,
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
import type { GroupInviteItem, NotifyItem, SystemAnnouncementItem } from '@/api/types';
import {
	normalizeNotifType,
	resolveNotifHref,
	type NotifyViewItem
} from './notify-data';

export const FILTER_TABS = [
	{ id: 'all', label: '全部' },
	{ id: 'unread', label: '未读' },
	{ id: 'reaction', label: '点赞' },
	{ id: 'follow', label: '关注' },
	{ id: 'comment', label: '评论' },
	{ id: 'nearby', label: '附近' }
];

const FILTER_IDS = new Set(FILTER_TABS.map((t) => t.id));

const NOTIFY_PAGE_SIZE = 30;

/** 分类筛选（点赞/关注/…）是语义分组，契约只支持单 type，仍在已加载页内过滤；
 * 「未读」走服务端 is_read 筛选，切档需重新拉第一页。 */
function readFilterOf(filter: string): boolean | null {
	return filter === 'unread' ? false : null;
}

function initialFilter(): string {
	const fromUrl = page.url.searchParams.get('filter') ?? '';
	return FILTER_IDS.has(fromUrl) ? fromUrl : 'all';
}

export function useNotifyData() {
	let notifies = $state<NotifyViewItem[]>([]);
	let invites = $state<GroupInviteItem[]>([]);
	let invitesLoaded = false;
	/** 平台公告（公开端点，游客也可见；与通知列表相互独立）。 */
	let announcements = $state<SystemAnnouncementItem[]>([]);
	let filter = $state(initialFilter());
	/** 仅「未读」档需要服务端筛选，其余档位在已加载页内过滤。 */
	const serverReadFilter = $derived(readFilterOf(filter));
	let nextCursor = $state<string | null>(null);
	let loadingMore = $state(false);
	/** 续页失败：停止哨兵自动重试并给出手动重试入口（与首屏 loadError 区分）。 */
	let loadMoreError = $state(false);

	let needsAuth = $state(false);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let reloadKey = $state(0);

	/** 上一次服务端已读筛选值：切档时清空旧列表（旧数据在新筛选条件下不可信）。 */
	let lastReadFilter: boolean | null = null;

	let query = $state('');
	let searchReloadKey = $state(0);
	const searching = $derived(query.trim().length > 0);
	let searchItems = $state<NotifyViewItem[]>([]);
	let searchLoaded = $state(0);
	let searchLoading = $state(false);
	let searchError = $state<string | null>(null);

	let followedBack = $state<Set<string>>(new Set());
	let invitePending = $state(new Set<string>());

	// WS 实时通知推送监听：250ms 合并（连续点赞/评论会连着推多条事件，
	// 每条都触发一次「通知+未读+邀请+关注」四请求全量重拉会成倍放大，W15）。
	let reloadTimer: ReturnType<typeof setTimeout> | null = null;
	$effect(() => {
		const onNotify = () => {
			if (reloadTimer) clearTimeout(reloadTimer);
			reloadTimer = setTimeout(() => {
				reloadTimer = null;
				reloadKey++;
			}, 250);
		};
		window.addEventListener(WS_NOTIFY_EVENT, onNotify);
		return () => {
			window.removeEventListener(WS_NOTIFY_EVENT, onNotify);
			if (reloadTimer) {
				clearTimeout(reloadTimer);
				reloadTimer = null;
			}
		};
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

	// 拉取通知与未读；「未读」档随筛选切换重新拉第一页（服务端 is_read）
	$effect(() => {
		void reloadKey;
		const readFilter = serverReadFilter;
		if (readFilter !== lastReadFilter) {
			lastReadFilter = readFilter;
			notifies = [];
			nextCursor = null;
			loading = true;
		}
		let cancelled = false;

		async function fetchNotifs() {
			try {
				needsAuth = false;
				if (untrack(() => !notifies.length)) loading = true;
				loadError = null;

				const [notifRes, unreadRes, invitesRes, followingRes, announcementsRes] = await Promise.all([
					listNotifies({ is_read: readFilter, pagination: { limit: NOTIFY_PAGE_SIZE } }).catch(() => netError({ items: [], next_cursor: null })),
					getUnreadCount().catch(() => netError({ total: 0 })),
					listMyInvites().catch(() => netError({ items: [] })),
					listFollowing({ user_id: auth.user?.user_id ?? '', pagination: { limit: 100 } }).catch(() => netError({ items: [], next_cursor: null })),
					listSystemAnnouncements().catch(() => null)
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

				// 公告失败不阻塞通知主流程（游客也能看到公告，无需登录态）
				if (announcementsRes?.code === 0 && announcementsRes.data) {
					announcements = announcementsRes.data.items ?? [];
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
		loadMoreError = false;
		try {
			const res = await listNotifies({ is_read: serverReadFilter, pagination: { limit: NOTIFY_PAGE_SIZE, cursor: nextCursor } }).catch(() => null);
			if (!res || res.code !== 0 || !res.data) {
				loadMoreError = true;
				return;
			}
			notifies = stripInviteRowActions([...notifies, ...toViewItems(res.data.items || [], notifies.length)]);
			nextCursor = res.data.next_cursor ?? null;
		} finally {
			loadingMore = false;
		}
	}

	/** 续页失败后手动重试（哨兵在 error=true 时不自动重试）。 */
	async function retryLoadMore() {
		loadMoreError = false;
		await loadMoreNotifs();
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
		// 「未读」档下全部已读后列表应清空（加载态/到底态同步复位）
		if (serverReadFilter === false) {
			notifies = [];
			nextCursor = null;
		}
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

	// 通知契约（notify/fetch）无 keyword 参数：搜索时先自动加载已加载分页（最多 10 页），
	// 再按通知内容/用户 ID/类型/跳转目标本地匹配；清空后回到常规分页。
	// 输入 300ms 防抖：此前每按一键都触发一轮最多 10 次串行 notify/fetch（W07）。
	$effect(() => {
		const q = query.trim();
		void searchReloadKey;
		if (!q) {
			searchItems = [];
			searchLoaded = 0;
			searchLoading = false;
			searchError = null;
			return;
		}
		let cancelled = false;
		searchLoading = true;
		searchError = null;

		async function runSearch() {
			const all: NotifyItem[] = [];
			let cursor: string | undefined;
			for (let pageIndex = 0; pageIndex < 10; pageIndex++) {
				const res = await listNotifies({ pagination: { limit: NOTIFY_PAGE_SIZE, cursor } }).catch(() => null);
				if (cancelled) return;
				if (!res || res.code !== 0 || !res.data) {
					searchError = res?.message ?? '搜索失败';
					break;
				}
				all.push(...(res.data.items ?? []));
				cursor = res.data.next_cursor ?? undefined;
				if (!cursor) break;
			}
			if (cancelled) return;
			searchLoaded = all.length;
			searchItems = stripInviteRowActions(toViewItems(all, 0)).filter((n) =>
				[n.user.name, n.user.id, n.text, n.type, n.href ?? '', n.id].join(' ').toLowerCase().includes(q.toLowerCase())
			);
			searchLoading = false;
		}

		const timer = setTimeout(() => void runSearch(), 300);
		return () => {
			cancelled = true;
			clearTimeout(timer);
		};
	});

	// 派生计算
	const unread = $derived(notifies.filter((n) => !n.read).length);
	/** 服务端未读总数（页头/「全部已读」按钮口径；未读散落在后续分页时也不失真）。 */
	const totalUnread = $derived(globalUnread.notify);
	const meId = $derived(auth.user?.user_id ?? '');
	const filteredNotifies = $derived(
		(searching ? searchItems : notifies).filter((n) =>
			// 「未读」档：服务端已只返回未读，单条标记已读后也要即时从当前列表消失
			filter === 'unread' ? !n.read : filter === 'all' || n.type === filter
		)
	);

	/** 筛选档写入 URL（刷新/返回保持；与 /me/shares 等页 ?tab= 约定一致）。 */
	function setFilter(v: string) {
		filter = v;
		if (typeof window === 'undefined') return;
		const url = new URL(window.location.href);
		if (v === 'all') url.searchParams.delete('filter');
		else url.searchParams.set('filter', v);
		window.history.replaceState(window.history.state, '', url.toString());
	}

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
		get query() { return query; },
		set query(v: string) { query = v; },
		get searching() { return searching; },
		get searchLoaded() { return searchLoaded; },
		get searchLoading() { return searchLoading; },
		get searchError() { return searchError; },
		retrySearch: () => { searchReloadKey++; },
		get filteredNotifies() { return filteredNotifies; },
		get invites() { return invites; },
		get announcements() { return announcements; },
		get filter() { return filter; },
		set filter(v: string) { setFilter(v); },
		get nextCursor() { return nextCursor; },
		get loadingMore() { return loadingMore; },
		get loadMoreError() { return loadMoreError; },
		retryLoadMore,
		get needsAuth() { return needsAuth; },
		get loading() { return loading; },
		get loadError() { return loadError; },
		get unread() { return unread; },
		get totalUnread() { return totalUnread; },
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