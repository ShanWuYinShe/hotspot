<script lang="ts">
  import { onMount } from 'svelte';
  import { momentsService } from '$lib/api/services';
  import { usePagination } from '$lib/hooks/use-pagination.svelte';
  import ListSkeleton from '$lib/components/common/list-skeleton.svelte';
  import EmptyState from '$lib/components/common/empty-state.svelte';
  import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
  import CreateMomentModal from '$lib/components/common/create-moment-modal.svelte';
  import SmartImage from '$lib/components/common/smart-image.svelte';
  import type { MomentItem } from '$lib/types/api';
  import { formatDate } from '$lib/utils/format';
  import { t } from '$lib/i18n';
  import Plus from 'lucide-svelte/icons/plus';
  import Image from 'lucide-svelte/icons/image';
  import RefreshCw from 'lucide-svelte/icons/refresh-cw';
  import AlertTriangle from 'lucide-svelte/icons/triangle-alert';

  const pagination = usePagination<MomentItem>({
    uniqueKey: 'moment_id'
  });
  let showCreateMoment = $state(false);

  async function loadMoments(refresh = false) {
    await pagination.load(
      (cursor, opts) => momentsService.getMyMoments({ pagination: { limit: 12, cursor } }, opts),
      refresh
    );
  }

  onMount(() => {
    loadMoments(true);
  });
</script>

<svelte:head>
  <title>{t('momentsMine.title')} - HotSpot</title>
</svelte:head>

<div class="flex flex-col">
  <!-- Sticky header -->
  <div
    class="sticky top-0 z-10 page-pad-x pt-3 pb-3"
    style="background-color: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
  >
    <div class="page-shell">
      <div class="flex items-center justify-between">
        <div>
          <h1 class="page-title" style="color: var(--text-primary);">{t('momentsMine.title')}</h1>
        </div>
        <div class="flex gap-2">
          <button
            class="text-meta flex items-center gap-1.5 rounded-full px-3 py-1.5 outline-none"
            style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme); color: var(--text-secondary)"
            onclick={() => loadMoments(true)}
            disabled={pagination.loading}
          >
            <RefreshCw size={13} class={pagination.loading ? 'animate-spin' : ''} />
            {t('momentsMine.refresh')}
          </button>
          <button
            class="text-meta flex items-center gap-1.5 rounded-full px-3 py-1.5 outline-none"
            style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn)"
            onclick={() => (showCreateMoment = true)}
          >
            <Plus size={13} />
            {t('momentsMine.publish')}
          </button>
        </div>
      </div>
    </div>
  </div>

  <div class="flex flex-col page-pad">
    <div class="page-shell flex flex-col gap-5">
      {#if pagination.loading}
        <ListSkeleton type="moment" count={3} layout="grid" />
      {:else if pagination.error}
        <EmptyState
          icon={AlertTriangle}
          title={t('momentsMine.loadFailed')}
          description={pagination.error}
          actionText={t('momentsMine.retry')}
          onAction={() => loadMoments(true)}
          fill
        />
      {:else if pagination.items.length === 0}
        <EmptyState
          icon={Image}
          title={t('momentsMine.emptyTitle')}
          description={t('momentsMine.emptyDesc')}
          actionText={t('momentsMine.publishMoment')}
          onAction={() => (showCreateMoment = true)}
          fill
        />
      {:else}
        <div class="density-grid">
          {#each pagination.items as moment (moment.moment_id)}
            <div style="content-visibility: auto; contain-intrinsic-size: 300px;">
              <div
                class="rounded-[var(--radius-lg)] overflow-hidden"
                style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
              >
                <a
                  href="/moments/{moment.moment_id}"
                  class="block p-4"
                  style="text-decoration: none;"
                >
                  <p
                    class="text-body-lg line-clamp-3 whitespace-pre-wrap"
                    style="color: var(--text-primary)"
                  >
                    {moment.description}
                  </p>
                  {#if moment.images && moment.images.length > 0}
                    <div class="mt-3 flex gap-2 overflow-x-auto">
                      {#each moment.images.slice(0, 4) as image, i (image.url)}
                        <SmartImage
                          src={image.url}
                          thumbnail={image.thumbnail_url}
                          alt={t('momentsMine.imageAlt', { index: i + 1 })}
                          width={image.width || 80}
                          height={image.height || 80}
                          sizes="80px"
                          class="h-20 w-20 rounded-[var(--radius-sm)] object-cover"
                        />
                      {/each}
                      {#if moment.images.length > 4}
                        <div
                          class="text-body flex h-20 w-20 items-center justify-center rounded-[var(--radius-sm)]"
                          style="border: 1px solid var(--border-theme); color: var(--text-secondary)"
                        >
                          +{moment.images.length - 4}
                        </div>
                      {/if}
                    </div>
                  {/if}
                  <div
                    class="text-meta mt-3 flex items-center gap-4"
                    style="color: var(--text-secondary)"
                  >
                    <span>{moment.likes_count} {t('momentsMine.likesLabel')}</span>
                    <span>{moment.comments_count} {t('momentsMine.commentsLabel')}</span>
                    <span>{moment.shares_count} {t('momentsMine.sharesLabel')}</span>
                    <span>{formatDate(moment.created_at)}</span>
                  </div>
                </a>
              </div>
            </div>
          {/each}
        </div>

        <LoadMoreButton
          loading={pagination.loadingMore}
          hasMore={pagination.hasMore}
          onLoadMore={() => loadMoments(false)}
        />
      {/if}
    </div>
  </div>
</div>

<CreateMomentModal bind:open={showCreateMoment} onCreated={() => loadMoments(true)} />
