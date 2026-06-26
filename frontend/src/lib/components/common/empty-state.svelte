<script lang="ts">
  import type { ComponentType, Snippet } from 'svelte';

  interface Props {
    icon: ComponentType;
    title: string;
    description?: string;
    actionText?: string;
    onAction?: () => void;
    fill?: boolean;
    class?: string;
    children?: Snippet;
  }

  let {
    icon: Icon,
    title,
    description,
    actionText,
    onAction,
    fill = false,
    class: className = '',
    children
  }: Props = $props();
</script>

{#snippet card()}
  <div
    class="flex w-full flex-col items-center justify-center rounded-[var(--radius-lg)] py-16 text-center {className}"
    style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
  >
    <div
      class="mb-4 flex items-center justify-center rounded-full"
      style="width: 72px; height: 72px; border: 1px solid var(--border-theme);"
    >
      <Icon size={28} strokeWidth={1.25} color="var(--text-secondary)" style="opacity: 0.5;" />
    </div>
    <p class="text-title" style="font-weight: 500; color: var(--text-primary)">{title}</p>
    {#if description}
      <p class="text-body" style="color: var(--text-secondary); margin-top: 6px; max-width: 260px">
        {description}
      </p>
    {/if}
    {#if actionText && onAction}
      <button
        class="text-body mt-5 rounded-full px-5 py-2 outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
        style="font-weight: 500; background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn)"
        onclick={onAction}
      >
        {actionText}
      </button>
    {/if}
    {@render children?.()}
  </div>
{/snippet}

{#if fill}
  <div
    class="flex w-full items-center justify-center"
    style="min-height: clamp(360px, 56vh, 620px);"
  >
    {@render card()}
  </div>
{:else}
  {@render card()}
{/if}
