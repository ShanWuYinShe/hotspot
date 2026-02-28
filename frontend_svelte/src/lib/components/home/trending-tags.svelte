<script lang="ts">
	import { dev } from '$app/environment';
	import { feedService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import type { TagItem } from '$lib/types/api';
	import { Badge } from '$lib/components/ui/badge';
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
			if (version !== loadVersion) return; // 丢弃过期响应
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
	<section class="space-y-4 rounded-3xl border border-border/60 bg-card/80 p-4 shadow-sm backdrop-blur-md">
		<div class="flex items-center gap-2 px-1">
			<div class="rounded-xl border border-border/50 bg-primary/10 p-2 text-primary">
				<Hash class="h-4 w-4" />
			</div>
			<h2 class="text-lg font-semibold tracking-tight">热门话题</h2>
		</div>
		<div class="flex flex-wrap gap-2">
			{#each tags as tag (tag.name)}
				<a href="/search?tag={encodeURIComponent(tag.name)}" class="group">
					<Badge
						variant="secondary"
						class="cursor-pointer rounded-2xl border border-border/50 bg-background/65 px-3 py-1.5 text-sm transition-all hover:border-primary/40 hover:bg-primary/10 hover:text-primary"
					>
						#{tag.name}
						<span class="ml-1.5 text-xs opacity-50 group-hover:opacity-100">{tag.usage_count}</span>
					</Badge>
				</a>
			{/each}
		</div>
	</section>
{/if}
