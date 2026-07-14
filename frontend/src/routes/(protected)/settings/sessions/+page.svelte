<script lang="ts">
  import { onMount } from 'svelte';
  import { withMinimumLoadingTime } from '$lib/utils/loading';
  import { formatDateTime } from '$lib/utils/format';
  import { authService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { toast } from '$lib/stores/toast.svelte';
  import { t } from '$lib/i18n';
  import PageHeader from '$lib/components/layout/page-header.svelte';
  import type { SessionItem } from '$lib/types/api';
  import RefreshCw from 'lucide-svelte/icons/refresh-cw';
  import EmptyState from '$lib/components/common/empty-state.svelte';
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
      toast.error(t('settings.loadSessionsFailed'));
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
        toast.success(t('settings.revokeSuccess'));
      } else {
        toast.error(response.message || t('settings.revokeFailed'));
      }
    } catch {
      toast.error(t('settings.revokeFailed'));
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
          revokedCount > 0
            ? t('settings.revokeOthersSuccess', { count: revokedCount })
            : t('settings.revokeOthersNoOthers')
        );
        await loadSessions();
      } else {
        toast.error(response.message || t('settings.revokeOthersFailed'));
      }
    } catch {
      toast.error(t('settings.revokeOthersFailed'));
    } finally {
      isRevokingOthers = false;
    }
  }

  onMount(() => {
    loadSessions();
  });
</script>

<svelte:head>
  <title>{t('settings.sessionsPageTitle')}</title>
</svelte:head>

<div class="page-pad">
  <div class="page-shell page-shell--form space-y-5">
    <PageHeader
      title={t('settings.sessionsHeader')}
      description={t('settings.sessionsHeaderDesc')}
      backHref="/settings"
      backLabel={t('settings.backToSettings')}
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
            {t('settings.refresh')}
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
            {t('settings.revokeOthers')}
          </button>
        </div>
      {/snippet}
    </PageHeader>

    {#if isLoading}
      <div class="space-y-3">
        {#each Array(4) as _, i (i)}
          <div
            class="flex items-center gap-4 rounded-[var(--radius-lg)] p-4"
            style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
          >
            <div
              class="h-12 w-12 rounded-[var(--radius-sm)]"
              style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
            ></div>
            <div class="flex-1 space-y-2">
              <div
                class="h-4 w-48 rounded-[var(--radius-sm)]"
                style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
              ></div>
              <div
                class="h-3 w-64 rounded-[var(--radius-sm)]"
                style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
              ></div>
            </div>
          </div>
        {/each}
      </div>
    {:else if sessions.length === 0}
      <EmptyState icon={Smartphone} title={t('settings.noSessions')} fill />
    {:else}
      <div class="space-y-3">
        {#each sessions as session (session.session_id)}
          <div
            class="flex flex-col gap-3 rounded-[var(--radius-lg)] p-4 sm:flex-row sm:items-center sm:justify-between"
            style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
          >
            <div class="flex items-center gap-3 min-w-0">
              <div
                class="flex flex-shrink-0 items-center justify-center rounded-full"
                style="width: 36px; height: 36px; box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme);"
              >
                <Smartphone size={15} strokeWidth={1.5} color="var(--brand)" />
              </div>
              <div>
                <div class="flex items-center gap-2">
                  <p class="text-body-lg" style="font-weight: 500; color: var(--text-primary)">
                    {session.device_name || session.device_type || t('settings.unknownDevice')}
                  </p>
                  {#if session.is_current}
                    <span
                      class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium"
                      style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);"
                      >{t('settings.currentDevice')}</span
                    >
                  {/if}
                </div>
                <div class="text-meta" style="color: var(--text-secondary); margin-top: 2px">
                  {session.ip_location || t('settings.unknownLocation')} · {t('settings.lastActive')} {formatDateTime(
                    session.last_active_at
                  )}
                </div>
              </div>
            </div>
            <div>
              <button
                class="flex items-center rounded-[var(--radius-md)] px-4 py-2 text-sm transition-opacity hover:opacity-80 disabled:opacity-50"
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
                {t('settings.revoke')}
              </button>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>
