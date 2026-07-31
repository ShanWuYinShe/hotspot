<script lang="ts">
	import { MapPin, Heart, MessageCircle, Share2, Bookmark } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import type { FeedItem } from './feed-interactions.svelte';

	// PostCard — 动态卡片，始终填满容器宽度（无内部 maxWidth）。
	// dense 属性减小内边距/字号（双列网格用）。
	// 迁移自 React MomentsFeedPage.tsx 内 PostCard。

	let {
		item,
		liked,
		saved,
		onLike,
		onSave,
		dense = false,
		onClick,
		hero = false,
		mediaRatio,
		edition
	}: {
		item: FeedItem;
		liked: boolean;
		saved: boolean;
		onLike: () => void;
		onSave: () => void;
		dense?: boolean;
		onClick?: () => void;
		hero?: boolean;
		mediaRatio?: string;
		edition?: string;
	} = $props();
</script>

<div
	class="neu-hover-card {hero
		? 'flex h-full flex-col rounded-[22px] overflow-hidden cursor-pointer w-full neu-surface-hero neu-card-hover'
		: 'flex h-full flex-col rounded-[18px] overflow-hidden cursor-pointer w-full neu-surface-flat neu-card-hover'}"
	onclick={onClick}
>
	<!-- Image — always 16/9 regardless of density -->
	<div class="relative min-h-[170px] flex-1 overflow-hidden" style="aspect-ratio:{mediaRatio ?? '16/9'};">
		{#if item.image}
			<img src={item.image} alt={item.tag} class="h-full w-full object-cover" />
		{:else}
			<div class="flex h-full w-full flex-col justify-between bg-[color-mix(in_srgb,var(--neu-accent-primary)_12%,var(--neu-bg))] p-4">
				<MapPin size={20} class="text-[var(--neu-accent-primary)]" strokeWidth={1.5} />
				<p class="max-w-[78%] text-[15px] leading-6 text-[var(--neu-text)]">来自 {item.location} 的附近动态</p>
			</div>
		{/if}
		{#if item.image}
			<div class="absolute inset-0" style="background:var(--surface-image-scrim);"></div>
		{/if}
		{#if edition}
			<div class="absolute right-3 top-3 z-[1] flex h-7 min-w-7 items-center justify-center rounded-full border border-white/35 bg-black/20 px-1.5 font-mono text-[9px] tracking-[0.12em] text-white backdrop-blur-sm">
				{edition}
			</div>
		{/if}
		<!-- Tag chip -->
		<div class="absolute top-2.5 left-2.5 px-2 py-0.5 rounded-full bg-[var(--neu-accent-primary)]" style="box-shadow:0 2px 8px var(--neu-glow);">
			<span class="text-white text-[10px]">{item.tag}</span>
		</div>
		<!-- Author + location on image -->
		<div class="absolute bottom-0 left-0 right-0 flex items-center gap-2 px-3 pb-2.5">
			<Avatar
				src={item.avatar}
				name={item.user}
				size={22}
				style={item.image ? 'border:1.5px solid rgba(255,255,255,0.65);' : 'border:1.5px solid var(--neu-border);'}
			/>
			<span class="text-[11px]" style="color:{item.image ? 'white' : 'var(--neu-text)'};font-weight:500;">{item.user}</span>
			<div class="flex items-center gap-0.5 ml-auto">
				<MapPin style="width:9px;height:9px;color:{item.image ? 'rgba(255,255,255,0.8)' : 'var(--neu-text-muted)'};" strokeWidth={1.5} />
				<span class="text-[10px]" style="color:{item.image ? 'rgba(255,255,255,0.8)' : 'var(--neu-text-muted)'};">{item.location}</span>
			</div>
		</div>
	</div>

	<!-- Text content -->
	<div style="padding:{dense ? '10px 14px 6px' : '12px 16px 8px'};">
		<p class="line-clamp-2 leading-[1.6] text-[var(--neu-text-muted)] {dense ? 'text-xs' : 'text-[13px]'}">
			{item.content}
		</p>
	</div>

	<!-- Actions -->
	<div class="flex items-center justify-between" style="padding:{dense ? '4px 14px 12px' : '4px 16px 14px'};">
		<div class="flex gap-3.5">
			<button
				type="button"
				class="neu-hover-action flex items-center gap-1.5 outline-none {liked ? 'text-[var(--neu-accent-primary)]' : 'text-[var(--neu-text-muted)]'}"
				onclick={(e) => {
					e.stopPropagation();
					onLike();
				}}
			>
				<Heart style="width:15px;height:15px;" strokeWidth={1.5} fill={liked ? 'currentColor' : 'none'} />
				<span class="text-[11px]">{item.likes + (liked ? 1 : 0)}</span>
			</button>
			<button type="button" class="neu-hover-action flex items-center gap-1.5 outline-none text-[var(--neu-text-muted)]" onclick={(e) => e.stopPropagation()}>
				<MessageCircle style="width:15px;height:15px;" strokeWidth={1.5} />
				<span class="text-[11px]">{item.comments}</span>
			</button>
		</div>
		<div class="flex gap-3">
			<button
				type="button"
				class="neu-hover-action outline-none {saved ? 'text-[var(--neu-accent-primary)]' : 'text-[var(--neu-text-muted)]'}"
				onclick={(e) => {
					e.stopPropagation();
					onSave();
				}}
			>
				<Bookmark style="width:15px;height:15px;" strokeWidth={1.5} fill={saved ? 'currentColor' : 'none'} />
			</button>
			<button type="button" class="neu-hover-action outline-none text-[var(--neu-text-muted)]" onclick={(e) => e.stopPropagation()}>
				<Share2 style="width:15px;height:15px;" strokeWidth={1.5} />
			</button>
		</div>
	</div>
</div>
