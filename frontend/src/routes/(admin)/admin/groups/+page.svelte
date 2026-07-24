<script lang="ts">
	  import { onMount } from 'svelte';
	  import { adminGroupsService } from '$lib/api/services';
	  import { isSuccess } from '$lib/api/client';
	  import { toast } from '$lib/stores/toast.svelte';
	  import { t } from '$lib/i18n';
  import { usePagination } from '$lib/hooks/use-pagination.svelte';
  import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuSeparator,
    DropdownMenuTrigger
  } from '$lib/components/ui/dropdown-menu';
  import ConfirmDialog from '$lib/components/common/confirm-dialog.svelte';
  import Avatar from '$lib/components/common/avatar.svelte';
  import type { AdminGroupItem } from '$lib/types/api';
  import { formatDate } from '$lib/utils/format';
  import AdminPageHeader from '$lib/components/common/admin-page-header.svelte';
  import Search from 'lucide-svelte/icons/search';
  import RefreshCw from 'lucide-svelte/icons/refresh-cw';
  import MoreHorizontal from 'lucide-svelte/icons/more-horizontal';
  import Eye from 'lucide-svelte/icons/eye';
  import Trash2 from 'lucide-svelte/icons/trash-2';
  import UsersRound from 'lucide-svelte/icons/users-round';
  import Users from 'lucide-svelte/icons/users';
  import Lock from 'lucide-svelte/icons/lock';

  const pagination = usePagination<AdminGroupItem>({
    uniqueKey: (item) => item.group.group_id,
    onError: (msg) => toast.error(msg)
  });

  let name = $state('');
  let dissolveConfirmOpen = $state(false);
  let pendingDissolveId = $state<string | null>(null);

  function loadGroups(refresh = true) {
    pagination.load(
      (cursor) =>
        adminGroupsService.search({
          name: name || undefined,
          pagination: { limit: 20, cursor }
        }),
      refresh
    );
  }

  function requestDissolve(groupId: string) {
    pendingDissolveId = groupId;
    dissolveConfirmOpen = true;
  }

  async function handleDissolve() {
    if (!pendingDissolveId) return;
    try {
      const response = await adminGroupsService.dismiss({
        group_id: pendingDissolveId,
        reason: '管理员操作'
      });
      if (isSuccess(response)) {
        const dissolvedId = pendingDissolveId;
	        pagination.removeItem((g) => g.group.group_id === dissolvedId);
	        toast.success(t('admin.groupDismissed'));
	      } else {
	        toast.error(response.message || t('admin.operationFailed'));
	      }
	    } catch {
	      toast.error(t('admin.operationFailed'));
    } finally {
      pendingDissolveId = null;
    }
  }

  onMount(() => {
    loadGroups();
  });
</script>

<svelte:head>
  <title>群组管理 - HotSpot 管理后台</title>
</svelte:head>

<AdminPageHeader title="群组管理" description="查看与管理群组">
  {#snippet actions()}
    <button
      class="text-body flex items-center gap-1.5 rounded-full px-4 py-2 outline-none"
      style="background: var(--surface-bg); border: 1px solid var(--border-theme); color: var(--text-primary)"
      onclick={() => loadGroups()}
      disabled={pagination.loading}
    >
      <RefreshCw size={14} strokeWidth={1.5} class={pagination.loading ? 'animate-spin' : ''} />
      刷新
    </button>
  {/snippet}
</AdminPageHeader>

<div class="page-pad">
  <div class="page-shell page-shell--wide space-y-6">
    <!-- Search -->
    <div class="flex gap-2">
      <div
        class="input-surface relative flex-1 flex items-center gap-3 rounded-[var(--radius-md)] px-4 py-2.5"
      >
        <Search size={15} strokeWidth={1.5} color="var(--text-secondary)" class="flex-shrink-0" />
        <input
          type="search"
          placeholder="搜索群组名称..."
          class="text-body flex-1 bg-transparent outline-none"
          style="color: var(--text-primary)"
          autocomplete="off"
          bind:value={name}
          onkeydown={(e) => e.key === 'Enter' && loadGroups()}
        />
      </div>
      <button
        class="text-body rounded-full px-4 py-2.5 outline-none"
        style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn)"
        onclick={() => loadGroups()}>搜索</button
      >
    </div>

    <!-- Groups table -->
    <div
      class="rounded-[var(--radius-lg)] overflow-hidden"
      style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
    >
      <div
        class="flex items-center gap-2 px-4 py-3"
        style="border-bottom: 1px solid var(--border-theme);"
      >
        <UsersRound size={16} strokeWidth={1.5} color="var(--brand)" />
        <span class="text-body-lg" style="font-weight: 500; color: var(--text-primary)"
          >群组列表</span
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
            暂无群组数据
          </p>
        {:else}
          <div class="space-y-2">
            {#each pagination.items as group (group.group.group_id)}
              <div
                class="flex items-center gap-3 rounded-[var(--radius-md)] p-3"
                style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
              >
                <Avatar
                  src={group.group.avatar_url}
                  name={group.group.name}
                  alt={group.group.name}
                  size={40}
                  type="group"
                />
                <div class="min-w-0 flex-1">
                  <div class="flex items-center gap-1">
                    <span class="text-body" style="font-weight: 500; color: var(--text-primary)"
                      >{group.group.name}</span
                    >
                    {#if group.has_password}
                      <Lock size={12} strokeWidth={1.5} color="var(--text-secondary)" />
                    {/if}
                  </div>
                  <div class="flex items-center gap-2 mt-0.5">
                    <div class="flex items-center gap-0.5">
                      <Users size={11} strokeWidth={1.5} color="var(--text-secondary)" />
                      <span class="text-caption" style="color: var(--text-secondary)"
                        >{group.group.members_count}</span
                      >
                    </div>
                    {#if group.created_at}
                      <span class="text-caption" style="color: var(--text-secondary)"
                        >创建: {formatDate(group.created_at)}</span
                      >
                    {/if}
                  </div>
                </div>
                <span
                  class="text-xs rounded-full px-2 py-0.5"
                  style="background: var(--brand); color: var(--text-on-accent)">正常</span
                >
                <DropdownMenu>
                  <DropdownMenuTrigger
                    class="surface-flat inline-flex h-8 w-8 items-center justify-center rounded-full outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
                    aria-label="更多操作"
                  >
                    <MoreHorizontal size={16} />
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end" class="w-36">
                    <DropdownMenuItem>
                      <a
                        href="/groups/{group.group.group_id}"
                        class="flex items-center text-body"
                        style="color: var(--text-primary)"
                      >
                        <Eye class="mr-2 h-4 w-4" />查看详情
                      </a>
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem onclick={() => requestDissolve(group.group.group_id)} class="text-destructive">
                      <Trash2 class="mr-2 h-4 w-4" />解散群组
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            {/each}
          </div>
          {#if pagination.hasMore}
            <button
              class="text-body mt-4 w-full rounded-full py-2.5 outline-none"
              style="background: var(--surface-bg); border: 1px solid var(--border-theme); color: var(--text-primary)"
              onclick={() => loadGroups(false)}>加载更多</button
            >
          {/if}
        {/if}
      </div>
    </div>
  </div>
</div>

<ConfirmDialog
  bind:open={dissolveConfirmOpen}
  title="解散群组"
  description="确定要解散该群组吗？此操作不可逆，所有群成员将被移出。"
  confirmText="解散"
  variant="destructive"
  onconfirm={handleDissolve}
/>
