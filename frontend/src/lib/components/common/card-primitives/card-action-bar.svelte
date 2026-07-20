<script lang="ts" module>
  /**
   * CardActionBar — 卡片底部操作栏容器。
   *
   * 收敛 user-card / group-card 各自抄写的 footer 结构:
   * `border-top + flex justify-between` 容器,左 primary 操作(带图标 + 文案的 card-action),
   * 右 secondary 操作(FollowButton / CardMoreMenu / 收藏分享等)。
   *
   * 用法:
   * ```svelte
   * <CardActionBar>
   *   {#snippet primary()}
   *     <button class="card-action" ...>...</button>
   *   {/snippet}
   *   {#snippet secondary()}
   *     <FollowButton ... />
   *     <CardMoreMenu ... />
   *   {/snippet}
   * </CardActionBar>
   * ```
   */
  import type { Snippet } from 'svelte';

  interface Props {
    /** 左侧主操作区(带图标 + 文案的按钮,如「私信」「加入群组」)。 */
    primary: Snippet;
    /** 右侧次操作区(FollowButton / CardMoreMenu / 收藏分享等)。 */
    secondary?: Snippet;
    class?: string;
  }
</script>

<script lang="ts">
  let { primary, secondary, class: className = '' }: Props = $props();
</script>

<div
  class="card-action-bar {className}"
  style="border-top: 1px solid color-mix(in srgb, var(--border-theme) 72%, transparent);"
>
  <div class="card-action-bar__primary">
    {@render primary()}
  </div>
  {#if secondary}
    <div class="card-action-bar__secondary">
      {@render secondary()}
    </div>
  {/if}
</div>

<style>
  .card-action-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    padding: 10px var(--card-pad-x, clamp(14px, 2vw, 18px)) var(--card-pad-y, clamp(13px, 1.8vw, 16px));
  }

  .card-action-bar__primary {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
  }

  .card-action-bar__secondary {
    display: flex;
    flex: 0 0 auto;
    align-items: center;
    gap: 8px;
  }
</style>
