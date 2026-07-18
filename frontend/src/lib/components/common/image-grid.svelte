<script lang="ts" module>
  /**
   * 统一的多图宫格展示 — 收敛全站 4 套互斥的图片布局。
   *
   * 布局规则(对齐主流社交产品):
   * - 1 张:大图,16:10 比例
   * - 2 张:两列均分,1:1
   * - 3 张:三列横排,1:1
   * - 4 张:2×2 网格,1:1
   * - 5-9 张:三列网格(3×3 封顶),第 9 张叠加「+N」角标
   *
   * 点击任意图触发 onSelect(index),由调用方决定是否打开 <ImagePreview>。
   */
  export interface GridImage {
    url: string;
    thumbnail_url?: string;
    width?: number;
    height?: number;
    alt?: string;
  }
</script>

<script lang="ts">
  import SmartImage from '$lib/components/common/smart-image.svelte';

  interface Props {
    images: GridImage[];
    /** 点击图片回调,index 为被点击图的原始索引。 */
    onSelect?: (index: number) => void;
    /** sizes 透传给 SmartImage,默认按网格场景优化。 */
    sizes?: string;
    class?: string;
  }

  let { images, onSelect, sizes = '(min-width: 768px) 360px, 33vw', class: className = '' }: Props =
    $props();

  // 超过 9 张:只渲染前 9 张,第 9 张显示溢出角标
  const MAX = 9;
  const visible = $derived(images.slice(0, MAX));
  const overflow = $derived(Math.max(0, images.length - MAX));
  const count = $derived(visible.length);

  // 根据数量决定 grid 模板
  const gridClass = $derived.by(() => {
    if (count === 1) return 'grid-cols-1';
    if (count === 2) return 'grid-cols-2';
    if (count === 3) return 'grid-cols-3';
    if (count === 4) return 'grid-cols-2';
    return 'grid-cols-3'; // 5-9
  });

  // 单图特殊处理:16:10 比例
  const isSingle = $derived(count === 1);
</script>

{#if count > 0}
  <div class="grid gap-1 {gridClass} {className}">
    {#each visible as image, i (image.url)}
      <button
        type="button"
        class="relative overflow-hidden rounded-[var(--radius-md)] bg-[var(--surface-secondary)] {isSingle
          ? 'col-span-full'
          : ''}"
        style={isSingle ? 'aspect-ratio: 16 / 10;' : 'aspect-ratio: 1 / 1;'}
        onclick={() => onSelect?.(i)}
        aria-label={image.alt ?? `图片 ${i + 1}`}
        data-card-action
      >
        <SmartImage
          src={image.url}
          alt={image.alt ?? ''}
          thumbnail={image.thumbnail_url}
          width={image.width}
          height={image.height}
          {sizes}
          class="h-full w-full object-cover"
        />
        {#if overflow > 0 && i === MAX - 1}
          <div
            class="absolute inset-0 flex items-center justify-center text-title-lg font-semibold"
            style="background: color-mix(in srgb, var(--theme-overlay-strong) 70%, transparent); color: var(--text-on-overlay);"
          >
            +{overflow}
          </div>
        {/if}
      </button>
    {/each}
  </div>
{/if}
