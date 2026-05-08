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
		'neu-input-base flex h-9 w-full select-text rounded-2xl outline-none disabled:cursor-not-allowed disabled:opacity-50 md:text-sm',
		className
	)}
	style="background: var(--neu-bg); border: 1px solid var(--neu-border); color: var(--neu-text);"
	bind:value
	{...restProps}
/>
