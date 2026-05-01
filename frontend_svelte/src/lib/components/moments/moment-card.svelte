<script lang="ts">
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/auth.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { momentsService } from '$lib/api/services';
	import { toast } from '$lib/stores/toast.svelte';
	import { useOptimisticToggle } from '$lib/hooks/use-optimistic.svelte';
	import { formatRelativeTime, formatDistance } from '$lib/utils/format';
	import type { MomentItem } from '$lib/types/api';
	import Heart from 'lucide-svelte/icons/heart';
	import MessageCircle from 'lucide-svelte/icons/message-circle';
	import Bookmark from 'lucide-svelte/icons/bookmark';
	import Share2 from 'lucide-svelte/icons/share-2';
	import MapPin from 'lucide-svelte/icons/map-pin';

	interface Props {
		moment: MomentItem;
		class?: string;
	}

	let { moment, class: className = '' }: Props = $props();

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
			? Boolean(target.closest('button, a, input, select, textarea, [data-card-action]'))
			: false;
	}

	function handleCardClick(event: MouseEvent) {
		if (event.defaultPrevented || isInteractiveTarget(event.target)) return;
		goto(`/moments/${moment.moment_id}`);
	}

	function handleLike(event: MouseEvent) {
		event.preventDefault();
		event.stopPropagation();
		if (!authStore.isAuthenticated) { uiStore.openLoginModal(likeAction.toggle); return; }
		likeAction.toggle();
	}

	function handleFavorite(event: MouseEvent) {
		event.preventDefault();
		event.stopPropagation();
		if (!authStore.isAuthenticated) { uiStore.openLoginModal(favoriteAction.toggle); return; }
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
		goto(`/users/${moment.user.user_id}`);
	}
</script>

<div
	class="cursor-pointer overflow-hidden rounded-[22px] transition-shadow {className}"
	style="background: var(--neu-bg); box-shadow: 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
	onclick={handleCardClick}
	role="link"
	tabindex={0}
	onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); goto(`/moments/${moment.moment_id}`); } }}
>
	<!-- Image -->
	{#if moment.images && moment.images.length > 0}
		<div class="relative overflow-hidden" style="aspect-ratio: 16/9;">
			<img
				src={moment.images[0]?.url ?? ""}
				alt="动态图片"
				class="h-full w-full object-cover"
				loading="lazy"
			/>
			<div class="absolute inset-0" style="background: linear-gradient(to bottom, transparent 50%, rgba(0,0,0,0.35));"></div>
			<!-- Tags overlay -->
			{#if moment.tags && moment.tags.length > 0}
				<div class="absolute left-2.5 top-2.5">
					<span
						class="rounded-full px-2 py-0.5"
						style="background: var(--neu-accent-primary); box-shadow: 0 2px 8px var(--neu-glow); font-size: 10px; color: var(--neu-active-text);"
					>
						{moment.tags[0]}
					</span>
				</div>
			{/if}
			<!-- User overlay on image bottom -->
			<div class="absolute bottom-0 left-0 right-0 flex items-center gap-2 px-3 pb-2.5">
				<a
					href="/users/{moment.user.user_id}"
					data-card-action
					onclick={handleAuthorClick}
					class="flex-shrink-0 overflow-hidden rounded-full"
					style="width: 24px; height: 24px; border: 1.5px solid rgba(255,255,255,0.6);"
				>
					<img src={moment.user.avatar_url || ''} alt={moment.user.nickname} class="h-full w-full object-cover" />
				</a>
				<a
					href="/users/{moment.user.user_id}"
					data-card-action
					onclick={handleAuthorClick}
					style="font-size: 11px; color: white; font-weight: 500;"
				>
					{moment.user.nickname}
				</a>
				{#if moment.location?.distance != null}
					<div class="ml-auto flex items-center gap-0.5">
						<MapPin size={9} strokeWidth={1.5} color="rgba(255,255,255,0.8)" />
						<span style="font-size: 10px; color: rgba(255,255,255,0.8);">{formatDistance(moment.location.distance)}</span>
					</div>
				{/if}
			</div>
		</div>
	{/if}

	<!-- Content -->
	<div class="px-4 pt-3 pb-2">
		{#if !moment.images || moment.images.length === 0}
			<!-- No image: show author info inline -->
			<div class="mb-2 flex items-center gap-2">
				<a
					href="/users/{moment.user.user_id}"
					data-card-action
					onclick={handleAuthorClick}
					class="flex-shrink-0 overflow-hidden rounded-full"
					style="width: 28px; height: 28px; border: 1.5px solid var(--neu-border);"
				>
					<img src={moment.user.avatar_url || ''} alt={moment.user.nickname} class="h-full w-full object-cover" />
				</a>
				<div class="min-w-0 flex-1">
					<a
						href="/users/{moment.user.user_id}"
						data-card-action
						onclick={handleAuthorClick}
						style="font-size: 13px; color: var(--neu-text); font-weight: 500;"
					>
						{moment.user.nickname}
					</a>
					<div class="flex items-center gap-2" style="margin-top: 1px;">
						<span style="font-size: 10px; color: var(--neu-text-muted);">{formatRelativeTime(moment.created_at)}</span>
						{#if moment.location?.distance != null}
							<span style="font-size: 10px; color: var(--neu-accent-primary);">{formatDistance(moment.location.distance)}</span>
						{/if}
					</div>
				</div>
			</div>
		{/if}
		{#if moment.description}
			<p class="line-clamp-2" style="font-size: 12px; color: var(--neu-text-muted); line-height: 1.6;">
				{moment.description}
			</p>
		{/if}
		{#if moment.images && moment.images.length > 0}
			<div class="mt-1 flex items-center gap-1" style="font-size: 10px; color: var(--neu-text-muted);">
				<span>{formatRelativeTime(moment.created_at)}</span>
			</div>
		{/if}
	</div>

	<!-- Actions -->
	<div class="flex items-center justify-between px-4 pb-3.5">
		<div class="flex gap-3">
			<button
				data-card-action
				class="flex items-center gap-1.5 py-1 outline-none"
				style="color: {likeAction.isOn ? 'var(--neu-accent-primary)' : 'var(--neu-text-muted)'};"
				onclick={handleLike}
				disabled={likeAction.isLoading}
			>
				<Heart size={16} strokeWidth={1.5} fill={likeAction.isOn ? 'currentColor' : 'none'} />
				<span style="font-size: 12px;">{likeAction.count}</span>
			</button>
			<button
				data-card-action
				class="flex items-center gap-1.5 py-1 outline-none"
				style="color: var(--neu-text-muted);"
				onclick={(e) => { e.preventDefault(); e.stopPropagation(); goto(`/moments/${moment.moment_id}`); }}
			>
				<MessageCircle size={16} strokeWidth={1.5} />
				<span style="font-size: 12px;">{moment.comments_count}</span>
			</button>
		</div>
		<div class="flex gap-3">
			<button
				data-card-action
				class="py-1 outline-none"
				style="color: {favoriteAction.isOn ? 'var(--neu-accent-primary)' : 'var(--neu-text-muted)'};"
				onclick={handleFavorite}
				disabled={favoriteAction.isLoading}
			>
				<Bookmark size={16} strokeWidth={1.5} fill={favoriteAction.isOn ? 'currentColor' : 'none'} />
			</button>
			<button
				data-card-action
				class="py-1 outline-none"
				style="color: var(--neu-text-muted);"
				onclick={handleShare}
			>
				<Share2 size={16} strokeWidth={1.5} />
			</button>
		</div>
	</div>
</div>
