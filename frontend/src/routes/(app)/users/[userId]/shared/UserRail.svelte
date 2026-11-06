<script lang="ts">
	import { goto } from '$app/navigation';
	import { ChevronRight, MapPin, Tag, Users } from '@lucide/svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import Badge from '@/components/data-display/Badge.svelte';
	import RailUserRow from '@/components/rail/RailUserRow.svelte';
	import { ROUTES } from '@/lib/routes';

	let {
		location,
		tags,
		mutualFriends
	}: {
		location: string;
		tags: string[];
		mutualFriends: Array<{ id: string; name: string; sub: string; avatar: string }>;
	} = $props();
</script>

<div class="flex flex-col gap-4 py-2">
	{#if location}
		<section class="rounded-lg bg-surface-island p-4 shadow-xs" aria-label="所在位置">
			<SectionHead variant="rail" title="所在位置" icon={MapPin} />
			<button
				type="button"
				class="hover-bg -mx-2 flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-left focus-ring"
				onclick={() => void goto(ROUTES.map)}
			>
				<span class="min-w-0 flex-1 truncate text-xs text-text-primary">{location}</span>
				<ChevronRight class="size-3.5 shrink-0 text-text-secondary" strokeWidth={1.5} />
			</button>
		</section>
	{/if}

	{#if tags.length > 0}
		<section class="rounded-lg bg-surface-island p-4 shadow-xs" aria-label="兴趣标签">
			<SectionHead variant="rail" title="兴趣标签" icon={Tag} />
			<div class="flex flex-wrap gap-2">
				{#each tags as t (t)}
					<Badge tone="accent" size="md">#{t}</Badge>
				{/each}
			</div>
		</section>
	{/if}

	{#if mutualFriends.length > 0}
		<section class="rounded-lg bg-surface-island p-4 shadow-xs" aria-label="共同好友">
			<SectionHead variant="rail" title="共同好友" icon={Users} />
			<div class="flex flex-col gap-2.5">
				{#each mutualFriends as f (f.id)}
					<RailUserRow avatar={f.avatar} name={f.name} sub={f.sub} onClick={() => void goto(ROUTES.user(f.id))} />
				{/each}
			</div>
		</section>
	{/if}
</div>
