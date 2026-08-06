<script lang="ts">
	// Dialog.Content — Portal + Overlay + Content 组合封装。
	// side 决定定位与入场/出场动画（项目弹层视觉的单一权威来源）：
	//   center  → 视口居中模态（圆角 24px + 深阴影）
	//   bottom  → 底部 sheet（圆角顶 28px + 上抛阴影）
	//   right   → 右侧全高面板（左边缘线 + 左抛阴影）
	//   left    → 左侧全高面板
	//   top     → 顶部面板
	// 对齐 Figma Make overlays/Sheet：遮罩用更深 scrim（black/50）；side sheet 可选自动右上角 X 关闭按钮。
	// 交互能力（Escape / 外部点击 / 焦点圈 / 滚动锁 / body 滚动锁定）由 bits-ui Dialog.Content 提供。
	import { Dialog } from 'bits-ui';
	import { X } from '@lucide/svelte';
	import { cn } from '@/lib/utils';

	export type DialogSide = 'center' | 'top' | 'right' | 'bottom' | 'left';

	type Props = Dialog.ContentProps & {
		side?: DialogSide;
		overlayClassName?: string;
		/** side sheet 自动渲染右上角 X 关闭按钮（center 永不渲染）。 */
		showCloseButton?: boolean;
	};

	let {
		class: className,
		overlayClassName = '',
		side = 'center',
		showCloseButton = false,
		children,
		...rest
	}: Props = $props();

	const BASE =
		'fixed z-[var(--cst-z-modal)] bg-surface-primary outline-none data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=closed]:animate-out data-[state=closed]:fade-out-0';

	const SIDE_CLASSES: Record<DialogSide, string> = {
		center:
			'left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-feature shadow-[var(--cst-elevation-modal)] data-[state=open]:zoom-in-95 data-[state=closed]:zoom-out-95',
		bottom:
			'bottom-0 left-0 right-0 rounded-t-sheet shadow-[var(--cst-elevation-sheet)] data-[state=open]:slide-in-from-bottom data-[state=closed]:slide-out-to-bottom',
		right:
			'top-0 bottom-0 right-0 w-3/4 border-l border-border-default shadow-[var(--cst-elevation-side-sheet)] data-[state=open]:slide-in-from-right data-[state=closed]:slide-out-to-right sm:max-w-sm',
		left:
			'top-0 bottom-0 left-0 w-3/4 border-r border-border-default shadow-[var(--cst-elevation-side-sheet)] data-[state=open]:slide-in-from-left data-[state=closed]:slide-out-to-left sm:max-w-sm',
		top:
			'top-0 left-0 right-0 rounded-b-sheet shadow-[var(--cst-elevation-sheet)] data-[state=open]:slide-in-from-top data-[state=closed]:slide-out-to-top'
	};

	// 对齐 Figma Make Sheet：遮罩统一用 black/50（比 surface-overlay 更深，与设计一致）。
	const canShowClose = $derived(showCloseButton && side !== 'center');
</script>

<Dialog.Portal>
	<Dialog.Overlay
		class={cn(
			'fixed inset-0 z-[var(--cst-z-overlay)] bg-surface-scrim data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=closed]:animate-out data-[state=closed]:fade-out-0',
			overlayClassName
		)}
	/>
	<Dialog.Content class={cn(BASE, SIDE_CLASSES[side], className)} {...rest}>
		{@render children?.()}
		{#if canShowClose}
			<Dialog.Close
				class="absolute top-4 right-4 z-10 rounded-xs opacity-70 transition-opacity hover:opacity-100 focus-visible:outline-2 focus-visible:outline-accent-default focus-visible:outline-offset-2"
				aria-label="关闭"
			>
				<X class="size-4" strokeWidth={2} />
				<span class="sr-only">Close</span>
			</Dialog.Close>
		{/if}
	</Dialog.Content>
</Dialog.Portal>
