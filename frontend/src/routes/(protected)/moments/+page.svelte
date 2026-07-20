<script lang="ts">
  import { onMount } from 'svelte';
  import { momentsService } from '$lib/api/services';
  import { usePagination } from '$lib/hooks/use-pagination.svelte';
  import ListSkeleton from '$lib/components/common/list-skeleton.svelte';
  import EmptyState from '$lib/components/common/empty-state.svelte';
  import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
  import CreateMomentModal from '$lib/components/common/create-moment-modal.svelte';
  import MomentCard from '$lib/components/moments/moment-card.svelte';
  import type { MomentItem } from '$lib/types/api';
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
            style="background: var(--surface-bg); border: 1px solid var(--border-theme); color: var(--text-secondary)"
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
            <MomentCard {moment} />
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
