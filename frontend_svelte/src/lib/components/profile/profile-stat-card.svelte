<script lang="ts">
	import type { HTMLButtonAttributes } from 'svelte/elements';

	interface Props extends HTMLButtonAttributes {
		value: number | string;
		label: string;
	}

	let { value, label, class: className = '', ...rest }: Props = $props();
</script>

<button
	type="button"
	class={`rounded-2xl border border-border/60 bg-background/70 p-4 text-left transition-all hover:-translate-y-0.5 hover:shadow-lg hover:shadow-black/10 dark:hover:shadow-black/30 ${className}`}
	{...rest}
>
	<div class="text-2xl font-bold">{value}</div>
	<div class="text-sm text-muted-foreground">{label}</div>
</button>
