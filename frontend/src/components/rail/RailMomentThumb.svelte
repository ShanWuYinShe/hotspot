<script lang="ts">
	import { ChevronRight, MapPin } from '@lucide/svelte';
	import { avatarImageWidths, buildImageSrcSet, fadeImage, optimizedImageUrl } from '@/lib/images';

	// 详情栏小尺寸帖子缩略。

	let {
		image,
		title,
		meta,
		onClick
	}: {
		image: string;
		title: string;
		meta: string;
		onClick?: () => void;
	} = $props();
</script>

<button type="button" class="tap-97 flex items-center gap-2.5 w-full text-left rounded-sm {onClick ? 'hover-bg cursor-pointer' : 'cursor-default focus-visible:outline-none'}" disabled={!onClick} onclick={onClick}>
	<div class="media-placeholder rounded-sm overflow-hidden shrink-0 size-11.5" >
		{#if image}
			<img
				src={optimizedImageUrl(image, { width: 92 })}
				srcset={buildImageSrcSet(image, avatarImageWidths(46))}
				sizes="46px"
				alt={title}
				loading="lazy"
				decoding="async"
				class="w-full h-full object-cover"
				use:fadeImage
			/>
		{:else}
			<!-- 无图动态占位（禁止 <img src=""> 破图） -->
			<div class="flex h-full w-full items-center justify-center">
				<MapPin class="size-4 text-accent-default" strokeWidth={1.5} />
			</div>
		{/if}
	</div>
	<div class="flex-1 min-w-0">
		<p   class="truncate text-xs text-text-primary">{title}</p>
		<p   class="truncate text-xxs mt-0.5 text-text-secondary">{meta}</p>
	</div>
	<ChevronRight class="size-3 shrink-0 text-text-secondary"   strokeWidth={1.5} />
</button>
