<script lang="ts">
	import { onMount } from 'svelte';
	import { withMinimumLoadingTime } from '$lib/utils/loading';
	import { groupsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import { formatDateTime } from '$lib/utils/format';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent } from '$lib/components/ui/card';
	import PageHeader from '$lib/components/layout/page-header.svelte';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { Badge } from '$lib/components/ui/badge';
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

<div class="space-y-5">
	<PageHeader title="我的群组" description="管理已加入群组、邀请与申请">
		{#snippet actions()}
			<div class="flex gap-2">
				<Button
					variant="outline"
					size="sm"
					class="h-10 rounded-2xl border-border/60 bg-background/70"
					onclick={loadGroups}
					disabled={isLoading}
				>
					<RefreshCw class="mr-2 h-4 w-4 {isLoading ? 'animate-spin' : ''}" />
					刷新
				</Button>
				<Button size="sm" class="h-10 rounded-2xl" onclick={() => (showCreateGroup = true)}>
					<Plus class="mr-2 h-4 w-4" />
					创建群组
				</Button>
			</div>
		{/snippet}
	</PageHeader>

	{#if isLoading}
		<ListSkeleton type="group" count={6} layout="grid" />
	{:else if groups.length === 0}
		<EmptyState 
			icon={Users} 
			title="还没有加入群组" 
			description="去探索或创建一个群组吧"
		>
			<div class="mt-4 flex gap-3">
				<a href="/groups/nearby">
					<Button variant="outline">发现附近群组</Button>
				</a>
				<Button onclick={() => (showCreateGroup = true)}>创建群组</Button>
			</div>
		</EmptyState>
	{:else}
		<div class="columns-1 gap-4 sm:columns-2 xl:columns-3">
			{#each groups as group (group.group.group_id)}
				<div class="mb-4 break-inside-avoid">
					<a href="/groups/{group.group.group_id}">
						<Card class="cursor-pointer rounded-3xl border border-border/60 bg-card/85 shadow-sm backdrop-blur-md transition-shadow hover:shadow-md">
							<CardContent class="flex items-center gap-4 p-4">
								<Avatar class="h-12 w-12">
									<AvatarImage src={group.group.avatar_url} alt={group.group.name} />
									<AvatarFallback>{group.group.name.slice(0, 2)}</AvatarFallback>
								</Avatar>
								<div class="min-w-0 flex-1">
									<div class="flex items-center justify-between">
										<p class="truncate font-medium">{group.group.name}</p>
							{#if (group.my_state?.unread_count ?? 0) > 0}
								<Badge variant="destructive">
									{(group.my_state?.unread_count ?? 0) > 99 ? '99+' : (group.my_state?.unread_count ?? 0)}
								</Badge>
							{/if}
									</div>
									<p class="text-sm text-muted-foreground">
										{group.group.members_count} 成员
									</p>
								</div>
							</CardContent>
						</Card>
					</a>
				</div>
			{/each}
		</div>
	{/if}

	<div class="grid gap-4 lg:grid-cols-2">
		<Card class="rounded-3xl border border-border/60 bg-card/80 shadow-sm backdrop-blur-md">
			<CardContent class="space-y-3 p-4">
				<div class="flex items-center justify-between">
					<div class="flex items-center gap-2">
						<Inbox class="h-4 w-4 text-muted-foreground" />
						<p class="text-sm font-semibold">收到的群邀请</p>
					</div>
					<Button variant="outline" size="sm" class="rounded-2xl border-border/60 bg-background/70" onclick={loadInvites} disabled={invitesLoading}>
						刷新
					</Button>
				</div>
				{#if invitesLoading && invites.length === 0}
					<p class="text-sm text-muted-foreground">加载中...</p>
				{:else if invites.length === 0}
					<p class="text-sm text-muted-foreground">暂无待处理邀请</p>
				{:else}
					<div class="space-y-2">
						{#each invites as invite (invite.invite_id)}
							<div class="rounded-2xl border border-border/60 bg-background/65 p-3">
								<p class="text-sm font-medium">{invite.group.name}</p>
								<p class="mt-1 text-xs text-muted-foreground">
									邀请人：{invite.inviter.nickname}
								</p>
								{#if invite.message}
									<p class="mt-1 text-xs text-muted-foreground">{invite.message}</p>
								{/if}
								<div class="mt-2 flex gap-2">
									<Button
										size="sm"
										class="rounded-xl"
										disabled={isInviteProcessing(invite.invite_id)}
										onclick={() => handleInvite(invite, 'accept')}
									>
										接受
									</Button>
									<Button
										variant="outline"
										size="sm"
										class="rounded-xl border-border/60 bg-background/70"
										disabled={isInviteProcessing(invite.invite_id)}
										onclick={() => handleInvite(invite, 'reject')}
									>
										拒绝
									</Button>
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</CardContent>
		</Card>

		<Card class="rounded-3xl border border-border/60 bg-card/80 shadow-sm backdrop-blur-md">
			<CardContent class="space-y-3 p-4">
				<div class="flex items-center justify-between">
					<div class="flex items-center gap-2">
						<CheckCircle2 class="h-4 w-4 text-muted-foreground" />
						<p class="text-sm font-semibold">我的入群申请</p>
					</div>
					<Button
						variant="outline"
						size="sm"
						class="rounded-2xl border-border/60 bg-background/70"
						onclick={loadApplications}
						disabled={applicationsLoading}
					>
						刷新
					</Button>
				</div>
				{#if applicationsLoading && applications.length === 0}
					<p class="text-sm text-muted-foreground">加载中...</p>
				{:else if applications.length === 0}
					<p class="text-sm text-muted-foreground">暂无待处理申请</p>
				{:else}
					<div class="space-y-2">
						{#each applications as application (application.application_id)}
							<div class="rounded-2xl border border-border/60 bg-background/65 p-3">
								<p class="text-sm font-medium">群组申请</p>
								{#if application.message}
									<p class="mt-1 text-xs text-muted-foreground">{application.message}</p>
								{/if}
								<p class="mt-1 text-xs text-muted-foreground">
									申请时间：{formatDateTime(application.created_at)}
								</p>
								<div class="mt-2 flex gap-2">
									<Button
										variant="outline"
										size="sm"
										class="rounded-xl border-border/60 bg-background/70"
										disabled={isApplicationProcessing(application.application_id)}
										onclick={() => handleCancelApplication(application)}
									>
										取消申请
									</Button>
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</CardContent>
		</Card>
	</div>
</div>

<CreateGroupModal bind:open={showCreateGroup} onCreated={loadGroups} />
