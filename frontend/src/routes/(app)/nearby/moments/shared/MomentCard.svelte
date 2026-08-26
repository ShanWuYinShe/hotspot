<script lang="ts">
	import { MapPin, Heart, MessageCircle, Bookmark } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import NearbyImageCard from '@/components/data-display/NearbyImageCard.svelte';
	import type { FeedItem } from './feed-interactions.svelte';

	// MomentCard — 动态卡片：基于 NearbyImageCard 统一图片优先原语。
	// 标签/作者/正文/操作栏全部以覆盖层显示在图片上（无图时用主题化占位背景）。
	let {
		item,
		liked,
		saved,
		onLike,
		onSave,
		onClick,
		onCommentClick
	}: {
		item: FeedItem;
		liked: boolean;
		saved: boolean;
		onLike: () => void;
		onSave: () => void;
		onClick?: () => void;
		/** 评论计数点击（直达详情页评论区）；缺省回退 onClick */
		onCommentClick?: () => void;
	} = $props();

	// 卡片可访问名（role=button 的 aria-label 来源）：tag 可为空，回退正文/作者并截断。
	const cardLabel = $derived((item.tag || item.content || item.user).slice(0, 30));
</script>

<NearbyImageCard
	imageSrc={item.image ?? item.avatar}
	imageAlt={cardLabel}
	{onClick}
	class="aspect-[4/3] rounded-md"
>
	{#snippet placeholder()}
		<div
			class="flex items-center justify-center rounded-sm size-11.5 bg-accent-bg"
			style="border:var(--cst-border-width-hairline) solid var(--cst-accent-border-soft);box-shadow:var(--shadow-glow-lg);"
		>
			<MapPin size={20} class="text-accent-default" strokeWidth={1.5} />
		</div>
	{/snippet}

	<!-- Tag chip -->
	<div class="absolute top-2.5 left-2.5 z-[2] px-2 py-0.5 rounded-full bg-accent-default shadow-glow-sm" >
		<span class="text-text-on-accent text-xxs">{item.tag}</span>
	</div>

	<!-- 底部信息托盘（surface-card-menu，由 NearbyImageCard footer 槽渲染） -->
	{#snippet footer()}
		<div class="px-3 pb-2.5 pt-2.5">
			<!-- 作者行：头像 + 昵称 + 位置 -->
			<div class="flex min-w-0 items-center gap-2">
				<Avatar src={item.avatar} name={item.user} size={22} style="border:var(--cst-border-width-regular) solid var(--cst-border-card-menu);" />
				<span class="truncate text-xs text-text-primary font-medium">{item.user}</span>
				<span class="ml-auto flex min-w-0 items-center gap-0.5 truncate text-xxs text-text-secondary" >
					<MapPin class="size-2.5 shrink-0 text-text-secondary"   strokeWidth={1.5} />
					{item.location}
				</span>
			</div>

			<!-- 正文：单行省略，muted -->
			<p class="mt-1 line-clamp-1 leading-card text-xs text-text-secondary" >
				{item.content}
			</p>

			<!-- Actions -->
			<div class="mt-1.5 flex items-center justify-between">
				<div class="flex gap-3.5">
					<button
						type="button"
						class="hover-action hit-slop flex items-center gap-1.5 focus-ring {liked ? 'text-accent-default' : 'text-text-secondary'}"
						aria-label={liked ? '取消点赞' : '点赞'}
						onclick={(e) => {
							e.stopPropagation();
							onLike();
						}}
					>
						<Heart class="size-4"  strokeWidth={1.5} fill={liked ? 'currentColor' : 'none'} />
						<span class="text-xs">{item.likes + (liked ? 1 : 0)}</span>
					</button>
					<!-- 评论计数：可点进入动态详情（评论区在详情页内） -->
					<button
						type="button"
						class="hover-action hit-slop flex items-center gap-1.5 focus-ring text-text-secondary"
						aria-label="查看评论"
						onclick={(e) => {
							e.stopPropagation();
							(onCommentClick ?? onClick)?.();
						}}
					>
						<MessageCircle class="size-4"  strokeWidth={1.5} />
						<span class="text-xs">{item.comments}</span>
					</button>
				</div>
				<div class="flex gap-3">
					<button
						type="button"
						class="hover-action hit-slop focus-ring {saved ? 'text-accent-default' : 'text-text-secondary'}"
						aria-label={saved ? '取消收藏' : '收藏'}
						onclick={(e) => {
							e.stopPropagation();
							onSave();
						}}
					>
						<Bookmark class="size-4"  strokeWidth={1.5} fill={saved ? 'currentColor' : 'none'} />
					</button>
				</div>
			</div>
		</div>
	{/snippet}
</NearbyImageCard>
