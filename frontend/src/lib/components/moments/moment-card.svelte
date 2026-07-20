<script lang="ts">
  import { goto } from '$app/navigation';
  import { authStore } from '$lib/stores/auth.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { momentsService, feedService } from '$lib/api/services';
  import { toast } from '$lib/stores/toast.svelte';
  import { useOptimisticToggle } from '$lib/hooks/use-optimistic.svelte';
  import { formatRelativeTime, formatDistance } from '$lib/utils/format';
  import { interactive } from '$lib/actions/interactive';
	  import { t } from '$lib/i18n';
	  import { copyShareUrl } from '$lib/utils/share';
  import type { MomentItem } from '$lib/types/api';
  import Heart from 'lucide-svelte/icons/heart';
  import MessageCircle from 'lucide-svelte/icons/message-circle';
  import Bookmark from 'lucide-svelte/icons/bookmark';
  import Share2 from 'lucide-svelte/icons/share-2';
  import X from 'lucide-svelte/icons/x';
  import Avatar from '$lib/components/common/avatar.svelte';
  import SmartImage from '$lib/components/common/smart-image.svelte';
  import ImagePreview from '$lib/components/common/image-preview.svelte';
  import { CardDistanceChip } from '$lib/components/common/card-primitives';

	  interface Props {
	    moment: MomentItem;
	    class?: string;
	    onDismiss?: () => void;
	    imageFetchpriority?: 'high' | 'low' | 'auto';
	  }

	  let { moment, class: className = '', onDismiss, imageFetchpriority = 'auto' }: Props = $props();

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

  // 图片预览:列表卡只显示首图,但点击可预览全部图。
  const previewImages = $derived(
    (moment.images ?? []).map((img) => ({
      src: img.url,
      alt: moment.description || t('moment.imageAlt')
    }))
  );
  let previewOpen = $state(false);
  let previewIndex = $state(0);

  function openPreview(e: MouseEvent) {
    e.preventDefault();
    e.stopPropagation();
    previewIndex = 0;
    previewOpen = true;
  }
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
  let heartBurstTimer: ReturnType<typeof setTimeout> | null = null;

  $effect(() => {
    return () => {
      if (heartBurstTimer) clearTimeout(heartBurstTimer);
    };
  });

  function handleLike(event: MouseEvent) {
    event.preventDefault();
    event.stopPropagation();
    if (!authStore.isAuthenticated) {
      uiStore.openLoginModal(likeAction.toggle);
      return;
    }
    likeAction.toggle();
    if (likeAction.isOn) {
      heartBurst = true;
      if (heartBurstTimer) clearTimeout(heartBurstTimer);
      heartBurstTimer = setTimeout(() => {
        heartBurst = false;
        heartBurstTimer = null;
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
      title: t('moment.shareTitle', { nickname: moment.user.nickname }),
      text: moment.description.slice(0, 100),
      url: `${window.location.origin}/moments/${moment.moment_id}`
    };
    if (navigator.share) {
      navigator.share(shareData).catch((error) => {
        if (error instanceof DOMException && error.name === 'AbortError') return;
        toast.error(t('moment.shareFailed'));
      });
	    } else {
	      copyShareUrl(shareData.url);
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
      toast.success(t('moment.dismissed'));
      onDismiss?.();
    } catch {
      toast.error(t('common.operationFailed'));
    }
  }
</script>

  <div
  class="moment-card card-surface card-hover hover:scale-[1.015] active:scale-[0.985] focus-visible:ring-2 focus-visible:ring-[var(--glow)] focus-visible:ring-offset-1 focus-visible:ring-offset-[var(--page-bg)] {className}"
  onclick={handleCardClick}
  role="link"
  tabindex={0}
  aria-label={t('moment.viewMomentAria', { nickname: moment.user.nickname })}
  onkeydown={(e) => {
    if ((e.key === 'Enter' || e.key === ' ') && !isInteractiveTarget(e.target)) {
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
      aria-label={t('moment.viewProfileAria', { nickname: moment.user.nickname })}
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
      <CardDistanceChip
        class="moment-card__location"
        iconSize={12}
        maxWidth="min(42%, 150px)"
        text={moment.location.location_name}
      />
    {/if}

    {#if onDismiss}
      <button
        type="button"
        data-card-action
        class="moment-card__dismiss"
        onclick={handleDismiss}
        use:interactive={{ type: 'action' }}
        aria-label={t('moment.dismissAria')}
        title={t('moment.dismissAria')}
      >
        <X size={14} strokeWidth={1.7} />
      </button>
    {/if}
  </header>

  {#if primaryImage}
    <button
      type="button"
      class="moment-card__media"
      onclick={openPreview}
      aria-label={t('moment.previewImageAria')}
      data-card-action
    >
	      <SmartImage
	        src={primaryImage.url}
	        alt={moment.description || moment.tags?.[0] || t('moment.imageAlt')}
	        thumbnail={primaryImage.thumbnail_url}
	        width={primaryImage.width}
	        height={primaryImage.height}
	        sizes="(min-width: 1536px) 25vw, (min-width: 1024px) 33vw, (min-width: 640px) 50vw, 100vw"
	        loading={imageFetchpriority === 'high' ? 'eager' : 'lazy'}
	        fetchpriority={imageFetchpriority}
	      />
      <div class="moment-card__media-shade"></div>
      {#if imageCount > 1}
        <span class="moment-card__image-count">+{imageCount - 1}</span>
      {/if}
    </button>
  {/if}

  <div class="moment-card__body">
    {#if moment.description}
      <p class="moment-card__description">
        {moment.description}
      </p>
    {/if}

    {#if visibleTags.length > 0}
      <div class="moment-card__tags" aria-label={t('moment.tagsAria')}>
        {#each visibleTags as tag (tag)}
          <span class="tag-chip tag-chip--sm">#{tag}</span>
        {/each}
      </div>
    {/if}
  </div>

  <footer class="moment-card__footer">
    <div class="moment-card__actions">
      <button
        type="button"
        data-card-action
        class="card-action"
        class:is-active={likeAction.isOn}
        onclick={handleLike}
        disabled={likeAction.isLoading}
        use:interactive={{ type: 'action' }}
        aria-label={likeAction.isOn ? t('common.unlike') : t('common.like')}
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
        class="card-action"
        onclick={(e) => {
          e.preventDefault();
          e.stopPropagation();
          openMoment();
        }}
        use:interactive={{ type: 'action' }}
        aria-label={t('moment.viewCommentsAria')}
      >
        <MessageCircle size={16} strokeWidth={1.7} />
        <span>{moment.comments_count}</span>
      </button>
    </div>

    <div class="moment-card__actions moment-card__actions--secondary">
      <button
        type="button"
        data-card-action
        class="card-icon-action"
        class:is-active={favoriteAction.isOn}
        onclick={handleFavorite}
        disabled={favoriteAction.isLoading}
        use:interactive={{ type: 'action' }}
        aria-label={favoriteAction.isOn ? t('moment.unsaveAria') : t('moment.saveAria')}
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
        class="card-icon-action"
        onclick={handleShare}
        use:interactive={{ type: 'action' }}
        aria-label={t('common.share')}
      >
        <Share2 size={16} strokeWidth={1.7} />
      </button>
    </div>
  </footer>
</div>

<ImagePreview
  images={previewImages}
  bind:open={previewOpen}
  bind:index={previewIndex}
/>

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
  }

  /* :global 穿透到子组件(Avatar / SmartImage)内的 <img>，否则 scoped CSS 不生效 */
  .moment-card__avatar :global(img) {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  /* media 图片:容器固定 aspect-ratio + overflow:hidden,内层 img 绝对定位
     填充容器不参与布局尺寸计算,避免高图/宽图撑破 aspect-ratio 造成卡片重叠。
     object-fit: cover 处理过宽/过高/过大的图片;hover 放大由 transform scale 实现。 */
  .moment-card__media :global(img) {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    object-fit: cover;
    object-position: center;
    transition: transform 0.35s cubic-bezier(0.22, 1, 0.36, 1);
  }
  @media (hover: hover) {
    .moment-card__media:hover :global(img) {
      transform: scale(1.05);
    }
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

  /* moment-card 的 location chip 在 CardDistanceChip 基础上略加高 padding
     (5px vs 默认 4px),与原设计一致;max-width 已通过 CardDistanceChip 的
     maxWidth prop 传入。 */
  .moment-card__location {
    flex: 0 1 auto;
    padding: 5px 8px;
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
    display: block;
    border: none;
    padding: 0;
    cursor: pointer;
    position: relative;
    margin: 0 var(--card-pad-x);
    min-height: 0;
    overflow: hidden;
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
    font-size: var(--text-body);
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
  /* tag chip 样式复用全局 .tag-chip.tag-chip--sm(app.css),不再本地重写。 */

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

  .card-action.is-active,
  .card-icon-action.is-active {
    color: var(--brand);
    background: color-mix(in srgb, var(--brand) 12%, transparent);
    border-color: color-mix(in srgb, var(--brand) 22%, var(--border-theme));
  }

  .card-action:disabled,
  .card-icon-action:disabled {
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
