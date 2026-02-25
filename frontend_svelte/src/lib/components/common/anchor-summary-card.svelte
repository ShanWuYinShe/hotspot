<script lang="ts">
	import { Card, CardContent } from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';
	import MapPin from 'lucide-svelte/icons/map-pin';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { cn } from '$lib/utils';
	interface AnchorSummaryCardProps {
		label?: string;
		description?: string | null;
		subtle?: boolean;
		showCoordinates?: boolean;
		onAdjust?: () => void;
		overrideName?: string | null;
		overrideLatitude?: number | null;
		overrideLongitude?: number | null;
	}

	let {
		label = '当前锚点',
		description = null,
		subtle = false,
		showCoordinates = true,
		onAdjust,
		overrideName = null,
		overrideLatitude = null,
		overrideLongitude = null
	}: AnchorSummaryCardProps = $props();

	const displayName = $derived(overrideName ?? mapAnchorStore.displayName);
	const displayLatitude = $derived(overrideLatitude ?? mapAnchorStore.latitude);
	const displayLongitude = $derived(overrideLongitude ?? mapAnchorStore.longitude);
	const hasActionsSlot = Boolean($$slots.actions);
</script>

<Card
	class={cn(
		'rounded-2xl border border-border/60 bg-background/90 shadow-sm backdrop-blur',
		subtle && 'bg-linear-to-r from-primary/5 to-primary/10'
	)}
>
	<CardContent class="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between">
		<div class="min-w-0 space-y-1">
			<div class="text-xs uppercase tracking-wide text-muted-foreground">{label}</div>
			<div class="truncate font-semibold text-foreground">{displayName || '未设置锚点'}</div>
			{#if description}
				<div class="text-xs text-muted-foreground">{description}</div>
			{/if}
			{#if showCoordinates}
				<div class="text-xs text-muted-foreground">
					{displayLatitude.toFixed(4)}, {displayLongitude.toFixed(4)}
				</div>
			{/if}
		</div>

		{#if hasActionsSlot}
			<div class="flex flex-col items-stretch gap-2 sm:flex-row sm:items-center sm:justify-end">
				<!-- svelte-ignore slot_element_deprecated -->
				<slot name="actions" />
			</div>
		{:else if onAdjust}
			<Button variant="outline" size="sm" class="gap-2" onclick={onAdjust}>
				<MapPin class="h-4 w-4" />
				调整位置
			</Button>
		{/if}
	</CardContent>
</Card>
