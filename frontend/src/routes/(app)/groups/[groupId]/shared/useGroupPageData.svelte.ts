import { goto } from '$app/navigation';
import { ROUTES } from '@/lib/routes';
import { showToast } from '@/stores/toast.svelte';
import { useGroupDetail } from '@/lib/group-data.svelte';
import { onlineUsers, presence, watchPresence } from '@/stores/presence.svelte';
import { auth, requireAuth } from '@/stores/auth.svelte';
import { setGroupMemberRole, setGroupMemberMute, banGroupMember, removeGroupMember } from '@/api';
import type { MemberAction } from '@/components/groups/MemberActionSheet.svelte';
import type { UpdateGroupRequest } from '@/api/types';

export type GroupTab = 'overview' | 'members' | 'announcements';
export type MemberSubTab = 'members' | 'applications' | 'invites' | 'banned';

export interface UseGroupPageDataOptions {
	groupId: () => string;
}

export function useGroupPageData(options: UseGroupPageDataOptions) {
	let tab = $state<GroupTab>('overview');
	let memberTab = $state<MemberSubTab>('members');

	// 解散跳转防重入：HTTP 成功与 WS GroupDismissed 都可能触发 onDismissed
	let dismissedHandled = false;
	const gd = useGroupDetail({
		groupId: options.groupId,
		isAuthenticated: () => auth.isAuthenticated,
		onDismissed: () => {
			if (dismissedHandled) return;
			dismissedHandled = true;
			showToast('该群组已被解散', 'danger');
			void goto(ROUTES.nearbyGroups);
		},
		watchPresence,
		isOnline: (id) => presence.isOnline(id),
		seedPresence: (ids) => {
			for (const id of ids) onlineUsers.add(id);
		}
	});

	// 公告管理 UI 状态（owner/admin 门控）
	let announceOpen = $state(false);
	let announceMode = $state<'create' | 'edit'>('create');
	let announceEditing = $state<{ id: string; title: string; content: string; isPinned: boolean } | null>(null);
	let deleteAnnounceId = $state<string | null>(null);
	let announcementDeleting = $state(false);

	const canManageAnnouncements = $derived(gd.joined && (gd.myRole === 'owner' || gd.myRole === 'admin'));

	const openCreateAnnouncement = () => {
		announceMode = 'create';
		announceEditing = null;
		announceOpen = true;
	};

	const openEditAnnouncement = (item: { id: string; title: string; body: string; isPinned: boolean }) => {
		announceMode = 'edit';
		announceEditing = { id: item.id, title: item.title, content: item.body, isPinned: item.isPinned };
		announceOpen = true;
	};

	const handleAnnouncementSubmit = async (title: string, content: string, isPinned: boolean): Promise<boolean> => {
		if (announceMode === 'create') {
			const res = await gd.createAnnouncement(title, content, isPinned);
			if (res.code === 0) {
				showToast('公告已发布', 'success');
				return true;
			}
			showToast(res.message || '发布失败', 'danger');
			return false;
		} else if (announceEditing) {
			const res = await gd.updateAnnouncement(announceEditing.id, { title, content, is_pinned: isPinned });
			if (res.code === 0) {
				showToast('公告已更新', 'success');
				return true;
			}
			showToast(res.message || '保存失败', 'danger');
			return false;
		}
		return false;
	};

	const handleDeleteAnnouncement = async () => {
		if (!deleteAnnounceId || announcementDeleting) return;
		announcementDeleting = true;
		try {
			const res = await gd.deleteAnnouncement(deleteAnnounceId);
			if (res.code === 0) {
				showToast('公告已删除', 'success');
				deleteAnnounceId = null;
			} else {
				showToast(res.message || '删除失败', 'danger');
			}
		} catch {
			showToast('删除失败，请重试', 'danger');
		} finally {
			announcementDeleting = false;
		}
	};

	// 成员管理（owner/admin；不能操作群主/自己）
	const meId = $derived(auth.user?.user_id ?? '');
	const canManageMembers = $derived(gd.joined && (gd.myRole === 'owner' || gd.myRole === 'admin'));
	const canManageMember = (m: { userId: string; role: string }) =>
		canManageMembers && m.role !== 'owner' && m.userId !== meId;

	let actionMember = $state<{ userId: string; name: string; role: string; mutedUntil?: string } | null>(null);
	let inviteOpen = $state(false);

	const inviteStatusText = (s: string) =>
		({ pending: '等待接受', accepted: '已接受', rejected: '已拒绝', revoked: '已失效' })[s] ?? s;

	const handleRevokeInvite = async (inviteId: string) => {
		const res = await gd.revokeSentInvite(inviteId);
		if (res.code === 0) showToast('邀请已撤销', 'success');
		else showToast(res.message || '撤销失败', 'danger');
	};

	const handleUnban = async (userId: string) => {
		const res = await gd.unbanMember(userId);
		if (res.code === 0) showToast('已解封', 'success');
		else showToast(res.message || '解封失败', 'danger');
	};

	// 申请审批（owner/admin）
	let applicationPending = $state(new Set<string>());
	const handleApproveApplication = async (applicationId: string) => {
		if (applicationPending.has(applicationId)) return;
		applicationPending = new Set(applicationPending).add(applicationId);
		try {
			const res = await gd.approveApplication(applicationId);
			if (res.code === 0) showToast('已同意申请', 'success');
			else showToast(res.message || '操作失败', 'danger');
		} finally {
			const rest = new Set(applicationPending);
			rest.delete(applicationId);
			applicationPending = rest;
		}
	};

	const handleRejectApplication = async (applicationId: string) => {
		if (applicationPending.has(applicationId)) return;
		applicationPending = new Set(applicationPending).add(applicationId);
		try {
			const res = await gd.rejectApplication(applicationId);
			if (res.code === 0) showToast('已拒绝申请', 'success');
			else showToast(res.message || '操作失败', 'danger');
		} finally {
			const rest = new Set(applicationPending);
			rest.delete(applicationId);
			applicationPending = rest;
		}
	};

	let memberActionPending = $state(false);
	const handleMemberAction = async (action: MemberAction, durationMinutes?: number) => {
		if (!actionMember || memberActionPending) return;
		const m = actionMember;
		const gid = options.groupId();
		memberActionPending = true;
		try {
			let res: { code: number; message?: string } = { code: -1 };
			if (action === 'set_admin') res = await setGroupMemberRole({ group_id: gid, user_id: m.userId, role: 'admin' });
			else if (action === 'unset_admin') res = await setGroupMemberRole({ group_id: gid, user_id: m.userId, role: 'member' });
			else if (action === 'mute') res = await setGroupMemberMute({ group_id: gid, user_id: m.userId, active: true, duration_minutes: durationMinutes });
			else if (action === 'unmute') res = await setGroupMemberMute({ group_id: gid, user_id: m.userId, active: false });
			else if (action === 'ban') res = await banGroupMember({ group_id: gid, user_id: m.userId });
			else if (action === 'remove') res = await removeGroupMember({ group_id: gid, user_id: m.userId });

			if (res.code === 0) {
				const ACTION_TOAST: Record<MemberAction, string> = {
					set_admin: '已设为管理员',
					unset_admin: '已取消管理员',
					mute: '已禁言成员',
					unmute: '已解除禁言',
					ban: '已封禁成员',
					remove: '已移出成员'
				};
				showToast(ACTION_TOAST[action], 'success');
				void gd.loadMembers(gid);
				if (action === 'ban') void gd.loadBannedMembers(gid);
				actionMember = null;
			} else {
				showToast(res.message || '操作失败', 'danger');
			}
		} catch {
			showToast('操作失败，请重试', 'danger');
		} finally {
			memberActionPending = false;
		}
	};

	// 群设置编辑（owner/admin）
	let settingsOpen = $state(false);
	const handleGroupSettingsSubmit = async (patch: Omit<UpdateGroupRequest, 'group_id'>): Promise<boolean> => {
		const res = await gd.updateGroupSettings(patch);
		if (res.code === 0) {
			showToast('群设置已更新', 'success');
			return true;
		}
		showToast(res.message || '保存失败', 'danger');
		return false;
	};

	// 危险区（仅 owner）：转让群主 / 解散群组
	const handleTransferOwnership = async (userId: string) => {
		const res = await gd.transferOwnership(userId);
		if (res.code === 0) showToast('群主已转让', 'success');
		return res;
	};

	const handleDismissGroup = async () => {
		const res = await gd.dismissGroup();
		if (res.code === 0) showToast('群组已解散', 'success');
		return res;
	};

	// 加入 / 退出群组
	let joinPop = $state(0);
	const requestJoin = () => {
		const gid = options.groupId();
		if (!requireAuth({ action: '加入群组', targetId: gid, run: () => requestJoin() })) return;
		joinPop += 1;
		gd.toggleJoin();
	};

	let leaveConfirmOpen = $state(false);
	const handleLeaveConfirm = () => {
		gd.toggleJoin();
		leaveConfirmOpen = false;
	};

	// 更多操作菜单
	let moreOpen = $state(false);
	let reportOpen = $state(false);
	const canReportGroup = $derived(gd.myRole !== 'owner');

	return {
		gd,
		get tab() { return tab; },
		set tab(v: GroupTab) { tab = v; },
		get memberTab() { return memberTab; },
		set memberTab(v: MemberSubTab) { memberTab = v; },
		// 公告
		get announceOpen() { return announceOpen; },
		set announceOpen(v: boolean) { announceOpen = v; },
		get announceMode() { return announceMode; },
		get announceEditing() { return announceEditing; },
		get deleteAnnounceId() { return deleteAnnounceId; },
		set deleteAnnounceId(v: string | null) { deleteAnnounceId = v; },
		get announcementDeleting() { return announcementDeleting; },
		get canManageAnnouncements() { return canManageAnnouncements; },
		openCreateAnnouncement,
		openEditAnnouncement,
		handleAnnouncementSubmit,
		handleDeleteAnnouncement,
		// 成员
		get canManageMembers() { return canManageMembers; },
		canManageMember,
		get actionMember() { return actionMember; },
		set actionMember(v: { userId: string; name: string; role: string; mutedUntil?: string } | null) { actionMember = v; },
		get inviteOpen() { return inviteOpen; },
		set inviteOpen(v: boolean) { inviteOpen = v; },
		inviteStatusText,
		handleRevokeInvite,
		handleUnban,
		get applicationPending() { return applicationPending; },
		handleApproveApplication,
		handleRejectApplication,
		get memberActionPending() { return memberActionPending; },
		handleMemberAction,
		// 设置与权限
		get settingsOpen() { return settingsOpen; },
		set settingsOpen(v: boolean) { settingsOpen = v; },
		handleGroupSettingsSubmit,
		handleTransferOwnership,
		handleDismissGroup,
		// 加入/退出
		get joinPop() { return joinPop; },
		requestJoin,
		get leaveConfirmOpen() { return leaveConfirmOpen; },
		set leaveConfirmOpen(v: boolean) { leaveConfirmOpen = v; },
		handleLeaveConfirm,
		// 菜单与举报
		get moreOpen() { return moreOpen; },
		set moreOpen(v: boolean) { moreOpen = v; },
		get reportOpen() { return reportOpen; },
		set reportOpen(v: boolean) { reportOpen = v; },
		get canReportGroup() { return canReportGroup; }
	};
}
