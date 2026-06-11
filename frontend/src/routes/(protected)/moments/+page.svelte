<script lang="ts">
  import { onMount } from 'svelte';
  import { momentsService } from '$lib/api/services';
  import { usePagination } from '$lib/hooks/use-pagination.svelte';
  import ListSkeleton from '$lib/components/common/list-skeleton.svelte';
  import EmptyState from '$lib/components/common/empty-state.svelte';
  import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
  import CreateMomentModal from '$lib/components/common/create-moment-modal.svelte';
  import type { MomentItem } from '$lib/types/api';
  import { formatDate } from '$lib/utils/format';
  import Plus from 'lucide-svelte/icons/plus';
  import Image from 'lucide-svelte/icons/image';
  import RefreshCw from 'lucide-svelte/icons/refresh-cw';
  import AlertTriangle from 'lucide-svelte/icons/triangle-alert';

  const pagination = usePagination<MomentItem>({
    uniqueKey: 'moment_id'
  });
  let showCreateMoment = $state(false);

  async function loadMoments(refresh = false) {
    await pagination.load(() => momentsService.getMyMoments(), refresh);
  }

  onMount(() => {
    loadMoments(true);
  });
</script>

<svelte:head>
  <title>我的瞬间 - HotSpot</title>
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
          <h1 style="font-size: 20px; font-weight: 600; color: var(--text-primary);">我的瞬间</h1>
          <p style="font-size: 12px; color: var(--text-secondary); margin-top: 2px;">
            记录和管理你发布的动态内容
          </p>
        </div>
        <div class="flex gap-2">
          <button
            class="flex items-center gap-1.5 rounded-full px-3 py-1.5 outline-none"
            style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme); font-size: 12px; color: var(--text-secondary);"
            onclick={() => loadMoments(true)}
            disabled={pagination.loading}
          >
            <RefreshCw size={13} class={pagination.loading ? 'animate-spin' : ''} />
            刷新
          </button>
          <button
            class="flex items-center gap-1.5 rounded-full px-3 py-1.5 outline-none"
            style="background: var(--brand); color: var(--text-on-accent); font-size: 12px; box-shadow: 0 2px 8px var(--glow);"
            onclick={() => (showCreateMoment = true)}
          >
            <Plus size={13} />
            发布
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
          title="加载失败"
          description={pagination.error}
          actionText="重试"
          onAction={() => loadMoments(true)}
        />
      {:else if pagination.items.length === 0}
        <EmptyState
          icon={Image}
          title="还没有瞬间"
          description="发布你的第一个瞬间吧"
          actionText="发布瞬间"
          onAction={() => (showCreateMoment = true)}
        />
      {:else}
        <div class="columns-1 gap-4 sm:columns-2 xl:columns-3">
          {#each pagination.items as moment (moment.moment_id)}
            <div class="mb-4 break-inside-avoid">
              <div
                class="rounded-[22px] overflow-hidden"
                style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
              >
                <a
                  href="/moments/{moment.moment_id}"
                  class="block p-4"
                  style="text-decoration: none;"
                >
                  <p
                    class="line-clamp-3 whitespace-pre-wrap"
                    style="font-size: 14px; color: var(--text-primary);"
                  >
                    {moment.description}
                  </p>
                  {#if moment.images && moment.images.length > 0}
                    <div class="mt-3 flex gap-2 overflow-x-auto">
                      {#each moment.images.slice(0, 4) as image, i (image.url)}
                        <img
                          src={image.url}
                          alt="图片 {i + 1}"
                          class="h-20 w-20 rounded-[12px] object-cover"
                        />
                      {/each}
                      {#if moment.images.length > 4}
                        <div
                          class="flex h-20 w-20 items-center justify-center rounded-[12px]"
                          style="box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme); font-size: 13px; color: var(--text-secondary);"
                        >
                          +{moment.images.length - 4}
                        </div>
                      {/if}
                    </div>
                  {/if}
                  <div
                    class="mt-3 flex items-center gap-4"
                    style="font-size: 12px; color: var(--text-secondary);"
                  >
                    <span>{moment.likes_count} 点赞</span>
                    <span>{moment.comments_count} 评论</span>
                    <span>{moment.shares_count} 分享</span>
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
