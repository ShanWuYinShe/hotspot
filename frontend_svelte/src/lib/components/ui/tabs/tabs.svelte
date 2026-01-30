<script lang="ts">
	import { Tabs as TabsPrimitive } from 'bits-ui';
	import type { Snippet } from 'svelte';

	interface Props {
		value?: string;
		onValueChange?: (value: string) => void;
		children?: Snippet;
		class?: string;
	}

	let { value = $bindable(''), onValueChange, children, ...restProps }: Props = $props();

	// 同步外部绑定到内部
	function handleValueChange(newValue: string) {
		value = newValue;
		onValueChange?.(newValue);
	}
</script>

<TabsPrimitive.Root value={value} onValueChange={handleValueChange} {...restProps}>
	{#if children}
		{@render children()}
	{/if}
</TabsPrimitive.Root>
