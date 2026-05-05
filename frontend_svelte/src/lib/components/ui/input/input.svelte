<script lang="ts">
	import type { HTMLInputAttributes } from 'svelte/elements';
	import { cn } from '$lib/utils';
	import { useKeyboardFocus } from '$lib/attachments';

	interface Props extends HTMLInputAttributes {
		class?: string;
		value?: string;
	}

	let { class: className, value = $bindable(''), ...restProps }: Props = $props();
</script>

<input
	{@attach useKeyboardFocus()}
	class={cn(
		'flex h-9 w-full select-text px-3 py-1 text-base rounded-2xl transition-colors outline-none disabled:cursor-not-allowed disabled:opacity-50 md:text-sm',
		className
	)}
	style="background: var(--neu-bg); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border); color: var(--neu-text);"
	bind:value
	{...restProps}
/>
