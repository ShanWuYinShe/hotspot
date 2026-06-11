<script lang="ts">
  import { onMount } from 'svelte';
  import { withMinimumLoadingTime } from '$lib/utils/loading';
  import { usersService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { toast } from '$lib/stores/toast.svelte';
  import EmptyState from '$lib/components/common/empty-state.svelte';
  import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
  import type { FriendItem } from '$lib/types/api';
  import RefreshCw from 'lucide-svelte/icons/refresh-cw';
  import Users from 'lucide-svelte/icons/users';

  let friends = $state.raw<FriendItem[]>([]);
  let isLoading = $state(true);
  let isLoadingMore = $state(false);
  let cursor = $state<string | undefined>(undefined);
  let hasMore = $state(true);

  async function loadFriends(refresh = false) {
    if (refresh) {
      isLoading = true;
      cursor = undefined;
      hasMore = true;
    } else {
      isLoadingMore = true;
    }

    try {
      const response = refresh
        ? await withMinimumLoadingTime(() =>
            usersService.getFriends({
              pagination: { limit: 24, cursor: undefined }
            })
          )
        : await usersService.getFriends({
            pagination: { limit: 24, cursor }
          });
      if (isSuccess(response) && response.data) {
        friends = refresh ? response.data.items : [...friends, ...response.data.items];
        hasMore = !!response.data.next_cursor;
        cursor = response.data.next_cursor;
      }
    } catch {
      toast.error('加载好友失败');
    } finally {
      isLoading = false;
      isLoadingMore = false;
    }
  }

  onMount(() => {
    loadFriends(true);
  });
</script>

<svelte:head>
  <title>好友 - HotSpot</title>
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
          <h1 style="font-size: 20px; font-weight: 600; color: var(--text-primary);">好友</h1>
          <p style="font-size: 12px; color: var(--text-secondary); margin-top: 2px;">
            你们的互相关注列表
          </p>
        </div>
        <button
          class="flex items-center gap-1.5 rounded-full px-3 py-1.5 outline-none"
          style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme); font-size: 12px; color: var(--text-secondary);"
          onclick={() => loadFriends(true)}
          disabled={isLoading}
        >
          <RefreshCw size={13} class={isLoading ? 'animate-spin' : ''} />
          刷新
        </button>
      </div>
    </div>
  </div>

  <div class="flex flex-col page-pad">
    <div class="page-shell flex flex-col gap-4">
      {#if isLoading}
        <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {#each Array(6) as _, i (i)}
            <div
              class="flex items-center gap-3 rounded-[18px] p-4"
              style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
            >
              <div
                class="rounded-full"
                style="width: 48px; height: 48px; background: var(--surface-bg); box-shadow: var(--shadow-inset);"
              ></div>
              <div class="flex-1 space-y-2">
                <div
                  style="width: 96px; height: 16px; border-radius: 8px; background: var(--surface-bg); box-shadow: var(--shadow-inset);"
                ></div>
                <div
                  style="width: 128px; height: 12px; border-radius: 6px; background: var(--surface-bg); box-shadow: var(--shadow-inset);"
                ></div>
              </div>
            </div>
          {/each}
        </div>
      {:else if friends.length === 0}
        <EmptyState icon={Users} title="暂无好友" />
      {:else}
        <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {#each friends as friend (friend.user.user_id)}
            <a href="/users/{friend.user.user_id}" class="block" style="text-decoration: none;">
              <div
                class="flex items-center gap-3 rounded-[18px] p-4"
                style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
              >
                <div
                  class="overflow-hidden rounded-full shrink-0"
                  style="width: 48px; height: 48px; border: 1.5px solid var(--border-theme);"
                >
                  {#if friend.user.avatar_url}
                    <img
                      src={friend.user.avatar_url}
                      alt={friend.user.nickname}
                      class="h-full w-full object-cover"
                    />
                  {:else}
                    <div
                      class="flex h-full w-full items-center justify-center"
                      style="background: var(--surface-bg); font-size: 14px; color: var(--text-secondary);"
                    >
                      {friend.user.nickname.slice(0, 2)}
                    </div>
                  {/if}
                </div>
                <div class="min-w-0">
                  <p
                    class="truncate"
                    style="font-size: 14px; font-weight: 500; color: var(--text-primary);"
                  >
                    {friend.user.nickname}
                  </p>
                  <p style="font-size: 12px; color: var(--text-secondary);">
                    {friend.remark ? `备注：${friend.remark}` : '互相关注'}
                  </p>
                </div>
              </div>
            </a>
          {/each}
        </div>
        <LoadMoreButton loading={isLoadingMore} {hasMore} onLoadMore={() => loadFriends(false)} />
      {/if}
    </div>
  </div>
</div>
