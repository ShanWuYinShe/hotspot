<script lang="ts">
	import { Check } from '@lucide/svelte';
	import type { Group } from './group-data';

	let { group, onToggle }: { group: Group; onToggle: () => void } = $props();
</script>

<button
	type="button"
	class="neu-hover-btn rounded-full outline-none text-[11px] px-[13px] py-1.5 {group.joined ? 'border border-[var(--neu-border-pressed)] text-[var(--neu-text-muted)]' : 'bg-[var(--neu-accent-primary)] text-white font-semibold'}"
	style={group.joined ? '' : 'box-shadow:0 2px 8px var(--neu-glow);'}
	onclick={(e) => {
		e.stopPropagation();
		onToggle();
	}}
>
	{#if group.joined}
		<span class="flex items-center gap-1">
			<Check size={12} />
			已加入
		</span>
	{:else}
		加入
	{/if}
</button>
