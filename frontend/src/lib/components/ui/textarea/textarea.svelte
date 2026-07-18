<script lang="ts" module>
  /**
   * 统一的多行文本输入。
   *
   * 收敛全站 9 处手写 <textarea>(field-inset / bg-transparent / compose__textarea 各写各的)。
   *
   * variant:
   * - `inset`(默认):凹陷交互面(field-inset),用于表单内的独立输入(群描述、公告、RBAC)。
   * - `transparent`:透明,贴在父级 input-surface 容器内(个人简介编辑)。
   *
   * size(决定 min-height 与 padding):
   * - `sm`:单行起步、自动撑开(聊天输入框)。
   * - `md`:约 3 行(简介、加入消息)。
   * - `lg`:约 5 行(公告、发瞬间)。
   */
  export type TextareaSize = 'sm' | 'md' | 'lg';
  export type TextareaVariant = 'inset' | 'transparent';
</script>

<script lang="ts">
  import type { HTMLTextareaAttributes } from 'svelte/elements';
  import { cn } from '$lib/utils';
  import { useKeyboardFocus, autofocus } from '$lib/attachments';

  interface Props extends HTMLTextareaAttributes {
    size?: TextareaSize;
    variant?: TextareaVariant;
    /** 传入一个 trigger 值,变化时自动聚焦(用于聊天发送后回焦)。 */
    autofocusTrigger?: unknown;
    class?: string;
    value?: string;
  }

  let {
    size = 'md',
    variant = 'inset',
    autofocusTrigger,
    class: className,
    value = $bindable(''),
    ...restProps
  }: Props = $props();

  // min-height / padding 档位:sm 给聊天(单行起步、自动撑开),md 给 3 行,lg 给 5 行。
  const sizeClass: Record<TextareaSize, string> = {
    sm: 'min-h-[38px] px-3 py-2',
    md: 'min-h-[84px] px-4 py-2.5',
    lg: 'min-h-[132px] px-4 py-2.5'
  };
  // transparent 变体贴在父级容器内,自身无背景/边框/inset 阴影。
  const baseClass = $derived(
    variant === 'transparent' ? 'bg-transparent border-none' : 'field-inset'
  );
</script>

<textarea
  {@attach useKeyboardFocus()}
  {@attach autofocusTrigger !== undefined ? autofocus(autofocusTrigger) : undefined}
  class={cn(
    'text-body w-full flex resize-none outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97] disabled:cursor-not-allowed disabled:opacity-50',
    sizeClass[size],
    baseClass,
    className
  )}
  style={variant === 'transparent'
    ? 'color: var(--text-primary); line-height: 1.65;'
    : 'background: var(--surface-bg); color: var(--text-primary); border-radius: var(--radius-md);'}
  bind:value
  {...restProps}
></textarea>
