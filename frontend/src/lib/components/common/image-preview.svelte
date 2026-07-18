<script lang="ts">
  /**
   * 图片大图预览(lightbox)— 全站统一。
   *
   * 收敛 chat 群聊里硬编码 z-50 的图片预览,并提供给 moment 详情等场景复用。
   * 支持:ESC 关闭、点击遮罩关闭、左右切换、键盘 ←→ 翻图、缩放进入动画。
   *
   * 用法:
   * ```svelte
   * <ImagePreview
   *   images={moment.images.map(i => ({ src: i.url, alt: '...' }))}
   *   bind:index
   *   bind:open
   * />
   * ```
   */
  import { browser } from '$app/environment';
  import X from 'lucide-svelte/icons/x';
  import ChevronLeft from 'lucide-svelte/icons/chevron-left';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import { fade, scale } from 'svelte/transition';

  interface PreviewImage {
    src: string;
    alt?: string;
  }

  interface Props {
    images: PreviewImage[];
    open?: boolean;
    /** 当前展示的图片索引(双向绑定)。 */
    index?: number;
    onClose?: () => void;
  }

  let {
    images,
    open = $bindable(false),
    index = $bindable(0),
    onClose
  }: Props = $props();

  function close() {
    open = false;
    onClose?.();
  }

  function prev(e: MouseEvent) {
    e.stopPropagation();
    if (images.length <= 1) return;
    index = (index - 1 + images.length) % images.length;
  }

  function next(e: MouseEvent) {
    e.stopPropagation();
    if (images.length <= 1) return;
    index = (index + 1) % images.length;
  }

  function onKeydown(e: KeyboardEvent) {
    if (!open) return;
    if (e.key === 'Escape') close();
    else if (e.key === 'ArrowLeft') index = (index - 1 + images.length) % images.length;
    else if (e.key === 'ArrowRight') index = (index + 1) % images.length;
  }

  // 锁定背景滚动 + 全局键盘监听
  $effect(() => {
    if (!browser || !open) return;
    document.addEventListener('keydown', onKeydown);
    const prevOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      document.removeEventListener('keydown', onKeydown);
      document.body.style.overflow = prevOverflow;
    };
  });

  // index 越界保护
  $effect(() => {
    if (open && index >= images.length) index = 0;
  });

  const current = $derived(images[index]);
  const showNav = $derived(images.length > 1);
</script>

{#if open && current}
  <!-- 遮罩 -->
  <div
    class="fixed inset-0 flex items-center justify-center p-4"
    style="z-index: var(--z-modal); background: var(--theme-overlay-strong);"
    transition:fade={{ duration: 200 }}
    onclick={close}
    role="presentation"
  >
    <!-- 关闭按钮 -->
    <button
      type="button"
      class="absolute right-4 top-4 flex items-center justify-center rounded-full outline-none transition-transform duration-150 hover:scale-110 active:scale-95"
      style="width: 40px; height: 40px; background: var(--theme-surface-glass); color: var(--text-on-overlay); backdrop-filter: blur(8px);"
      onclick={(e) => {
        e.stopPropagation();
        close();
      }}
      aria-label="关闭预览"
    >
      <X size={24} />
    </button>

    <!-- 上一张 -->
    {#if showNav}
      <button
        type="button"
        class="absolute left-4 flex items-center justify-center rounded-full outline-none transition-transform duration-150 hover:scale-110 active:scale-95"
        style="width: 40px; height: 40px; background: var(--theme-surface-glass); color: var(--text-on-overlay); backdrop-filter: blur(8px);"
        onclick={prev}
        aria-label="上一张"
      >
        <ChevronLeft size={24} />
      </button>
      <button
        type="button"
        class="absolute right-4 flex items-center justify-center rounded-full outline-none transition-transform duration-150 hover:scale-110 active:scale-95"
        style="width: 40px; height: 40px; background: var(--theme-surface-glass); color: var(--text-on-overlay); backdrop-filter: blur(8px);"
        onclick={next}
        aria-label="下一张"
      >
        <ChevronRight size={24} />
      </button>
    {/if}

    <!-- 图片 -->
    <img
      src={current.src}
      alt={current.alt ?? ''}
      class="max-h-full max-w-full rounded-[var(--radius-sm)] object-contain"
      transition:scale={{ duration: 200, start: 0.95 }}
      onclick={(e) => e.stopPropagation()}
      role="presentation"
    />

    <!-- 计数 -->
    {#if showNav}
      <span
        class="absolute bottom-4 left-1/2 -translate-x-1/2 rounded-full px-3 py-1 text-caption"
        style="background: var(--theme-surface-glass); color: var(--text-on-overlay); backdrop-filter: blur(8px);"
      >
        {index + 1} / {images.length}
      </span>
    {/if}
  </div>
{/if}
