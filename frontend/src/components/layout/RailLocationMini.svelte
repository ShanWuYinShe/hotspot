<script lang="ts">
	import { goto } from '$app/navigation';
	import { MapPin, ChevronRight } from '@lucide/svelte';
	import { ROUTES } from '@/lib/routes';

	// 详情栏位置 mini 卡（迁移自旧版 DetailRail 内 RailLocationMini）。

	let { name, desc }: { name: string; desc?: string } = $props();
	let hovered = $state(false);
</script>

<button
	type="button"
	class="neu-hover-bg flex items-center gap-2.5 w-full outline-none text-left rounded-[12px] p-2 -m-2"
	onclick={() => void goto(ROUTES.map)}
	onmouseenter={() => (hovered = true)}
	onmouseleave={() => (hovered = false)}
>
	<div class="flex items-center justify-center rounded-full flex-shrink-0" style="width:32px;height:32px;background:var(--neu-accent-soft);">
		<MapPin style="width:13px;height:13px;color:var(--neu-accent-primary);" strokeWidth={1.75} />
	</div>
	<div class="flex-1 min-w-0">
		<p style="font-size:12px;color:var(--neu-text);" class="truncate">{name}</p>
		{#if desc}
			<p style="font-size:10px;color:var(--neu-text-muted);margin-top:1px;" class="truncate">{desc}</p>
		{/if}
	</div>
	<ChevronRight style="width:12px;height:12px;color:var(--neu-text-muted);flex-shrink:0;" strokeWidth={1.5} />
</button>
