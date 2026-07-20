<script lang="ts">
  import type { Snippet } from 'svelte';
  import { untrack } from 'svelte';
  import { dev } from '$app/environment';
  import { goto } from '$app/navigation';
  import { authStore } from '$lib/stores/auth.svelte';
	  import { t } from '$lib/i18n';
	  import { copyShareUrl } from '$lib/utils/share';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { toast } from '$lib/stores/toast.svelte';
  import { presenceStore } from '$lib/stores/presence.svelte';
  import { isSuccess } from '$lib/api/client';
  import { reportsService, usersService } from '$lib/api/services';
  import { formatDistance } from '$lib/utils/format';
  import User from 'lucide-svelte/icons/user';
  import MessageCircle from 'lucide-svelte/icons/message-circle';
  import FollowButton from '$lib/components/common/follow-button.svelte';
  import Share2 from 'lucide-svelte/icons/share-2';
  import Flag from 'lucide-svelte/icons/flag';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import type { UserInfo, UserItem } from '$lib/types/api';
  import Avatar from '$lib/components/common/avatar.svelte';
  import { CardDistanceChip, CardMoreMenu, CardActionBar } from '$lib/components/common/card-primitives';

  interface Props {
    user: UserInfo | UserItem;
    showDistance?: boolean;
    showBio?: boolean;
    showTags?: boolean;
    showOnlineStatus?: boolean;
    showMenu?: boolean;
    href?: string;
    class?: string;
    variant?: 'default' | 'compact' | 'detailed' | 'square' | 'nearby-row' | 'member-row';
    /** member-row 专用:角色 badge 文案(如「群主」「管理员」)。 */
    roleLabel?: string;
    /** member-row 专用:副信息行(如「加入时间」)。 */
    subtitle?: string;
    /** member-row 专用:禁言标记文案(红色提示)。 */
    mutedLabel?: string;
    /** member-row 专用:右侧自定义操作区(如 DropdownMenu)。 */
    actions?: Snippet;
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
    variant = 'default',
    roleLabel,
    subtitle,
    mutedLabel,
    actions
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
  // 外部关注状态(派生)+ pending 乐观覆盖(null=回落到外部值)。
  // 旧实现用 $effect 写 isFollowing,父组件在 API 成功后重传同一 user 对象时,
  // effect 会再次覆盖乐观值 → 按钮闪烁回退。改用派生层叠加避免该问题。
  const externalIsFollowing = $derived(
    ('relation' in user ? (user as UserItem).relation : undefined)?.is_following ?? false
  );
  let pendingFollowing = $state<boolean | null>(null);
  // user_id 变化时清除 pending(切换到不同用户卡片)
  $effect(() => {
    const _id = userInfo.user_id;
    untrack(() => {
      pendingFollowing = null;
    });
    void _id;
  });
  const isFollowing = $derived(pendingFollowing !== null ? pendingFollowing : externalIsFollowing);
  let isFollowPending = $state(false);
  let isReporting = $state(false);

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
    const nextFollowing = !isFollowing;
    pendingFollowing = nextFollowing; // 乐观覆盖
    isFollowPending = true;
    try {
      const response = nextFollowing
        ? await usersService.follow({ user_id: userInfo.user_id, source: 'profile' })
        : await usersService.unfollow({ user_id: userInfo.user_id });
      if (!isSuccess(response)) {
        toast.error(response.message || t('common.operationFailed'));
        pendingFollowing = null; // 回退到外部值
        return;
      }
      // 成功:保留 pending 直到外部数据刷新接管(派生会自动反映新外部值;
      // 若父组件未刷新,下次 user_id 切换或 relation 引用变化时也会清掉)
      toast.success(nextFollowing ? t('userProfile.followSuccess') : t('userProfile.unfollowed'));
    } catch {
      pendingFollowing = null; // 回退
      toast.error(nextFollowing ? t('user.followFailed') : t('user.unfollowFailed'));
    } finally {
      isFollowPending = false;
    }
  }

  async function handleShare(e: MouseEvent) {
    e.preventDefault();
    e.stopPropagation();
    const shareData = {
      title: t('user.shareTitle', { nickname: userInfo.nickname }),
      text: t('user.shareText', { nickname: userInfo.nickname }),
      url: `${window.location.origin}/users/${userInfo.user_id}`
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
        target_type: 'user',
        target_id: userInfo.user_id,
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
  class="user-card card-surface card-hover cursor-pointer outline-none focus-visible:ring-2 focus-visible:ring-[var(--glow)] focus-visible:ring-offset-1 focus-visible:ring-offset-[var(--surface-bg)] {className}"
  class:hover:scale-[1.015]={variant !== 'nearby-row' && variant !== 'member-row'}
  class:active:scale-[0.985]={variant !== 'nearby-row' && variant !== 'member-row'}
  class:user-card--row={variant === 'nearby-row' || variant === 'member-row'}
  tabindex={0}
  role="link"
  onclick={handleCardClick}
  onkeydown={handleCardKeydown}
>
  {#if variant === 'nearby-row'}
    <!-- Nearby row: horizontal row for nearby all page -->
    <div class="flex items-center gap-3.5" style="padding: var(--card-pad-y) var(--card-pad-x)">
      <Avatar
        src={userInfo.avatar_url}
        name={userInfo.nickname}
        alt={userInfo.nickname}
        size={48}
        border="brand"
        online={isOnline}
        onlineSize="md"
      />
      <div class="min-w-0 flex-1">
        <p class="text-body-lg truncate min-w-0" style="color: var(--text-primary)">
          {userInfo.nickname}
        </p>
        <div class="mt-0.5 flex items-center gap-1.5">
          <span
            class="text-caption"
            style="color: {isOnline ? 'var(--semantic-online)' : 'var(--text-secondary)'}"
          >
            {isOnline ? t('common.online') : t('user.offline')}
          </span>
          {#if displayTags.length > 0}
            <span class="text-xs" style="color: var(--text-secondary); opacity: 0.5">·</span>
            <span class="text-caption" style="color: var(--brand); opacity: 0.8"
              >#{displayTags[0]}</span
            >
          {/if}
        </div>
      </div>
      <div class="flex flex-shrink-0 flex-col items-end gap-2">
        {#if distance}
          <CardDistanceChip text={formatDistance(distance)} />
        {/if}
        {#if !isMe}
          <FollowButton
            variant="pill"
            following={isFollowing}
            loading={isFollowPending}
            onclick={handleFollow}
          />
        {/if}
      </div>
    </div>
  {:else if variant === 'member-row'}
    <!-- Member row: 用户列表行(粉丝/关注/群成员统一) -->
    <div class="flex items-center gap-3" style="padding: var(--card-pad-y) var(--card-pad-x)">
      <Avatar
        src={userInfo.avatar_url}
        name={userInfo.nickname}
        alt={userInfo.nickname}
        size={48}
        border="brand"
        online={showOnlineStatus && isOnline}
        onlineSize="md"
      />
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2">
          <p class="text-body-lg truncate" style="font-weight: 650; color: var(--text-primary)">
            {userInfo.nickname}
          </p>
          {#if roleLabel}
            <span class="tag-chip tag-chip--sm" style="color: var(--brand);">{roleLabel}</span>
          {/if}
          {#if mutedLabel}
            <span class="text-xs" style="color: var(--theme-error); opacity: 0.8;">{mutedLabel}</span>
          {/if}
        </div>
        {#if subtitle}
          <p class="text-meta" style="color: var(--text-secondary)">{subtitle}</p>
        {/if}
      </div>
      {#if actions}
        <div class="flex flex-shrink-0 items-center gap-2" data-card-action>
          {@render actions()}
        </div>
      {/if}
    </div>
  {:else if variant === 'square'}
    <!-- Square: info-dense card -->
    <div class="p-4">
      <div
        class="surface-pressed rounded-[var(--radius-md)] p-3"
      >
        <div class="flex items-start justify-between gap-3">
          <div class="flex min-w-0 items-center gap-2.5">
            <Avatar
              src={userInfo.avatar_url}
              name={userInfo.nickname}
              alt={userInfo.nickname}
              size={48}
              border="brand"
              online={isOnline}
              onlineSize="md"
            />
            <div class="min-w-0 flex-1">
              <p class="text-body-lg" style="font-weight: 650; color: var(--text-primary)">
                {userInfo.nickname}
              </p>
              <p class="text-caption" style="color: var(--text-secondary); margin-top: 1px">
                {isOnline ? t('common.online') : t('user.offline')}
              </p>
            </div>
          </div>
          {#if distance}
            <CardDistanceChip class="flex-shrink-0" text={formatDistance(distance)} />
          {/if}
        </div>

        {#if showBio && 'bio' in user && user.bio}
          <p
            class="text-meta mt-3 line-clamp-3"
            style="color: var(--text-secondary); line-height: 1.65"
          >
            {user.bio}
          </p>
        {:else if showBio}
          <p class="text-caption" style="color: var(--text-secondary); margin-top: 10px">
            {t('user.bioEmpty')}
          </p>
        {/if}
      </div>

      {#if displayTags.length > 0}
        <div class="mt-3 flex flex-wrap gap-1.5">
          {#each displayTags as tag (tag)}
            <span class="tag-chip tag-chip--sm">#{tag}</span>
          {/each}
        </div>
      {/if}
    </div>
  {:else if variant === 'compact'}
    <!-- Compact: horizontal list row -->
    <div class="flex items-center gap-3 p-3.5">
      <Avatar
        src={userInfo.avatar_url}
        name={userInfo.nickname}
        alt={userInfo.nickname}
        size={40}
        border="brand"
        online={isOnline}
        onlineSize="sm"
      />
      <div class="min-w-0 flex-1">
        <div class="flex items-center justify-between">
          <p class="text-body-lg" style="font-weight: 650; color: var(--text-primary)">
            {userInfo.nickname}
          </p>
          {#if distance}
            <CardDistanceChip text={formatDistance(distance)} />
          {/if}
        </div>
        {#if showBio && 'bio' in user && user.bio}
          <p class="text-meta mt-0.5 truncate" style="color: var(--text-secondary)">
            {user.bio}
          </p>
        {/if}
      </div>
      <ChevronRight size={16} strokeWidth={1.5} color="var(--text-secondary)" />
    </div>
  {:else if variant === 'detailed'}
    <!-- Detailed: vertical with more info -->
    <div style="padding: var(--card-pad-y) var(--card-pad-x)">
      <div class="flex items-center gap-4">
        <Avatar
          src={userInfo.avatar_url}
          name={userInfo.nickname}
          alt={userInfo.nickname}
          size={48}
          border="brand"
          online={isOnline}
          onlineSize="md"
        />
        <div class="min-w-0 flex-1">
          <p class="text-title" style="font-weight: 650; color: var(--text-primary)">
            {userInfo.nickname}
          </p>
          {#if distance}
            <CardDistanceChip class="mt-1" iconSize={12} text={formatDistance(distance)} />
          {/if}
        </div>
      </div>
      {#if showBio && 'bio' in user && user.bio}
        <p class="text-body mt-3" style="color: var(--text-secondary); line-height: 1.65">
          {user.bio}
        </p>
      {/if}
      {#if displayTags.length > 0}
        <div class="mt-3 flex flex-wrap gap-1.5">
          {#each displayTags as tag (tag)}
            <span class="tag-chip tag-chip--sm">#{tag}</span>
          {/each}
        </div>
      {/if}
    </div>
  {:else}
    <!-- Default: balanced horizontal -->
    <div style="padding: var(--card-pad-y) var(--card-pad-x)">
      <div class="flex items-center gap-3.5">
        <Avatar
          src={userInfo.avatar_url}
          name={userInfo.nickname}
          alt={userInfo.nickname}
          size={40}
          border="brand"
          online={isOnline}
          onlineSize="sm"
        />
        <div class="min-w-0 flex-1">
          <p class="text-body-lg" style="font-weight: 650; color: var(--text-primary)">
            {userInfo.nickname}
          </p>
          {#if distance}
            <CardDistanceChip class="mt-0.5" text={formatDistance(distance)} />
          {/if}
        </div>
        <ChevronRight size={16} strokeWidth={1.5} color="var(--text-secondary)" />
      </div>
      {#if showBio && 'bio' in user && user.bio}
        <p
          class="text-meta mt-2.5 line-clamp-2"
          style="color: var(--text-secondary); line-height: 1.65"
        >
          {user.bio}
        </p>
      {/if}
      {#if displayTags.length > 0}
        <div class="mt-2.5 flex flex-wrap gap-1.5">
          {#each displayTags as tag (tag)}
            <span class="tag-chip tag-chip--sm">#{tag}</span>
          {/each}
        </div>
      {/if}
    </div>
  {/if}

  <!-- Action buttons -->
  {#if !isMe && variant !== 'nearby-row' && variant !== 'member-row'}
    <CardActionBar>
      {#snippet primary()}
        <button data-card-action class="card-action" onclick={handleSendMessage}>
          <MessageCircle size={15} strokeWidth={1.5} />
          <span>{t('user.messageAction')}</span>
        </button>
      {/snippet}
      {#snippet secondary()}
        <FollowButton
          variant="ghost"
          following={isFollowing}
          loading={isFollowPending}
          onclick={handleFollow}
          class="card-action"
        />
        <button data-card-action class="card-icon-action" onclick={handleShare} aria-label={t('common.share')}>
          <Share2 size={16} strokeWidth={1.7} />
        </button>
        {#if showMenu}
          <CardMoreMenu
            ariaLabel={t('userProfile.moreAria')}
            items={[
              { icon: User, label: t('userProfile.viewProfile'), onclick: handleViewProfile },
              { icon: Flag, label: t('common.report'), onclick: handleReport, destructive: true }
            ]}
          />
        {/if}
      {/snippet}
    </CardActionBar>
  {/if}
</div>

<style>
  .user-card {
    --card-pad-x: clamp(14px, 2vw, 18px);
    --card-pad-y: clamp(13px, 1.8vw, 16px);
    position: relative;
    display: flex;
    min-width: 0;
    flex-direction: column;
  }

  /* 列表行变体(nearby-row / member-row)行高矮,更小圆角改善矮行的视觉比例。 */
  .user-card--row {
    border-radius: var(--radius-sm);
  }
</style>
