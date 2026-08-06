<script lang="ts">
	import { ChevronRight } from '@lucide/svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import type { Activity } from './activity-data';

	// 活动行：34px 色块 icon + title/sub + chevron（首页「附近活动」与发现页共用）。
	// 对齐 Figma Make 活动列表项；onClick 携带 activity id 交给页面导航。

	let { item, onClick }: { item: Activity; onClick: () => void } = $props();
</script>

<ListItem
	onClick={onClick}
	ariaLabel={item.title}
	class="flex items-center gap-3 px-4 py-3"
>
	<div
		class="rounded-tight flex items-center justify-center shrink-0"
		style="width:34px;height:34px;background:{item.bg};"
	>
		<item.icon style="width:16px;height:16px;color:{item.color};" strokeWidth={1.75} />
	</div>
	<div class="flex-1 min-w-0">
		<p class="text-base"  style="color:var(--cst-text-primary)">{item.title}</p>
		<p class="text-xs mt-px"  style="color:var(--cst-text-secondary)">{item.sub}</p>
	</div>
	<ChevronRight class="size-3.25"  style="color:var(--cst-text-secondary)" strokeWidth={1.5} />
</ListItem>
