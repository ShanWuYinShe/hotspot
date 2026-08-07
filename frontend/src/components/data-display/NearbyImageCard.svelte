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
		placeholder,
		footer,
		children,
		class: className = ''
	}: {
		imageSrc: string;
		imageAlt?: string;
		onClick?: () => void;
		onImageError?: () => void;
		/** 无图时的占位内容（渲染在 accent 渐变层上；缺省为空白渐变层） */
		placeholder?: Snippet;
		footer?: Snippet;
		children?: Snippet;
		class?: string;
	} = $props();

	// hover 反馈（图片缩放 + accent 边框）完全由 CSS 驱动
	// （.image-hover-frame:hover 规则，单一事实源——JS 状态与 :hover 不同步
	// 会导致鼠标在卡片上滑动时反馈闪烁/丢失）。
	// 整卡按压 scale 由 --cst-card-tap-scale 驱动（vibrant 0.94 / forest 0.97，
	// 对齐 Figma useHoverConfig tap）。
	let pressed = $state(false);
	const cardTransform = $derived(pressed ? 'scale(var(--cst-card-tap-scale, 0.94))' : 'scale(1)');
</script>

<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
<article
	class={cn(
		'image-hover-frame card-hover relative h-full w-full cursor-pointer overflow-hidden',
		className
	)}
	style="transform:{cardTransform};transition:transform 0.10s var(--cst-ease-standard);"
	role={onClick ? 'button' : undefined}
	tabindex={onClick ? 0 : undefined}
	onpointerdown={() => (pressed = true)}
	onpointerup={() => (pressed = false)}
	onpointerleave={() => (pressed = false)}
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
			onerror={onImageError}
		/>
	{:else if placeholder}
		<div class="surface-accent-gradient absolute inset-0 z-[1] flex items-center justify-center">
			{@render placeholder()}
		</div>
	{/if}
	{#if imageSrc}
		<div class="absolute inset-0" style="background:var(--cst-surface-media-scrim-card);"></div>
	{/if}
	{#if children}
		{@render children()}
	{/if}
	{#if footer}
		<div
			class="absolute inset-x-0 bottom-0 border-t"
			style="background:var(--cst-surface-card-menu);border-color:var(--cst-border-card-menu);"
		>
			{@render footer()}
		</div>
	{/if}
</article>
