<script lang="ts">
	import type { HTMLTextareaAttributes } from 'svelte/elements';
	import { cn } from '$lib/utils';
	import { useKeyboardFocus } from '$lib/attachments';

	interface Props extends HTMLTextareaAttributes {
		class?: string;
		value?: string;
	}

	let { class: className, value = $bindable(''), ...restProps }: Props = $props();
</script>

<textarea
	{@attach useKeyboardFocus()}
	class={cn(
		'flex min-h-[60px] w-full select-text px-3 py-2 text-base outline-none disabled:cursor-not-allowed disabled:opacity-50 md:text-sm',
		className
	)}
	style="background: var(--neu-bg); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border); border-radius: 22px; color: var(--neu-text);"
	bind:value
	{...restProps}
></textarea>
