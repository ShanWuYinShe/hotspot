<script lang="ts">
	import { goto } from '$app/navigation';
	import { MapPin, Heart, MessageCircle, Share2, Bookmark, Check, Flag } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import LoadMoreButton from '@/components/list/LoadMoreButton.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import ImageViewer from '@/components/overlays/ImageViewer.svelte';
	import { ROUTES } from '@/lib/routes';
	import { platform } from '@/stores/platform.svelte';
	import { auth } from '@/stores/auth.svelte';

	// 动态详情内容体：作者/正文/图片（单图全幅封面、多图网格）/操作条/评论区。
	// 评论输入栏由页面经 SplitPageLayout footer 渲染，本组件只负责内容体。
	// 页面私有组件（colocate），由 moments/[momentId]/+page.svelte 渲染。

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
		onToggleLike,
		onToggleSave,
		onShare,
		onCommentClick,
		onReply,
		onExpandReplies,
		onToggleCommentLike,
		onReportComment,
		onDeleteComment,
		onLoadMoreComments
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
	} = $props();

	// 大图查看器状态
	let viewerOpen = $state(false);
	let viewerIndex = $state(0);

	const cover = $derived(moment.images[0] ?? '');
	// moment.likes 由父页维护（服务端快照 + 本页乐观增量，已含本人状态），
	// 这里不能再按 liked 叠加——否则刷新后计数双倍。
	const likeCount = $derived(moment.likes);
</script>

<!-- Post images（全幅封面：负 margin 抵消外层 SplitPageLayout 的 gutter，与 users 详情一致）。
     单图保持全幅封面；多图改为网格（首图全幅 + 其余 3 列网格，点击任一张打开大图查看器）。 -->

<!-- 统一度极高的沉浸式画卷流，消除碎片化盒子嵌套 -->
<article class="flex flex-col rounded-lg bg-surface-island p-4 sm:p-5 md:p-7 shadow-xs">
	<!-- 1. 作者 Header -->
	<header class="flex items-center justify-between gap-3 pb-4">
		<div class="flex items-center gap-3 min-w-0">
			<button
				type="button"
				class="hover-btn size-11 shrink-0 cursor-pointer rounded-full border border-border-subtle/80 bg-transparent p-0 overflow-hidden hover:opacity-hover focus-ring focus-visible:ring-offset-surface-primary shadow-xs"
				aria-label={`查看${moment.user}的个人主页`}
				onclick={() => { if (moment.authorId) void goto(ROUTES.user(moment.authorId)); }}
			>
				<Avatar src={moment.avatar} name={moment.user} size="lg" />
			</button>
			<div class="flex-1 min-w-0">
				<button
					type="button"
					class="link-btn text-left text-text-primary text-sm font-semibold truncate hover:text-accent-default focus-ring rounded-sm block"
					onclick={() => { if (moment.authorId) void goto(ROUTES.user(moment.authorId)); }}
				>
					{moment.user}
				</button>
				<div class="flex items-center gap-1.5 mt-0.5 text-text-secondary text-xs">
					{#if moment.location}
						<div class="flex items-center gap-1 truncate text-accent-default font-medium">
							<MapPin class="size-3 shrink-0" strokeWidth={1.75} />
							<span class="truncate">{moment.location}</span>
						</div>
						<span class="text-text-secondary/60">·</span>
					{/if}
					<span class="shrink-0">{moment.time}</span>
				</div>
			</div>
		</div>
	</header>

	<!-- 2. 正文叙述 -->
	{#if moment.content}
		<p class="text-text-primary text-base leading-relaxed whitespace-pre-wrap break-words pb-4 font-normal tracking-normal select-text">
			{moment.content}
		</p>
	{/if}

	<!-- 3. 话题标签 -->
	{#if moment.tag}
		<div class="pb-4">
			<span class="inline-flex items-center px-3 py-1 rounded-full bg-accent-soft text-accent-default border border-accent-border/40 text-xs font-medium shadow-xs">
				#{moment.tag}
			</span>
		</div>
	{/if}

	<!-- 4. 配图画廊（单图全幅封面或多图网格） -->
	{#if cover}
		<div class="pb-5">
			<!-- 媒体容器（例外登记）：去 1px 描边，用 bg-subtle + 圆角 + 阴影表达边界 -->
			<div class="relative overflow-hidden rounded-xl shadow-xs h-cover-height-lg bg-surface-subtle">
				<button
					type="button"
					class="block h-full w-full cursor-zoom-in focus-ring"
					aria-label="查看大图"
					onclick={() => {
						viewerIndex = 0;
						viewerOpen = true;
					}}
				>
					<img src={cover} alt="动态图片" class="w-full h-full object-cover transition-transform duration-standard hover:scale-[1.01]" />
				</button>
				{#if moment.images.length > 1}
					<button
						type="button"
						class="hover-btn absolute right-3 bottom-3 inline-flex h-6 items-center justify-center rounded-full bg-surface-media-scrim-strong px-3 text-xxs font-medium leading-none text-content-inverse hover:opacity-hover focus-ring shadow-xs"
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
			{#if moment.images.length > 1}
				<div class="mt-2.5 grid grid-cols-3 gap-2">
					{#each moment.images.slice(1) as img, i (img + i)}
						<button
							type="button"
							class="aspect-square w-full overflow-hidden rounded-lg focus-ring shadow-xs"
							aria-label={`查看第 ${i + 2} 张图片`}
							onclick={() => {
								viewerIndex = i + 1;
								viewerOpen = true;
							}}
						>
							<img src={img} alt="动态图片" class="aspect-square w-full object-cover cursor-zoom-in hover:scale-105 transition-transform duration-standard" />
						</button>
					{/each}
				</div>
			{/if}
		</div>
	{:else if !moment.content && moment.location}
		<div class="mb-5 relative flex flex-col items-center justify-center gap-2 overflow-hidden rounded-xl bg-accent-soft/30 h-cover-height-lg">
			<MapPin size={28} class="text-accent-default" strokeWidth={1.5} />
			<span class="text-text-secondary text-xs">来自 {moment.location} 的瞬间</span>
		</div>
	{/if}

	<!-- 5. 交互动作条（桌面端行内舒展展示） -->
	{#if platform.platform !== 'mobile'}
		<div class="flex items-center justify-between py-3 border-y border-border-subtle/40">
			<div class="flex items-center gap-6">
				<button
					type="button"
					class="hover-action hit-slop flex items-center gap-1.5 py-1 focus-ring transition-colors {liked ? 'text-status-danger' : 'text-text-secondary hover:text-status-danger'}"
					aria-label={liked ? '取消点赞' : '点赞'}
					aria-pressed={liked}
					onclick={onToggleLike}
				>
					<Heart class="size-4.5" strokeWidth={1.75} fill={liked ? 'currentColor' : 'none'} />
					{#key likeCount}
						<span class="count-pop text-xs font-medium">{likeCount}</span>
					{/key}
				</button>
				<button
					type="button"
					class="hover-action hit-slop flex items-center gap-1.5 py-1 focus-ring text-text-secondary hover:text-accent-default transition-colors"
					aria-label="查看评论"
					onclick={onCommentClick}
				>
					<MessageCircle class="size-4.5" strokeWidth={1.75} />
					<span class="text-xs font-medium">{commentCount}</span>
				</button>
			</div>
			<div class="flex items-center gap-4">
				<button
					type="button"
					class="hover-action hit-slop flex items-center gap-1.5 py-1 focus-ring transition-colors {saved ? 'text-accent-default' : 'text-text-secondary hover:text-accent-default'}"
					aria-label={saved ? '取消收藏' : '收藏'}
					aria-pressed={saved}
					onclick={onToggleSave}
				>
					<Bookmark class="size-4.5" strokeWidth={1.75} fill={saved ? 'currentColor' : 'none'} />
				</button>
				<button
					type="button"
					onclick={onShare}
					aria-label={shareStatus === 'failed' ? '复制动态链接失败' : '复制动态链接'}
					class="hover-action hit-slop flex items-center py-1 focus-ring transition-colors {shareStatus === 'copied' ? 'text-accent-default' : 'text-text-secondary hover:text-accent-default'}"
				>
					{#if shareStatus === 'copied'}
						<Check class="size-4.5" strokeWidth={2} />
					{:else}
						<Share2 class="size-4.5" strokeWidth={1.75} />
					{/if}
				</button>
			</div>
		</div>
	{/if}

	<!-- 6. 评论讨论区：自然在画卷内延展，不再嵌套独立卡片框 -->
	<section class="pt-6" id="moment-comments">
		<h2 class="mb-4 text-sm font-semibold text-text-primary flex items-center gap-2">
			<span>评论讨论</span>
			<span class="text-xs font-normal text-text-secondary bg-surface-subtle px-2 py-0.5 rounded-full">{commentCount}</span>
		</h2>
		{#if comments.length === 0}
			<div class="py-6">
				<EmptyState bare title="还没有评论" description="来说点什么吧">
					{#snippet icon()}
						<MessageCircle class="size-6 opacity-subtle text-text-secondary" strokeWidth={1.5} />
					{/snippet}
				</EmptyState>
			</div>
		{/if}
		<div class="flex flex-col divide-y divide-border-subtle/30">
			{#each comments as c (c.id)}
				<div class="flex gap-3 py-3.5 first:pt-0 last:pb-0">
					<button
						type="button"
						class="hover-btn hit-slop shrink-0 rounded-full focus-ring"
						aria-label={`查看${c.user}的个人主页`}
						onclick={() => { if (c.userId) void goto(ROUTES.user(c.userId)); }}
					>
						<Avatar src={c.avatar} name={c.user} size="md" />
					</button>
					<div class="flex-1 min-w-0">
						<div class="flex items-baseline gap-2">
							<button
								type="button"
								class="link-btn hit-slop -my-2.5 py-2.5 text-text-primary text-sm font-medium focus-ring rounded-sm truncate"
								onclick={() => { if (c.userId) void goto(ROUTES.user(c.userId)); }}
							>{c.user}</button>
							<span class="text-text-secondary text-xs shrink-0">{c.time}</span>
						</div>
						{#if c.isDeleted}
							<p class="text-text-secondary text-sm mt-0.5 leading-relaxed italic">该评论已删除</p>
						{:else}
							<p class="text-text-primary text-md mt-1 leading-relaxed break-words">{c.text}</p>
							<div class="flex items-center gap-4 mt-2">
								<button
									type="button"
									class="hover-action hit-slop -my-2 flex items-center gap-1 py-2 focus-ring transition-colors {c.liked ? 'text-accent-default' : 'text-text-secondary hover:text-text-primary'}"
									aria-label={c.liked ? '取消点赞评论' : '点赞评论'}
									aria-pressed={c.liked}
									onclick={() => onToggleCommentLike(c.id)}
								>
									<Heart class="size-3.5" strokeWidth={1.5} fill={c.liked ? 'currentColor' : 'none'} />
									{#key c.likes}
										<span class="count-pop text-xs">{c.likes}</span>
									{/key}
								</button>
								<button
									type="button"
									class="hover-action hit-slop -my-2 py-2 focus-ring text-text-secondary hover:text-text-primary text-xs"
									onclick={() => onReply(c.user, c.id)}
								>回复</button>
								{#if c.userId && c.userId === auth.user?.user_id}
									<button
										type="button"
										class="hover-action hit-slop -my-2 py-2 focus-ring text-text-secondary hover:text-status-danger text-xs"
										aria-label="删除我的评论"
										onclick={() => onDeleteComment(c.id)}
									>删除</button>
								{:else if c.userId && c.userId !== auth.user?.user_id}
									<button
										type="button"
										class="hover-action hit-slop -my-2 py-2 desktop-reveal focus-ring text-text-secondary hover:text-status-danger text-xs"
										aria-label={`举报 ${c.user} 的评论`}
										onclick={() => onReportComment(c.id)}
									>举报</button>
								{/if}
							</div>
						{/if}
						{#if c.replies.length > 0}
							<div class="mt-3 flex flex-col gap-3 rounded-lg bg-surface-subtle/50 p-3">
								{#each c.replies as r (r.id)}
									<div class="flex gap-2.5">
										<button
											type="button"
											class="hover-btn hit-slop shrink-0 rounded-full focus-ring"
											aria-label={`查看${r.user}的个人主页`}
											onclick={() => { if (r.userId) void goto(ROUTES.user(r.userId)); }}
										>
											<Avatar src={r.avatar} name={r.user} size="sm" />
										</button>
										<div class="flex-1 min-w-0">
											<div class="flex items-baseline gap-2">
												<button
													type="button"
													class="link-btn hit-slop -my-2 py-2 text-text-primary text-sm font-medium focus-ring rounded-sm truncate"
													onclick={() => { if (r.userId) void goto(ROUTES.user(r.userId)); }}
												>{r.user}</button>
												{#if !r.isDeleted}
													<span class="text-text-secondary text-xs">回复 @{c.user}</span>
												{/if}
												<span class="text-text-secondary text-xs shrink-0">{r.time}</span>
											</div>
											{#if r.isDeleted}
												<p class="text-text-secondary text-sm mt-0.5 leading-relaxed italic">该评论已删除</p>
											{:else}
												<p class="text-text-primary text-sm mt-0.5 leading-relaxed break-words">{r.text}</p>
											{/if}
											{#if !r.isDeleted}
												<div class="flex items-center gap-3 mt-1">
													<button
														type="button"
														class="hover-action hit-slop -my-2 py-2 focus-ring text-text-secondary hover:text-text-primary text-xs"
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
										class="link-btn text-accent-default text-sm self-start focus-ring rounded-sm font-medium pt-1"
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
