<script lang="ts">
	import type { HTMLAttributes } from 'svelte/elements';
	import { cn } from '$lib/utils';

	type Size = 'sm' | 'md' | 'lg' | 'xl' | 'hero';

	interface Props extends HTMLAttributes<HTMLDivElement> {
		src: string;
		size?: Size;
		alt?: string;
		active?: boolean;
		class?: string;
	}

	let {
		src,
		size = 'md',
		alt = 'Avatar',
		active = false,
		class: className,
		...rest
	}: Props = $props();

	const sizeClass = $derived(
		size === 'sm'
			? 'w-10 h-10 p-[3px]'
			: size === 'md'
				? 'w-14 h-14 p-[4px]'
				: size === 'lg'
					? 'w-20 h-20 p-[5px]'
					: size === 'xl'
						? 'w-28 h-28 p-[6px]'
						: 'w-[120px] h-[120px] p-[6px]'
	);
</script>

<div
	class={cn(
		'relative flex flex-shrink-0 items-center justify-center rounded-full neu-flat transition-all duration-300',
		active && 'neu-pressed',
		sizeClass,
		className
	)}
	{...rest}
>
	<div class="neu-pressed relative flex h-full w-full items-center justify-center overflow-hidden rounded-full">
		<img
			{src}
			{alt}
			class="pointer-events-none h-[120%] w-[120%] rounded-full object-cover opacity-90 blur-[1px]"
		/>
	</div>
</div>
