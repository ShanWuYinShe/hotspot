<script lang="ts">
	import { MapPin, Heart, MessageCircle, Bookmark } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import NearbyImageCard from '@/components/data-display/NearbyImageCard.svelte';
	import type { FeedItem } from './feed-interactions.svelte';

	// PostCard — 动态卡片：基于 NearbyImageCard 统一图片优先原语。
	// 标签/作者/正文/操作栏全部以覆盖层显示在图片上（无图时用主题化占位背景）。
	// dense 用于 BSP 瓦片（高度由布局单元决定）；hero 用于大卡（aspect 4/3 撑高）。

	let {
		item,
		liked,
		saved,
		onLike,
		onSave,
		dense = false,
		onClick,
		hero = false
	}: {
		item: FeedItem;
		liked: boolean;
		saved: boolean;
		onLike: () => void;
		onSave: () => void;
		dense?: boolean;
		onClick?: () => void;
		hero?: boolean;
	} = $props();
</script>

<NearbyImageCard
	imageSrc={item.image ?? item.avatar}
	imageAlt={item.tag}
	{onClick}
	class="aspect-[4/3] {hero ? 'rounded-[22px] surface-hero' : 'rounded-[18px] surface-flat'}"
>
	<!-- 无图时占位：覆盖在图片层之上（NearbyImageCard 已渲染 item.image，此处仅在无图时遮挡） -->
	{#if !item.image}
		<div
			class="absolute inset-0 z-[1] flex items-center justify-center"
			style="background:linear-gradient(135deg, color-mix(in srgb, var(--accent-default) 18%, var(--surface-primary)) 0%, color-mix(in srgb, var(--accent-default) 5%, var(--surface-primary)) 100%);"
		>
			<div
				class="flex items-center justify-center rounded-[14px]"
				style="width:46px;height:46px;background:color-mix(in srgb, var(--accent-default) 14%, var(--surface-primary));border:1px solid color-mix(in srgb, var(--accent-default) 25%, transparent);box-shadow:0 6px 18px color-mix(in srgb, var(--accent-default) 18%, transparent);"
			>
				<MapPin size={20} class="text-[var(--accent-default)]" strokeWidth={1.5} />
			</div>
		</div>
	{/if}

	<!-- Tag chip -->
	<div class="absolute top-2.5 left-2.5 z-[2] px-2 py-0.5 rounded-full bg-[var(--accent-default)]" style="box-shadow:0 2px 8px var(--glow-accent);">
		<span class="text-white text-[10px]">{item.tag}</span>
	</div>

	<!-- 底部信息托盘（surface-card-menu，由 NearbyImageCard footer 槽渲染） -->
	{#snippet footer()}
		<div class="px-3 pb-2.5 pt-2.5">
			<!-- 作者行：头像 + 昵称 + 位置（对齐 Figma Make PostCard footer 顺序） -->
			<div class="flex min-w-0 items-center gap-2">
				<Avatar src={item.avatar} name={item.user} size={22} style="border:1.5px solid var(--border-card-menu);" />
				<span class="truncate text-[11px] text-[var(--text-primary)]" style="font-weight:500;">{item.user}</span>
				<span class="ml-auto flex min-w-0 items-center gap-0.5 truncate text-[10px]" style="color:var(--text-secondary);">
					<MapPin style="width:9px;height:9px;flex-shrink:0;color:var(--text-secondary);" strokeWidth={1.5} />
					{item.location}
				</span>
			</div>

			<!-- 正文：单行省略，muted（对齐 Figma Make：fontSize 11 / muted） -->
			<p class="mt-1 line-clamp-1 leading-[1.55] text-[11px]" style="color:var(--text-secondary);">
				{item.content}
			</p>

			<!-- Actions -->
			<div class="mt-1.5 flex items-center justify-between">
				<div class="flex gap-3.5">
					<button
						type="button"
						class="hover-action flex items-center gap-1.5 outline-none {liked ? 'text-[var(--accent-default)]' : 'text-[var(--text-secondary)]'}"
						onclick={(e) => {
							e.stopPropagation();
							onLike();
						}}
					>
						<Heart style="width:15px;height:15px;" strokeWidth={1.5} fill={liked ? 'currentColor' : 'none'} />
						<span class="text-[11px]">{item.likes + (liked ? 1 : 0)}</span>
					</button>
					<span class="flex items-center gap-1.5 text-[11px] text-[var(--text-secondary)]">
						<MessageCircle style="width:15px;height:15px;" strokeWidth={1.5} />
						{item.comments}
					</span>
				</div>
				<div class="flex gap-3">
					<button
						type="button"
						class="hover-action outline-none {saved ? 'text-[var(--accent-default)]' : 'text-[var(--text-secondary)]'}"
						onclick={(e) => {
							e.stopPropagation();
							onSave();
						}}
					>
						<Bookmark style="width:15px;height:15px;" strokeWidth={1.5} fill={saved ? 'currentColor' : 'none'} />
					</button>
				</div>
			</div>
		</div>
	{/snippet}
</NearbyImageCard>
