<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	// 数据展示组件（data-display）。
	// NearbyImageCard — 图片优先的附近内容卡片统一原语。
	// 对齐 Figma Make data-display/NearbyImageCard 契约（Guidelines 标记为附近内容的唯一卡片实现）：
	//   - 图片 object-cover 填满卡片，底部 tray 承载元数据/操作（不用页面色块垫在图片下）
	//   - hover 仅内部图片微妙缩放（image-hover-frame__image），框体永不移动/缩放
	//   - 底部 tray 用 --surface-card-menu + --border-card-menu，不用裸页面背景
	// 替代 PostCard / FeaturedGroupCard / GroupCard / UserCard 内联重复实现。
	//
	// ┌──────────────── 页面内容区 ────────────┐
	// │ ┌──────── NearbyImageCard ──────┐         │
	// │ │ ┌───────── image ─────────┐   │         │
	// │ │ │  (object-cover)         │   │         │
	// │ │ │  children overlay       │   │         │
	// │ │ └─────────────────────────┘   │         │
	// │ │ ┌── footer tray (card-menu) ─┐│         │
	// │ │ │ 元数据 / 操作              ││         │
	// │ │ └────────────────────────────┘│         │
	// │ └───────────────────────────────┘         │
	// └─────────────────────────────────────────────┘
	// 设备范围：桌面、平板和手机通用。

	let {
		imageSrc,
		imageAlt = '',
		onClick,
		onImageError,
		footer,
		children,
		class: className = ''
	}: {
		imageSrc: string;
		imageAlt?: string;
		onClick?: () => void;
		onImageError?: () => void;
		footer?: Snippet;
		children?: Snippet;
		class?: string;
	} = $props();

	// 对齐 Figma Make NearbyImageCard：hover 时图片缩放 1.055 / 0.32s [0.22,1,0.36,1]
	// （组件内联覆盖共享 CSS 的 1.045/240ms）；整卡按压 scale 由 --card-tap-scale 驱动
	// （vibrant 0.94 / forest 0.97，对齐 Figma useHoverConfig tap）。
	let pressed = $state(false);
	let hovered = $state(false);
	const imageTransform = $derived(hovered ? 'scale(1.055)' : 'scale(1)');
	const cardTransform = $derived(pressed ? 'scale(var(--card-tap-scale, 0.94))' : 'scale(1)');
</script>

<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
<article
	class={cn(
		'image-hover-frame card-hover relative h-full w-full cursor-pointer overflow-hidden',
		className
	)}
	style="transform:{cardTransform};transition:transform 0.10s cubic-bezier(0.22,1,0.36,1);"
	role={onClick ? 'button' : undefined}
	tabindex={onClick ? 0 : undefined}
	onpointerdown={() => (pressed = true)}
	onpointerup={() => (pressed = false)}
	onpointerleave={() => {
		pressed = false;
		hovered = false;
	}}
	onmouseenter={() => (hovered = true)}
	onmouseleave={() => (hovered = false)}
	onclick={onClick}
	onkeydown={(e) => {
		if (onClick && (e.key === 'Enter' || e.key === ' ')) {
			e.preventDefault();
			onClick();
		}
	}}
>
	{#if imageSrc}
		<img
			src={imageSrc}
			alt={imageAlt}
			class="image-hover-frame__image absolute inset-0 h-full w-full object-cover will-change-transform"
			style="transform:{imageTransform};transition:transform 0.32s cubic-bezier(0.22,1,0.36,1);"
			onerror={onImageError}
		/>
	{/if}
	<div class="absolute inset-0" style="background:var(--surface-media-scrim-card);"></div>
	{#if children}
		{@render children()}
	{/if}
	{#if footer}
		<div
			class="absolute inset-x-0 bottom-0 border-t"
			style="background:var(--surface-card-menu);border-color:var(--border-card-menu);"
		>
			{@render footer()}
		</div>
	{/if}
</article>
