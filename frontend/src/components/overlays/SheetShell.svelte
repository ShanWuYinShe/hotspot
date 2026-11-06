<script lang="ts">
	import { X } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import * as Dialog from '@/components/ui/dialog';
	import { platform } from '@/stores/platform.svelte';

	// SheetShell — 底部弹层统一外壳（handle + 标题栏 + 关闭按钮 + 内容槽）。
	// 供 CreateGroupSheet / NewConversationSheet 等全屏 bottom sheet 复用，消除逐字重复的外壳标记。
	// handle 支持下拉拖拽关闭（bottom sheet 核心手势；Esc/外点由外层 Dialog 承担）。

	let {
		title,
		onClose,
		headerExtra,
		children
	}: {
		title: string;
		onClose: () => void;
		/** 标题栏标题旁扩展（如「返回发布动态」入口） */
		headerExtra?: Snippet;
		children: Snippet;
	} = $props();

	// 桌面端居中模态复用本壳时不显示拖拽把手：鼠标环境没有下拉手势语义。
	const isDesktop = $derived(platform.platform === 'desktop');

	// 下拉拖拽关闭（P0-3 改造）：
	// 1. 跟手位移：pointer 拖动把手，sheet 逐帧跟随；
	// 2. 速度与距离综合判定：位移 >= 64px 或向下快速甩动（vy >= 0.4px/ms 且位移 > 24px）判定关闭；
	// 3. 顺势退出：判定关闭时直接从当前位移顺势加速滑出屏幕底端，绝不「先弹回 0 再下滑」；
	// 4. 弹性复位：未达关闭阈值时使用 spring + bounce 弹性回弹复位。
	let dragOffset = $state(0);
	let dragging = $state(false);
	let isClosing = $state(false);
	let isSpringingBack = $state(false);
	let springTimer: ReturnType<typeof setTimeout> | number | null = null;

	let startY = 0;
	let lastY = 0;
	let lastTime = 0;
	let velocityY = 0;
	const DRAG_CLOSE_THRESHOLD = 64;

	function onHandlePointerDown(e: PointerEvent) {
		if (e.pointerType === 'mouse' && e.button !== 0) return;
		if (isClosing) return;
		if (springTimer) clearTimeout(springTimer);
		isSpringingBack = false;
		dragging = true;
		startY = e.clientY;
		lastY = e.clientY;
		lastTime = performance.now();
		velocityY = 0;
		(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
	}

	function onHandlePointerMove(e: PointerEvent) {
		if (!dragging) return;
		const now = performance.now();
		const dt = now - lastTime;
		const dy = e.clientY - lastY;
		if (dt > 8) {
			velocityY = dy / dt; // px/ms
			lastY = e.clientY;
			lastTime = now;
		}
		dragOffset = Math.max(0, e.clientY - startY);
	}

	function onHandlePointerUp() {
		if (!dragging) return;
		dragging = false;

		const isFastFlingDown = velocityY > 0.4 && dragOffset > 24;
		const shouldClose = dragOffset >= DRAG_CLOSE_THRESHOLD || isFastFlingDown;

		if (shouldClose) {
			isClosing = true;
			// 顺势向下滑出视口，绝不回弹为 0
			dragOffset = Math.max(dragOffset + 240, 500);
			window.setTimeout(() => {
				onClose();
			}, 140);
		} else {
			// 未达阈值：保持动画层并弹性回位至 0
			isSpringingBack = true;
			dragOffset = 0;
			if (springTimer) clearTimeout(springTimer);
			springTimer = window.setTimeout(() => {
				isSpringingBack = false;
			}, 280);
		}
	}
</script>

<div
	class="flex h-full flex-col bg-surface-primary"
	style={dragOffset > 0 || dragging || isClosing || isSpringingBack
		? `transform:translateY(${dragOffset}px);transition:transform ${
				dragging
					? 'none'
					: isClosing
						? 'var(--cst-motion-rapid) var(--cst-ease-standard)'
						: 'var(--cst-motion-spring) var(--cst-ease-bounce)'
		  };`
		: ''}
>
	<!-- Handle：可拖拽下拉关闭（grab 光标 + 触控手势；把手本身不可聚焦，键盘走关闭钮） -->
	<!-- svelte-ignore a11y_no_static_element_interactions, a11y_click_events_have_key_events -->
	{#if !isDesktop}
		<div
			class="flex shrink-0 justify-center pt-3 pb-1"
			style="cursor:grab;touch-action:none;"
			aria-hidden="true"
			onpointerdown={onHandlePointerDown}
			onpointermove={onHandlePointerMove}
			onpointerup={onHandlePointerUp}
			onpointercancel={onHandlePointerUp}
		>
			<div class="pointer-events-none rounded-full bg-border-default w-9 h-1"></div>
		</div>
	{/if}

	<!-- Header -->
	<div class="flex shrink-0 items-center justify-between gap-2 px-5 py-3 border-b border-border-default">
			<div class="flex min-w-0 items-center gap-2">
				<Dialog.Title class="truncate text-lg font-semibold tracking-title text-text-primary">{title}</Dialog.Title>
				{#if headerExtra}
					{@render headerExtra()}
				{/if}
			</div>
			<Dialog.Close
				class="hover-btn scale-btn hit-slop flex items-center justify-center rounded-full border border-border-default bg-surface-primary size-8 shadow-raised-sm focus-ring"
				aria-label="关闭"
			>
				<X class="size-3.5 text-text-secondary" strokeWidth={2} />
			</Dialog.Close>
	</div>

	{@render children()}
</div>
