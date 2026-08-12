// 群详情业务工厂（收敛群详情页的状态/加载/WS/操作）。
// 参照 chat.svelte.ts 的依赖注入 + 竞态防护 + getter 范本；工厂零视图依赖
// （goto/ROUTES/presence 走 options 注入），保持纯净可测。
//
// 调用方式：组件 <script> 顶层 const gd = useGroupDetail({...})；工厂内的
// $state/$effect 生命周期绑定组件实例，卸载自动清理（WS 订阅 + presence 订阅）。

import { getGroupDetail, getGroupMembers, getGroupAnnouncements, joinGroup, leaveGroup, createGroupAnnouncement, updateGroupAnnouncement, deleteGroupAnnouncement } from '@/api';
import { wsClient } from '@/lib/ws/client';
import { formatDistance, formatDateTime } from '@/lib/utils';
import type { GroupRole } from '@/api/types';

export interface GroupDetail {
	name: string;
	category: string;
	members: number;
	online: number;
	distance: string;
	description: string;
	banner: string;
}

export interface GroupMember {
	id: number;
	userId: string;
	name: string;
	role: string;
	avatar: string;
}

export interface GroupAnnouncement {
	id: string;
	title: string;
	body: string;
	time: string;
	isPinned: boolean;
}

export interface UseGroupDetailOptions {
	/** 群 id getter（私聊/路由参数）；变化时自动重载并重绑 WS。 */
	groupId: () => string;
	/** GroupDismissed 回调（页面注入 alert + goto，工厂不 import 导航）。 */
	onDismissed?: () => void;
	/** presence 订阅（页面注入 watchPresence）。 */
	watchPresence?: (ids: string[]) => () => void;
	/** 在线判定（页面注入 presence.isOnline）。 */
	isOnline?: (userId: string) => boolean;
}

export function useGroupDetail(opts: UseGroupDetailOptions) {
	let detail = $state<GroupDetail>({ name: '', category: '', members: 0, online: 0, distance: '', description: '', banner: '' });
	let members = $state<GroupMember[]>([]);
	let announcements = $state<GroupAnnouncement[]>([]);
	let joined = $state(false);
	/** 加入/退出请求进行中——防连点 */
	let joinPending = $state(false);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	/** 当前用户在该群的角色（owner/admin/member）；未加入或缺省为 null。 */
	let myRole = $state<GroupRole | null>(null);
	let unwatchMembers: (() => void) | null = null;
	/** 竞态防护：仅最后一次发起的加载允许落地。 */
	let loadToken = 0;

	const currentGid = () => opts.groupId();

	// --- 加载函数（初始加载与 WS 事件刷新复用） ---

	async function loadDetail(gid: string) {
		try {
			const detailRes = await getGroupDetail(gid).catch(() => ({ code: -1, message: '网络异常', data: null }));
			// 竞态防护：页面已切到别的群组则丢弃本次结果
			if (currentGid() !== gid) return;
			const detailData = detailRes.code === 0 ? detailRes.data : null;
			if (detailData) {
				const g = detailData.group;
				detail = {
					name: g.name,
					category: (detailData.tags?.[0] ?? detail.category) || '同城',
					members: g.members_count,
					online: detailData.online_members_count ?? detail.online,
					distance: formatDistance(detailData.location?.distance) || detail.distance,
					description: detailData.description ?? detail.description,
					banner: g.avatar_url ?? detail.banner
				};
				if (detailData.my_state?.is_joined) joined = true;
				myRole = detailData.my_state?.role ?? null;
			} else if (detailRes.code !== 0) {
				loadError = detailRes.message ?? '加载失败，请稍后重试';
			}
		} catch {
			loadError = '加载失败，请稍后重试';
		}
	}

	async function loadMembers(gid: string) {
		try {
			const membersRes = await getGroupMembers(gid).catch(() => ({ code: -1, message: '网络异常', data: { items: [] } }));
			if (currentGid() !== gid) return;
			const membersData = membersRes.code === 0 ? membersRes.data : null;
			if (membersData) {
				const items: GroupMember[] = (membersData.items || []).map((m, i) => ({
					id: i + 1,
					userId: m.user?.user_id ?? '',
					name: m.user?.nickname ?? '',
					role: m.role ?? '成员',
					avatar: m.user?.avatar_url ?? ''
				}));
				members = items;
				// 成员在线状态实时化：订阅全部成员（先释放旧订阅再建新订阅）
				unwatchMembers?.();
				if (opts.watchPresence) {
					unwatchMembers = opts.watchPresence(items.map((m) => m.userId).filter(Boolean));
				}
			}
		} catch {
			// empty data is normal
		}
	}

	async function loadAnnouncements(gid: string) {
		try {
			const announcementsRes = await getGroupAnnouncements(gid).catch(() => ({ code: -1, message: '网络异常', data: { items: [] } }));
			if (currentGid() !== gid) return;
			const announcementsData = announcementsRes.code === 0 ? announcementsRes.data : null;
			if (announcementsData) {
				announcements = (announcementsData.items || []).map((a) => ({
					id: a.announcement_id,
					title: a.title ?? '',
					body: a.content ?? '',
					time: formatDateTime(a.created_at),
					isPinned: Boolean(a.is_pinned)
				}));
			}
		} catch {
			// empty data is normal
		}
	}

	async function loadAll(gid: string) {
		const token = ++loadToken;
		loading = true;
		loadError = null;
		await Promise.all([loadDetail(gid), loadMembers(gid), loadAnnouncements(gid)]);
		if (token !== loadToken) return; // 已被更新的请求取代
		loading = false;
	}

	// 加入/退出：乐观更新 + 失败回滚（spec: groups/join、groups/leave）。
	// join 对需审核群组返回 pending_approval——UI 仍标记 joined 简化（真实后端
	// 以 my_state.is_joined 为准，审核通过前详情不会把它标为已加入）。
	function toggleJoin() {
		if (joinPending) return;
		joinPending = true;
		const next = !joined;
		joined = next;
		const done = (res: { code: number }) => {
			joinPending = false;
			if (res.code !== 0) {
				joined = !next;
			} else if (next) {
				// 加入成功：乐观置 member（my_state.role 在下次 loadDetail 时以服务端为准）
				myRole = 'member';
			}
		};
		const gid = currentGid();
		if (next) {
			void joinGroup(gid).then(done).catch(() => done({ code: -1 }));
		} else {
			void leaveGroup(gid).then(done).catch(() => done({ code: -1 }));
		}
	}

	// 公告管理（owner/admin；createGroupAnnouncement 自带幂等键）。成功后刷新公告列表（WS 也会推，幂等）。
	async function createAnnouncement(title: string, content: string, isPinned?: boolean) {
		const res = await createGroupAnnouncement({ group_id: currentGid(), title, content, is_pinned: isPinned });
		if (res.code === 0) void loadAnnouncements(currentGid());
		return res;
	}
	async function updateAnnouncement(announcementId: string, patch: { title?: string; content?: string; is_pinned?: boolean }) {
		const res = await updateGroupAnnouncement({ group_id: currentGid(), announcement_id: announcementId, ...patch });
		if (res.code === 0) void loadAnnouncements(currentGid());
		return res;
	}
	async function deleteAnnouncement(announcementId: string) {
		const res = await deleteGroupAnnouncement({ group_id: currentGid(), announcement_id: announcementId });
		if (res.code === 0) void loadAnnouncements(currentGid());
		return res;
	}

	// 初始加载 + 会话切换重载
	$effect(() => {
		const gid = opts.groupId();
		if (!gid) return;
		void loadAll(gid);
	});

	// WS 群组订阅：成员/详情/公告变更实时刷新；解散由调用方处理。
	$effect(() => {
		const gid = opts.groupId();
		if (!gid) return;

		wsClient.setSubscriptions('subscribe', [{ type: 'group', ids: [gid] }]);

		const isForThisGroup = (p: unknown) => (p as { group_id?: string })?.group_id === gid;
		const offs = [
			wsClient.on('GroupMemberJoined', (p) => {
				if (isForThisGroup(p)) void loadMembers(gid);
			}),
			wsClient.on('GroupMemberLeft', (p) => {
				if (isForThisGroup(p)) void loadMembers(gid);
			}),
			wsClient.on('GroupMemberRemoved', (p) => {
				if (isForThisGroup(p)) void loadMembers(gid);
			}),
			wsClient.on('GroupMemberMuted', (p) => {
				if (isForThisGroup(p)) void loadMembers(gid);
			}),
			wsClient.on('GroupMemberUnmuted', (p) => {
				if (isForThisGroup(p)) void loadMembers(gid);
			}),
			wsClient.on('GroupMemberRoleChanged', (p) => {
				if (isForThisGroup(p)) void loadMembers(gid);
			}),
			wsClient.on('GroupUpdated', (p) => {
				if (isForThisGroup(p)) void loadDetail(gid);
			}),
			wsClient.on('GroupAnnouncementCreated', (p) => {
				if (isForThisGroup(p)) void loadAnnouncements(gid);
			}),
			wsClient.on('GroupDismissed', (p) => {
				if (!isForThisGroup(p)) return;
				opts.onDismissed?.();
			})
		];

		return () => {
			for (const off of offs) off();
			wsClient.setSubscriptions('unsubscribe', [{ type: 'group', ids: [gid] }]);
			unwatchMembers?.();
			unwatchMembers = null;
		};
	});

	// 成员在线实时派生：绿点与"X 人在线"计数均以 presence 为准（HTTP 快照仅初始兜底）。
	const onlineMemberIds = $derived(
		new Set(members.filter((m) => opts.isOnline?.(m.userId) ?? false).map((m) => m.userId))
	);
	const onlineMemberCount = $derived(onlineMemberIds.size);

	return {
		get detail() { return detail; },
		get members() { return members; },
		get announcements() { return announcements; },
		get joined() { return joined; },
		get joinPending() { return joinPending; },
		get loading() { return loading; },
		get loadError() { return loadError; },
		get myRole() { return myRole; },
		get onlineMemberIds() { return onlineMemberIds; },
		get onlineMemberCount() { return onlineMemberCount; },
		loadAll,
		loadDetail,
		loadMembers,
		loadAnnouncements,
		createAnnouncement,
		updateAnnouncement,
		deleteAnnouncement,
		toggleJoin
	};
}
