<script lang="ts">
	import { onMount } from 'svelte';
	import { adminUsersService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
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

	$effect(() => { if (pagination.error) toast.error(pagination.error); });

	async function handleBan(userId: string) {
		try {
			const response = await adminUsersService.ban({ user_id: userId, reason: '违规操作' });
			if (isSuccess(response)) {
				pagination.updateItem((u) => u.user.user_id === userId ? { ...u, status: 'banned' as const } : null);
				toast.success('已封禁');
			} else { toast.error(response.message || '操作失败'); }
		} catch { toast.error('操作失败'); }
	}

	async function handleUnban(userId: string) {
		try {
			const response = await adminUsersService.unban({ user_id: userId });
			if (isSuccess(response)) {
				pagination.updateItem((u) => u.user.user_id === userId ? { ...u, status: 'normal' as const } : null);
				toast.success('已解封');
			} else { toast.error(response.message || '操作失败'); }
		} catch { toast.error('操作失败'); }
	}

	const STATUS_TEXT: Record<string, string> = { normal: '正常', banned: '已封禁' };

	onMount(() => { loadUsers(); });
</script>

<svelte:head>
	<title>用户管理 - HotSpot 管理后台</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 style="font-size: 22px; font-weight: 600; color: var(--neu-text);">用户管理</h1>
		<button class="flex items-center gap-1.5 rounded-full px-4 py-2 outline-none" style="background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); font-size: 13px; color: var(--neu-text);" onclick={() => loadUsers()} disabled={pagination.loading}>
			<RefreshCw size={14} strokeWidth={1.5} class={pagination.loading ? 'animate-spin' : ''} />
			刷新
		</button>
	</div>

	<!-- Search -->
	<div class="flex gap-2">
		<div class="relative flex-1 flex items-center gap-3 rounded-[18px] px-4 py-2.5" style="box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);">
			<Search size={15} strokeWidth={1.5} color="var(--neu-text-muted)" />
			<input type="search" placeholder="搜索昵称..." class="flex-1 bg-transparent outline-none" style="font-size: 13px; color: var(--neu-text);" autocomplete="off" bind:value={nickname} onkeydown={(e) => e.key === 'Enter' && loadUsers()} />
		</div>
		<button class="rounded-full px-4 py-2.5 outline-none" style="background: var(--neu-accent-primary); color: var(--neu-active-text); font-size: 13px; box-shadow: 0 3px 10px var(--neu-glow);" onclick={() => loadUsers()}>搜索</button>
	</div>

	<!-- Users table -->
	<div class="rounded-[24px] overflow-hidden" style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);">
		<div class="flex items-center gap-2 px-4 py-3" style="border-bottom: 1px solid var(--neu-border);">
			<Users size={16} strokeWidth={1.5} color="var(--neu-accent-primary)" />
			<span style="font-size: 14px; font-weight: 500; color: var(--neu-text);">用户列表</span>
		</div>
		<div class="p-4">
			{#if pagination.loading && pagination.items.length === 0}
				<div class="space-y-3">
					{#each Array(5) as _, i (i)}
						<div class="flex items-center gap-4">
							<div class="rounded-full" style="width: 40px; height: 40px; background: var(--neu-border);"></div>
							<div class="flex-1 space-y-2">
								<div class="rounded" style="height: 12px; width: 120px; background: var(--neu-border);"></div>
								<div class="rounded" style="height: 10px; width: 80px; background: var(--neu-border);"></div>
							</div>
						</div>
					{/each}
				</div>
			{:else if pagination.items.length === 0}
				<p class="py-8 text-center" style="font-size: 13px; color: var(--neu-text-muted);">暂无用户数据</p>
			{:else}
				<div class="space-y-2">
					{#each pagination.items as user (user.user.user_id)}
						<div class="flex items-center gap-3 rounded-[18px] p-3" style="background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border);">
							<div class="flex-shrink-0 overflow-hidden rounded-full" style="width: 40px; height: 40px; border: 1.5px solid var(--neu-border);">
								{#if user.user.avatar_url}
									<img src={user.user.avatar_url} alt={user.user.nickname} class="h-full w-full object-cover" />
								{:else}
									<div class="flex h-full w-full items-center justify-center" style="background: var(--neu-fallback-bg);">
										<span style="font-size: 14px; font-weight: 600; color: var(--neu-fallback-text);">{user.user.nickname.slice(0, 2)}</span>
									</div>
								{/if}
							</div>
							<div class="min-w-0 flex-1">
								<span style="font-size: 13px; font-weight: 500; color: var(--neu-text);">{user.user.nickname}</span>
								<p style="font-size: 11px; color: var(--neu-text-muted); margin-top: 1px;">ID: {user.user.user_id}</p>
							</div>
							<span class="rounded-full px-2 py-0.5" style="font-size: 10px; font-weight: 500; {user.status === 'banned' ? 'background: var(--neu-error); color: var(--neu-active-text);' : 'background: var(--neu-accent-primary); color: var(--neu-active-text);'}">{STATUS_TEXT[user.status] ?? user.status}</span>
							<DropdownMenu.Root>
								<DropdownMenu.Trigger>
									{#snippet child({ props })}
										<button class="flex items-center justify-center rounded-full outline-none" style="width: 30px; height: 30px; background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border);" aria-label="更多操作" {...props}>
											<MoreVertical size={14} strokeWidth={1.5} color="var(--neu-text-muted)" />
										</button>
									{/snippet}
								</DropdownMenu.Trigger>
								<DropdownMenu.Content align="end">
									<DropdownMenu.Item>
										{#snippet child({ props })}
											<a href="/users/{user.user.user_id}" style="display: flex; align-items: center; gap: 6px; font-size: 13px; color: var(--neu-text);" {...props}>
												<Eye size={14} strokeWidth={1.5} />
												查看详情
											</a>
										{/snippet}
									</DropdownMenu.Item>
									<DropdownMenu.Item>
										{#snippet child({ props })}
											<a href="/admin/users/{user.user.user_id}/roles" style="display: flex; align-items: center; gap: 6px; font-size: 13px; color: var(--neu-text);" {...props}>
												<Shield size={14} strokeWidth={1.5} />
												管理角色
											</a>
										{/snippet}
									</DropdownMenu.Item>
									<DropdownMenu.Separator />
									{#if user.status === 'banned'}
										<DropdownMenu.Item>
											{#snippet child({ props })}
												<button style="display: flex; align-items: center; gap: 6px; font-size: 13px; color: var(--neu-text);" {...props} onclick={() => handleUnban(user.user.user_id)}>
													<UserX size={14} strokeWidth={1.5} />
													解除封禁
												</button>
											{/snippet}
										</DropdownMenu.Item>
									{:else}
										<DropdownMenu.Item class="text-destructive">
											{#snippet child({ props })}
												<button style="display: flex; align-items: center; gap: 6px; font-size: 13px;" {...props} onclick={() => handleBan(user.user.user_id)}>
													<Ban size={14} strokeWidth={1.5} />
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
					<button class="mt-4 w-full rounded-full py-2.5 outline-none" style="background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); font-size: 13px; color: var(--neu-text);" onclick={() => loadUsers(false)}>加载更多</button>
				{/if}
			{/if}
		</div>
	</div>
</div>
