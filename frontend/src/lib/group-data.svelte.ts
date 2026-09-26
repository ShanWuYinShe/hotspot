// 群详情业务工厂（收敛群详情页的状态/加载/WS/操作）。
// 参照 chat.svelte.ts 的依赖注入 + 竞态防护 + getter 范本；工厂零视图依赖
// （goto/ROUTES/presence 走 options 注入），保持纯净可测。
//
// 调用方式：组件 <script> 顶层 const gd = useGroupDetail({...})；工厂内的
// $state/$effect 生命周期绑定组件实例，卸载自动清理（WS 订阅 + presence 订阅）。

import { getGroupDetail, getGroupMembers, getGroupAnnouncements, joinGroup, leaveGroup, createGroupAnnouncement, updateGroupAnnouncement, deleteGroupAnnouncement, listSentGroupInvites, revokeSentGroupInvite, listBannedGroupMembers, unbanGroupMember, updateGroup, listGroupApplications, approveGroupApplication, rejectGroupApplication, transferGroupOwnership, dismissGroup as dismissGroupApi, listMyApplications, cancelGroupApplication, netError } from '@/api';
import { wsClient } from '@/lib/ws/client';
import { formatDistance, formatDateTime } from '@/lib/utils';
import { showToast } from '@/stores/toast.svelte';
import type { GroupRole, GroupSentInviteItem, AdminBannedMemberItem, GroupItem, UpdateGroupRequest, GroupApplicationItem, GroupMemberItem } from '@/api/types';

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
	presence: 'online' | 'offline' | null;
	mutedUntil?: string;
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
	/** 登录态判定（页面注入 auth.isAuthenticated）：未登录时跳过待审查询。 */
	isAuthenticated?: () => boolean;
	/** 成员快照在线状态播种（页面注入 onlineUsers.add：HTTP 快照 → WS 实时的既定升级路径）。 */
	seedPresence?: (ids: string[]) => void;
}

export function useGroupDetail(opts: UseGroupDetailOptions) {
	let detail = $state<GroupDetail>({ name: '', category: '', members: 0, online: 0, distance: '', description: '', banner: '' });
	/** 群详情原始响应（GroupItem），供设置编辑等场景回填；未加载成功为 null。 */
	let rawGroup = $state<GroupItem | null>(null);
	let members = $state<GroupMember[]>([]);
	let announcements = $state<GroupAnnouncement[]>([]);
	/** 已发邀请（owner/admin；API 原始项，页面直读字段） */
	let sentInvites = $state<GroupSentInviteItem[]>([]);
	/** 封禁名单（owner/admin；API 原始项） */
	let bannedMembers = $state<AdminBannedMemberItem[]>([]);
	/** 入群申请（owner/admin 工作台；默认仅 pending） */
	let applications = $state<GroupApplicationItem[]>([]);
	let joined = $state(false);
	/** 入群申请待审（需审核群）：joined=false 且此标记 true 时展示“审核中”而非“已加入”。 */
	let joinPendingApproval = $state(false);
	/** 待审申请 id（取消申请用；重载后经自查列表回填）。 */
	let pendingApplicationId = $state<string | null>(null);
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
			const detailRes = await getGroupDetail(gid).catch(() => netError(null));
			// 竞态防护：页面已切到别的群组则丢弃本次结果
			if (currentGid() !== gid) return;
			const detailData = detailRes.code === 0 ? detailRes.data : null;
			if (detailData) {
				rawGroup = detailData;
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
				if (detailData.viewer_state) {
					joined = true;
					joinPendingApproval = false;
					pendingApplicationId = null;
				} else {
					joined = false;
					// 非成员时回填待审态（重载后亦诚实）：仅登录用户查一次自查列表。
					if (opts.isAuthenticated?.()) void refreshPendingState(gid);
					else {
						joinPendingApproval = false;
						pendingApplicationId = null;
					}
				}
				myRole = detailData.viewer_state?.role ?? null;
			} else if (detailRes.code !== 0) {
				loadError = detailRes.message ?? '加载失败，请稍后重试';
			}
		} catch {
			loadError = '加载失败，请稍后重试';
		}
	}

	/** 成员分页（审查 M5）：大群此前一次全量拉取渲染；改 cursor 分页 + 手动加载更多。 */
	const MEMBERS_PAGE_SIZE = 50;
	let membersCursor = $state<string | null>(null);
	let membersLoadingMore = $state(false);

	async function loadMembers(gid: string) {
		try {
			const membersRes = await getGroupMembers(gid, { limit: MEMBERS_PAGE_SIZE }).catch(() => netError({ items: [], next_cursor: null }));
			if (currentGid() !== gid) return;
			const membersData = membersRes.code === 0 ? membersRes.data : null;
			if (membersData) {
				const items = toGroupMembers(membersData.items || [], 0);
				members = items;
				membersCursor = membersData.next_cursor ?? null;
				// 成员快照的在线状态播种进 presence store（后续 WS 事件继续覆盖）。
				const snapshotOnline = items.filter((m) => m.presence === 'online' && m.userId).map((m) => m.userId);
				if (snapshotOnline.length > 0) opts.seedPresence?.(snapshotOnline);
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

	function toGroupMembers(raw: GroupMemberItem[], offset: number): GroupMember[] {
		return raw.map((m, i) => ({
			id: offset + i + 1,
			userId: m.user?.user_id ?? '',
			name: m.user?.nickname ?? '',
			role: m.role ?? '成员',
			avatar: m.user?.avatar_url ?? '',
			presence: m.presence_status ?? null
			// 禁言截止时间在群管理视角（groups/admin/members/list，AdminGroupMemberItem）；
			// 公开 members/list 已不含 muted_until（视角分离）。
		}));
	}

	/** 是否还有更多成员（MembersList 据此渲染 LoadMoreButton）。 */
	function hasMoreMembers(): boolean {
		return membersCursor !== null;
	}

	function isLoadingMoreMembers(): boolean {
		return membersLoadingMore;
	}

	/** 加载下一页成员并追加（含 presence 播种与订阅扩容）。gid 取当前工厂
	 * 绑定的群（currentGid），翻页期间切群由竞态守卫丢弃迟到响应。 */
	async function loadMoreMembers() {
		const gid = currentGid();
		if (membersCursor === null || membersLoadingMore) return;
		membersLoadingMore = true;
		try {
			const res = await getGroupMembers(gid, { limit: MEMBERS_PAGE_SIZE, cursor: membersCursor }).catch(() => netError({ items: [], next_cursor: null }));
			if (currentGid() !== gid) return;
			if (res.code === 0 && res.data) {
				const items = toGroupMembers(res.data.items || [], members.length);
				members = [...members, ...items];
				membersCursor = res.data.next_cursor ?? null;
				const snapshotOnline = items.filter((m) => m.presence === 'online' && m.userId).map((m) => m.userId);
				if (snapshotOnline.length > 0) opts.seedPresence?.(snapshotOnline);
				// 订阅扩容：新页成员纳入 presence 实时订阅（复用既有 watch 句柄重建）。
				unwatchMembers?.();
				if (opts.watchPresence) {
					unwatchMembers = opts.watchPresence(members.map((m) => m.userId).filter(Boolean));
				}
			}
		} catch {
			// keep current page on failure
		} finally {
			membersLoadingMore = false;
		}
	}

	async function loadAnnouncements(gid: string) {
		try {
			const announcementsRes = await getGroupAnnouncements(gid).catch(() => netError({ items: [] }));
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

	async function loadSentInvites(gid: string) {
		try {
			const res = await listSentGroupInvites({ group_id: gid }).catch(() => netError({ items: [] }));
			if (currentGid() !== gid) return;
			if (res.code === 0) sentInvites = res.data?.items ?? [];
		} catch {
			// empty data is normal
		}
	}

	async function loadBannedMembers(gid: string) {
		try {
			const res = await listBannedGroupMembers({ group_id: gid }).catch(() => netError({ items: [] }));
			if (currentGid() !== gid) return;
			if (res.code === 0) bannedMembers = res.data?.items ?? [];
		} catch {
			// empty data is normal
		}
	}

	async function loadApplications(gid: string) {
		try {
			const res = await listGroupApplications({ group_id: gid }).catch(() => netError({ items: [] }));
			if (currentGid() !== gid) return;
			if (res.code === 0) applications = res.data?.items ?? [];
		} catch {
			// empty data is normal
		}
	}

	async function loadAll(gid: string) {
		const token = ++loadToken;
		loading = true;
		loadError = null;
		await Promise.all([loadDetail(gid), loadMembers(gid), loadAnnouncements(gid), loadSentInvites(gid), loadBannedMembers(gid), loadApplications(gid)]);
		if (token !== loadToken) return; // 已被更新的请求取代
		loading = false;
	}

	// 待审态回填：在自查列表中找本群 pending 申请（重载/外跳返回后亦诚实）。
	async function refreshPendingState(gid: string) {
		try {
			const res = await listMyApplications().catch(() => netError(undefined));
			if (currentGid() !== gid) return;
			const hit = res.code === 0
				? (res.data?.items ?? []).find((a) => a.group?.group_id === gid && a.status === 'pending')
				: undefined;
			if (hit) {
				joinPendingApproval = true;
				pendingApplicationId = hit.application_id;
			} else {
				joinPendingApproval = false;
				pendingApplicationId = null;
			}
		} catch {
			// 查询失败保持现状（按钮仍为加入群组，可重试）。
		}
	}

	// 加入/退出：乐观更新 + 失败回滚（spec: groups/enter、groups/exit）。
	// 需审核群返回 pending_approval 时进入待审态（审核中·可取消），不再冒充已加入。
	function toggleJoin() {
		if (joinPending) return;
		joinPending = true;
		const next = !joined;
		joined = next;
		const done = (res: { code: number; message?: string; data?: unknown }) => {
			joinPending = false;
			if (res.code !== 0) {
				joined = !next;
				// 失败必须反馈（此前静默回滚：群主退群被 10400「群主需先转让群主身份」
				// 拒绝时按钮闪一下回弹，用户完全不知道原因）
				showToast(res.message || '操作失败', 'danger');
			} else if (next) {
				const joinData = res.data as { status?: string; application_id?: string | null } | null | undefined;
				if (joinData?.status === 'pending_approval') {
					// 待审：纠正乐观的 joined，记录申请 id 供取消。
					joined = false;
					joinPendingApproval = true;
					pendingApplicationId = joinData.application_id ?? null;
					showToast('申请已提交，待群主审核', 'success');
				} else {
					// 加入成功：乐观置 member（my_state.role 在下次 loadDetail 时以服务端为准）
					myRole = 'member';
				}
			}
		};
		const gid = currentGid();
		if (next) {
			void joinGroup(gid).then(done).catch(() => done(netError(undefined)));
		} else {
			void leaveGroup(gid).then(done).catch(() => done(netError(undefined)));
		}
	}

	// 取消入群申请（待审态）：成功后回到未加入态，可重新申请。
	async function cancelJoinApplication() {
		if (!pendingApplicationId) return { code: -1 };
		const gid = currentGid();
		const appId = pendingApplicationId;
		const res = await cancelGroupApplication({ group_id: gid, application_id: appId }).catch(() => netError(undefined));
		if (res.code === 0) {
			joinPendingApproval = false;
			pendingApplicationId = null;
			showToast('已取消申请', 'success');
		} else {
			showToast(res.message || '取消失败', 'danger');
		}
		return res;
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

	// 已发邀请管理（owner/admin）：撤销后刷新列表（受邀人可能已处理，以服务端为准）。
	async function revokeSentInvite(inviteId: string) {
		const res = await revokeSentGroupInvite({ group_id: currentGid(), invite_id: inviteId });
		if (res.code === 0) void loadSentInvites(currentGid());
		return res;
	}
	// 封禁名单管理（owner/admin）：解封后刷新名单。
	async function unbanMember(userId: string) {
		const res = await unbanGroupMember({ group_id: currentGid(), user_id: userId });
		if (res.code === 0) void loadBannedMembers(currentGid());
		return res;
	}
	// 群设置编辑（owner/admin）：更新后刷新详情（响应即最新 GroupItem，rawGroup 随之更新）。
	async function updateGroupSettings(patch: Omit<UpdateGroupRequest, 'group_id'>) {
		const res = await updateGroup({ group_id: currentGid(), ...patch });
		if (res.code === 0) void loadDetail(currentGid());
		return res;
	}
	// 申请审批（owner/admin）：通过后刷新申请列表 + 成员列表（新成员已入群）；
	// 拒绝带 reason（spec 0-200 字符，随通知送达申请人）。
	async function approveApplication(applicationId: string) {
		const res = await approveGroupApplication({ group_id: currentGid(), application_id: applicationId });
		if (res.code === 0) {
			void loadApplications(currentGid());
			void loadMembers(currentGid());
		}
		return res;
	}
	async function rejectApplication(applicationId: string, reason?: string) {
		const res = await rejectGroupApplication({ group_id: currentGid(), application_id: applicationId, reason });
		if (res.code === 0) void loadApplications(currentGid());
		return res;
	}
	// 转让群主（owner）：成功后刷新详情（myRole 降级）与成员列表（角色变化）。
	async function transferOwnership(newOwnerId: string) {
		const res = await transferGroupOwnership({ group_id: currentGid(), new_owner_id: newOwnerId });
		if (res.code === 0) {
			void loadDetail(currentGid());
			void loadMembers(currentGid());
		}
		return res;
	}
	// 解散群组（owner，危险操作）：HTTP 成功后立即走 onDismissed（页面 alert + 跳转）；
	// 真实后端还会推 GroupDismissed（组件卸载后订阅自动清理，页面侧已做防重入）。
	async function dismissGroup(reason?: string) {
		const res = await dismissGroupApi({ group_id: currentGid(), reason });
		if (res.code === 0) opts.onDismissed?.();
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
			// 受邀人接受/拒绝 → 推给邀请发起方（管理员），刷新已发邀请列表
			wsClient.on('GroupInviteResponse', (p) => {
				if (isForThisGroup(p)) void loadSentInvites(gid);
			}),
			wsClient.on('GroupInviteRejected', (p) => {
				if (isForThisGroup(p)) void loadSentInvites(gid);
			}),
			// 新申请到达 → 推给群管理员，刷新申请列表
			wsClient.on('GroupApplicationReceived', (p) => {
				if (isForThisGroup(p)) void loadApplications(gid);
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

	// 成员在线实时派生：绿点以 presence 为准（含快照播种）。
	// 「X 人在线」计数口径：服务端快照（detail.online，全群真值——成员列表分页
	// 加载不全，实时 presence 只能覆盖已加载成员）优先；快照缺失回退实时计数。
	const onlineMemberIds = $derived(
		new Set(members.filter((m) => opts.isOnline?.(m.userId) ?? false).map((m) => m.userId))
	);
	const onlineMemberCount = $derived(detail.online > 0 ? detail.online : onlineMemberIds.size);

	return {
		get detail() { return detail; },
		get rawGroup() { return rawGroup; },
		get members() { return members; },
		get announcements() { return announcements; },
		get sentInvites() { return sentInvites; },
		get bannedMembers() { return bannedMembers; },
		get applications() { return applications; },
		get joined() { return joined; },
		get joinPending() { return joinPending; },
		get joinPendingApproval() { return joinPendingApproval; },
		get loading() { return loading; },
		get loadError() { return loadError; },
		get myRole() { return myRole; },
		get onlineMemberIds() { return onlineMemberIds; },
		get onlineMemberCount() { return onlineMemberCount; },
		loadAll,
		loadDetail,
		loadMembers,
		loadBannedMembers,
		loadSentInvites,
		hasMoreMembers,
		isLoadingMoreMembers,
		loadMoreMembers,
		loadAnnouncements,
		createAnnouncement,
		updateAnnouncement,
		deleteAnnouncement,
		revokeSentInvite,
		cancelJoinApplication,
		unbanMember,
		updateGroupSettings,
		approveApplication,
		rejectApplication,
		transferOwnership,
		dismissGroup,
		toggleJoin
	};
}
