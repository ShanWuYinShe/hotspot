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

<Card class="cursor-pointer overflow-hidden border-0 bg-card/70 backdrop-blur-sm transition-all hover:shadow-xl hover:shadow-primary/10 hover:-translate-y-0.5">
	<CardHeader class="pb-3">
		<div class="flex items-start justify-between gap-3">
			<a href="/users/{moment.user.user_id}" class="group flex items-center gap-3">
				<Avatar class="h-10 w-10 ring-2 ring-primary/10 transition-all group-hover:ring-primary/30">
					<AvatarImage src={moment.user.avatar_url} alt={moment.user.nickname} />
					<AvatarFallback class="bg-primary/10 text-primary">{moment.user.nickname.slice(0, 2)}</AvatarFallback>
				</Avatar>
				<div>
					<p class="font-medium transition-colors group-hover:text-primary">{moment.user.nickname}</p>
					<div class="flex items-center gap-2 text-xs text-muted-foreground">
						<span>{formatTime(moment.created_at)}</span>
						{#if 'location' in moment && moment.location && 'distance' in moment.location}
							<span class="flex items-center gap-0.5">
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
					<Badge variant="secondary" class="gap-1 rounded-full border-0 px-2 py-1 text-[11px] font-medium {recMeta.className}">
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
		<a href="/moments/{moment.moment_id}" class="block">
			<p class="whitespace-pre-wrap text-sm leading-relaxed">{moment.description}</p>

			{#if moment.images && moment.images.length > 0}
				<div class="mt-3 grid gap-1.5 {moment.images.length === 1 ? '' : moment.images.length === 2 ? 'grid-cols-2' : 'grid-cols-3'}">
					{#each moment.images.slice(0, 9) as image, i}
						<div class="relative aspect-square overflow-hidden rounded-xl bg-muted {moment.images.length === 1 ? 'max-h-80' : ''}">
							<img
								src={image.url}
								alt="瞬间图片 {i + 1}"
								class="h-full w-full object-cover transition-transform duration-300 hover:scale-105"
								loading="lazy"
							/>
							{#if i === 8 && moment.images.length > 9}
								<div class="absolute inset-0 flex items-center justify-center bg-black/50 text-lg font-medium text-white">
									+{moment.images.length - 9}
								</div>
							{/if}
						</div>
					{/each}
				</div>
			{/if}

			{#if moment.tags && moment.tags.length > 0}
				<div class="mt-3 flex flex-wrap gap-1.5">
					{#each moment.tags as tag}
						<Badge variant="secondary" class="rounded-full text-xs font-normal">#{tag}</Badge>
					{/each}
				</div>
			{/if}
		</a>
	</CardContent>

	<CardFooter class="border-t border-border/50 pt-3">
		<div class="flex w-full items-center justify-between">
			<div class="flex gap-0.5">
				<Button
					variant="ghost"
					size="sm"
					class="gap-1.5 rounded-full transition-all {isLiked ? 'text-red-500 hover:text-red-600 hover:bg-red-500/10' : 'hover:text-red-500 hover:bg-red-500/10'}"
					onclick={handleLike}
				>
					<Heart class="h-4 w-4 {isLiked ? 'fill-current' : ''}" />
					<span class="text-xs">{likesCount > 0 ? likesCount : ''}</span>
				</Button>
				<Button 
					variant="ghost" 
					size="sm" 
					class="gap-1.5 rounded-full hover:text-primary hover:bg-primary/10" 
					onclick={() => goto(`/moments/${moment.moment_id}`)}
				>
					<MessageCircle class="h-4 w-4" />
					<span class="text-xs">{moment.comments_count > 0 ? moment.comments_count : ''}</span>
				</Button>
			</div>
			<div class="flex gap-0.5">
				<Button
					variant="ghost"
					size="sm"
					class="rounded-full transition-all {isFavorited ? 'text-yellow-500 hover:text-yellow-600 hover:bg-yellow-500/10' : 'hover:text-yellow-500 hover:bg-yellow-500/10'}"
					onclick={handleFavorite}
				>
					<Bookmark class="h-4 w-4 {isFavorited ? 'fill-current' : ''}" />
				</Button>
				<Button 
					variant="ghost" 
					size="sm" 
					class="rounded-full hover:text-primary hover:bg-primary/10"
					onclick={handleShare}
				>
					<Share2 class="h-4 w-4" />
				</Button>
			</div>
		</div>
	</CardFooter>
</Card>
