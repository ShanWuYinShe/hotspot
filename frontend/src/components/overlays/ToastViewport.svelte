<script lang="ts">
	import type { Component } from 'svelte';
	import { Check, CircleAlert, Info, X } from '@lucide/svelte';
	import { toast, dismissToast, type ToastKind } from '@/stores/toast.svelte';
	import { toastExit } from '@/lib/motion';

	// ToastViewport — 全局 toast 容器（顶部居中堆叠）。
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
			iconClass: 'text-status-online',
			badgeBg: 'bg-status-online-bg',
			badgeRing: 'ring-status-online-border',
			borderClass: 'border-status-online-border'
		},
		danger: {
			icon: CircleAlert,
			iconClass: 'text-status-danger',
			badgeBg: 'bg-danger-bg',
			badgeRing: 'ring-status-danger-border',
			borderClass: 'border-status-danger-border'
		},
		info: {
			icon: Info,
			iconClass: 'text-accent-default',
			badgeBg: 'bg-accent-soft',
			badgeRing: 'ring-accent-border',
			borderClass: 'border-border-default/80'
		}
	};
</script>

<!-- 容器常驻（不包 {#if}）：关闭最后一条时 each 的 out 退场才能播放；
     空容器无内容不占位、不影响布局与 aria-live。 -->
<div
	class="pointer-events-none fixed inset-x-0 top-4 z-[var(--cst-z-toast)] flex flex-col items-center gap-2 px-4"
	aria-live="polite"
>
		{#each toast.items as item (item.id)}
			{@const cfg = CONFIGS[item.kind]}
			<div
				class="pointer-events-auto flex w-full max-w-sm items-center gap-3 rounded-xl border {cfg.borderClass} bg-surface-primary/95 p-3.5 shadow-raised-lg backdrop-blur-md animate-in fade-in-0 slide-in-from-top-2"
				role="status"
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
