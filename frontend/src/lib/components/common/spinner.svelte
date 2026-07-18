<script lang="ts" module>
  /**
   * 统一的加载旋转指示器。
   *
   * 收敛全站手写的 `<Loader2 class="animate-spin" size={N}>` 与 `<div class="border spinner">`。
   * 尺寸档位:xs=12(聊天气泡内极小)/ sm=14(按钮内)/ md=16(表单、卡片操作)/ lg=20(页面级、大块加载)/ xl=32(全屏占位)。
   * 颜色默认跟随 currentColor,可用 `color` prop 覆盖(传 CSS 变量或颜色值)。
   */
  export type SpinnerSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl';
</script>

<script lang="ts">
  import Loader2 from 'lucide-svelte/icons/loader-2';

  interface Props {
    size?: SpinnerSize;
    color?: string;
    strokeWidth?: number;
    class?: string;
  }

  let { size = 'md', color, strokeWidth = 1.75, class: className = '' }: Props = $props();

  const sizeMap: Record<SpinnerSize, number> = {
    xs: 12,
    sm: 14,
    md: 16,
    lg: 20,
    xl: 32
  };
  const px = $derived(sizeMap[size]);
</script>

<Loader2
  size={px}
  {strokeWidth}
  class="inline-block animate-spin {className}"
  color={color}
  aria-hidden="true"
/>
