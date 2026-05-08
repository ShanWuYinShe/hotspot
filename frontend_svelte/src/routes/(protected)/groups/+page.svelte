<script lang="ts">
	import { onMount } from 'svelte';
	import { withMinimumLoadingTime } from '$lib/utils/loading';
	import { groupsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import { formatDateTime } from '$lib/utils/format';
	import ListSkeleton from '$lib/components/common/list-skeleton.svelte';
	import EmptyState from '$lib/components/common/empty-state.svelte';
	import CreateGroupModal from '$lib/components/common/create-group-modal.svelte';
	import type { GroupApplicationItem, GroupInviteItem, GroupItem } from '$lib/types/api';
	import Plus from 'lucide-svelte/icons/plus';
	import Users from 'lucide-svelte/icons/users';
	import RefreshCw from 'lucide-svelte/icons/refresh-cw';
	import Inbox from 'lucide-svelte/icons/inbox';
	import CheckCircle2 from 'lucide-svelte/icons/check-circle-2';

	let groups = $state.raw<GroupItem[]>([]);
	let invites = $state.raw<GroupInviteItem[]>([]);
	let applications = $state.raw<GroupApplicationItem[]>([]);
	let isLoading = $state(true);
	let invitesLoading = $state(false);
	let applicationsLoading = $state(false);
	let showCreateGroup = $state(false);
	let processingInviteIds = $state<Set<string>>(new Set());
	let processingApplicationIds = $state<Set<string>>(new Set());

	async function loadGroups() {
		isLoading = true;
		try {
			const response = await withMinimumLoadingTime(() =>
				groupsService.getMyGroups({ pagination: { limit: 50 } })
			);
			if (isSuccess(response) && response.data) {
				groups = response.data.items;
			}
		} catch {
			toast.error('加载失败');
		} finally {
			isLoading = false;
		}
	}

	async function loadInvites() {
		invitesLoading = true;
		try {
			const response = await groupsService.getInvites({
				status: 'pending',
				pagination: { limit: 50 }
			});
			if (isSuccess(response) && response.data) {
				invites = response.data.items;
			}
		} catch {
			toast.error('加载群邀请失败');
		} finally {
			invitesLoading = false;
		}
	}

	async function loadApplications() {
		applicationsLoading = true;
		try {
			const response = await groupsService.getMyApplications({
				status: 'pending',
				pagination: { limit: 50 }
			});
			if (isSuccess(response) && response.data) {
				applications = response.data.items;
			}
		} catch {
			toast.error('加载群申请失败');
		} finally {
			applicationsLoading = false;
		}
	}

	function setInviteProcessing(inviteId: string, processing: boolean) {
		const next = new Set(processingInviteIds);
		if (processing) next.add(inviteId);
		else next.delete(inviteId);
		processingInviteIds = next;
	}

	function setApplicationProcessing(applicationId: string, processing: boolean) {
		const next = new Set(processingApplicationIds);
		if (processing) next.add(applicationId);
		else next.delete(applicationId);
		processingApplicationIds = next;
	}

	function isInviteProcessing(inviteId: string) {
		return processingInviteIds.has(inviteId);
	}

	function isApplicationProcessing(applicationId: string) {
		return processingApplicationIds.has(applicationId);
	}

	async function handleInvite(invite: GroupInviteItem, action: 'accept' | 'reject') {
		if (isInviteProcessing(invite.invite_id)) return;
		setInviteProcessing(invite.invite_id, true);
		try {
			const response =
				action === 'accept'
					? await groupsService.acceptInvite({ invite_id: invite.invite_id })
					: await groupsService.rejectInvite({ invite_id: invite.invite_id });
			if (isSuccess(response)) {
				invites = invites.filter((item) => item.invite_id !== invite.invite_id);
				toast.success(action === 'accept' ? '已接受群邀请' : '已拒绝群邀请');
				if (action === 'accept') {
					await loadGroups();
				}
			} else {
				toast.error(response.message || '操作失败');
			}
		} catch {
			toast.error('操作失败');
		} finally {
			setInviteProcessing(invite.invite_id, false);
		}
	}

	async function handleCancelApplication(application: GroupApplicationItem) {
		if (isApplicationProcessing(application.application_id)) return;
		setApplicationProcessing(application.application_id, true);
		try {
			const response = await groupsService.cancelApplication({
				application_id: application.application_id
			});
			if (isSuccess(response)) {
				applications = applications.filter(
					(item) => item.application_id !== application.application_id
				);
				toast.success('已取消申请');
			} else {
				toast.error(response.message || '取消申请失败');
			}
		} catch {
			toast.error('取消申请失败');
		} finally {
			setApplicationProcessing(application.application_id, false);
		}
	}

	onMount(() => {
		loadGroups();
		loadInvites();
		loadApplications();
	});
</script>

<svelte:head>
	<title>我的群组 - HotSpot</title>
</svelte:head>

<div class="flex flex-col">
	<!-- Sticky header -->
	<div
		class="sticky top-0 z-10 px-5 pt-3 pb-3"
		style="background-color: var(--neu-bg); border-bottom: 1px solid var(--neu-border);"
	>
		<div class="flex items-center justify-between">
			<div>
				<h1 style="font-size: 20px; font-weight: 600; color: var(--neu-text);">我的群组</h1>
				<p style="font-size: 12px; color: var(--neu-text-muted); margin-top: 2px;">管理已加入群组、邀请与申请</p>
			</div>
			<div class="flex gap-2">
				<button
					class="flex items-center gap-1.5 rounded-full px-3 py-1.5 outline-none"
					style="background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); font-size: 12px; color: var(--neu-text-muted);"
					onclick={loadGroups}
					disabled={isLoading}
				>
					<RefreshCw size={13} class={isLoading ? 'animate-spin' : ''} />
					刷新
				</button>
				<button
					class="flex items-center gap-1.5 rounded-full px-3 py-1.5 outline-none"
					style="background: var(--neu-accent-primary); color: var(--neu-active-text); font-size: 12px; box-shadow: 0 2px 8px var(--neu-glow);"
					onclick={() => (showCreateGroup = true)}
				>
					<Plus size={13} />
					创建群组
				</button>
			</div>
		</div>
	</div>

	<div class="flex flex-col px-4 pt-4 pb-6 gap-5">
		{#if isLoading}
			<ListSkeleton type="group" count={6} layout="grid" />
		{:else if groups.length === 0}
			<EmptyState
				icon={Users}
				title="还没有加入群组"
				description="去探索或创建一个群组吧"
			>
				<div class="mt-4 flex gap-3">
					<a href="/groups/nearby" class="rounded-full px-4 py-2 text-sm" style="background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text); text-decoration: none;">发现附近群组</a>
					<button class="rounded-full px-4 py-2 text-sm" style="background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 2px 8px var(--neu-glow);" onclick={() => (showCreateGroup = true)}>创建群组</button>
				</div>
			</EmptyState>
		{:else}
			<div class="columns-1 gap-4 sm:columns-2 xl:columns-3">
				{#each groups as group (group.group.group_id)}
					<div class="mb-4 break-inside-avoid">
						<a href="/groups/{group.group.group_id}" style="text-decoration: none;">
							<div class="flex items-center gap-4 rounded-[18px] p-4" style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);">
								<div class="overflow-hidden rounded-full shrink-0" style="width: 48px; height: 48px; border: 1.5px solid var(--neu-border);">
									{#if group.group.avatar_url}
										<img src={group.group.avatar_url} alt={group.group.name} class="h-full w-full object-cover" />
									{:else}
										<div class="flex h-full w-full items-center justify-center" style="background: var(--neu-bg); font-size: 14px; color: var(--neu-text-muted);">{group.group.name.slice(0, 2)}</div>
									{/if}
								</div>
								<div class="min-w-0 flex-1">
									<div class="flex items-center justify-between">
										<p class="truncate" style="font-size: 14px; font-weight: 500; color: var(--neu-text);">{group.group.name}</p>
										{#if (group.my_state?.unread_count ?? 0) > 0}
											<div class="flex items-center justify-center rounded-full" style="width: 20px; height: 20px; background: var(--neu-accent-primary); color: var(--neu-active-text); font-size: 11px; font-weight: 600; box-shadow: 0 2px 8px var(--neu-glow);">
												{(group.my_state?.unread_count ?? 0) > 99 ? '99+' : (group.my_state?.unread_count ?? 0)}
											</div>
										{/if}
									</div>
									<p style="font-size: 13px; color: var(--neu-text-muted);">
										{group.group.members_count} 成员
									</p>
								</div>
							</div>
						</a>
					</div>
				{/each}
			</div>
		{/if}

		<!-- Invites & Applications -->
		<div class="grid gap-4 lg:grid-cols-2">
			<!-- Invites -->
			<div class="rounded-[22px] p-4" style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);">
				<div class="flex items-center justify-between mb-3">
					<div class="flex items-center gap-2">
						<div class="flex items-center justify-center rounded-full" style="width: 28px; height: 28px; box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);">
							<Inbox size={12} color="var(--neu-text-muted)" strokeWidth={1.5} />
						</div>
						<span style="font-size: 14px; font-weight: 600; color: var(--neu-text);">收到的群邀请</span>
					</div>
					<button
						class="rounded-full px-3 py-1 text-xs outline-none"
						style="background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted);"
						onclick={loadInvites}
						disabled={invitesLoading}
					>
						刷新
					</button>
				</div>
				{#if invitesLoading && invites.length === 0}
					<p style="font-size: 13px; color: var(--neu-text-muted);">加载中...</p>
				{:else if invites.length === 0}
					<p style="font-size: 13px; color: var(--neu-text-muted);">暂无待处理邀请</p>
				{:else}
					<div class="space-y-2">
						{#each invites as invite (invite.invite_id)}
							<div class="rounded-[14px] p-3" style="box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);">
								<p style="font-size: 13px; font-weight: 500; color: var(--neu-text);">{invite.group.name}</p>
								<p style="font-size: 12px; color: var(--neu-text-muted); margin-top: 4px;">
									邀请人：{invite.inviter.nickname}
								</p>
								{#if invite.message}
									<p style="font-size: 12px; color: var(--neu-text-muted); margin-top: 2px;">{invite.message}</p>
								{/if}
								<div class="mt-2 flex gap-2">
									<button
										class="rounded-full px-3 py-1 text-xs outline-none"
										style="background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 2px 6px var(--neu-glow);"
										disabled={isInviteProcessing(invite.invite_id)}
										onclick={() => handleInvite(invite, 'accept')}
									>
										接受
									</button>
									<button
										class="rounded-full px-3 py-1 text-xs outline-none"
										style="box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);"
										disabled={isInviteProcessing(invite.invite_id)}
										onclick={() => handleInvite(invite, 'reject')}
									>
										拒绝
									</button>
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</div>

			<!-- Applications -->
			<div class="rounded-[22px] p-4" style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);">
				<div class="flex items-center justify-between mb-3">
					<div class="flex items-center gap-2">
						<div class="flex items-center justify-center rounded-full" style="width: 28px; height: 28px; box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);">
							<CheckCircle2 size={12} color="var(--neu-text-muted)" strokeWidth={1.5} />
						</div>
						<span style="font-size: 14px; font-weight: 600; color: var(--neu-text);">我的入群申请</span>
					</div>
					<button
						class="rounded-full px-3 py-1 text-xs outline-none"
						style="background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted);"
						onclick={loadApplications}
						disabled={applicationsLoading}
					>
						刷新
					</button>
				</div>
				{#if applicationsLoading && applications.length === 0}
					<p style="font-size: 13px; color: var(--neu-text-muted);">加载中...</p>
				{:else if applications.length === 0}
					<p style="font-size: 13px; color: var(--neu-text-muted);">暂无待处理申请</p>
				{:else}
					<div class="space-y-2">
						{#each applications as application (application.application_id)}
							<div class="rounded-[14px] p-3" style="box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);">
								<p style="font-size: 13px; font-weight: 500; color: var(--neu-text);">群组申请</p>
								{#if application.message}
									<p style="font-size: 12px; color: var(--neu-text-muted); margin-top: 4px;">{application.message}</p>
								{/if}
								<p style="font-size: 12px; color: var(--neu-text-muted); margin-top: 2px;">
									申请时间：{formatDateTime(application.created_at)}
								</p>
								<div class="mt-2 flex gap-2">
									<button
										class="rounded-full px-3 py-1 text-xs outline-none"
										style="box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);"
										disabled={isApplicationProcessing(application.application_id)}
										onclick={() => handleCancelApplication(application)}
									>
										取消申请
									</button>
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</div>
		</div>
	</div>
</div>

<CreateGroupModal bind:open={showCreateGroup} onCreated={loadGroups} />
