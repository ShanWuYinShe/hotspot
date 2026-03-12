<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		title: string;
		description?: string;
		actions?: Snippet;
	}

	let { title, description, actions }: Props = $props();
</script>

<div class="rounded-3xl border border-border/60 bg-linear-to-br from-background via-card/85 to-muted/40 p-5 shadow-sm">
	<div class="flex items-center justify-between gap-4">
		<div>
			<h1 class="text-2xl font-bold">{title}</h1>
			{#if description}
				<p class="mt-1 text-sm text-muted-foreground">{description}</p>
			{/if}
		</div>
		{#if actions}
			<div class="shrink-0">
				{@render actions()}
			</div>
		{/if}
	</div>
</div>
