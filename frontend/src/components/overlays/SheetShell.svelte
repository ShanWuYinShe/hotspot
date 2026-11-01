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

	// 下拉拖拽关闭：pointer 拖动把手 sheet 跟随下移，超过阈值（64px）松手关闭
	let dragOffset = $state(0);
	let dragging = $state(false);
	let startY = 0;
	const DRAG_CLOSE_THRESHOLD = 64;

	function onHandlePointerDown(e: PointerEvent) {
		if (e.pointerType === 'mouse' && e.button !== 0) return;
		dragging = true;
		startY = e.clientY;
		(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
	}
	function onHandlePointerMove(e: PointerEvent) {
		if (!dragging) return;
		dragOffset = Math.max(0, e.clientY - startY);
	}
	function onHandlePointerUp() {
		if (!dragging) return;
		dragging = false;
		const shouldClose = dragOffset >= DRAG_CLOSE_THRESHOLD;
		dragOffset = 0;
		if (shouldClose) onClose();
	}
</script>

<div
	class="flex h-full flex-col bg-surface-primary"
	style={dragOffset || dragging
		? `transform:translateY(${dragOffset}px);transition:transform ${dragging ? 'none' : 'var(--cst-motion-snappy) var(--cst-ease-standard)'};`
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
				<Dialog.Title class="truncate text-lg text-text-primary">{title}</Dialog.Title>
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
