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

	// 头像 src 变化（如上传新头像）后重置错误态，否则永远停留首字母兜底
	$effect(() => {
		void src;
		hasError = false;
	});

	const showImage = $derived(Boolean(src) && !hasError);
	const initial = $derived((name || '?').slice(0, 1));
	const borderRadius = $derived(shape === 'circle' ? '9999px' : `${Math.round(size * 0.28)}px`);
</script>

<div
	class="relative shrink-0 overflow-hidden bg-surface-primary {className}"
	style="width:{size}px;height:{size}px;border-radius:{borderRadius};{ring ? 'border:var(--cst-border-width-hairline) solid var(--cst-accent-border-soft);' : ''}{style}"
>
	{#if showImage}
		<img src={src} loading="lazy" decoding="async" alt={name ?? ''} class="h-full w-full object-cover" onerror={() => (hasError = true)} />
	{:else}
		<span
			class="flex h-full w-full items-center justify-center text-text-secondary bg-accent-bg"
			style="font-size:{Math.round(size * 0.42)}px;"
		>
			{initial}
		</span>
	{/if}
	{#if badge}
		{@render badge()}
	{/if}
</div>
