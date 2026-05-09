<script lang="ts">
  import { dev } from '$app/environment';
  import { goto } from '$app/navigation';
  import { authStore } from '$lib/stores/auth.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { toast } from '$lib/stores/toast.svelte';
  import { isSuccess } from '$lib/api/client';
  import { groupsService, reportsService } from '$lib/api/services';
  import { formatDistance } from '$lib/utils/format';
  import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuSeparator,
    DropdownMenuTrigger
  } from '$lib/components/ui/dropdown-menu';
  import Users from 'lucide-svelte/icons/users';
  import MapPin from 'lucide-svelte/icons/map-pin';
  import Lock from 'lucide-svelte/icons/lock';
  import MoreVertical from 'lucide-svelte/icons/more-vertical';
  import Info from 'lucide-svelte/icons/info';
  import UserPlus from 'lucide-svelte/icons/user-plus';
  import Loader2 from 'lucide-svelte/icons/loader-2';
  import Share2 from 'lucide-svelte/icons/share-2';
  import Flag from 'lucide-svelte/icons/flag';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import type { GroupItem, GroupInfo } from '$lib/types/api';

  interface Props {
    group: GroupItem | GroupInfo;
    showDistance?: boolean;
    showMembers?: boolean;
    showUnread?: boolean;
    showMenu?: boolean;
    href?: string;
    class?: string;
    variant?: 'default' | 'compact' | 'square';
  }

  let {
    group,
    showDistance = false,
    showMembers = true,
    showUnread = false,
    showMenu = true,
    href,
    class: className = '',
    variant = 'default'
  }: Props = $props();

  const groupInfo = $derived('group' in group ? group.group : group);
  const actualHref = $derived(href ?? `/groups/${groupInfo.group_id}`);

  const distance = $derived.by(() => {
    if (!showDistance || !('location' in group)) return null;
    const loc = group.location as { distance?: number };
    return loc?.distance ?? null;
  });

  const unreadCount = $derived.by(() => {
    if (!showUnread) return 0;
    const groupWithState = group as { my_state?: { unread_count?: number } };
    if (groupWithState.my_state?.unread_count != null) return groupWithState.my_state.unread_count;
    const groupWithUnread = group as { unread_count?: number };
    return groupWithUnread.unread_count ?? 0;
  });

  const isPrivate = $derived('has_password' in group ? group.has_password : false);
  let joinedOverride = $state<boolean | null>(null);
  let isJoining = $state(false);
  let isReporting = $state(false);
  const isJoined = $derived.by(() => {
    if (joinedOverride != null) return joinedOverride;
    const groupWithState = group as { my_state?: { is_joined?: boolean } };
    return groupWithState.my_state?.is_joined ?? false;
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

  function handleViewDetail(e: MouseEvent) {
    e.preventDefault();
    e.stopPropagation();
    goto(actualHref);
  }

  async function handleJoinGroup(e: MouseEvent) {
    e.preventDefault();
    e.stopPropagation();
    if (!authStore.isAuthenticated) {
      uiStore.openLoginModal();
      return;
    }
    if (isJoining) return;
    if (isJoined) {
      await goto(`/chat/groups/${groupInfo.group_id}`);
      return;
    }
    isJoining = true;
    try {
      const response = await groupsService.join({ group_id: groupInfo.group_id });
      if (!isSuccess(response)) {
        toast.error(response.message || '加入群组失败');
        return;
      }
      if (response.data?.status === 'pending_approval') {
        toast.info('已提交入群申请，等待管理员审核');
        return;
      }
      joinedOverride = true;
      toast.success('加入群组成功');
      await goto(`/chat/groups/${groupInfo.group_id}`);
    } catch {
      toast.error('加入群组失败');
    } finally {
      isJoining = false;
    }
  }

  async function handleShare(e: MouseEvent) {
    e.preventDefault();
    e.stopPropagation();
    const shareData = {
      title: `${groupInfo.name} - HotSpot`,
      text: `加入 ${groupInfo.name} 群组`,
      url: `${window.location.origin}/groups/${groupInfo.group_id}`
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
        target_type: 'group',
        target_id: groupInfo.group_id,
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
  class="neu-card-hover cursor-pointer overflow-hidden rounded-[22px] outline-none transition-[box-shadow,outline,transform] duration-200 hover:scale-[1.008] active:scale-[0.98] focus-visible:ring-2 focus-visible:ring-[var(--neu-glow)] focus-visible:ring-offset-1 focus-visible:ring-offset-[var(--neu-bg)] {className}"
  style="background: var(--neu-bg); box-shadow: 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
  tabindex={0}
  role="link"
  onclick={handleCardClick}
  onkeydown={handleCardKeydown}
>
  {#if variant === 'square'}
    <!-- Square: info-dense card -->
    <div class="p-4">
      <div
        class="rounded-[18px] p-3"
        style="background: var(--neu-bg); box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);"
      >
        <div class="flex items-start justify-between gap-3">
          <div class="flex min-w-0 items-center gap-2.5">
            <div class="relative flex-shrink-0">
              {#if groupInfo.avatar_url}
                <div
                  class="overflow-hidden rounded-full"
                  style="width: 44px; height: 44px; border: 1.5px solid var(--neu-border);"
                >
                  <img
                    src={groupInfo.avatar_url}
                    alt={groupInfo.name}
                    class="h-full w-full object-cover"
                  />
                </div>
              {:else}
                <div
                  class="flex items-center justify-center rounded-full"
                  style="width: 44px; height: 44px; background: var(--neu-fallback-bg); border: 1.5px solid var(--neu-border);"
                >
                  <span style="font-size: 16px; font-weight: 600; color: var(--neu-fallback-text);"
                    >{groupInfo.name.charAt(0)}</span
                  >
                </div>
              {/if}
              {#if unreadCount > 0}
                <div
                  class="absolute -top-1 -right-1 flex items-center justify-center rounded-full"
                  style="width: 18px; height: 18px; background: var(--neu-accent-primary); color: var(--neu-active-text); font-size: 9px; font-weight: 600; box-shadow: 0 2px 6px var(--neu-glow);"
                >
                  {unreadCount > 99 ? '99+' : unreadCount}
                </div>
              {/if}
            </div>
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-1">
                <p style="font-size: 14px; font-weight: 500; color: var(--neu-text);">
                  {groupInfo.name}
                </p>
                {#if isPrivate}
                  <Lock size={12} strokeWidth={1.5} color="var(--neu-text-muted)" />
                {/if}
              </div>
              {#if showMembers}
                <div class="mt-0.5 flex items-center gap-0.5">
                  <Users size={11} strokeWidth={1.5} color="var(--neu-text-muted)" />
                  <span style="font-size: 11px; color: var(--neu-text-muted);"
                    >{groupInfo.members_count}人</span
                  >
                </div>
              {/if}
            </div>
          </div>
          {#if distance}
            <div class="flex flex-shrink-0 items-center gap-0.5">
              <MapPin size={10} strokeWidth={1.5} color="var(--neu-accent-primary)" />
              <span style="font-size: 10px; color: var(--neu-accent-primary);"
                >{formatDistance(distance)}</span
              >
            </div>
          {/if}
        </div>

        {#if 'description' in group}
          {@const groupWithDesc = group as { description?: string }}
          {#if groupWithDesc.description}
            {@const desc = groupWithDesc.description}
            <p
              class="mt-3 line-clamp-3"
              style="font-size: 12px; color: var(--neu-text-muted); line-height: 1.5;"
            >
              {desc}
            </p>
          {:else}
            <p style="font-size: 11px; color: var(--neu-text-muted); margin-top: 10px;">
              群主很懒，还没有添加群组描述
            </p>
          {/if}
        {/if}
      </div>

      {#if 'tags' in group}
        {@const groupWithTags = group as { tags?: string[] }}
        {#if groupWithTags.tags && groupWithTags.tags.length > 0}
          <div class="mt-3 flex flex-wrap gap-1.5">
            {#each groupWithTags.tags.slice(0, 5) as tag (tag)}
              <span
                class="rounded-full px-2 py-0.5"
                style="font-size: 10px; color: var(--neu-accent-primary); background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border);"
                >#{tag}</span
              >
            {/each}
          </div>
        {/if}
      {/if}
    </div>
  {:else if variant === 'compact'}
    <!-- Compact: horizontal list row -->
    <div class="flex items-center gap-3 p-3.5">
      <div class="flex-shrink-0">
        {#if groupInfo.avatar_url}
          <div
            class="overflow-hidden rounded-full"
            style="width: 44px; height: 44px; border: 1.5px solid var(--neu-border);"
          >
            <img
              src={groupInfo.avatar_url}
              alt={groupInfo.name}
              class="h-full w-full object-cover"
            />
          </div>
        {:else}
          <div
            class="flex items-center justify-center rounded-full"
            style="width: 44px; height: 44px; background: var(--neu-fallback-bg); border: 1.5px solid var(--neu-border);"
          >
            <span style="font-size: 16px; font-weight: 600; color: var(--neu-fallback-text);"
              >{groupInfo.name.charAt(0)}</span
            >
          </div>
        {/if}
      </div>
      <div class="min-w-0 flex-1">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-1">
            <p style="font-size: 14px; font-weight: 500; color: var(--neu-text);">
              {groupInfo.name}
            </p>
            {#if isPrivate}
              <Lock size={12} strokeWidth={1.5} color="var(--neu-text-muted)" />
            {/if}
          </div>
          {#if distance}
            <div class="flex items-center gap-0.5">
              <MapPin size={10} strokeWidth={1.5} color="var(--neu-accent-primary)" />
              <span style="font-size: 10px; color: var(--neu-accent-primary);"
                >{formatDistance(distance)}</span
              >
            </div>
          {/if}
        </div>
        {#if showMembers}
          <div class="mt-0.5 flex items-center gap-0.5">
            <Users size={11} strokeWidth={1.5} color="var(--neu-text-muted)" />
            <span style="font-size: 11px; color: var(--neu-text-muted);"
              >{groupInfo.members_count}人</span
            >
          </div>
        {/if}
      </div>
      {#if unreadCount > 0}
        <div
          class="flex flex-shrink-0 items-center justify-center rounded-full"
          style="width: 20px; height: 20px; background: var(--neu-accent-primary); color: var(--neu-active-text); font-size: 11px; font-weight: 600; box-shadow: 0 2px 6px var(--neu-glow);"
        >
          {unreadCount > 99 ? '99+' : unreadCount}
        </div>
      {:else}
        <ChevronRight size={16} strokeWidth={1.5} color="var(--neu-text-muted)" />
      {/if}
    </div>
  {:else}
    <!-- Default: horizontal with actions -->
    <div class="p-4">
      <div class="flex items-center gap-3.5">
        <div class="flex-shrink-0">
          {#if groupInfo.avatar_url}
            <div
              class="overflow-hidden rounded-full"
              style="width: 48px; height: 48px; border: 1.5px solid var(--neu-border);"
            >
              <img
                src={groupInfo.avatar_url}
                alt={groupInfo.name}
                class="h-full w-full object-cover"
              />
            </div>
          {:else}
            <div
              class="flex items-center justify-center rounded-full"
              style="width: 48px; height: 48px; background: var(--neu-fallback-bg); border: 1.5px solid var(--neu-border);"
            >
              <span style="font-size: 18px; font-weight: 600; color: var(--neu-fallback-text);"
                >{groupInfo.name.charAt(0)}</span
              >
            </div>
          {/if}
        </div>
        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-1">
            <p style="font-size: 14px; font-weight: 500; color: var(--neu-text);">
              {groupInfo.name}
            </p>
            {#if isPrivate}
              <Lock size={12} strokeWidth={1.5} color="var(--neu-text-muted)" />
            {/if}
          </div>
          <div class="mt-0.5 flex items-center gap-2">
            {#if showMembers}
              <div class="flex items-center gap-0.5">
                <Users size={11} strokeWidth={1.5} color="var(--neu-text-muted)" />
                <span style="font-size: 11px; color: var(--neu-text-muted);"
                  >{groupInfo.members_count}人</span
                >
              </div>
            {/if}
            {#if distance}
              <div class="flex items-center gap-0.5">
                <MapPin size={10} strokeWidth={1.5} color="var(--neu-accent-primary)" />
                <span style="font-size: 10px; color: var(--neu-accent-primary);"
                  >{formatDistance(distance)}</span
                >
              </div>
            {/if}
          </div>
        </div>
        {#if unreadCount > 0}
          <div
            class="flex flex-shrink-0 items-center justify-center rounded-full"
            style="width: 22px; height: 22px; background: var(--neu-accent-primary); color: var(--neu-active-text); font-size: 11px; font-weight: 600; box-shadow: 0 2px 6px var(--neu-glow);"
          >
            {unreadCount > 99 ? '99+' : unreadCount}
          </div>
        {/if}
      </div>
    </div>
  {/if}

  <!-- Action buttons -->
  <div style="border-top: 1px solid var(--neu-border);" class="px-4 py-2.5">
    <div class="flex items-center justify-between">
      <button
        data-card-action
        class="flex items-center gap-1.5 py-1 outline-none transition-transform duration-150 active:scale-[0.93]"
        style="color: {isJoined ? 'var(--neu-accent-primary)' : 'var(--neu-text-muted)'};"
        onclick={handleJoinGroup}
        disabled={isJoining}
      >
        {#if isJoining}
          <Loader2 size={15} strokeWidth={1.5} class="animate-spin" />
        {:else}
          <UserPlus size={15} strokeWidth={1.5} />
        {/if}
        <span style="font-size: 12px;">
          {#if isJoining}加入中{:else if isJoined}进入群聊{:else}加入群组{/if}
        </span>
      </button>
      {#if showMenu}
        <DropdownMenu>
          <DropdownMenuTrigger>
            <button
              data-card-action
              class="flex items-center justify-center rounded-full outline-none transition-transform duration-150 active:scale-[0.92]"
              style="width: 30px; height: 30px; background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border);"
              onclick={(e: MouseEvent) => {
                e.preventDefault();
                e.stopPropagation();
              }}
              aria-label="更多操作"
            >
              <MoreVertical size={14} strokeWidth={1.5} color="var(--neu-text-muted)" />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" class="w-48">
            <DropdownMenuItem onclick={handleViewDetail}>
              <Info class="mr-2 h-4 w-4" />
              <span>查看详情</span>
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
