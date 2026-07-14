<script lang="ts">
  import { onMount } from 'svelte';
  import { shareService } from '$lib/api/services';
  import { usePagination } from '$lib/hooks/use-pagination.svelte';
  import { formatDateTime, formatRelativeTime } from '$lib/utils/format';
  import PageHeader from '$lib/components/layout/page-header.svelte';
  import EmptyState from '$lib/components/common/empty-state.svelte';
  import ListSkeleton from '$lib/components/common/list-skeleton.svelte';
  import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
  import { toast } from '$lib/stores/toast.svelte';
  import { isSuccess } from '$lib/api/client';
  import { t } from '$lib/i18n';
  import { SHARE_TYPE_BADGE_CLASS, SHARE_TYPE_LABEL } from '$lib/utils/content-labels';
  import type { ShareListItem, ShareType } from '$lib/types/api';
  import RefreshCw from 'lucide-svelte/icons/refresh-cw';
  import Share2 from 'lucide-svelte/icons/share-2';
  import AlertTriangle from 'lucide-svelte/icons/triangle-alert';
  import Ban from 'lucide-svelte/icons/ban';
  import ExternalLink from 'lucide-svelte/icons/external-link';
  import Loader2 from 'lucide-svelte/icons/loader-2';

  const shareTypeOptions = $derived<Array<{ value: ShareType | 'all'; label: string }>>([
    { value: 'all', label: t('settings.filterAll') },
    { value: 'user', label: SHARE_TYPE_LABEL.user },
    { value: 'group', label: SHARE_TYPE_LABEL.group },
    { value: 'moment', label: SHARE_TYPE_LABEL.moment },
    { value: 'location', label: SHARE_TYPE_LABEL.location }
  ]);

  const pagination = usePagination<ShareListItem>({
    uniqueKey: (item) => item.share_id
  });

  let activeType = $state<ShareType | 'all'>('all');
  let revokingShareId = $state<string | null>(null);

  function getShareTitle(item: ShareListItem): string {
    return item.preview?.title || t('settings.shareTitleFallback', { type: SHARE_TYPE_LABEL[item.share_type] });
  }

  function getShareSubtitle(item: ShareListItem): string {
    return item.preview?.subtitle || t('settings.shareSubtitleFallback', { code: item.share_code });
  }

  function getShareHref(item: ShareListItem): string {
    return `/share/${item.share_code}`;
  }

  async function loadShares(refresh = false) {
    await pagination.load(
      (cursor) =>
        shareService.listMyShares({
          share_type: activeType === 'all' ? undefined : activeType,
          pagination: { limit: 20, cursor }
        }),
      refresh
    );
  }

  async function revokeShare(item: ShareListItem) {
    if (item.is_revoked || revokingShareId) return;
    revokingShareId = item.share_id;
    try {
      const response = await shareService.revoke({ share_id: item.share_id });
      if (isSuccess(response)) {
        pagination.updateItem((current) =>
          current.share_id === item.share_id ? { ...current, is_revoked: true } : current
        );
        toast.success(t('settings.revokeShareSuccess'));
      } else {
        toast.error(response.message || t('settings.revokeShareFailed'));
      }
    } catch {
      toast.error(t('settings.revokeShareFailed'));
    } finally {
      revokingShareId = null;
    }
  }

  onMount(() => {
    loadShares(true);
  });
</script>

<svelte:head>
  <title>{t('settings.sharesPageTitle')}</title>
</svelte:head>

<div class="page-pad">
  <div class="page-shell space-y-5">
    <PageHeader
      title={t('settings.sharesHeader')}
      description={t('settings.sharesHeaderDesc')}
      backHref="/settings"
      backLabel={t('settings.backToSettings')}
    >
      {#snippet actions()}
        <button
          class="flex items-center rounded-full px-4 py-2 text-sm transition-opacity hover:opacity-80 disabled:opacity-50"
          style="background: var(--surface-bg); box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme); color: var(--text-primary);"
          onclick={() => loadShares(true)}
          disabled={pagination.loading}
        >
          <RefreshCw size={14} class="mr-2 {pagination.loading ? 'animate-spin' : ''}" />
          {t('settings.refresh')}
        </button>
      {/snippet}
    </PageHeader>

    <!-- Type filter pills -->
    <div class="flex flex-wrap gap-2">
      {#each shareTypeOptions as option (option.value)}
        <button
          class="filter-chip disabled:opacity-50"
          class:filter-chip--active={activeType === option.value}
          disabled={pagination.loading}
          onclick={async () => {
            if (pagination.loading || activeType === option.value) return;
            activeType = option.value;
            await loadShares(true);
          }}
        >
          {option.label}
        </button>
      {/each}
    </div>

    {#if pagination.loading}
      <ListSkeleton type="notification" count={4} layout="list" />
    {:else if pagination.error}
      <EmptyState
        icon={AlertTriangle}
        title={t('settings.loadSharesFailed')}
        description={pagination.error}
        actionText={t('settings.retry')}
        onAction={() => loadShares(true)}
      />
    {:else if pagination.items.length === 0}
      <EmptyState
        icon={Share2}
        title={t('settings.sharesHeader')}
        description={activeType === 'all'
          ? t('settings.noSharesAll')
          : t('settings.noSharesFiltered', { type: shareTypeOptions.find((option) => option.value === activeType)?.label ?? '' })}
      />
    {:else}
      <div class="space-y-3">
        {#each pagination.items as item (item.share_id)}
          <div
            class="rounded-[var(--radius-lg)] p-4 sm:p-5"
            style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
          >
            <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div class="min-w-0 flex-1 space-y-2">
                <div class="flex flex-wrap items-center gap-2">
                  <span
                    class={`inline-flex items-center rounded-full border px-3 py-0.5 text-xs font-medium ${SHARE_TYPE_BADGE_CLASS[item.share_type]}`}
                  >
                    {SHARE_TYPE_LABEL[item.share_type]}
                  </span>
                  {#if item.is_revoked}
                    <span
                      class="inline-flex items-center rounded-full px-3 py-0.5 text-xs font-medium"
                      style="border: 1px solid var(--theme-error-border); color: var(--theme-error);"
                    >
                      {t('settings.shareRevoked')}
                    </span>
                  {:else}
                    <span
                      class="inline-flex items-center rounded-full px-3 py-0.5 text-xs font-medium"
                      style="background: var(--surface-bg); border: 1px solid var(--border-theme); color: var(--brand);"
                    >
                      {t('settings.shareAccessible')}
                    </span>
                  {/if}
                </div>
                <div>
                  <h2
                    style="font-weight: 600; color: var(--text-primary)"
                    class="text-body-lg truncate"
                  >
                    {getShareTitle(item)}
                  </h2>
                  <p class="text-body" style="color: var(--text-secondary); margin-top: 4px">
                    {getShareSubtitle(item)}
                  </p>
                </div>
              </div>
              <div class="flex shrink-0 flex-wrap gap-2">
                <a href={getShareHref(item)} target="_blank" rel="noreferrer">
                  <button
                    class="flex items-center rounded-[var(--radius-md)] px-4 py-2 text-sm transition-opacity hover:opacity-80"
                    style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme); color: var(--text-primary);"
                  >
                    <ExternalLink size={14} class="mr-2" />
                    {t('settings.viewShare')}
                  </button>
                </a>
                <button
                  class="flex items-center rounded-[var(--radius-md)] px-4 py-2 text-sm transition-opacity hover:opacity-80 disabled:opacity-50"
                  style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--theme-error-border); color: var(--theme-error);"
                  onclick={() => revokeShare(item)}
                  disabled={item.is_revoked || revokingShareId === item.share_id}
                >
                  {#if revokingShareId === item.share_id}
                    <Loader2 size={14} class="mr-2 animate-spin" />
                  {:else}
                    <Ban size={14} class="mr-2" />
                  {/if}
                  {t('settings.revokeShare')}
                </button>
              </div>
            </div>

            <div
              class="mt-4 grid gap-3 rounded-[var(--radius-md)] px-4 py-3 text-sm sm:grid-cols-3"
              style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
            >
              <div>
                <p class="text-caption" style="color: var(--text-secondary)">{t('settings.createdAt')}</p>
                <p
                  class="text-body"
                  style="font-weight: 500; color: var(--text-primary); margin-top: 4px"
                >
                  {formatRelativeTime(item.created_at)}
                </p>
                <p class="text-caption" style="color: var(--text-secondary); margin-top: 2px">
                  {formatDateTime(item.created_at)}
                </p>
              </div>
              <div>
                <p class="text-caption" style="color: var(--text-secondary)">{t('settings.shareCode')}</p>
                <p
                  class="text-body"
                  style="font-weight: 500; font-family: monospace; text-transform: uppercase; letter-spacing: 0.12em; color: var(--text-primary); margin-top: 4px"
                >
                  {item.share_code}
                </p>
              </div>
              <div>
                <p class="text-caption" style="color: var(--text-secondary)">{t('settings.validity')}</p>
                <p
                  class="text-body"
                  style="font-weight: 500; color: var(--text-primary); margin-top: 4px"
                >
                  {item.expires_at ? formatDateTime(item.expires_at) : t('settings.validityPermanent')}
                </p>
              </div>
            </div>
          </div>
        {/each}
      </div>

      <LoadMoreButton
        loading={pagination.loadingMore}
        hasMore={pagination.hasMore}
        onLoadMore={() => loadShares(false)}
      />
    {/if}
  </div>
</div>
