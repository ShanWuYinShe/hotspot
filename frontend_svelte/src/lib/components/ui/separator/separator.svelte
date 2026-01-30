<script lang="ts">
	import { Separator as SeparatorPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils';

	interface Props {
		orientation?: 'horizontal' | 'vertical';
		decorative?: boolean;
		class?: string;
	}

	let { class: className, orientation = 'horizontal', decorative = true, ...restProps }: Props = $props();
</script>

<SeparatorPrimitive.Root
	class={cn(
		'shrink-0 bg-border',
		orientation === 'horizontal' ? 'h-[1px] w-full' : 'h-full w-[1px]',
		className
	)}
	{orientation}
	{decorative}
	{...restProps}
/>
