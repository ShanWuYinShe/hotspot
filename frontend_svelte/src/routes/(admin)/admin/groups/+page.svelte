<script lang="ts">
	import { onMount } from 'svelte';
	import { adminGroupsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import type { AdminGroupSearchItem } from '$lib/types/api';
	import { formatDate } from '$lib/utils/format';
	import Search from 'lucide-svelte/icons/search';
	import RefreshCw from 'lucide-svelte/icons/refresh-cw';
	import MoreVertical from 'lucide-svelte/icons/more-vertical';
	import Eye from 'lucide-svelte/icons/eye';
	import Trash2 from 'lucide-svelte/icons/trash-2';
	import UsersRound from 'lucide-svelte/icons/users-round';
	import Users from 'lucide-svelte/icons/users';
	import Lock from 'lucide-svelte/icons/lock';

	let groups = $state.raw<AdminGroupSearchItem[]>([]);
	let isLoading = $state(true);
	let cursor = $state<string | undefined>(undefined);
	let hasMore = $state(false);
	let name = $state('');

	async function loadGroups(append = false) {
		if (!append) isLoading = true;
		try {
			const response = await adminGroupsService.search({
				name: name || undefined,
				pagination: {
					limit: 20,
					cursor: append ? cursor : undefined
				}
			});

			if (isSuccess(response) && response.data) {
				if (append) {
					groups = [...groups, ...response.data.items];
				} else {
					groups = response.data.items;
				}
				cursor = response.data.pagination.next_cursor;
				hasMore = response.data.pagination.has_more;
			}
		} catch {
			toast.error('加载失败');
		} finally {
			isLoading = false;
		}
	}

	async function handleDissolve(groupId: string) {
		if (!confirm('确定要解散该群组吗？此操作不可逆。')) return;

		try {
			const response = await adminGroupsService.dismiss({ group_id: groupId, reason: '管理员操作' });
			if (isSuccess(response)) {
				groups = groups.filter(g => g.group.group_id !== groupId);
				toast.success('已解散群组');
			} else {
				toast.error(response.message || '操作失败');
			}
		} catch {
			toast.error('操作失败');
		}
	}

	onMount(() => {
		loadGroups();
	});
</script>

<svelte:head>
	<title>群组管理 - HotSpot 管理后台</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-bold">群组管理</h1>
		<Button variant="outline" size="sm" onclick={() => loadGroups()} disabled={isLoading}>
			<RefreshCw class="mr-2 h-4 w-4 {isLoading ? 'animate-spin' : ''}" />
			刷新
		</Button>
	</div>

	<!-- Search -->
	<div class="flex gap-2">
		<div class="relative flex-1">
			<Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
			<Input
				type="search"
				placeholder="搜索群组名称..."
				class="pl-9"
				autocomplete="off"
				bind:value={name}
				onkeydown={(e) => e.key === 'Enter' && loadGroups()}
			/>
		</div>
		<Button onclick={() => loadGroups()}>搜索</Button>
	</div>

	<!-- Groups table -->
	<Card>
		<CardHeader>
			<CardTitle class="flex items-center gap-2">
				<UsersRound class="h-5 w-5" />
				群组列表
			</CardTitle>
		</CardHeader>
		<CardContent>
			{#if isLoading && groups.length === 0}
				<div class="space-y-4">
					{#each Array(5) as _}
						<div class="flex items-center gap-4">
							<Skeleton class="h-10 w-10 rounded-full" />
							<div class="flex-1 space-y-2">
								<Skeleton class="h-4 w-32" />
								<Skeleton class="h-3 w-24" />
							</div>
							<Skeleton class="h-6 w-16" />
						</div>
					{/each}
				</div>
			{:else if groups.length === 0}
				<div class="py-8 text-center text-muted-foreground">暂无群组数据</div>
			{:else}
				<div class="divide-y">
					{#each groups as group (group.group.group_id)}
						<div class="flex items-center gap-4 py-4">
							<Avatar class="h-10 w-10">
								<AvatarImage src={group.group.avatar_url} alt={group.group.name} />
								<AvatarFallback>{group.group.name.slice(0, 2)}</AvatarFallback>
							</Avatar>
							<div class="min-w-0 flex-1">
								<div class="flex items-center gap-2">
									<p class="font-medium">{group.group.name}</p>
									{#if group.has_password}
										<Lock class="h-3 w-3 text-muted-foreground" />
									{/if}
								</div>
								<div class="flex items-center gap-3 text-xs text-muted-foreground">
									<span class="flex items-center gap-1">
										<Users class="h-3 w-3" />
										{group.group.members_count}
									</span>
									{#if group.created_at}
										<span>创建: {formatDate(group.created_at)}</span>
									{/if}
								</div>
							</div>
							<Badge variant="default">正常</Badge>
							<DropdownMenu.Root>
								<DropdownMenu.Trigger>
									{#snippet child({ props })}
										<Button variant="ghost" size="icon" aria-label="更多操作" {...props}>
											<MoreVertical class="h-4 w-4" />
										</Button>
									{/snippet}
								</DropdownMenu.Trigger>
								<DropdownMenu.Content align="end">
									<DropdownMenu.Item>
										{#snippet child({ props })}
											<a href="/groups/{group.group.group_id}" {...props}>
												<Eye class="mr-2 h-4 w-4" />
												查看详情
											</a>
										{/snippet}
									</DropdownMenu.Item>
									<DropdownMenu.Separator />
									<DropdownMenu.Item class="text-destructive">
										{#snippet child({ props })}
											<button {...props} onclick={() => handleDissolve(group.group.group_id)}>
												<Trash2 class="mr-2 h-4 w-4" />
												解散群组
											</button>
										{/snippet}
									</DropdownMenu.Item>
								</DropdownMenu.Content>
							</DropdownMenu.Root>
						</div>
					{/each}
				</div>

				{#if hasMore}
					<Button variant="outline" class="mt-4 w-full" onclick={() => loadGroups(true)}>
						加载更多
					</Button>
				{/if}
			{/if}
		</CardContent>
	</Card>
</div>
