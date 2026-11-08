<script lang="ts">
	import { goto } from '$app/navigation';
	import { MoreHorizontal } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import DetailHeader from '@/components/layout/page/DetailHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import { formatCount } from '@/lib/utils';
	import { ROUTES } from '@/lib/routes';
	import { smartBack } from '@/lib/smart-back';
	import { platform } from '@/stores/platform.svelte';

	import { useGroupPageData, type GroupTab, type MemberSubTab } from './shared/useGroupPageData.svelte';
	import GroupHero from './shared/GroupHero.svelte';
	import GroupOverviewTab from './shared/GroupOverviewTab.svelte';
	import GroupActionCard from './shared/GroupActionCard.svelte';
	import GroupOverlays from './shared/GroupOverlays.svelte';
	import MembersList from './shared/MembersList.svelte';
	import ApplicationsList from './shared/ApplicationsList.svelte';
	import InvitesList from './shared/InvitesList.svelte';
	import BannedList from './shared/BannedList.svelte';
	import AnnouncementsList from './shared/AnnouncementsList.svelte';
	import { formatDateTime } from '@/lib/utils';

	let { params }: { params: { groupId: string } } = $props();

	const page = useGroupPageData({
		groupId: () => params.groupId
	});
	const gd = page.gd;
</script>

<svelte:head>
	<title>群组主页 · HotSpot</title>
</svelte:head>

{#snippet groupHeaderActions()}
	{#if page.canReportGroup}
		<ActionIconButton icon={MoreHorizontal} label="更多操作" onClick={() => (page.moreOpen = true)} variant="ghost" size={36} />
	{/if}
{/snippet}

<StandardPageLayout contentClassName={platform.platform === 'mobile' ? 'pb-8' : 'pb-6'}>
	{#snippet header()}
		<DetailHeader
			category="社群"
			entityName={gd.detail.name || '群组主页'}
			author={gd.detail.name && !gd.loading ? {
				id: params.groupId,
				name: gd.detail.name,
				avatar: gd.detail.banner,
				location: gd.detail.category || '社群',
				time: `${formatCount(gd.detail.members ?? 0)} 位成员`,
				role: '社群'
			} : undefined}
			onBack={() => smartBack()}
			trailing={groupHeaderActions}
		/>
	{/snippet}

	{#if gd.loading}
		<h1 class="sr-only">群组主页</h1>
		<InlineLoading label="加载群组…" />
	{:else if gd.loadError}
		<h1 class="sr-only">群组主页</h1>
		<LoadError message={gd.loadError} onRetry={() => void gd.loadAll(params.groupId ?? '')} />
	{:else}
		<GroupHero detail={gd.detail} />

		<div class="grid gap-y-6 pt-6 lg:grid-cols-[minmax(0,1fr)_20rem] lg:gap-x-8">
			<div class="min-w-0">
				<div>
					<TabBar
						items={[
							{ id: 'overview', label: '简介' },
							{ id: 'members', label: '成员' },
							{ id: 'announcements', label: '公告' }
						]}
						value={page.tab}
						onChange={(id) => (page.tab = id as GroupTab)}
					/>
				</div>

				{#if page.tab === 'overview'}
					<GroupOverviewTab
						detail={gd.detail}
						members={gd.members}
						announcements={gd.announcements}
						onViewAllMembers={() => (page.tab = 'members')}
						onViewAnnouncements={() => (page.tab = 'announcements')}
					/>
				{:else if page.tab === 'members'}
					{#if page.canManageMembers}
						<div class="mt-6">
							<TabBar
								items={[
									{ id: 'members', label: '成员' },
									{ id: 'applications', label: '申请审批' },
									{ id: 'invites', label: '邀请管理' },
									{ id: 'banned', label: '封禁名单' }
								]}
								value={page.memberTab}
								onChange={(id) => (page.memberTab = id as MemberSubTab)}
							/>
						</div>
					{/if}

					{#if !page.canManageMembers || page.memberTab === 'members'}
						<MembersList
							members={gd.members}
							total={gd.detail.members}
							onlineMemberIds={gd.onlineMemberIds}
							canManageMembers={page.canManageMembers}
							canManageMember={page.canManageMember}
							hasMore={gd.hasMoreMembers()}
							loadingMore={gd.isLoadingMoreMembers()}
							onInvite={() => (page.inviteOpen = true)}
							onManage={(m) => (page.actionMember = m)}
							onLoadMore={() => void gd.loadMoreMembers()}
						/>
					{:else if page.memberTab === 'applications'}
						<ApplicationsList
							applications={gd.applications}
							{formatDateTime}
							pending={page.applicationPending}
							onApprove={(id) => void page.handleApproveApplication(id)}
							onReject={(id) => void page.handleRejectApplication(id)}
						/>
					{:else if page.memberTab === 'invites'}
						<InvitesList
							invites={gd.sentInvites}
							inviteStatusText={page.inviteStatusText}
							{formatDateTime}
							onRevoke={(id) => void page.handleRevokeInvite(id)}
						/>
					{:else}
						<BannedList
							banned={gd.bannedMembers}
							{formatDateTime}
							onUnban={(id) => void page.handleUnban(id)}
						/>
					{/if}
				{:else}
					<AnnouncementsList
						announcements={gd.announcements}
						canManage={page.canManageAnnouncements}
						onCreate={page.openCreateAnnouncement}
						onEdit={page.openEditAnnouncement}
						onDelete={(id) => (page.deleteAnnounceId = id)}
					/>
				{/if}
			</div>

			<GroupActionCard
				joined={gd.joined}
				myRole={gd.myRole}
				joinPendingApproval={gd.joinPendingApproval}
				joinPop={page.joinPop}
				onOpenChat={() => void goto(ROUTES.groupMessage(params.groupId))}
				onOpenSettings={() => (page.settingsOpen = true)}
				onLeave={() => {
					if (gd.myRole === 'owner') {
						import('@/stores/toast.svelte').then(({ showToast }) =>
							showToast('群主需先转让群主身份（群管理 → 转让群主）', 'danger')
						);
						return;
					}
					page.leaveConfirmOpen = true;
				}}
				onCancelApplication={() => void gd.cancelJoinApplication()}
				onRequestJoin={page.requestJoin}
			/>
		</div>
	{/if}
</StandardPageLayout>

<GroupOverlays
	groupId={params.groupId}
	groupName={gd.detail.name}
	deleteAnnounceId={page.deleteAnnounceId}
	announcementDeleting={page.announcementDeleting}
	onCancelDeleteAnnounce={() => (page.deleteAnnounceId = null)}
	onConfirmDeleteAnnounce={() => void page.handleDeleteAnnouncement()}
	leaveConfirmOpen={page.leaveConfirmOpen}
	onCancelLeave={() => (page.leaveConfirmOpen = false)}
	onConfirmLeave={page.handleLeaveConfirm}
	bind:moreOpen={page.moreOpen}
	canReportGroup={page.canReportGroup}
	bind:reportOpen={page.reportOpen}
	bind:announceOpen={page.announceOpen}
	announceMode={page.announceMode}
	announceEditing={page.announceEditing}
	onSubmitAnnounce={page.handleAnnouncementSubmit}
	bind:actionMember={page.actionMember}
	memberActionPending={page.memberActionPending}
	onMemberAction={(act, dur) => void page.handleMemberAction(act, dur)}
	bind:inviteOpen={page.inviteOpen}
	onInvited={() => {
		void gd.loadMembers(params.groupId);
		void gd.loadSentInvites(params.groupId);
	}}
	bind:settingsOpen={page.settingsOpen}
	rawGroup={gd.rawGroup}
	isOwner={gd.myRole === 'owner'}
	members={gd.members}
	onSubmitSettings={page.handleGroupSettingsSubmit}
	onTransferOwnership={page.handleTransferOwnership}
	onDismissGroup={page.handleDismissGroup}
/>
