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

<div
	class="rounded-[22px] p-4 sm:p-5"
	style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);"
>
	<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
		<div class="min-w-0">
			{#if backHref}
				<a
					href={backHref}
					class="mb-3 inline-flex items-center gap-1.5 outline-none"
					style="font-size: 13px; color: var(--neu-text-muted);"
				>
					<ChevronLeft size={16} strokeWidth={1.5} />
					<span>{backLabel}</span>
				</a>
			{/if}
			<h1 style="font-size: 20px; font-weight: 600; color: var(--neu-text);">{title}</h1>
			{#if description}
				<p style="font-size: 13px; color: var(--neu-text-muted); margin-top: 4px; line-height: 1.6;">{description}</p>
			{/if}
		</div>
		{#if actions}
			<div class="flex shrink-0 items-center gap-2 self-start sm:self-auto">
				{@render actions()}
			</div>
		{/if}
	</div>
</div>
