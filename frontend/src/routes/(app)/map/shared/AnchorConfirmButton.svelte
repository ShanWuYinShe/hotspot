<script lang="ts">
	import { Check, MapPin, RotateCcw } from '@lucide/svelte';
	import type { AnchorFeedbackState } from './anchor-feedback.svelte';

	// 浏览锚点确认按钮：状态反馈收在按钮本体上（替代成功 toast）。
	// idle：主按钮「将地图中心设为浏览锚点」；pending：同一按钮内 spinner + 正在生效…；
	// success：soft success + ✓ 浏览锚点已生效；error：danger soft + 点击重试；
	// 动画结束后回到 idle 主按钮——不出现灰色 disabled，避免「以为不能更新」。
	let {
		state = 'idle',
		label,
		onclick
	}: {
		state?: AnchorFeedbackState;
		label: string;
		onclick: () => void;
	} = $props();

	const isPending = $derived(state === 'pending');
	const isSuccess = $derived(state === 'success');
	const isError = $derived(state === 'error');

	const variantClass = $derived(
		isPending
			? 'bg-accent-default text-text-on-accent shadow-raised-sm cursor-progress'
			: isSuccess
				? 'border border-status-online-border bg-status-online-bg text-text-status-online-on-soft cursor-default'
				: isError
					? 'border border-status-danger-border bg-danger-bg text-text-status-danger-on-soft cursor-pointer'
					: 'bg-accent-default text-text-on-accent shadow-raised-sm cursor-pointer hover:shadow-raised-md'
	);

	function handleClick() {
		if (isPending || isSuccess) return;
		onclick();
	}
</script>

<button
	type="button"
	class="hover-btn focus-ring relative inline-flex h-11 w-full items-center justify-center gap-2 overflow-hidden rounded-md px-5 text-sm font-medium {variantClass}"
	aria-disabled={isPending || undefined}
	aria-busy={isPending || undefined}
	aria-live="polite"
	onclick={handleClick}
>
	{#if isPending}
		<span class="anchor-cta-shimmer" aria-hidden="true"></span>
		<span class="spinner size-4" aria-hidden="true"></span>
		<span>正在生效…</span>
	{:else if isSuccess}
		<Check class="size-4" strokeWidth={2.5} aria-hidden="true" />
		<span>浏览锚点已生效</span>
	{:else if isError}
		<RotateCcw class="size-4" strokeWidth={2} aria-hidden="true" />
		<span>设置失败，点击重试</span>
	{:else}
		<MapPin class="size-4" strokeWidth={1.75} aria-hidden="true" />
		<span class="truncate">{label}</span>
	{/if}
</button>

<style>
	/* 进行中扫光：给「正在生效…」一个可感知的进行态；reduced-motion 下完全关闭。 */
	.anchor-cta-shimmer {
		position: absolute;
		inset: 0;
		pointer-events: none;
		background: linear-gradient(
			100deg,
			transparent 20%,
			color-mix(in srgb, var(--cst-content-inverse) 22%, transparent) 45%,
			transparent 70%
		);
		animation: anchor-cta-shimmer 1.1s linear infinite;
	}
	@keyframes anchor-cta-shimmer {
		from {
			transform: translateX(-60%);
		}
		to {
			transform: translateX(60%);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.anchor-cta-shimmer {
			display: none;
		}
	}
</style>
