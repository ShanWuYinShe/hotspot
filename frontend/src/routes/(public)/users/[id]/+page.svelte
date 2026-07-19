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
  import Spinner from '$lib/components/common/spinner.svelte';
  import EmptyState from '$lib/components/common/empty-state.svelte';
  import FollowButton from '$lib/components/common/follow-button.svelte';
  import Share2 from 'lucide-svelte/icons/share-2';
  import MoreHorizontal from 'lucide-svelte/icons/more-horizontal';
  import Grid3X3 from 'lucide-svelte/icons/grid-3x3';
  import Avatar from '$lib/components/common/avatar.svelte';
  import OnlineDot from '$lib/components/common/online-dot.svelte';
  import UserCard from '$lib/components/common/user-card.svelte';
  import Skeleton from '$lib/components/ui/skeleton/skeleton.svelte';
  import SmartImage from '$lib/components/common/smart-image.svelte';
  import { t } from '$lib/i18n';
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
      // 首屏优先使用 getProfile 返回的近期瞬间预览，避免额外请求
      if (user) {
        if (data.recentMoments && data.recentMoments.items.length > 0) {
          momentsPagination.setItems(data.recentMoments.items);
        } else {
          loadMoments(true);
        }
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
          toast.success(t('userProfile.unfollowed'));
        } else {
          toast.error(response.message || t('userProfile.operationFailed'));
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
          toast.success(t('userProfile.followSuccess'));
        } else {
          // 统一使用 response.message 展示错误（spec 哲学：code 驱动行为，message 描述上下文）
          toast.error(response.message || t('userProfile.operationFailed'));
        }
      }
    } catch {
      toast.error(t('userProfile.operationFailed'));
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
      toast.error(t('userProfile.loadMomentsFailed'));
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
      toast.error(t('userProfile.loadFollowersFailed'));
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
      toast.error(t('userProfile.loadFollowingFailed'));
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
  <title>{user?.user.nickname ?? t('userProfile.titleDefault')} - HotSpot</title>
</svelte:head>

<div class="page-pad flex h-full flex-col overflow-y-auto no-scrollbar">
  <div class="page-shell page-shell--narrow min-h-full">
    {#if !user}
      <div class="flex flex-col items-center justify-center px-4 py-20">
        <p style="color: var(--text-secondary);">{data.error || t('userProfile.userNotExist')}</p>
        <button
          class="mt-4 flex items-center gap-2 rounded-full px-5 py-2 font-medium outline-none"
          style="background: var(--brand); color: var(--text-on-accent);"
          onclick={() => history.back()}
        >
          <ChevronLeft size={16} />
          {t('userProfile.back')}
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
            aria-label={t('userProfile.backAria')}
          >
            <ChevronLeft size={15} color="var(--text-on-overlay)" strokeWidth={1.75} />
          </button>
          <div class="flex gap-2">
            <button
              class="flex items-center justify-center rounded-full outline-none"
              style="width: 38px; height: 38px; background: rgba(0,0,0,0.35); backdrop-filter: blur(10px); border: 1px solid var(--glass-border);"
              aria-label={t('userProfile.shareAria')}
            >
              <Share2 size={14} color="var(--text-on-overlay)" strokeWidth={1.75} />
            </button>
            <button
              class="flex items-center justify-center rounded-full outline-none"
              style="width: 38px; height: 38px; background: rgba(0,0,0,0.35); backdrop-filter: blur(10px); border: 1px solid var(--glass-border);"
              aria-label={t('userProfile.moreAria')}
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
              style="width: 76px; height: 76px; border: 3px solid var(--surface-bg);"
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
            <OnlineDot size="lg" />
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
                {t('userProfile.message')}
              </button>
              <FollowButton
                following={isFollowing}
                loading={isActionLoading}
                onclick={handleFollow}
              />
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
              {t('userProfile.mutualFriends', { count: user.social_context.mutual_friends_count })}
            </p>
          {/if}
        </div>

        <!-- Stats as single segmented bar -->
        <div class="surface-flat mt-5 flex rounded-[var(--radius-lg)] overflow-hidden">
          {#each [{ label: t('userProfile.statPosts'), value: formatCount(user.stats?.moments_count ?? 0) }, { label: t('userProfile.statFollowers'), value: formatCount(user.stats?.followers_count ?? 0) }, { label: t('userProfile.statFollowing'), value: formatCount(user.stats?.following_count ?? 0) }] as stat, i (stat.label)}
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
        {#each [{ id: 'moments' as const, icon: Grid3X3, label: t('userProfile.tabMoments') }, { id: 'followers' as const, icon: UserCheck, label: t('userProfile.tabFollowers') }, { id: 'following' as const, icon: UserPlus, label: t('userProfile.tabFollowing') }] as tab (tab.id)}
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
              <Skeleton
                style="aspect-ratio: 1; border-radius: var(--radius-xs);"
              />
            {/each}
          </div>
        {:else if momentsPagination.items.length === 0}
          <EmptyState
            icon={Image}
            title={t('userProfile.noMoments')}
            variant="inline"
          />
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
                  <SmartImage
                    src={moment.images![0]!.url}
                    thumbnail={moment.images![0]!.thumbnail_url}
                    alt={moment.description}
                    width={moment.images![0]!.width}
                    height={moment.images![0]!.height}
                    sizes="150px"
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
                  <Spinner size="md" />
                {/if}
                {t('userProfile.loadMore')}
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
              <Skeleton
                style="aspect-ratio: 1; border-radius: var(--radius-xs);"
              />
            {/each}
          </div>
        {:else if followersPagination.items.length === 0}
          <EmptyState
            icon={UserCheck}
            title={t('userProfile.noFollowers')}
            variant="inline"
          />
        {:else}
          <div class="grid gap-3 p-4 sm:grid-cols-2">
            {#each followersPagination.items as follower (follower.user.user_id)}
              <UserCard
                variant="member-row"
                user={follower.user}
                subtitle={t('userProfile.viewProfile')}
              />
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
                  <Spinner size="md" />
                {/if}
                {t('userProfile.loadMore')}
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
              <Skeleton
                style="aspect-ratio: 1; border-radius: var(--radius-xs);"
              />
            {/each}
          </div>
        {:else if followingPagination.items.length === 0}
          <EmptyState
            icon={UserPlus}
            title={t('userProfile.noFollowing')}
            variant="inline"
          />
        {:else}
          <div class="grid gap-3 p-4 sm:grid-cols-2">
            {#each followingPagination.items as follow (follow.user.user_id)}
              <UserCard
                variant="member-row"
                user={follow.user}
                subtitle={t('userProfile.viewProfile')}
              />
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
                  <Spinner size="md" />
                {/if}
                {t('userProfile.loadMore')}
              </button>
            </div>
          {/if}
        {/if}
      {/if}
    {/if}
  </div>
</div>
