<script lang="ts">
  import { dev } from '$app/environment';
  import { goto } from '$app/navigation';
  import { authStore } from '$lib/stores/auth.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { t } from '$lib/i18n';
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
  import Spinner from '$lib/components/common/spinner.svelte';
  import Share2 from 'lucide-svelte/icons/share-2';
  import Flag from 'lucide-svelte/icons/flag';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import Avatar from '$lib/components/common/avatar.svelte';
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
  class="surface-hero card-hover cursor-pointer overflow-hidden rounded-[var(--radius-lg)] outline-none hover:scale-[1.02] active:scale-[0.98] focus-visible:ring-2 focus-visible:ring-[var(--glow)] focus-visible:ring-offset-1 focus-visible:ring-offset-[var(--surface-bg)] {className}"
  tabindex={0}
  role="link"
  onclick={handleCardClick}
  onkeydown={handleCardKeydown}
>
  {#if variant === 'square'}
    <!-- Square: info-dense card -->
    <div class="p-4">
      <div
        class="rounded-[var(--radius-md)] p-3"
        style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
      >
        <div class="flex items-start justify-between gap-3">
          <div class="flex min-w-0 items-center gap-2.5">
            <div class="relative flex-shrink-0">
              <Avatar
                src={groupInfo.avatar_url}
                name={groupInfo.name}
                alt={groupInfo.name}
                size={40}
                type="group"
              />
              {#if unreadCount > 0}
                <div
                  class="text-xs absolute -top-1 -right-1 flex items-center justify-center rounded-full"
                  style="width: 18px; height: 18px; background: var(--brand); color: var(--text-on-accent); font-weight: 600; box-shadow: var(--shadow-brand-btn)"
                >
                  {unreadCount > 99 ? '99+' : unreadCount}
                </div>
              {/if}
            </div>
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-1">
                <p class="text-body-lg" style="font-weight: 500; color: var(--text-primary)">
                  {groupInfo.name}
                </p>
                {#if isPrivate}
                  <Lock size={12} strokeWidth={1.5} color="var(--text-secondary)" />
                {/if}
              </div>
              {#if showMembers}
                <div class="mt-0.5 flex items-center gap-0.5">
                  <Users size={11} strokeWidth={1.5} color="var(--text-secondary)" />
                  <span class="text-caption" style="color: var(--text-secondary)"
                    >{groupInfo.members_count}人</span
                  >
                </div>
              {/if}
            </div>
          </div>
          {#if distance}
            <div class="flex flex-shrink-0 items-center gap-0.5">
              <MapPin size={10} strokeWidth={1.5} color="var(--brand)" />
              <span class="text-xs" style="color: var(--brand)">{formatDistance(distance)}</span>
            </div>
          {/if}
        </div>

        {#if 'description' in group}
          {@const groupWithDesc = group as { description?: string }}
          {#if groupWithDesc.description}
            {@const desc = groupWithDesc.description}
            <p
              class="text-meta mt-3 line-clamp-3"
              style="color: var(--text-secondary); line-height: 1.5"
            >
              {desc}
            </p>
          {:else}
            <p class="text-caption" style="color: var(--text-secondary); margin-top: 10px">
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
              <span class="tag-chip tag-chip--sm">#{tag}</span>
            {/each}
          </div>
        {/if}
      {/if}
    </div>
  {:else if variant === 'compact'}
    <!-- Compact: horizontal list row -->
    <div class="flex items-center gap-3 p-3.5">
      <div class="flex-shrink-0">
        <Avatar
          src={groupInfo.avatar_url}
          name={groupInfo.name}
          alt={groupInfo.name}
          size={40}
          type="group"
        />
      </div>
      <div class="min-w-0 flex-1">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-1">
            <p class="text-body-lg" style="font-weight: 500; color: var(--text-primary)">
              {groupInfo.name}
            </p>
            {#if isPrivate}
              <Lock size={12} strokeWidth={1.5} color="var(--text-secondary)" />
            {/if}
          </div>
          {#if distance}
            <div class="flex items-center gap-0.5">
              <MapPin size={10} strokeWidth={1.5} color="var(--brand)" />
              <span class="text-xs" style="color: var(--brand)">{formatDistance(distance)}</span>
            </div>
          {/if}
        </div>
        {#if showMembers}
          <div class="mt-0.5 flex items-center gap-0.5">
            <Users size={11} strokeWidth={1.5} color="var(--text-secondary)" />
            <span class="text-caption" style="color: var(--text-secondary)"
              >{groupInfo.members_count}人</span
            >
          </div>
        {/if}
      </div>
      {#if unreadCount > 0}
        <div
          class="text-caption flex flex-shrink-0 items-center justify-center rounded-full"
          style="width: 20px; height: 20px; background: var(--brand); color: var(--text-on-accent); font-weight: 600; box-shadow: var(--shadow-brand-btn)"
        >
          {unreadCount > 99 ? '99+' : unreadCount}
        </div>
      {:else}
        <ChevronRight size={16} strokeWidth={1.5} color="var(--text-secondary)" />
      {/if}
    </div>
  {:else}
    <!-- Default: horizontal with actions -->
    <div class="p-4">
      <div class="flex items-center gap-3.5">
        <div class="flex-shrink-0">
          <Avatar
            src={groupInfo.avatar_url}
            name={groupInfo.name}
            alt={groupInfo.name}
            size={48}
            type="group"
          />
        </div>
        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-1">
            <p class="text-body-lg" style="font-weight: 500; color: var(--text-primary)">
              {groupInfo.name}
            </p>
            {#if isPrivate}
              <Lock size={12} strokeWidth={1.5} color="var(--text-secondary)" />
            {/if}
          </div>
          <div class="mt-0.5 flex items-center gap-2">
            {#if showMembers}
              <div class="flex items-center gap-0.5">
                <Users size={11} strokeWidth={1.5} color="var(--text-secondary)" />
                <span class="text-caption" style="color: var(--text-secondary)"
                  >{groupInfo.members_count}人</span
                >
              </div>
            {/if}
            {#if distance}
              <div class="flex items-center gap-0.5">
                <MapPin size={10} strokeWidth={1.5} color="var(--brand)" />
                <span class="text-xs" style="color: var(--brand)">{formatDistance(distance)}</span>
              </div>
            {/if}
          </div>
        </div>
        {#if unreadCount > 0}
          <div
            class="text-caption flex flex-shrink-0 items-center justify-center rounded-full"
            style="width: 22px; height: 22px; background: var(--brand); color: var(--text-on-accent); font-weight: 600; box-shadow: var(--shadow-brand-btn)"
          >
            {unreadCount > 99 ? '99+' : unreadCount}
          </div>
        {/if}
      </div>
    </div>
  {/if}

  <!-- Action buttons -->
  <div style="border-top: 1px solid var(--border-theme);" class="px-4 py-2.5">
    <div class="flex items-center justify-between">
      <button
        data-card-action
        class="flex items-center gap-1.5 py-1 outline-none transition-transform duration-150 active:scale-[0.93]"
        style="color: {isJoined ? 'var(--brand)' : 'var(--text-secondary)'};"
        onclick={handleJoinGroup}
        disabled={isJoining}
      >
        {#if isJoining}
          <Spinner size="sm" strokeWidth={1.5} />
        {:else}
          <UserPlus size={15} strokeWidth={1.5} />
        {/if}
        <span class="text-meta">
          {#if isJoining}加入中{:else if isJoined}进入群聊{:else}加入群组{/if}
        </span>
      </button>
      {#if showMenu}
        <DropdownMenu>
          <DropdownMenuTrigger>
            <button
              data-card-action
              class="surface-flat flex items-center justify-center rounded-full outline-none transition-transform duration-150 active:scale-[0.88]"
              style="width: 30px; height: 30px;"
              onclick={(e: MouseEvent) => {
                e.preventDefault();
                e.stopPropagation();
              }}
              aria-label={t('userProfile.moreAria')}
            >
              <MoreVertical size={14} strokeWidth={1.5} color="var(--text-secondary)" />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" class="w-48">
            <DropdownMenuItem onclick={handleViewDetail}>
              <Info class="mr-2 h-4 w-4" />
              <span>查看详情</span>
            </DropdownMenuItem>
            <DropdownMenuItem onclick={handleShare}>
              <Share2 class="mr-2 h-4 w-4" />
              <span>{t('userProfile.shareAria')}</span>
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
