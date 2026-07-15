<script lang="ts">
  import { goto } from '$app/navigation';
  import { authStore } from '$lib/stores/auth.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { momentsService, feedService } from '$lib/api/services';
  import { toast } from '$lib/stores/toast.svelte';
  import { useOptimisticToggle } from '$lib/hooks/use-optimistic.svelte';
  import { formatRelativeTime, formatDistance } from '$lib/utils/format';
  import type { MomentItem } from '$lib/types/api';
  import Heart from 'lucide-svelte/icons/heart';
  import MessageCircle from 'lucide-svelte/icons/message-circle';
  import Bookmark from 'lucide-svelte/icons/bookmark';
  import Share2 from 'lucide-svelte/icons/share-2';
  import MapPin from 'lucide-svelte/icons/map-pin';
  import X from 'lucide-svelte/icons/x';
  import Avatar from '$lib/components/common/avatar.svelte';
  import SmartImage from '$lib/components/common/smart-image.svelte';

  interface Props {
    moment: MomentItem;
    class?: string;
    onDismiss?: () => void;
  }

  let { moment, class: className = '', onDismiss }: Props = $props();

  const likeAction = useOptimisticToggle({
    initialValue: () => moment.my_state?.is_liked ?? false,
    initialCount: () => moment.likes_count,
    onTrue: () => momentsService.like({ moment_id: moment.moment_id }),
    onFalse: () => momentsService.unlike({ moment_id: moment.moment_id })
  });

  const favoriteAction = useOptimisticToggle({
    initialValue: () => moment.my_state?.is_favorited ?? false,
    onTrue: () => momentsService.favorite({ moment_id: moment.moment_id }),
    onFalse: () => momentsService.unfavorite({ moment_id: moment.moment_id })
  });

  const primaryImage = $derived(moment.images?.[0]);
  const imageCount = $derived(moment.images?.length ?? 0);
  const visibleTags = $derived((moment.tags ?? []).slice(0, 3));
  const timeText = $derived(
    formatRelativeTime(moment.created_at, undefined, { fallback: 'date' })
  );
  const distanceText = $derived(
    moment.location?.distance != null ? formatDistance(moment.location.distance) : ''
  );

  function isInteractiveTarget(target: EventTarget | null): boolean {
    return target instanceof HTMLElement
      ? Boolean(target.closest('button, a, input, select, textarea, [data-card-action]'))
      : false;
  }

  function openMoment() {
    void goto(`/moments/${moment.moment_id}`);
  }

  function handleCardClick(event: MouseEvent) {
    if (event.defaultPrevented || isInteractiveTarget(event.target)) return;
    openMoment();
  }

  let heartBurst = $state(false);

  function handleLike(event: MouseEvent) {
    event.preventDefault();
    event.stopPropagation();
    if (!authStore.isAuthenticated) {
      uiStore.openLoginModal(likeAction.toggle);
      return;
    }
    likeAction.toggle();
    if (!likeAction.isOn) {
      heartBurst = true;
      setTimeout(() => {
        heartBurst = false;
      }, 300);
    }
  }

  function handleFavorite(event: MouseEvent) {
    event.preventDefault();
    event.stopPropagation();
    if (!authStore.isAuthenticated) {
      uiStore.openLoginModal(favoriteAction.toggle);
      return;
    }
    favoriteAction.toggle();
  }

  function handleShare(event: MouseEvent) {
    event.preventDefault();
    event.stopPropagation();
    const shareData = {
      title: `${moment.user.nickname} 的瞬间`,
      text: moment.description.slice(0, 100),
      url: `${window.location.origin}/moments/${moment.moment_id}`
    };
    if (navigator.share) {
      navigator.share(shareData).catch((error) => {
        if (error instanceof DOMException && error.name === 'AbortError') return;
        toast.error('分享失败');
      });
    } else {
      navigator.clipboard.writeText(shareData.url);
      toast.success('链接已复制');
    }
  }

  function handleAuthorClick(event: MouseEvent) {
    event.preventDefault();
    event.stopPropagation();
    void goto(`/users/${moment.user.user_id}`);
  }

  async function handleDismiss(event: MouseEvent) {
    event.preventDefault();
    event.stopPropagation();
    try {
      await feedService.dismiss({ target_type: 'moment', target_id: moment.moment_id });
      toast.success('已标记为不感兴趣');
      onDismiss?.();
    } catch {
      toast.error('操作失败');
    }
  }
</script>

<div
  class="moment-card {className}"
  onclick={handleCardClick}
  role="link"
  tabindex={0}
  aria-label={`查看 ${moment.user.nickname} 的动态`}
  onkeydown={(e) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      openMoment();
    }
  }}
>
  <header class="moment-card__header">
    <a
      href="/users/{moment.user.user_id}"
      data-card-action
      onclick={handleAuthorClick}
      class="moment-card__avatar"
      aria-label={`查看 ${moment.user.nickname} 的主页`}
    >
      <Avatar
        src={moment.user.avatar_url}
        name={moment.user.nickname}
        alt={moment.user.nickname}
        size={40}
        border={false}
      />
    </a>

    <div class="moment-card__author">
      <a href="/users/{moment.user.user_id}" data-card-action onclick={handleAuthorClick}>
        {moment.user.nickname}
      </a>
      <div class="moment-card__meta">
        <span>{timeText}</span>
        {#if distanceText}
          <span aria-hidden="true">·</span>
          <span>{distanceText}</span>
        {/if}
      </div>
    </div>

    {#if moment.location?.location_name}
      <span class="moment-card__location">
        <MapPin size={12} strokeWidth={1.7} />
        <span>{moment.location.location_name}</span>
      </span>
    {/if}

    {#if onDismiss}
      <button
        type="button"
        data-card-action
        class="moment-card__dismiss"
        onclick={handleDismiss}
        aria-label="不感兴趣"
        title="不感兴趣"
      >
        <X size={14} strokeWidth={1.7} />
      </button>
    {/if}
  </header>

  {#if primaryImage}
    <div class="moment-card__media">
      <SmartImage
        src={primaryImage.url}
        alt={moment.description || moment.tags?.[0] || '动态图片'}
        thumbnail={primaryImage.thumbnail_url}
        width={primaryImage.width}
        height={primaryImage.height}
        sizes="(min-width: 1536px) 25vw, (min-width: 1024px) 33vw, (min-width: 640px) 50vw, 100vw"
      />
      <div class="moment-card__media-shade"></div>
      {#if imageCount > 1}
        <span class="moment-card__image-count">+{imageCount - 1}</span>
      {/if}
    </div>
  {/if}

  <div class="moment-card__body">
    {#if moment.description}
      <p class="moment-card__description">
        {moment.description}
      </p>
    {/if}

    {#if visibleTags.length > 0}
      <div class="moment-card__tags" aria-label="动态标签">
        {#each visibleTags as tag (tag)}
          <span>#{tag}</span>
        {/each}
      </div>
    {/if}
  </div>

  <footer class="moment-card__footer">
    <div class="moment-card__actions">
      <button
        type="button"
        data-card-action
        class="moment-card__action"
        class:is-active={likeAction.isOn}
        onclick={handleLike}
        disabled={likeAction.isLoading}
        aria-label={likeAction.isOn ? '取消点赞' : '点赞'}
        aria-pressed={likeAction.isOn}
      >
        <Heart
          size={16}
          strokeWidth={1.7}
          fill={likeAction.isOn ? 'currentColor' : 'none'}
          class={heartBurst ? 'heart-burst' : ''}
        />
        <span>{likeAction.count}</span>
      </button>

      <button
        type="button"
        data-card-action
        class="moment-card__action"
        onclick={(e) => {
          e.preventDefault();
          e.stopPropagation();
          openMoment();
        }}
        aria-label="查看评论"
      >
        <MessageCircle size={16} strokeWidth={1.7} />
        <span>{moment.comments_count}</span>
      </button>
    </div>

    <div class="moment-card__actions moment-card__actions--secondary">
      <button
        type="button"
        data-card-action
        class="moment-card__icon-action"
        class:is-active={favoriteAction.isOn}
        onclick={handleFavorite}
        disabled={favoriteAction.isLoading}
        aria-label={favoriteAction.isOn ? '取消收藏' : '收藏'}
        aria-pressed={favoriteAction.isOn}
      >
        <Bookmark
          size={16}
          strokeWidth={1.7}
          fill={favoriteAction.isOn ? 'currentColor' : 'none'}
        />
      </button>

      <button
        type="button"
        data-card-action
        class="moment-card__icon-action"
        onclick={handleShare}
        aria-label="分享"
      >
        <Share2 size={16} strokeWidth={1.7} />
      </button>
    </div>
  </footer>
</div>

<style>
  .moment-card {
    --card-pad-x: clamp(14px, 2vw, 18px);
    --card-pad-y: clamp(13px, 1.8vw, 16px);
    --media-ratio: 16 / 10;
    position: relative;
    display: flex;
    min-width: 0;
    cursor: pointer;
    flex-direction: column;
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
    outline: none;
    transition:
      transform 0.22s ease,
      border-color 0.22s ease,
      box-shadow 0.22s ease;
  }

  .moment-card:hover {
    border-color: color-mix(in srgb, var(--brand) 30%, var(--border-theme));
    box-shadow: var(--hover-shadow-card);
  }

  .moment-card:active {
    transform: scale(0.99);
  }

  .moment-card:focus-visible {
    box-shadow:
      0 0 0 2px var(--glow),
      var(--hover-shadow-card);
  }

  .moment-card__header {
    display: flex;
    min-width: 0;
    align-items: center;
    gap: 10px;
    padding: var(--card-pad-y) var(--card-pad-x) 10px;
  }

  .moment-card__avatar {
    display: block;
    width: 38px;
    height: 38px;
    flex: 0 0 auto;
    overflow: clip;
    border: 2px solid color-mix(in srgb, var(--brand) 18%, var(--border-theme));
    border-radius: var(--radius-pill);
    background: var(--surface-bg);
    box-shadow: var(--shadow-sm);
  }

  /* :global 穿透到子组件(Avatar / SmartImage)内的 <img>，否则 scoped CSS 不生效 */
  .moment-card__avatar :global(img),
  .moment-card__media :global(img) {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .moment-card__author {
    min-width: 0;
    flex: 1;
  }

  .moment-card__author a {
    display: block;
    overflow: clip;
    color: var(--text-primary);
    font-size: var(--text-body-lg);
    font-weight: 650;
    line-height: 1.25;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .moment-card__meta {
    display: flex;
    align-items: center;
    gap: 5px;
    margin-top: 3px;
    color: var(--text-secondary);
    font-size: var(--text-caption);
    line-height: 1.2;
  }

  .moment-card__location {
    display: flex;
    max-width: min(42%, 150px);
    flex: 0 1 auto;
    align-items: center;
    gap: 4px;
    overflow: clip;
    border: 1px solid color-mix(in srgb, var(--brand) 24%, var(--border-theme));
    border-radius: var(--radius-pill);
    padding: 5px 8px;
    color: var(--brand);
    font-size: var(--text-caption);
    font-weight: 600;
    line-height: 1;
    background: color-mix(in srgb, var(--brand) 9%, var(--surface-bg));
  }

  .moment-card__location span {
    min-width: 0;
    overflow: clip;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .moment-card__dismiss {
    display: flex;
    flex: 0 0 auto;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    border: 1px solid transparent;
    border-radius: var(--radius-pill);
    color: var(--text-secondary);
    background: transparent;
    outline: none;
    opacity: 0.55;
    transition:
      opacity 0.18s ease,
      transform 0.15s ease,
      color 0.18s ease,
      background 0.18s ease;
  }

  .moment-card__dismiss:hover,
  .moment-card__dismiss:focus-visible {
    opacity: 1;
    color: var(--text-secondary);
    background: color-mix(in srgb, var(--text-secondary) 10%, transparent);
  }

  .moment-card__dismiss:active {
    transform: scale(0.92);
  }

  .moment-card__media {
    position: relative;
    margin: 0 var(--card-pad-x);
    overflow: clip;
    border-radius: var(--radius-md);
    aspect-ratio: var(--media-ratio);
    background: var(--border-theme);
    box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--text-primary) 8%, transparent);
  }

  .moment-card__media-shade {
    position: absolute;
    inset: 0;
    background:
      linear-gradient(to bottom, rgba(0, 0, 0, 0.02), transparent 38%),
      linear-gradient(to top, rgba(0, 0, 0, 0.16), transparent 45%);
    pointer-events: none;
  }

  .moment-card__image-count {
    position: absolute;
    right: 10px;
    bottom: 10px;
    border: 1px solid rgba(255, 255, 255, 0.24);
    border-radius: var(--radius-pill);
    padding: 4px 8px;
    color: var(--text-on-overlay);
    font-size: var(--text-caption);
    font-weight: 700;
    line-height: 1;
    background: rgba(15, 23, 42, 0.58);
    backdrop-filter: blur(10px);
  }

  .moment-card__body {
    display: flex;
    flex: 1;
    flex-direction: column;
    gap: 10px;
    padding: 12px var(--card-pad-x) 8px;
  }

  .moment-card__description {
    display: -webkit-box;
    overflow: clip;
    color: var(--text-primary);
    font-size: clamp(13px, 1.2vw, 14px);
    line-height: 1.65;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 3;
    line-clamp: 3;
  }

  .moment-card__tags {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .moment-card__tags span {
    border: 1px solid color-mix(in srgb, var(--brand) 20%, var(--border-theme));
    border-radius: var(--radius-pill);
    padding: 4px 8px;
    color: var(--brand);
    font-size: var(--text-caption);
    font-weight: 600;
    line-height: 1;
    background: color-mix(in srgb, var(--brand) 7%, transparent);
  }

  .moment-card__footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    border-top: 1px solid color-mix(in srgb, var(--border-theme) 72%, transparent);
    padding: 10px var(--card-pad-x) var(--card-pad-y);
  }

  .moment-card__actions {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
  }

  .moment-card__actions--secondary {
    flex: 0 0 auto;
  }

  .moment-card__action,
  .moment-card__icon-action {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border: 1px solid transparent;
    border-radius: var(--radius-pill);
    color: var(--text-secondary);
    background: transparent;
    outline: none;
    transition:
      transform 0.15s ease,
      color 0.18s ease,
      background 0.18s ease,
      border-color 0.18s ease;
  }

  .moment-card__action {
    gap: 5px;
    min-height: 32px;
    padding: 6px 9px;
    font-size: var(--text-meta);
    font-weight: 650;
  }

  .moment-card__icon-action {
    width: 32px;
    height: 32px;
  }

  .moment-card__action:hover,
  .moment-card__icon-action:hover,
  .moment-card__action:focus-visible,
  .moment-card__icon-action:focus-visible {
    color: var(--brand);
    border-color: color-mix(in srgb, var(--brand) 20%, var(--border-theme));
    background: color-mix(in srgb, var(--brand) 8%, transparent);
  }

  .moment-card__action:active,
  .moment-card__icon-action:active {
    transform: scale(0.94);
  }

  .moment-card__action.is-active,
  .moment-card__icon-action.is-active {
    color: var(--brand);
    background: color-mix(in srgb, var(--brand) 10%, transparent);
  }

  .moment-card__action:disabled,
  .moment-card__icon-action:disabled {
    cursor: wait;
    opacity: 0.62;
  }

  :global(.heart-burst) {
    animation: heartBurst 0.3s ease-out;
  }

  @keyframes heartBurst {
    0% {
      transform: scale(1);
    }
    50% {
      transform: scale(1.3);
    }
    100% {
      transform: scale(1);
    }
  }
</style>
