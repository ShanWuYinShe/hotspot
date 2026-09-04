<script lang="ts">
	import { goto } from '$app/navigation';
	import { MapPin, ChevronRight } from '@lucide/svelte';
	import { ROUTES } from '@/lib/routes';

	// 详情栏位置 mini 卡。

	let { name, desc }: { name: string; desc?: string } = $props();
</script>

<button
	type="button"
	class="hover-bg hit-slop flex items-center gap-2.5 w-full focus-ring text-left rounded-sm p-2"
	onclick={() => void goto(ROUTES.map)}
>
	<div class="flex items-center justify-center rounded-full shrink-0 size-8 bg-accent-soft"  >
		<MapPin class="size-3.5 text-accent-default"   strokeWidth={1.75} />
	</div>
	<div class="flex-1 min-w-0">
		<p   class="truncate text-xs text-text-primary">{name}</p>
		{#if desc}
			<p   class="truncate text-xxs mt-px text-text-secondary">{desc}</p>
		{/if}
	</div>
	<ChevronRight class="size-3 shrink-0 text-text-secondary"   strokeWidth={1.5} />
</button>
