<script lang="ts">
	import { MapPin } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import Avatar from '@/components/shared/Avatar.svelte';
	import NearbyImageCard from '@/components/data-display/NearbyImageCard.svelte';
	import ImageCardActions from '@/components/data-display/ImageCardActions.svelte';
	import type { FeedItem } from './moment-feed.svelte';
	import { shareMoment } from '@/lib/moment-actions';
	import { ROUTES } from '@/lib/routes';

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
		priority = false,
		compact = false,
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
		/** 首屏图（列表前 2 张）：透传 NearbyImageCard 的 eager/high */
		priority?: boolean;
		/** 窄卡（手机 2 列，约 170px 宽）：操作行切紧凑档，四类操作不裁切 */
		compact?: boolean;
		class?: string;
	} = $props();

	// 卡片可访问名（role=button 的 aria-label 来源）：tag 可为空，回退正文/作者并截断。
	const cardLabel = $derived((item.tag || item.content || item.user).slice(0, 30));

	const canOpenAuthor = $derived(Boolean(item.userId));
	function openAuthor() {
		if (item.userId) void goto(ROUTES.user(item.userId));
	}
	// 分享与首页信息流卡同一实现（lib/moment-actions.shareMoment）
	function handleShare() {
		void shareMoment({ momentId: item.id, authorName: item.user, text: item.content });
	}
</script>

<NearbyImageCard
	imageSrc={item.image || item.avatar}
	imageAlt={cardLabel}
	{onClick}
	{priority}
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
		<span class="media-badge absolute top-2.5 left-2.5 z-[2]">
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
				{#if canOpenAuthor}
					<!-- 作者区与首页信息流卡同契约：头像 hover 放大、昵称 hover 下划线，点击进主页并阻断整卡跳转 -->
					<button
						type="button"
						class="hover-action hit-slop-44 shrink-0 rounded-full focus-ring"
						aria-label={`查看${item.user || '作者'}的个人主页`}
						onclick={(e) => {
							e.stopPropagation();
							openAuthor();
						}}
					>
						<Avatar src={item.avatar} name={item.user} size="sm" />
					</button>
					<button
						type="button"
						class="min-w-0 truncate text-left text-md font-semibold hit-slop-44 focus-ring {onCover ? 'text-content-inverse hover:underline' : 'text-text-primary hover:text-accent-default hover:underline'} transition-colors"
						onclick={(e) => {
							e.stopPropagation();
							openAuthor();
						}}
					>
						{item.user || '匿名用户'}
					</button>
				{:else}
					<Avatar src={item.avatar} name={item.user} size="sm" />
					<span class="min-w-0 flex-1 truncate text-md font-semibold {onCover ? 'text-content-inverse' : 'text-text-primary'}">{item.user || '匿名用户'}</span>
				{/if}
				{#if item.authorRelation === true}
					<span class="shrink-0 {onCover ? 'media-badge' : 'badge bg-accent-soft text-text-accent-on-soft'}">已关注</span>
				{/if}
			{/if}
			{#if item.time}
				<span class="ml-auto shrink-0 text-xs tabular-nums {onCover ? 'text-content-inverse opacity-hover' : 'text-text-secondary'}">{item.time}</span>
			{/if}
		</div>

		<!-- 正文：最多 2 行 -->
		{#if item.content}
			<p class="mt-1.5 line-clamp-2 text-sm leading-card {onCover ? 'text-content-inverse opacity-hover' : 'text-text-secondary'}">
				{item.content}
			</p>
		{/if}

		<!-- 操作行：与首页信息流卡共用 ImageCardActions（同一套图标/间距/hover/press/状态语义） -->
		<div class="mt-2">
			<ImageCardActions
				likes={item.likes}
				comments={item.comments}
				{liked}
				{saved}
				onImage={onCover}
				onLike={() => onLike()}
				onComment={() => (onCommentClick ?? onClick)?.()}
				onSave={() => onSave()}
				onShare={handleShare}
				size={compact ? 'sm' : 'md'}
			/>
		</div>
	</div>
{/snippet}
</NearbyImageCard>