<script lang="ts">
  import { dev } from '$app/environment';
  import { goto } from '$app/navigation';
  import { authStore } from '$lib/stores/auth.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
	  import { t } from '$lib/i18n';
	  import { copyShareUrl } from '$lib/utils/share';
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
        toast.error(response.message || t('group.joinFailed'));
        return;
      }
      if (response.data?.status === 'pending_approval') {
        toast.info(t('group.joinPending'));
        return;
      }
      joinedOverride = true;
      toast.success(t('group.joinSuccess'));
      await goto(`/chat/groups/${groupInfo.group_id}`);
    } catch {
      toast.error(t('group.joinFailed'));
    } finally {
      isJoining = false;
    }
  }

  async function handleShare(e: MouseEvent) {
    e.preventDefault();
    e.stopPropagation();
    const shareData = {
      title: t('group.shareTitle', { name: groupInfo.name }),
      text: t('group.shareText', { name: groupInfo.name }),
      url: `${window.location.origin}/groups/${groupInfo.group_id}`
    };
    if (navigator.share) {
      try {
        await navigator.share(shareData);
      } catch (err) {
        if ((err as Error).name !== 'AbortError' && dev) console.error('Share failed:', err);
      }
	    } else {
	      await copyShareUrl(shareData.url);
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
        toast.error(response.message || t('common.reportFailed'));
        return;
      }
      toast.success(t('common.reportSuccess'));
    } catch {
      toast.error(t('common.reportFailed'));
    } finally {
      isReporting = false;
    }
  }
</script>

<div
  class="group-card card-hover cursor-pointer overflow-hidden rounded-[var(--radius-lg)] outline-none hover:scale-[1.02] active:scale-[0.98] focus-visible:ring-2 focus-visible:ring-[var(--glow)] focus-visible:ring-offset-1 focus-visible:ring-offset-[var(--surface-bg)] {className}"
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
              <div class="group-card__avatar">
              <Avatar
                src={groupInfo.avatar_url}
                name={groupInfo.name}
                alt={groupInfo.name}
                size={40}
                type="group"
              />
              </div>
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
                    >{t('chat.membersCount', { count: groupInfo.members_count })}</span
                  >
                </div>
              {/if}
            </div>
          </div>
          {#if distance}
            <div class="group-card__distance flex-shrink-0">
              <MapPin size={10} strokeWidth={1.5} />
              <span>{formatDistance(distance)}</span>
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
              {t('group.descEmpty')}
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
      <div class="group-card__avatar">
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
            <div class="group-card__distance">
              <MapPin size={10} strokeWidth={1.5} />
              <span>{formatDistance(distance)}</span>
            </div>
          {/if}
        </div>
        {#if showMembers}
          <div class="mt-0.5 flex items-center gap-0.5">
            <Users size={11} strokeWidth={1.5} color="var(--text-secondary)" />
            <span class="text-caption" style="color: var(--text-secondary)"
              >{t('chat.membersCount', { count: groupInfo.members_count })}</span
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
        <div class="group-card__avatar">
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
                  >{t('chat.membersCount', { count: groupInfo.members_count })}</span
                >
              </div>
            {/if}
            {#if distance}
              <div class="group-card__distance">
                <MapPin size={10} strokeWidth={1.5} />
                <span>{formatDistance(distance)}</span>
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
        class="group-card__action"
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
          {#if isJoining}{t('group.joining')}{:else if isJoined}{t('group.enterChat')}{:else}{t('group.joinAction')}{/if}
        </span>
      </button>
      {#if showMenu}
        <DropdownMenu>
          <DropdownMenuTrigger>
              <button
                data-card-action
                class="group-card__icon-action"
                onclick={(e: MouseEvent) => {
                  e.preventDefault();
                  e.stopPropagation();
                }}
                aria-label={t('userProfile.moreAria')}
              >
                <MoreVertical size={14} strokeWidth={1.5} />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" class="w-48">
            <DropdownMenuItem onclick={handleViewDetail}>
              <Info class="mr-2 h-4 w-4" />
              <span>{t('common.viewDetail')}</span>
            </DropdownMenuItem>
            <DropdownMenuItem onclick={handleShare}>
              <Share2 class="mr-2 h-4 w-4" />
              <span>{t('userProfile.shareAria')}</span>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem onclick={handleReport} class="text-destructive">
              <Flag class="mr-2 h-4 w-4" />
              <span>{t('common.report')}</span>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      {/if}
    </div>
  </div>
</div>

<style>
  .group-card {
    position: relative;
    overflow: clip;
    border: 1px solid color-mix(in srgb, var(--border-theme) 82%, transparent);
    border-radius: var(--radius-lg);
    background:
      linear-gradient(
        180deg,
        color-mix(in srgb, var(--surface-bg) 96%, white 4%),
        var(--surface-bg)
      ),
      var(--surface-bg);
    box-shadow: none;
    transition:
      border-color 0.22s ease,
      box-shadow 0.22s ease;
  }

  .group-card:hover,
  .group-card:focus-visible {
    border-color: color-mix(in srgb, var(--brand) 38%, var(--border-theme));
    box-shadow: 0 2px 14px color-mix(in srgb, var(--brand) 10%, transparent);
  }

  .group-card__avatar {
    display: block;
    width: fit-content;
    height: fit-content;
    flex: 0 0 auto;
    overflow: clip;
    border: 2px solid color-mix(in srgb, var(--brand) 18%, var(--border-theme));
    border-radius: var(--radius-pill);
    background: var(--surface-bg);
  }

  .group-card__avatar :global(img) {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .group-card__distance {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    border: 1px solid color-mix(in srgb, var(--brand) 24%, var(--border-theme));
    border-radius: var(--radius-pill);
    padding: 4px 8px;
    color: var(--brand);
    font-size: var(--text-caption);
    font-weight: 600;
    line-height: 1;
    background: color-mix(in srgb, var(--brand) 9%, var(--surface-bg));
  }

  .group-card__action,
  .group-card__icon-action {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border: 1px solid transparent;
    border-radius: var(--radius-pill);
    color: var(--text-secondary);
    background: transparent;
    outline: none;
    -webkit-tap-highlight-color: transparent;
    user-select: none;
    cursor: pointer;
    transition:
      transform 0.15s cubic-bezier(0.34, 1.56, 0.64, 1),
      color 0.18s ease,
      background 0.18s ease,
      border-color 0.18s ease;
  }

  .group-card__action {
    gap: 5px;
    min-height: 32px;
    padding: 6px 9px;
    font-size: var(--text-meta);
    font-weight: 650;
  }

  .group-card__icon-action {
    width: 32px;
    height: 32px;
  }

  .group-card__action:hover,
  .group-card__icon-action:hover,
  .group-card__action:focus-visible,
  .group-card__icon-action:focus-visible {
    color: var(--brand);
    border-color: color-mix(in srgb, var(--brand) 20%, var(--border-theme));
    background: color-mix(in srgb, var(--brand) 8%, transparent);
  }

  .group-card__action:active,
  .group-card__icon-action:active {
    transform: scale(0.9);
    background: color-mix(in srgb, var(--brand) 14%, transparent);
  }
</style>
