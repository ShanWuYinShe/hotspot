<script lang="ts">
	import { MapPin, Heart, MessageCircle, Bookmark } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import NearbyImageCard from '@/components/data-display/NearbyImageCard.svelte';
	import type { FeedItem } from './moment-feed.svelte';

	// MomentCard — 全站统一动态卡片：基于 NearbyImageCard 图片优先原语。
	// 卡片比例由调用方按列宽传（2 列窄卡用 3:4 竖版，1 列/桌面用 4:3）：
	// 图片是整卡背景、底部托盘叠在上面，卡太矮时托盘会吃掉 70%+ 的图片（2026-10-20 实测 171×128 卡）。
	// 标签/作者/正文/操作栏统一渲染，适用于 nearby/moments 与 me/moments 等所有场景。
	let {
		item,
		liked,
		saved,
		onLike,
		onSave,
		onClick,
		onCommentClick,
		class: className = 'aspect-[4/3] rounded-md'
	}: {
		item: FeedItem;
		liked: boolean;
		saved: boolean;
		onLike: () => void;
		onSave: () => void;
		onClick?: () => void;
		/** 评论计数点击（直达详情页评论区）；缺省回退 onClick */
		onCommentClick?: () => void;
		class?: string;
	} = $props();

	// 卡片可访问名（role=button 的 aria-label 来源）：tag 可为空，回退正文/作者并截断。
	const cardLabel = $derived((item.tag || item.content || item.user).slice(0, 30));
</script>

<NearbyImageCard
	imageSrc={item.image || item.avatar}
	imageAlt={cardLabel}
	{onClick}
	class={className}
>
	{#snippet watermark()}
		<MapPin size={72} strokeWidth={1} />
	{/snippet}

	{#snippet placeholder()}
		<!-- 占位图标容器：只留描边（border），不叠 glow 投影 -->
		<div
			class="flex items-center justify-center rounded-sm size-11.5 bg-accent-bg"
			style="border:var(--cst-border-width-hairline) solid var(--cst-accent-border-soft);"
		>
			<MapPin size={20} class="text-accent-default" strokeWidth={1.5} />
		</div>
	{/snippet}

	<!-- Tag chip：标准居中胶囊，严格锁定 h-5 与 leading-none 避免行高继承膨胀偏心 -->
	{#if item.tag}
		<!-- Tag chip：毛玻璃底与卡片徽章语言统一（Gemini 审查 r04：降噪，亮蓝让位核心交互），
	     严格锁定 h-5 与 leading-none 避免行高继承膨胀偏心 -->
		<span class="absolute top-2.5 left-2.5 z-[2] inline-flex h-5 items-center justify-center rounded-full bg-surface-overlay-strong px-2.5 text-xxs font-medium leading-none text-content-inverse" style="backdrop-filter:blur(var(--cst-blur-sm));">
			{item.tag}
		</span>
	{/if}

	<!-- 底部信息托盘（surface-card-menu，由 NearbyImageCard footer 槽渲染） -->
	{#snippet footer({ onImage, horizontal }: { onImage: boolean; horizontal: boolean })}
	{@const onCover = onImage && !horizontal}
	<div class="p-4">
		<!-- 作者行：头像 + 昵称 + 时间（压图态白字，fallback 用主题色） -->
		<div class="flex min-w-0 items-center gap-2">
			{#if item.avatar || item.user}
				<Avatar src={item.avatar} name={item.user} size="xs" />
				<span class="min-w-0 flex-1 truncate text-sm font-semibold {onCover ? 'text-content-inverse' : 'text-text-primary'}">{item.user || '匿名用户'}</span>
				{#if item.authorRelation === true}
					<span class="shrink-0 text-xxs {onCover ? 'text-content-inverse opacity-emphasis' : 'text-text-secondary'}">已关注</span>
				{/if}
			{/if}
			{#if item.time}
				<span class="ml-auto shrink-0 text-xxs tabular-nums {onCover ? 'text-content-inverse opacity-emphasis' : 'text-text-secondary'}">{item.time}</span>
			{/if}
		</div>

		<!-- 正文：最多 2 行 -->
		{#if item.content}
			<p class="mt-1 line-clamp-2 leading-card text-xs {onCover ? 'text-content-inverse opacity-hover' : 'text-text-secondary'}">
				{item.content}
			</p>
		{/if}

		<!-- 操作行：压图态白图标；红/主色态用填充+白描边（不依赖色相） -->
		<div class="mt-1.5 flex items-center justify-between {onCover ? 'text-content-inverse' : 'text-text-secondary'}">
			<div class="flex gap-3.5">
				<button
					type="button"
					class="hover-action hit-slop -my-2 flex items-center gap-1.5 py-2 focus-ring {liked
						? (onCover ? 'text-content-inverse' : 'text-accent-default')
						: (onCover ? 'opacity-hover' : 'hover:text-accent-default')}"
					aria-label={liked ? '取消点赞' : '点赞'}
					aria-pressed={liked}
					onclick={(e) => {
						e.stopPropagation();
						onLike();
					}}
				>
					<Heart class="size-4.5 {onCover && liked ? 'fill-status-danger text-content-inverse' : onCover ? 'text-content-inverse' : ''}" strokeWidth={1.5} fill={!onCover && liked ? 'currentColor' : 'none'} />
					{#key item.likes + (liked ? 1 : 0)}
						<span class="count-pop text-xs">{item.likes + (liked ? 1 : 0)}</span>
					{/key}
				</button>
				<button
					type="button"
					class="hover-action hit-slop -my-2 flex items-center gap-1.5 py-2 focus-ring {onCover ? 'opacity-hover' : ''}"
					aria-label="查看评论"
					onclick={(e) => {
						e.stopPropagation();
						(onCommentClick ?? onClick)?.();
					}}
				>
					<MessageCircle class="size-4.5" strokeWidth={1.5} />
					<span class="text-xs">{item.comments}</span>
				</button>
			</div>
			<button
				type="button"
				class="hover-action hit-slop -my-2 py-2 focus-ring {saved
					? (onCover ? 'text-content-inverse' : 'text-accent-default')
					: (onCover ? 'opacity-hover' : 'hover:text-accent-default')}"
				aria-label={saved ? '取消收藏' : '收藏'}
				aria-pressed={saved}
				onclick={(e) => {
					e.stopPropagation();
					onSave();
				}}
			>
				<Bookmark class="size-4.5 {onCover && saved ? 'fill-accent-default text-content-inverse' : onCover ? 'text-content-inverse' : ''}" strokeWidth={1.5} fill={!onCover && saved ? 'currentColor' : 'none'} />
			</button>
		</div>
	</div>
{/snippet}
</NearbyImageCard>
