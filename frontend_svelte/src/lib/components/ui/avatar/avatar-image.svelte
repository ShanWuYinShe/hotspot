<script lang="ts">
	import { Avatar as AvatarPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils';

	interface Props {
		src?: string;
		alt?: string;
		class?: string;
	}

	let { class: className, src, alt, ...restProps }: Props = $props();
</script>

<AvatarPrimitive.Image
	class={cn('aspect-square h-full w-full', className)}
	{src}
	{alt}
	{...restProps}
/>
