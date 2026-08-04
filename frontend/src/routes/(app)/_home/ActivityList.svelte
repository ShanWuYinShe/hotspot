<script lang="ts">
	import type { Component } from 'svelte';
	import { ChevronRight } from '@lucide/svelte';
	import BorderedList from '@/components/data-display/BorderedList.svelte';
	import InteractiveListRow from '@/components/navigation/InteractiveListRow.svelte';

	// 附近活动列表（对齐 Figma Make HomePage ActivityList）。
	// BorderedList 容器（圆角外框、1px 行分隔、无行阴影）+ InteractiveListRow 行。

	export interface Activity {
		id: number;
		icon: Component;
		title: string;
		sub: string;
		color: string;
		bg: string;
	}

	let {
		items,
		onClick
	}: {
		items: Activity[];
		onClick: (id: number) => void;
	} = $props();
</script>

{#if items.length > 0}
	<BorderedList>
		{#each items as item (item.id)}
			<InteractiveListRow onClick={() => onClick(item.id)} class="flex items-center gap-3 px-4 py-3 cursor-pointer">
				<div
					class="rounded-tight flex items-center justify-center flex-shrink-0"
					style="width:34px;height:34px;background:{item.bg};"
				>
					<item.icon style="width:16px;height:16px;color:{item.color};" strokeWidth={1.75} />
				</div>
				<div class="flex-1 min-w-0">
					<p style="font-size:13px;color:var(--text-primary);">{item.title}</p>
					<p style="font-size:11px;color:var(--text-secondary);margin-top:1px;">{item.sub}</p>
				</div>
				<ChevronRight style="width:13px;height:13px;color:var(--text-secondary);" strokeWidth={1.5} />
			</InteractiveListRow>
		{/each}
	</BorderedList>
{/if}
