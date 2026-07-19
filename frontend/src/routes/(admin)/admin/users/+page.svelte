<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { adminUsersService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { toast } from '$lib/stores/toast.svelte';
  import { usePagination } from '$lib/hooks/use-pagination.svelte';
  import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
  import Avatar from '$lib/components/common/avatar.svelte';
  import type { UserItem } from '$lib/types/api';
  import AdminPageHeader from '$lib/components/common/admin-page-header.svelte';
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
    uniqueKey: (item) => item.user.user_id,
    onError: (msg) => toast.error(msg)
  });

  let nickname = $state('');
  let searchTimer: ReturnType<typeof setTimeout> | undefined;

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

  // 输入即搜（防抖 350ms），与发现 / 附近的即时搜索保持一致
  function handleSearchInput() {
    clearTimeout(searchTimer);
    searchTimer = setTimeout(() => loadUsers(), 350);
  }

  onDestroy(() => clearTimeout(searchTimer));

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

  const STATUS_TEXT: Record<string, string> = { normal: '正常', banned: '已封禁' };

  onMount(() => {
    loadUsers();
  });
</script>

<svelte:head>
  <title>用户管理 - HotSpot 管理后台</title>
</svelte:head>

<AdminPageHeader title="用户管理" description="搜索、封禁、解封用户">
  {#snippet actions()}
    <button
      class="text-body flex items-center gap-1.5 rounded-full px-4 py-2 outline-none"
      style="background: var(--surface-bg); border: 1px solid var(--border-theme); color: var(--text-primary)"
      onclick={() => loadUsers()}
      disabled={pagination.loading}
    >
      <RefreshCw size={14} strokeWidth={1.5} class={pagination.loading ? 'animate-spin' : ''} />
      刷新
    </button>
  {/snippet}
</AdminPageHeader>

<div class="page-pad">
  <div class="page-shell page-shell--wide space-y-6">
    <!-- Search — 输入即搜（防抖），与全站搜索交互一致 -->
    <div
      class="input-surface relative flex items-center gap-3 rounded-[var(--radius-md)] px-4 py-2.5"
    >
      <Search size={15} strokeWidth={1.5} color="var(--text-secondary)" class="flex-shrink-0" />
      <input
        type="search"
        placeholder="搜索昵称..."
        class="text-body flex-1 bg-transparent outline-none"
        style="color: var(--text-primary)"
        autocomplete="off"
        bind:value={nickname}
        oninput={handleSearchInput}
        onkeydown={(e) => {
          if (e.key === 'Enter') {
            clearTimeout(searchTimer);
            loadUsers();
          }
        }}
      />
    </div>

    <!-- Users table -->
    <div
      class="rounded-[var(--radius-lg)] overflow-hidden"
      style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
    >
      <div
        class="flex items-center gap-2 px-4 py-3"
        style="border-bottom: 1px solid var(--border-theme);"
      >
        <Users size={16} strokeWidth={1.5} color="var(--brand)" />
        <span class="text-body-lg" style="font-weight: 500; color: var(--text-primary)"
          >用户列表</span
        >
      </div>
      <div class="p-4">
        {#if pagination.loading && pagination.items.length === 0}
          <div class="space-y-3">
            {#each Array(5) as _, i (i)}
              <div class="flex items-center gap-4">
                <div
                  class="rounded-full"
                  style="width: 40px; height: 40px; background: var(--border-theme);"
                ></div>
                <div class="flex-1 space-y-2">
                  <div
                    class="rounded"
                    style="height: 12px; width: 120px; background: var(--border-theme);"
                  ></div>
                  <div
                    class="rounded"
                    style="height: 10px; width: 80px; background: var(--border-theme);"
                  ></div>
                </div>
              </div>
            {/each}
          </div>
        {:else if pagination.items.length === 0}
          <p class="text-body py-8 text-center" style="color: var(--text-secondary)">
            暂无用户数据
          </p>
        {:else}
          <div class="space-y-2">
            {#each pagination.items as user (user.user.user_id)}
              <div
                class="flex items-center gap-3 rounded-[var(--radius-md)] p-3"
                style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
              >
                <Avatar
                  src={user.user.avatar_url}
                  name={user.user.nickname}
                  alt={user.user.nickname}
                  size={40}
                />
                <div class="min-w-0 flex-1">
                  <span class="text-body" style="font-weight: 500; color: var(--text-primary)"
                    >{user.user.nickname}</span
                  >
                  <p class="text-caption" style="color: var(--text-secondary); margin-top: 1px">
                    ID: {user.user.user_id}
                  </p>
                </div>
                <span
                  class="text-xs rounded-full px-2 py-0.5"
                  style="font-weight: 500; {user.status === 'banned'
                    ? 'background: var(--theme-error); color: var(--text-on-accent);'
                    : 'background: var(--brand); color: var(--text-on-accent);'}"
                  >{STATUS_TEXT[user.status] ?? user.status}</span
                >
                <DropdownMenu.Root>
                  <DropdownMenu.Trigger>
                    {#snippet child({ props })}
                      <button
                        class="surface-flat flex items-center justify-center rounded-full outline-none transition-transform duration-150 active:scale-[0.88]"
                        style="width: 30px; height: 30px;"
                        aria-label="更多操作"
                        {...props}
                      >
                        <MoreVertical size={14} strokeWidth={1.5} color="var(--text-secondary)" />
                      </button>
                    {/snippet}
                  </DropdownMenu.Trigger>
                  <DropdownMenu.Content align="end">
                    <DropdownMenu.Item>
                      {#snippet child({ props })}
                        <a
                          href="/users/{user.user.user_id}"
                          class="text-body"
                          style="display: flex; align-items: center; gap: 6px; color: var(--text-primary)"
                          {...props}
                        >
                          <Eye size={14} strokeWidth={1.5} />
                          查看详情
                        </a>
                      {/snippet}
                    </DropdownMenu.Item>
                    <DropdownMenu.Item>
                      {#snippet child({ props })}
                        <a
                          href="/admin/rbac"
                          class="text-body"
                          style="display: flex; align-items: center; gap: 6px; color: var(--text-primary)"
                          {...props}
                        >
                          <Shield size={14} strokeWidth={1.5} />
                          管理角色
                        </a>
                      {/snippet}
                    </DropdownMenu.Item>
                    <DropdownMenu.Separator />
                    {#if user.status === 'banned'}
                      <DropdownMenu.Item>
                        {#snippet child({ props })}
                          <button
                            class="text-body"
                            style="display: flex; align-items: center; gap: 6px; color: var(--text-primary)"
                            {...props}
                            onclick={() => handleUnban(user.user.user_id)}
                          >
                            <UserX size={14} strokeWidth={1.5} />
                            解除封禁
                          </button>
                        {/snippet}
                      </DropdownMenu.Item>
                    {:else}
                      <DropdownMenu.Item class="text-destructive">
                        {#snippet child({ props })}
                          <button
                            class="text-body"
                            style="display: flex; align-items: center; gap: 6px"
                            {...props}
                            onclick={() => handleBan(user.user.user_id)}
                          >
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
            <button
              class="text-body mt-4 w-full rounded-full py-2.5 outline-none"
              style="background: var(--surface-bg); border: 1px solid var(--border-theme); color: var(--text-primary)"
              onclick={() => loadUsers(false)}>加载更多</button
            >
          {/if}
        {/if}
      </div>
    </div>
  </div>
</div>
