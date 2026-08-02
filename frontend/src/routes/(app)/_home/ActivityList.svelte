<script lang="ts">
	import type { Component } from 'svelte';
	import { ChevronRight } from '@lucide/svelte';

	// 附近话题活动列表（迁移自旧版 NearbyFeedPage 内 ActivityList/ActivityRow）。

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
	<div class="flex flex-col bg-[var(--neu-border)]" style="gap:1px;border-radius:16px;box-shadow:0 0 0 1px var(--neu-border);overflow:hidden;">
		<div style="overflow:hidden;border-radius:16px;">
			{#each items as item (item.id)}
				<button
					type="button"
					class="neu-hover-bg neu-row-hover flex w-full items-center gap-3 px-4 py-3 text-left outline-none"
					style="background:var(--neu-bg);"
					onclick={() => onClick(item.id)}
				>
					<div
						class="rounded-[10px] flex items-center justify-center flex-shrink-0"
						style="width:34px;height:34px;background:{item.bg};"
					>
						<item.icon style="width:16px;height:16px;color:{item.color};" strokeWidth={1.75} />
					</div>
					<div class="flex-1 min-w-0">
						<p class="text-[13px]" style="color:var(--neu-text);">{item.title}</p>
						<p class="text-[11px]" style="color:var(--neu-text-muted);margin-top:1px;">{item.sub}</p>
					</div>
					<ChevronRight style="width:13px;height:13px;color:var(--neu-text-muted);" strokeWidth={1.5} />
				</button>
			{/each}
		</div>
	</div>
{/if}
