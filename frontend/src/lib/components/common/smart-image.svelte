<script lang="ts" module>
  /**
   * SmartImage — 响应式图片原语（frontend/AGENTS.md「图片响应式」）
   *
   * 统一三件事：
   *  1. `loading="lazy"` + `decoding="async"` —— 懒加载、不阻塞主线程解码；
   *  2. `width`/`height` —— 写入 `<img>` 撑开占位、消除布局抖动（CLS）；
   *  3. `srcset` + `sizes` —— 传入 `thumbnail` 时自动生成「缩略 480w / 原图 1080w」两档，
   *     浏览器按 `sizes` 描述的显示宽度 + 设备 DPR 择优下载，移动端不再硬拉大图。
   *
   * 适用：**会随断点改变显示尺寸的远程大图**（动态图、封面、画廊、聊天图片）。
   * 不适用：固定尺寸的小图（头像走 `avatar.svelte`，只补 `decoding`）、本地静态资源
   * （`/logo`、`/map/*.svg` 等本就极小，无需 srcset）。
   *
   * 后端已为 `MomentImage` / `MessageMedia` / `UploadedMedia` 提供 `thumbnail_url` +
   * 原始 `width`/`height`，直接透传即可，不依赖额外的图片处理服务。
   */
</script>

<script lang="ts">
  interface Props {
    /** 原图 URL（必填，同时也是无 srcset 时的唯一源与回退）。 */
    src: string;
    alt: string;
    /** 缩略图 URL；提供后与 src 组成「480w / 1080w」两档 srcset。 */
    thumbnail?: string;
    /** 原始宽，写入 <img> 防 CLS。尽量提供；缺失时退化为依赖 CSS 容器尺寸。 */
    width?: number;
    /** 原始高，写入 <img> 防 CLS。 */
    height?: number;
    /**
     * sizes 属性：描述各断点下图片的**显示宽度**，供浏览器择源。
     * 默认按 feed 网格估算（移动单列 / 平板双列 / 桌面三列）；非网格场景由调用方覆盖。
     */
    sizes?: string;
    loading?: 'lazy' | 'eager';
    fetchpriority?: 'high' | 'low' | 'auto';
    class?: string;
  }

  let {
    src,
    alt,
    thumbnail,
    width,
    height,
    sizes = '(min-width: 1280px) 33vw, (min-width: 768px) 50vw, 100vw',
    loading = 'lazy',
    fetchpriority = 'auto',
    class: className = ''
  }: Props = $props();

  // 仅在提供缩略图时构造 srcset，避免对单源图片输出无意义的同名档。
  const srcset = $derived(thumbnail ? `${thumbnail} 480w, ${src} 1080w` : undefined);
</script>

<img
  {src}
  {alt}
  {width}
  {height}
  {sizes}
  {loading}
  {fetchpriority}
  {srcset}
  decoding="async"
  class={className}
/>
