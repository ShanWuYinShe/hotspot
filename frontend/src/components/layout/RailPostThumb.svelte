<script lang="ts">
	import { ChevronRight } from '@lucide/svelte';

	// 详情栏小尺寸帖子缩略（迁移自 React DetailRail.tsx 内 RailPostThumb）。

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

<div class="neu-hover-row flex items-center gap-2.5 {onClick ? 'cursor-pointer' : ''}" onclick={onClick}>
	<div class="rounded-[10px] overflow-hidden flex-shrink-0" style="width:46px;height:46px;">
		<img src={image} alt={title} class="w-full h-full object-cover" />
	</div>
	<div class="flex-1 min-w-0">
		<p style="font-size:12px;color:var(--neu-text);" class="truncate">{title}</p>
		<p style="font-size:10px;color:var(--neu-text-muted);margin-top:2px;" class="truncate">{meta}</p>
	</div>
	<ChevronRight style="width:12px;height:12px;color:var(--neu-text-muted);flex-shrink:0;" strokeWidth={1.5} />
</div>
