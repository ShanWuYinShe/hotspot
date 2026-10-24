<script lang="ts">
	import { goto } from '$app/navigation';
	import { Heart, MessageCircle, Share2, MapPin, Sparkles } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import type { MomentItem } from '@/api/types';
	import { formatDistance, formatRelativeDays } from '@/lib/utils';
	import { ROUTES } from '@/lib/routes';

	// ═══ 现代社交 Feed 项（HomeFeedItem）═══
	// 左侧作者头像 + 右侧完整社交实体（昵称/时间/位置/正文/图集/标签/互动栏）。
	// 彻底替代过往单行抽象色块菜单（ActivityRow），使首页呈现有温度、真实的社区氛围。

	let {
		moment,
		onClick,
		showDistance = false,
		ariaLabel
	}: {
		moment: MomentItem;
		onClick?: () => void;
		showDistance?: boolean;
		ariaLabel?: string;
	} = $props();

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
		}
	}
</script>

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
