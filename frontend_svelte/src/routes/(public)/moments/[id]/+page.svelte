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
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader } from '$lib/components/ui/card';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { Textarea } from '$lib/components/ui/textarea';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Badge } from '$lib/components/ui/badge';
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
		authStore.isAuthenticated && moment !== null && authStore.currentAccount?.user_id === moment.user.user_id
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
			const response = await commentsService.getComments({ moment_id: id, pagination: { limit: 50 } });
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
	<div class="flex items-center gap-4">
		<Button variant="ghost" size="icon" onclick={handleBackClick} aria-label="返回">
			<ArrowLeft class="h-5 w-5" />
		</Button>
		<h1 class="text-xl font-bold">瞬间详情</h1>
		{#if isOwner}
			<div class="ml-auto">
				<Button variant="ghost" size="icon" onclick={() => (showEditModal = true)} aria-label="编辑">
					<Pencil class="h-4 w-4" />
				</Button>
			</div>
		{/if}
	</div>

	{#if isLoading}
		<Card>
			<CardContent class="space-y-4 p-6">
				<div class="flex items-center gap-3">
					<Skeleton class="h-12 w-12 rounded-full" />
					<div class="space-y-2">
						<Skeleton class="h-4 w-24" />
						<Skeleton class="h-3 w-16" />
					</div>
				</div>
				<Skeleton class="h-20 w-full" />
				<Skeleton class="h-64 w-full" />
			</CardContent>
		</Card>
	{:else if moment}
		<Card>
			<CardHeader>
				<a href="/users/{moment.user.user_id}" class="flex items-center gap-3">
					<Avatar class="h-12 w-12">
						<AvatarImage src={moment.user.avatar_url} alt={moment.user.nickname} />
						<AvatarFallback>{moment.user.nickname.slice(0, 2)}</AvatarFallback>
					</Avatar>
					<div>
						<p class="font-medium hover:underline">{moment.user.nickname}</p>
						<div class="flex items-center gap-2 text-sm text-muted-foreground">
							<span>{formatTime(moment.created_at)}</span>
							{#if moment.location}
								<span class="flex items-center gap-0.5">
									<MapPin class="h-3 w-3" />
									{moment.location.location_name}
								</span>
							{/if}
						</div>
					</div>
				</a>
			</CardHeader>

			<CardContent class="space-y-4">
				<p class="whitespace-pre-wrap">{moment.description}</p>

				{#if moment.images && moment.images.length > 0}
					<div class="grid gap-2 {moment.images.length === 1 ? '' : 'grid-cols-2'}">
						{#each moment.images as image, i (image.url)}
							<img
								src={image.url}
								alt="图片 {i + 1}"
								class="w-full rounded-lg"
								loading="lazy"
							/>
						{/each}
					</div>
				{/if}

				{#if moment.tags && moment.tags.length > 0}
					<div class="flex flex-wrap gap-2">
						{#each moment.tags as tag (tag)}
							<Badge variant="secondary">#{tag}</Badge>
						{/each}
					</div>
				{/if}

				<!-- Actions -->
				<div class="flex items-center justify-between border-t pt-4">
					<div class="flex gap-2">
						<Button
							variant="ghost"
							size="sm"
							class="gap-1.5 {isLiked ? 'text-red-500' : ''}"
							onclick={handleLike}
						>
							<Heart class="h-4 w-4 {isLiked ? 'fill-current' : ''}" />
							{likesCount}
						</Button>
						<Button variant="ghost" size="sm" class="gap-1.5">
							<MessageCircle class="h-4 w-4" />
							{commentsCount}
						</Button>
					</div>
					<div class="flex gap-2">
						<Button
							variant="ghost"
							size="sm"
							class={isFavorited ? 'text-yellow-500' : ''}
							onclick={handleFavorite}
						>
							<Bookmark class="h-4 w-4 {isFavorited ? 'fill-current' : ''}" />
						</Button>
						<Button variant="ghost" size="sm">
							<Share2 class="h-4 w-4" />
						</Button>
					</div>
				</div>
			</CardContent>
		</Card>

		<!-- Comment input -->
		{#if authStore.isAuthenticated}
			<Card>
				<CardContent class="p-4">
					<div class="flex gap-3">
						<Avatar class="h-10 w-10">
							<AvatarImage src={authStore.currentAccount?.avatar_url} />
							<AvatarFallback>
								{authStore.currentAccount?.nickname?.slice(0, 2) ?? 'U'}
							</AvatarFallback>
						</Avatar>
						<div class="flex-1">
							<Textarea
								{@attach useKeyboardFocus()}
								placeholder="写下你的评论..."
								bind:value={commentText}
								rows={2}
								disabled={isSubmitting}
							/>
							<div class="mt-2 flex justify-end">
								<Button
									size="sm"
									onclick={submitComment}
									disabled={!commentText.trim() || isSubmitting}
								>
									{#if isSubmitting}
										<Loader2 class="mr-2 h-4 w-4 animate-spin" />
									{:else}
										<Send class="mr-2 h-4 w-4" />
									{/if}
									发送
								</Button>
							</div>
						</div>
					</div>
				</CardContent>
			</Card>
		{/if}

		<!-- Comments -->
		<Card>
			<CardHeader>
				<h2 class="text-lg font-semibold">评论 ({commentsCount})</h2>
			</CardHeader>
			<CardContent>
				{#if isLoadingComments}
					<div class="space-y-4">
						{#each Array(3) as _, i (i)}
							<div class="flex gap-3">
								<Skeleton class="h-10 w-10 rounded-full" />
								<div class="flex-1 space-y-2">
									<Skeleton class="h-4 w-24" />
									<Skeleton class="h-4 w-full" />
								</div>
							</div>
						{/each}
					</div>
				{:else if commentsError}
					<div class="flex flex-col items-center gap-3 py-8 text-center">
						<p class="text-sm text-muted-foreground">加载评论失败</p>
						<Button variant="outline" size="sm" onclick={() => {
							const v = ++loadVersion;
							isLoadingComments = true;
							commentsError = false;
							loadComments(momentId, v);
						}}>重试</Button>
					</div>
				{:else if comments.length === 0}
					<p class="py-8 text-center text-muted-foreground">暂无评论</p>
				{:else}
					<div class="space-y-4">
						{#each comments as comment (comment.comment_id)}
							<div class="flex gap-3">
								<a href="/users/{comment.user.user_id}">
									<Avatar class="h-10 w-10">
										<AvatarImage src={comment.user.avatar_url} alt={comment.user.nickname} />
										<AvatarFallback>{comment.user.nickname.slice(0, 2)}</AvatarFallback>
									</Avatar>
								</a>
								<div class="flex-1">
									<div class="flex items-center gap-2">
										<a href="/users/{comment.user.user_id}" class="font-medium hover:underline">
											{comment.user.nickname}
										</a>
										<span class="text-xs text-muted-foreground">
											{formatTime(comment.created_at)}
										</span>
									</div>
									<p class="mt-1 whitespace-pre-wrap text-sm">{comment.content}</p>
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</CardContent>
		</Card>
	{:else}
		<Card>
			<CardContent class="py-12 text-center">
				<p class="text-muted-foreground">瞬间不存在或已被删除</p>
				<Button variant="outline" class="mt-4" onclick={() => goto('/')}>
					返回首页
				</Button>
			</CardContent>
		</Card>
	{/if}

	{#if isOwner && moment}
		<CreateMomentModal
			bind:open={showEditModal}
			editMoment={moment}
			onEdited={handleEdited}
		/>
	{/if}
</div>
