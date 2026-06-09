<script lang="ts">
  import { onMount } from 'svelte';
  import { adminGroupsService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { toast } from '$lib/stores/toast.svelte';
  import { usePagination } from '$lib/hooks/use-pagination.svelte';
  import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
  import ConfirmDialog from '$lib/components/common/confirm-dialog.svelte';
  import type { AdminGroupItem } from '$lib/types/api';
  import { formatDate } from '$lib/utils/format';
  import Search from 'lucide-svelte/icons/search';
  import RefreshCw from 'lucide-svelte/icons/refresh-cw';
  import MoreVertical from 'lucide-svelte/icons/more-vertical';
  import Eye from 'lucide-svelte/icons/eye';
  import Trash2 from 'lucide-svelte/icons/trash-2';
  import UsersRound from 'lucide-svelte/icons/users-round';
  import Users from 'lucide-svelte/icons/users';
  import Lock from 'lucide-svelte/icons/lock';

  const pagination = usePagination<AdminGroupItem>({
    uniqueKey: (item) => item.group.group_id
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

  $effect(() => {
    if (pagination.error) toast.error(pagination.error);
  });

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
        toast.success('已解散群组');
      } else {
        toast.error(response.message || '操作失败');
      }
    } catch {
      toast.error('操作失败');
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

<div class="space-y-6">
  <div class="flex items-center justify-between">
    <h1 style="font-size: 22px; font-weight: 600; color: var(--text-primary);">群组管理</h1>
    <button
      class="flex items-center gap-1.5 rounded-full px-4 py-2 outline-none"
      style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme); font-size: 13px; color: var(--text-primary);"
      onclick={() => loadGroups()}
      disabled={pagination.loading}
    >
      <RefreshCw size={14} strokeWidth={1.5} class={pagination.loading ? 'animate-spin' : ''} />
      刷新
    </button>
  </div>

  <!-- Search -->
  <div class="flex gap-2">
    <div class="input-surface relative flex-1 flex items-center gap-3 rounded-[18px] px-4 py-2.5">
      <Search size={15} strokeWidth={1.5} color="var(--text-secondary)" class="flex-shrink-0" />
      <input
        type="search"
        placeholder="搜索群组名称..."
        class="flex-1 bg-transparent outline-none"
        style="font-size: 13px; color: var(--text-primary);"
        autocomplete="off"
        bind:value={name}
        onkeydown={(e) => e.key === 'Enter' && loadGroups()}
      />
    </div>
    <button
      class="rounded-full px-4 py-2.5 outline-none"
      style="background: var(--brand); color: var(--text-on-accent); font-size: 13px; box-shadow: 0 3px 10px var(--glow);"
      onclick={() => loadGroups()}>搜索</button
    >
  </div>

  <!-- Groups table -->
  <div
    class="rounded-[22px] overflow-hidden"
    style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
  >
    <div
      class="flex items-center gap-2 px-4 py-3"
      style="border-bottom: 1px solid var(--border-theme);"
    >
      <UsersRound size={16} strokeWidth={1.5} color="var(--brand)" />
      <span style="font-size: 14px; font-weight: 500; color: var(--text-primary);">群组列表</span>
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
        <p class="py-8 text-center" style="font-size: 13px; color: var(--text-secondary);">
          暂无群组数据
        </p>
      {:else}
        <div class="space-y-2">
          {#each pagination.items as group (group.group.group_id)}
            <div
              class="flex items-center gap-3 rounded-[18px] p-3"
              style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme);"
            >
              <div
                class="flex-shrink-0 overflow-hidden rounded-full"
                style="width: 40px; height: 40px; border: 1.5px solid var(--border-theme);"
              >
                {#if group.group.avatar_url}
                  <img
                    src={group.group.avatar_url}
                    alt={group.group.name}
                    class="h-full w-full object-cover"
                  />
                {:else}
                  <div
                    class="flex h-full w-full items-center justify-center"
                    style="background: var(--surface-secondary);"
                  >
                    <span style="font-size: 14px; font-weight: 600; color: var(--text-tertiary);"
                      >{group.group.name.slice(0, 2)}</span
                    >
                  </div>
                {/if}
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-1">
                  <span style="font-size: 13px; font-weight: 500; color: var(--text-primary);"
                    >{group.group.name}</span
                  >
                  {#if group.has_password}
                    <Lock size={12} strokeWidth={1.5} color="var(--text-secondary)" />
                  {/if}
                </div>
                <div class="flex items-center gap-2 mt-0.5">
                  <div class="flex items-center gap-0.5">
                    <Users size={11} strokeWidth={1.5} color="var(--text-secondary)" />
                    <span style="font-size: 11px; color: var(--text-secondary);"
                      >{group.group.members_count}</span
                    >
                  </div>
                  {#if group.created_at}
                    <span style="font-size: 11px; color: var(--text-secondary);"
                      >创建: {formatDate(group.created_at)}</span
                    >
                  {/if}
                </div>
              </div>
              <span
                class="rounded-full px-2 py-0.5"
                style="font-size: 10px; background: var(--brand); color: var(--text-on-accent);"
                >正常</span
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
                        href="/groups/{group.group.group_id}"
                        style="display: flex; align-items: center; gap: 6px; font-size: 13px; color: var(--text-primary);"
                        {...props}
                      >
                        <Eye size={14} strokeWidth={1.5} />
                        查看详情
                      </a>
                    {/snippet}
                  </DropdownMenu.Item>
                  <DropdownMenu.Separator />
                  <DropdownMenu.Item class="text-destructive">
                    {#snippet child({ props })}
                      <button
                        style="display: flex; align-items: center; gap: 6px; font-size: 13px;"
                        {...props}
                        onclick={() => requestDissolve(group.group.group_id)}
                      >
                        <Trash2 size={14} strokeWidth={1.5} />
                        解散群组
                      </button>
                    {/snippet}
                  </DropdownMenu.Item>
                </DropdownMenu.Content>
              </DropdownMenu.Root>
            </div>
          {/each}
        </div>
        {#if pagination.hasMore}
          <button
            class="mt-4 w-full rounded-full py-2.5 outline-none"
            style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme); font-size: 13px; color: var(--text-primary);"
            onclick={() => loadGroups(false)}>加载更多</button
          >
        {/if}
      {/if}
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
