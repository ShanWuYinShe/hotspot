<script lang="ts">
import { formatCount } from '@/lib/utils';
	import { goto } from '$app/navigation';
	import { ArrowLeft, Bell, Check, ChevronRight, Flag, LogOut, Megaphone, MessageCircle, MoreHorizontal, PencilLine, Pin, Settings, UserPlus, Users, X } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import MetricRow from '@/components/data-display/MetricRow.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';
	import ReportDialog from '@/components/overlays/ReportDialog.svelte';
	import Button from '@/components/ui/button.svelte';
	import * as Dialog from '@/components/ui/dialog';
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
	import { auth, requireAuth } from '@/stores/auth.svelte';
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
		isAuthenticated: () => auth.isAuthenticated,
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

	// 加入群组是写操作：访客必须先登录。此前直接调 toggleJoin 会打到 groups/enter
	// 拿到 10100「Access Token 无效或过期」技术错误 toast，而同页的关注/点赞等都弹
	// 登录层（全站复审 s2-02）。requireAuth 会存下 pending action，登录成功后自动重放。
	const requestJoin = () => {
		if (!requireAuth({ action: '加入群组', targetId: params.groupId, run: () => requestJoin() })) return;
		gd.toggleJoin();
	};

	// 退出群组需确认（危险操作确认范式，参照删除公告 Dialog）；加入无需确认
	let leaveConfirmOpen = $state(false);
	const handleLeaveConfirm = () => {
		gd.toggleJoin(); // 当前 joined=true，toggleJoin 执行离开分支
		leaveConfirmOpen = false;
	};

	// 更多操作菜单（封面右上角）：举报群组（spec: api_spec_report.md；群主举报自己的群
	// 服务端 10400，故群主不展示入口）
	let moreOpen = $state(false);
	let reportOpen = $state(false);
	const canReportGroup = $derived(gd.myRole !== 'owner');

	// 申请审批（owner/admin）：同意后成员列表由工厂同步刷新
	// 审批防重复提交（对齐 notify 页 invitePending 范式）：连点双发会产生
	// 第二次「申请不存在」的失败提示甚至重复入群。
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
				// 封禁后同步刷新封禁名单（此前只刷成员列表，切到名单 tab 仍是旧数据）。
				if (action === 'ban') void gd.loadBannedMembers(gid);
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


<StandardPageLayout contentClassName="animate-fade-in pb-6 max-w-6xl xl:max-w-7xl 2xl:max-w-[88rem] mx-auto w-full">
	<!-- 错误态收敛进 layout children（与正文同 gutter，不再偏移） -->
	{#if gd.loading}
		<InlineLoading label="加载群组…" />
	{:else if gd.loadError}
		<LoadError message={gd.loadError} onRetry={() => void gd.loadAll(params.groupId ?? '')} />
	{:else}
	<!-- 现代高质感社群 Hero 封面（圆角温润，磨砂悬浮操作区） -->
	<!-- Hero 封面（例外登记）：去 1px 描边，保留圆角/阴影/加载占位；图片 object-cover -->
	<div class="relative h-cover-height overflow-hidden rounded-2xl shadow-raised-sm bg-surface-subtle">
		{#if gd.detail.banner}
			<img src={gd.detail.banner} alt={`${gd.detail.name} 的群组封面`} class="h-full w-full object-cover transition-transform duration-standard hover:scale-[1.01]" />
		{:else}
			<div class="h-full w-full bg-accent-soft/40 flex items-center justify-center">
				<Users size={48} class="text-accent-default/30" strokeWidth={1.5} />
			</div>
		{/if}
		<!-- 沉浸式暗角天光渐变 -->
		<div class="absolute inset-0" style="background:var(--cst-surface-cover-scrim);"></div>
		<div class="absolute left-4 top-4">
			<ActionIconButton icon={ArrowLeft} label="返回" onClick={() => smartBack()} variant="overlay" tone="inverse" size={42} />
		</div>
		<div class="absolute right-4 top-4">
			<ActionIconButton icon={MoreHorizontal} label="更多操作" onClick={() => (moreOpen = true)} variant="overlay" tone="inverse" size={42} />
		</div>
		<div class="absolute bottom-5 left-5 right-5">
			{#if gd.detail.category}
				<span class="inline-flex h-6 items-center justify-center rounded-full bg-surface-media-scrim-strong px-3 text-xs font-medium leading-none text-content-inverse backdrop-blur-md shadow-xs">
					{gd.detail.category}
				</span>
			{/if}
			<h1 class="mt-2.5 text-2xl md:text-3xl font-bold tracking-title text-content-inverse drop-shadow-sm">{gd.detail.name}</h1>
		</div>
	</div>

	<div class="grid gap-y-6 pt-6 lg:grid-cols-[minmax(0,1fr)_20rem] lg:gap-x-8">
		<div class="min-w-0">
			<!-- 分类 Tab（胶囊样式，轻柔微发丝底线） -->
			<div class="pb-3">
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
				<div class="pt-5 flex flex-col gap-5">
					<!-- 去页级外框（Round 2）：概览直接落在连续画布上（原白卡+描边读起来像把整页内容框住）；
					     内部指标/成员块仍作为内容单元保留浅底。 -->
					<section class="flex flex-col gap-5" aria-label="群组概览">
						<!-- 描述简介 -->
						<p class="text-sm md:text-base leading-relaxed text-text-primary whitespace-pre-wrap break-words">
							{gd.detail.description || '这个群还没有填写简介。'}
						</p>

						<!-- 社群活力 Bento 矩阵（四宫格通透指标） -->
						<div class="grid grid-cols-2 sm:grid-cols-4 gap-3 pt-1">
							<div class="flex flex-col p-3 rounded-lg bg-surface-subtle/60">
								<span class="text-xs text-text-secondary font-medium">群成员</span>
								<span class="text-lg font-bold text-text-primary mt-1">{formatCount(gd.detail.members)}</span>
							</div>
							<div class="flex flex-col p-3 rounded-lg bg-surface-subtle/60">
								<span class="text-xs text-text-secondary font-medium flex items-center gap-1.5">
									<span>在线</span>
									<span class="size-1.5 rounded-full bg-status-online animate-pulse"></span>
								</span>
								<span class="text-lg font-bold text-status-online mt-1">{gd.detail.online}</span>
							</div>
							<div class="flex flex-col p-3 rounded-lg bg-surface-subtle/60">
								<span class="text-xs text-text-secondary font-medium">距你</span>
								<span class="text-lg font-bold text-text-primary mt-1">{gd.detail.distance || '—'}</span>
							</div>
							<div class="flex flex-col p-3 rounded-lg bg-surface-subtle/60">
								<span class="text-xs text-text-secondary font-medium">分类</span>
								<span class="text-lg font-bold text-text-primary mt-1">{gd.detail.category || '—'}</span>
							</div>
						</div>

						<!-- 活跃成员画廊与跳转入廊（融为一体，去除外层割裂白框） -->
						{#if gd.members.length > 0}
							<button
								type="button"
								class="hover-btn flex items-center justify-between p-3.5 rounded-lg bg-surface-subtle/40 focus-ring transition-all hover:bg-interactive-hover group mt-1"
								onclick={() => (tab = 'members')}
							>
								<div class="flex items-center gap-3 min-w-0">
									<div class="flex -space-x-2 shrink-0">
										{#each gd.members.slice(0, 6) as m, i (m.userId ?? i)}
											<Avatar src={m.avatar} name={m.name} size="md" class="rounded-full border-2 border-surface-primary object-cover shadow-xs" />
										{/each}
									</div>
									<span class="text-sm font-medium text-text-primary truncate">{formatCount(gd.detail.members)} 位成员在群中</span>
								</div>
								<span class="text-xs text-accent-default font-medium flex items-center gap-1 shrink-0 group-hover:translate-x-0.5 transition-transform">
									查看全部成员
									<ChevronRight class="size-3.5" strokeWidth={1.75} />
								</span>
							</button>
						{/if}
					</section>
				</div>
				{:else if tab === 'members'}
					{#if canManageMembers}
						<!-- 成员 Tab 子区（owner/admin）：成员 / 申请审批 / 邀请管理 / 封禁名单 -->
						<div class="mt-6 pb-3">
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
						pending={applicationPending}
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
		</div>

		<!-- 桌面右栏用圆润卡片承载；移动端平滑自适应，主次操作分明 -->
		<section aria-label="群组操作" class="flex h-fit flex-col gap-3 rounded-lg bg-surface-island p-5 shadow-xs">
			{#if gd.joined}
				<Button
					type="button"
					variant="ghost"
					size="lg"
					flat
					class="w-full bg-surface-subtle/60 text-accent-default hover:bg-surface-subtle"
					onclick={() => {
						if (gd.joinPendingApproval) {
							void gd.cancelJoinApplication();
							return;
						}
						if (gd.joined && gd.myRole === 'owner') {
							showToast('群主需先转让群主身份（群管理 → 转让群主）', 'danger');
							return;
						}
						gd.joined ? (leaveConfirmOpen = true) : requestJoin();
					}}
				>
					{#if gd.joinPendingApproval}
						<span class="flex items-center justify-center gap-1.5">
							<Check size={15} />
							审核中 · 点击取消
						</span>
					{:else}
						<span class="flex items-center justify-center gap-1.5">
							<Check size={15} />
							已加入 · 点击退出
						</span>
					{/if}
				</Button>
			{:else}
				<Button type="button" variant="primary" size="lg" class="w-full" onclick={requestJoin}>加入群组</Button>
			{/if}
				{#if gd.joined && (gd.myRole === 'owner' || gd.myRole === 'admin')}
					<Button type="button" variant="ghost" size="lg" flat class="w-full bg-surface-subtle/60 hover:bg-surface-subtle" onclick={() => (settingsOpen = true)}>
						<Settings size={15} />
						群管理
					</Button>
				{/if}
			{#if gd.joined}
				<Button type="button" variant="primary" size="lg" class="w-full" onclick={() => void goto(ROUTES.groupMessage(params.groupId))}>
					<MessageCircle size={15} />
					打开群聊
				</Button>
			{:else}
				<p class="px-2 text-center text-xs leading-5 text-text-secondary">
					<Bell size={13} class="mr-1 inline text-accent-default" />
					加入后可管理群组通知并进入群聊
				</p>
			{/if}
		</section>
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

<!-- 更多操作菜单（bottom sheet，对齐用户详情页「更多操作」交互模式） -->
<Dialog.Root open={moreOpen} onOpenChange={(next) => (next ? undefined : (moreOpen = false))}>
	<Dialog.Content
		side="bottom"
		class="inset-x-4 bottom-4 mx-auto max-w-md rounded-xl p-5"
		aria-label={`${gd.detail.name} 的更多操作`}
		style="box-shadow:var(--cst-elevation-modal);"
	>
		<div class="flex items-center justify-between">
			<div>
				<p class="eyebrow">更多操作</p>
				<p class="mt-1 text-md text-text-primary">{gd.detail.name}</p>
			</div>
			<ActionIconButton icon={X} label="关闭" onClick={() => (moreOpen = false)} />
		</div>
		{#if canReportGroup}
			<Button type="button" variant="ghost" class="mt-4 w-full text-status-danger hover:bg-danger-bg" onclick={() => { moreOpen = false; reportOpen = true; }}>
				<Flag size={14} />
				举报群组
			</Button>
		{/if}
	</Dialog.Content>
</Dialog.Root>

<ReportDialog
	open={reportOpen}
	targetType="group"
	targetId={params.groupId ?? ''}
	targetLabel="该群组"
	onClose={() => (reportOpen = false)}
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
	onInvited={() => {
		// 邀请后同步刷新已发邀请列表（此前只刷成员，切到邀请管理 tab 仍是旧数据）。
		void gd.loadMembers(params.groupId);
		void gd.loadSentInvites(params.groupId);
	}}
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
