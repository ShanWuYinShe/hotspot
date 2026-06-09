<script lang="ts">
  import { dev } from '$app/environment';
  import { goto } from '$app/navigation';
  import { authStore } from '$lib/stores/auth.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { toast } from '$lib/stores/toast.svelte';
  import { presenceStore } from '$lib/stores/presence.svelte';
  import { isSuccess } from '$lib/api/client';
  import { reportsService, usersService } from '$lib/api/services';
  import { formatDistance } from '$lib/utils/format';
  import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuSeparator,
    DropdownMenuTrigger
  } from '$lib/components/ui/dropdown-menu';
  import MapPin from 'lucide-svelte/icons/map-pin';
  import MoreVertical from 'lucide-svelte/icons/more-vertical';
  import User from 'lucide-svelte/icons/user';
  import MessageCircle from 'lucide-svelte/icons/message-circle';
  import UserPlus from 'lucide-svelte/icons/user-plus';
  import Loader2 from 'lucide-svelte/icons/loader-2';
  import Share2 from 'lucide-svelte/icons/share-2';
  import Flag from 'lucide-svelte/icons/flag';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import type { UserInfo, UserItem } from '$lib/types/api';

  interface Props {
    user: UserInfo | UserItem;
    showDistance?: boolean;
    showBio?: boolean;
    showTags?: boolean;
    showOnlineStatus?: boolean;
    showMenu?: boolean;
    href?: string;
    class?: string;
    variant?: 'default' | 'compact' | 'detailed' | 'square' | 'nearby-row';
  }

  let {
    user,
    showDistance = false,
    showBio = false,
    showTags = false,
    showOnlineStatus = false,
    showMenu = true,
    href,
    class: className = '',
    variant = 'default'
  }: Props = $props();

  const userInfo = $derived('user' in user ? user.user : user);
  const actualHref = $derived(href ?? `/users/${userInfo.user_id}`);
  const distance = $derived(
    showDistance && 'location' in user && user.location?.distance ? user.location.distance : null
  );

  const isOnline = $derived.by(() => {
    if (!showOnlineStatus) return false;
    const presenceStatus = presenceStore.getStatus(userInfo.user_id);
    if (presenceStatus !== 'offline') return presenceStore.isOnline(userInfo.user_id);
    return 'presence_status' in user && user.presence_status === 'online';
  });

  const displayTags = $derived(
    showTags && 'tags' in user && user.tags ? user.tags.slice(0, 3) : []
  );

  const isMe = $derived(authStore.currentUserId === userInfo.user_id);
  let isFollowing = $state(false);
  let isFollowPending = $state(false);
  let isReporting = $state(false);

  $effect(() => {
    const relation = 'relation' in user ? (user as UserItem).relation : undefined;
    isFollowing = relation?.is_following ?? false;
  });

  function isInteractiveTarget(target: EventTarget | null): boolean {
    return target instanceof HTMLElement
      ? Boolean(
          target.closest(
            'button, a, input, select, textarea, summary, [role="button"], [role="menuitem"], [data-card-action]'
          )
        )
      : false;
  }

  function handleCardClick(event: MouseEvent) {
    if (event.defaultPrevented || isInteractiveTarget(event.target)) return;
    goto(actualHref);
  }

  function handleCardKeydown(event: KeyboardEvent) {
    if (event.defaultPrevented || isInteractiveTarget(event.target)) return;
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      goto(actualHref);
    }
  }

  function handleViewProfile(e: MouseEvent) {
    e.preventDefault();
    e.stopPropagation();
    goto(actualHref);
  }

  function handleSendMessage(e: MouseEvent) {
    e.preventDefault();
    e.stopPropagation();
    if (!authStore.isAuthenticated) {
      uiStore.openLoginModal();
      return;
    }
    goto(`/chat/users/${userInfo.user_id}`);
  }

  async function handleFollow(e: MouseEvent) {
    e.preventDefault();
    e.stopPropagation();
    if (!authStore.isAuthenticated) {
      uiStore.openLoginModal();
      return;
    }
    if (isFollowPending) return;
    isFollowPending = true;
    try {
      const response = isFollowing
        ? await usersService.unfollow({ user_id: userInfo.user_id })
        : await usersService.follow({ user_id: userInfo.user_id, source: 'profile' });
      if (!isSuccess(response)) {
        toast.error(response.message || '操作失败');
        return;
      }
      isFollowing = !isFollowing;
      toast.success(isFollowing ? '关注成功' : '已取消关注');
    } catch {
      toast.error(isFollowing ? '取消关注失败' : '关注失败');
    } finally {
      isFollowPending = false;
    }
  }

  async function handleShare(e: MouseEvent) {
    e.preventDefault();
    e.stopPropagation();
    const shareData = {
      title: `${userInfo.nickname} - HotSpot`,
      text: `查看 ${userInfo.nickname} 的个人主页`,
      url: `${window.location.origin}/users/${userInfo.user_id}`
    };
    if (navigator.share) {
      try {
        await navigator.share(shareData);
      } catch (err) {
        if ((err as Error).name !== 'AbortError' && dev) console.error('分享失败:', err);
      }
    } else {
      await navigator.clipboard.writeText(shareData.url);
      toast.success('链接已复制到剪贴板');
    }
  }

  async function handleReport(e: MouseEvent) {
    e.preventDefault();
    e.stopPropagation();
    if (!authStore.isAuthenticated) {
      uiStore.openLoginModal();
      return;
    }
    if (isReporting) return;
    isReporting = true;
    try {
      const response = await reportsService.create({
        target_type: 'user',
        target_id: userInfo.user_id,
        reason: 'other'
      });
      if (!isSuccess(response)) {
        toast.error(response.message || '举报提交失败');
        return;
      }
      toast.success('举报已提交，我们会尽快处理');
    } catch {
      toast.error('举报提交失败');
    } finally {
      isReporting = false;
    }
  }
</script>

<div
  class="card-hover cursor-pointer overflow-hidden rounded-[22px] outline-none transition-[box-shadow,outline,transform] duration-200 hover:scale-[1.008] active:scale-[0.98] focus-visible:ring-2 focus-visible:ring-[var(--glow)] focus-visible:ring-offset-1 focus-visible:ring-offset-[var(--surface-bg)] {className}"
  style="background: var(--surface-bg); box-shadow: 6px 6px 14px var(--shadow-dark), -6px -6px 14px var(--shadow-light); border: 1px solid var(--border-theme);"
  tabindex={0}
  role="link"
  onclick={handleCardClick}
  onkeydown={handleCardKeydown}
>
  {#if variant === 'nearby-row'}
    <!-- Nearby row: horizontal row for nearby all page -->
    <div class="row-hover flex cursor-pointer items-center gap-3.5 rounded-[22px] px-4 py-3.5">
      <div class="relative flex-shrink-0">
        <div
          class="overflow-hidden rounded-full"
          style="width: 46px; height: 46px; border: 1.5px solid var(--border-theme);"
        >
          <img
            src={userInfo.avatar_url || ''}
            alt={userInfo.nickname}
            class="h-full w-full object-cover"
          />
        </div>
        {#if isOnline}
          <span
            class="absolute bottom-0 right-0 rounded-full"
            style="width: 11px; height: 11px; background: #22c55e; border: 2px solid var(--surface-bg);"
          ></span>
        {/if}
      </div>
      <div class="min-w-0 flex-1">
        <p style="font-size: 14px; color: var(--text-primary);">{userInfo.nickname}</p>
        <div class="mt-0.5 flex items-center gap-1.5">
          <span style="font-size: 11px; color: {isOnline ? '#22c55e' : 'var(--text-secondary)'};">
            {isOnline ? '在线' : '离线'}
          </span>
          {#if distance}
            <span style="font-size: 10px; color: var(--text-secondary); opacity: 0.5;">·</span>
            <div class="flex items-center gap-0.5">
              <MapPin size={10} strokeWidth={1.5} color="var(--brand)" />
              <span style="font-size: 12px; color: var(--brand);">{formatDistance(distance)}</span>
            </div>
          {/if}
          {#if displayTags.length > 0}
            <span style="font-size: 10px; color: var(--text-secondary); opacity: 0.5;">·</span>
            <span style="font-size: 11px; color: var(--brand); opacity: 0.8;"
              >#{displayTags[0]}</span
            >
          {/if}
        </div>
      </div>
      <div class="flex flex-col items-end gap-2">
        {#if distance}
          <div class="flex items-center gap-0.5">
            <MapPin size={10} strokeWidth={1.5} color="var(--brand)" />
            <span style="font-size: 12px; color: var(--brand);">{formatDistance(distance)}</span>
          </div>
        {/if}
        {#if !isMe}
          <button
            data-card-action
            class="rounded-full px-3.5 py-1 outline-none transition-all"
            style={isFollowing
              ? 'box-shadow: inset 3px 3px 6px var(--shadow-dark), inset -3px -3px 6px var(--shadow-light); border: 1px solid var(--border-active); color: var(--brand); background: var(--surface-bg); font-size: 11px;'
              : 'box-shadow: 3px 3px 7px var(--shadow-dark), -3px -3px 7px var(--shadow-light); border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg); font-size: 11px;'}
            onclick={handleFollow}
            disabled={isFollowPending}
          >
            {#if isFollowPending}
              ...
            {:else}
              {isFollowing ? '已关注' : '关注'}
            {/if}
          </button>
        {/if}
      </div>
    </div>
  {:else if variant === 'square'}
    <!-- Square: info-dense card -->
    <div class="p-4">
      <div
        class="rounded-[18px] p-3"
        style="background: var(--surface-bg); box-shadow: inset 3px 3px 6px var(--shadow-dark), inset -3px -3px 6px var(--shadow-light); border: 1px solid var(--border-theme);"
      >
        <div class="flex items-start justify-between gap-3">
          <div class="flex min-w-0 items-center gap-2.5">
            <div class="relative flex-shrink-0">
              <div
                class="overflow-hidden rounded-full"
                style="width: 46px; height: 46px; border: 1.5px solid var(--border-theme);"
              >
                <img
                  src={userInfo.avatar_url || ''}
                  alt={userInfo.nickname}
                  class="h-full w-full object-cover"
                />
              </div>
              {#if isOnline}
                <span
                  class="absolute bottom-0 right-0 rounded-full"
                  style="width: 11px; height: 11px; background: #22c55e; border: 2px solid var(--surface-bg);"
                ></span>
              {/if}
            </div>
            <div class="min-w-0 flex-1">
              <p style="font-size: 14px; font-weight: 500; color: var(--text-primary);">
                {userInfo.nickname}
              </p>
              <p style="font-size: 11px; color: var(--text-secondary); margin-top: 1px;">
                {isOnline ? '在线' : '离线'}
              </p>
            </div>
          </div>
          {#if distance}
            <div class="flex flex-shrink-0 items-center gap-0.5">
              <MapPin size={10} strokeWidth={1.5} color="var(--brand)" />
              <span style="font-size: 10px; color: var(--brand);">{formatDistance(distance)}</span>
            </div>
          {/if}
        </div>

        {#if showBio && 'bio' in user && user.bio}
          <p
            class="mt-3 line-clamp-3"
            style="font-size: 12px; color: var(--text-secondary); line-height: 1.5;"
          >
            {user.bio}
          </p>
        {:else if showBio}
          <p style="font-size: 11px; color: var(--text-secondary); margin-top: 10px;">
            这个人很神秘，暂时还没有留下简介
          </p>
        {/if}
      </div>

      {#if displayTags.length > 0}
        <div class="mt-3 flex flex-wrap gap-1.5">
          {#each displayTags as tag (tag)}
            <span
              class="rounded-full px-2 py-0.5"
              style="font-size: 10px; color: var(--brand); background: rgba(99,102,241,0.12); border: 1px solid rgba(99,102,241,0.25);"
              >#{tag}</span
            >
          {/each}
        </div>
      {/if}
    </div>
  {:else if variant === 'compact'}
    <!-- Compact: horizontal list row -->
    <div class="flex items-center gap-3 p-3.5">
      <div class="relative flex-shrink-0">
        <div
          class="overflow-hidden rounded-full"
          style="width: 44px; height: 44px; border: 1.5px solid var(--border-theme);"
        >
          <img
            src={userInfo.avatar_url || ''}
            alt={userInfo.nickname}
            class="h-full w-full object-cover"
          />
        </div>
        {#if isOnline}
          <span
            class="absolute bottom-0 right-0 rounded-full"
            style="width: 11px; height: 11px; background: #22c55e; border: 2px solid var(--surface-bg);"
          ></span>
        {/if}
      </div>
      <div class="min-w-0 flex-1">
        <div class="flex items-center justify-between">
          <p style="font-size: 14px; font-weight: 500; color: var(--text-primary);">
            {userInfo.nickname}
          </p>
          {#if distance}
            <div class="flex items-center gap-0.5">
              <MapPin size={10} strokeWidth={1.5} color="var(--brand)" />
              <span style="font-size: 10px; color: var(--brand);">{formatDistance(distance)}</span>
            </div>
          {/if}
        </div>
        {#if showBio && 'bio' in user && user.bio}
          <p class="mt-0.5 truncate" style="font-size: 12px; color: var(--text-secondary);">
            {user.bio}
          </p>
        {/if}
      </div>
      <ChevronRight size={16} strokeWidth={1.5} color="var(--text-secondary)" />
    </div>
  {:else if variant === 'detailed'}
    <!-- Detailed: vertical with more info -->
    <div class="p-4">
      <div class="flex items-center gap-4">
        <div class="relative flex-shrink-0">
          <div
            class="overflow-hidden rounded-full"
            style="width: 52px; height: 52px; border: 1.5px solid var(--border-theme);"
          >
            <img
              src={userInfo.avatar_url || ''}
              alt={userInfo.nickname}
              class="h-full w-full object-cover"
            />
          </div>
          {#if isOnline}
            <span
              class="absolute bottom-0 right-0 rounded-full"
              style="width: 11px; height: 11px; background: #22c55e; border: 2px solid var(--surface-bg);"
            ></span>
          {/if}
        </div>
        <div class="min-w-0 flex-1">
          <p style="font-size: 16px; font-weight: 600; color: var(--text-primary);">
            {userInfo.nickname}
          </p>
          {#if distance}
            <div class="mt-1 flex items-center gap-0.5">
              <MapPin size={12} strokeWidth={1.5} color="var(--brand)" />
              <span style="font-size: 12px; color: var(--brand);">{formatDistance(distance)}</span>
            </div>
          {/if}
        </div>
      </div>
      {#if showBio && 'bio' in user && user.bio}
        <p class="mt-3" style="font-size: 13px; color: var(--text-secondary); line-height: 1.5;">
          {user.bio}
        </p>
      {/if}
      {#if displayTags.length > 0}
        <div class="mt-3 flex flex-wrap gap-1.5">
          {#each displayTags as tag (tag)}
            <span
              class="rounded-full px-2 py-0.5"
              style="font-size: 10px; color: var(--brand); background: rgba(99,102,241,0.12); border: 1px solid rgba(99,102,241,0.25);"
              >#{tag}</span
            >
          {/each}
        </div>
      {/if}
    </div>
  {:else}
    <!-- Default: balanced horizontal -->
    <div class="p-4">
      <div class="flex items-center gap-3.5">
        <div class="relative flex-shrink-0">
          <div
            class="overflow-hidden rounded-full"
            style="width: 44px; height: 44px; border: 1.5px solid var(--border-theme);"
          >
            <img
              src={userInfo.avatar_url || ''}
              alt={userInfo.nickname}
              class="h-full w-full object-cover"
            />
          </div>
          {#if isOnline}
            <span
              class="absolute bottom-0 right-0 rounded-full"
              style="width: 11px; height: 11px; background: #22c55e; border: 2px solid var(--surface-bg);"
            ></span>
          {/if}
        </div>
        <div class="min-w-0 flex-1">
          <p style="font-size: 14px; font-weight: 500; color: var(--text-primary);">
            {userInfo.nickname}
          </p>
          {#if distance}
            <div class="mt-0.5 flex items-center gap-0.5">
              <MapPin size={10} strokeWidth={1.5} color="var(--brand)" />
              <span style="font-size: 10px; color: var(--brand);">{formatDistance(distance)}</span>
            </div>
          {/if}
        </div>
        <ChevronRight size={16} strokeWidth={1.5} color="var(--text-secondary)" />
      </div>
      {#if showBio && 'bio' in user && user.bio}
        <p
          class="mt-2.5 line-clamp-2"
          style="font-size: 12px; color: var(--text-secondary); line-height: 1.5;"
        >
          {user.bio}
        </p>
      {/if}
      {#if displayTags.length > 0}
        <div class="mt-2.5 flex flex-wrap gap-1.5">
          {#each displayTags as tag (tag)}
            <span
              class="rounded-full px-2 py-0.5"
              style="font-size: 10px; color: var(--brand); background: rgba(99,102,241,0.12); border: 1px solid rgba(99,102,241,0.25);"
              >#{tag}</span
            >
          {/each}
        </div>
      {/if}
    </div>
  {/if}

  <!-- Action buttons -->
  {#if !isMe}
    <div style="border-top: 1px solid var(--border-theme);" class="px-4 py-2.5">
      <div class="flex items-center justify-between">
        <button
          data-card-action
          class="flex items-center gap-1.5 py-1 outline-none transition-transform duration-150 active:scale-[0.93]"
          style="color: var(--text-secondary);"
          onclick={handleSendMessage}
        >
          <MessageCircle size={15} strokeWidth={1.5} />
          <span style="font-size: 12px;">消息</span>
        </button>
        <div class="flex items-center gap-2">
          <button
            data-card-action
            class="flex items-center gap-1.5 py-1 outline-none transition-transform duration-150 active:scale-[0.93]"
            style="color: {isFollowing ? 'var(--brand)' : 'var(--text-secondary)'};"
            onclick={handleFollow}
            disabled={isFollowPending}
          >
            {#if isFollowPending}
              <Loader2 size={15} strokeWidth={1.5} class="animate-spin" />
            {:else}
              <UserPlus size={15} strokeWidth={1.5} />
            {/if}
            <span style="font-size: 11px;">{isFollowing ? '已关注' : '关注'}</span>
          </button>
          {#if showMenu}
            <DropdownMenu>
              <DropdownMenuTrigger>
                <button
                  data-card-action
                  class="flex items-center justify-center rounded-full outline-none transition-transform duration-150 active:scale-[0.92]"
                  style="width: 30px; height: 30px; background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme);"
                  onclick={(e: MouseEvent) => {
                    e.preventDefault();
                    e.stopPropagation();
                  }}
                  aria-label="更多操作"
                >
                  <MoreVertical size={14} strokeWidth={1.5} color="var(--text-secondary)" />
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" class="w-48">
                <DropdownMenuItem onclick={handleViewProfile}>
                  <User class="mr-2 h-4 w-4" />
                  <span>查看主页</span>
                </DropdownMenuItem>
                <DropdownMenuItem onclick={handleShare}>
                  <Share2 class="mr-2 h-4 w-4" />
                  <span>分享</span>
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem onclick={handleReport} class="text-destructive">
                  <Flag class="mr-2 h-4 w-4" />
                  <span>举报</span>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          {/if}
        </div>
      </div>
    </div>
  {/if}
</div>
