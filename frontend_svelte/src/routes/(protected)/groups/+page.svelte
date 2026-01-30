<script lang="ts">
	import { onMount } from 'svelte';
	import { groupsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import CreateGroupModal from '$lib/components/common/create-group-modal.svelte';
	import type { MyGroupItem } from '$lib/types/api';
	import Plus from 'lucide-svelte/icons/plus';
	import Users from 'lucide-svelte/icons/users';
	import RefreshCw from 'lucide-svelte/icons/refresh-cw';

	let groups = $state<MyGroupItem[]>([]);
	let isLoading = $state(true);
	let showCreateGroup = $state(false);

	async function loadGroups() {
		isLoading = true;
		try {
			const response = await groupsService.getMyGroups({ limit: 50 });
			if (isSuccess(response) && response.data) {
				groups = response.data.items;
			}
		} catch {
			toast.error('加载失败');
		} finally {
			isLoading = false;
		}
	}

	onMount(() => {
		loadGroups();
	});
</script>

<svelte:head>
	<title>我的群组 - HotSpot</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-bold">我的群组</h1>
		<div class="flex gap-2">
			<Button variant="outline" size="sm" onclick={loadGroups} disabled={isLoading}>
				<RefreshCw class="mr-2 h-4 w-4 {isLoading ? 'animate-spin' : ''}" />
				刷新
			</Button>
			<Button size="sm" onclick={() => (showCreateGroup = true)}>
				<Plus class="mr-2 h-4 w-4" />
				创建群组
			</Button>
		</div>
	</div>

	{#if isLoading}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each Array(6) as _}
				<Card>
					<CardContent class="flex items-center gap-4 p-4">
						<Skeleton class="h-12 w-12 rounded-full" />
						<div class="flex-1 space-y-2">
							<Skeleton class="h-4 w-24" />
							<Skeleton class="h-3 w-16" />
						</div>
					</CardContent>
				</Card>
			{/each}
		</div>
	{:else if groups.length === 0}
		<Card>
			<CardContent class="flex flex-col items-center justify-center py-12">
				<Users class="mb-4 h-12 w-12 text-muted-foreground" />
				<p class="text-lg font-medium">还没有加入群组</p>
				<p class="mt-1 text-muted-foreground">去探索或创建一个群组吧</p>
				<div class="mt-4 flex gap-3">
					<a href="/explore?tab=nearby&nearby=groups">
						<Button variant="outline">发现附近群组</Button>
					</a>
					<Button onclick={() => (showCreateGroup = true)}>创建群组</Button>
				</div>
			</CardContent>
		</Card>
	{:else}
		<div class="columns-1 gap-4 sm:columns-2 xl:columns-3">
			{#each groups as group (group.group_chat_id)}
				<div class="mb-4 break-inside-avoid">
					<a href="/groups/{group.group_chat_id}">
						<Card class="cursor-pointer border-0 bg-card/70 shadow-sm shadow-black/5 transition-shadow hover:shadow-lg hover:shadow-black/10">
							<CardContent class="flex items-center gap-4 p-4">
								<Avatar class="h-12 w-12">
									<AvatarImage src={group.avatar_url} alt={group.name} />
									<AvatarFallback>{group.name.slice(0, 2)}</AvatarFallback>
								</Avatar>
								<div class="min-w-0 flex-1">
									<div class="flex items-center justify-between">
										<p class="truncate font-medium">{group.name}</p>
										{#if group.unread_count > 0}
											<Badge variant="destructive">
												{group.unread_count > 99 ? '99+' : group.unread_count}
											</Badge>
										{/if}
									</div>
									<p class="text-sm text-muted-foreground">
										{group.members_count} 成员
									</p>
								</div>
							</CardContent>
						</Card>
					</a>
				</div>
			{/each}
		</div>
	{/if}
</div>

<CreateGroupModal bind:open={showCreateGroup} onCreated={loadGroups} />
