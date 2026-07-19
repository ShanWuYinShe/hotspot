<script lang="ts">
  import User from 'lucide-svelte/icons/user';
  import UsersRound from 'lucide-svelte/icons/users-round';
  import OnlineDot, { type OnlineDotSize } from './online-dot.svelte';

  interface Props {
    src?: string | null;
    name?: string | null;
    alt?: string;
    size?: 32 | 36 | 40 | 48 | 64 | 80 | 96 | 128;
    type?: 'user' | 'group';
    shape?: 'circle' | 'rounded';
    class?: string;
    border?: boolean;
    /** 是否显示在线指示点(绝对定位在右下角)。 */
    online?: boolean;
    /** 在线点尺寸档位,默认按 size 自动:≤40 用 sm,48-64 用 md,≥80 用 lg。 */
    onlineSize?: OnlineDotSize;
  }

  let {
    src = null,
    name = '',
    alt,
    size = 40,
    type = 'user',
    shape = 'circle',
    class: className = '',
    border = false,
    online = false,
    onlineSize
  }: Props = $props();

  const imageSrc = $derived(src?.trim() ?? '');
  const label = $derived(name?.trim() ?? '');
  const initials = $derived(Array.from(label).slice(0, 2).join('').toUpperCase());
  const radius = $derived(shape === 'circle' ? '9999px' : 'var(--radius-md)');
  const textClass = $derived(
    size <= 32 ? 'text-caption'
    : size <= 40 ? 'text-meta'
    : size <= 48 ? 'text-body'
    : size <= 64 ? 'text-body-lg'
    : size <= 80 ? 'text-title'
    : size <= 96 ? 'text-h2'
    : 'text-h1'
  );
  const rootStyle = $derived(
    `width: ${size}px; height: ${size}px; border-radius: ${radius}; border: ${border ? '1.5px solid var(--border-theme)' : 'none'};`
  );
  const fallbackStyle = $derived(
    `background: var(--surface-secondary); color: var(--text-secondary); border-radius: ${radius};`
  );
  // 未显式指定 onlineSize 时,按头像尺寸自动选档,避免调用方关心细节。
  const resolvedOnlineSize = $derived(
    onlineSize ?? (size <= 40 ? 'sm' : size <= 64 ? 'md' : 'lg')
  );
</script>

<span class="relative inline-flex shrink-0 {className}">
  <span
    class="inline-flex overflow-hidden"
    style={rootStyle}
    aria-hidden={alt === '' ? 'true' : undefined}
  >
    {#if imageSrc}
      <img
        src={imageSrc}
        alt={alt ?? label}
        class="h-full w-full object-cover"
        loading="lazy"
        decoding="async"
      />
    {:else}
      <span class="flex h-full w-full items-center justify-center" style={fallbackStyle}>
        {#if initials}
          <span class={textClass} style="font-weight: 650;">
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
  {#if online}
    <OnlineDot size={resolvedOnlineSize} />
  {/if}
</span>
