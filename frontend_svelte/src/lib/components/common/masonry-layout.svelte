<script lang="ts">
  /**
   * 高性能 Masonry 瀑布流布局组件
   *
   * 使用绝对定位 + Masonry 算法实现真正的瀑布流效果
   * 参考 Pinterest、小红书的实现方式
   *
   * 特点：
   * 1. 真正的瀑布流布局（项目按高度顺序填充到最短的列）
   * 2. 高性能（只在必要时重新计算）
   * 3. 支持响应式（自动调整列数）
   * 4. 支持动态添加/删除项目
   * 5. 支持入场动画
   */
  import { onMount, tick } from 'svelte';
  import type { Snippet } from 'svelte';
  import { cn } from '$lib/utils';

  interface Props {
    /**
     * 子元素 snippet
     */
    children?: Snippet;

    /**
     * 列数配置
     * @default { mobile: 1, tablet: 2, desktop: 3, wide: 4 }
     */
    columns?: {
      mobile?: number;
      tablet?: number;
      desktop?: number;
      wide?: number;
    };

    /**
     * 列间距（px）
     * @default 16
     */
    gap?: number;

    /**
     * 额外的 CSS class
     */
    class?: string;

    /**
     * 是否显示入场动画
     * @default true
     */
    animated?: boolean;

    /**
     * 动画延迟基数（ms）
     * @default 50
     */
    animationDelay?: number;
  }

  let {
    children,
    columns = {},
    gap = 16,
    class: className,
    animated = true,
    animationDelay = 50
  }: Props = $props();

  // 列数配置（默认值）- 使用 $derived 确保响应式
  const columnConfig = $derived({
    mobile: columns.mobile ?? 1,
    tablet: columns.tablet ?? 2,
    desktop: columns.desktop ?? 3,
    wide: columns.wide ?? 4
  });

  let container: HTMLDivElement;
  let items = $state.raw<HTMLElement[]>([]);
  let containerWidth = $state(0);
  let containerHeight = $state(0);
  let currentColumns = $state(1);
  let isReady = $state(false);
  let queuedLayoutFrame: number | null = null;
  let itemResizeObserver: ResizeObserver | null = null;

  // 记录每列的当前高度
  let columnHeights = $state<number[]>([]);

  /**
   * 根据窗口宽度计算列数
   */
  function calculateColumns(width: number): number {
    if (width >= 1536) return columnConfig.wide;
    if (width >= 1024) return columnConfig.desktop;
    if (width >= 640) return columnConfig.tablet;
    return columnConfig.mobile;
  }

  /**
   * 计算每列的宽度
   */
  function calculateColumnWidth(totalWidth: number, cols: number, gapSize: number): number {
    const totalGap = (cols - 1) * gapSize;
    return (totalWidth - totalGap) / cols;
  }

  /**
   * 找到当前高度最小的列
   */
  function getShortestColumn(): number {
    return columnHeights.indexOf(Math.min(...columnHeights));
  }

  function scheduleLayout() {
    if (queuedLayoutFrame != null) {
      cancelAnimationFrame(queuedLayoutFrame);
    }

    queuedLayoutFrame = requestAnimationFrame(() => {
      queuedLayoutFrame = null;
      layoutMasonry();
    });
  }

  function observeItems() {
    itemResizeObserver?.disconnect();
    itemResizeObserver = new ResizeObserver(() => {
      scheduleLayout();
    });

    items.forEach((item) => {
      itemResizeObserver?.observe(item);
    });
  }

  /**
   * Masonry 布局算法
   * 将每个项目放置到当前高度最小的列
   */
  function layoutMasonry() {
    if (!container || items.length === 0) return;

    const cols = currentColumns;
    const gapSize = gap;
    const colWidth = calculateColumnWidth(containerWidth, cols, gapSize);

    // 初始化列高度数组
    columnHeights = new Array(cols).fill(0);

    // 对每个项目进行定位
    items.forEach((item, index) => {
      // 找到高度最小的列
      const columnIndex = getShortestColumn();

      // 计算 x 位置（列索引 * (列宽 + 间距)）
      const x = columnIndex * (colWidth + gapSize);

      // 计算 y 位置（当前列的高度）
      const y = columnHeights[columnIndex];

      // 设置项目的位置和宽度
      item.style.position = 'absolute';
      item.style.left = `${x}px`;
      item.style.top = `${y}px`;
      item.style.width = `${colWidth}px`;

      // 设置入场动画延迟
      if (animated) {
        item.style.animationDelay = `${index * animationDelay}ms`;
      }

      // 更新该列的高度（当前高度 + 项目高度 + 间距）
      columnHeights[columnIndex] = (columnHeights[columnIndex] ?? 0) + item.offsetHeight + gapSize;
    });

    // 设置容器高度为最高列的高度
    containerHeight = Math.max(...columnHeights) - gapSize;
    isReady = true;
    observeItems();
  }

  /**
   * 更新容器宽度和列数
   */
  function updateLayout() {
    if (!container) return;

    const newWidth = container.offsetWidth;
    const newColumns = calculateColumns(newWidth);

    // 只有在宽度或列数变化时才重新布局
    if (newWidth !== containerWidth || newColumns !== currentColumns) {
      containerWidth = newWidth;
      currentColumns = newColumns;
      tick().then(scheduleLayout);
    }
  }

  /**
   * 观察容器和子元素的变化
   */
  let resizeObserver: ResizeObserver;
  let mutationObserver: MutationObserver;

  onMount(() => {
    // 获取所有子元素
    items = Array.from(container.children) as HTMLElement[];

    // 初始化布局
    updateLayout();

    // 监听窗口大小变化
    window.addEventListener('resize', updateLayout);

    // 使用 ResizeObserver 监听容器大小变化
    resizeObserver = new ResizeObserver(() => {
      updateLayout();
    });
    resizeObserver.observe(container);

    // 使用 MutationObserver 监听子元素变化
    mutationObserver = new MutationObserver(() => {
      items = Array.from(container.children) as HTMLElement[];
      observeItems();
      tick().then(scheduleLayout);
    });
    mutationObserver.observe(container, {
      childList: true,
      subtree: false
    });

    // 监听图片加载完成
    const images = container.querySelectorAll('img');
    let loadedCount = 0;
    const totalImages = images.length;
    const imgLoadHandlers: Array<{ img: HTMLImageElement; handler: () => void }> = [];

    if (totalImages > 0) {
      images.forEach((img) => {
        if (img.complete) {
          loadedCount++;
          if (loadedCount === totalImages) {
            tick().then(scheduleLayout);
          }
        } else {
          const handler = () => {
            loadedCount++;
            if (loadedCount === totalImages) {
              tick().then(scheduleLayout);
            }
          };
          img.addEventListener('load', handler);
          imgLoadHandlers.push({ img, handler });
        }
      });
    }

    return () => {
      window.removeEventListener('resize', updateLayout);
      resizeObserver?.disconnect();
      mutationObserver?.disconnect();
      itemResizeObserver?.disconnect();
      if (queuedLayoutFrame != null) {
        cancelAnimationFrame(queuedLayoutFrame);
      }
      // 清理图片 load 事件监听器
      imgLoadHandlers.forEach(({ img, handler }) => {
        img.removeEventListener('load', handler);
      });
    };
  });

  /**
   * 公开方法：手动触发布局更新
   */
  export function refresh() {
    tick().then(scheduleLayout);
  }

  /**
   * 公开方法：添加新项目后调用
   */
  export function append() {
    items = Array.from(container.children) as HTMLElement[];
    observeItems();
    tick().then(scheduleLayout);
  }
</script>

<div
  bind:this={container}
  class={cn(
    'masonry-layout relative w-full',
    !isReady && 'opacity-0',
    isReady && 'opacity-100 transition-opacity duration-300',
    className
  )}
  style:height={`${containerHeight}px`}
>
  {#if children}
    {@render children()}
  {/if}
</div>

<style>
  .masonry-layout {
    /* 容器使用相对定位 */
    position: relative;
  }

  /* 子元素默认样式 */
  .masonry-layout > :global(*) {
    /* 绝对定位由 JS 设置 */
    box-sizing: border-box;
  }

  /* 入场动画 */
  .masonry-layout.opacity-100 > :global(*) {
    animation: masonry-fade-in 0.4s cubic-bezier(0.4, 0, 0.2, 1) forwards;
    opacity: 0;
  }

  @keyframes masonry-fade-in {
    from {
      opacity: 0;
      transform: translateY(20px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  /* 禁用动画时 */
  @media (prefers-reduced-motion: reduce) {
    .masonry-layout > :global(*) {
      animation: none !important;
      opacity: 1 !important;
    }
  }
</style>
