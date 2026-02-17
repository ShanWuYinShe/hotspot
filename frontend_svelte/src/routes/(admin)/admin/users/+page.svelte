<script lang="ts">
	import { onMount } from 'svelte';
	import { adminUsersService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import type { UserItem } from '$lib/types/api';
	import Search from 'lucide-svelte/icons/search';
	import RefreshCw from 'lucide-svelte/icons/refresh-cw';
	import MoreVertical from 'lucide-svelte/icons/more-vertical';
	import Ban from 'lucide-svelte/icons/ban';
	import UserX from 'lucide-svelte/icons/user-x';
	import Eye from 'lucide-svelte/icons/eye';
	import Shield from 'lucide-svelte/icons/shield';
	import Users from 'lucide-svelte/icons/users';

	// ============ 分页 ============

	type AdminUserItem = UserItem & { status: 'normal' | 'banned' };

	const pagination = usePagination<AdminUserItem>({
		uniqueKey: (item) => item.user.user_id
	});

	let nickname = $state('');

	function loadUsers(refresh = true) {
		pagination.load(
			(cursor) =>
				adminUsersService.search({
					nickname: nickname || undefined,
					pagination: { limit: 20, cursor }
				}),
			refresh
		);
	}

	$effect(() => {
		if (pagination.error) toast.error(pagination.error);
	});

	// ============ 操作 ============

	async function handleBan(userId: string) {
		try {
			const response = await adminUsersService.ban({ user_id: userId, reason: '违规操作' });
			if (isSuccess(response)) {
				pagination.updateItem((u) =>
					u.user.user_id === userId ? { ...u, status: 'banned' as const } : null
				);
				toast.success('已封禁');
			} else {
				toast.error(response.message || '操作失败');
			}
		} catch {
			toast.error('操作失败');
		}
	}

	async function handleUnban(userId: string) {
		try {
			const response = await adminUsersService.unban({ user_id: userId });
			if (isSuccess(response)) {
				pagination.updateItem((u) =>
					u.user.user_id === userId ? { ...u, status: 'normal' as const } : null
				);
				toast.success('已解封');
			} else {
				toast.error(response.message || '操作失败');
			}
		} catch {
			toast.error('操作失败');
		}
	}

	// ============ 常量 ============

	type BadgeVariant = 'default' | 'secondary' | 'destructive' | 'outline';

	const USER_STATUS_BADGE: Record<string, { variant: BadgeVariant; text: string }> = {
		normal: { variant: 'default', text: '正常' },
		banned: { variant: 'destructive', text: '已封禁' }
	};

	onMount(() => {
		loadUsers();
	});
</script>

<svelte:head>
	<title>用户管理 - HotSpot 管理后台</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-bold">用户管理</h1>
		<Button variant="outline" size="sm" onclick={() => loadUsers()} disabled={pagination.loading}>
			<RefreshCw class="mr-2 h-4 w-4 {pagination.loading ? 'animate-spin' : ''}" />
			刷新
		</Button>
	</div>

	<!-- Search -->
	<div class="flex gap-2">
		<div class="relative flex-1">
			<Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
			<Input
				type="search"
				placeholder="搜索昵称..."
				class="pl-9"
				autocomplete="off"
				bind:value={nickname}
				onkeydown={(e) => e.key === 'Enter' && loadUsers()}
			/>
		</div>
		<Button onclick={() => loadUsers()}>搜索</Button>
	</div>

	<!-- Users table -->
	<Card>
		<CardHeader>
			<CardTitle class="flex items-center gap-2">
				<Users class="h-5 w-5" />
				用户列表
			</CardTitle>
		</CardHeader>
		<CardContent>
			{#if pagination.loading && pagination.items.length === 0}
				<div class="space-y-4">
					{#each Array(5) as _, i (i)}
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
			{:else if pagination.items.length === 0}
				<div class="py-8 text-center text-muted-foreground">暂无用户数据</div>
			{:else}
				<div class="divide-y">
					{#each pagination.items as user (user.user.user_id)}
						{@const badge = USER_STATUS_BADGE[user.status] ?? { variant: 'outline' as const, text: user.status }}
						<div class="flex items-center gap-4 py-4">
							<Avatar class="h-10 w-10">
								<AvatarImage src={user.user.avatar_url} alt={user.user.nickname} />
								<AvatarFallback>{user.user.nickname.slice(0, 2)}</AvatarFallback>
							</Avatar>
							<div class="min-w-0 flex-1">
								<div class="flex items-center gap-2">
									<p class="font-medium">{user.user.nickname}</p>
								</div>
								<div class="flex items-center gap-3 text-xs text-muted-foreground">
									<span>ID: {user.user.user_id}</span>
								</div>
							</div>
							<Badge variant={badge.variant}>
								{badge.text}
							</Badge>
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
											<a href="/users/{user.user.user_id}" {...props}>
												<Eye class="mr-2 h-4 w-4" />
												查看详情
											</a>
										{/snippet}
									</DropdownMenu.Item>
									<DropdownMenu.Item>
										{#snippet child({ props })}
											<a href="/admin/users/{user.user.user_id}/roles" {...props}>
												<Shield class="mr-2 h-4 w-4" />
												管理角色
											</a>
										{/snippet}
									</DropdownMenu.Item>
									<DropdownMenu.Separator />
									{#if user.status === 'banned'}
										<DropdownMenu.Item>
											{#snippet child({ props })}
												<button {...props} onclick={() => handleUnban(user.user.user_id)}>
													<UserX class="mr-2 h-4 w-4" />
													解除封禁
												</button>
											{/snippet}
										</DropdownMenu.Item>
									{:else}
										<DropdownMenu.Item class="text-destructive">
											{#snippet child({ props })}
												<button {...props} onclick={() => handleBan(user.user.user_id)}>
													<Ban class="mr-2 h-4 w-4" />
													封禁用户
												</button>
											{/snippet}
										</DropdownMenu.Item>
									{/if}
								</DropdownMenu.Content>
							</DropdownMenu.Root>
						</div>
					{/each}
				</div>

			{#if pagination.hasMore}
				<Button variant="outline" class="mt-4 w-full" onclick={() => loadUsers(false)}>
					加载更多
				</Button>
			{/if}
			{/if}
		</CardContent>
	</Card>
</div>
