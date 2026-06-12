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
  import { SHARE_TYPE_BADGE_CLASS, SHARE_TYPE_LABEL } from '$lib/utils/content-labels';
  import type { ShareListItem, ShareType } from '$lib/types/api';
  import RefreshCw from 'lucide-svelte/icons/refresh-cw';
  import Share2 from 'lucide-svelte/icons/share-2';
  import AlertTriangle from 'lucide-svelte/icons/triangle-alert';
  import Ban from 'lucide-svelte/icons/ban';
  import ExternalLink from 'lucide-svelte/icons/external-link';
  import Loader2 from 'lucide-svelte/icons/loader-2';

  const shareTypeOptions: Array<{ value: ShareType | 'all'; label: string }> = [
    { value: 'all', label: '全部' },
    { value: 'user', label: SHARE_TYPE_LABEL.user },
    { value: 'group', label: SHARE_TYPE_LABEL.group },
    { value: 'moment', label: SHARE_TYPE_LABEL.moment },
    { value: 'location', label: SHARE_TYPE_LABEL.location }
  ];

  const pagination = usePagination<ShareListItem>({
    uniqueKey: (item) => item.share_id
  });

  let activeType = $state<ShareType | 'all'>('all');
  let revokingShareId = $state<string | null>(null);

  function getShareTitle(item: ShareListItem): string {
    return item.preview?.title || `${SHARE_TYPE_LABEL[item.share_type]}分享`;
  }

  function getShareSubtitle(item: ShareListItem): string {
    return item.preview?.subtitle || `分享码 ${item.share_code}`;
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
        toast.success('已撤销分享');
      } else {
        toast.error(response.message || '撤销失败');
      }
    } catch {
      toast.error('撤销失败');
    } finally {
      revokingShareId = null;
    }
  }

  onMount(() => {
    loadShares(true);
  });
</script>

<svelte:head>
  <title>我的分享 - HotSpot</title>
</svelte:head>

<div class="page-pad">
  <div class="page-shell space-y-5">
    <PageHeader
      title="我的分享"
      description="查看你创建过的分享，并管理其可访问状态"
      backHref="/settings"
      backLabel="返回设置"
    >
      {#snippet actions()}
        <button
          class="flex items-center rounded-full px-4 py-2 text-sm transition-opacity hover:opacity-80 disabled:opacity-50"
          style="background: var(--surface-bg); box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme); color: var(--text-primary);"
          onclick={() => loadShares(true)}
          disabled={pagination.loading}
        >
          <RefreshCw size={14} class="mr-2 {pagination.loading ? 'animate-spin' : ''}" />
          刷新
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
        title="加载分享失败"
        description={pagination.error}
        actionText="重试"
        onAction={() => loadShares(true)}
      />
    {:else if pagination.items.length === 0}
      <EmptyState
        icon={Share2}
        title="还没有分享记录"
        description={activeType === 'all'
          ? '你创建过的分享会显示在这里。'
          : `当前筛选下暂无${shareTypeOptions.find((option) => option.value === activeType)?.label ?? ''}分享。`}
      />
    {:else}
      <div class="space-y-3">
        {#each pagination.items as item (item.share_id)}
          <div
            class="rounded-[22px] p-4 sm:p-5"
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
                      已撤销
                    </span>
                  {:else}
                    <span
                      class="inline-flex items-center rounded-full px-3 py-0.5 text-xs font-medium"
                      style="background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme); color: var(--brand);"
                    >
                      可访问
                    </span>
                  {/if}
                </div>
                <div>
                  <h2
                    style="font-size: 15px; font-weight: 600; color: var(--text-primary);"
                    class="truncate"
                  >
                    {getShareTitle(item)}
                  </h2>
                  <p style="font-size: 13px; color: var(--text-secondary); margin-top: 4px;">
                    {getShareSubtitle(item)}
                  </p>
                </div>
              </div>
              <div class="flex shrink-0 flex-wrap gap-2">
                <a href={getShareHref(item)} target="_blank" rel="noreferrer">
                  <button
                    class="flex items-center rounded-[18px] px-4 py-2 text-sm transition-opacity hover:opacity-80"
                    style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme); color: var(--text-primary);"
                  >
                    <ExternalLink size={14} class="mr-2" />
                    查看
                  </button>
                </a>
                <button
                  class="flex items-center rounded-[18px] px-4 py-2 text-sm transition-opacity hover:opacity-80 disabled:opacity-50"
                  style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--theme-error-border); color: var(--theme-error);"
                  onclick={() => revokeShare(item)}
                  disabled={item.is_revoked || revokingShareId === item.share_id}
                >
                  {#if revokingShareId === item.share_id}
                    <Loader2 size={14} class="mr-2 animate-spin" />
                  {:else}
                    <Ban size={14} class="mr-2" />
                  {/if}
                  撤销
                </button>
              </div>
            </div>

            <div
              class="mt-4 grid gap-3 rounded-[18px] px-4 py-3 text-sm sm:grid-cols-3"
              style="background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);"
            >
              <div>
                <p style="font-size: 11px; color: var(--text-secondary);">创建时间</p>
                <p
                  style="font-size: 13px; font-weight: 500; color: var(--text-primary); margin-top: 4px;"
                >
                  {formatRelativeTime(item.created_at)}
                </p>
                <p style="font-size: 11px; color: var(--text-secondary); margin-top: 2px;">
                  {formatDateTime(item.created_at)}
                </p>
              </div>
              <div>
                <p style="font-size: 11px; color: var(--text-secondary);">分享码</p>
                <p
                  style="font-size: 13px; font-weight: 500; font-family: monospace; text-transform: uppercase; letter-spacing: 0.12em; color: var(--text-primary); margin-top: 4px;"
                >
                  {item.share_code}
                </p>
              </div>
              <div>
                <p style="font-size: 11px; color: var(--text-secondary);">有效期</p>
                <p
                  style="font-size: 13px; font-weight: 500; color: var(--text-primary); margin-top: 4px;"
                >
                  {item.expires_at ? formatDateTime(item.expires_at) : '长期有效'}
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
