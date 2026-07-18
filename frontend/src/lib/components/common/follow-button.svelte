<script lang="ts" module>
  /**
   * 关注/取关按钮 — 全站统一的视觉与交互。
   *
   * 展示层组件:状态(following/loading)与点击行为(onclick)由调用方控制,
   * 这样不同页面可以保留各自的 state 同步逻辑(如个人主页需同步 relation 对象)。
   *
   * variant:
   * - `solid`(默认):未关注=实心 brand,已关注=描边 brand。用于个人主页、详情页等主操作位。
   * - `ghost`:图标+文字、无背景,brand/secondary 色切换。用于卡片底部工具栏等行内紧凑场景。
   * - `pill`:小号胶囊实心/描边。用于 nearby-row 等极紧凑列表行。
   *
   * size:sm(图标14/字号meta)/ md(图标14/字号meta,内边距更大)。
   * loading 统一用 <Spinner>,文案走 i18n(调用方传 followText/followingText)。
   */
  export type FollowButtonVariant = 'solid' | 'ghost' | 'pill';
</script>

<script lang="ts">
  import { t } from '$lib/i18n';
  import UserPlus from 'lucide-svelte/icons/user-plus';
  import UserCheck from 'lucide-svelte/icons/user-check';
  import Spinner from './spinner.svelte';

  interface Props {
    following: boolean;
    loading?: boolean;
    onclick?: (e: MouseEvent) => void;
    variant?: FollowButtonVariant;
    size?: 'sm' | 'md';
    /** 文案默认走 i18n userProfile.follow / userProfile.following,特殊场景可覆盖。 */
    followText?: string;
    followingText?: string;
    class?: string;
  }

  let {
    following,
    loading = false,
    onclick,
    variant = 'solid',
    size = 'md',
    followText,
    followingText,
    class: className = ''
  }: Props = $props();

  const resolvedFollow = $derived(followText ?? t('userProfile.follow'));
  const resolvedFollowing = $derived(followingText ?? t('userProfile.following'));
  const iconSize = $derived(size === 'sm' ? 14 : 14);
</script>

{#if variant === 'ghost'}
  <button
    data-card-action
    class="flex items-center gap-1.5 py-1 outline-none transition-transform duration-150 active:scale-[0.93] {className}"
    style="color: {following ? 'var(--brand)' : 'var(--text-secondary)'};"
    {onclick}
    disabled={loading}
    aria-pressed={following}
  >
    {#if loading}
      <Spinner size="sm" strokeWidth={1.5} />
    {:else if following}
      <UserCheck size={15} strokeWidth={1.5} />
    {:else}
      <UserPlus size={15} strokeWidth={1.5} />
    {/if}
    <span class="text-caption">{following ? resolvedFollowing : resolvedFollow}</span>
  </button>
{:else if variant === 'pill'}
  <button
    data-card-action
    class="surface-flat flex items-center gap-1 rounded-full px-3.5 py-1 outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97] {className}"
    style={following
      ? 'border: 1px solid var(--brand); color: var(--brand); font-size: var(--text-caption);'
      : 'background: var(--brand); color: var(--text-on-accent); font-size: var(--text-caption); box-shadow: var(--shadow-brand-btn); border: 1px solid transparent;'}
    {onclick}
    disabled={loading}
    aria-pressed={following}
  >
    {#if loading}
      <Spinner size="xs" color="var(--text-secondary)" />
    {/if}
    {following ? resolvedFollowing : resolvedFollow}
  </button>
{:else}
  <!-- solid: 主操作位 -->
  <button
    class="surface-flat flex items-center gap-1.5 rounded-full px-4 py-2 outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97] {className}"
    style={following
      ? 'border: 1.5px solid var(--brand); color: var(--brand); font-size: var(--text-meta);'
      : 'background: var(--brand); color: var(--text-on-accent); font-size: var(--text-meta); box-shadow: var(--shadow-brand-btn); border: 1px solid transparent;'}
    {onclick}
    disabled={loading}
    aria-pressed={following}
  >
    {#if loading}
      <Spinner size="sm" />
    {:else if following}
      <UserCheck size={iconSize} strokeWidth={1.5} />
      {resolvedFollowing}
    {:else}
      <UserPlus size={iconSize} strokeWidth={1.5} />
      {resolvedFollow}
    {/if}
  </button>
{/if}
