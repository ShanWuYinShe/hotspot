<script lang="ts">
	import { Dialog } from 'bits-ui';
	import { ChevronLeft, ChevronRight, X } from '@lucide/svelte';

	// ImageViewer — 全屏大图查看器（对齐 harmonyapp PhotoPreviewPage 的交互集合的 Web 版）。
	// 能力：多图轮播（按钮/键盘/未放大时横向滑动）、单击图片切换放大（以点击位置为锚，
	// 对齐 Google Photos 网页版习惯）、滚轮连续缩放（1x~4x）、放大后拖拽平移、
	// 缩略图条与页码、Esc/点击背景关闭。
	// 底座用 bits-ui Dialog：Escape、外部点击、焦点圈、滚动锁由其统一提供，
	// 入出场动画沿用项目弹层契约（fade + zoom）。
	export interface ViewerImage {
		url: string;
		alt?: string;
	}

	let {
		open = $bindable(false),
		images = [],
		index = $bindable(0),
		onClose
	}: {
		open?: boolean;
		images: ViewerImage[];
		index?: number;
		onClose?: () => void;
	} = $props();

	// 缩放/平移状态（切换图片时重置）。originX/Y 是缩放锚点（图片内百分比），
	// 单击/滚轮缩放都锚定在指针位置，复位回中心。
	let scale = $state(1);
	let originX = $state(50);
	let originY = $state(50);
	let panX = $state(0);
	let panY = $state(0);
	let dragging = $state(false);
	// 指针拖拽起点（缩放态平移 / 常规态横向滑动切图共用）
	let dragStart = { x: 0, y: 0, panX: 0, panY: 0, moved: false };
	// 拖拽/滑动结束后的 click 不再触发缩放（click 在 pointerup 后补发）
	let suppressClick = false;

	const MAX_SCALE = 4;
	const count = $derived(images.length);
	const current = $derived(images[Math.min(index, count - 1)] ?? null);

	$effect(() => {
		// 图片切换时重置缩放视口
		index;
		scale = 1;
		originX = 50;
		originY = 50;
		panX = 0;
		panY = 0;
	});

	function step(delta: number) {
		if (count === 0) return;
		index = (index + delta + count) % count;
	}

	/** 指针事件位置 → 图片内锚点百分比 */
	function originAt(e: MouseEvent, el: HTMLElement) {
		const rect = el.getBoundingClientRect();
		if (rect.width === 0 || rect.height === 0) return;
		originX = ((e.clientX - rect.left) / rect.width) * 100;
		originY = ((e.clientY - rect.top) / rect.height) * 100;
	}

	/** 单击图片：1x ↔ 2x 切换放大，锚定点击位置 */
	function onImageClick(e: MouseEvent) {
		if (suppressClick) {
			suppressClick = false;
			return;
		}
		const el = e.currentTarget as HTMLElement;
		if (scale > 1) {
			scale = 1;
			panX = 0;
			panY = 0;
			originX = 50;
			originY = 50;
		} else {
			originAt(e, el);
			scale = 2;
		}
	}

	function onWheel(e: WheelEvent) {
		e.preventDefault();
		originAt(e, e.currentTarget as HTMLElement);
		const next = Math.min(MAX_SCALE, Math.max(1, scale - e.deltaY * 0.0025));
		scale = next;
		if (next === 1) {
			panX = 0;
			panY = 0;
			originX = 50;
			originY = 50;
		}
	}

	function onPointerDown(e: PointerEvent) {
		if (e.button !== 0) return;
		dragging = true;
		dragStart = { x: e.clientX, y: e.clientY, panX, panY, moved: false };
		(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
	}

	function onPointerMove(e: PointerEvent) {
		if (!dragging) return;
		const dx = e.clientX - dragStart.x;
		const dy = e.clientY - dragStart.y;
		if (Math.abs(dx) > 3 || Math.abs(dy) > 3) dragStart.moved = true;
		if (scale > 1) {
			panX = dragStart.panX + dx;
			panY = dragStart.panY + dy;
		}
	}

	function onPointerUp(e: PointerEvent) {
		if (!dragging) return;
		dragging = false;
		if (dragStart.moved) {
			// 拖拽平移 / 滑动切图结束，紧随的 click 不触发缩放
			suppressClick = true;
			if (scale === 1 && count > 1) {
				// 未放大时的横向滑动 = 切图（纵向滑动忽略，避免误切）
				const dx = e.clientX - dragStart.x;
				if (Math.abs(dx) > 48) step(dx < 0 ? 1 : -1);
			}
		}
	}

	function handleOpenChange(next: boolean) {
		open = next;
		if (!next) onClose?.();
	}
</script>

<Dialog.Root bind:open onOpenChange={handleOpenChange}>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 z-[var(--cst-z-overlay)] bg-black/80 data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=closed]:animate-out data-[state=closed]:fade-out-0" />
		<Dialog.Content
			class="fixed inset-0 z-[var(--cst-z-modal)] flex touch-none select-none flex-col focus-ring data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=open]:zoom-in-95 data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95"
			aria-label="图片查看器"
		>
			<!-- 顶栏：页码 + 关闭 -->
			<div class="flex items-center justify-between px-4 pt-3 pb-1 text-text-primary">
				<span class="rounded-full bg-black/40 px-2.5 py-0.5 text-xs tabular-nums">
					{count > 0 ? index + 1 : 0} / {count}
				</span>
				<button
					type="button"
					class="hover-btn scale-btn hit-slop flex size-9 items-center justify-center rounded-full bg-black/40 text-text-primary focus-ring"
					aria-label="关闭查看器"
					onclick={() => handleOpenChange(false)}
				>
					<X class="size-4" strokeWidth={2} />
				</button>
			</div>

			<!-- 主体：左右切换 + 图片（点击背景关闭；Esc 由 Dialog 承载） -->
			<!-- svelte-ignore a11y_click_events_have_key_events -->
			<!-- svelte-ignore a11y_no_static_element_interactions -->
			<div class="relative flex min-h-0 flex-1 items-center justify-center" onclick={() => handleOpenChange(false)}>
				{#if count > 1}
					<button
						type="button"
						class="hover-btn scale-btn hit-slop absolute left-2 z-10 flex size-10 items-center justify-center rounded-full bg-black/40 text-text-primary focus-ring sm:left-4"
						aria-label="上一张"
						onclick={(e) => {
							e.stopPropagation();
							step(-1);
						}}
					>
						<ChevronLeft class="size-5" strokeWidth={2} />
					</button>
				{/if}

				{#if current}
					<!-- svelte-ignore a11y_click_events_have_key_events -->
					<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
					<img
						src={current.url}
						alt={current.alt ?? ''}
						draggable="false"
						class="max-h-full max-w-full object-contain"
						style="transform:translate({panX}px, {panY}px) scale({scale});transform-origin:{originX}% {originY}%;cursor:{dragging ? 'grabbing' : scale > 1 ? 'grab' : 'zoom-in'};transition:{dragging ? 'none' : 'transform var(--cst-motion-instant) var(--cst-ease-standard)'};"
						onwheel={onWheel}
						onpointerdown={onPointerDown}
						onpointermove={onPointerMove}
						onpointerup={onPointerUp}
						onpointercancel={onPointerUp}
						onclick={(e) => {
							e.stopPropagation();
							onImageClick(e);
						}}
					/>
				{/if}

				{#if count > 1}
					<button
						type="button"
						class="hover-btn scale-btn hit-slop absolute right-2 z-10 flex size-10 items-center justify-center rounded-full bg-black/40 text-text-primary focus-ring sm:right-4"
						aria-label="下一张"
						onclick={(e) => {
							e.stopPropagation();
							step(1);
						}}
					>
						<ChevronRight class="size-5" strokeWidth={2} />
					</button>
				{/if}
			</div>

			<!-- 底部缩略图条（多图） -->
			{#if count > 1}
				<div class="flex items-center justify-center gap-2 overflow-x-auto px-4 pt-1 pb-4">
					{#each images as img, i (img.url + i)}
						<button
							type="button"
							class="shrink-0 overflow-hidden rounded-sm focus-ring {i === index ? 'opacity-full outline-2 outline-accent-default' : 'opacity-soft hover:opacity-full'}"
							style="width:44px;height:44px;"
							aria-label={`查看第 ${i + 1} 张`}
							aria-current={i === index}
							onclick={() => (index = i)}
						>
							<img src={img.url} alt="" loading="lazy" class="h-full w-full object-cover" draggable="false" />
						</button>
					{/each}
				</div>
			{/if}
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>

<svelte:window
	onkeydown={(e) => {
		if (!open) return;
		if (e.key === 'ArrowLeft') step(-1);
		else if (e.key === 'ArrowRight') step(1);
	}}
/>
