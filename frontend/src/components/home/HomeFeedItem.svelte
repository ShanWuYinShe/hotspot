<script lang="ts">
	import { goto } from '$app/navigation';
	import { MapPin, Sparkles, Images } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import ImageCardActions from '@/components/data-display/ImageCardActions.svelte';
	import { fadeImage, presetImageUrl, presetSrcSet } from '@/lib/images';
	import type { MomentItem } from '@/api/types';
	import { formatDistance, formatRelativeDays } from '@/lib/utils';
	import { ROUTES } from '@/lib/routes';
	import { cardPress } from '@/lib/motion';
	import { setMomentLiked, setMomentSaved, shareMoment } from '@/lib/moment-actions';
	import { requireAuth } from '@/stores/auth.svelte';

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
		onComment,
		showDistance = false,
		ariaLabel,
		priority = false,
		variant = 'card'
	}: {
		moment: MomentItem;
		onClick?: () => void;
		/** 评论入口（缺省回退整卡点击 → 详情页） */
		onComment?: () => void;
		showDistance?: boolean;
		ariaLabel?: string;
		/** 首屏图（feed 前 2 张）：eager + fetchpriority=high + 更高 1x 宽度；其余 lazy */
		priority?: boolean;
		variant?: 'card' | 'row';
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

{#if variant === 'card'}
	{@const coverUrl = moment.images?.[0]?.url ?? ''}
	{@const imageCount = moment.images?.length ?? 0}
	{@const useFallback = !coverUrl || coverError}
	<!-- 整卡图片卡（Image-first Card）：图片铺满整卡、信息覆盖图面、无外框；
	     顶部遮罩承载作者行，底部遮罩承载标题/元信息/操作行（三主题 token 遮罩）。
	     hover 统一走 card-hover（主题化上浮 + elevation-button）与 image-hover-frame__image
	     （图片 1.045 缩放），不再手写 hover:shadow/scale；无图 fallback 用主题化渐变底 +
	     主题文字色（不使用白字遮罩，已登记例外）。 -->
	<div
		class="card-hover isolate group relative aspect-[3/4] w-full overflow-hidden rounded-lg bg-surface-subtle shadow-xs sm:aspect-[4/5] [&_button:not([data-cover])]:relative [&_button:not([data-cover])]:z-20 [&_a]:relative [&_a]:z-20"
		use:cardPress
	>
		{#if !useFallback}
			{#if !coverLoaded}
				<!-- 加载中占位：与 NearbyImageCard 同款主题 accent 渐变，避免纯灰浅带；
				     脉冲限时 3 次后转静态（.media-placeholder--pulse，R17）——旧 animate-pulse
				     在慢图/lazy 卡上会无限常驻；reduced-motion 由全局规则压制。 -->
				<div class="surface-accent-gradient media-placeholder--pulse pointer-events-none absolute inset-0" aria-hidden="true"></div>
			{/if}
			<img
				src={presetImageUrl(coverUrl, 'card', priority ? 900 : 800)}
				srcset={presetSrcSet(coverUrl, 'card')}
				sizes="(min-width: 1024px) 400px, (min-width: 640px) 50vw, 100vw"
				alt=""
				loading={priority ? 'eager' : 'lazy'}
				fetchpriority={priority ? 'high' : 'auto'}
				decoding="async"
				class="image-hover-frame__image absolute inset-0 h-full w-full object-cover"
				use:fadeImage
				onload={() => (coverLoaded = true)}
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
					class="pointer-events-auto hover-action hit-slop-44 shrink-0 rounded-full focus-ring"
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
							class="pointer-events-auto hit-slop-44 truncate text-left text-md font-semibold {useFallback
								? 'text-text-primary'
								: 'text-content-inverse'} hover:underline"
							onclick={handleUserClick}
						>
							{moment.user?.nickname ?? '匿名用户'}
						</button>
						{#if moment.author_relation?.is_following}
							<span class="shrink-0 {useFallback ? 'badge bg-surface-subtle text-text-secondary' : 'media-badge'}">
								已关注
							</span>
						{/if}
						{#if recommendationBadge}
							<span class="shrink-0 {useFallback ? 'badge bg-surface-subtle text-text-secondary' : 'media-badge'}">
								{recommendationBadge}
							</span>
						{/if}
					</div>
					{#if relativeTime}
						<!-- 时间行提到 opacity-hover（0.90）：BLOCKER 复测中 0.80 时纯白底 min 4.25/4.35，
						     0.90 后字形区实测 ≥5.9；昵称/徽章仍保持 1.0/胶囊底。 -->
						<span class="text-sm {useFallback ? 'text-text-secondary' : 'text-content-inverse opacity-hover'}">
							{relativeTime}
						</span>
					{/if}
				</div>
			</div>

			<div class="pointer-events-none flex shrink-0 items-center gap-1.5">
				{#if imageCount > 1}
					<span class="media-badge gap-1">
						<Images class="size-3" strokeWidth={1.75} aria-hidden="true" />
						1/{imageCount}
					</span>
				{/if}
				{#if locationText}
					<span class="max-w-36 gap-1 {useFallback ? 'badge bg-surface-subtle text-text-secondary' : 'media-badge'}">
						<MapPin class="size-3 shrink-0" strokeWidth={1.75} aria-hidden="true" />
						<span class="truncate">{locationText}{distanceLabel ? ` · ${distanceLabel}` : ''}</span>
					</span>
				{/if}
			</div>
		</div>

		<!-- 底部：标题/正文摘要（最多 2 行）+ 元信息 + 操作行 -->
		<div class="absolute inset-x-0 bottom-0 flex flex-col gap-1.5 p-4 sm:p-5">
			{#if moment.description}
				<p class="line-clamp-2 text-base font-semibold leading-tight sm:text-lg {useFallback
					? 'text-text-primary'
					: 'text-content-inverse'}">
					{moment.description}
				</p>
			{/if}
			<!-- 元信息：时间只在作者行出现一次（同卡不重复），此处只留首个标签（12px） -->
			{#if moment.tags?.[0]}
				<div class="flex min-w-0 items-center gap-1.5 text-sm {useFallback ? 'text-text-secondary' : 'text-content-inverse opacity-emphasis'}">
					<span class="min-w-0 truncate">#{moment.tags[0]}</span>
				</div>
			{/if}
			<div class="mt-1">
				<ImageCardActions
					likes={totalReactions}
					comments={moment.comments_count}
					shares={moment.shares_count}
					{liked}
					{saved}
					onImage={!useFallback}
					onLike={(next) => void handleLike(next)}
					onComment={handleCommentClick}
					onSave={(next) => void handleSave(next)}
					onShare={handleShare}
				/>
			</div>
		</div>

		<!-- 整卡点击覆盖层（最后绘制，z-10）：真实 button 承载点击/键盘/aria，
		     行内作者/点赞/评论/收藏/分享按钮经根上 auto-lift(z-20) 保持可点；
		     根不再持 button 语义（axe nested-interactive）。 -->
		<button
			type="button"
			class="absolute inset-0 z-10 cursor-pointer focus-ring"
			data-cover
			aria-label={ariaLabel ?? (moment.user?.nickname ? `查看${moment.user.nickname}的瞬间详情` : (moment.description || '查看瞬间详情'))}
			onclick={onClick}
		></button>
	</div>
{:else}
	<!-- 紧凑行形态（保留紧凑内嵌式列表语言） -->
	<div
		class="group relative isolate flex items-start gap-3 sm:gap-3.5 px-4 py-4 sm:px-5 hover:bg-surface-subtle/70 transition-colors cursor-pointer border-b border-border-subtle/30 [&_button:not([data-cover])]:relative [&_button:not([data-cover])]:z-20 [&_a]:relative [&_a]:z-20"
	>
		<!-- 左侧：作者头像 -->
		<div class="shrink-0 pt-0.5">
			<button
				type="button"
				class="hover-action hit-slop-44 block rounded-full focus-ring cursor-pointer"
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
						<span class="inline-flex items-center px-1.5 py-0.5 rounded text-xxs font-normal bg-accent-soft text-text-accent-on-soft shrink-0">
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
						<div class="overflow-hidden rounded-xl max-h-[280px] max-w-[420px] bg-surface-subtle shadow-xs">
							<img
								src={presetImageUrl(moment.images[0].url, 'card', 800)}
								srcset={presetSrcSet(moment.images[0].url, 'card')}
								sizes="(min-width: 640px) 420px, 100vw"
								alt="动态配图"
								loading="lazy"
								decoding="async"
								class="w-full h-full object-cover max-h-[280px] hover:scale-[1.01] transition-transform"
								use:fadeImage
							/>
						</div>
					{:else if moment.images.length <= 4}
						<div class="grid grid-cols-2 gap-1.5 max-w-[420px]">
							{#each moment.images.slice(0, 4) as img, idx (img.url + idx)}
								<div class="aspect-square overflow-hidden rounded-xl bg-surface-subtle shadow-xs">
									<img
										src={presetImageUrl(img.url, 'thumb', 400)}
										srcset={presetSrcSet(img.url, 'thumb')}
										sizes="(min-width: 640px) 210px, 50vw"
										alt={`动态配图 ${idx + 1}`}
										loading="lazy"
										decoding="async"
										class="w-full h-full object-cover hover:scale-[1.02] transition-transform"
										use:fadeImage
									/>
								</div>
							{/each}
						</div>
					{:else}
						<div class="grid grid-cols-3 gap-1.5 max-w-[460px]">
							{#each moment.images.slice(0, 9) as img, idx (img.url + idx)}
								<div class="aspect-square overflow-hidden rounded-xl bg-surface-subtle shadow-xs">
									<img
										src={presetImageUrl(img.url, 'thumb', 400)}
										srcset={presetSrcSet(img.url, 'thumb')}
										sizes="(min-width: 640px) 140px, 33vw"
										alt={`动态配图 ${idx + 1}`}
										loading="lazy"
										decoding="async"
										class="w-full h-full object-cover hover:scale-[1.02] transition-transform"
										use:fadeImage
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
			<div class="mt-3">
				<ImageCardActions
					likes={totalReactions}
					comments={moment.comments_count}
					shares={moment.shares_count}
					{liked}
					{saved}
					onImage={false}
					onLike={(next) => void handleLike(next)}
					onComment={handleCommentClick}
					onSave={(next) => void handleSave(next)}
					onShare={handleShare}
				/>
			</div>
		</div>

		<!-- 行点击覆盖层（最后绘制，z-10）：真实 button 承载点击/键盘；行内按钮 auto-lift -->
		<button
			type="button"
			class="absolute inset-0 z-10 cursor-pointer focus-ring"
			data-cover
			aria-label={ariaLabel ?? (moment.user?.nickname ? `查看${moment.user.nickname}的瞬间详情` : (moment.description || '查看瞬间详情'))}
			onclick={onClick}
		></button>
	</div>
{/if}