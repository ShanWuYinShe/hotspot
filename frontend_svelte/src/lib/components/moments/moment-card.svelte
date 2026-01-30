<script lang="ts">
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/auth.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { momentsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { Card, CardContent, CardFooter, CardHeader } from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import type { FeedMomentItem, MomentCard as MomentCardType, MomentSearchItem } from '$lib/types/api';
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
	moment: FeedMomentItem | MomentCardType | MomentSearchItem;
}

	let { moment }: Props = $props();

	// 从 moment 获取初始状态的辅助函数
function getInitialLiked(m: FeedMomentItem | MomentCardType | MomentSearchItem): boolean {
	if ('is_liked' in m) return m.is_liked;
	if ('my_state' in m) return m.my_state?.is_liked ?? false;
	return false;
}

function getInitialFavorited(m: FeedMomentItem | MomentCardType | MomentSearchItem): boolean {
	if ('is_favorited' in m) return m.is_favorited;
	if ('my_state' in m) return m.my_state?.is_favorited ?? false;
	return false;
}

	// 本地状态（乐观更新）
	let isLiked = $state(false);
	let isFavorited = $state(false);
	let likesCount = $state(0);

	// 当 moment prop 变化时重新初始化状态
	$effect(() => {
		isLiked = getInitialLiked(moment);
		isFavorited = getInitialFavorited(moment);
		likesCount = moment.likes_count;
	});

	async function doLike() {
		// 乐观更新
		const wasLiked = isLiked;
		isLiked = !isLiked;
		likesCount += isLiked ? 1 : -1;

		try {
			const response = isLiked
				? await momentsService.like({ moment_id: moment.moment_id })
				: await momentsService.unlike({ moment_id: moment.moment_id });

			if (!isSuccess(response)) {
				// 回滚
				isLiked = wasLiked;
				likesCount += wasLiked ? 1 : -1;
				toast.error(response.message || '操作失败');
			}
		} catch {
			// 回滚
			isLiked = wasLiked;
			likesCount += wasLiked ? 1 : -1;
			toast.error('操作失败');
		}
	}

	function handleLike() {
		if (!authStore.isAuthenticated) {
			// 打开登录模态框，登录成功后执行点赞
			uiStore.openLoginModal(doLike);
			return;
		}
		doLike();
	}

	async function doFavorite() {
		const wasFavorited = isFavorited;
		isFavorited = !isFavorited;

		try {
			const response = isFavorited
				? await momentsService.favorite({ moment_id: moment.moment_id })
				: await momentsService.unfavorite({ moment_id: moment.moment_id });

			if (!isSuccess(response)) {
				isFavorited = wasFavorited;
				toast.error(response.message || '操作失败');
			}
		} catch {
			isFavorited = wasFavorited;
			toast.error('操作失败');
		}
	}

	function handleFavorite() {
		if (!authStore.isAuthenticated) {
			uiStore.openLoginModal(doFavorite);
			return;
		}
		doFavorite();
	}

	function handleShare() {
		if (navigator.share) {
			navigator.share({
				title: `${moment.user.nickname} 的瞬间`,
				text: moment.description.slice(0, 100),
				url: `${window.location.origin}/moments/${moment.moment_id}`
			}).catch(() => {});
		} else {
			// 复制链接
			navigator.clipboard.writeText(`${window.location.origin}/moments/${moment.moment_id}`);
			toast.success('链接已复制');
		}
	}

	function formatTime(dateStr: string): string {
		const date = new Date(dateStr);
		const now = new Date();
		const diff = now.getTime() - date.getTime();
		const minutes = Math.floor(diff / 60000);
		const hours = Math.floor(diff / 3600000);
		const days = Math.floor(diff / 86400000);

		if (minutes < 1) return '刚刚';
		if (minutes < 60) return `${minutes}分钟前`;
		if (hours < 24) return `${hours}小时前`;
		if (days < 7) return `${days}天前`;
		return date.toLocaleDateString('zh-CN');
	}

	function formatDistance(meters: number): string {
		if (meters < 1000) return `${Math.round(meters)}m`;
		return `${(meters / 1000).toFixed(1)}km`;
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

<Card class="group/card cursor-pointer overflow-hidden border border-border/60 bg-card/90 dark:bg-card/30 backdrop-blur-md transition-all duration-300 hover:border-primary/50 hover:bg-card hover:shadow-lg hover:shadow-primary/10 hover:-translate-y-1 dark:hover:bg-card/60 dark:hover:border-primary/40">
	<CardHeader class="pb-3 pt-4">
		<div class="flex items-start justify-between gap-3">
			<a href="/users/{moment.user.user_id}" class="group/user flex items-center gap-3">
				<div class="relative">
					<Avatar class="h-10 w-10 ring-2 ring-transparent transition-all duration-300 group-hover/user:ring-primary/20">
						<AvatarImage src={moment.user.avatar_url} alt={moment.user.nickname} />
						<AvatarFallback class="bg-primary/5 text-primary font-medium">{moment.user.nickname.slice(0, 2)}</AvatarFallback>
					</Avatar>
				</div>
				<div>
					<p class="font-semibold text-foreground transition-colors group-hover/user:text-primary">{moment.user.nickname}</p>
					<div class="flex items-center gap-2 text-xs text-muted-foreground">
						<span>{formatTime(moment.created_at)}</span>
						{#if 'location' in moment && moment.location && 'distance' in moment.location}
							<span class="flex items-center gap-0.5 rounded-full bg-muted/50 px-1.5 py-0.5">
								<MapPin class="h-3 w-3" />
								{formatDistance(moment.location.distance)}
							</span>
						{/if}
					</div>
				</div>
			</a>
			{#if 'recommendation' in moment && moment.recommendation}
				{@const recMeta = getRecommendationMeta(moment.recommendation.type)}
				{#if recMeta}
					<Badge variant="secondary" class="gap-1 rounded-full border-0 px-2.5 py-0.5 text-[10px] font-medium shadow-sm {recMeta.className}">
						{#if moment.recommendation.type === 'friend_liked'}
							<Users class="h-3 w-3" />
						{:else if moment.recommendation.type === 'nearby'}
							<MapPin class="h-3 w-3" />
						{:else if moment.recommendation.type === 'trending'}
							<TrendingUp class="h-3 w-3" />
						{:else if moment.recommendation.type === 'following'}
							<UserCheck class="h-3 w-3" />
						{:else if moment.recommendation.type === 'tag_match'}
							<Sparkles class="h-3 w-3" />
						{/if}
						<span>{recMeta.label}</span>
					</Badge>
				{/if}
			{/if}
		</div>
	</CardHeader>

	<CardContent class="pb-3">
		<a href="/moments/{moment.moment_id}" class="block space-y-3">
			{#if moment.description}
				<p class="whitespace-pre-wrap text-sm leading-relaxed text-foreground/90">{moment.description}</p>
			{/if}

			{#if moment.images && moment.images.length > 0}
				<div class="grid gap-1.5 overflow-hidden rounded-xl border border-border/20 {moment.images.length === 1 ? '' : moment.images.length === 2 ? 'grid-cols-2' : 'grid-cols-3'}">
					{#each moment.images.slice(0, 9) as image, i}
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
				<div class="flex flex-wrap gap-1.5 pt-1">
					{#each moment.tags as tag}
						<Badge variant="outline" class="rounded-md border-primary/10 bg-primary/5 px-2 py-0 text-xs font-normal text-primary hover:bg-primary/10">#{tag}</Badge>
					{/each}
				</div>
			{/if}
		</a>
	</CardContent>

	<CardFooter class="border-t border-border/40 bg-muted/5 py-2 transition-colors group-hover/card:bg-muted/20">
		<div class="flex w-full items-center justify-between px-1">
			<div class="flex gap-4">
				<Button
					variant="ghost"
					size="sm"
					class="h-8 gap-1.5 rounded-full px-2 text-muted-foreground transition-all hover:bg-red-500/10 hover:text-red-500 {isLiked ? 'text-red-500 bg-red-500/5' : ''}"
					onclick={handleLike}
				>
					<Heart class="h-4 w-4 {isLiked ? 'fill-current' : ''}" />
					<span class="text-xs font-medium">{likesCount > 0 ? likesCount : '点赞'}</span>
				</Button>
				<Button 
					variant="ghost" 
					size="sm" 
					class="h-8 gap-1.5 rounded-full px-2 text-muted-foreground hover:bg-primary/10 hover:text-primary" 
					onclick={() => goto(`/moments/${moment.moment_id}`)}
				>
					<MessageCircle class="h-4 w-4" />
					<span class="text-xs font-medium">{moment.comments_count > 0 ? moment.comments_count : '评论'}</span>
				</Button>
			</div>
			<div class="flex gap-1">
				<Button
					variant="ghost"
					size="icon"
					class="h-8 w-8 rounded-full text-muted-foreground transition-all hover:bg-yellow-500/10 hover:text-yellow-500 {isFavorited ? 'text-yellow-500 bg-yellow-500/5' : ''}"
					onclick={handleFavorite}
				>
					<Bookmark class="h-4 w-4 {isFavorited ? 'fill-current' : ''}" />
				</Button>
				<Button 
					variant="ghost" 
					size="icon" 
					class="h-8 w-8 rounded-full text-muted-foreground hover:bg-primary/10 hover:text-primary"
					onclick={handleShare}
				>
					<Share2 class="h-4 w-4" />
				</Button>
			</div>
		</div>
	</CardFooter>
</Card>
