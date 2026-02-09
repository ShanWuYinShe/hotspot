<script lang="ts">
	import { dev } from '$app/environment';
	import { tagsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import type { TagItem } from '$lib/types/api';
	import { Badge } from '$lib/components/ui/badge';
	import Hash from 'lucide-svelte/icons/hash';

	interface Props {
		refreshTrigger?: number;
	}

	let { refreshTrigger = 0 }: Props = $props();

	let tags = $state.raw<TagItem[]>([]);

	async function loadTags() {
		try {
			const response = await tagsService.getTrending();
			if (isSuccess(response) && response.data) {
				tags = response.data.items ?? [];
			}
		} catch (error) {
			if (dev) console.error('Failed to load trending tags:', error);
			// 不显示 toast，避免影响首页体验
		}
	}

	$effect(() => {
		refreshTrigger;
		loadTags();
	});
</script>

{#if tags.length > 0}
	<section class="space-y-4">
		<div class="flex items-center gap-2 px-1">
			<Hash class="h-5 w-5 text-primary" />
			<h2 class="text-lg font-semibold tracking-tight">热门话题</h2>
		</div>
		<div class="flex flex-wrap gap-2">
			{#each tags as tag (tag.name)}
				<a href="/search?tag={encodeURIComponent(tag.name)}" class="group">
					<Badge variant="secondary" class="cursor-pointer rounded-full px-3 py-1.5 text-sm transition-all hover:bg-primary/10 hover:text-primary">
						#{tag.name}
						<span class="ml-1.5 text-xs opacity-50 group-hover:opacity-100">{tag.usage_count}</span>
					</Badge>
				</a>
			{/each}
		</div>
	</section>
{/if}
