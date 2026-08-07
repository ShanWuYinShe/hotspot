<script lang="ts">
	import { ChevronRight } from '@lucide/svelte';

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

<button type="button" class="hover-bg tap-985 flex items-center gap-2.5 w-full text-left rounded-sm {onClick ? 'cursor-pointer' : ''}" onclick={onClick}>
	<div class="rounded-sm overflow-hidden shrink-0 size-11.5" >
		<img src={image} alt={title} class="w-full h-full object-cover" />
	</div>
	<div class="flex-1 min-w-0">
		<p  style="color:var(--cst-text-primary)" class="truncate text-xs">{title}</p>
		<p  style="color:var(--cst-text-secondary)" class="truncate text-xxs mt-0.5">{meta}</p>
	</div>
	<ChevronRight class="size-3 shrink-0"  style="color:var(--cst-text-secondary)" strokeWidth={1.5} />
</button>
