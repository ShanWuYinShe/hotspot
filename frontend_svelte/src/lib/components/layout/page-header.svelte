<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		title: string;
		description?: string;
		actions?: Snippet;
	}

	let { title, description, actions }: Props = $props();
</script>

<div class="rounded-[1.4rem] border border-border/60 bg-linear-to-br from-background via-card/90 to-muted/30 p-4 shadow-sm sm:rounded-3xl sm:p-5">
	<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
		<div class="min-w-0">
			<h1 class="truncate text-[1.35rem] font-bold tracking-tight sm:text-2xl">{title}</h1>
			{#if description}
				<p class="mt-1 text-sm leading-6 text-muted-foreground">{description}</p>
			{/if}
		</div>
		{#if actions}
			<div class="flex shrink-0 items-center gap-2 self-start sm:self-auto">
				{@render actions()}
			</div>
		{/if}
	</div>
</div>
