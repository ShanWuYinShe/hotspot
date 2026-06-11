<script lang="ts">
  import { onMount } from 'svelte';
  import { withMinimumLoadingTime } from '$lib/utils/loading';
  import { formatDateTime } from '$lib/utils/format';
  import { authService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { toast } from '$lib/stores/toast.svelte';
  import PageHeader from '$lib/components/layout/page-header.svelte';
  import type { SessionItem } from '$lib/types/api';
  import RefreshCw from 'lucide-svelte/icons/refresh-cw';
  import Shield from 'lucide-svelte/icons/shield';
  import Loader2 from 'lucide-svelte/icons/loader-2';
  import Smartphone from 'lucide-svelte/icons/smartphone';

  let sessions = $state.raw<SessionItem[]>([]);
  let isLoading = $state(true);
  let isRevoking = $state<string | null>(null);
  let isRevokingOthers = $state(false);

  async function loadSessions() {
    isLoading = true;
    try {
      const response = await withMinimumLoadingTime(() => authService.getSessions());
      if (isSuccess(response) && response.data) {
        sessions = response.data.items;
      }
    } catch {
      toast.error('加载会话失败');
    } finally {
      isLoading = false;
    }
  }

  async function revoke(sessionId: string) {
    isRevoking = sessionId;
    try {
      const response = await authService.revokeSession({ session_id: sessionId });
      if (isSuccess(response)) {
        sessions = sessions.filter((item) => item.session_id !== sessionId);
        toast.success('已撤销会话');
      } else {
        toast.error(response.message || '撤销失败');
      }
    } catch {
      toast.error('撤销失败');
    } finally {
      isRevoking = null;
    }
  }

  async function revokeOthers() {
    if (isRevokingOthers) return;
    isRevokingOthers = true;
    try {
      const response = await authService.revokeAllSessions();
      if (isSuccess(response)) {
        const revokedCount = response.data?.revoked_count ?? 0;
        toast.success(
          revokedCount > 0 ? `已撤销 ${revokedCount} 个其他设备会话` : '没有其他设备会话需要撤销'
        );
        await loadSessions();
      } else {
        toast.error(response.message || '撤销其他设备失败');
      }
    } catch {
      toast.error('撤销其他设备失败');
    } finally {
      isRevokingOthers = false;
    }
  }

  onMount(() => {
    loadSessions();
  });
</script>

<svelte:head>
  <title>会话管理 - HotSpot</title>
</svelte:head>

<div class="page-pad">
  <div class="page-shell page-shell--form space-y-5">
    <PageHeader
      title="会话管理"
      description="查看并管理已登录设备，仅保留当前设备在线"
      backHref="/settings"
      backLabel="返回设置"
    >
      {#snippet actions()}
        <div class="flex flex-wrap items-center gap-2">
          <button
            class="flex items-center rounded-full px-4 py-2 text-sm transition-opacity hover:opacity-80 disabled:opacity-50"
            style="background: var(--surface-bg); box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme); color: var(--text-primary);"
            onclick={loadSessions}
            disabled={isLoading}
          >
            <RefreshCw size={14} class="mr-2 {isLoading ? 'animate-spin' : ''}" />
            刷新
          </button>
          <button
            class="flex items-center rounded-full px-4 py-2 text-sm transition-opacity hover:opacity-80 disabled:opacity-50"
            style="background: var(--surface-bg); box-shadow: var(--shadow-sm); border: 1px solid var(--theme-error-border); color: var(--theme-error);"
            onclick={revokeOthers}
            disabled={isLoading || isRevokingOthers}
          >
            {#if isRevokingOthers}
              <Loader2 size={14} class="mr-2 animate-spin" />
            {/if}
            撤销其他设备
          </button>
        </div>
      {/snippet}
    </PageHeader>

    {#if isLoading}
      <div class="space-y-3">
        {#each Array(4) as _, i (i)}
          <div
            class="flex items-center gap-4 rounded-[22px] p-4"
            style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
          >
            <div
              class="h-12 w-12 rounded-[12px]"
              style="background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);"
            ></div>
            <div class="flex-1 space-y-2">
              <div
                class="h-4 w-48 rounded-[12px]"
                style="background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);"
              ></div>
              <div
                class="h-3 w-64 rounded-[12px]"
                style="background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);"
              ></div>
            </div>
          </div>
        {/each}
      </div>
    {:else if sessions.length === 0}
      <div
        class="flex flex-col items-center justify-center rounded-[22px] py-12"
        style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
      >
        <div
          class="flex items-center justify-center rounded-full mb-4"
          style="width: 48px; height: 48px; box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);"
        >
          <Shield size={20} color="var(--text-secondary)" />
        </div>
        <p style="font-size: 14px; color: var(--text-secondary);">暂无会话记录</p>
      </div>
    {:else}
      <div class="space-y-3">
        {#each sessions as session (session.session_id)}
          <div
            class="flex flex-col gap-3 rounded-[22px] p-4 sm:flex-row sm:items-center sm:justify-between"
            style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
          >
            <div class="flex items-center gap-3 min-w-0">
              <div
                class="flex flex-shrink-0 items-center justify-center rounded-full"
                style="width: 36px; height: 36px; box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);"
              >
                <Smartphone size={15} strokeWidth={1.5} color="var(--brand)" />
              </div>
              <div>
                <div class="flex items-center gap-2">
                  <p style="font-size: 14px; font-weight: 500; color: var(--text-primary);">
                    {session.device_name || session.device_type || '未知设备'}
                  </p>
                  {#if session.is_current}
                    <span
                      class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium"
                      style="background: var(--brand); color: var(--text-on-accent); box-shadow: 0 2px 6px var(--glow);"
                      >当前设备</span
                    >
                  {/if}
                </div>
                <div style="font-size: 12px; color: var(--text-secondary); margin-top: 2px;">
                  {session.ip_location || '未知位置'} · 最后活跃 {formatDateTime(
                    session.last_active_at
                  )}
                </div>
              </div>
            </div>
            <div>
              <button
                class="flex items-center rounded-[18px] px-4 py-2 text-sm transition-opacity hover:opacity-80 disabled:opacity-50"
                style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid {session.is_current
                  ? 'var(--border-theme)'
                  : 'var(--theme-error-border)'}; color: {session.is_current
                  ? 'var(--text-secondary)'
                  : 'var(--theme-error)'};"
                disabled={session.is_current || isRevoking === session.session_id}
                onclick={() => revoke(session.session_id)}
              >
                {#if isRevoking === session.session_id}
                  <Loader2 size={14} class="mr-2 animate-spin" />
                {/if}
                撤销
              </button>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>
