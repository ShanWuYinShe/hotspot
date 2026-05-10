<script lang="ts">
  import { page } from '$app/state';
  import { untrack } from 'svelte';
  import { withMinimumLoadingTime } from '$lib/utils/loading';
  import { momentsService, commentsService } from '$lib/api/services';
  import { formatDateTime } from '$lib/utils/format';
  import { isSuccess } from '$lib/api/client';
  import { authStore } from '$lib/stores/auth.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { toast } from '$lib/stores/toast.svelte';
  import { goto } from '$app/navigation';
  import { dismissActiveKeyboardFocus, useKeyboardFocus } from '$lib/attachments';
  import type { MomentItem, CommentItem } from '$lib/types/api';
  import Heart from 'lucide-svelte/icons/heart';
  import MessageCircle from 'lucide-svelte/icons/message-circle';
  import Bookmark from 'lucide-svelte/icons/bookmark';
  import Share2 from 'lucide-svelte/icons/share-2';
  import MapPin from 'lucide-svelte/icons/map-pin';
  import ArrowLeft from 'lucide-svelte/icons/arrow-left';
  import Send from 'lucide-svelte/icons/send';
  import Loader2 from 'lucide-svelte/icons/loader-2';
  import Pencil from 'lucide-svelte/icons/pencil';
  import CreateMomentModal from '$lib/components/common/create-moment-modal.svelte';

  const momentId = $derived(page.params.id!);

  let moment = $state<MomentItem | null>(null);
  let comments = $state.raw<CommentItem[]>([]);
  let isLoading = $state(true);
  let isLoadingComments = $state(true);
  let commentsError = $state(false);
  let commentText = $state('');
  let isSubmitting = $state(false);
  let loadVersion = 0;
  let showEditModal = $state(false);
  const isOwner = $derived(
    authStore.isAuthenticated &&
      moment !== null &&
      authStore.currentAccount?.user_id === moment.user.user_id
  );

  // 交互状态
  let isLiked = $state(false);
  let isFavorited = $state(false);
  let likesCount = $state(0);
  let commentsCount = $state(0);

  async function loadMoment(id: string, version: number) {
    try {
      const response = await withMinimumLoadingTime(() =>
        momentsService.getMoment({ moment_id: id })
      );
      if (version !== loadVersion) return; // 丢弃过期响应
      if (isSuccess(response) && response.data) {
        moment = response.data;
        isLiked = moment.my_state?.is_liked ?? false;
        isFavorited = moment.my_state?.is_favorited ?? false;
        likesCount = moment.likes_count;
        commentsCount = moment.comments_count;
      } else {
        toast.error(response.message || '加载失败');
      }
    } catch {
      if (version !== loadVersion) return;
      toast.error('加载失败');
    } finally {
      if (version === loadVersion) isLoading = false;
    }
  }

  function handleEdited() {
    if (momentId) {
      loadVersion++;
      loadMoment(momentId, loadVersion);
    }
  }

  async function loadComments(id: string, version: number) {
    commentsError = false;
    try {
      const response = await commentsService.getComments({
        moment_id: id,
        pagination: { limit: 50 }
      });
      if (version !== loadVersion) return; // 丢弃过期响应
      if (isSuccess(response) && response.data) {
        comments = response.data.items;
      } else {
        commentsError = true;
      }
    } catch {
      if (version !== loadVersion) return;
      commentsError = true;
    } finally {
      if (version === loadVersion) isLoadingComments = false;
    }
  }

  async function handleLike() {
    if (!authStore.isAuthenticated) {
      uiStore.openLoginModalWithRedirect({ onSuccess: handleLike });
      return;
    }

    const wasLiked = isLiked;
    isLiked = !isLiked;
    likesCount += isLiked ? 1 : -1;

    try {
      const response = isLiked
        ? await momentsService.like({ moment_id: momentId })
        : await momentsService.unlike({ moment_id: momentId });

      if (!isSuccess(response)) {
        isLiked = wasLiked;
        likesCount += wasLiked ? 1 : -1;
      }
    } catch {
      isLiked = wasLiked;
      likesCount += wasLiked ? 1 : -1;
    }
  }

  async function handleFavorite() {
    if (!authStore.isAuthenticated) {
      uiStore.openLoginModalWithRedirect({ onSuccess: handleFavorite });
      return;
    }

    const wasFavorited = isFavorited;
    isFavorited = !isFavorited;

    try {
      const response = isFavorited
        ? await momentsService.favorite({ moment_id: momentId })
        : await momentsService.unfavorite({ moment_id: momentId });

      if (!isSuccess(response)) {
        isFavorited = wasFavorited;
      }
    } catch {
      isFavorited = wasFavorited;
    }
  }

  async function submitComment() {
    if (!commentText.trim()) return;
    if (!authStore.isAuthenticated) {
      uiStore.openLoginModalWithRedirect({ onSuccess: submitComment });
      return;
    }

    isSubmitting = true;
    try {
      const response = await commentsService.create({
        moment_id: momentId,
        content: commentText.trim()
      });

      if (isSuccess(response) && response.data) {
        comments = [response.data, ...comments];
        commentsCount++;
        commentText = '';
        toast.success('评论成功');
      } else {
        toast.error(response.message || '评论失败');
      }
    } catch {
      toast.error('评论失败');
    } finally {
      isSubmitting = false;
    }
  }

  function formatTime(dateStr: string): string {
    return formatDateTime(dateStr);
  }

  function handleBackClick() {
    if (typeof window !== 'undefined' && dismissActiveKeyboardFocus(window)) {
      return;
    }
    history.back();
  }

  // momentId 变化时重置状态并重新加载（含首次加载）
  $effect(() => {
    const id = momentId;
    untrack(() => {
      const version = ++loadVersion;
      moment = null;
      comments = [];
      isLoading = true;
      isLoadingComments = true;
      commentsError = false;
      commentText = '';
      isLiked = false;
      isFavorited = false;
      likesCount = 0;
      commentsCount = 0;
      loadMoment(id, version);
      loadComments(id, version);
    });
  });
</script>

<svelte:head>
  <title>{moment?.description?.slice(0, 30) ?? '瞬间详情'} - HotSpot</title>
</svelte:head>

<div class="mx-auto max-w-2xl space-y-6">
  <!-- Header -->
  <div
    class="sticky top-0 z-10 flex items-center gap-4 px-4 py-3"
    style="background-color: var(--neu-bg); border-bottom: 1px solid var(--neu-border);"
  >
    <button
      class="flex items-center justify-center rounded-full outline-none"
      style="width: 36px; height: 36px; background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border);"
      onclick={handleBackClick}
      aria-label="返回"
    >
      <ArrowLeft size={18} color="var(--neu-text)" />
    </button>
    <h1 style="font-size: 16px; color: var(--neu-text);">瞬间详情</h1>
    {#if isOwner}
      <div class="ml-auto">
        <button
          class="flex items-center justify-center rounded-full outline-none"
          style="width: 36px; height: 36px; background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border);"
          onclick={() => (showEditModal = true)}
          aria-label="编辑"
        >
          <Pencil size={16} color="var(--neu-text-muted)" />
        </button>
      </div>
    {/if}
  </div>

  {#if isLoading}
    <div
      class="rounded-[22px] p-6 space-y-4"
      style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);"
    >
      <div class="flex items-center gap-3">
        <div
          class="rounded-full"
          style="width: 48px; height: 48px; background: var(--neu-border);"
        ></div>
        <div class="space-y-2">
          <div
            class="rounded"
            style="width: 96px; height: 14px; background: var(--neu-border);"
          ></div>
          <div
            class="rounded"
            style="width: 64px; height: 12px; background: var(--neu-border);"
          ></div>
        </div>
      </div>
      <div class="rounded" style="width: 100%; height: 80px; background: var(--neu-border);"></div>
      <div class="rounded" style="width: 100%; height: 256px; background: var(--neu-border);"></div>
    </div>
  {:else if moment}
    <div
      class="rounded-[22px] p-6"
      style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);"
    >
      <!-- Author -->
      <a href="/users/{moment.user.user_id}" class="flex items-center gap-3">
        <div
          class="overflow-hidden rounded-full"
          style="width: 44px; height: 44px; border: 2px solid var(--neu-accent-primary);"
        >
          <img
            src={moment.user.avatar_url}
            alt={moment.user.nickname}
            class="h-full w-full object-cover"
          />
        </div>
        <div>
          <p style="font-weight: 500; font-size: 15px; color: var(--neu-text);">
            {moment.user.nickname}
          </p>
          <div
            class="flex items-center gap-2"
            style="font-size: 13px; color: var(--neu-text-muted);"
          >
            <span>{formatTime(moment.created_at)}</span>
            {#if moment.location}
              <span class="flex items-center gap-0.5">
                <MapPin size={12} />
                {moment.location.location_name}
              </span>
            {/if}
          </div>
        </div>
      </a>

      <!-- Content -->
      <div class="mt-4 space-y-4">
        <p class="whitespace-pre-wrap" style="color: var(--neu-text);">{moment.description}</p>

        {#if moment.images && moment.images.length > 0}
          <div class="grid gap-2 {moment.images.length === 1 ? '' : 'grid-cols-2'}">
            {#each moment.images as image, i (image.url)}
              <div class="relative">
                <img
                  src={image.url}
                  alt="图片 {i + 1}"
                  class="w-full rounded-[14px]"
                  style="height: clamp(220px, 38vw, 380px); object-fit: cover;"
                  loading="lazy"
                />
                {#if i === 0 && moment.tags && moment.tags.length > 0}
                  <div
                    class="absolute left-3 top-3 rounded-full px-2.5 py-0.5"
                    style="background: var(--neu-accent-primary); box-shadow: 0 2px 8px var(--neu-glow);"
                  >
                    <span style="font-size: 10px; color: white;">#{moment.tags[0]}</span>
                  </div>
                {/if}
              </div>
            {/each}
          </div>
        {/if}

        {#if moment.tags && moment.tags.length > 1}
          <div class="flex flex-wrap gap-2">
            {#each moment.tags.slice(1) as tag (tag)}
              <span
                class="rounded-full px-2.5 py-1"
                style="font-size: 11px; color: var(--neu-accent-primary); background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border);"
              >
                #{tag}
              </span>
            {/each}
          </div>
        {/if}
      </div>

      <!-- Actions -->
      <div
        class="mt-4 flex items-center justify-between"
        style="border-top: 1px solid var(--neu-border); padding-top: 16px;"
      >
        <div class="flex gap-2">
          <button
            class="flex items-center gap-1.5 rounded-full px-3 py-1.5 outline-none"
            style="background: var(--neu-bg); box-shadow: 3px 3px 6px var(--neu-shadow-dark), -3px -3px 6px var(--neu-shadow-light); border: 1px solid var(--neu-border); color: {isLiked
              ? 'var(--neu-accent-primary)'
              : 'var(--neu-text-muted)'};"
            onclick={handleLike}
          >
            <Heart size={19} class={isLiked ? 'fill-current' : ''} />
            <span style="font-size: 13px;">{likesCount}</span>
          </button>
          <button
            class="flex items-center gap-1.5 rounded-full px-3 py-1.5 outline-none"
            style="background: var(--neu-bg); box-shadow: 3px 3px 6px var(--neu-shadow-dark), -3px -3px 6px var(--neu-shadow-light); border: 1px solid var(--neu-border); color: var(--neu-text-muted);"
          >
            <MessageCircle size={19} />
            <span style="font-size: 13px;">{commentsCount}</span>
          </button>
        </div>
        <div class="flex gap-2">
          <button
            class="flex items-center gap-1.5 rounded-full px-3 py-1.5 outline-none"
            style="background: var(--neu-bg); box-shadow: 3px 3px 6px var(--neu-shadow-dark), -3px -3px 6px var(--neu-shadow-light); border: 1px solid var(--neu-border); color: {isFavorited
              ? 'var(--neu-warning)'
              : 'var(--neu-text-muted)'};"
            onclick={handleFavorite}
          >
            <Bookmark size={19} class={isFavorited ? 'fill-current' : ''} />
          </button>
          <button
            class="flex items-center gap-1.5 rounded-full px-3 py-1.5 outline-none"
            style="background: var(--neu-bg); box-shadow: 3px 3px 6px var(--neu-shadow-dark), -3px -3px 6px var(--neu-shadow-light); border: 1px solid var(--neu-border); color: var(--neu-text-muted);"
          >
            <Share2 size={19} />
          </button>
        </div>
      </div>
    </div>

    <!-- Comment input -->
    {#if authStore.isAuthenticated}
      <div
        class="rounded-[22px] p-4"
        style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);"
      >
        <div class="flex items-center gap-3">
          <div class="overflow-hidden rounded-full" style="width: 34px; height: 34px;">
            <img
              src={authStore.currentAccount?.avatar_url || ''}
              alt=""
              class="h-full w-full object-cover"
            />
          </div>
          <div
            class="neu-input-surface flex flex-1 items-center"
            style="height: 38px; border-radius: 9999px;"
          >
            <input
              type="text"
              {@attach useKeyboardFocus()}
              placeholder="写下你的评论..."
              bind:value={commentText}
              disabled={isSubmitting}
              class="bg-transparent flex-1 outline-none text-sm"
              style="color: var(--neu-text);"
            />
            <button
              class="ml-2 mr-1 flex items-center justify-center rounded-full outline-none"
              style="width: 36px; height: 36px; background: var(--neu-accent-primary); box-shadow: 0 2px 8px var(--neu-glow);"
              onclick={submitComment}
              disabled={!commentText.trim() || isSubmitting}
            >
              {#if isSubmitting}
                <Loader2 size={14} color="var(--neu-active-text)" class="animate-spin" />
              {:else}
                <Send size={14} color="var(--neu-active-text)" />
              {/if}
            </button>
          </div>
        </div>
      </div>
    {/if}

    <!-- Comments -->
    <div
      class="rounded-[22px] p-6"
      style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);"
    >
      <p
        class="uppercase"
        style="font-size: 10px; letter-spacing: 0.14em; color: var(--neu-text-muted); margin-bottom: 16px;"
      >
        评论 · {commentsCount}
      </p>

      {#if isLoadingComments}
        <div class="space-y-4">
          {#each Array(3) as _, i (i)}
            <div class="flex gap-3">
              <div
                class="rounded-full"
                style="width: 34px; height: 34px; background: var(--neu-border);"
              ></div>
              <div class="flex-1 space-y-2">
                <div
                  class="rounded"
                  style="width: 96px; height: 14px; background: var(--neu-border);"
                ></div>
                <div
                  class="rounded"
                  style="width: 100%; height: 14px; background: var(--neu-border);"
                ></div>
              </div>
            </div>
          {/each}
        </div>
      {:else if commentsError}
        <div class="flex flex-col items-center gap-3 py-8 text-center">
          <p style="font-size: 14px; color: var(--neu-text-muted);">加载评论失败</p>
          <button
            class="rounded-full px-4 py-2 font-medium outline-none"
            style="box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);"
            onclick={() => {
              const v = ++loadVersion;
              isLoadingComments = true;
              commentsError = false;
              loadComments(momentId, v);
            }}
          >
            重试
          </button>
        </div>
      {:else if comments.length === 0}
        <p class="py-8 text-center" style="color: var(--neu-text-muted);">暂无评论</p>
      {:else}
        <div class="space-y-4">
          {#each comments as comment (comment.comment_id)}
            <div class="flex gap-3">
              <a href="/users/{comment.user.user_id}">
                <div class="overflow-hidden rounded-full" style="width: 34px; height: 34px;">
                  <img
                    src={comment.user.avatar_url}
                    alt={comment.user.nickname}
                    class="h-full w-full object-cover"
                  />
                </div>
              </a>
              <div class="flex-1">
                <div class="flex items-center gap-2">
                  <a
                    href="/users/{comment.user.user_id}"
                    style="font-weight: 500; font-size: 14px; color: var(--neu-text);"
                  >
                    {comment.user.nickname}
                  </a>
                  <span style="font-size: 12px; color: var(--neu-text-muted);">
                    {formatTime(comment.created_at)}
                  </span>
                </div>
                <p
                  class="mt-1 whitespace-pre-wrap"
                  style="font-size: 14px; color: var(--neu-text);"
                >
                  {comment.content}
                </p>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {:else}
    <div
      class="rounded-[22px] py-12 text-center"
      style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);"
    >
      <p style="color: var(--neu-text-muted);">瞬间不存在或已被删除</p>
      <button
        class="mt-4 rounded-full px-5 py-2.5 font-medium outline-none"
        style="box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);"
        onclick={() => goto('/')}
      >
        返回首页
      </button>
    </div>
  {/if}

  {#if isOwner && moment}
    <CreateMomentModal bind:open={showEditModal} editMoment={moment} onEdited={handleEdited} />
  {/if}
</div>
