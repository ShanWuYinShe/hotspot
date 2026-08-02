<script lang="ts">
	import type { Snippet } from 'svelte';

	/** Semantic state tag for online, selected, pending, and similar compact labels. */
	let {
		children,
		tone = 'accent'
	}: {
		children: Snippet;
		tone?: 'accent' | 'positive' | 'warning';
	} = $props();

	const toneClass = {
		positive: 'border-[color-mix(in_srgb,var(--semantic-online)_25%,transparent)] bg-[color-mix(in_srgb,var(--semantic-online)_12%,transparent)] text-[var(--semantic-online)]',
		warning: 'border-[color-mix(in_srgb,var(--semantic-warning)_25%,transparent)] bg-[color-mix(in_srgb,var(--semantic-warning)_12%,transparent)] text-[var(--semantic-warning)]',
		accent: 'border-[var(--neu-accent-soft-border)] bg-[var(--neu-accent-soft)] text-[var(--neu-accent-primary)]'
	} as const;
</script>

<span class="inline-flex items-center rounded-full border px-2 py-0.5 text-[10px] font-medium {toneClass[tone]}">
	{@render children()}
</span>
