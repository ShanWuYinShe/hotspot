<script lang="ts">
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/auth.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { momentsService } from '$lib/api/services';
	import { toast } from '$lib/stores/toast.svelte';
	import { useOptimisticToggle } from '$lib/hooks/use-optimistic.svelte';
	import { formatRelativeTime, formatDistance } from '$lib/utils/format';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { Card, CardContent, CardFooter, CardHeader } from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import type { MomentItem } from '$lib/types/api';
	import Heart from 'lucide-svelte/icons/heart';
	import MessageCircle from 'lucide-svelte/icons/message-circle';
	import Bookmark from 'lucide-svelte/icons/bookmark';
	import Share2 from 'lucide-svelte/icons/share-2';
	import MapPin from 'lucide-svelte/icons/map-pin';
	import TrendingUp from 'lucide-svelte/icons/trending-up';
	import Sparkles from 'lucide-svelte/icons/sparkles';
	import Users from 'lucide-svelte/icons/users';
	import UserCheck from 'lucide-svelte/icons/user-check';

	interface Props {
		moment: MomentItem;
	class?: string;
	}

let { moment, class: className = '' }: Props = $props();

	// 乐观更新 Hooks
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
		if (event.defaultPrevented || isInteractiveTarget(event.target)) {
			return;
		}

		goto(`/moments/${moment.moment_id}`);
	}

	function handleCardKeydown(event: KeyboardEvent) {
		if (event.defaultPrevented || isInteractiveTarget(event.target)) {
			return;
		}

		if (event.key === 'Enter' || event.key === ' ') {
			event.preventDefault();
			goto(`/moments/${moment.moment_id}`);
		}
	}

	function handleLike(event: MouseEvent) {
		event.preventDefault();
		event.stopPropagation();
		if (!authStore.isAuthenticated) {
			uiStore.openLoginModal(likeAction.toggle);
			return;
		}
		likeAction.toggle();
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
				// 用户取消分享（AbortError）不需要提示
				if (error instanceof DOMException && error.name === 'AbortError') return;
				toast.error('分享失败');
			});
		} else {
			// 复制链接
			navigator.clipboard.writeText(shareData.url);
			toast.success('链接已复制');
		}
	}

	function handleMomentAuthorClick(event: MouseEvent) {
		event.preventDefault();
		event.stopPropagation();
		goto(`/users/${moment.user.user_id}`);
	}

	function handleCommentClick(event: MouseEvent) {
		event.preventDefault();
		event.stopPropagation();
		goto(`/moments/${moment.moment_id}`);
	}


	function getRecommendationMeta(type?: string) {
		switch (type) {
			case 'friend_liked':
				return { label: '好友觉得赞', icon: Users, className: 'bg-amber-500/10 text-amber-600 dark:text-amber-400' };
			case 'nearby':
				return { label: '附近热门', icon: MapPin, className: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400' };
			case 'trending':
				return { label: '正在升温', icon: TrendingUp, className: 'bg-orange-500/10 text-orange-600 dark:text-orange-400' };
			case 'following':
				return { label: '你关注的人', icon: UserCheck, className: 'bg-sky-500/10 text-sky-600 dark:text-sky-400' };
			case 'tag_match':
				return { label: '兴趣匹配', icon: Sparkles, className: 'bg-purple-500/10 text-purple-600 dark:text-purple-400' };
			default:
				return null;
		}
	}
</script>

<Card
	variant="accent"
	glow="soft"
	interactive
	tabindex={0}
	role="link"
	onclick={handleCardClick}
	onkeydown={handleCardKeydown}
	class="group/card cursor-pointer overflow-hidden animate-bounce-in {className}"
>
	<CardHeader class="pb-3 pt-4">
		<div class="flex items-start justify-between gap-3">
			<a
				href="/users/{moment.user.user_id}"
				data-card-action
				class="group/user flex items-center gap-3"
				onclick={handleMomentAuthorClick}
			>
				<div class="relative">
					<Avatar glow class="h-10 w-10">
						<AvatarImage src={moment.user.avatar_url} alt={moment.user.nickname} />
						<AvatarFallback class="bg-linear-to-br from-primary/20 to-secondary/20 text-primary font-bold">{moment.user.nickname.slice(0, 2)}</AvatarFallback>
					</Avatar>
				</div>
				<div>
					<p class="font-semibold text-foreground transition-colors group-hover/user:text-primary">{moment.user.nickname}</p>
					<div class="flex items-center gap-2 text-xs text-muted-foreground">
						<span>{formatRelativeTime(moment.created_at)}</span>
						{#if moment.location?.distance != null}
							{@const distance = moment.location.distance}
							<span class="flex items-center gap-0.5 rounded-2xl bg-muted/50 px-1.5 py-0.5">
								<MapPin class="h-3 w-3" />
								{formatDistance(distance)}
							</span>
						{/if}
					</div>
				</div>
			</a>
			{#if 'recommendation' in moment && moment.recommendation}
				{@const recMeta = getRecommendationMeta(moment.recommendation.type)}
				{#if recMeta}
					<Badge variant="accent" class="gap-1.5 px-3 py-1.5 {recMeta.className}">
						{#if moment.recommendation.type === 'friend_liked'}
							<Users class="h-3.5 w-3.5" />
						{:else if moment.recommendation.type === 'nearby'}
							<MapPin class="h-3.5 w-3.5" />
						{:else if moment.recommendation.type === 'trending'}
							<TrendingUp class="h-3.5 w-3.5" />
						{:else if moment.recommendation.type === 'following'}
							<UserCheck class="h-3.5 w-3.5" />
						{:else if moment.recommendation.type === 'tag_match'}
							<Sparkles class="h-3.5 w-3.5" />
						{/if}
						<span class="font-bold text-[11px]">{recMeta.label}</span>
					</Badge>
				{/if}
			{/if}
		</div>
	</CardHeader>

	<CardContent class="pb-3">
		<div class="block space-y-3">
			{#if moment.description}
				<p class="whitespace-pre-wrap text-sm leading-relaxed text-foreground/90">{moment.description}</p>
			{/if}

			{#if moment.images && moment.images.length > 0}
				<div class="grid gap-2 overflow-hidden rounded-2xl border-2 border-primary/10 {moment.images.length === 1 ? '' : moment.images.length === 2 ? 'grid-cols-2' : 'grid-cols-3'}">
					{#each moment.images.slice(0, 9) as image, i (image.url)}
						{#if moment.images.length === 1}
							<div class="relative overflow-hidden bg-muted/30">
								<img
									src={image.url}
									alt="瞬间图片"
									class="h-auto w-full object-cover transition-transform duration-500 hover:scale-105"
									loading="lazy"
								/>
							</div>
						{:else}
							<div class="relative aspect-square overflow-hidden bg-muted/30">
								<img
									src={image.url}
									alt="瞬间图片 {i + 1}"
									class="h-full w-full object-cover transition-transform duration-500 hover:scale-105"
									loading="lazy"
								/>
								{#if i === 8 && moment.images.length > 9}
									<div class="absolute inset-0 flex items-center justify-center bg-black/60 text-lg font-medium text-white backdrop-blur-[2px]">
										+{moment.images.length - 9}
									</div>
								{/if}
							</div>
						{/if}
					{/each}
				</div>
			{/if}

			{#if moment.tags && moment.tags.length > 0}
				<div class="flex flex-wrap gap-2 pt-1">
					{#each moment.tags as tag (tag)}
						<Badge variant="default" class="text-xs">#{tag}</Badge>
					{/each}
				</div>
			{/if}
		</div>
	</CardContent>

	<CardFooter class="border-t border-border/40 bg-muted/10 py-2 md:bg-muted/5 md:transition-colors md:group-hover/card:bg-muted/20">
		<div class="flex w-full flex-col gap-2 px-1 sm:flex-row sm:items-center sm:justify-between">
			<div class="flex flex-wrap gap-2">
				<Button
					variant="ghost"
					size="sm"
					data-card-action
					class={likeAction.isOn ? 'bg-red-500/15 text-red-500' : ''}
					onclick={handleLike}
					disabled={likeAction.isLoading}
				>
					<Heart class="h-5 w-5 transition-all {likeAction.isOn && 'fill-current animate-bounce-in'}" />
					<span class="font-semibold text-sm">{likeAction.count > 0 ? likeAction.count : '点赞'}</span>
				</Button>
				<Button
					variant="ghost"
					size="sm"
					data-card-action
					onclick={handleCommentClick}
				>
					<MessageCircle class="h-5 w-5" />
					<span class="font-semibold text-sm">{moment.comments_count > 0 ? moment.comments_count : '评论'}</span>
				</Button>
			</div>
			<div class="flex flex-wrap items-center justify-end gap-2">
				<Button
					variant="ghost"
					size="icon"
					data-card-action
					class={favoriteAction.isOn ? 'bg-yellow-500/15 text-yellow-500' : ''}
					onclick={handleFavorite}
					disabled={favoriteAction.isLoading}
					aria-label={favoriteAction.isOn ? '取消收藏' : '收藏'}
				>
					<Bookmark class="h-5 w-5 {favoriteAction.isOn && 'fill-current animate-bounce-in'}" />
				</Button>
				<Button
					variant="ghost"
					size="icon"
					data-card-action
					onclick={handleShare}
					aria-label="分享"
				>
					<Share2 class="h-5 w-5" />
				</Button>
			</div>
		</div>
	</CardFooter>
</Card>
