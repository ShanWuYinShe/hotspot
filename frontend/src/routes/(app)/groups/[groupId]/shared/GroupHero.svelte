<script lang="ts">
	import { Users } from '@lucide/svelte';
	import Badge from '@/components/data-display/Badge.svelte';
	import { fadeImage, presetImageUrl, presetSrcSet } from '@/lib/images';

	let { detail }: {
		detail: {
			name: string;
			banner?: string;
			category?: string;
		};
	} = $props();
</script>

{#if detail.banner}
	<div class="media-placeholder relative h-cover-height overflow-hidden rounded-xl shadow-raised-sm">
		<img
			src={presetImageUrl(detail.banner, 'cover', 1600)}
			srcset={presetSrcSet(detail.banner, 'cover')}
			sizes="(min-width: 1024px) 56.25rem, 100vw"
			alt={`${detail.name} 的群组封面`}
			loading="eager"
			fetchpriority="high"
			decoding="async"
			class="h-full w-full object-cover"
			use:fadeImage
		/>
		<div class="absolute inset-0" style="background:var(--cst-surface-cover-scrim);"></div>
		<div class="absolute bottom-5 left-5 right-5">
			{#if detail.category}
				<span class="media-badge">
					{detail.category}
				</span>
			{/if}
			<h1 class="mt-2.5 text-2xl md:text-3xl font-bold tracking-title text-content-inverse drop-shadow-sm">{detail.name}</h1>
		</div>
	</div>
{:else}
	<div class="relative h-cover-height overflow-hidden rounded-xl bg-surface-island p-6 flex flex-col justify-between shadow-raised-sm">
		<div class="pointer-events-none absolute -right-16 -top-16 size-72 rounded-full bg-accent-soft/50 blur-3xl" aria-hidden="true"></div>
		<div class="pointer-events-none absolute -left-10 -bottom-10 size-48 rounded-full bg-surface-subtle blur-2xl" aria-hidden="true"></div>

		<div class="relative z-10 flex items-center justify-between">
			{#if detail.category}
				<Badge tone="accent" size="md">{detail.category}</Badge>
			{:else}
				<Badge tone="neutral" size="md">群组主页</Badge>
			{/if}
			<div class="flex size-11 items-center justify-center rounded-xl bg-accent-soft text-accent-default shadow-raised-sm">
				<Users size={22} strokeWidth={1.75} />
			</div>
		</div>

		<div class="relative z-10">
			<h1 class="text-2xl md:text-3xl font-bold tracking-title text-text-primary">{detail.name}</h1>
		</div>
	</div>
{/if}
