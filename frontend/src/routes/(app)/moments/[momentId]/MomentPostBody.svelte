<script lang="ts">
	import { goto } from '$app/navigation';
	import { MapPin, Heart, MessageCircle, Share2, Bookmark, Send, Check } from '@lucide/svelte';
	import ContentColumn from '@/components/layout/page/ContentColumn.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import { auth } from '@/stores/auth.svelte';
	import { ROUTES } from '@/lib/routes';

	// 动态详情内容体：作者/正文/操作条/评论区 + 底部评论输入栏。
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
		id: number;
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
		input = $bindable(''),
		onToggleLike,
		onToggleSave,
		onShare,
		onComment
	}: {
		moment: MomentData;
		comments: MomentComment[];
		commentCount: number;
		liked: boolean;
		saved: boolean;
		shareStatus: 'idle' | 'copied' | 'failed';
		input?: string;
		onToggleLike: () => void;
		onToggleSave: () => void;
		onShare: () => void;
		onComment: () => void;
	} = $props();
</script>

<!-- Post image -->
<div class="relative overflow-hidden" style="height:clamp(220px, 38vw, 380px);">
	{#if moment.image}
		<img src={moment.image} alt="moment" class="w-full h-full object-cover" />
	{:else}
		<div class="flex h-full w-full flex-col items-center justify-center gap-2 bg-[color-mix(in_srgb,var(--neu-accent-primary)_12%,var(--neu-bg))]">
			<MapPin size={28} class="text-[var(--neu-accent-primary)]" strokeWidth={1.5} />
			<span class="text-[var(--neu-text-muted)] text-xs">来自 {moment.location || '附近'} 的动态</span>
		</div>
	{/if}
	<div class="absolute top-3 left-3 px-2.5 py-0.5 rounded-full bg-[var(--neu-accent-primary)]" style="box-shadow:0 2px 8px var(--neu-glow);">
		<span class="text-[var(--text-on-accent)] text-[10px]">#{moment.tag}</span>
	</div>
</div>

<ContentColumn class="pt-4 pb-3">
	<!-- Author -->
	<div class="flex items-center gap-3 mb-3">
		<button
			type="button"
			class="neu-hover-btn rounded-full overflow-hidden flex-shrink-0 cursor-pointer neu-focus-ring focus-visible:ring-offset-[var(--neu-bg)]"
			style="width:44px;height:44px;padding:0;background:transparent;border:2px solid var(--neu-accent-primary);"
			aria-label={`查看${moment.user}的个人主页`}
			onclick={() => void goto(ROUTES.user(moment.id))}
		>
			<Avatar src={moment.avatar} name={moment.user} size={44} style="border:none;" />
		</button>
		<div class="flex-1">
			<p class="text-[var(--neu-text)] text-[15px]">{moment.user}</p>
			<div class="flex items-center gap-1.5">
				<span class="text-[var(--neu-text-muted)] text-xs">{moment.handle}</span>
				<span class="text-[var(--neu-text-muted)] text-[10px]" style="opacity:0.5;">·</span>
				<MapPin style="width:10px;height:10px;color:var(--neu-accent-primary);" strokeWidth={1.5} />
				<span class="text-[var(--neu-text-muted)] text-[11px]">{moment.location}</span>
			</div>
		</div>
		<span class="text-[var(--neu-text-muted)] text-[11px]">{moment.time}</span>
	</div>

	<!-- Content -->
	<p class="text-[var(--neu-text)] text-sm" style="line-height:1.75;">{moment.content}</p>

	<!-- Actions -->
	<div class="flex items-center justify-between mt-4 py-3" style="border-top:1px solid var(--neu-border);border-bottom:1px solid var(--neu-border);">
		<div class="flex gap-4">
			<button
				type="button"
				class="neu-tap-84 flex items-center gap-1.5 outline-none {liked ? 'text-[var(--neu-accent-primary)]' : 'text-[var(--neu-text-muted)]'}"
				onclick={onToggleLike}
			>
				<Heart style="width:19px;height:19px;" strokeWidth={1.5} fill={liked ? 'currentColor' : 'none'} />
				<span class="text-[13px]">{moment.likes + (liked ? 1 : 0)}</span>
			</button>
			<div class="flex items-center gap-1.5 text-[var(--neu-text-muted)]">
				<MessageCircle style="width:19px;height:19px;" strokeWidth={1.5} />
				<span class="text-[13px]">{commentCount}</span>
			</div>
		</div>
		<div class="flex gap-3">
			<button
				type="button"
				class="neu-tap-84 outline-none {saved ? 'text-[var(--neu-accent-primary)]' : 'text-[var(--neu-text-muted)]'}"
				onclick={onToggleSave}
			>
				<Bookmark style="width:19px;height:19px;" strokeWidth={1.5} fill={saved ? 'currentColor' : 'none'} />
			</button>
			<button
				type="button"
				onclick={onShare}
				aria-label={shareStatus === 'failed' ? '复制动态链接失败' : '复制动态链接'}
				class="neu-tap-84 rounded-full neu-focus-ring {shareStatus === 'copied' ? 'text-[var(--neu-accent-primary)]' : 'text-[var(--neu-text-muted)]'}"
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
	<p class="uppercase mt-4 mb-3 text-[var(--neu-text-muted)] text-[10px]" style="letter-spacing:0.14em;">
		评论 · {commentCount}
	</p>
	<div class="flex flex-col gap-4">
		{#each comments as c (c.id)}
			<div class="flex gap-3">
				<Avatar src={c.avatar} name={c.user} size={34} />
				<div class="flex-1">
					<div class="flex items-baseline gap-2">
						<span class="text-[var(--neu-text)] text-[13px]">{c.user}</span>
						<span class="text-[var(--neu-text-muted)] text-[11px]">{c.time}</span>
					</div>
					<p class="text-[var(--neu-text-muted)] text-[13px]" style="line-height:1.6;margin-top:2px;">{c.text}</p>
					<div class="flex items-center gap-1 mt-1.5">
						<button type="button" class="neu-tap-84 flex items-center gap-1 outline-none text-[var(--neu-text-muted)]">
							<Heart style="width:12px;height:12px;" strokeWidth={1.5} />
							<span class="text-[11px]">{c.likes}</span>
						</button>
						<span class="text-[var(--neu-text-muted)] text-[11px]" style="margin-left:8px;cursor:pointer;">回复</span>
					</div>
				</div>
			</div>
		{/each}
	</div>
	<div style="height:80px;"></div>
</ContentColumn>

<!-- Input bar -->
<div class="flex-shrink-0 flex items-center gap-2.5 px-4 py-3 bg-[var(--neu-bg)]" style="border-top:1px solid var(--neu-border);">
	<Avatar src={auth.user?.avatar_url} name={auth.user?.nickname} size={30} style="border:1px solid var(--neu-border);" />
	<div class="flex-1 flex items-center px-4 rounded-relaxed neu-input-surface" style="height:38px;">
		<input
			bind:value={input}
			onkeydown={(e) => {
				if (e.key === 'Enter') onComment();
			}}
			placeholder="写下你的评论…"
			class="flex-1 bg-transparent outline-none text-[var(--neu-text)] text-[13px]"
		/>
	</div>
	<button
		type="button"
		class="neu-hover-btn neu-tap-88 flex items-center justify-center rounded-full outline-none flex-shrink-0"
		style="width:36px;height:36px;background:{input.trim() ? 'var(--neu-accent-primary)' : 'var(--neu-bg)'};box-shadow:{input.trim() ? '0 4px 14px var(--neu-glow)' : '4px 4px 8px var(--neu-shadow-dark), -4px -4px 8px var(--neu-shadow-light)'};border:1px solid var(--neu-border);transition:all 0.2s;"
		onclick={onComment}
	>
		<Send style="width:14px;height:14px;color:{input.trim() ? 'var(--text-on-accent)' : 'var(--neu-text-muted)'};transform:translateX(1px);" strokeWidth={2} />
	</button>
</div>
