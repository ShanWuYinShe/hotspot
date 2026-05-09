<script lang="ts">
  import { Separator as SeparatorPrimitive, type WithoutChildrenOrChild } from 'bits-ui';
  import { cn } from '$lib/utils';

  interface Props extends WithoutChildrenOrChild<SeparatorPrimitive.RootProps> {
    class?: string;
  }

  let {
    class: className,
    orientation = 'horizontal',
    decorative = true,
    ...restProps
  }: Props = $props();
</script>

<SeparatorPrimitive.Root
  class={cn(
    'shrink-0',
    orientation === 'horizontal' ? 'h-[1px] w-full' : 'h-full w-[1px]',
    className
  )}
  style="background: var(--neu-border);"
  {orientation}
  {decorative}
  {...restProps}
/>
