<script lang="ts">
	import { Card, CardContent } from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';
	import MapPin from 'lucide-svelte/icons/map-pin';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';
	interface AnchorSummaryCardProps {
		label?: string;
		description?: string | null;
		subtle?: boolean;
		onAdjust?: () => void;
		overrideName?: string | null;
		actions?: Snippet;
	}

	let {
		label = '当前锚点',
		description = null,
		subtle = false,
		onAdjust,
		overrideName = null,
		actions
	}: AnchorSummaryCardProps = $props();

	const displayName = $derived(overrideName ?? mapAnchorStore.displayName);
</script>

<Card
	class={cn(
		'rounded-[1.35rem] border border-border/60 bg-card/85 shadow-sm backdrop-blur-sm sm:rounded-3xl sm:backdrop-blur-md',
		subtle && 'bg-linear-to-r from-background via-card/90 to-primary/8'
	)}
>
	<CardContent class="flex flex-col gap-3 p-3.5 sm:flex-row sm:items-center sm:justify-between sm:p-4">
		<div class="min-w-0 space-y-1">
			<div class="text-[11px] uppercase tracking-[0.18em] text-muted-foreground">{label}</div>
			<div class="truncate text-sm font-semibold text-foreground sm:text-base">{displayName || '未设置锚点'}</div>
			{#if description}
				<div class="text-xs leading-5 text-muted-foreground">{description}</div>
			{/if}
		</div>

		{#if actions}
			<div class="flex flex-col items-stretch gap-2 sm:flex-row sm:items-center sm:justify-end">
				{@render actions()}
			</div>
		{:else if onAdjust}
		<Button variant="outline" size="sm" class="h-10 gap-2 rounded-2xl border-border/60 bg-background/70" onclick={onAdjust}>
				<MapPin class="h-4 w-4" />
				调整位置
			</Button>
		{/if}
	</CardContent>
</Card>
