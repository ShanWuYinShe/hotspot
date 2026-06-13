<script lang="ts">
  import { onMount } from 'svelte';
  import { adminAuditService } from '$lib/api/services';
  import { toast } from '$lib/stores/toast.svelte';
  import { usePagination } from '$lib/hooks/use-pagination.svelte';
  import type { AuditLogItem, AuditAction } from '$lib/types/api';
  import { formatDateTime } from '$lib/utils/format';
  import AdminPageHeader from '$lib/components/common/admin-page-header.svelte';
  import Search from 'lucide-svelte/icons/search';
  import RefreshCw from 'lucide-svelte/icons/refresh-cw';
  import FileText from 'lucide-svelte/icons/file-text';
  import User from 'lucide-svelte/icons/user';
  import Clock from 'lucide-svelte/icons/clock';

  const pagination = usePagination<AuditLogItem>({ uniqueKey: 'log_id' });
  let actionFilter = $state('');

  function loadLogs(refresh = true) {
    pagination.load(
      (cursor) =>
        adminAuditService.getList({
          action: (actionFilter || undefined) as AuditAction | undefined,
          pagination: { limit: 30, cursor }
        }),
      refresh
    );
  }

  $effect(() => {
    if (pagination.error) toast.error(pagination.error);
  });

  onMount(() => {
    loadLogs();
  });
</script>

<svelte:head>
  <title>审计日志 - HotSpot 管理后台</title>
</svelte:head>

<AdminPageHeader title="审计日志" description="系统操作记录">
  {#snippet actions()}
    <button
      class="flex items-center gap-1.5 rounded-full px-4 py-2 outline-none"
      style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme); font-size: 13px; color: var(--text-primary);"
      onclick={() => loadLogs()}
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
      <div class="input-surface relative flex-1 flex items-center gap-3 rounded-[18px] px-4 py-2.5">
        <Search size={15} strokeWidth={1.5} color="var(--text-secondary)" class="flex-shrink-0" />
        <input
          type="search"
          placeholder="按操作类型筛选..."
          class="flex-1 bg-transparent outline-none"
          style="font-size: 13px; color: var(--text-primary);"
          autocomplete="off"
          bind:value={actionFilter}
          onkeydown={(e) => e.key === 'Enter' && loadLogs()}
        />
      </div>
      <button
        class="rounded-full px-4 py-2.5 outline-none"
        style="background: var(--brand); color: var(--text-on-accent); font-size: 13px; box-shadow: 0 3px 10px var(--glow);"
        onclick={() => loadLogs()}>筛选</button
      >
    </div>

    <!-- Audit logs -->
    <div
      class="rounded-[22px] overflow-hidden"
      style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
    >
      <div
        class="flex items-center gap-2 px-4 py-3"
        style="border-bottom: 1px solid var(--border-theme);"
      >
        <FileText size={16} strokeWidth={1.5} color="var(--brand)" />
        <span style="font-size: 14px; font-weight: 500; color: var(--text-primary);">操作日志</span>
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
                    style="height: 12px; width: 160px; background: var(--border-theme);"
                  ></div>
                  <div
                    class="rounded"
                    style="height: 10px; width: 100px; background: var(--border-theme);"
                  ></div>
                </div>
              </div>
            {/each}
          </div>
        {:else if pagination.items.length === 0}
          <p class="py-8 text-center" style="font-size: 13px; color: var(--text-secondary);">
            暂无审计日志
          </p>
        {:else}
          <div class="space-y-3">
            {#each pagination.items as log (log.log_id)}
              <div
                class="flex items-start gap-3 rounded-[18px] p-3"
                style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme);"
              >
                <div
                  class="flex flex-shrink-0 items-center justify-center rounded-full"
                  style="width: 36px; height: 36px; box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);"
                >
                  <User size={15} strokeWidth={1.5} color="var(--text-secondary)" />
                </div>
                <div class="min-w-0 flex-1">
                  <div class="flex items-center gap-2">
                    <span style="font-size: 13px; font-weight: 500; color: var(--text-primary);"
                      >{log.operator?.nickname || '系统'}</span
                    >
                    <span
                      class="rounded-full px-2 py-0.5"
                      style="font-size: 10px; background: var(--brand); color: var(--text-on-accent);"
                      >{log.action}</span
                    >
                  </div>
                  <p style="font-size: 12px; color: var(--text-secondary); margin-top: 2px;">
                    {log.action}
                  </p>
                  {#if log.target_type && log.target_id}
                    <p style="font-size: 11px; color: var(--text-secondary);">
                      对象: {log.target_type} ({log.target_id})
                    </p>
                  {/if}
                  {#if log.ip_address}
                    <p style="font-size: 11px; color: var(--text-secondary);">
                      IP: {log.ip_address}
                    </p>
                  {/if}
                </div>
                <div class="flex flex-shrink-0 items-center gap-1">
                  <Clock size={11} strokeWidth={1.5} color="var(--text-secondary)" />
                  <span style="font-size: 11px; color: var(--text-secondary);"
                    >{formatDateTime(log.created_at)}</span
                  >
                </div>
              </div>
            {/each}
          </div>
          {#if pagination.hasMore}
            <button
              class="mt-4 w-full rounded-full py-2.5 outline-none"
              style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme); font-size: 13px; color: var(--text-primary);"
              onclick={() => loadLogs(false)}>加载更多</button
            >
          {/if}
        {/if}
      </div>
    </div>
  </div>
</div>
