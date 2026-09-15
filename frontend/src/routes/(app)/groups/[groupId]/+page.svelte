<script lang="ts">
	import { goto } from '$app/navigation';
	import { ArrowLeft, Bell, Check, LogOut, Megaphone, MessageCircle, MoreHorizontal, PencilLine, Pin, Settings, UserPlus } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';
	import GroupAnnouncementDialog from '@/components/groups/GroupAnnouncementDialog.svelte';
	import MembersList from './shared/MembersList.svelte';
	import ApplicationsList from './shared/ApplicationsList.svelte';
	import InvitesList from './shared/InvitesList.svelte';
	import BannedList from './shared/BannedList.svelte';
	import AnnouncementsList from './shared/AnnouncementsList.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import { ROUTES } from '@/lib/routes';
	import { useGroupDetail } from '@/lib/group-data.svelte';
	import { formatDateTime } from '@/lib/utils';
	import { onlineUsers, presence, watchPresence } from '@/stores/presence.svelte';
	import { auth } from '@/stores/auth.svelte';
	import { setGroupMemberRole, setGroupMemberMute, banGroupMember, removeGroupMember } from '@/api';
	import MemberActionSheet from '@/components/groups/MemberActionSheet.svelte';
	import InviteMembersSheet from '@/components/groups/InviteMembersSheet.svelte';
	import GroupSettingsSheet from '@/components/groups/GroupSettingsSheet.svelte';
	import type { MemberAction } from '@/components/groups/MemberActionSheet.svelte';
	import type { UpdateGroupRequest } from '@/api/types';
	import { smartBack } from '@/lib/smart-back';

	// 群组详情：封面 + 简介/成员/公告三 Tab + 加入按钮 + 桌面侧栏。
	// 状态/加载/WS/操作收敛到 useGroupDetail 工厂；本页只保留 UI + 角色门控。
	// WS 实时：工厂订阅 group 频道，成员/详情/公告变更触发对应刷新；解散由 onDismissed 处理。

	let { params }: { params: { groupId: string } } = $props();

	type GroupTab = 'overview' | 'members' | 'announcements';
	let tab = $state<GroupTab>('overview');

	// 解散跳转防重入：HTTP 成功与 WS GroupDismissed 都可能触发 onDismissed
	let dismissedHandled = false;
	const gd = useGroupDetail({
		groupId: () => params.groupId,
		onDismissed: () => {
			if (dismissedHandled) return;
			dismissedHandled = true;
			// 全项目统一无原生对话框（UI 一致性：解散通知走 Toast）
			showToast('该群组已被解散', 'danger');
			void goto(ROUTES.nearbyGroups);
		},
		watchPresence,
		isOnline: (id) => presence.isOnline(id),
		// 成员快照在线状态播种（HTTP 快照 → WS 实时升级路径）
		seedPresence: (ids) => {
			for (const id of ids) onlineUsers.add(id);
		}
	});

	// 公告管理 UI 状态（owner/admin 门控）
	let announceOpen = $state(false);
	let announceMode = $state<'create' | 'edit'>('create');
	let announceEditing = $state<{ id: string; title: string; content: string; isPinned: boolean } | null>(null);
	let deleteAnnounceId = $state<string | null>(null);
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
			if (res.code === 0) { showToast('公告已发布', 'success'); return true; }
			showToast(res.message || '发布失败', 'danger'); return false;
		} else if (announceEditing) {
			const res = await gd.updateAnnouncement(announceEditing.id, { title, content, is_pinned: isPinned });
			if (res.code === 0) { showToast('公告已更新', 'success'); return true; }
			showToast(res.message || '保存失败', 'danger'); return false;
		}
		return false;
	};
	const handleDeleteAnnouncement = async () => {
		if (!deleteAnnounceId) return;
		const res = await gd.deleteAnnouncement(deleteAnnounceId);
		if (res.code === 0) showToast('公告已删除', 'success');
		else showToast(res.message || '删除失败', 'danger');
		deleteAnnounceId = null;
	};

	// 成员管理（owner/admin；不能操作群主/自己）
	const meId = $derived(auth.user?.user_id ?? '');
	const canManageMembers = $derived(gd.joined && (gd.myRole === 'owner' || gd.myRole === 'admin'));
	const canManageMember = (m: { userId: string; role: string }) =>
		canManageMembers && m.role !== 'owner' && m.userId !== meId;
	let actionMember = $state<{ userId: string; name: string; role: string; mutedUntil?: string } | null>(null);
	let inviteOpen = $state(false);

	// 成员 Tab 子区（owner/admin 可见：成员/邀请管理/封禁名单）
		type MemberSubTab = 'members' | 'applications' | 'invites' | 'banned';
	let memberTab = $state<MemberSubTab>('members');
	// 邀请状态文案（契约：revoked 含过期自动撤销，统一展示"已失效"，与 rejected 区分）
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

	// 群设置编辑（owner/admin；入口在侧栏「群管理」）
	let settingsOpen = $state(false);
	const handleGroupSettingsSubmit = async (patch: Omit<UpdateGroupRequest, 'group_id'>): Promise<boolean> => {
		const res = await gd.updateGroupSettings(patch);
		if (res.code === 0) { showToast('群设置已更新', 'success'); return true; }
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

	// 退出群组需确认（危险操作确认范式，参照删除公告 Dialog）；加入无需确认
	let leaveConfirmOpen = $state(false);
	const handleLeaveConfirm = () => {
		gd.toggleJoin(); // 当前 joined=true，toggleJoin 执行离开分支
		leaveConfirmOpen = false;
	};

	// 申请审批（owner/admin）：同意后成员列表由工厂同步刷新
	const handleApproveApplication = async (applicationId: string) => {
		const res = await gd.approveApplication(applicationId);
		if (res.code === 0) showToast('已同意申请', 'success');
		else showToast(res.message || '操作失败', 'danger');
	};
	const handleRejectApplication = async (applicationId: string) => {
		const res = await gd.rejectApplication(applicationId);
		if (res.code === 0) showToast('已拒绝申请', 'success');
		else showToast(res.message || '操作失败', 'danger');
	};

	const handleMemberAction = async (action: MemberAction, durationMinutes?: number) => {
		if (!actionMember) return;
		const m = actionMember;
		const gid = params.groupId;
		try {
			let res: { code: number; message?: string } = { code: -1 };
			if (action === 'set_admin') res = await setGroupMemberRole({ group_id: gid, user_id: m.userId, role: 'admin' });
			else if (action === 'unset_admin') res = await setGroupMemberRole({ group_id: gid, user_id: m.userId, role: 'member' });
			else if (action === 'mute') res = await setGroupMemberMute({ group_id: gid, user_id: m.userId, active: true, duration_minutes: durationMinutes });
			else if (action === 'unmute') res = await setGroupMemberMute({ group_id: gid, user_id: m.userId, active: false });
			else if (action === 'ban') res = await banGroupMember({ group_id: gid, user_id: m.userId });
			else if (action === 'remove') res = await removeGroupMember({ group_id: gid, user_id: m.userId });
			if (res.code === 0) {
				showToast('操作成功', 'success');
				void gd.loadMembers(gid);
				actionMember = null;
			} else {
				showToast(res.message || '操作失败', 'danger');
			}
		} catch {
			showToast('操作失败，请重试', 'danger');
		}
	};
</script>
<svelte:head>
	<title>群组主页 · HotSpot</title>
</svelte:head>


<StandardPageLayout contentClassName="animate-fade-in pb-6">
	<!-- 错误态收敛进 layout children（与正文同 gutter，不再偏移） -->
	{#if gd.loadError}
		<LoadError message={gd.loadError} onRetry={() => void gd.loadAll(params.groupId ?? '')} />
	{:else}
	<!-- 全幅封面（负 margin 抵消 ContentColumn gutter） -->
	<div class="relative h-cover-height overflow-hidden -mx-page-gutter">
		{#if gd.detail.banner}
			<img src={gd.detail.banner} alt={`${gd.detail.name} 的群组封面`} class="h-full w-full object-cover" />
		{:else}
			<div class="h-full w-full bg-accent-bg"></div>
		{/if}
		<!-- gradient scrim 走 inline background（gradient 不能作 background-color 值，
		     与 NearbyImageCard 的 --cst-surface-media-scrim-card 用法一致） -->
		<div class="absolute inset-0" style="background:var(--cst-surface-cover-scrim);"></div>
		<div class="absolute left-5 top-4">
			<ActionIconButton icon={ArrowLeft} label="返回" onClick={() => smartBack()} variant="overlay" tone="inverse" />
		</div>
		<div class="absolute bottom-5 left-6 right-6">
			<span class="inline-flex h-6 items-center justify-center rounded-full bg-surface-glass-badge px-2.5 text-xs font-medium leading-none text-content-inverse backdrop-blur-[var(--cst-blur-sm)]">{gd.detail.category}</span>
			<h1 class="mt-3 text-2xl md:text-3xl font-medium tracking-title text-content-inverse">{gd.detail.name}</h1>
		</div>
	</div>

	<div class="grid gap-8 pt-6 lg:grid-cols-[minmax(0,1fr)_20rem]">
		<main class="min-w-0">
			<!-- 元信息（成员/在线/距你）不再在此重复展示：简介 tab 四宫格数据卡承载
			     同一份数据（大版优先），与鸿蒙 GroupDetailPage 保持一致 -->

			<!-- 分类 Tab（统一 TabBar 胶囊样式；pb-3 让胶囊与 border-b 分离，对齐 PageHeader below 约定） -->
			<div class="mt-4 pb-3 border-b border-border-default">
				<TabBar
					items={[
						{ id: 'overview', label: '简介' },
						{ id: 'members', label: '成员' },
						{ id: 'announcements', label: '公告' }
					]}
					value={tab}
					onChange={(id) => (tab = id as GroupTab)}
				/>
			</div>

			{#if tab === 'overview'}
				<div class="py-6 flex flex-col gap-6">
					<p class="max-w-3xl text-sm leading-7 text-text-secondary">{gd.detail.description || '这个群还没有填写简介。'}</p>

					<!-- 群组档案：数字档案 + 兴趣标签，填充简介页信息密度 -->
					<div class="grid grid-cols-2 gap-3 max-w-3xl sm:grid-cols-4">
						<div class="rounded-lg border border-border-default bg-surface-primary p-3.5">
							<p class="text-xl font-semibold text-text-primary">{gd.detail.members}</p>
							<p class="mt-0.5 text-xs text-text-secondary">成员</p>
						</div>
						<div class="rounded-lg border border-border-default bg-surface-primary p-3.5">
							<p class="text-xl font-semibold text-status-online">{gd.detail.online}</p>
							<p class="mt-0.5 text-xs text-text-secondary">在线</p>
						</div>
						<div class="rounded-lg border border-border-default bg-surface-primary p-3.5">
							<p class="text-xl font-semibold text-text-primary">{gd.detail.distance || '—'}</p>
							<p class="mt-0.5 text-xs text-text-secondary">距你</p>
						</div>
						<div class="rounded-lg border border-border-default bg-surface-primary p-3.5">
							<p class="text-xl font-semibold text-text-primary">{gd.detail.category || '—'}</p>
							<p class="mt-0.5 text-xs text-text-secondary">分类</p>
						</div>
					</div>

					{#if gd.members.length > 0}
						<!-- 成员预览：头像堆叠 + 成员数，引导到成员 Tab -->
						<button
							type="button"
							class="hover-btn w-fit flex items-center gap-3 rounded-lg border border-border-default bg-surface-primary px-4 py-3 focus-ring transition-colors hover:bg-interactive-hover"
							onclick={() => (tab = 'members')}
						>
							<div class="flex -space-x-2">
								{#each gd.members.slice(0, 6) as m, i (m.userId ?? i)}
									<!-- Avatar 原语：自带懒加载与 initials 兜底（审查 M6，替换裸 img） -->
									<Avatar src={m.avatar} name={m.name} size="md" class="rounded-full border-2 border-surface-primary object-cover" />
								{/each}
							</div>
							<span class="text-sm text-text-primary">{gd.detail.members} 位成员 · 查看全部</span>
						</button>
					{/if}
				</div>
				{:else if tab === 'members'}
					{#if canManageMembers}
						<!-- 成员 Tab 子区（owner/admin）：成员 / 申请审批 / 邀请管理 / 封禁名单 -->
						<div class="mt-4 pb-3 border-b border-border-default">
							<TabBar
								items={[
									{ id: 'members', label: '成员' },
									{ id: 'applications', label: '申请审批' },
									{ id: 'invites', label: '邀请管理' },
									{ id: 'banned', label: '封禁名单' }
								]}
								value={memberTab}
								onChange={(id) => (memberTab = id as MemberSubTab)}
							/>
						</div>
					{/if}
				{#if !canManageMembers || memberTab === 'members'}
					<MembersList
						members={gd.members}
						onlineMemberIds={gd.onlineMemberIds}
						canManageMembers={canManageMembers}
						canManageMember={canManageMember}
						hasMore={gd.hasMoreMembers()}
						loadingMore={gd.isLoadingMoreMembers()}
						onInvite={() => (inviteOpen = true)}
						onManage={(m) => (actionMember = m)}
						onLoadMore={() => void gd.loadMoreMembers()}
					/>
					{:else if memberTab === 'applications'}
						<ApplicationsList
							applications={gd.applications}
						{formatDateTime}
						onApprove={(id) => void handleApproveApplication(id)}
						onReject={(id) => void handleRejectApplication(id)}
					/>
					{:else if memberTab === 'invites'}
					<InvitesList
						invites={gd.sentInvites}
						{inviteStatusText}
						{formatDateTime}
						onRevoke={(id) => void handleRevokeInvite(id)}
					/>
				{:else}
					<BannedList
						banned={gd.bannedMembers}
						{formatDateTime}
						onUnban={(id) => void handleUnban(id)}
					/>
				{/if}
			{:else}
				<AnnouncementsList
					announcements={gd.announcements}
					canManage={canManageAnnouncements}
					onCreate={openCreateAnnouncement}
					onEdit={openEditAnnouncement}
					onDelete={(id) => (deleteAnnounceId = id)}
				/>
			{/if}
		</main>

		<aside class="flex h-fit flex-col gap-3 rounded-lg border border-border-default bg-surface-primary p-4">
			<!-- 已加入态 hover 只变底（不换字色，单信号 feedback） -->
			<button
				type="button"
				class="hover-btn w-full rounded-full border px-4 py-3 text-sm font-medium focus-ring {gd.joined ? 'border-accent-border text-accent-default hover:bg-interactive-hover' : 'border-transparent bg-accent-default text-text-on-accent shadow-raised-sm hover:shadow-raised-md'}"
				onclick={() => (gd.joined ? (leaveConfirmOpen = true) : gd.toggleJoin())}
			>
				{#if gd.joined}
					<span class="flex items-center justify-center gap-1.5">
						<Check size={15} />
						已加入 · 点击退出
					</span>
				{:else}
					加入群组
				{/if}
			</button>
				{#if gd.joined && (gd.myRole === 'owner' || gd.myRole === 'admin')}
					<!-- 角色门控：仅群主/管理员可见的管理入口（群设置编辑；转让/解散为批次 4d） -->
					<button
						type="button"
						onclick={() => (settingsOpen = true)}
						class="hover-btn flex w-full items-center justify-center gap-2 rounded-full border border-border-default px-4 py-3 text-sm text-text-primary focus-ring"
					>
						<Settings size={15} />
						群管理
					</button>
				{/if}
			{#if gd.joined}
				<button type="button" onclick={() => void goto(ROUTES.groupMessage(params.groupId))} class="hover-btn flex w-full items-center justify-center gap-2 rounded-full border border-border-default px-4 py-3 text-sm text-text-primary focus-ring">
					<MessageCircle size={15} />
					打开群聊
				</button>
			{:else}
				<p class="px-2 text-center text-xs leading-5 text-text-secondary">
					<Bell size={13} class="mr-1 inline" />
					加入后可管理群组通知并进入群聊
				</p>
			{/if}
		</aside>
	</div>
	{/if}
	</StandardPageLayout>

<ConfirmDialog
	open={deleteAnnounceId !== null}
	title="删除这条公告？"
	description="删除后无法恢复。"
	confirmLabel="确认删除"
	onCancel={() => (deleteAnnounceId = null)}
	onConfirm={handleDeleteAnnouncement}
/>

<ConfirmDialog
	open={leaveConfirmOpen}
	title="退出群组「{gd.detail.name}」？"
	description="退出后将不再接收该群的聊天与公告。"
	icon={LogOut}
	confirmLabel="确认退出"
	onCancel={() => (leaveConfirmOpen = false)}
	onConfirm={handleLeaveConfirm}
/>

<GroupAnnouncementDialog
	open={announceOpen}
	mode={announceMode}
	initial={announceEditing ?? undefined}
	onClose={() => (announceOpen = false)}
	onSubmit={handleAnnouncementSubmit}
/>

<MemberActionSheet
	open={actionMember !== null}
	member={actionMember}
	onClose={() => (actionMember = null)}
	onSelect={handleMemberAction}
/>

<InviteMembersSheet
	open={inviteOpen}
	groupId={params.groupId}
	onClose={() => (inviteOpen = false)}
	onInvited={() => void gd.loadMembers(params.groupId)}
/>

<GroupSettingsSheet
	open={settingsOpen}
	initial={gd.rawGroup}
	isOwner={gd.myRole === 'owner'}
	members={gd.members}
	onClose={() => (settingsOpen = false)}
	onSubmit={handleGroupSettingsSubmit}
	onTransferOwnership={handleTransferOwnership}
	onDismiss={handleDismissGroup}
/>
