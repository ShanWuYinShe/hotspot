<script lang="ts">
	import { goto } from '$app/navigation';
	import { MapPin, Heart, MessageCircle, Share2, Bookmark, Check } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import { ROUTES } from '@/lib/routes';

	// 动态详情内容体：作者/正文/操作条/评论区 + 底部评论输入栏。
	// 评论输入栏由页面经 SplitPageLayout footer 渲染，本组件只负责内容体。
	// 页面私有组件（colocate），由 moments/[momentId]/+page.svelte 渲染。

	export interface MomentComment {
		id: number;
		user: string;
		avatar: string;
		text: string;
		time: string;
		likes: number;
	}

	export interface MomentData {
		id: string;
		/** 作者 user_id（spec: UserInfo.user_id），用于作者卡跳转。 */
		authorId: string;
		user: string;
		handle: string;
		avatar: string;
		location: string;
		image: string;
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
		liked,
		saved,
		shareStatus,
		onToggleLike,
		onToggleSave,
		onShare
	}: {
		moment: MomentData;
		comments: MomentComment[];
		commentCount: number;
		liked: boolean;
		saved: boolean;
		shareStatus: 'idle' | 'copied' | 'failed';
		onToggleLike: () => void;
		onToggleSave: () => void;
		onShare: () => void;
	} = $props();
</script>

<!-- Post image（全幅封面：负 margin 抵消外层 SplitPageLayout 的 gutter，与 events/users 一致） -->
<div class="relative overflow-hidden -mx-[var(--page-gutter)]" style="height:clamp(220px, 38vw, 380px);">
	{#if moment.image}
		<img src={moment.image} alt="moment" class="w-full h-full object-cover" />
	{:else}
		<div class="flex h-full w-full flex-col items-center justify-center gap-2 bg-accent-bg">
			<MapPin size={28} class="text-accent-default" strokeWidth={1.5} />
			<span class="text-text-secondary text-xs">来自 {moment.location || '附近'} 的动态</span>
		</div>
	{/if}
	<div class="absolute top-3 left-3 px-2.5 py-0.5 rounded-full bg-accent-default" style="box-shadow:0 2px 8px var(--glow-accent);">
		<span class="text-text-on-accent text-xxs">#{moment.tag}</span>
	</div>
</div>

<div class="pt-4 pb-3">
	<!-- Author -->
	<div class="flex items-center gap-3 mb-3">
		<button
			type="button"
			class="hover-btn rounded-full overflow-hidden shrink-0 cursor-pointer focus-ring focus-visible:ring-offset-surface-primary"
			style="width:44px;height:44px;padding:0;background:transparent;border:2px solid var(--accent-default);"
			aria-label={`查看${moment.user}的个人主页`}
			onclick={() => { if (moment.authorId) void goto(ROUTES.user(moment.authorId)); }}
		>
			<Avatar src={moment.avatar} name={moment.user} size={44} style="border:none;" />
		</button>
		<div class="flex-1">
			<p class="text-text-primary text-[15px]">{moment.user}</p>
			<div class="flex items-center gap-1.5">
				<span class="text-text-secondary text-xs">{moment.handle}</span>
				<span class="text-text-secondary text-xxs" style="opacity:0.5;">·</span>
				<MapPin style="width:10px;height:10px;color:var(--accent-default);" strokeWidth={1.5} />
				<span class="text-text-secondary text-xs">{moment.location}</span>
			</div>
		</div>
		<span class="text-text-secondary text-xs">{moment.time}</span>
	</div>

	<!-- Content -->
	<p class="text-text-primary text-sm" style="line-height:1.75;">{moment.content}</p>

	<!-- Actions -->
	<div class="flex items-center justify-between mt-4 py-3" style="border-top:1px solid var(--border-default);border-bottom:1px solid var(--border-default);">
		<div class="flex gap-4">
			<button
				type="button"
				class="tap-84 flex items-center gap-1.5 outline-none {liked ? 'text-accent-default' : 'text-text-secondary'}"
				onclick={onToggleLike}
			>
				<Heart style="width:19px;height:19px;" strokeWidth={1.5} fill={liked ? 'currentColor' : 'none'} />
				<span class="text-base">{moment.likes + (liked ? 1 : 0)}</span>
			</button>
			<div class="flex items-center gap-1.5 text-text-secondary">
				<MessageCircle style="width:19px;height:19px;" strokeWidth={1.5} />
				<span class="text-base">{commentCount}</span>
			</div>
		</div>
		<div class="flex gap-3">
			<button
				type="button"
				class="tap-84 outline-none {saved ? 'text-accent-default' : 'text-text-secondary'}"
				onclick={onToggleSave}
			>
				<Bookmark style="width:19px;height:19px;" strokeWidth={1.5} fill={saved ? 'currentColor' : 'none'} />
			</button>
			<button
				type="button"
				onclick={onShare}
				aria-label={shareStatus === 'failed' ? '复制动态链接失败' : '复制动态链接'}
				class="tap-84 rounded-full focus-ring {shareStatus === 'copied' ? 'text-accent-default' : 'text-text-secondary'}"
			>
				{#if shareStatus === 'copied'}
					<Check style="width:19px;height:19px;" strokeWidth={1.8} />
				{:else}
					<Share2 style="width:19px;height:19px;" strokeWidth={1.5} />
				{/if}
			</button>
		</div>
	</div>

	<!-- Comments -->
	<p class="uppercase mt-4 mb-3 text-text-secondary text-xxs" style="letter-spacing:0.14em;">
		评论 · {commentCount}
	</p>
	<div class="flex flex-col gap-4">
		{#each comments as c (c.id)}
			<div class="flex gap-3">
				<Avatar src={c.avatar} name={c.user} size={34} />
				<div class="flex-1">
					<div class="flex items-baseline gap-2">
						<span class="text-text-primary text-base">{c.user}</span>
						<span class="text-text-secondary text-xs">{c.time}</span>
					</div>
					<p class="text-text-secondary text-base" style="line-height:1.6;margin-top:2px;">{c.text}</p>
					<div class="flex items-center gap-1 mt-1.5">
						<button type="button" class="tap-84 flex items-center gap-1 outline-none text-text-secondary">
							<Heart style="width:12px;height:12px;" strokeWidth={1.5} />
							<span class="text-xs">{c.likes}</span>
						</button>
						<span class="text-text-secondary text-xs" style="margin-left:8px;cursor:pointer;">回复</span>
					</div>
				</div>
			</div>
		{/each}
	</div>
</div>
