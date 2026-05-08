<script lang="ts">
	import { dev } from '$app/environment';
	import { feedService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import type { TagItem } from '$lib/types/api';
	import Hash from 'lucide-svelte/icons/hash';

	interface Props {
		refreshTrigger?: number;
	}

	let { refreshTrigger = 0 }: Props = $props();

	let tags = $state.raw<TagItem[]>([]);
	let loadVersion = 0;

	async function loadTags() {
		const version = ++loadVersion;
		try {
			const response = await feedService.getTrending();
			if (version !== loadVersion) return;
			if (isSuccess(response) && response.data) {
				tags = response.data.trending_tags ?? [];
			}
		} catch (error) {
			if (version !== loadVersion) return;
			if (dev) console.error('Failed to load trending tags:', error);
		}
	}

	$effect(() => {
		refreshTrigger;
		loadTags();
	});
</script>

{#if tags.length > 0}
	<div>
		<div class="mb-2 flex items-center gap-1.5">
			<Hash size={13} color="var(--neu-accent-primary)" />
			<p class="uppercase" style="font-size: 10px; letter-spacing: 0.14em; color: var(--neu-text-muted);">热门话题</p>
		</div>
		<div class="flex flex-wrap gap-1.5">
			{#each tags as tag (tag.name)}
				<a
					href="/search?tag={encodeURIComponent(tag.name)}"
					class="rounded-full px-3 py-1 text-xs neu-pill-hover transition-transform duration-150 active:scale-[0.93]"
					style="background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted);"
				>
					#{tag.name}
					<span style="opacity: 0.5;">{tag.usage_count}</span>
				</a>
			{/each}
		</div>
	</div>
{/if}
