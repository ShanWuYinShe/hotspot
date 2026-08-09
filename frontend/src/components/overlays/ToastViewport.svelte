<script lang="ts">
	import type { Component } from 'svelte';
	import { Check, CircleAlert, Info, X } from '@lucide/svelte';
	import { toast, dismissToast, type ToastKind } from '@/stores/toast.svelte';

	// ToastViewport — 全局 toast 容器（顶部居中堆叠）。
	// 容器 aria-live="polite"：新增 toast 由屏幕阅读器播报；每条带关闭按钮
	// （aria-label），自动消失由 store 的 setTimeout 负责。
	// 层叠走 --cst-z-toast token；表面/描边/阴影全部用 design token。

	const ICONS: Record<ToastKind, Component> = {
		success: Check,
		danger: CircleAlert,
		info: Info
	};

	const COLORS: Record<ToastKind, string> = {
		success: 'text-status-online',
		danger: 'text-status-danger',
		info: 'text-accent-default'
	};
</script>

{#if toast.items.length > 0}
	<div
		class="pointer-events-none fixed inset-x-0 top-4 z-[var(--cst-z-toast)] flex flex-col items-center gap-2 px-4"
		aria-live="polite"
	>
		{#each toast.items as item (item.id)}
			<div
				class="pointer-events-auto flex w-full max-w-sm items-center gap-2.5 rounded-md border border-border-default bg-surface-primary px-4 py-3 shadow-[var(--cst-elevation-modal)] animate-in fade-in-0 slide-in-from-top-2"
				role="status"
			>
				<svelte:component this={ICONS[item.kind]} class="size-4 shrink-0 {COLORS[item.kind]}" strokeWidth={2} />
				<p class="min-w-0 flex-1 text-sm text-text-primary">{item.message}</p>
				<button
					type="button"
					class="hit-slop shrink-0 focus-ring text-text-secondary"
					aria-label="关闭通知"
					onclick={() => dismissToast(item.id)}
				>
					<X class="size-3.5" strokeWidth={2} />
				</button>
			</div>
		{/each}
	</div>
{/if}
