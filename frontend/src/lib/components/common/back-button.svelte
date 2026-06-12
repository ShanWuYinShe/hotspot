<script lang="ts">
  import ChevronLeft from 'lucide-svelte/icons/chevron-left';

  interface Props {
    onclick?: () => void;
    href?: string;
    ariaLabel?: string;
    size?: number;
    class?: string;
  }

  let { onclick, href, ariaLabel = '返回', size = 38, class: className = '' }: Props = $props();
</script>

{#if href}
  <a
    {href}
    class="back-button {className}"
    style="width: {size}px; height: {size}px;"
    aria-label={ariaLabel}
  >
    <ChevronLeft size={Math.round(size * 0.55)} strokeWidth={2} color="currentColor" />
  </a>
{:else}
  <button
    type="button"
    {onclick}
    class="back-button {className}"
    style="width: {size}px; height: {size}px;"
    aria-label={ariaLabel}
  >
    <ChevronLeft size={Math.round(size * 0.55)} strokeWidth={2} color="currentColor" />
  </button>
{/if}

<style>
  .back-button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    border-radius: 9999px;
    background: transparent;
    color: var(--text-secondary);
    outline: none;
    cursor: pointer;
    transition:
      background-color 0.18s ease,
      color 0.18s ease,
      transform 0.15s ease;
  }

  .back-button:hover {
    background: var(--theme-accent-soft);
    color: var(--text-primary);
  }

  .back-button:active {
    transform: scale(0.88);
  }

  .back-button:focus-visible {
    outline: 2px solid var(--glow);
    outline-offset: 2px;
  }
</style>
