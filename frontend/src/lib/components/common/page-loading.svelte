<!--
 * 页面切换加载指示器
 * 显示在页面顶部的进度条
 -->
<script lang="ts">
  import { navigating } from '$app/state';
</script>

{#if navigating}
  <div
    class="fixed left-0 right-0 top-0 z-[var(--z-loading)] h-1 animate-glow-pulse bg-gradient-to-r from-primary via-accent to-primary"
    style="box-shadow: 0 0 20px var(--color-glow-primary);"
    role="progressbar"
    aria-label="页面加载中"
  >
    <div
      class="h-full animate-pulse bg-gradient-to-r from-transparent via-white/50 to-transparent dark:via-white/25"
      style="animation: shimmer 1.5s infinite;"
    ></div>
  </div>
{/if}

<style>
  @keyframes shimmer {
    0% {
      transform: translateX(-100%);
    }
    100% {
      transform: translateX(100%);
    }
  }
</style>
