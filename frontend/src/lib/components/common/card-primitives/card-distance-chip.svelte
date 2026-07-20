<script lang="ts" module>
  /**
   * CardDistanceChip — 卡片场景统一的小型品牌色胶囊 chip。
   *
   * 收敛 moment-card 的 `__location`(地点名)、user/group-card 的 `__distance`(距离)
   * 三处字节级一致的样式:brand color-mix 描边 + brand 软底 + MapPin 图标 + caption 文本。
   *
   * 用法:
   * ```svelte
   * <CardDistanceChip text="5km" />
   * <CardDistanceChip text={locationName} />
   * ```
   */
  import type { Snippet } from 'svelte';
  import type Icon from 'lucide-svelte/icons/map-pin';
  import MapPin from 'lucide-svelte/icons/map-pin';

  type IconComponent = typeof Icon;

  interface Props {
    /** chip 内的主文本(距离 / 地点名等)。 */
    text: string;
    /** 自定义图标组件,默认 MapPin。传 lucide 图标组件即可。 */
    icon?: IconComponent;
    /** 图标尺寸,默认 10(distance chip 用)。 */
    iconSize?: number;
    /** 最大宽度限制(地点名可能较长),默认不限。 */
    maxWidth?: string;
    /** 透传额外 class。 */
    class?: string;
    /** 自定义图标 snippet(优先于 icon prop,用于需要传 color 的场景)。 */
    iconSnippet?: Snippet;
  }
</script>

<script lang="ts">
  let {
    text,
    icon = MapPin,
    iconSize = 10,
    maxWidth,
    class: className = '',
    iconSnippet
  }: Props = $props();

  const IconComp = $derived(icon);
  const styleDecl = $derived(maxWidth ? `max-width: ${maxWidth};` : undefined);
</script>

<span class="card-distance-chip {className}" style={styleDecl}>
  {#if iconSnippet}
    {@render iconSnippet()}
  {:else}
    <IconComp size={iconSize} strokeWidth={1.5} />
  {/if}
  <span class="card-distance-chip__text">{text}</span>
</span>

<style>
  .card-distance-chip {
    display: inline-flex;
    flex: 0 0 auto;
    align-items: center;
    gap: 4px;
    border: 1px solid color-mix(in srgb, var(--brand) 24%, var(--border-theme));
    border-radius: var(--radius-pill);
    padding: 4px 8px;
    color: var(--brand);
    background: color-mix(in srgb, var(--brand) 9%, var(--surface-bg));
    font-size: var(--text-caption);
    font-weight: 600;
    line-height: 1;
  }

  .card-distance-chip__text {
    min-width: 0;
    overflow: clip;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
