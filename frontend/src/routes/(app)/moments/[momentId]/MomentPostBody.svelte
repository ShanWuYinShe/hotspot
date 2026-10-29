<script lang="ts">
	import { goto } from '$app/navigation';
	import type { Snippet } from 'svelte';
	import { MapPin, Heart, MessageCircle, Share2, Bookmark, Check } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import Badge from '@/components/data-display/Badge.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import LoadMoreButton from '@/components/list/LoadMoreButton.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import ImageViewer from '@/components/overlays/ImageViewer.svelte';
	import { fadeImage, presetImageUrl, presetSrcSet } from '@/lib/images';
	import { enterOnce, popOn } from '@/lib/motion';
	// 点击驱动的 pop 计数（仅点击处理器递增；liked/saved 初始同步不播）
	let likePop = $state(0);
	let savePop = $state(0);
	let commentPops = $state<Record<string, number>>({});
	import { ROUTES } from '@/lib/routes';
	import { platform } from '@/stores/platform.svelte';
	import { auth } from '@/stores/auth.svelte';

	// 动态详情内容体：作者/正文/图片（单图按原比例、多图网格）/操作条/评论区。
	// 评论输入器通过 commentComposer snippet 注入评论区（桌面内嵌）或由页面移动端悬浮 dock 渲染；
	// 本组件只负责内容体与评论区骨架。
	// 页面私有组件（colocate），由 moments/[momentId]/+page.svelte 渲染。
	//
	// R13 视觉改版口径（对齐 MomentCard 已升级语言）：
	// - 作者行 昵称 14px semibold、头像 44/36、位置+时间 11px 次级；正文 14px/1.75 阅读行高；
	// - 标签回归标准胶囊（h-6，无描边无投影，避免「底色+描边+投影」三层堆叠）；
	// - 操作行图标 18 + 计数 12px tabular，点赞态用 accent（与卡片一致，不再用 danger 红）；
	// - 评论行昵称 14 semibold / 正文 14 / 时间 11，回复块落在 soft 底内；
	// - 评论区分区标题改用全站 SectionHead 原语，空态走 EmptyState（裸图标 28）。

	export interface MomentComment {
		/** 真实 comment_id（reactions/set target_id 用） */
		id: string;
		userId: string;
		user: string;
		avatar: string;
		text: string;
		time: string;
		likes: number;
		liked: boolean;
		/** 软删除占位（comments/delete：content 清空，展示「该评论已删除」） */
		isDeleted: boolean;
		/** 一级回复（spec: 两级评论模型；fetch 顶层列表的 top_replies 预览） */
		replies: MomentComment[];
		/** 回复总数（comments/fetch 的 replies_count；> replies.length 表示预览被截断） */
		repliesCount: number;
	}

	export interface MomentData {
		id: string;
		/** 作者 user_id（spec: UserInfo.user_id），用于作者卡跳转。 */
		authorId: string;
		user: string;
		avatar: string;
		location: string;
		/** 全部图片 URL（无图为空数组；首图兼作单图封面） */
		images: string[];
		content: string;
		time: string;
		likes: number;
		tag: string;
		comments: MomentComment[];
	}

	let {
		moment,
		comments,
		commentCount,
		hasMoreComments,
		loadingMoreComments,
		liked,
		saved,
		shareStatus,
		newCommentId = null,
		newReplyId = null,
		onToggleLike,
		onToggleSave,
		onShare,
		onCommentClick,
		onReply,
		onExpandReplies,
		onToggleCommentLike,
		onReportComment,
		onDeleteComment,
		onLoadMoreComments,
		commentComposer
	}: {
		moment: MomentData;
		comments: MomentComment[];
		commentCount: number;
		/** 评论 cursor 分页：还有下一页（null cursor 时不渲染加载更多） */
		hasMoreComments: boolean;
		loadingMoreComments: boolean;
		liked: boolean;
		saved: boolean;
		shareStatus: 'idle' | 'copied' | 'failed';
		/** 本次新增的评论/回复 id：仅它们播放 160ms 进场；初始加载/缓存不播 */
		newCommentId?: string | null;
		newReplyId?: string | null;
		onToggleLike: () => void;
		onToggleSave: () => void;
		onShare: () => void;
		/** 操作条评论计数点击：滚动到评论区并聚焦输入栏 */
		onCommentClick: () => void;
		/** 评论区"回复"点击：携带目标评论 ID，页面聚焦输入栏并 @ 对方（提交带 parent_id） */
		onReply: (user: string, commentId: string) => void;
		/** 「查看全部回复」点击：页面拉取 fetch_replies 全量并回填 */
		onExpandReplies: (commentId: string) => void | Promise<void>;
		/** 评论点赞切换（reactions/set target_type='comment'） */
		onToggleCommentLike: (commentId: string) => void;
		/** 评论举报（reports/create target_type='comment'） */
		onReportComment: (commentId: string) => void;
		/** 删除自己的评论（comments/delete，作者本人；页面负责确认层） */
		onDeleteComment: (commentId: string) => void;
		onLoadMoreComments: () => void;
		/** 桌面/平板：评论区顶部内嵌评论输入器（移动端由页面底部悬浮 dock 承载） */
		commentComposer?: Snippet;
	} = $props();

	// 大图查看器状态
	let viewerOpen = $state(false);
	let viewerIndex = $state(0);

	const cover = $derived(moment.images[0] ?? '');
	// moment.likes 由父页维护（服务端快照 + 本页乐观增量，已含本人状态），
	// 这里不能再按 liked 叠加——否则刷新后计数双倍。
	const likeCount = $derived(moment.likes);
	const isDesktop = $derived(platform.platform === 'desktop');
</script>

<!-- 评论区操作条/评论行共用的轻量动作按钮语言（图标 18 + 12px tabular 计数） -->
{#snippet countText(value: number)}
	<span class="text-sm font-medium tabular-nums">{value}</span>
{/snippet}

<!-- 统一度极高的沉浸式画卷流，消除碎片化盒子嵌套 -->
<article class="flex flex-col overflow-hidden rounded-lg bg-surface-island p-4 sm:p-5 md:p-7 shadow-xs">
	<!-- 1. 画廊首屏（单图：整幅展陈；多图：首图 hero + 缩略图） -->
	{#if cover}
		<div class="pb-5">
			<!-- 画廊展陈：整幅出血到卡片边缘；主图保持原始比例完整展示（object-contain），
			     当主图比例与展台不一致时，用同图放大模糊铺底（ambient matte）填充留白，
			     而不是露出空底色——两侧不再出现「留白条」，暗色主题下也像展墙承托。 -->
			<div class="relative -mx-4 -mt-4 overflow-hidden rounded-lg sm:-mx-5 sm:-mt-5 md:-mx-7 md:-mt-7">
				<div class="relative overflow-hidden rounded-lg bg-surface-subtle">
					<img
						data-gallery-backdrop
						src={presetImageUrl(cover, 'cover', 800)}
						alt=""
						aria-hidden="true"
						loading="lazy"
						decoding="async"
						class="pointer-events-none absolute inset-0 h-full w-full scale-110 object-cover opacity-soft"
						style="filter:blur(var(--cst-blur-lg));"
					/>
					<div class="pointer-events-none absolute inset-0" style="background:var(--cst-surface-media-scrim-card);"></div>
					<button
						type="button"
						class="relative z-10 block w-full cursor-zoom-in focus-ring"
						aria-label="查看大图"
						onclick={() => {
							viewerIndex = 0;
							viewerOpen = true;
						}}
					>
						<img
							data-gallery-image
							src={presetImageUrl(cover, 'cover', 1600)}
							srcset={presetSrcSet(cover, 'cover')}
							sizes="(min-width: 768px) 812px, 100vw"
							alt="动态图片"
							loading="eager"
							fetchpriority="high"
							decoding="async"
							class="block h-auto max-h-[min(78vh,44rem)] w-full object-contain"
							use:fadeImage
						/>
					</button>
					{#if moment.images.length > 1}
						<button
							type="button"
							class="media-badge hover-btn absolute bottom-3 right-3 z-20 shadow-xs hover:opacity-hover focus-ring"
							aria-label={`查看全部 ${moment.images.length} 张图片`}
							onclick={() => {
								viewerIndex = 0;
								viewerOpen = true;
							}}
						>
							共 {moment.images.length} 张
						</button>
					{/if}
				</div>
			</div>
			{#if moment.images.length > 1}
				<div class="mt-2.5 grid grid-cols-3 gap-2">
					{#each moment.images.slice(1) as img, i (img + i)}
						<button
							type="button"
							class="media-placeholder aspect-square w-full overflow-hidden rounded-lg focus-ring shadow-xs"
							aria-label={`查看第 ${i + 2} 张图片`}
							onclick={() => {
								viewerIndex = i + 1;
								viewerOpen = true;
							}}
						>
							<img
								src={presetImageUrl(img, 'thumb', 400)}
								srcset={presetSrcSet(img, 'thumb')}
								sizes="(min-width: 640px) 240px, 33vw"
								alt="动态图片"
								loading="lazy"
								decoding="async"
								class="aspect-square w-full cursor-zoom-in object-cover transition-transform duration-standard hover:scale-105"
								use:fadeImage
							/>
						</button>
					{/each}
				</div>
			{/if}
		</div>
	{:else if !moment.content && moment.location}
		<div class="relative mb-5 flex h-cover-height-lg flex-col items-center justify-center gap-2 overflow-hidden rounded-lg bg-accent-soft/30">
			<MapPin size={28} class="text-accent-default" strokeWidth={1.5} />
			<span class="text-xs text-text-secondary">来自 {moment.location} 的瞬间</span>
		</div>
	{/if}


	<!-- 2. 作者 Header：头像（桌面 44 / 移动 36）+ 昵称 14 semibold + 位置·时间 11 次级 -->
	<header class="flex items-center justify-between gap-3 pb-4">
		<div class="flex min-w-0 items-center gap-3">
			<button
				type="button"
				class="hover-btn hit-slop shrink-0 cursor-pointer overflow-hidden rounded-full p-0 focus-ring {isDesktop ? 'size-11' : 'size-9'}"
				aria-label={`查看${moment.user}的个人主页`}
				onclick={() => { if (moment.authorId) void goto(ROUTES.user(moment.authorId)); }}
			>
				<Avatar src={moment.avatar} name={moment.user} size={isDesktop ? 'lg' : 'md'} />
			</button>
			<div class="min-w-0 flex-1">
				<button
					type="button"
					class="link-btn hit-slop-44 -my-2 block truncate rounded-sm py-2 text-left text-md font-semibold text-text-primary hover:text-accent-default focus-ring"
					onclick={() => { if (moment.authorId) void goto(ROUTES.user(moment.authorId)); }}
				>
					{moment.user}
				</button>
				<div class="mt-0.5 flex items-center gap-1.5 text-sm text-text-secondary">
					{#if moment.location}
						<span class="flex min-w-0 items-center gap-1 font-medium text-accent-default">
							<MapPin class="size-3.5 shrink-0" strokeWidth={1.75} />
							<span class="truncate">{moment.location}</span>
						</span>
						<span class="opacity-muted" aria-hidden="true">·</span>
					{/if}
					<span class="shrink-0 tabular-nums">{moment.time}</span>
				</div>
			</div>
		</div>
	</header>

	<!-- 3. 正文叙述：14px / 1.75 阅读行高（与卡片 12px 摘要拉开层级） -->
	{#if moment.content}
		<p class="select-text whitespace-pre-wrap break-words pb-3 text-md font-normal leading-copy tracking-normal text-text-primary">
			{moment.content}
		</p>
	{/if}

	<!-- 4. 话题标签：标准胶囊档（h-6 + leading-none），只保留一层 accent 淡底 -->
	{#if moment.tag}
		<div class="pb-4">
			<Badge size="md" tone="accent">#{moment.tag}</Badge>
		</div>
	{/if}

	<!-- 5. 交互动作条：所有端都在内容内展示（2026-10-29 移除移动端底部独立动作条，避免与输入器叠成两条横杠） -->
	<div class="flex items-center justify-between py-3">
			<div class="flex items-center gap-6">
				<button
					type="button"
					class="hover-action hit-slop-44 flex items-center gap-1.5 py-1 focus-ring transition-colors {liked ? 'text-accent-default' : 'text-text-secondary hover:text-text-primary'}"
					aria-label={liked ? '取消点赞' : '点赞'}
					aria-pressed={liked}
					onclick={() => {
						if (!liked) likePop += 1;
						onToggleLike();
					}}
				>
					<span class="inline-flex" use:popOn={likePop}>
						<Heart class="size-4.5" strokeWidth={1.75} fill={liked ? 'currentColor' : 'none'} />
					</span>
					{#key likeCount}
						<span class="count-pop text-sm font-medium tabular-nums">{likeCount}</span>
					{/key}
				</button>
				<button
					type="button"
					class="hover-action hit-slop-44 flex items-center gap-1.5 py-1 text-text-secondary transition-colors hover:text-text-primary focus-ring"
					aria-label="查看评论"
					onclick={onCommentClick}
				>
					<MessageCircle class="size-4.5" strokeWidth={1.75} />
					{@render countText(commentCount)}
				</button>
			</div>
			<div class="flex items-center gap-4">
				<button
					type="button"
					class="hover-action hit-slop-44 flex items-center gap-1.5 py-1 focus-ring transition-colors {saved ? 'text-accent-default' : 'text-text-secondary hover:text-text-primary'}"
					aria-label={saved ? '取消收藏' : '收藏'}
					aria-pressed={saved}
					onclick={() => {
						if (!saved) savePop += 1;
						onToggleSave();
					}}
				>
					<span class="inline-flex" use:popOn={savePop}>
						<Bookmark class="size-4.5" strokeWidth={1.75} fill={saved ? 'currentColor' : 'none'} />
					</span>
				</button>
				<button
					type="button"
					onclick={onShare}
					aria-label={shareStatus === 'failed' ? '复制动态链接失败' : '复制动态链接'}
					class="hover-action hit-slop-44 flex items-center py-1 focus-ring transition-colors {shareStatus === 'copied' ? 'text-accent-default' : 'text-text-secondary hover:text-text-primary'}"
				>
					{#if shareStatus === 'copied'}
						<Check class="size-4.5" strokeWidth={2} />
					{:else}
						<Share2 class="size-4.5" strokeWidth={1.75} />
					{/if}
				</button>
			</div>
		</div>

	<!-- 6. 评论讨论区：与动作条之间用一条内容分隔发丝线（非卡片边界），自然在画卷内延展 -->
	<section class="mt-2 border-t border-border-subtle/40 pt-6" id="moment-comments">
		<SectionHead title="评论讨论" count={commentCount} />
		{#if commentComposer}
			<!-- 桌面/平板：输入器内嵌评论区顶部（与卡片同语言），不再固定在视口底部形成独立横条 -->
			<div class="mt-3 mb-4">{@render commentComposer()}</div>
		{/if}
		{#if comments.length === 0}
			<div class="py-4">
				<EmptyState bare title="还没有评论" description="来说点什么吧，成为第一个评论的人。">
					{#snippet icon()}
						<MessageCircle class="size-7" strokeWidth={1.5} />
					{/snippet}
				</EmptyState>
			</div>
		{/if}
		<div class="flex flex-col divide-y divide-border-subtle/30">
			{#each comments as c (c.id)}
				<div class="flex gap-3 py-4 first:pt-0 last:pb-0" use:enterOnce={c.id === newCommentId}>
					<button
						type="button"
						class="hover-btn hit-slop-44 shrink-0 rounded-full focus-ring"
						aria-label={`查看${c.user}的个人主页`}
						onclick={() => { if (c.userId) void goto(ROUTES.user(c.userId)); }}
					>
						<Avatar src={c.avatar} name={c.user} size="md" />
					</button>
					<div class="min-w-0 flex-1">
						<div class="flex items-baseline gap-2">
							<button
								type="button"
								class="link-btn hit-slop-44 -my-2.5 truncate rounded-sm py-2.5 text-left text-md font-semibold text-text-primary focus-ring"
								onclick={() => { if (c.userId) void goto(ROUTES.user(c.userId)); }}
							>{c.user}</button>
							<span class="shrink-0 text-xs tabular-nums text-text-secondary">{c.time}</span>
						</div>
						{#if c.isDeleted}
							<p class="mt-1 text-md italic leading-copy text-text-secondary">该评论已删除</p>
						{:else}
							<p class="mt-1 break-words text-md leading-copy text-text-primary">{c.text}</p>
							<div class="mt-2 flex items-center gap-4">
								<button
									type="button"
									class="hover-action hit-slop-44 -my-2 flex items-center gap-1 py-2 focus-ring transition-colors {c.liked ? 'text-accent-default' : 'text-text-secondary hover:text-text-primary'}"
									aria-label={c.liked ? '取消点赞评论' : '点赞评论'}
									aria-pressed={c.liked}
									onclick={() => {
										if (!c.liked) commentPops = { ...commentPops, [c.id]: (commentPops[c.id] ?? 0) + 1 };
										onToggleCommentLike(c.id);
									}}
								>
									<span class="inline-flex" use:popOn={commentPops[c.id] ?? 0}>
										<Heart class="size-3.5" strokeWidth={1.5} fill={c.liked ? 'currentColor' : 'none'} />
									</span>
									{#key c.likes}
										<span class="count-pop text-sm tabular-nums">{c.likes}</span>
									{/key}
								</button>
								<button
									type="button"
									class="hover-action hit-slop-44 -my-2 py-2 text-sm text-text-secondary transition-colors hover:text-text-primary focus-ring"
									onclick={() => onReply(c.user, c.id)}
								>回复</button>
								{#if c.userId && c.userId === auth.user?.user_id}
									<button
										type="button"
										class="hover-action hit-slop-44 -my-2 py-2 text-sm text-text-secondary transition-colors hover:text-status-danger focus-ring"
										aria-label="删除我的评论"
										onclick={() => onDeleteComment(c.id)}
									>删除</button>
								{:else if c.userId && c.userId !== auth.user?.user_id}
									<button
										type="button"
										class="hover-action desktop-reveal hit-slop-44 -my-2 py-2 text-sm text-text-secondary transition-colors hover:text-status-danger focus-ring"
										aria-label={`举报 ${c.user} 的评论`}
										onclick={() => onReportComment(c.id)}
									>举报</button>
								{/if}
							</div>
						{/if}
						{#if c.replies.length > 0}
							<div class="mt-3 flex flex-col gap-3 rounded-md bg-surface-subtle/50 p-3">
								{#each c.replies as r (r.id)}
									<div class="flex gap-2.5" use:enterOnce={r.id === newReplyId}>
										<button
											type="button"
											class="hover-btn hit-slop-44 shrink-0 rounded-full focus-ring"
											aria-label={`查看${r.user}的个人主页`}
											onclick={() => { if (r.userId) void goto(ROUTES.user(r.userId)); }}
										>
											<Avatar src={r.avatar} name={r.user} size="sm" />
										</button>
										<div class="min-w-0 flex-1">
											<div class="flex flex-wrap items-baseline gap-x-2 gap-y-0.5">
												<button
													type="button"
													class="link-btn hit-slop-44 -my-2 truncate rounded-sm py-2 text-left text-sm font-semibold text-text-primary focus-ring"
													onclick={() => { if (r.userId) void goto(ROUTES.user(r.userId)); }}
												>{r.user}</button>
												{#if !r.isDeleted}
													<span class="text-xs text-text-secondary">回复 @{c.user}</span>
												{/if}
												<span class="shrink-0 text-xs tabular-nums text-text-secondary">{r.time}</span>
											</div>
											{#if r.isDeleted}
												<p class="mt-0.5 text-sm italic leading-relaxed text-text-secondary">该评论已删除</p>
											{:else}
												<p class="mt-0.5 break-words text-sm leading-relaxed text-text-primary">{r.text}</p>
											{/if}
											{#if !r.isDeleted}
												<div class="mt-1 flex items-center gap-3">
													<button
														type="button"
														class="hover-action hit-slop-44 -my-2 py-2 text-sm text-text-secondary transition-colors hover:text-text-primary focus-ring"
														onclick={() => onReply(r.user, c.id)}
													>回复</button>
												</div>
											{/if}
										</div>
									</div>
								{/each}
								{#if c.repliesCount > c.replies.length}
									<button
										type="button"
										class="link-btn self-start rounded-sm pt-1 text-sm font-medium text-accent-default focus-ring"
										onclick={() => onExpandReplies(c.id)}
									>查看全部 {c.repliesCount} 条回复</button>
								{/if}
							</div>
						{/if}
					</div>
				</div>
			{/each}
		</div>
		{#if hasMoreComments}
			<div class="mt-4">
				<LoadMoreButton label="加载更多评论" loading={loadingMoreComments} onLoad={onLoadMoreComments} />
			</div>
		{:else if comments.length > 0}
			<div class="mt-4">
				<ListEnd />
			</div>
		{/if}
	</section>
</article>

<ImageViewer bind:open={viewerOpen} images={moment.images.map((url) => ({ url, alt: '动态图片' }))} bind:index={viewerIndex} />