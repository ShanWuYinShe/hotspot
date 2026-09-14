<script lang="ts">
	// Dialog.Content — Portal + Overlay + Content 组合封装。
	// side 决定定位与入场/出场动画（项目弹层视觉的单一权威来源）：
	//   center  → 视口居中模态（圆角 24px + 深阴影）
	//   bottom  → 底部 sheet（圆角顶 28px + 上抛阴影）
	//   right   → 右侧全高面板（左边缘线 + 左抛阴影）
	//   left    → 左侧全高面板
	//   top     → 顶部面板
	// center 定位用 inset + margin:auto（无 translate）：translate 定位会被调用方
	// 的 inset-x-* 覆盖类经 tailwind-merge 撕裂（left 被删而 translate 残留→左移半宽），
	// 且 zoom 动画的 keyframes transform 会整体覆盖 translate 造成入场位置跳动。
	// 调用方只需传宽度（max-w-*/w-*）与内距，定位由组件统一承担。
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
		'fixed z-[var(--cst-z-modal)] bg-surface-primary focus-ring data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=closed]:animate-out data-[state=closed]:fade-out-0';

	const SIDE_CLASSES: Record<DialogSide, string> = {
		center:
			'inset-x-4 inset-y-4 m-auto h-fit rounded-md border border-border-default/80 shadow-modal data-[state=open]:zoom-in-95 data-[state=closed]:zoom-out-95',
		bottom:
			// sm 起居中限宽：手机保持全宽 bottom sheet 手势形态；桌面/平板全宽 sheet
			// 会把表单输入框/按钮拉满视口（设置类弹层尤其难看）。需要更宽容器的
			// 页面型 sheet（发布编辑器/聊天容器）由调用方传 sm:max-w-* 覆盖（cn 走 twMerge）。
			'bottom-0 left-0 right-0 sm:max-w-xl sm:mx-auto rounded-t-sheet border-t border-border-default/80 shadow-sheet data-[state=open]:slide-in-from-bottom data-[state=closed]:slide-out-to-bottom',
		right:
			'top-0 bottom-0 right-0 w-3/4 border-l border-border-default/80 shadow-side-sheet data-[state=open]:slide-in-from-right data-[state=closed]:slide-out-to-right sm:max-w-sm',
		left:
			'top-0 bottom-0 left-0 w-3/4 border-r border-border-default/80 shadow-side-sheet data-[state=open]:slide-in-from-left data-[state=closed]:slide-out-to-left sm:max-w-sm',
		top:
			'top-0 left-0 right-0 rounded-b-sheet border-b border-border-default/80 shadow-sheet data-[state=open]:slide-in-from-top data-[state=closed]:slide-out-to-top'
	};

	// 遮罩走 --cst-surface-scrim（themes/*.css 按主题/明暗定义，vibrant≈0.38-0.42），叠加微虚化提升景深。
	const canShowClose = $derived(showCloseButton && side !== 'center');
</script>

<Dialog.Portal>
	<Dialog.Overlay
		class={cn(
			'fixed inset-0 z-[var(--cst-z-overlay)] bg-surface-scrim/80 backdrop-blur-sm data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=closed]:animate-out data-[state=closed]:fade-out-0',
			overlayClassName
		)}
	/>
	<Dialog.Content class={cn(BASE, SIDE_CLASSES[side], className)} {...rest}>
		{@render children?.()}
		{#if canShowClose}
			<Dialog.Close
				class="hover-action focus-ring absolute top-3 right-3 z-10 flex size-8 items-center justify-center rounded-full text-text-secondary transition-opacity duration-rapid ease-standard hover:text-text-primary"
				aria-label="关闭"
			>
				<X class="size-4" strokeWidth={2} />
			</Dialog.Close>
		{/if}
	</Dialog.Content>
</Dialog.Portal>
