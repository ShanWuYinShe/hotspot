<script lang="ts">
  import { page } from '$app/state';
  import { untrack } from 'svelte';
  import { t } from '$lib/i18n';
  import { withMinimumLoadingTime } from '$lib/utils/loading';
  import { momentsService, commentsService, shareService } from '$lib/api/services';
  import { formatRelativeTime } from '$lib/utils/format';
  import { isSuccess } from '$lib/api/client';
  import { authStore } from '$lib/stores/auth.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { toast } from '$lib/stores/toast.svelte';
  import { configStore } from '$lib/stores/config.svelte';
  import { goto } from '$app/navigation';
  import { dismissActiveKeyboardFocus } from '$lib/attachments';
  import type { MomentItem, CommentItem } from '$lib/types/api';
  import Heart from 'lucide-svelte/icons/heart';
  import MessageCircle from 'lucide-svelte/icons/message-circle';
  import Bookmark from 'lucide-svelte/icons/bookmark';
  import Share2 from 'lucide-svelte/icons/share-2';
  import MapPin from 'lucide-svelte/icons/map-pin';
  import BackButton from '$lib/components/common/back-button.svelte';
  import Avatar from '$lib/components/common/avatar.svelte';
  import SmartImage from '$lib/components/common/smart-image.svelte';
  import ImagePreview from '$lib/components/common/image-preview.svelte';
  import Send from 'lucide-svelte/icons/send';
  import Spinner from '$lib/components/common/spinner.svelte';
  import LogIn from 'lucide-svelte/icons/log-in';
  import Pencil from 'lucide-svelte/icons/pencil';
  import Trash2 from 'lucide-svelte/icons/trash-2';
  import CornerDownRight from 'lucide-svelte/icons/corner-down-right';
  import { Textarea } from '$lib/components/ui/textarea';
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
  let imagePreviewOpen = $state(false);
  let imagePreviewIndex = $state(0);
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
        toast.error(response.message || t('momentDetail.loadFailed'));
      }
    } catch {
      if (version !== loadVersion) return;
      toast.error(t('momentDetail.loadFailed'));
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
        toast.success(t('momentDetail.commentSuccess'));
      } else {
        toast.error(response.message || t('momentDetail.commentFailed'));
      }
    } catch {
      toast.error(t('momentDetail.commentFailed'));
    } finally {
      isSubmitting = false;
    }
  }

  // ── 评论操作:点赞(乐观更新)/ 回复 / 删除 ───────────────────────
  const commentMaxLength = $derived(configStore.commentMaxLength);

  /** 当前展开回复输入框的评论 id(同一时间只展开一条)。 */
  let replyingTo = $state<string | null>(null);
  let replyText = $state('');
  let isSubmittingReply = $state(false);
  /** 进行中删除的评论 id 集合。 */
  let deletingComments = $state<Set<string>>(new Set());

  async function toggleCommentLike(comment: CommentItem) {
    if (!authStore.isAuthenticated) {
      uiStore.openLoginModalWithRedirect({ onSuccess: () => toggleCommentLike(comment) });
      return;
    }
    const wasLiked = comment.is_liked ?? false;
    // 乐观更新
    comment.is_liked = !wasLiked;
    comment.likes_count += wasLiked ? -1 : 1;
    comments = [...comments];
    try {
      const response = wasLiked
        ? await commentsService.unlike({ comment_id: comment.comment_id })
        : await commentsService.like({ comment_id: comment.comment_id });
      if (!isSuccess(response)) {
        // 回滚
        comment.is_liked = wasLiked;
        comment.likes_count += wasLiked ? 1 : -1;
        comments = [...comments];
        toast.error(response.message || t('common.operationFailed'));
      }
    } catch {
      comment.is_liked = wasLiked;
      comment.likes_count += wasLiked ? 1 : -1;
      comments = [...comments];
      toast.error(t('common.operationFailed'));
    }
  }

  function startReply(comment: CommentItem) {
    if (!authStore.isAuthenticated) {
      uiStore.openLoginModalWithRedirect({ onSuccess: () => startReply(comment) });
      return;
    }
    replyingTo = replyingTo === comment.comment_id ? null : comment.comment_id;
    replyText = '';
  }

  async function submitReply(parentComment: CommentItem) {
    if (!replyText.trim()) return;
    isSubmittingReply = true;
    try {
      const response = await commentsService.create({
        moment_id: momentId,
        content: replyText.trim(),
        parent_id: parentComment.comment_id
      });
      if (isSuccess(response) && response.data) {
        // 追加到 top_replies 预览(若无则新建)
        const updated = { ...parentComment };
        updated.top_replies = [...(updated.top_replies ?? []), response.data];
        updated.replies_count += 1;
        comments = comments.map((c) =>
          c.comment_id === parentComment.comment_id ? updated : c
        );
        replyText = '';
        replyingTo = null;
        toast.success(t('momentDetail.replySuccess'));
      } else {
        toast.error(response.message || t('momentDetail.replyFailed'));
      }
    } catch {
      toast.error(t('momentDetail.replyFailed'));
    } finally {
      isSubmittingReply = false;
    }
  }

  async function deleteComment(comment: CommentItem) {
    if (deletingComments.has(comment.comment_id)) return;
    deletingComments = new Set([...deletingComments, comment.comment_id]);
    try {
      const response = await commentsService.delete({ comment_id: comment.comment_id });
      if (isSuccess(response)) {
        comments = comments.filter((c) => c.comment_id !== comment.comment_id);
        commentsCount = Math.max(0, commentsCount - 1);
        toast.success(t('momentDetail.commentDeleted'));
      } else {
        toast.error(response.message || t('momentDetail.deleteFailed'));
      }
    } catch {
      toast.error(t('momentDetail.deleteFailed'));
    } finally {
      const next = new Set(deletingComments);
      next.delete(comment.comment_id);
      deletingComments = next;
    }
  }

  async function handleShare() {
    if (!authStore.isAuthenticated) {
      uiStore.openLoginModalWithRedirect({ onSuccess: handleShare });
      return;
    }

    try {
      const response = await shareService.create({
        share_type: 'moment',
        payload: { shared_moment_id: momentId }
      });
      if (isSuccess(response) && response.data) {
        const url = `${window.location.origin}/share/${response.data.share_code}`;
        await navigator.clipboard.writeText(url);
        toast.success(t('momentDetail.shareCopied'));
      } else {
        toast.error(response.message || t('momentDetail.shareFailed'));
      }
    } catch {
      toast.error(t('momentDetail.shareFailed'));
    }
  }

  async function loadAllReplies(commentId: string) {
    try {
      const response = await commentsService.getReplies({ comment_id: commentId });
      if (isSuccess(response) && response.data) {
        const replies = response.data.items;
        comments = comments.map((c) =>
          c.comment_id === commentId ? { ...c, top_replies: replies, replies_count: replies.length } : c
        );
      } else {
        toast.error(response.message || t('momentDetail.repliesLoadFailed'));
      }
    } catch {
      toast.error(t('momentDetail.repliesLoadFailed'));
    }
  }

  function formatTime(dateStr: string): string {
    return formatRelativeTime(dateStr, undefined, { fallback: 'date' });
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
  <title>{moment?.description?.slice(0, 30) ?? t('momentDetail.pageTitle')} - HotSpot</title>
</svelte:head>

<!-- Header — full-bleed sticky, content aligned to the same narrow shell as the body -->
<div
  class="sticky top-0 z-10 page-pad-x pt-3 pb-3"
  style="background-color: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
>
  <div class="page-shell flex items-center gap-4">
    <BackButton onclick={handleBackClick} size={36} />
    <h1 class="text-h1" style="font-weight: 600; color: var(--text-primary)">{t('momentDetail.pageTitle')}</h1>
    {#if isOwner}
      <div class="ml-auto">
        <button
          class="surface-flat flex items-center justify-center rounded-full outline-none transition-transform duration-150 active:scale-[0.88]"
          style="width: 36px; height: 36px;"
          onclick={() => (showEditModal = true)}
          aria-label={t('momentDetail.editAria')}
        >
          <Pencil size={15} strokeWidth={1.5} color="var(--text-secondary)" />
        </button>
      </div>
    {/if}
  </div>
</div>

<div class="page-pad">
  <div class="page-shell space-y-6">
    {#if isLoading}
      <div
        class="rounded-[var(--radius-lg)] p-6 space-y-4"
        style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
      >
        <div class="flex items-center gap-3">
          <div
            class="rounded-full"
            style="width: 48px; height: 48px; background: var(--border-theme);"
          ></div>
          <div class="space-y-2">
            <div
              class="rounded"
              style="width: 96px; height: 14px; background: var(--border-theme);"
            ></div>
            <div
              class="rounded"
              style="width: 64px; height: 12px; background: var(--border-theme);"
            ></div>
          </div>
        </div>
        <div
          class="rounded"
          style="width: 100%; height: 80px; background: var(--border-theme);"
        ></div>
        <div
          class="rounded"
          style="width: 100%; height: 256px; background: var(--border-theme);"
        ></div>
      </div>
    {:else if moment}
      <div
        class="rounded-[var(--radius-lg)] p-6"
        style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
      >
        <!-- Author -->
        <a href="/users/{moment.user.user_id}" class="flex items-center gap-3">
          <Avatar
            src={moment.user.avatar_url}
            name={moment.user.nickname}
            alt={moment.user.nickname}
            size={40}
          />
          <div>
            <p class="text-body-lg" style="font-weight: 500; color: var(--text-primary)">
              {moment.user.nickname}
            </p>
            <div class="text-body flex items-center gap-2" style="color: var(--text-secondary)">
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
          <p class="whitespace-pre-wrap" style="color: var(--text-primary);">
            {moment.description}
          </p>

          {#if moment.images && moment.images.length > 0}
            <div class="grid gap-2 {moment.images.length === 1 ? '' : 'grid-cols-2'}">
              {#each moment.images as image, i (image.url)}
                <button
                  type="button"
                  class="relative block w-full border-none bg-transparent p-0"
                  onclick={() => {
                    imagePreviewIndex = i;
                    imagePreviewOpen = true;
                  }}
                  aria-label={t('momentDetail.imageAlt', { index: i + 1 })}
                >
                  <SmartImage
                    src={image.url}
                    alt={t('momentDetail.imageAlt', { index: i + 1 })}
                    thumbnail={image.thumbnail_url}
                    width={image.width}
                    height={image.height}
                    sizes={moment.images.length === 1
                      ? '(min-width: 720px) 720px, 100vw'
                      : '(min-width: 768px) 360px, 50vw'}
                    class="w-full rounded-[var(--radius-md)] h-[clamp(220px,38vw,380px)] object-cover"
                  />
                  {#if i === 0 && moment.tags && moment.tags.length > 0}
                    <div
                      class="absolute left-3 top-3 rounded-full px-2.5 py-0.5"
                      style="background: var(--brand); box-shadow: var(--shadow-brand-btn);"
                    >
                      <span class="text-xs" style="color: var(--text-on-accent)"
                        >#{moment.tags[0]}</span
                      >
                    </div>
                  {/if}
                </button>
              {/each}
            </div>
          {/if}

          {#if moment.tags && moment.tags.length > 1}
            <div class="flex flex-wrap gap-2">
              {#each moment.tags.slice(1) as tag (tag)}
                <span
                  class="text-caption rounded-full px-2.5 py-1"
                  style="color: var(--brand); background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme)"
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
          style="border-top: 1px solid var(--border-theme); padding-top: 16px;"
        >
          <div class="flex gap-2">
            <button
              class="surface-flat flex items-center gap-1.5 rounded-full px-3 py-1.5 outline-none"
              style="color: {isLiked ? 'var(--brand)' : 'var(--text-secondary)'};"
              onclick={handleLike}
            >
              <Heart size={19} class={isLiked ? 'fill-current' : ''} />
              <span class="text-body">{likesCount}</span>
            </button>
            <button
              class="surface-flat flex items-center gap-1.5 rounded-full px-3 py-1.5 outline-none"
              style="color: var(--text-secondary);"
            >
              <MessageCircle size={19} />
              <span class="text-body">{commentsCount}</span>
            </button>
          </div>
          <div class="flex gap-2">
            <button
              class="surface-flat flex items-center gap-1.5 rounded-full px-3 py-1.5 outline-none"
              style="color: {isFavorited ? 'var(--brand)' : 'var(--text-secondary)'};"
              onclick={handleFavorite}
              aria-pressed={isFavorited}
              aria-label={isFavorited ? t('momentDetail.unsave') : t('momentDetail.save')}
            >
              <Bookmark size={19} fill={isFavorited ? 'currentColor' : 'none'} />
            </button>
            <button
              class="surface-flat flex items-center gap-1.5 rounded-full px-3 py-1.5 outline-none"
              style="color: var(--text-secondary);"
              onclick={handleShare}
            >
              <Share2 size={19} />
            </button>
          </div>
        </div>
      </div>

      <!-- Comments section（输入框 + 列表合并为单个卡片） -->
      <div
        class="rounded-[var(--radius-lg)] p-6"
        style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
      >
        <!-- Comment input -->
        {#if authStore.isAuthenticated}
          <div class="flex items-center gap-3 pb-4" style="border-bottom: 1px solid var(--border-theme);">
            <Avatar
              src={authStore.currentAccount?.avatar_url}
              name={authStore.currentAccount?.nickname}
              alt=""
              size={32}
            />
            <Textarea
              size="sm"
              variant="transparent"
              class="flex-1 text-sm"
              placeholder={t('momentDetail.commentPlaceholder')}
              maxlength={commentMaxLength}
              bind:value={commentText}
              disabled={isSubmitting}
              onkeydown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault();
                  submitComment();
                }
              }}
            />
            <button
              class="flex flex-shrink-0 items-center justify-center rounded-full outline-none transition-transform active:scale-90"
              style="width: 30px; height: 30px; background: var(--brand); box-shadow: var(--shadow-brand-btn);"
              onclick={submitComment}
              disabled={!commentText.trim() || isSubmitting}
              aria-label={t('common.send')}
            >
              {#if isSubmitting}
                <Spinner size="sm" color="var(--text-on-accent)" />
              {:else}
                <Send size={14} color="var(--text-on-accent)" />
              {/if}
            </button>
          </div>
          {#if commentText.length > commentMaxLength * 0.85}
            <span
              class="text-caption self-end"
              style="color: {commentText.length > commentMaxLength
                ? 'var(--theme-error)'
                : 'var(--brand)'};"
            >
              {commentText.length}/{commentMaxLength}
            </span>
          {/if}
        {:else}
          <!-- 未登录：引导登录后评论 -->
          <button
            type="button"
            class="flex w-full items-center gap-3 pb-4 text-left transition-colors"
            style="border-bottom: 1px solid var(--border-theme);"
            onclick={() => uiStore.openLoginModalWithRedirect({ onSuccess: submitComment })}
          >
            <span
              class="flex flex-shrink-0 items-center justify-center rounded-full"
              style="width: 32px; height: 32px; background: var(--brand); color: var(--text-on-accent);"
            >
              <LogIn size={16} />
            </span>
            <span class="text-sm" style="color: var(--text-secondary);">{t('momentDetail.loginToComment')}</span>
          </button>
        {/if}

        <p
          class="text-xs uppercase"
          style="letter-spacing: 0.14em; color: var(--text-secondary); margin-top: 16px; margin-bottom: 16px"
        >
          {t('momentDetail.commentsTitle', { count: commentsCount })}
        </p>

        {#if isLoadingComments}
          <div class="space-y-4">
            {#each Array(3) as _, i (i)}
              <div class="flex gap-3">
                <div
                  class="rounded-full"
                  style="width: 34px; height: 34px; background: var(--border-theme);"
                ></div>
                <div class="flex-1 space-y-2">
                  <div
                    class="rounded"
                    style="width: 96px; height: 14px; background: var(--border-theme);"
                  ></div>
                  <div
                    class="rounded"
                    style="width: 100%; height: 14px; background: var(--border-theme);"
                  ></div>
                </div>
              </div>
            {/each}
          </div>
        {:else if commentsError}
          <div class="flex flex-col items-center gap-3 py-8 text-center">
            <p class="text-body-lg" style="color: var(--text-secondary)">{t('momentDetail.commentsLoadFailed')}</p>
            <button
              class="rounded-full px-4 py-2 font-medium outline-none"
              style="box-shadow: var(--shadow-md); border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
              onclick={() => {
                const v = ++loadVersion;
                isLoadingComments = true;
                commentsError = false;
                loadComments(momentId, v);
              }}
            >
              {t('common.retry')}
            </button>
          </div>
        {:else if comments.length === 0}
          <p class="py-8 text-center" style="color: var(--text-secondary);">{t('momentDetail.noComments')}</p>
        {:else}
          <div class="space-y-4">
            {#each comments as comment (comment.comment_id)}
              <div class="flex gap-3">
                <a href="/users/{comment.user.user_id}">
                  <Avatar
                    src={comment.user.avatar_url}
                    name={comment.user.nickname}
                    alt={comment.user.nickname}
                    size={32}
                  />
                </a>
                <div class="flex-1">
                  <div class="flex items-center gap-2">
                    <a
                      href="/users/{comment.user.user_id}"
                      class="text-body-lg"
                      style="font-weight: 500; color: var(--text-primary)"
                    >
                      {comment.user.nickname}
                    </a>
                    <span class="text-meta" style="color: var(--text-secondary)">
                      {formatTime(comment.created_at)}
                    </span>
                  </div>
                  <p
                    class="text-body-lg mt-1 whitespace-pre-wrap"
                    style="color: var(--text-primary)"
                  >
                    {comment.content}
                  </p>

                  <!-- 操作行:点赞 / 回复 / 删除(作者本人) -->
                  <div class="mt-1.5 flex items-center gap-3">
                    <button
                      type="button"
                      class="flex items-center gap-1 text-caption outline-none transition-transform active:scale-95"
                      style="color: {comment.is_liked
                        ? 'var(--brand)'
                        : 'var(--text-secondary)'};"
                      onclick={() => toggleCommentLike(comment)}
                      aria-pressed={comment.is_liked ?? false}
                      aria-label={comment.is_liked ? t('common.unlike') : t('common.like')}
                    >
                      <Heart
                        size={13}
                        fill={comment.is_liked ? 'currentColor' : 'none'}
                        strokeWidth={1.8}
                      />
                      {#if comment.likes_count > 0}
                        <span>{comment.likes_count}</span>
                      {/if}
                    </button>
                    <button
                      type="button"
                      class="flex items-center gap-1 text-caption outline-none transition-transform active:scale-95"
                      style="color: var(--text-secondary);"
                      onclick={() => startReply(comment)}
                      aria-expanded={replyingTo === comment.comment_id}
                    >
                      <CornerDownRight size={13} strokeWidth={1.8} />
                      <span>{t('common.reply')}</span>
                    </button>
                    {#if authStore.isAuthenticated && authStore.currentUserId === comment.user.user_id}
                      <button
                        type="button"
                        class="flex items-center gap-1 text-caption outline-none transition-transform active:scale-95"
                        style="color: var(--text-secondary);"
                        onclick={() => deleteComment(comment)}
                        disabled={deletingComments.has(comment.comment_id)}
                        aria-label={t('common.delete')}
                      >
                        {#if deletingComments.has(comment.comment_id)}
                          <Spinner size="xs" />
                        {:else}
                          <Trash2 size={13} strokeWidth={1.8} />
                        {/if}
                      </button>
                    {/if}
                  </div>

                  <!-- 内联回复输入框 -->
                  {#if replyingTo === comment.comment_id}
                    <div class="mt-2 flex items-center gap-2">
                      <Textarea
                        size="sm"
                        variant="inset"
                        class="flex-1"
                        maxlength={commentMaxLength}
                        placeholder={t('momentDetail.replyPlaceholder')}
                        bind:value={replyText}
                        onkeydown={(e) => {
                          if (e.key === 'Enter' && !e.shiftKey) {
                            e.preventDefault();
                            submitReply(comment);
                          }
                        }}
                      />
                      <button
                        type="button"
                        class="flex shrink-0 items-center justify-center rounded-full outline-none transition-transform active:scale-90"
                        style="width: 30px; height: 30px; background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);"
                        onclick={() => submitReply(comment)}
                        disabled={!replyText.trim() || isSubmittingReply}
                        aria-label={t('common.send')}
                      >
                        {#if isSubmittingReply}
                          <Spinner size="xs" color="var(--text-on-accent)" />
                        {:else}
                          <Send size={14} color="var(--text-on-accent)" />
                        {/if}
                      </button>
                    </div>
                  {/if}

                  <!-- top_replies 预览 -->
                  {#if comment.top_replies && comment.top_replies.length > 0}
                    <div class="mt-2 space-y-2">
                      {#each comment.top_replies as reply (reply.comment_id)}
                        <div class="flex items-start gap-2">
                          <a href="/users/{reply.user.user_id}" class="shrink-0">
                            <Avatar
                              src={reply.user.avatar_url}
                              name={reply.user.nickname}
                              alt={reply.user.nickname}
                              size={32}
                            />
                          </a>
                          <p class="text-body" style="color: var(--text-secondary)">
                            <a
                              href="/users/{reply.user.user_id}"
                              style="font-weight: 500; color: var(--text-primary)"
                            >
                              {reply.user.nickname}
                            </a>
                            {reply.content}
                          </p>
                        </div>
                      {/each}
                      {#if comment.replies_count > (comment.top_replies?.length ?? 0)}
                        <button
                          class="text-meta outline-none"
                          style="color: var(--brand)"
                          onclick={() => loadAllReplies(comment.comment_id)}
                        >
                          {t('momentDetail.viewAllReplies', { count: comment.replies_count })}
                        </button>
                      {/if}
                    </div>
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {:else}
      <div
        class="rounded-[var(--radius-lg)] py-12 text-center"
        style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
      >
        <p style="color: var(--text-secondary);">{t('momentDetail.notExist')}</p>
        <button
          class="mt-4 rounded-full px-5 py-2.5 font-medium outline-none"
          style="box-shadow: var(--shadow-md); border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
          onclick={() => goto('/')}
        >
          {t('common.backHome')}
        </button>
      </div>
    {/if}

    {#if isOwner && moment}
      <CreateMomentModal bind:open={showEditModal} editMoment={moment} onEdited={handleEdited} />
    {/if}
  </div>
</div>

{#if moment?.images && moment.images.length > 0}
  <ImagePreview
    images={moment.images.map((img, i) => ({
      src: img.url,
      alt: t('momentDetail.imageAlt', { index: i + 1 })
    }))}
    bind:open={imagePreviewOpen}
    bind:index={imagePreviewIndex}
  />
{/if}
