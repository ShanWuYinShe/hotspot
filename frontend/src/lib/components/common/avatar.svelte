<script lang="ts">
  import User from 'lucide-svelte/icons/user';
  import UsersRound from 'lucide-svelte/icons/users-round';

  interface Props {
    src?: string | null;
    name?: string | null;
    alt?: string;
    size?: number;
    type?: 'user' | 'group';
    shape?: 'circle' | 'rounded';
    class?: string;
    border?: boolean;
  }

  let {
    src = null,
    name = '',
    alt,
    size = 40,
    type = 'user',
    shape = 'circle',
    class: className = '',
    border = true
  }: Props = $props();

  const imageSrc = $derived(src?.trim() ?? '');
  const label = $derived(name?.trim() ?? '');
  const initials = $derived(Array.from(label).slice(0, 2).join('').toUpperCase());
  const radius = $derived(
    shape === 'circle' ? '9999px' : `${Math.max(12, Math.round(size * 0.28))}px`
  );
  const rootStyle = $derived(
    `width: ${size}px; height: ${size}px; border-radius: ${radius}; border: ${border ? '1.5px solid var(--border-theme)' : 'none'};`
  );
  const fallbackStyle = $derived(
    `background: var(--surface-secondary); color: var(--text-secondary); border-radius: ${radius};`
  );
</script>

<span
  class="inline-flex shrink-0 overflow-hidden {className}"
  style={rootStyle}
  aria-hidden={alt === '' ? 'true' : undefined}
>
  {#if imageSrc}
    <img src={imageSrc} alt={alt ?? label} class="h-full w-full object-cover" loading="lazy" />
  {:else}
    <span class="flex h-full w-full items-center justify-center" style={fallbackStyle}>
      {#if initials}
        <span style="font-size: {Math.max(10, Math.round(size * 0.34))}px; font-weight: 650;">
          {initials}
        </span>
      {:else if type === 'group'}
        <UsersRound size={Math.max(14, Math.round(size * 0.42))} strokeWidth={1.6} />
      {:else}
        <User size={Math.max(14, Math.round(size * 0.42))} strokeWidth={1.6} />
      {/if}
    </span>
  {/if}
</span>
