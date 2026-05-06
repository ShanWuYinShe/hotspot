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
	style="background: var(--neu-bg); box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border); border-radius: var(--neu-radius-lg); color: var(--neu-text);"
	bind:value
	{...restProps}
></textarea>
