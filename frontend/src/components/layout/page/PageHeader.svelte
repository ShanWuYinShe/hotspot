<script lang="ts">
	import type { Snippet } from 'svelte';
	import { ArrowLeft } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import ContentColumn from './ContentColumn.svelte';
	import { platform } from '@/stores/platform.svelte';
	import { cn } from '@/lib/utils';

	// 布局组件（layout/page）。
	// PageHeader — 与正文共用同一 ContentColumn gutter 的页面页头。
	// 对齐 Figma Make layout/page/PageHeader 契约：
	//   - 内容包在 ContentColumn 内，左/右边缘与页面正文严格对齐
	//   - 背景 var(--cst-surface-page)（桌面统一画布契约，sticky 时与滚动内容同底）
	//   - 返回按钮默认仅非桌面显示；showBackOnDesktop 可显式开启
	//
	// ┌──────────────── 路由页面 ──────────────────────────┐
	// │ ┌──────── ContentColumn ────────┐                  │
	// │ │ ←  标题              trailing  │  (sticky 可选)   │
	// │ └───────────────────────────────┘                  │
	// └────────────────────────────────────────────────────┘
	// 设备范围：桌面、平板和手机通用。

	let {
		title,
		subtitle,
		subtitleIcon,
		onBack,
		backLabel = '返回',
		trailing,
		below,
		sticky = true,
		showBackOnDesktop = false,
		maxWidth,
		/** 标题字号（对齐 Figma Make 各页手写 header：消息 22 / 附近动态 22-24 / 详情 16 等）。 */
		titleSize = 18,
		/** 副标题字号（对齐 Figma Make：多数页 12，PageHeader 默认 11）。 */
		subtitleSize = 11,
		/** glass：半透明毛玻璃背景（对齐 Figma nearby 页 sticky header）。 */
		glass = false,
		class: className = '',
		contentClassName = ''
	}: {
		title: string;
		subtitle?: string;
		/** 副标题行内小图标（对齐 Figma nearby/groups 的 MapPin 副标题）。 */
		subtitleIcon?: Snippet;
		onBack?: () => void;
		backLabel?: string;
		trailing?: Snippet;
		below?: Snippet;
		sticky?: boolean;
		showBackOnDesktop?: boolean;
		maxWidth?: number | string;
		titleSize?: number;
		subtitleSize?: number;
		glass?: boolean;
		class?: string;
		/** 内部 ContentColumn 的 className（默认 pt-4 pb-0，对齐样板 nearby/moments；传其他值可覆盖） */
		contentClassName?: string;
	} = $props();

	const showBack = $derived(Boolean(onBack) && (platform.platform !== 'desktop' || showBackOnDesktop));
	// below 存在时（搜索框/Tab 栏等），header 底部预留呼吸：border-bottom 画在 below 内容
	// 之后，与正文拉开间距，避免 Tab 栏紧贴正文。页面 contentClassName 的 pb-0 仍生效于
	// 无 below 场景（纯标题 header）。
	const belowPad = $derived(below ? 'pb-3' : '');
</script>

<header
	class={cn('shrink-0', sticky && 'sticky top-0 z-20', className)}
	style={glass
		? 'background-color:var(--cst-surface-nav-glass);backdrop-filter:blur(var(--cst-blur-md));border-bottom:1px solid var(--cst-border-default);'
		: 'background-color:var(--cst-surface-page);border-bottom:1px solid var(--cst-border-default);'}
>
	<ContentColumn {maxWidth} class={cn('pt-4 pb-0', contentClassName, belowPad)}>
		<div class="flex items-center gap-3">
			{#if showBack}
				<ActionIconButton icon={ArrowLeft} label={backLabel} onClick={onBack} size={36} />
			{/if}
			<div class={cn('min-w-0', trailing ? '' : 'flex-1')}>
				<h1 class="text-text-primary font-medium leading-none" style="font-size:{titleSize}px;">{title}</h1>
				{#if subtitle}
					<p class="mt-1 flex items-center gap-1 text-text-secondary" style="font-size:{subtitleSize}px;">
						{#if subtitleIcon}
							{@render subtitleIcon()}
						{/if}
						{subtitle}
					</p>
				{/if}
			</div>
			{#if trailing}
				{@render trailing()}
			{/if}
		</div>
		{#if below}
			{@render below()}
		{/if}
	</ContentColumn>
</header>
