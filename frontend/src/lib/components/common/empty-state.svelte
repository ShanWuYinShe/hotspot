<script lang="ts">
  import type { ComponentType, Snippet } from 'svelte';

  interface Props {
    icon: ComponentType;
    title: string;
    description?: string;
    actionText?: string;
    onAction?: () => void;
    fill?: boolean;
    /** 渲染密度：'card'(默认，独立卡) | 'inline'(嵌入内容流，无卡背景) */
    variant?: 'card' | 'inline';
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
    variant = 'card',
    class: className = '',
    children
  }: Props = $props();
</script>

{#snippet card()}
  <div
    class="flex w-full flex-col items-center justify-center text-center {variant === 'card'
      ? 'rounded-[var(--radius-lg)] py-16'
      : 'py-14'} {className}"
    style={variant === 'card'
      ? 'background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);'
      : ''}
  >
    <!-- 图标徽标：带柔和品牌渐变光晕，避免「裸图标」的工程师感 -->
    <div
      class="mb-5 flex items-center justify-center rounded-full"
      style="width: 84px; height: 84px; background: color-mix(in srgb, var(--brand) 8%, transparent); border: 1px solid color-mix(in srgb, var(--brand) 16%, transparent);"
    >
      <div
        class="flex items-center justify-center rounded-full"
        style="width: 56px; height: 56px; background: color-mix(in srgb, var(--brand) 12%, var(--surface-bg));"
      >
        <Icon size={26} strokeWidth={1.5} color="var(--brand)" />
      </div>
    </div>
    <p
      class="text-h2"
      style="font-weight: 600; color: var(--text-primary); letter-spacing: -0.01em"
    >
      {title}
    </p>
    {#if description}
      <p
        class="text-body"
        style="color: var(--text-secondary); margin-top: 8px; max-width: 280px; line-height: 1.5"
      >
        {description}
      </p>
    {/if}
    {#if actionText && onAction}
      <button
        class="text-body mt-6 rounded-full px-6 py-2.5 outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
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
