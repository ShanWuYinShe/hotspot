<script lang="ts">
  import { goto } from '$app/navigation';
  import { untrack } from 'svelte';
  import { usersService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { formatHandle, formatCount } from '$lib/utils/format';
  import { authStore } from '$lib/stores/auth.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { toast } from '$lib/stores/toast.svelte';
  import { usePagination } from '$lib/hooks/use-pagination.svelte';
  import type { UserItem, MomentItem, FollowerItem, FollowingItem } from '$lib/types/api';
  import ChevronLeft from 'lucide-svelte/icons/chevron-left';
  import MapPin from 'lucide-svelte/icons/map-pin';
  import Image from 'lucide-svelte/icons/image';
  import MessageCircle from 'lucide-svelte/icons/message-circle';
  import UserPlus from 'lucide-svelte/icons/user-plus';
  import UserCheck from 'lucide-svelte/icons/user-check';
  import Loader2 from 'lucide-svelte/icons/loader-2';
  import Share2 from 'lucide-svelte/icons/share-2';
  import MoreHorizontal from 'lucide-svelte/icons/more-horizontal';
  import Grid3X3 from 'lucide-svelte/icons/grid-3x3';
  import Avatar from '$lib/components/common/avatar.svelte';
  import type { PageData } from './$types';

  let { data }: { data: PageData } = $props();

  const userId = $derived(data.userId);

  let user = $state<UserItem | null>(null);
  let isFollowing = $state(false);
  let isActionLoading = $state(false);
  let currentTab = $state<'moments' | 'followers' | 'following'>('moments');

  // userId 变化时重置子 tab 分页状态并触发首次加载
  $effect(() => {
    userId; // 依赖声明
    untrack(() => {
      user = data.user;
      isFollowing = data.user?.relation?.is_following ?? false;
      currentTab = 'moments';
      momentsPagination.reset();
      followersPagination.reset();
      followingPagination.reset();
      // 首次加载 moments（仅当 user 数据可用时）
      if (user) {
        loadMoments(true);
      }
    });
  });

  // 使用 usePagination hook 统一管理三个 tab 的分页状态
  const momentsPagination = usePagination<MomentItem>({
    initialLoading: false,
    uniqueKey: 'moment_id'
  });

  const followersPagination = usePagination<FollowerItem>({
    initialLoading: false,
    uniqueKey: (item) => item.user.user_id
  });

  const followingPagination = usePagination<FollowingItem>({
    initialLoading: false,
    uniqueKey: (item) => item.user.user_id
  });

  async function handleFollow() {
    if (!authStore.isAuthenticated) {
      uiStore.openLoginModalWithRedirect({ onSuccess: handleFollow });
      return;
    }

    isActionLoading = true;
    try {
      if (isFollowing) {
        const response = await usersService.unfollow({ user_id: userId });
        if (isSuccess(response)) {
          isFollowing = false;
          // 更新 user.relation
          if (user) {
            user = {
              ...user,
              relation: user.relation ? { ...user.relation, is_following: false } : undefined
            };
          }
          toast.success('已取消关注');
        } else {
          toast.error(response.message || '操作失败');
        }
      } else {
        const response = await usersService.follow({ user_id: userId });
        if (isSuccess(response)) {
          isFollowing = true;
          // 更新 user.relation
          if (user) {
            user = {
              ...user,
              relation: user.relation
                ? { ...user.relation, is_following: true }
                : { is_following: true, is_followed_by: false, is_blocked: false }
            };
          }
          toast.success('关注成功');
        } else {
          // 统一使用 response.message 展示错误（spec 哲学：code 驱动行为，message 描述上下文）
          toast.error(response.message || '操作失败');
        }
      }
    } catch {
      toast.error('操作失败');
    } finally {
      isActionLoading = false;
    }
  }

  // 使用 usePagination hook，大大简化分页逻辑
  async function loadMoments(refresh = false) {
    await momentsPagination.load(
      (cursor) =>
        usersService.getMoments({
          user_id: userId,
          pagination: { limit: 12, cursor }
        }),
      refresh
    );

    if (momentsPagination.error) {
      toast.error('加载瞬间失败');
    }
  }

  async function loadFollowers(refresh = false) {
    await followersPagination.load(
      (cursor) =>
        usersService.getFollowers({
          user_id: userId,
          pagination: { limit: 20, cursor }
        }),
      refresh
    );

    if (followersPagination.error) {
      toast.error('加载粉丝失败');
    }
  }

  async function loadFollowing(refresh = false) {
    await followingPagination.load(
      (cursor) =>
        usersService.getFollowing({
          user_id: userId,
          pagination: { limit: 20, cursor }
        }),
      refresh
    );

    if (followingPagination.error) {
      toast.error('加载关注失败');
    }
  }

  function handleTabChange(next: string) {
    const target = next as 'moments' | 'followers' | 'following';
    currentTab = target;
    if (target === 'moments' && momentsPagination.items.length === 0) loadMoments(true);
    if (target === 'followers' && followersPagination.items.length === 0) loadFollowers(true);
    if (target === 'following' && followingPagination.items.length === 0) loadFollowing(true);
  }

  // 数据错误提示（独立 effect，避免 user 变化导致重复触发）
  $effect(() => {
    if (data.error) {
      toast.error(data.error);
    }
  });
</script>

<svelte:head>
  <title>{user?.user.nickname ?? '用户'} - HotSpot</title>
</svelte:head>

<div class="page-pad">
  <div class="page-shell page-shell--narrow min-h-screen">
    {#if !user}
      <div class="flex flex-col items-center justify-center px-4 py-20">
        <p style="color: var(--text-secondary);">{data.error || '用户不存在'}</p>
        <button
          class="mt-4 flex items-center gap-2 rounded-full px-5 py-2 font-medium outline-none"
          style="background: var(--brand); color: var(--text-on-accent);"
          onclick={() => history.back()}
        >
          <ChevronLeft size={16} />
          返回
        </button>
      </div>
    {:else}
      <!-- Banner -->
      <div
        class="relative overflow-hidden"
        style="height: clamp(150px, 25vw, 210px); flex-shrink: 0;"
      >
        {#if user.user.avatar_url}
          <img
            src={user.user.avatar_url}
            alt="banner"
            class="absolute inset-0 h-full w-full object-cover"
          />
        {:else}
          <div
            class="absolute inset-0"
            style="background: linear-gradient(135deg, var(--brand), color-mix(in srgb, var(--brand) 48%, var(--surface-bg)));"
          ></div>
        {/if}
        <!-- Gradient overlay -->
        <div
          class="absolute inset-0"
          style="background: linear-gradient(to bottom, rgba(0,0,0,0.15) 0%, rgba(0,0,0,0.55) 100%);"
        ></div>

        <!-- Top floating glass buttons -->
        <div class="absolute left-0 right-0 top-0 flex items-center justify-between px-4 pt-4">
          <button
            class="flex items-center justify-center rounded-full outline-none"
            style="width: 38px; height: 38px; background: rgba(0,0,0,0.35); backdrop-filter: blur(10px); border: 1px solid var(--glass-border);"
            onclick={() => history.back()}
            aria-label="返回"
          >
            <ChevronLeft size={15} color="var(--text-on-overlay)" strokeWidth={1.75} />
          </button>
          <div class="flex gap-2">
            <button
              class="flex items-center justify-center rounded-full outline-none"
              style="width: 38px; height: 38px; background: rgba(0,0,0,0.35); backdrop-filter: blur(10px); border: 1px solid var(--glass-border);"
              aria-label="分享"
            >
              <Share2 size={14} color="var(--text-on-overlay)" strokeWidth={1.75} />
            </button>
            <button
              class="flex items-center justify-center rounded-full outline-none"
              style="width: 38px; height: 38px; background: rgba(0,0,0,0.35); backdrop-filter: blur(10px); border: 1px solid var(--glass-border);"
              aria-label="更多"
            >
              <MoreHorizontal size={14} color="var(--text-on-overlay)" strokeWidth={1.75} />
            </button>
          </div>
        </div>
      </div>

      <!-- Avatar + info -->
      <div class="px-5 -mt-10 pb-4 flex flex-col">
        <div class="flex items-end justify-between mb-3">
          <!-- Avatar overlapping banner -->
          <div class="relative">
            <div
              class="overflow-hidden rounded-[var(--radius-lg)]"
              style="width: 76px; height: 76px; border: 3px solid var(--surface-bg); box-shadow: var(--shadow-lg);"
            >
              <Avatar
                src={user.user.avatar_url}
                name={user.user.nickname}
                alt={user.user.nickname}
                size={80}
                shape="rounded"
                border={false}
              />
            </div>
            <!-- Online status dot -->
            <span
              class="absolute bottom-1 right-1 rounded-full"
              style="width: 12px; height: 12px; background: var(--semantic-online); border: 2px solid var(--surface-bg);"
            ></span>
          </div>

          <!-- Action buttons -->
          {#if authStore.isAuthenticated && authStore.currentUserId !== userId}
            <div class="flex gap-2 pb-1">
              <button
                class="text-meta surface-flat flex items-center gap-1.5 rounded-full px-3.5 py-2 outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
                style="color: var(--text-secondary)"
                onclick={() => goto(`/chat/users/${userId}`)}
              >
                <MessageCircle size={14} strokeWidth={1.5} />
                私信
              </button>
              <button
                class={isFollowing
                  ? 'surface-flat flex items-center gap-1.5 rounded-full px-4 py-2 outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]'
                  : 'flex items-center gap-1.5 rounded-full px-4 py-2 outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]'}
                style={isFollowing
                  ? 'border: 1.5px solid var(--brand); color: var(--brand); font-size: var(--text-meta);'
                  : 'background: var(--brand); color: var(--text-on-accent); font-size: var(--text-meta); box-shadow: var(--shadow-brand-btn); border: 1px solid transparent;'}
                onclick={handleFollow}
                disabled={isActionLoading}
              >
                {#if isActionLoading}
                  <Loader2 size={14} class="animate-spin" />
                {:else if isFollowing}
                  <UserCheck size={14} strokeWidth={1.5} />
                  已关注
                {:else}
                  <UserPlus size={14} strokeWidth={1.5} />
                  关注
                {/if}
              </button>
            </div>
          {/if}
        </div>

        <!-- Profile card: name / handle / bio / tags -->
        <div class="surface-flat rounded-[var(--radius-lg)] px-5 py-4">
          <h2 class="text-h1" style="color: var(--text-primary)">
            {user.user.nickname}
          </h2>
          <p class="text-body" style="color: var(--text-secondary); margin-top: 1px">
            @{formatHandle(user.user)}
          </p>
          {#if user.bio}
            <p
              class="text-body"
              style="color: var(--text-secondary); line-height: 1.65; margin-top: 8px"
            >
              {user.bio}
            </p>
          {/if}
          {#if user.location}
            <div class="mt-2 flex items-center gap-1.5">
              <MapPin size={11} color="var(--brand)" strokeWidth={1.5} />
              <span class="text-caption" style="color: var(--text-secondary)"
                >{user.location.location_name}</span
              >
            </div>
          {/if}
          {#if user.tags && user.tags.length > 0}
            <div class="mt-3 flex flex-wrap gap-2">
              {#each user.tags as tag (tag)}
                <span class="tag-chip">#{tag}</span>
              {/each}
            </div>
          {/if}

          {#if user.social_context?.mutual_friends_count}
            <p class="text-body" style="color: var(--text-secondary); margin-top: 8px">
              你们有 {user.social_context.mutual_friends_count} 个共同好友
            </p>
          {/if}
        </div>

        <!-- Stats as single segmented bar -->
        <div class="surface-flat mt-5 flex rounded-[var(--radius-lg)] overflow-hidden">
          {#each [{ label: '帖子', value: formatCount(user.stats?.moments_count ?? 0) }, { label: '粉丝', value: formatCount(user.stats?.followers_count ?? 0) }, { label: '关注', value: formatCount(user.stats?.following_count ?? 0) }] as stat, i (stat.label)}
            <div class="relative flex flex-1 flex-col items-center py-4">
              {#if i > 0}
                <div
                  class="absolute left-0 top-1/2 -translate-y-1/2"
                  style="width: 1px; height: 28px; background: var(--border-theme);"
                ></div>
              {/if}
              <span class="text-h1" style="color: var(--text-primary)">{stat.value}</span>
              <span
                class="text-xs mt-1 uppercase"
                style="letter-spacing: 0.1em; color: var(--text-secondary)"
              >
                {stat.label}
              </span>
            </div>
          {/each}
        </div>
      </div>

      <!-- Tabs -->
      <div
        class="flex"
        style="border-bottom: 1px solid var(--border-theme); padding-left: 16px; padding-right: 16px;"
      >
        {#each [{ id: 'moments' as const, icon: Grid3X3, label: '帖子' }, { id: 'followers' as const, icon: UserCheck, label: '粉丝' }, { id: 'following' as const, icon: UserPlus, label: '关注' }] as tab (tab.id)}
          {@const active = currentTab === tab.id}
          <button
            class="text-body relative flex flex-1 items-center justify-center gap-1.5 pb-3 pt-1.5 outline-none transition-all duration-150"
            style="color: {active ? 'var(--brand)' : 'var(--text-secondary)'}; font-weight: {active
              ? 500
              : 400}"
            onclick={() => handleTabChange(tab.id)}
          >
            <tab.icon size={13} strokeWidth={active ? 2 : 1.5} />
            {tab.label}
            {#if active}
              <div
                class="absolute bottom-[-1px]"
                style="left: 18%; right: 18%; height: 2px; border-radius: 1px; background: var(--brand);"
              ></div>
            {/if}
          </button>
        {/each}
      </div>

      <!-- Tab content: Moments / Gallery -->
      {#if currentTab === 'moments'}
        {#if momentsPagination.loading}
          <div
            style="display: grid; grid-template-columns: repeat(3, 1fr); gap: 3px; padding: 3px;"
          >
            {#each Array(9) as _, i (i)}
              <div
                style="aspect-ratio: 1; background: var(--border-theme); border-radius: 4px;"
                class="animate-pulse"
              ></div>
            {/each}
          </div>
        {:else if momentsPagination.items.length === 0}
          <div class="flex flex-col items-center justify-center py-12">
            <Image size={40} color="var(--text-secondary)" />
            <p style="margin-top: 16px; color: var(--text-secondary);">TA 还没有发布瞬间</p>
          </div>
        {:else}
          <!-- 3-column gallery grid from moments' images -->
          <div
            style="display: grid; grid-template-columns: repeat(3, 1fr); gap: 3px; padding: 3px;"
          >
            {#each momentsPagination.items as moment (moment.moment_id)}
              {#if moment.images && moment.images.length > 0}
                <a
                  href="/moments/{moment.moment_id}"
                  class="relative cursor-pointer overflow-hidden"
                  style="aspect-ratio: 1;"
                >
                  <img
                    src={moment.images![0]!.thumbnail_url || moment.images![0]!.url}
                    alt={moment.description}
                    class="h-full w-full object-cover transition-transform duration-150 hover:scale-[1.03]"
                  />
                </a>
              {/if}
            {/each}
          </div>
          {#if momentsPagination.hasMore}
            <div class="flex justify-center pt-4">
              <button
                class="surface-flat flex items-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
                style="color: var(--text-secondary);"
                onclick={() => loadMoments(false)}
                disabled={momentsPagination.loadingMore}
              >
                {#if momentsPagination.loadingMore}
                  <Loader2 size={16} class="animate-spin" />
                {/if}
                加载更多
              </button>
            </div>
          {/if}
        {/if}
      {/if}

      <!-- Tab content: Followers (粉丝) -->
      {#if currentTab === 'followers'}
        {#if followersPagination.loading}
          <div
            style="display: grid; grid-template-columns: repeat(3, 1fr); gap: 3px; padding: 3px;"
          >
            {#each Array(9) as _, i (i)}
              <div
                style="aspect-ratio: 1; background: var(--border-theme); border-radius: 4px;"
                class="animate-pulse"
              ></div>
            {/each}
          </div>
        {:else if followersPagination.items.length === 0}
          <div class="flex flex-col items-center justify-center py-12">
            <UserCheck size={40} color="var(--text-secondary)" />
            <p style="margin-top: 16px; color: var(--text-secondary);">暂无粉丝</p>
          </div>
        {:else}
          <div class="grid gap-3 p-4 sm:grid-cols-2">
            {#each followersPagination.items as follower (follower.user.user_id)}
              <a href="/users/{follower.user.user_id}">
                <div
                  class="surface-flat flex cursor-pointer items-center gap-3 rounded-[var(--radius-lg)] p-4 transition-all active:scale-[0.98]"
                >
                  <Avatar
                    src={follower.user.avatar_url}
                    name={follower.user.nickname}
                    alt={follower.user.nickname}
                    size={48}
                  />
                  <div class="min-w-0">
                    <p
                      class="text-body-lg truncate"
                      style="font-weight: 500; color: var(--text-primary)"
                    >
                      {follower.user.nickname}
                    </p>
                    <p class="text-meta" style="color: var(--text-secondary)">查看主页</p>
                  </div>
                </div>
              </a>
            {/each}
          </div>
          {#if followersPagination.hasMore}
            <div class="flex justify-center pt-4">
              <button
                class="surface-flat flex items-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
                style="color: var(--text-secondary);"
                onclick={() => loadFollowers(false)}
                disabled={followersPagination.loadingMore}
              >
                {#if followersPagination.loadingMore}
                  <Loader2 size={16} class="animate-spin" />
                {/if}
                加载更多
              </button>
            </div>
          {/if}
        {/if}
      {/if}

      <!-- Tab content: Following (关注) -->
      {#if currentTab === 'following'}
        {#if followingPagination.loading}
          <div
            style="display: grid; grid-template-columns: repeat(3, 1fr); gap: 3px; padding: 3px;"
          >
            {#each Array(9) as _, i (i)}
              <div
                style="aspect-ratio: 1; background: var(--border-theme); border-radius: 4px;"
                class="animate-pulse"
              ></div>
            {/each}
          </div>
        {:else if followingPagination.items.length === 0}
          <div class="flex flex-col items-center justify-center py-12">
            <UserPlus size={40} color="var(--text-secondary)" />
            <p style="margin-top: 16px; color: var(--text-secondary);">暂无关注</p>
          </div>
        {:else}
          <div class="grid gap-3 p-4 sm:grid-cols-2">
            {#each followingPagination.items as follow (follow.user.user_id)}
              <a href="/users/{follow.user.user_id}">
                <div
                  class="surface-flat flex cursor-pointer items-center gap-3 rounded-[var(--radius-lg)] p-4 transition-all active:scale-[0.98]"
                >
                  <Avatar
                    src={follow.user.avatar_url}
                    name={follow.user.nickname}
                    alt={follow.user.nickname}
                    size={48}
                  />
                  <div class="min-w-0">
                    <p
                      class="text-body-lg truncate"
                      style="font-weight: 500; color: var(--text-primary)"
                    >
                      {follow.user.nickname}
                    </p>
                    <p class="text-meta" style="color: var(--text-secondary)">查看主页</p>
                  </div>
                </div>
              </a>
            {/each}
          </div>
          {#if followingPagination.hasMore}
            <div class="flex justify-center pt-4">
              <button
                class="surface-flat flex items-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
                style="color: var(--text-secondary);"
                onclick={() => loadFollowing(false)}
                disabled={followingPagination.loadingMore}
              >
                {#if followingPagination.loadingMore}
                  <Loader2 size={16} class="animate-spin" />
                {/if}
                加载更多
              </button>
            </div>
          {/if}
        {/if}
      {/if}
    {/if}
  </div>
</div>
