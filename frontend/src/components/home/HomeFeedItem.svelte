<script lang="ts">
	import { goto } from '$app/navigation';
	import { Heart, MessageCircle, Bookmark, Share2, MapPin, Sparkles, Images } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import type { MomentItem } from '@/api/types';
	import { formatDistance, formatRelativeDays } from '@/lib/utils';
	import { ROUTES } from '@/lib/routes';
	import { showToast } from '@/stores/toast.svelte';

	// ═══ 现代社交 Feed 项（HomeFeedItem）═══
	// 两种形态：
	// - variant="card"（默认，首页/信息流独立高质感卡片）：
	//   顶部完整作者行（头像 + 昵称/时间/推荐 + 位置胶囊）+ 全宽正文 + 杂志画廊式配图 + 标签胶囊 + 底部操作条。
	//   彻底告别传统侧边头像缩进带来的留白失衡，让多图与宽图获得最佳视觉延展。
	// - variant="row"（紧凑列表行）：
	//   用于搜索发现/附近枢纽等紧凑列表容器内。

	let {
		moment,
		onClick,
		showDistance = false,
		ariaLabel,
		variant = 'card'
	}: {
		moment: MomentItem;
		onClick?: () => void;
		showDistance?: boolean;
		ariaLabel?: string;
		variant?: 'card' | 'row';
	} = $props();

	/** 整卡图片卡：封面加载失败时回退主题化渐变底（不压白字）。 */
	let coverError = $state(false);

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

	function handleShareClick(e: MouseEvent) {
		e.stopPropagation();
		if (navigator.share) {
			void navigator.share({
				title: `${moment.user?.nickname ?? 'HotSpot 用户'}的瞬间`,
				text: moment.description,
				url: window.location.origin + ROUTES.moment(moment.moment_id)
			}).catch(() => {});
		} else {
			void navigator.clipboard?.writeText(window.location.origin + ROUTES.moment(moment.moment_id));
			showToast('动态链接已复制到剪贴板', 'success');
		}
	}
</script>

{#if variant === 'card'}
	{@const coverUrl = moment.images?.[0]?.url ?? ''}
	{@const imageCount = moment.images?.length ?? 0}
	{@const useFallback = !coverUrl || coverError}
	<!-- 整卡图片卡（Image-first Card，Round 2）：图片铺满整卡、信息覆盖图面、无外框；
	     顶部遮罩承载作者行，底部遮罩承载标题/元信息/操作行（三主题 token 遮罩）。
	     无图 fallback 用主题化渐变底 + 主题文字色（不使用白字遮罩，已登记例外）。 -->
	<div
		class="group relative aspect-[4/5] w-full cursor-pointer overflow-hidden rounded-lg bg-surface-subtle shadow-xs transition-[box-shadow,transform] duration-standard hover:shadow-raised-md focus-ring"
		role="button"
		tabindex="0"
		onclick={onClick}
		onkeydown={(e) => {
			if (e.key === 'Enter' || e.key === ' ') {
				e.preventDefault();
				onClick?.();
			}
		}}
		aria-label={ariaLabel ?? (moment.user?.nickname ? `查看${moment.user.nickname}的瞬间详情` : (moment.description || '查看瞬间详情'))}
	>
		{#if !useFallback}
			<img
				src={coverUrl}
				alt=""
				loading="lazy"
				class="absolute inset-0 h-full w-full object-cover transition-transform duration-slow group-hover:scale-[1.02]"
				onerror={() => (coverError = true)}
			/>
			<!-- 顶部遮罩 128px（BLOCKER 修复）：作者行两行（昵称 16–36px + 时间 37–52px），
			     96px 时时间行落在 0.64→transparent 陡降区（alpha≈0.49，白字<4.5）；128px 后
			     该行位于 40% 附近 alpha≈0.63，纯白图最坏情况实测 ≥6:1。 -->
			<div class="pointer-events-none absolute inset-x-0 top-0 h-32" style="background:var(--cst-surface-card-scrim-top);"></div>
			<div class="pointer-events-none absolute inset-x-0 bottom-0 h-[58%]" style="background:var(--cst-surface-card-scrim-bottom);"></div>
		{:else}
			<!-- 无图 fallback（登记例外，不压白字）：主题渐变底 + 低透明主题水印 -->
			<div class="absolute inset-0" style="background:var(--cst-surface-card-fallback);"></div>
			<div class="pointer-events-none absolute inset-0 flex items-center justify-center text-accent-default opacity-watermark" aria-hidden="true">
				<Images class="size-16" strokeWidth={1} />
			</div>
		{/if}

		<!-- 顶部：作者行（头像 32 + 白字昵称 semibold + 关注/推荐徽章）与右侧徽章 -->
		<div class="absolute inset-x-0 top-0 flex items-start justify-between gap-3 p-4 sm:p-5">
			<div class="pointer-events-none flex min-w-0 items-center gap-2.5">
				<button
					type="button"
					class="pointer-events-auto hover-action hit-slop shrink-0 rounded-full focus-ring"
					aria-label={`查看${moment.user?.nickname ?? '作者'}的个人主页`}
					onclick={handleUserClick}
				>
					<Avatar
						src={moment.user?.avatar_url ?? null}
						name={moment.user?.nickname ?? '热点网友'}
						size={32}
					/>
				</button>
				<div class="flex min-w-0 flex-col gap-0.5">
					<div class="flex min-w-0 items-center gap-1.5">
						<button
							type="button"
							class="pointer-events-auto truncate text-left text-md font-semibold {useFallback
								? 'text-text-primary'
								: 'text-content-inverse'} hover:underline"
							onclick={handleUserClick}
						>
							{moment.user?.nickname ?? '匿名用户'}
						</button>
						{#if moment.author_relation?.is_following}
							<span
								class="badge h-5 shrink-0 {useFallback
									? 'bg-surface-subtle text-text-secondary'
									: 'bg-surface-media-scrim-strong text-content-inverse'}"
							>
								已关注
							</span>
						{/if}
						{#if recommendationBadge}
							<span
								class="badge h-5 shrink-0 {useFallback
									? 'bg-surface-subtle text-text-secondary'
									: 'bg-surface-media-scrim-strong text-content-inverse'}"
							>
								{recommendationBadge}
							</span>
						{/if}
					</div>
					{#if relativeTime}
						<!-- 时间行提到 opacity-hover（0.90）：BLOCKER 复测中 0.80 时纯白底 min 4.25/4.35，
						     0.90 后字形区实测 ≥5.9；昵称/徽章仍保持 1.0/胶囊底。 -->
						<span class="text-xs {useFallback ? 'text-text-secondary' : 'text-content-inverse opacity-hover'}">
							{relativeTime}
						</span>
					{/if}
				</div>
			</div>

			<div class="pointer-events-none flex shrink-0 items-center gap-1.5">
				{#if imageCount > 1}
					<span class="badge h-5 gap-1 bg-surface-media-scrim-strong text-content-inverse">
						<Images class="size-3" strokeWidth={1.75} aria-hidden="true" />
						1/{imageCount}
					</span>
				{/if}
				{#if locationText}
					<span
						class="badge h-5 max-w-36 gap-1 {useFallback
							? 'bg-surface-subtle text-text-secondary'
							: 'bg-surface-media-scrim-strong text-content-inverse'}"
					>
						<MapPin class="size-3 shrink-0" strokeWidth={1.75} aria-hidden="true" />
						<span class="truncate">{locationText}{distanceLabel ? ` · ${distanceLabel}` : ''}</span>
					</span>
				{/if}
			</div>
		</div>

		<!-- 底部：标题/正文摘要（最多 2 行）+ 元信息 + 操作行 -->
		<div class="absolute inset-x-0 bottom-0 flex flex-col gap-1.5 p-4 sm:p-5">
			{#if moment.description}
				<p class="line-clamp-2 text-base font-semibold leading-snug {useFallback
					? 'text-text-primary'
					: 'text-content-inverse'}">
					{moment.description}
				</p>
			{/if}
			<!-- 元信息：位置只在右上胶囊出现一次（避免同卡重复），此处只留时间 · 标签 -->
			<div class="flex min-w-0 items-center gap-1.5 text-xs {useFallback ? 'text-text-secondary' : 'text-content-inverse opacity-emphasis'}">
				<span class="shrink-0">{relativeTime}</span>
				{#if moment.tags?.[0]}
					<span aria-hidden="true">·</span>
					<span class="min-w-0 truncate">#{moment.tags[0]}</span>
				{/if}
			</div>
			<div class="mt-1 flex items-center justify-between">
				<div class="flex items-center gap-5 sm:gap-6 {useFallback ? 'text-text-secondary' : 'text-content-inverse'}">
					<div
						class="hover-action hit-slop flex items-center gap-1.5 text-sm focus-ring transition-colors {hasLiked
							? 'text-status-danger'
							: ''}"
						aria-label={`点赞 ${totalReactions} 次`}
					>
						<!-- 压图红态：红色填充 + 白色描边（深色遮罩/浅色遮罩上都可辨；形状变化不依赖色相） -->
						<Heart
							class="size-4.5 shrink-0 transition-transform active:scale-125 {hasLiked
								? 'fill-status-danger text-content-inverse'
								: ''}"
							strokeWidth={1.75}
						/>
						<span class="font-medium">{totalReactions > 0 ? totalReactions : '赞'}</span>
					</div>
					<div
						class="hover-action hit-slop flex items-center gap-1.5 text-sm focus-ring transition-colors"
						aria-label={`评论 ${moment.comments_count} 条`}
					>
						<MessageCircle class="size-4.5 shrink-0" strokeWidth={1.75} />
						<span class="font-medium">{moment.comments_count > 0 ? moment.comments_count : '评论'}</span>
					</div>
					<div
						class="hover-action hit-slop flex items-center gap-1.5 text-sm focus-ring transition-colors"
						aria-label={moment.is_favorited ? '已收藏' : '收藏'}
					>
						<Bookmark
							class="size-4.5 shrink-0 transition-transform active:scale-125 {moment.is_favorited
								? 'fill-accent-default text-content-inverse'
								: ''}"
							strokeWidth={1.75}
						/>
					</div>
				</div>
				<button
					type="button"
					class="hover-action hit-slop flex items-center gap-1.5 text-sm focus-ring transition-colors"
					aria-label="分享动态"
					onclick={handleShareClick}
				>
					<Share2 class="size-4.5 shrink-0" strokeWidth={1.75} />
					{#if moment.shares_count && moment.shares_count > 0}
						<span class="font-medium">{moment.shares_count}</span>
					{/if}
				</button>
			</div>
		</div>
	</div>
{:else}
	<!-- 紧凑行形态（保留紧凑内嵌式列表语言） -->
	<div
		class="group relative flex items-start gap-3 sm:gap-3.5 px-4 py-4 sm:px-5 hover:bg-surface-subtle/70 transition-colors cursor-pointer border-b border-border-subtle/30"
		role="button"
		tabindex="0"
		onclick={onClick}
		onkeydown={(e) => {
			if (e.key === 'Enter' || e.key === ' ') {
				e.preventDefault();
				onClick?.();
			}
		}}
		aria-label={ariaLabel ?? (moment.user?.nickname ? `查看${moment.user.nickname}的瞬间详情` : (moment.description || '查看瞬间详情'))}
	>
		<!-- 左侧：作者头像 -->
		<div class="shrink-0 pt-0.5">
			<button
				type="button"
				class="hover-action hit-slop block rounded-full focus-ring cursor-pointer"
				aria-label={`查看${moment.user?.nickname ?? '作者'}的个人主页`}
				onclick={handleUserClick}
			>
				<Avatar
					src={moment.user?.avatar_url ?? null}
					name={moment.user?.nickname ?? '热点网友'}
					size="md"
				/>
			</button>
		</div>

		<!-- 右侧：完整社交信息区 -->
		<div class="min-w-0 flex-1">
			<!-- 头部：作者名 + 关注状态 + 推荐标记 + 相对时间 -->
			<div class="flex items-center justify-between gap-2">
				<div class="flex items-center gap-1.5 min-w-0 flex-wrap">
					<button
						type="button"
						class="font-semibold text-sm text-text-primary hover:text-accent-default hover:underline transition-colors truncate text-left"
						onclick={handleUserClick}
					>
						{moment.user?.nickname ?? '匿名用户'}
					</button>

					{#if moment.author_relation?.is_following}
						<span class="inline-flex items-center px-1.5 py-0.5 rounded text-xxs font-normal bg-accent-soft text-accent-default shrink-0">
							已关注
						</span>
					{/if}

					{#if recommendationBadge}
						<span class="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded text-xxs font-normal bg-surface-subtle text-text-secondary shrink-0">
							<Sparkles class="size-2.5 text-accent-default" />
							{recommendationBadge}
						</span>
					{/if}

					{#if relativeTime}
						<span class="text-text-secondary text-xs shrink-0">· {relativeTime}</span>
					{/if}
				</div>

				<!-- 位置微标签 -->
				{#if locationText}
					<div class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-surface-subtle text-text-secondary text-xs max-w-[140px] sm:max-w-[180px] shrink-0 truncate">
						<MapPin class="size-3 text-accent-default shrink-0" strokeWidth={1.5} />
						<span class="truncate">{locationText}</span>
						{#if distanceLabel}
							<span class="text-text-secondary shrink-0">· {distanceLabel}</span>
						{/if}
					</div>
				{/if}
			</div>

			<!-- 动态正文 -->
			{#if moment.description}
				<p class="mt-1.5 text-sm text-text-primary leading-relaxed whitespace-pre-wrap break-words line-clamp-4">
					{moment.description}
				</p>
			{/if}

			<!-- 配图展示：单图舒展、多图网格 -->
			{#if moment.images && moment.images.length > 0}
				<div class="mt-2.5">
					{#if moment.images.length === 1}
						<div class="overflow-hidden rounded-xl border border-border-subtle/40 max-h-[280px] max-w-[420px] bg-surface-subtle shadow-xs">
							<img
								src={moment.images[0].url}
								alt="动态配图"
								loading="lazy"
								class="w-full h-full object-cover max-h-[280px] hover:scale-[1.01] transition-transform"
							/>
						</div>
					{:else if moment.images.length <= 4}
						<div class="grid grid-cols-2 gap-1.5 max-w-[420px]">
							{#each moment.images.slice(0, 4) as img, idx (img.url + idx)}
								<div class="aspect-square overflow-hidden rounded-xl border border-border-subtle/30 bg-surface-subtle shadow-xs">
									<img
										src={img.url}
										alt={`动态配图 ${idx + 1}`}
										loading="lazy"
										class="w-full h-full object-cover hover:scale-[1.02] transition-transform"
									/>
								</div>
							{/each}
						</div>
					{:else}
						<div class="grid grid-cols-3 gap-1.5 max-w-[460px]">
							{#each moment.images.slice(0, 9) as img, idx (img.url + idx)}
								<div class="aspect-square overflow-hidden rounded-xl border border-border-subtle/30 bg-surface-subtle shadow-xs">
									<img
										src={img.url}
										alt={`动态配图 ${idx + 1}`}
										loading="lazy"
										class="w-full h-full object-cover hover:scale-[1.02] transition-transform"
									/>
								</div>
							{/each}
						</div>
					{/if}
				</div>
			{/if}

			<!-- 标签胶囊 -->
			{#if moment.tags && moment.tags.length > 0}
				<div class="mt-2 flex flex-wrap gap-1.5">
					{#each moment.tags as tag (tag)}
						<button
							type="button"
							class="text-xs text-accent-default hover:underline font-medium hit-slop focus-ring"
							onclick={(e) => handleTagClick(e, tag)}
						>
							#{tag}
						</button>
					{/each}
				</div>
			{/if}

			<!-- 底部轻量交互操作条 -->
			<div class="mt-3 flex items-center gap-6 sm:gap-8 text-text-secondary">
				<!-- 点赞 -->
				<div
					class="hover-action hit-slop flex items-center gap-1.5 text-xs focus-ring transition-colors {hasLiked ? 'text-status-danger' : 'hover:text-status-danger'}"
					aria-label={`点赞 ${totalReactions} 次`}
				>
					<Heart
						class="size-4 shrink-0 transition-transform active:scale-125 {hasLiked ? 'fill-status-danger text-status-danger' : ''}"
						strokeWidth={1.5}
					/>
					<span>{totalReactions > 0 ? totalReactions : '赞'}</span>
				</div>

				<!-- 评论 -->
				<div
					class="hover-action hit-slop flex items-center gap-1.5 text-xs hover:text-accent-default focus-ring transition-colors"
					aria-label={`评论 ${moment.comments_count} 条`}
				>
					<MessageCircle class="size-4 shrink-0" strokeWidth={1.5} />
					<span>{moment.comments_count > 0 ? moment.comments_count : '评论'}</span>
				</div>

				<!-- 分享 -->
				<button
					type="button"
					class="hover-action hit-slop flex items-center gap-1.5 text-xs hover:text-accent-default focus-ring transition-colors ml-auto sm:ml-0"
					aria-label="分享动态"
					onclick={handleShareClick}
				>
					<Share2 class="size-4 shrink-0" strokeWidth={1.5} />
					{#if moment.shares_count && moment.shares_count > 0}
						<span>{moment.shares_count}</span>
					{/if}
				</button>
			</div>
		</div>
	</div>
{/if}
