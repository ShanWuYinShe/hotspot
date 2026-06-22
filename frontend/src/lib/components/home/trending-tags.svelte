<script lang="ts">
  import { feedService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { useWidget } from '$lib/hooks/use-widget.svelte';
  import type { TagItem } from '$lib/types/api';
  import Hash from 'lucide-svelte/icons/hash';

  interface Props {
    refreshTrigger?: number;
  }

  let { refreshTrigger = 0 }: Props = $props();

  const { data: tags, reload } = useWidget<TagItem[]>(async () => {
    const response = await feedService.getTrending();
    if (!isSuccess(response) || !response.data) throw new Error('Failed to load trending');
    return response.data.trending_tags ?? [];
  });

  $effect(() => {
    refreshTrigger;
    reload();
  });
</script>

{#if (tags?.length ?? 0) > 0}
  <div>
    <div class="mb-2 flex items-center gap-1.5">
      <Hash size={13} color="var(--brand)" />
      <p
        class="uppercase"
        style="font-size: 10px; letter-spacing: 0.14em; color: var(--text-secondary);"
      >
        热门话题
      </p>
    </div>
    <div class="flex flex-wrap gap-1.5">
      {#each tags ?? [] as tag (tag.name)}
        <a href="/search?tag={encodeURIComponent(tag.name)}" class="tag-chip">
          #{tag.name}
          <span class="tag-chip__count">{tag.usage_count}</span>
        </a>
      {/each}
    </div>
  </div>
{/if}
