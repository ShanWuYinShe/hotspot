<script lang="ts">
	// Dialog.Content — Portal + Overlay + Content 组合封装。
	// side 决定定位与入场/出场动画（项目弹层视觉的单一权威来源）：
	//   center  → 视口居中模态（圆角 24px + 深阴影）
	//   bottom  → 底部 sheet（圆角顶 28px + 上抛阴影）
	//   right   → 右侧全高面板（左边缘线 + 左抛阴影）
	//   left    → 左侧全高面板
	//   top     → 顶部面板
	// 交互能力（Escape / 外部点击 / 焦点圈 / 滚动锁 / body 滚动锁定）由 bits-ui Dialog.Content 提供。
	import { Dialog } from 'bits-ui';
	import { cn } from '@/lib/utils';

	export type DialogSide = 'center' | 'top' | 'right' | 'bottom' | 'left';

	type Props = Dialog.ContentProps & { side?: DialogSide; overlayClassName?: string };

	let { class: className, overlayClassName = '', side = 'center', children, ...rest }: Props =
		$props();

	const BASE =
		'fixed z-[var(--z-modal)] bg-[var(--neu-bg)] outline-none data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=closed]:animate-out data-[state=closed]:fade-out-0';

	const SIDE_CLASSES: Record<DialogSide, string> = {
		center:
			'left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-[24px] shadow-[0_24px_80px_rgba(0,0,0,0.24),0_0_0_1px_var(--neu-border)] data-[state=open]:zoom-in-95 data-[state=closed]:zoom-out-95',
		bottom:
			'bottom-0 left-0 right-0 rounded-t-[28px] shadow-[0_-8px_40px_rgba(0,0,0,0.18),0_-2px_0_var(--neu-border)] data-[state=open]:slide-in-from-bottom data-[state=closed]:slide-out-to-bottom',
		right:
			'top-0 bottom-0 right-0 w-[min(420px,92vw)] border-l border-[var(--neu-border)] shadow-[-8px_0_40px_rgba(0,0,0,0.18)] data-[state=open]:slide-in-from-right data-[state=closed]:slide-out-to-right',
		left:
			'top-0 bottom-0 left-0 w-[min(420px,92vw)] border-r border-[var(--neu-border)] shadow-[8px_0_40px_rgba(0,0,0,0.18)] data-[state=open]:slide-in-from-left data-[state=closed]:slide-out-to-left',
		top:
			'top-0 left-0 right-0 rounded-b-[28px] shadow-[0_8px_40px_rgba(0,0,0,0.18),0_2px_0_var(--neu-border)] data-[state=open]:slide-in-from-top data-[state=closed]:slide-out-to-top'
	};
</script>

<Dialog.Portal>
	<Dialog.Overlay
		class={cn(
			'fixed inset-0 z-[var(--z-overlay)] bg-[rgba(0,0,0,0.35)] backdrop-blur-[3px] data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=closed]:animate-out data-[state=closed]:fade-out-0',
			overlayClassName
		)}
	/>
	<Dialog.Content class={cn(BASE, SIDE_CLASSES[side], className)} {...rest}>
		{@render children?.()}
	</Dialog.Content>
</Dialog.Portal>
