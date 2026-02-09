<script lang="ts">
	import { ScrollArea as ScrollAreaPrimitive, type WithoutChildrenOrChild } from 'bits-ui';
	import { cn } from '$lib/utils';

	interface Props extends WithoutChildrenOrChild<ScrollAreaPrimitive.ScrollbarProps> {
		class?: string;
	}

	let { class: className, orientation = 'vertical', ...restProps }: Props = $props();
</script>

<ScrollAreaPrimitive.Scrollbar
	{orientation}
	class={cn(
		'flex touch-none select-none transition-colors',
		orientation === 'vertical' && 'h-full w-2.5 border-l border-l-transparent p-[1px]',
		orientation === 'horizontal' && 'h-2.5 flex-col border-t border-t-transparent p-[1px]',
		className
	)}
	{...restProps}
>
	<ScrollAreaPrimitive.Thumb
		class={cn('relative flex-1 rounded-full bg-border', orientation === 'vertical' && 'flex-1')}
	/>
</ScrollAreaPrimitive.Scrollbar>
