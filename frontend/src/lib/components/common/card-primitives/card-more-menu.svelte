<script lang="ts" module>
  /**
   * CardMoreMenu — 卡片右上角"⋯"更多菜单。
   *
   * 收敛 user-card / group-card 各自抄写的 ~30 行 DropdownMenu 样板:
   * surface-flat trigger + DropdownMenuContent + 一组 item(查看 / 分享 / 举报)。
   *
   * 用法:
   * ```svelte
   * <CardMoreMenu items={[
   *   { icon: User, label: t('userProfile.viewProfile'), onclick: handleViewProfile },
   *   { icon: Share2, label: t('userProfile.shareAria'), onclick: handleShare },
   *   { icon: Flag, label: t('common.report'), onclick: handleReport, destructive: true }
   * ]} />
   * ```
   */
  import type Icon from 'lucide-svelte/icons/more-vertical';
  import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuSeparator,
    DropdownMenuTrigger
  } from '$lib/components/ui/dropdown-menu';
  import MoreHorizontal from 'lucide-svelte/icons/more-horizontal';

  type IconComponent = typeof Icon;

  export interface CardMenuItem {
    icon?: IconComponent;
    label: string;
    onclick: (e: MouseEvent) => void;
    /** 是否危险操作(举报等),渲染为 destructive 红色 + 上方加分隔线。 */
    destructive?: boolean;
  }

  interface Props {
    items: CardMenuItem[];
    /** trigger 按钮的 aria-label。 */
    ariaLabel?: string;
  }
</script>

<script lang="ts">
  let { items, ariaLabel }: Props = $props();

  // 把 destructive 项与普通项分开:destructive 前插分隔线
  const normal = $derived(items.filter((i) => !i.destructive));
  const danger = $derived(items.filter((i) => i.destructive));
</script>

<DropdownMenu>
  <DropdownMenuTrigger>
    <button
      type="button"
      data-card-action
      class="surface-flat inline-flex h-8 w-8 items-center justify-center rounded-full outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
      onclick={(e: MouseEvent) => {
        e.preventDefault();
        e.stopPropagation();
      }}
      aria-label={ariaLabel}
    >
      <MoreHorizontal size={16} />
    </button>
  </DropdownMenuTrigger>
  <DropdownMenuContent align="end" class="w-48">
    {#each normal as item (item.label)}
      <DropdownMenuItem onclick={item.onclick}>
        {#if item.icon}
          {@const Icon = item.icon}
          <Icon class="mr-2 h-4 w-4" />
        {/if}
        <span>{item.label}</span>
      </DropdownMenuItem>
    {/each}
    {#if danger.length > 0}
      <DropdownMenuSeparator />
      {#each danger as item (item.label)}
        <DropdownMenuItem onclick={item.onclick} class="text-destructive">
          {#if item.icon}
            {@const Icon = item.icon}
            <Icon class="mr-2 h-4 w-4" />
          {/if}
          <span>{item.label}</span>
        </DropdownMenuItem>
      {/each}
    {/if}
  </DropdownMenuContent>
</DropdownMenu>
