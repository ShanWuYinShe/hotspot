<script lang="ts">
	import type { Snippet } from 'svelte';

	// Avatar — 用户/群组头像统一展示组件。
	// 包装元素持有 border-radius + outline + overflow:hidden，徽标作为子元素
	// 与描边同一元素，避免「分离边框的 broken-border」问题。

	let {
		src,
		name,
		size = 40,
		shape = 'circle',
		ring = false,
		badge,
		class: className = '',
		style = ''
	}: {
		/** Image URL. When empty/undefined or the image fails to load, the name initial is shown instead. */
		src?: string;
		/** Display name — supplies the alt text and the fallback initial. */
		name?: string;
		/** Square dimension in px. */
		size?: number;
		/** Circle (people) or rounded rectangle (groups). */
		shape?: 'circle' | 'rounded';
		/** Draw a thin themed ring around the avatar. */
		ring?: boolean;
		/** Optional badge (e.g. online dot) rendered absolutely positioned inside the wrapper. */
		badge?: Snippet;
		class?: string;
		style?: string;
	} = $props();

	let hasError = $state(false);

	const showImage = $derived(Boolean(src) && !hasError);
	const initial = $derived((name || '?').slice(0, 1));
	const borderRadius = $derived(shape === 'circle' ? '9999px' : `${Math.round(size * 0.28)}px`);
</script>

<div
	class="relative shrink-0 {className}"
	style="width:{size}px;height:{size}px;border-radius:{borderRadius};overflow:hidden;background:var(--cst-surface-primary);{ring ? 'border:1px solid var(--cst-accent-border-soft);' : ''}{style}"
>
	{#if showImage}
		<img src={src} alt={name ?? ''} class="h-full w-full object-cover" onerror={() => (hasError = true)} />
	{:else}
		<span
			class="flex h-full w-full items-center justify-center"
			style="font-size:{Math.round(size * 0.42)}px;color:var(--cst-text-secondary);background:var(--cst-accent-soft-bg);"
		>
			{initial}
		</span>
	{/if}
	{#if badge}
		{@render badge()}
	{/if}
</div>
