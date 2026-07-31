<script lang="ts">
	import type { Snippet } from 'svelte';

	/** A compact label/value pair for settings, metadata, and side rails. */
	let {
		label,
		value,
		detail
	}: {
		label: string;
		value: Snippet;
		detail?: Snippet;
	} = $props();
</script>

<div class="flex min-w-0 items-center justify-between gap-4">
	<div class="min-w-0">
		<p class="text-[12px] text-[var(--neu-text-muted)]">{label}</p>
		{#if detail}
			<p class="mt-0.5 truncate text-[11px] text-[var(--neu-text-muted)]">{@render detail()}</p>
		{/if}
	</div>
	<p class="shrink-0 text-[13px] font-medium text-[var(--neu-text)]">{@render value()}</p>
</div>
