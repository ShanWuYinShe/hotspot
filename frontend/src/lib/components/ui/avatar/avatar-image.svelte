<script lang="ts">
  import { Avatar as AvatarPrimitive, type WithoutChildrenOrChild } from 'bits-ui';
  import { cn } from '$lib/utils';

  interface Props extends WithoutChildrenOrChild<AvatarPrimitive.ImageProps> {
    class?: string;
  }

  let { class: className, src, alt, ...restProps }: Props = $props();
</script>

<AvatarPrimitive.Image
  class={cn('aspect-square h-full w-full object-cover', className)}
  {src}
  {alt}
  {...restProps}
/>
