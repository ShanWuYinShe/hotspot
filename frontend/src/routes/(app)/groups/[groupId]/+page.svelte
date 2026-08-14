<script lang="ts">
	import { goto } from '$app/navigation';
	import { ArrowLeft, Bell, Check, MapPin, Megaphone, MessageCircle, MoreHorizontal, PencilLine, Pin, Settings, Trash2, UserPlus, Users } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import GroupAnnouncementDialog from '@/components/groups/GroupAnnouncementDialog.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import { ROUTES } from '@/lib/routes';
	import { useGroupDetail } from '@/lib/group-data.svelte';
	import { formatDateTime } from '@/lib/utils';
	import { presence, watchPresence } from '@/stores/presence.svelte';
	import { auth } from '@/stores/auth.svelte';
	import { setGroupMemberRole, setGroupMemberMute, banGroupMember, removeGroupMember } from '@/api';
	import MemberActionSheet from '@/components/groups/MemberActionSheet.svelte';
	import InviteMembersSheet from '@/components/groups/InviteMembersSheet.svelte';
	import GroupSettingsSheet from '@/components/groups/GroupSettingsSheet.svelte';
	import type { MemberAction } from '@/components/groups/MemberActionSheet.svelte';
	import type { UpdateGroupRequest } from '@/api/types';

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
		isOnline: (id) => presence.isOnline(id)
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


{#if gd.loadError}
	<LoadError message={gd.loadError} onRetry={() => void gd.loadAll(params.groupId ?? '')} />
{:else}
<StandardPageLayout contentClassName="animate-fade-in pb-6">
	<!-- 全幅封面（负 margin 抵消 ContentColumn gutter） -->
	<div class="relative h-[var(--cst-cover-height)] overflow-hidden -mx-[var(--cst-page-gutter)]">
		<img src={gd.detail.banner} alt={`${gd.detail.name} 的群组封面`} class="h-full w-full object-cover" />
		<!-- gradient scrim 走 inline background（gradient 不能作 background-color 值，
		     与 NearbyImageCard 的 --cst-surface-media-scrim-card 用法一致） -->
		<div class="absolute inset-0" style="background:var(--cst-surface-cover-scrim);"></div>
		<div class="absolute left-5 top-4">
			<ActionIconButton icon={ArrowLeft} label="返回" onClick={() => history.back()} variant="overlay" tone="inverse" />
		</div>
		<div class="absolute bottom-5 left-6 right-6">
			<span class="rounded-full bg-surface-glass-badge px-2.5 py-1 text-xs text-content-inverse backdrop-blur-sm">{gd.detail.category}</span>
			<h1 class="mt-3 text-2xl md:text-3xl font-medium tracking-title text-content-inverse">{gd.detail.name}</h1>
		</div>
	</div>

	<div class="grid gap-8 pt-6 lg:grid-cols-[minmax(0,1fr)_20rem]">
		<main class="min-w-0">
			<div class="flex flex-wrap items-center gap-x-5 gap-y-2 text-sm text-text-secondary">
				<span class="flex items-center gap-1.5">
					<Users size={14} />
					{gd.detail.members} 位成员
				</span>
				<span class="flex items-center gap-1.5 text-status-online">
					<span class="h-2 w-2 rounded-full bg-status-online"></span>
					{gd.onlineMemberCount} 人在线
				</span>
				<span class="flex items-center gap-1.5">
					<MapPin size={14} />
					距你 {gd.detail.distance}
				</span>
			</div>

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
				<div class="py-6">
					<p class="max-w-3xl text-sm leading-7 text-text-secondary">{gd.detail.description}</p>
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
					{#if canManageMembers}
						<div class="mt-6 flex justify-end">
							<button type="button" onclick={() => (inviteOpen = true)} class="hover-btn flex items-center gap-1 rounded-full border border-border-default px-3 py-1.5 text-xs text-text-primary focus-ring">
								<UserPlus size={12} strokeWidth={1.75} />
								添加成员
							</button>
						</div>
					{/if}
					<!-- 空旷内容区 → List bordered（有边框列表契约）；列表行不设独立边框 -->
					<ListContainer bordered class="mt-4">
						{#each gd.members as member (member.id)}
							<ListItem
								onClick={() => {
									if (member.userId) void goto(ROUTES.user(member.userId));
								}}
								ariaLabel={`查看 ${member.name} 的资料`}
								class="flex list-row-standard items-center gap-3 px-3.5 py-3"
							>
								<div class="relative">
									<img src={member.avatar} alt="" loading="lazy" class="h-10 w-10 rounded-full object-cover" />
									{#if gd.onlineMemberIds.has(member.userId)}
										<span class="absolute bottom-0 right-0 h-2.5 w-2.5 rounded-full border-2 border-surface-primary bg-status-online"></span>
									{/if}
								</div>
								<span class="min-w-0 flex-1">
									<span class="block text-sm text-text-primary">{member.name}</span>
									<span class="text-xs text-text-secondary">{member.role}</span>
								</span>
								<span class="text-xs text-text-secondary">查看资料</span>
								{#if canManageMember(member)}
									<button
										type="button"
										onclick={(e) => { e.stopPropagation(); actionMember = member; }}
										class="hit-slop flex h-7 w-7 items-center justify-center rounded-full text-text-secondary outline-none hover:bg-interactive-hover hover-btn scale-btn focus-visible:ring-2 focus-visible:ring-accent-default"
										aria-label={`管理 ${member.name}`}
									>
										<MoreHorizontal size={15} strokeWidth={2} />
									</button>
								{/if}
							</ListItem>
						{/each}
					</ListContainer>
					{:else if memberTab === 'applications'}
						<!-- 申请审批（owner/admin；默认仅 pending，管理员工作台语义） -->
						<ListContainer bordered class="mt-4">
							{#each gd.applications as app (app.application_id)}
								<ListItem class="flex list-row-standard items-center gap-3 px-3.5 py-3" ariaLabel={`${app.applicant.nickname} 的入群申请`}>
									<img src={app.applicant.avatar_url} alt="" loading="lazy" class="h-10 w-10 rounded-full object-cover" />
									<span class="min-w-0 flex-1">
										<span class="block truncate text-sm text-text-primary">{app.applicant.nickname}</span>
										<span class="text-xs text-text-secondary">{formatDateTime(app.created_at)}{app.message ? ` · ${app.message}` : ''}</span>
									</span>
									<button type="button" onclick={() => void handleApproveApplication(app.application_id)} class="hover-btn rounded-full bg-accent-default px-3 py-1.5 text-xs font-medium text-text-on-accent focus-ring">同意</button>
									<button type="button" onclick={() => void handleRejectApplication(app.application_id)} class="hover-btn rounded-full border border-border-default px-3 py-1.5 text-xs text-text-primary focus-ring">拒绝</button>
								</ListItem>
							{:else}
								<EmptyState bare title="暂无待处理的申请" />
							{/each}
						</ListContainer>
					{:else if memberTab === 'invites'}
					<!-- 已发邀请（owner/admin；默认仅 pending，revoked 含过期自动撤销） -->
					<ListContainer bordered class="mt-4">
						{#each gd.sentInvites as inv (inv.invite_id)}
							<ListItem class="flex list-row-standard items-center gap-3 px-3.5 py-3" ariaLabel={`${inv.invitee.nickname} 的入群邀请`}>
								<img src={inv.invitee.avatar_url} alt="" loading="lazy" class="h-10 w-10 rounded-full object-cover" />
								<span class="min-w-0 flex-1">
									<span class="block truncate text-sm text-text-primary">{inv.invitee.nickname}</span>
									<span class="text-xs text-text-secondary">{inviteStatusText(inv.status)} · {formatDateTime(inv.created_at)}</span>
								</span>
								{#if inv.status === 'pending'}
									<button type="button" onclick={() => void handleRevokeInvite(inv.invite_id)} class="hover-btn rounded-full border border-border-default px-3 py-1.5 text-xs text-text-primary focus-ring">撤销</button>
								{/if}
							</ListItem>
						{:else}
							<EmptyState bare title="暂无待处理的邀请" />
						{/each}
					</ListContainer>
				{:else}
					<!-- 封禁名单（owner/admin） -->
					<ListContainer bordered class="mt-4">
						{#each gd.bannedMembers as item (item.user.user_id)}
							<ListItem class="flex list-row-standard items-center gap-3 px-3.5 py-3" ariaLabel={`${item.user.nickname} 的封禁记录`}>
								<img src={item.user.avatar_url} alt="" loading="lazy" class="h-10 w-10 rounded-full object-cover" />
								<span class="min-w-0 flex-1">
									<span class="block truncate text-sm text-text-primary">{item.user.nickname}</span>
									<span class="text-xs text-text-secondary">{formatDateTime(item.banned_at)}{item.reason ? ` · ${item.reason}` : ''}</span>
								</span>
								<button type="button" onclick={() => void handleUnban(item.user.user_id)} class="hover-btn rounded-full border border-border-default px-3 py-1.5 text-xs text-text-primary focus-ring">解封</button>
							</ListItem>
						{:else}
							<EmptyState bare title="暂无封禁成员" />
						{/each}
					</ListContainer>
				{/if}
			{:else}
				<div class="mt-6 flex items-center justify-between">
					<p class="text-sm text-text-secondary">群公告</p>
					{#if canManageAnnouncements}
						<button type="button" onclick={openCreateAnnouncement} class="hover-btn flex items-center gap-1 rounded-full border border-border-default px-3 py-1.5 text-xs text-text-primary focus-ring">
							<Megaphone size={12} strokeWidth={1.75} />
							发公告
						</button>
					{/if}
				</div>
				<!-- 空旷内容区 → List bordered（有边框列表契约） -->
				<ListContainer bordered class="mt-4">
					{#each gd.announcements as item (item.id)}
						<ListItem class="flex flex-col items-start px-4 py-4" ariaLabel={item.title}>
							<div class="flex items-center gap-1.5 pr-16">
								{#if item.isPinned}
									<Pin size={12} class="shrink-0 text-accent-default" strokeWidth={1.75} />
								{/if}
								<p class="text-sm font-medium text-text-primary">{item.title}</p>
							</div>
							<p class="mt-1 text-sm leading-5 text-text-secondary">{item.body}</p>
							<p class="mt-2 text-xs text-text-secondary">{item.time}</p>
							{#if canManageAnnouncements}
								<div class="absolute right-3 top-3 flex gap-1">
									<button type="button" onclick={() => openEditAnnouncement(item)} class="hit-slop flex h-6 w-6 items-center justify-center rounded-full text-text-secondary outline-none hover:bg-interactive-hover hover-btn scale-btn focus-visible:ring-2 focus-visible:ring-accent-default" aria-label="编辑公告">
										<PencilLine size={12} strokeWidth={1.75} />
									</button>
									<button type="button" onclick={() => (deleteAnnounceId = item.id)} class="hit-slop flex h-6 w-6 items-center justify-center rounded-full text-status-danger outline-none hover:bg-interactive-hover hover-btn scale-btn focus-visible:ring-2 focus-visible:ring-status-danger" aria-label="删除公告">
										<Trash2 size={12} strokeWidth={1.75} />
									</button>
								</div>
							{/if}
						</ListItem>
					{:else}
						<EmptyState bare title="暂无公告" />
					{/each}
				</ListContainer>
			{/if}
		</main>

		<aside class="flex h-fit flex-col gap-3 rounded-lg border border-border-default bg-surface-primary p-4">
			<button
				type="button"
				class="hover-btn w-full rounded-full px-4 py-3 text-sm font-medium focus-ring {gd.joined ? 'border border-accent-default text-accent-default' : 'bg-accent-default text-text-on-accent'}"
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
	</StandardPageLayout>
{/if}

<Dialog.Root open={deleteAnnounceId !== null} onOpenChange={(next) => (next ? undefined : (deleteAnnounceId = null))}>
	<Dialog.Content side="center" class="max-w-sm p-5" aria-label="确认删除公告">
		<p class="text-md text-text-primary">删除这条公告？</p>
		<p class="mt-1 text-sm text-text-secondary">删除后无法恢复。</p>
		<div class="mt-4 flex items-center justify-end gap-3">
			<button type="button" onclick={() => (deleteAnnounceId = null)} class="hover-btn rounded-full px-4 py-2 text-sm text-text-secondary outline-none focus-visible:ring-2 focus-visible:ring-accent-default">取消</button>
			<button type="button" onclick={handleDeleteAnnouncement} class="hover-btn rounded-full px-5 py-2 text-sm font-medium text-text-on-accent outline-none focus-visible:ring-2 focus-visible:ring-status-danger enabled:bg-status-danger">确认删除</button>
		</div>
	</Dialog.Content>
</Dialog.Root>

<Dialog.Root open={leaveConfirmOpen} onOpenChange={(next) => (next ? undefined : (leaveConfirmOpen = false))}>
	<Dialog.Content side="center" class="max-w-sm p-5" aria-label="确认退出群组">
		<p class="text-md text-text-primary">退出群组「{gd.detail.name}」？</p>
		<p class="mt-1 text-sm text-text-secondary">退出后将不再接收该群的聊天与公告。</p>
		<div class="mt-4 flex items-center justify-end gap-3">
			<button type="button" onclick={() => (leaveConfirmOpen = false)} class="hover-btn rounded-full px-4 py-2 text-sm text-text-secondary outline-none focus-visible:ring-2 focus-visible:ring-accent-default">取消</button>
			<button type="button" onclick={handleLeaveConfirm} class="hover-btn rounded-full px-5 py-2 text-sm font-medium text-text-on-accent outline-none focus-visible:ring-2 focus-visible:ring-status-danger enabled:bg-status-danger">确认退出</button>
		</div>
	</Dialog.Content>
</Dialog.Root>

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
