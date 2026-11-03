<script lang="ts">
	import { goto } from '$app/navigation';
	import { MapPin, Sparkles, Images } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import Badge from '@/components/data-display/Badge.svelte';
	import ImageCardActions from '@/components/data-display/ImageCardActions.svelte';
	import MediaCard from '@/components/data-display/MediaCard.svelte';
	import { fadeImage, presetImageUrl, presetSrcSet } from '@/lib/images';
	import type { MomentItem } from '@/api/types';
	import { formatDistance, formatRelativeDays } from '@/lib/utils';
	import { ROUTES } from '@/lib/routes';
	import { cardPress } from '@/lib/motion';
	import { setMomentLiked, setMomentSaved, shareMoment } from '@/lib/moment-actions';
	import { auth, requireAuth } from '@/stores/auth.svelte';
	import { follow } from '@/api';
	import { showToast } from '@/stores/toast.svelte';
	import FollowButton from '@/components/ui/FollowButton.svelte';

	// ═══ 现代社交 Feed 项（HomeFeedItem）═══
	// 首页/信息流独立高质感卡片：
	//   顶部完整作者行（头像 + 昵称/时间/推荐 + 位置胶囊）+ 全宽正文 + 杂志画廊式配图 + 标签胶囊 + 底部操作条。
	// 彻底告别传统侧边头像缩进带来的留白失衡，让多图与宽图获得最佳视觉延展。

	let {
		moment,
		onClick,
		onComment,
		showDistance = false,
		ariaLabel,
		priority = false
	}: {
		moment: MomentItem;
		onClick?: () => void;
		/** 评论入口（缺省回退整卡点击 → 详情页） */
		onComment?: () => void;
		showDistance?: boolean;
		ariaLabel?: string;
		/** 首屏图（feed 前 2 张）：eager + fetchpriority=high + 更高 1x 宽度；其余 lazy */
		priority?: boolean;
	} = $props();

	/** 整卡图片卡：封面加载失败时回退主题化渐变底（不压白字）。 */
	let coverError = $state(false);
	/** 封面加载完成前 opacity-0，onload 后渐显（reduced-motion 下 transition 由全局规则压至瞬时）。 */
	let coverLoaded = $state(false);
	$effect(() => {
		void moment.images?.[0]?.url;
		coverLoaded = false;
		coverError = false;
	});

	const coverUrl = $derived(moment.images?.[0]?.url ?? '');
	const imageCount = $derived(moment.images?.length ?? 0);
	const useFallback = $derived(!coverUrl || coverError);

	// 格式化相对时间（优先小时/分钟，超出 24h 回退 formatRelativeDays）
	const relativeTime = $derived.by(() => {
		const raw = moment.created_at;
		if (!raw) return '';
		const t = Date.parse(raw);
		if (Number.isNaN(t)) return '';
		const diffMs = Date.now() - t;
		if (diffMs < 0) return '刚刚';
		const diffMin = Math.floor(diffMs / 60_000);
		if (diffMin < 1) return '刚刚';
		if (diffMin < 60) return `${diffMin}分钟前`;
		const diffHours = Math.floor(diffMin / 60);
		if (diffHours < 24) return `${diffHours}小时前`;
		const diffDays = Math.floor(diffHours / 24);
		return formatRelativeDays(diffDays);
	});

	// 位置与距离标签
	const distanceLabel = $derived(showDistance && moment.location?.distance ? formatDistance(moment.location.distance) : '');
	const locationText = $derived(moment.location?.location_name ?? '');

	// 点赞与互动统计
	const totalReactions = $derived(
		moment.reactions?.reduce((sum, r) => sum + (r.count || 0), 0) ?? 0
	);
	const hasLiked = $derived(
		moment.reactions?.some((r) => r.reacted_by_me) ?? false
	);

	// 推荐理由文案映射
	const recommendationBadge = $derived.by(() => {
		switch (moment.recommendation) {
			case 'trending':
				return '热门';
			case 'following':
				return '来自关注';
			case 'nearby':
				return '身边动态';
			case 'friend_liked':
				return '好友赞过';
			case 'tag_match':
				return '兴趣推荐';
			default:
				return null;
		}
	});

	function handleUserClick(e: MouseEvent) {
		e.stopPropagation();
		if (moment.user?.user_id) {
			void goto(ROUTES.user(moment.user.user_id));
		}
	}

	function handleTagClick(e: MouseEvent, tag: string) {
		e.stopPropagation();
		void goto(`${ROUTES.search}?q=${encodeURIComponent('#' + tag)}`);
	}

	// ── 卡片作者关注态（2026-10-29 修复用户反馈）────────────────────────────
	// 「已关注」曾是不可点徽章：pointer-events-none 让点击穿透到整卡覆盖层，
	// 表现为「点已关注却打开了详情」。现在它是真实按钮（FollowButton chip）：
	// 点击 = 取消关注（乐观态 + toast），阻止冒泡，绝不再触发整卡导航。
	// 展示策略（视觉降噪）：默认只在「非本人且已关注」作者卡显示「已关注」chip；
	// 用户在本卡取关后就地变为「关注」（可重新关注）；未关注作者卡不新增常驻按钮。
	// 无论哪态，chip 都是可点按钮，绝不让点击穿透到整卡覆盖层。
	let followOverride = $state<boolean | null>(null);
	let followPending = $state(false);
	$effect(() => {
		void moment.moment_id;
		followOverride = null;
		followPending = false;
	});
	const followTargetId = $derived(moment.user?.user_id ?? '');
	const canFollow = $derived(Boolean(followTargetId) && followTargetId !== auth.user?.user_id);
	const isFollowing = $derived(followOverride ?? Boolean(moment.author_relation?.is_following));
	// 视觉降噪（2026-10-29 评审）：默认只在「已关注」作者卡上出现关注 chip（原设计），
	// 用户在本卡发生过取关后保留「关注」chip（允许就地重新关注），其余未关注作者卡不新增长期按钮，
	// 避免首页每张卡都顶一颗 accent 实底胶囊把图片压花。
	const showFollowChip = $derived(canFollow && (isFollowing || followOverride !== null));

	async function toggleAuthorFollow(next: boolean) {
		if (!followTargetId || followPending) return;
		followPending = true;
		followOverride = next;
		const name = moment.user?.nickname ?? '';
		try {
			await follow({ user_id: followTargetId, active: next });
			showToast(next ? '已关注 ' + name : '已取消关注 ' + name, 'success');
		} catch {
			followOverride = !next;
			showToast(next ? '关注失败，请重试' : '取消关注失败，请重试', 'danger');
		} finally {
			followPending = false;
		}
	}

	function handleFollowToggle(e: MouseEvent) {
		e.stopPropagation();
		if (!followTargetId || followPending) return;
		const next = !isFollowing;
		if (!requireAuth({ action: next ? '关注' : '取消关注', targetId: followTargetId, run: () => void toggleAuthorFollow(next) })) return;
		void toggleAuthorFollow(next);
	}

	// ── 卡片覆盖操作（R14 卡片交互契约）──────────────────────────────────────
	// 与网格卡（MomentCard）共用 lib/moment-actions 的请求/失败语义；本组件持有乐观态，
	// 切换瞬间时复位。登录拦截走 requireAuth（未登录先弹登录，登录后重放该动作）。
	let likedOverride = $state<boolean | null>(null);
	let savedOverride = $state<boolean | null>(null);
	$effect(() => {
		void moment.moment_id;
		likedOverride = null;
		savedOverride = null;
	});
	const liked = $derived(likedOverride ?? hasLiked);
	const saved = $derived(savedOverride ?? Boolean(moment.is_favorited));

	async function handleLike(next: boolean) {
		if (!requireAuth({ action: next ? '点赞' : '取消点赞', targetId: moment.moment_id, run: () => void handleLike(next) })) return;
		likedOverride = next;
		const ok = await setMomentLiked(moment.moment_id, next, moment.reactions?.[0]?.emoji ?? '👍');
		if (!ok) likedOverride = !next;
	}

	async function handleSave(next: boolean) {
		if (!requireAuth({ action: next ? '收藏' : '取消收藏', targetId: moment.moment_id, run: () => void handleSave(next) })) return;
		savedOverride = next;
		const ok = await setMomentSaved(moment.moment_id, next);
		if (!ok) savedOverride = !next;
	}

	function handleCommentClick() {
		if (onComment) {
			onComment();
			return;
		}
		// 与网格卡（MomentCard）一致：评论入口直达详情评论区，而非只打开详情页顶部。
		void goto(`${ROUTES.moment(moment.moment_id)}?focus=comments`);
	}

	function handleShare() {
		void shareMoment({
			momentId: moment.moment_id,
			authorName: moment.user?.nickname,
			text: moment.description
		});
	}
</script>

	<!-- 基于 MediaCard 统一图片优先原语（全站唯一 Image-first 卡片规范）：
	     - 顶部遮罩承载作者行与媒体状态徽章；
	     - 底部遮罩承载标题、首标签与 ImageCardActions 互动行；
	     - 非交互元素（徽章/文字/胶囊/间距）经 data-card-children / data-card-footer 严格 pointer-events-none，
	       点击百分百直透整卡 data-cover，绝无“仅有交互样式却无实际动作”的吃点击现象；
	     - 交互元素（头像/关注/点赞/评论/分享）由卡根统一恢复 auto 并抬升至 z-20 保持独立响应。 -->
	<MediaCard
		imageSrc={coverUrl}
		imageAlt={ariaLabel ?? (moment.user?.nickname ? `查看${moment.user.nickname}的瞬间详情` : (moment.description || '查看瞬间详情'))}
		{priority}
		sizes="(min-width: 1024px) 25rem, (min-width: 640px) 50vw, 100vw"
		{onClick}
		onImageError={() => (coverError = true)}
		class="aspect-[3/4] rounded-lg shadow-xs sm:aspect-[4/5]"
	>
		<!-- 顶部：极简作者行（左侧头像 + 昵称，右侧多图角标；彻底移除时间与地点，移至底部，杜绝重叠） -->
		<div class="pointer-events-none absolute inset-x-0 top-0 flex items-center justify-between gap-2 p-2.5 sm:p-4">
			<div class="pointer-events-none flex min-w-0 items-center gap-2">
				<button
					type="button"
					class="pointer-events-auto hover-action hit-slop-44 shrink-0 rounded-full focus-ring"
					aria-label={`查看${moment.user?.nickname ?? '作者'}的个人主页`}
					onclick={handleUserClick}
				>
					<Avatar
						src={moment.user?.avatar_url ?? null}
						name={moment.user?.nickname ?? '热点网友'}
						size={24}
					/>
				</button>
				<button
					type="button"
					class="pointer-events-auto hit-slop-44 min-h-[1.125rem] truncate text-left text-xs font-semibold sm:text-sm {useFallback
						? 'text-text-primary'
						: 'text-content-inverse'} hover:underline"
					onclick={handleUserClick}
				>
					{moment.user?.nickname ?? '匿名用户'}
				</button>
				{#if showFollowChip}
					<span class="hidden sm:inline-flex">
						<FollowButton
							size="chip"
							active={isFollowing}
							pending={followPending}
							onMedia={!useFallback}
							ariaLabel={(isFollowing ? '取消关注 ' : '关注 ') + (moment.user?.nickname ?? '作者')}
							onToggle={handleFollowToggle}
						/>
					</span>
				{/if}
				{#if recommendationBadge}
					<span class="pointer-events-none hidden sm:inline-flex shrink-0">
						{#if useFallback}
							<Badge tone="neutral" size="sm">{recommendationBadge}</Badge>
						{:else}
							<span class="media-badge">{recommendationBadge}</span>
						{/if}
					</span>
				{/if}
			</div>

			{#if imageCount > 1}
				<div class="pointer-events-none flex shrink-0 items-center">
					<span class="pointer-events-none media-badge gap-0.5 sm:gap-1 text-xxs sm:text-xs whitespace-nowrap">
						<Images class="size-2.5 sm:size-3" strokeWidth={1.75} aria-hidden="true" />
						1/{imageCount}
					</span>
				</div>
			{/if}
		</div>

		<!-- 无图/加载失败降级：§17.2 要求「主题渐变底 + 居中 64px 主题水印图标」——
		     由 MediaCard 在 accent 渐变层内以既有 token（--cst-accent-default / --cst-opacity-watermark）渲染。 -->
		{#snippet watermark()}
			<Sparkles class="size-16" strokeWidth={1.5} aria-hidden="true" />
		{/snippet}

		<!-- 底部信息托盘（由 NearbyImageCard footer 槽渲染，默认即带 pointer-events-none 包装） -->
		{#snippet footer({ onImage, horizontal }: { onImage: boolean; horizontal: boolean })}
			{@const onCover = onImage && !horizontal}
			<div class="pointer-events-none flex flex-col gap-1 p-2 sm:gap-1.5 sm:p-4" data-card-info>
				{#if moment.description}
					<p class="pointer-events-none line-clamp-2 text-md font-semibold leading-tight sm:text-lg {onCover
						? 'text-content-inverse'
						: 'text-text-primary'}">
						{moment.description}
					</p>
				{/if}
				<!-- 元信息单行流：标签 + 地点 + 相对时间，优雅排布，信息一目了然且绝不重叠 -->
				<div class="pointer-events-none flex min-w-0 items-center gap-1.5 text-xxs sm:text-xs {onCover ? 'text-content-inverse opacity-emphasis' : 'text-text-secondary'}">
					{#if moment.tags?.[0]}
						<span class="pointer-events-none shrink-0 font-medium">#{moment.tags[0]}</span>
					{/if}
					{#if locationText}
						{#if moment.tags?.[0]}
							<span class="pointer-events-none opacity-40">·</span>
						{/if}
						<span class="pointer-events-none flex min-w-0 items-center gap-0.5 truncate">
							<MapPin class="size-2.5 shrink-0" strokeWidth={1.75} aria-hidden="true" />
							<span class="truncate">{locationText}{distanceLabel ? ` · ${distanceLabel}` : ''}</span>
						</span>
					{/if}
					{#if relativeTime}
						{#if moment.tags?.[0] || locationText}
							<span class="pointer-events-none opacity-40">·</span>
						{/if}
						<span class="pointer-events-none shrink-0 whitespace-nowrap">{relativeTime}</span>
					{/if}
				</div>
				<div class="mt-0.5">
					<!-- 卡片操作行极简原则：仅保留核心点赞与评论计数，移除未消费即收藏/分享的冗余按钮 -->
					<ImageCardActions
						likes={totalReactions}
						comments={moment.comments_count}
						{liked}
						size="sm"
						onImage={onCover}
						onLike={(next) => void handleLike(next)}
						onComment={handleCommentClick}
					/>
				</div>
			</div>
		{/snippet}
	</MediaCard>
