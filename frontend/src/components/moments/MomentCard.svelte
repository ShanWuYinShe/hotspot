<script lang="ts">
	import { MapPin, Heart, MessageCircle, Bookmark } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import NearbyImageCard from '@/components/data-display/NearbyImageCard.svelte';
	import type { FeedItem } from './moment-feed.svelte';

	// MomentCard — 全站统一动态卡片：基于 NearbyImageCard 图片优先原语。
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
	{#snippet footer()}
		<div class="px-3 pb-2.5 pt-2.5">
			<!-- 作者行：头像 + 昵称 + 位置 -->
			<div class="flex min-w-0 items-center gap-2">
				{#if item.avatar || item.user}
					<Avatar src={item.avatar} name={item.user} size="xs" style="border:var(--cst-border-width-regular) solid var(--cst-border-card-menu);" />
					<span class="truncate text-sm text-text-primary font-semibold">{item.user || '匿名用户'}</span>
				{/if}
				<!-- 位置与时间并列渲染（全站复审 s3 P2-7）：此前 time 只在「无位置」的
				     {:else if} 兜底分支里，而个人中心的 location 又兜了账号位置 →
				     location 恒真、发布时间永不渲染。两者都是有效元信息，改为同时展示：
				     位置可截断、时间不压缩（tabular-nums 对齐）。 -->
				{#if item.location || item.time}
					<span class="ml-auto flex min-w-0 items-center gap-1 text-xxs text-text-secondary">
						{#if item.location}
							<span class="flex min-w-0 items-center gap-0.5 truncate">
								<MapPin class="size-2.5 shrink-0" strokeWidth={1.5} />
								{item.location}
							</span>
						{/if}
						{#if item.location && item.time}
							<span class="opacity-dim" aria-hidden="true">·</span>
						{/if}
						{#if item.time}
							<span class="shrink-0 tabular-nums">{item.time}</span>
						{/if}
					</span>
				{/if}
			</div>

			<!-- 正文：单行或双行省略 -->
			{#if item.content}
				<p class="mt-1 line-clamp-1 leading-card text-xs text-text-secondary">
					{item.content}
				</p>
			{/if}

			<!-- Actions 动作栏 -->
			<div class="mt-1.5 flex items-center justify-between">
				<div class="flex gap-3.5">
					<button
						type="button"
						class="hover-action hit-slop -my-2 flex items-center gap-1.5 py-2 focus-ring {liked
							? 'text-accent-default'
							: 'text-text-secondary hover:text-accent-default'}"
						aria-label={liked ? '取消点赞' : '点赞'}
						aria-pressed={liked}
						onclick={(e) => {
							e.stopPropagation();
							onLike();
						}}
					>
						<Heart class="size-4" strokeWidth={1.5} fill={liked ? 'currentColor' : 'none'} />
						{#key item.likes + (liked ? 1 : 0)}
							<span class="count-pop text-xs">{item.likes + (liked ? 1 : 0)}</span>
						{/key}
					</button>
					<!-- 评论计数：可点进入动态详情（评论区在详情页内） -->
					<button
						type="button"
						class="hover-action hit-slop -my-2 flex items-center gap-1.5 py-2 focus-ring text-text-secondary"
						aria-label="查看评论"
						onclick={(e) => {
							e.stopPropagation();
							(onCommentClick ?? onClick)?.();
						}}
					>
						<MessageCircle class="size-4" strokeWidth={1.5} />
						<span class="text-xs">{item.comments}</span>
					</button>
				</div>
				<div class="flex gap-3">
					<button
						type="button"
						class="hover-action hit-slop -my-2 py-2 focus-ring {saved
							? 'text-accent-default'
							: 'text-text-secondary hover:text-accent-default'}"
						aria-label={saved ? '取消收藏' : '收藏'}
						aria-pressed={saved}
						onclick={(e) => {
							e.stopPropagation();
							onSave();
						}}
					>
						<Bookmark class="size-4" strokeWidth={1.5} fill={saved ? 'currentColor' : 'none'} />
					</button>
				</div>
			</div>
		</div>
	{/snippet}
</NearbyImageCard>
