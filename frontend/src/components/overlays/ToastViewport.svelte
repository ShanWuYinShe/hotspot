<script lang="ts">
	import type { Component } from 'svelte';
	import { Check, CircleAlert, Info, X } from '@lucide/svelte';
	import { flip } from 'svelte/animate';
	import { cubicOut } from 'svelte/easing';
	import { fly } from 'svelte/transition';
	import { toast, dismissToast, type ToastKind } from '@/stores/toast.svelte';
	import { MOTION, motionDuration, toastExit } from '@/lib/motion';

	// ToastViewport — 全局 toast 容器。
	// 位置：桌面/平板横向（lg+）固定右上、顶栏下方——避开顶部常驻灵动岛，
	// 且靠近分享/顶栏动作的触发位置；移动端固定底部、导航/评论输入器上方，
	// 不遮挡顶部页头与常驻输入区。
	// 容器 aria-live="polite"：新增 toast 由屏幕阅读器播报；每条带关闭按钮
	// （aria-label），自动消失由 store 的 setTimeout 负责。
	// 层叠走 --cst-z-toast token；表面/描边/阴影全部用 design token。

	interface ToastConfig {
		icon: Component;
		iconClass: string;
		badgeBg: string;
		badgeRing: string;
		borderClass: string;
	}

	const CONFIGS: Record<ToastKind, ToastConfig> = {
		success: {
			icon: Check,
			iconClass: 'text-text-status-online-on-soft',
			badgeBg: 'bg-status-online-bg',
			badgeRing: 'ring-status-online-border',
			borderClass: 'border-status-online-border'
		},
		danger: {
			icon: CircleAlert,
			iconClass: 'text-text-status-danger-on-soft',
			badgeBg: 'bg-danger-bg',
			badgeRing: 'ring-status-danger-border',
			borderClass: 'border-status-danger-border'
		},
		info: {
			icon: Info,
			iconClass: 'text-text-accent-on-soft',
			badgeBg: 'bg-accent-soft',
			badgeRing: 'ring-accent-border',
			borderClass: 'border-border-default/80'
		}
	};
</script>

<!-- 容器常驻（不包 {#if}）：关闭最后一条时 each 的 out 退场才能播放；
     空容器无内容不占位、不影响布局与 aria-live。 -->
<div
	class="pointer-events-none fixed inset-x-0 bottom-[calc(env(safe-area-inset-bottom,0px)+6rem)] z-[var(--cst-z-toast)] flex flex-col items-center gap-2 px-4 lg:inset-x-auto lg:bottom-auto lg:right-6 lg:top-[calc(var(--cst-desktop-header-height)+0.75rem)] lg:w-96 lg:items-end lg:px-0"
	aria-live="polite"
>
		{#each toast.items as item (item.id)}
			{@const cfg = CONFIGS[item.kind]}
			<!-- 进出同源（P1-3）：入场/出场都用 Svelte fly + 同一 MOTION.exit token，不再入场走
			     tw-animate 150ms/ease、出场走 toastExit 两套曲线；flip 让中间一条消失时其余补间，
			     不再瞬时上跳。reduced 下 duration 均折算为 0（直接到终态）。 -->
			<div
				class="pointer-events-auto flex w-full max-w-sm items-center gap-3 rounded-xl border {cfg.borderClass} bg-surface-primary/95 p-3.5 shadow-raised-lg backdrop-blur-md"
				role="status"
				animate:flip={{ duration: motionDuration(MOTION.exit) }}
				in:fly={{ y: 8, duration: motionDuration(MOTION.exit), easing: cubicOut }}
				out:toastExit
			>
				<div class="flex size-7 shrink-0 items-center justify-center rounded-full {cfg.badgeBg} ring-1 {cfg.badgeRing}">
					<svelte:component this={cfg.icon} class="size-4 {cfg.iconClass}" strokeWidth={2.2} />
				</div>
				<p class="min-w-0 flex-1 text-sm font-medium text-text-primary leading-snug">{item.message}</p>
				<button
					type="button"
					class="hit-slop shrink-0 focus-ring rounded-lg p-1 text-text-secondary transition-colors hover:bg-interactive-hover hover:text-text-primary"
					aria-label="关闭通知"
					onclick={() => dismissToast(item.id)}
				>
					<X class="size-3.5" strokeWidth={2} />
				</button>
			</div>
	{/each}
</div>
