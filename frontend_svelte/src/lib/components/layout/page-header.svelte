<script lang="ts">
	import type { Snippet } from 'svelte';
	import ChevronLeft from 'lucide-svelte/icons/chevron-left';

	interface Props {
		title: string;
		description?: string;
		actions?: Snippet;
		backHref?: string;
		backLabel?: string;
	}

	let { title, description, actions, backHref, backLabel = '返回' }: Props = $props();
</script>

<div class="rounded-[1.4rem] border border-border/60 bg-linear-to-br from-background via-card/90 to-muted/30 p-4 shadow-sm sm:rounded-3xl sm:p-5">
	<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
		<div class="min-w-0">
			{#if backHref}
				<a href={backHref} class="mb-3 inline-flex items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground">
					<ChevronLeft class="h-4 w-4" />
					<span>{backLabel}</span>
				</a>
			{/if}
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
