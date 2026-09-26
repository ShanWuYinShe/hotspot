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
{#if cover}
	<div class="relative overflow-hidden -mx-page-gutter h-cover-height-lg">
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
		<img
			src={cover}
			alt="动态图片"
			class="w-full h-full object-cover cursor-zoom-in"
			onclick={() => {
				viewerIndex = 0;
				viewerOpen = true;
			}}
		/>
		{#if moment.images.length > 1}
			<button
				type="button"
				class="hover-btn absolute right-3 bottom-3 inline-flex h-5 items-center justify-center rounded-full bg-surface-media-scrim-strong px-2.5 text-xxs font-medium leading-none text-content-inverse hover:opacity-hover focus-ring"
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
		<div class="mt-0.5 grid grid-cols-3 gap-0.5">
			{#each moment.images.slice(1) as img, i (img + i)}
				<!-- svelte-ignore a11y_click_events_have_key_events -->
				<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
				<img
					src={img}
					alt={`动态图片 ${i + 2}`}
					loading="lazy"
					class="aspect-square w-full object-cover cursor-zoom-in"
					onclick={() => {
						viewerIndex = i + 1;
						viewerOpen = true;
					}}
				/>
			{/each}
		</div>
	{/if}
{:else}
	<div class="relative flex flex-col items-center justify-center gap-2 overflow-hidden -mx-page-gutter bg-accent-bg h-cover-height-lg">
		<MapPin size={28} class="text-accent-default" strokeWidth={1.5} />
		<span class="text-text-secondary text-xs">来自 {moment.location || '附近'} 的动态</span>
	</div>
{/if}
{#if moment.tag}
	<div class="mt-3 inline-flex px-2.5 py-0.5 rounded-full bg-accent-default shadow-raised-sm" >
		<span class="text-text-on-accent text-xxs">#{moment.tag}</span>
	</div>
{/if}

<div class="pt-4 pb-3">
	<!-- Author -->
	<div class="flex items-center gap-3 mb-3">
		<button
			type="button"
			class="hover-btn size-11 shrink-0 cursor-pointer rounded-full border-2 border-accent-default bg-transparent p-0 overflow-hidden hover:opacity-hover focus-ring focus-visible:ring-offset-surface-primary"
			aria-label={`查看${moment.user}的个人主页`}
			onclick={() => { if (moment.authorId) void goto(ROUTES.user(moment.authorId)); }}
		>
			<Avatar src={moment.avatar} name={moment.user} size="lg" />
		</button>
		<div class="flex-1">
			<p class="text-text-primary text-sm">{moment.user}</p>
			<div class="flex items-center gap-1.5">
				<MapPin class="size-2.5 text-accent-default"   strokeWidth={1.5} />
				<span class="text-text-secondary text-xs">{moment.location}</span>
			</div>
		</div>
		<span class="text-text-secondary text-xs">{moment.time}</span>
	</div>

	<!-- Content -->
	<p class="text-text-primary text-sm leading-copy">{moment.content}</p>

	<!-- Actions（对齐交互契约：行内小图标动作用 hover-action——hover 1.20/按下 0.94）。
	     移动端互动已下沉为页面底部操作条（SplitPageLayout footer），内容区只读 -->
	{#if platform.platform !== 'mobile'}
	<div class="flex items-center justify-between py-3 border-t border-border-default border-b border-border-default" >
		<div class="flex gap-4">
			<button
				type="button"
				class="hover-action hit-slop flex items-center gap-1.5 focus-ring {liked ? 'text-accent-default' : 'text-text-secondary'}"
				aria-label={liked ? '取消点赞' : '点赞'}
				aria-pressed={liked}
				onclick={onToggleLike}
			>
				<Heart class="size-5"  strokeWidth={1.5} fill={liked ? 'currentColor' : 'none'} />
				{#key likeCount}
					<span class="count-pop text-sm">{likeCount}</span>
				{/key}
			</button>
			<button
				type="button"
				class="hover-action hit-slop flex items-center gap-1.5 focus-ring text-text-secondary"
				aria-label="查看评论"
				onclick={onCommentClick}
			>
				<MessageCircle class="size-5"  strokeWidth={1.5} />
				<span class="text-sm">{commentCount}</span>
			</button>
		</div>
		<div class="flex gap-3">
			<button
				type="button"
				class="hover-action hit-slop flex items-center gap-1.5 focus-ring {saved ? 'text-accent-default' : 'text-text-secondary'}"
				aria-label={saved ? '取消收藏' : '收藏'}
				aria-pressed={saved}
				onclick={onToggleSave}
			>
				<Bookmark class="size-5"  strokeWidth={1.5} fill={saved ? 'currentColor' : 'none'} />
			</button>
			<button
				type="button"
				onclick={onShare}
				aria-label={shareStatus === 'failed' ? '复制动态链接失败' : '复制动态链接'}
				class="hover-action hit-slop flex items-center focus-ring {shareStatus === 'copied' ? 'text-accent-default' : 'text-text-secondary'}"
			>
				{#if shareStatus === 'copied'}
					<Check class="size-5"  strokeWidth={1.75} />
				{:else}
					<Share2 class="size-5"  strokeWidth={1.5} />
				{/if}
			</button>
		</div>
	</div>
	{/if}

	<!-- Comments -->
	<p class="eyebrow mt-4 mb-3" id="moment-comments">
		评论 · {commentCount}
	</p>
	{#if comments.length === 0}
		<EmptyState bare title="还没有评论" description="来说点什么吧">
			{#snippet icon()}
				<MessageCircle class="size-6 opacity-subtle text-text-secondary" strokeWidth={1.5} />
			{/snippet}
		</EmptyState>
	{/if}
	<div class="flex flex-col gap-4">
		{#each comments as c (c.id)}
			<div class="flex gap-3">
				<button
					type="button"
					class="hover-btn shrink-0 rounded-full focus-ring"
					aria-label={`查看${c.user}的个人主页`}
					onclick={() => { if (c.userId) void goto(ROUTES.user(c.userId)); }}
				>
					<Avatar src={c.avatar} name={c.user} size="md" />
				</button>
				<div class="flex-1">
					<div class="flex items-baseline gap-2">
						<button
							type="button"
							class="link-btn text-text-primary text-sm focus-ring rounded-sm"
							onclick={() => { if (c.userId) void goto(ROUTES.user(c.userId)); }}
						>{c.user}</button>
						<span class="text-text-secondary text-xs">{c.time}</span>
					</div>
					{#if c.isDeleted}
						<!-- 软删除占位（spec: api_spec_comment.md comments/delete）：内容清空，回复保留 -->
						<p class="text-text-secondary text-sm mt-0.5 leading-relaxed italic">该评论已删除</p>
					{:else}
					<p class="text-text-secondary text-sm mt-0.5 leading-relaxed">{c.text}</p>
					<div class="flex items-center gap-3 mt-1.5">
						<button
							type="button"
							class="hover-action hit-slop flex items-center gap-1 focus-ring {c.liked ? 'text-accent-default' : 'text-text-secondary'}"
							aria-label={c.liked ? '取消点赞评论' : '点赞评论'}
							aria-pressed={c.liked}
							onclick={() => onToggleCommentLike(c.id)}
						>
							<Heart class="size-3"  strokeWidth={1.5} fill={c.liked ? 'currentColor' : 'none'} />
							{#key c.likes}
								<span class="count-pop text-xs">{c.likes}</span>
							{/key}
						</button>
						<button
							type="button"
							class="hover-action hit-slop focus-ring text-text-secondary text-xs"
							onclick={() => onReply(c.user, c.id)}
						>回复</button>
						{#if c.userId && c.userId === auth.user?.user_id}
							<!-- 本人评论可删除（spec: api_spec_comment.md comments/delete，仅作者） -->
							<button
								type="button"
								class="hover-action hit-slop focus-ring text-text-secondary text-xs"
								aria-label="删除我的评论"
								onclick={() => onDeleteComment(c.id)}
							>删除</button>
						{:else if c.userId && c.userId !== auth.user?.user_id}
							<!-- 非本人评论可举报（spec: api_spec_report.md，目标属于当前用户时服务端 10400） -->
							<button
								type="button"
								class="hover-action hit-slop focus-ring text-text-secondary text-xs"
								aria-label={`举报 ${c.user} 的评论`}
								onclick={() => onReportComment(c.id)}
							>举报</button>
						{/if}
					</div>
					{/if}
					{#if c.replies.length > 0}
						<div class="mt-3 flex flex-col gap-3 border-l-2 border-border-default pl-3">
							{#each c.replies as r (r.id)}
								<div class="flex gap-2.5">
									<button
										type="button"
										class="hover-btn shrink-0 rounded-full focus-ring"
										aria-label={`查看${r.user}的个人主页`}
										onclick={() => { if (r.userId) void goto(ROUTES.user(r.userId)); }}
									>
										<Avatar src={r.avatar} name={r.user} size="sm" />
									</button>
									<div class="flex-1">
										<div class="flex items-baseline gap-2">
											<button
												type="button"
												class="link-btn text-text-primary text-sm focus-ring rounded-sm"
												onclick={() => { if (r.userId) void goto(ROUTES.user(r.userId)); }}
											>{r.user}</button>
											{#if !r.isDeleted}
											<span class="text-text-secondary text-xs">回复 @{c.user}</span>
											{/if}
											<span class="text-text-secondary text-xs">{r.time}</span>
										</div>
										{#if r.isDeleted}
											<p class="text-text-secondary text-sm mt-0.5 leading-relaxed italic">该评论已删除</p>
										{:else}
											<p class="text-text-secondary text-sm mt-0.5 leading-relaxed">{r.text}</p>
										{/if}
										{#if !r.isDeleted}
											<div class="flex items-center gap-3 mt-1">
												<button
													type="button"
													class="hover-action hit-slop focus-ring text-text-secondary text-xs"
													onclick={() => onReply(r.user, c.id)}
												>回复</button>
											</div>
										{/if}
									</div>
								</div>
							{/each}
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
		<!-- 分页耗尽契约（frontend/AGENTS.md）：到底必须渲染 ListEnd -->
		<div class="mt-4">
			<ListEnd />
		</div>
	{/if}
</div>

<ImageViewer bind:open={viewerOpen} images={moment.images.map((url) => ({ url, alt: '动态图片' }))} bind:index={viewerIndex} />
