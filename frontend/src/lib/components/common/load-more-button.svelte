<script lang="ts">
  import Spinner from '$lib/components/common/spinner.svelte';
  import ArrowDown from 'lucide-svelte/icons/arrow-down';
  import { t } from '$lib/i18n';

  interface Props {
    loading: boolean;
    hasMore: boolean;
    onLoadMore: () => void;
    class?: string;
    infinite?: boolean;
  }

  let { loading, hasMore, onLoadMore, class: className = '', infinite = true }: Props = $props();

  let viewport: HTMLDivElement | undefined = $state();
  let wasIntersecting = $state(false);

  $effect(() => {
    if (!infinite || !hasMore || !viewport) return;
    const observer = new IntersectionObserver(
      (entries) => {
        const isIntersecting = entries[0]?.isIntersecting ?? false;
        if (isIntersecting && !wasIntersecting && !loading) {
          onLoadMore();
        }
        wasIntersecting = isIntersecting;
      },
      { rootMargin: '400px' }
    );
    observer.observe(viewport);
    return () => {
      observer.disconnect();
    };
  });
</script>

{#if hasMore || loading}
  <div bind:this={viewport} class="flex justify-center py-8 {className}">
    {#if loading}
      <div class="flex items-center gap-2" role="status" aria-live="polite">
        <Spinner size="lg" strokeWidth={1.5} color="var(--text-secondary)" />
        <span class="text-body" style="color: var(--text-secondary)">{t('common.loading')}</span>
      </div>
    {:else}
      <button
        type="button"
        class="flex flex-col items-center gap-1 rounded-full p-0 outline-none pill-hover"
        style="background: transparent; color: var(--text-secondary);"
        onclick={onLoadMore}
        aria-label={t('common.loadMore')}
      >
        <ArrowDown class="h-6 w-6 animate-bounce" strokeWidth={2} />
        <span class="text-caption" style="color: var(--text-secondary)"
          >{t('loadMoreIndicator.clickToLoad')}</span
        >
      </button>
    {/if}
  </div>
{/if}
