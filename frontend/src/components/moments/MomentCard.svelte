<script lang="ts">
	import { MapPin } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import Avatar from '@/components/shared/Avatar.svelte';
	import MediaCard from '@/components/data-display/MediaCard.svelte';
	import ImageCardActions from '@/components/data-display/ImageCardActions.svelte';
	import type { FeedItem } from './moment-feed.svelte';
	import { shareMoment } from '@/lib/moment-actions';
	import { ROUTES } from '@/lib/routes';
	import FollowButton from '@/components/ui/FollowButton.svelte';
	import type { Tier } from '@/lib/bsp-layout';
	import { auth, requireAuth } from '@/stores/auth.svelte';
	import { follow } from '@/api';
	import { showToast } from '@/stores/toast.svelte';

	// MomentCard — 全站统一动态卡片：基于 NearbyImageCard 图片优先原语。
	// 卡片比例由调用方按列宽传（2 列窄卡用 3:4 竖版，1 列/桌面用 4:3）：
	// 图片是整卡背景、底部托盘叠在上面，卡太矮时托盘会吃掉 70%+ 的图片（2026-10-20 实测 171×128 卡）。
	// 标签/作者/正文/操作栏统一渲染，适用于 nearby/moments 与 me/moments 等所有场景。
	let {
		item,
		liked,
		saved = false,
		onLike,
		onSave,
		onClick,
		onCommentClick,
		priority = false,
		compact = false,
		tier = '1x1' as Tier,
		horizontal = false,
		class: className = 'aspect-[4/3] rounded-lg'
	}: {
		item: FeedItem;
		liked: boolean;
		saved?: boolean;
		onLike: () => void;
		onSave?: () => void;
		onClick?: () => void;
		/** 评论计数点击（直达详情页评论区）；缺省回退 onClick */
		onCommentClick?: () => void;
		/** 首屏图（列表前 2 张）：透传 NearbyImageCard 的 eager/high */
		priority?: boolean;
		/** 窄卡（手机 2 列，约 170px 宽）：操作行切紧凑档 */
		compact?: boolean;
		/** BSP 尺寸档位：1x1, 1x2, 2x1, 2x2 */
		tier?: Tier;
		/** 横排变体（左图右文） */
		horizontal?: boolean;
		class?: string;
	} = $props();

	const isHorizontal = $derived(horizontal || tier === '2x1');
	const isHero = $derived(tier === '2x2');

	// 卡片可访问名（role=button 的 aria-label 来源）：tag 可为空，回退正文/作者并截断。
	const cardLabel = $derived((item.tag || item.content || item.user).slice(0, 30));

	const canOpenAuthor = $derived(Boolean(item.userId));
	function openAuthor() {
		if (item.userId) void goto(ROUTES.user(item.userId));
	}

	// 作者关注态（2026-10-29）：与 HomeFeedItem 同契约——「已关注」必须是可点按钮，
	// 点击取消关注而不是穿透整卡打开详情；未关注显示「关注」。
	let followOverride = $state<boolean | null>(null);
	let followPending = $state(false);
	$effect(() => {
		void item.id;
		followOverride = null;
		followPending = false;
	});
	const canFollow = $derived(Boolean(item.userId) && item.userId !== auth.user?.user_id);
	const isFollowing = $derived(followOverride ?? Boolean(item.authorRelation));
	// 与 HomeFeedItem 同口径：默认只在已关注作者卡显示，用户取关后就地保留「关注」chip。
	const showFollowChip = $derived(canFollow && (isFollowing || followOverride !== null));

	async function toggleAuthorFollow(next: boolean) {
		if (!item.userId || followPending) return;
		followPending = true;
		followOverride = next;
		try {
			await follow({ user_id: item.userId, active: next });
			showToast(next ? '已关注 ' + item.user : '已取消关注 ' + item.user, 'success');
		} catch {
			followOverride = !next;
			showToast(next ? '关注失败，请重试' : '取消关注失败，请重试', 'danger');
		} finally {
			followPending = false;
		}
	}

	function handleFollowToggle(e: MouseEvent) {
		e.stopPropagation();
		if (!item.userId || followPending) return;
		const next = !isFollowing;
		if (!requireAuth({ action: next ? '关注' : '取消关注', targetId: item.userId, run: () => void toggleAuthorFollow(next) })) return;
		void toggleAuthorFollow(next);
	}
</script>

<MediaCard
	imageSrc={item.image}
	imageAlt={cardLabel}
	{onClick}
	{priority}
	horizontal={isHorizontal}
	class={className}
>
	<!-- Tag chip：左上角类别/标签 -->
	{#if item.tag}
		<span class="pointer-events-none media-badge absolute top-2.5 left-2.5 z-[2]">
			{item.tag}
		</span>
	{/if}

	<!-- 时间徽章：右上角轻量显示，与左上角标签平衡且不占用底部操作行空间 -->
	{#if item.time}
		<span class="pointer-events-none media-badge absolute top-2.5 right-2.5 z-[2] text-xxs tabular-nums font-medium">
			{item.time}
		</span>
	{/if}

	<!-- 底部信息托盘：紧凑排布在卡片底部 25% 区域，彻底腾出上方图片空间 -->
	{#snippet footer({ onImage, horizontal }: { onImage: boolean; horizontal: boolean })}
		{@const onCover = onImage && !horizontal}
		<div class="pointer-events-none flex flex-col gap-1 p-2 sm:p-2.5 {isHorizontal ? 'gap-3 p-3 sm:p-4 justify-center' : ''}" data-card-footer>
			<!-- 正文：普通卡 2 行，横排/巨幕卡 3 行 -->
			{#if item.content}
				<p class="pointer-events-none line-clamp-2 {isHorizontal || isHero ? 'line-clamp-3 sm:text-base' : ''} text-xs sm:text-sm font-semibold leading-snug drop-shadow-xs {onCover ? 'text-content-inverse' : 'text-text-primary'}">
					{item.content}
				</p>
			{/if}

			<!-- 底行：左侧作者信息（头像+昵称），右侧核心操作（仅点赞与评论，无分享/收藏冗余） -->
			<div class="flex items-center justify-between gap-1.5 pt-0.5">
				<div class="flex min-w-0 items-center gap-1.5">
					{#if item.avatar || item.user}
						{#if canOpenAuthor}
							<button
								type="button"
								class="pointer-events-auto hover-action hit-slop-44 shrink-0 rounded-full focus-ring"
								aria-label={`查看${item.user || '作者'}的个人主页`}
								onclick={(e) => {
									e.stopPropagation();
									openAuthor();
								}}
							>
								<Avatar src={item.avatar} name={item.user} size="xs" />
							</button>
							<button
								type="button"
								class="pointer-events-auto min-w-0 truncate text-left text-xs font-semibold hit-slop-44 focus-ring {onCover ? 'text-content-inverse hover:underline' : 'text-text-primary hover:text-accent-default hover:underline'} transition-colors"
								onclick={(e) => {
									e.stopPropagation();
									openAuthor();
								}}
							>
								{item.user || '匿名用户'}
							</button>
						{:else}
							<Avatar src={item.avatar} name={item.user} size="xs" />
							<span class="min-w-0 truncate text-xs font-semibold {onCover ? 'text-content-inverse' : 'text-text-primary'}">{item.user || '匿名用户'}</span>
						{/if}
						{#if showFollowChip}
							<span class="pointer-events-auto">
								<FollowButton
									size="chip"
									active={isFollowing}
									pending={followPending}
									onMedia={onCover}
									ariaLabel={(isFollowing ? '取消关注 ' : '关注 ') + item.user}
									onToggle={handleFollowToggle}
								/>
							</span>
						{/if}
					{/if}
				</div>

				<!-- 操作区：遵循极简原则，卡片仅保留点赞与评论；去除未消费前即分享/收藏的冗余 UI -->
				<div class="pointer-events-auto shrink-0 flex items-center">
					<ImageCardActions
						likes={item.likes}
						comments={item.comments}
						{liked}
						saved={saved}
						onImage={onCover}
						onLike={() => onLike()}
						onComment={() => (onCommentClick ?? onClick)?.()}
						onSave={onSave ? () => onSave?.() : undefined}
						size="sm"
					/>
				</div>
			</div>
		</div>
	{/snippet}
</MediaCard>