<script lang="ts">
	import { goto } from '$app/navigation';
	import { MapPin, ChevronRight } from '@lucide/svelte';
	import { ROUTES } from '@/lib/routes';

	// 详情栏位置 mini 卡。

	let { name, desc }: { name: string; desc?: string } = $props();
	let hovered = $state(false);
</script>

<button
	type="button"
	class="hover-bg flex items-center gap-2.5 w-full outline-none text-left rounded-sm p-2 -m-2"
	onclick={() => void goto(ROUTES.map)}
	onmouseenter={() => (hovered = true)}
	onmouseleave={() => (hovered = false)}
>
	<div class="flex items-center justify-center rounded-full shrink-0 size-8"  style="background:var(--cst-accent-soft)">
		<MapPin class="size-3.25"  style="color:var(--cst-accent-default)" strokeWidth={1.75} />
	</div>
	<div class="flex-1 min-w-0">
		<p  style="color:var(--cst-text-primary)" class="truncate text-xs">{name}</p>
		{#if desc}
			<p  style="color:var(--cst-text-secondary)" class="truncate text-xxs mt-px">{desc}</p>
		{/if}
	</div>
	<ChevronRight class="size-3 shrink-0"  style="color:var(--cst-text-secondary)" strokeWidth={1.5} />
</button>
