<script lang="ts" module>
  /**
   * 在线状态指示点。
   *
   * 统一全站「用户在线」的视觉:实心点 + surface-bg 描边,绝对定位在容器右下角。
   * 尺寸档位:sm=8 / md=10 / lg=12,对应不同头像规格(避免历史上 7/8/9/10/11/12 六种直径)。
   * 非头像场景(如统计卡上的光晕点)可用 `glow` 变体。
   */
  export type OnlineDotSize = 'sm' | 'md' | 'lg';
  export type OnlineDotVariant = 'dot' | 'glow';
</script>

<script lang="ts">
  interface Props {
    size?: OnlineDotSize;
    variant?: OnlineDotVariant;
    class?: string;
  }

  let { size = 'md', variant = 'dot', class: className = '' }: Props = $props();

  // 档位 → 像素,与 Avatar 的尺寸阶梯对应:size≤40 用 sm,40-64 用 md,≥80 用 lg。
  const sizeMap: Record<OnlineDotSize, number> = {
    sm: 8,
    md: 10,
    lg: 12
  };
  const px = $derived(sizeMap[size]);
</script>

{#if variant === 'glow'}
  <!-- 光晕变体:用于装饰性统计卡等非头像场景,无描边,带柔光 -->
  <span
    class="rounded-full {className}"
    style="width: {px}px; height: {px}px; background: var(--semantic-online); box-shadow: 0 0 0 3px color-mix(in srgb, var(--semantic-online) 22%, transparent);"
  ></span>
{:else}
  <span
    class="absolute bottom-0 right-0 rounded-full {className}"
    style="width: {px}px; height: {px}px; background: var(--semantic-online); border: 2px solid var(--surface-bg);"
    aria-hidden="true"
  ></span>
{/if}
