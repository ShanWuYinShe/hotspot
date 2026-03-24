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
		'flex h-9 w-full select-text rounded-md border border-input bg-background/80 px-3 py-1 text-base shadow-sm shadow-black/5 dark:shadow-black/20 transition-colors file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-foreground placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 md:text-sm',
		className
	)}
	bind:value
	{...restProps}
/>
