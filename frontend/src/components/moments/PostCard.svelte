<script lang="ts">
	import { MapPin, Heart, MessageCircle, Share2, Bookmark } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import type { FeedItem } from './feed-interactions.svelte';

	// PostCard — 动态卡片：图片铺满整卡，标签/作者/正文/操作栏全部以覆盖层
	// 显示在图片上（无图时用主题化占位背景）。
	// dense 用于 BSP 瓦片（高度由布局单元决定）；hero 用于大卡（aspect 4/3 撑高）。

	let {
		item,
		liked,
		saved,
		onLike,
		onSave,
		dense = false,
		onClick,
		hero = false,
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
		edition?: string;
	} = $props();
</script>

<div
	class="neu-hover-card relative h-full w-full overflow-hidden aspect-[4/3] cursor-pointer {hero
		? 'rounded-[22px] neu-surface-hero'
		: 'rounded-[18px] neu-surface-flat'} neu-card-hover"
	style="background:var(--neu-bg);"
	role="button"
	tabindex="0"
	aria-label={item.user ? `查看 ${item.user} 的动态详情` : '查看动态详情'}
	onclick={onClick}
	onkeydown={(e) => {
		if (e.target !== e.currentTarget) return;
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			onClick?.();
		}
	}}
>
	{#if item.image}
		<img src={item.image} alt={item.tag} class="absolute inset-0 h-full w-full object-cover" />
	{:else}
		<!-- 无图：accent 主题渐变占位（底部覆盖层已含正文/作者/操作，此处仅做装饰） -->
		<div
			class="absolute inset-0 flex items-center justify-center"
			style="background:linear-gradient(135deg, color-mix(in srgb, var(--neu-accent-primary) 18%, var(--neu-bg)) 0%, color-mix(in srgb, var(--neu-accent-primary) 5%, var(--neu-bg)) 100%);"
		>
			<div
				class="flex items-center justify-center rounded-[14px]"
				style="width:46px;height:46px;background:color-mix(in srgb, var(--neu-accent-primary) 14%, var(--neu-bg));border:1px solid color-mix(in srgb, var(--neu-accent-primary) 25%, transparent);box-shadow:0 6px 18px color-mix(in srgb, var(--neu-accent-primary) 18%, transparent);"
			>
				<MapPin size={20} class="text-[var(--neu-accent-primary)]" strokeWidth={1.5} />
			</div>
		</div>
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

	<!-- 底部：正文 / 作者与位置 / 操作栏（半透明深色玻璃背景，有图无图都保证可读） -->
	<div class="absolute inset-x-0 bottom-0 bg-[var(--card-scrim)] px-3 pt-2 pb-2.5" style="backdrop-filter:blur(12px);">
		<p class="{dense ? 'line-clamp-1 text-xs' : 'line-clamp-2 text-[13px]'} leading-[1.55]" style="color:rgba(255,255,255,0.92);">
			{item.content}
		</p>

		<div class="mt-1.5 flex items-center gap-2">
			<Avatar src={item.avatar} name={item.user} size={22} style="border:1.5px solid rgba(255,255,255,0.65);" />
			<span class="text-[11px] text-white" style="font-weight:500;">{item.user}</span>
			<div class="flex items-center gap-0.5 ml-auto">
				<MapPin style="width:9px;height:9px;color:rgba(255,255,255,0.8);" strokeWidth={1.5} />
				<span class="text-[10px]" style="color:rgba(255,255,255,0.8);">{item.location}</span>
			</div>
		</div>

		<!-- Actions -->
		<div class="mt-1.5 flex items-center justify-between">
			<div class="flex gap-3.5">
				<button
					type="button"
					class="neu-hover-action flex items-center gap-1.5 outline-none {liked ? 'text-[var(--neu-accent-primary)]' : 'text-white/90'}"
					onclick={(e) => {
						e.stopPropagation();
						onLike();
					}}
				>
					<Heart style="width:15px;height:15px;" strokeWidth={1.5} fill={liked ? 'currentColor' : 'none'} />
					<span class="text-[11px]">{item.likes + (liked ? 1 : 0)}</span>
				</button>
				<button type="button" class="neu-hover-action flex items-center gap-1.5 outline-none text-white/90" onclick={(e) => e.stopPropagation()}>
					<MessageCircle style="width:15px;height:15px;" strokeWidth={1.5} />
					<span class="text-[11px]">{item.comments}</span>
				</button>
			</div>
			<div class="flex gap-3">
				<button
					type="button"
					class="neu-hover-action outline-none {saved ? 'text-[var(--neu-accent-primary)]' : 'text-white/90'}"
					onclick={(e) => {
						e.stopPropagation();
						onSave();
					}}
				>
					<Bookmark style="width:15px;height:15px;" strokeWidth={1.5} fill={saved ? 'currentColor' : 'none'} />
				</button>
				<button type="button" class="neu-hover-action outline-none text-white/90" onclick={(e) => e.stopPropagation()}>
					<Share2 style="width:15px;height:15px;" strokeWidth={1.5} />
				</button>
			</div>
		</div>
	</div>
</div>
