<script lang="ts">
  import { onMount } from 'svelte';
  import { reportsService } from '$lib/api/services';
  import { usePagination } from '$lib/hooks/use-pagination.svelte';
  import { formatDateTime, formatRelativeTime } from '$lib/utils/format';
  import {
    REPORT_REASON_LABEL,
    REPORT_STATUS_BADGE_CLASS,
    REPORT_STATUS_LABEL,
    REPORT_TARGET_TYPE_LABEL
  } from '$lib/utils/content-labels';
  import PageHeader from '$lib/components/layout/page-header.svelte';
  import EmptyState from '$lib/components/common/empty-state.svelte';
  import ListSkeleton from '$lib/components/common/list-skeleton.svelte';
  import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
  import { t } from '$lib/i18n';
  import type { ReportItem, ReportStatus, ReportTargetType } from '$lib/types/api';
  import RefreshCw from 'lucide-svelte/icons/refresh-cw';
  import AlertTriangle from 'lucide-svelte/icons/triangle-alert';
  import ShieldAlert from 'lucide-svelte/icons/shield-alert';

  const statusOptions = $derived<Array<{ value: ReportStatus | 'all'; label: string }>>([
    { value: 'all', label: t('settings.statusAll') },
    { value: 'pending', label: t('settings.statusPending') },
    { value: 'reviewing', label: t('settings.statusReviewing') },
    { value: 'resolved', label: t('settings.statusResolved') },
    { value: 'dismissed', label: t('settings.statusDismissed') }
  ]);

  const targetTypeOptions = $derived<Array<{ value: ReportTargetType | 'all'; label: string }>>([
    { value: 'all', label: t('settings.typeAll') },
    { value: 'user', label: REPORT_TARGET_TYPE_LABEL.user },
    { value: 'moment', label: REPORT_TARGET_TYPE_LABEL.moment },
    { value: 'comment', label: REPORT_TARGET_TYPE_LABEL.comment },
    { value: 'message', label: REPORT_TARGET_TYPE_LABEL.message },
    { value: 'group', label: REPORT_TARGET_TYPE_LABEL.group }
  ]);

  const pagination = usePagination<ReportItem>({
    uniqueKey: (item) => item.report_id
  });

  let activeStatus = $state<ReportStatus | 'all'>('all');
  let activeTargetType = $state<ReportTargetType | 'all'>('all');

  async function loadReports(refresh = false) {
    await pagination.load(
      (cursor) =>
        reportsService.getMyReports({
          status: activeStatus === 'all' ? undefined : activeStatus,
          target_type: activeTargetType === 'all' ? undefined : activeTargetType,
          pagination: { limit: 20, cursor }
        }),
      refresh
    );
  }

  onMount(() => {
    loadReports(true);
  });
</script>

<svelte:head>
  <title>{t('settings.reportsPageTitle')}</title>
</svelte:head>

<div class="page-pad">
  <div class="page-shell space-y-5">
    <PageHeader
      title={t('settings.reportsHeader')}
      description={t('settings.reportsHeaderDesc')}
      backHref="/settings"
      backLabel={t('settings.backToSettings')}
    >
      {#snippet actions()}
        <button
          class="flex items-center rounded-full px-4 py-2 text-sm transition-opacity hover:opacity-80 disabled:opacity-50"
          style="background: var(--surface-bg); box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme); color: var(--text-primary);"
          onclick={() => loadReports(true)}
          disabled={pagination.loading}
        >
          <RefreshCw size={14} class="mr-2 {pagination.loading ? 'animate-spin' : ''}" />
          {t('settings.refresh')}
        </button>
      {/snippet}
    </PageHeader>

    <!-- Filter Card -->
    <div
      class="rounded-[var(--radius-lg)] p-4"
      style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
    >
      <div class="mb-3">
        <p class="text-body" style="font-weight: 500; color: var(--text-primary)">{t('settings.filterConditions')}</p>
        <p class="text-meta" style="color: var(--text-secondary); margin-top: 2px">
          {t('settings.filterConditionsDesc')}
        </p>
      </div>
      <div class="space-y-3">
        <div class="flex flex-wrap gap-2">
          {#each statusOptions as option (option.value)}
            <button
              class="filter-chip disabled:opacity-50"
              class:filter-chip--active={activeStatus === option.value}
              disabled={pagination.loading}
              onclick={async () => {
                if (pagination.loading || activeStatus === option.value) return;
                activeStatus = option.value;
                await loadReports(true);
              }}
            >
              {option.label}
            </button>
          {/each}
        </div>
        <div class="flex flex-wrap gap-2">
          {#each targetTypeOptions as option (option.value)}
            <button
              class="filter-chip disabled:opacity-50"
              class:filter-chip--active={activeTargetType === option.value}
              disabled={pagination.loading}
              onclick={async () => {
                if (pagination.loading || activeTargetType === option.value) return;
                activeTargetType = option.value;
                await loadReports(true);
              }}
            >
              {option.label}
            </button>
          {/each}
        </div>
      </div>
    </div>

    {#if pagination.loading}
      <ListSkeleton type="notification" count={4} layout="list" />
    {:else if pagination.error}
      <EmptyState
        icon={AlertTriangle}
        title={t('settings.loadReportsFailed')}
        description={pagination.error}
        actionText={t('settings.retry')}
        onAction={() => loadReports(true)}
      />
    {:else if pagination.items.length === 0}
      <EmptyState
        icon={ShieldAlert}
        title={t('settings.noReports')}
        description={t('settings.noReportsDesc')}
      />
    {:else}
      <div class="space-y-3">
        {#each pagination.items as item (item.report_id)}
          <div
            class="rounded-[var(--radius-lg)] p-4 sm:p-5"
            style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
          >
            <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div class="min-w-0 flex-1 space-y-2">
                <div class="flex flex-wrap items-center gap-2">
                  <span
                    class="inline-flex items-center rounded-full px-3 py-0.5 text-xs font-medium"
                    style="background: var(--surface-bg); border: 1px solid var(--border-theme); color: var(--text-secondary);"
                  >
                    {REPORT_TARGET_TYPE_LABEL[item.target_type]}
                  </span>
                  <span
                    class={`inline-flex items-center rounded-full border px-3 py-0.5 text-xs font-medium ${REPORT_STATUS_BADGE_CLASS[item.status]}`}
                  >
                    {REPORT_STATUS_LABEL[item.status]}
                  </span>
                </div>
                <div>
                  <h2 class="text-body-lg" style="font-weight: 600; color: var(--text-primary)">
                    {REPORT_REASON_LABEL[item.reason]}
                  </h2>
                  <p class="text-body" style="color: var(--text-secondary); margin-top: 4px">
                    {t('settings.targetId', { id: item.target_id })}
                  </p>
                </div>
              </div>
              <div class="text-body" style="color: var(--text-secondary)">
                <p>{formatRelativeTime(item.created_at)}</p>
              </div>
            </div>

            <div
              class="mt-4 grid gap-3 rounded-[var(--radius-md)] px-4 py-3 text-sm sm:grid-cols-2 lg:grid-cols-3"
              style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
            >
              <div>
                <p class="text-caption" style="color: var(--text-secondary)">{t('settings.submittedAt')}</p>
                <p
                  class="text-body"
                  style="font-weight: 500; color: var(--text-primary); margin-top: 4px"
                >
                  {formatDateTime(item.created_at)}
                </p>
              </div>
              <div>
                <p class="text-caption" style="color: var(--text-secondary)">{t('settings.handledAt')}</p>
                <p
                  class="text-body"
                  style="font-weight: 500; color: var(--text-primary); margin-top: 4px"
                >
                  {item.handled_at ? formatDateTime(item.handled_at) : t('settings.notHandled')}
                </p>
              </div>
              <div>
                <p class="text-caption" style="color: var(--text-secondary)">{t('settings.reportId')}</p>
                <p
                  class="text-meta"
                  style="font-weight: 500; font-family: monospace; color: var(--text-primary); margin-top: 4px"
                >
                  {item.report_id}
                </p>
              </div>
            </div>

            {#if item.description}
              <div
                class="mt-3 rounded-[var(--radius-md)] px-4 py-3"
                style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
              >
                <p class="text-caption" style="color: var(--text-secondary)">{t('settings.supplementaryNote')}</p>
                <p
                  class="text-body"
                  style="line-height: 1.6; color: var(--text-primary); margin-top: 4px"
                >
                  {item.description}
                </p>
              </div>
            {/if}
          </div>
        {/each}
      </div>

      <LoadMoreButton
        loading={pagination.loadingMore}
        hasMore={pagination.hasMore}
        onLoadMore={() => loadReports(false)}
      />
    {/if}
  </div>
</div>
