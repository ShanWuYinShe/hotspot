<script lang="ts">
  import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle
  } from '$lib/components/ui/dialog';
  import Spinner from '$lib/components/common/spinner.svelte';

  interface Props {
    open?: boolean;
    title?: string;
    description?: string;
    confirmText?: string;
    cancelText?: string;
    variant?: 'default' | 'destructive';
    loading?: boolean;
    onconfirm?: () => void | Promise<void>;
    oncancel?: () => void;
  }

  let {
    open = $bindable(false),
    title = '确认操作',
    description = '确定要执行此操作吗？',
    confirmText = '确认',
    cancelText = '取消',
    variant = 'default',
    loading = false,
    onconfirm,
    oncancel
  }: Props = $props();

  function handleCancel() {
    open = false;
    oncancel?.();
  }

  async function handleConfirm() {
    await onconfirm?.();
    open = false;
  }
</script>

<Dialog bind:open>
  <DialogContent
    class="sm:max-w-md"
    style="background: var(--surface-bg); border: 1px solid var(--border-theme); box-shadow: var(--shadow-lg);"
  >
    <DialogHeader>
      <DialogTitle>{title}</DialogTitle>
      <DialogDescription>{description}</DialogDescription>
    </DialogHeader>
    <DialogFooter class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
      <button
        class="text-body rounded-full px-5 py-2 outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
        style="background: var(--surface-bg); color: var(--text-primary); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme)"
        onclick={handleCancel}
        disabled={loading}
      >
        {cancelText}
      </button>
      <button
        class="text-body rounded-full px-5 py-2 outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
        style="background: {variant === 'destructive'
          ? 'var(--error)'
          : 'var(--brand)'}; color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn)"
        onclick={handleConfirm}
        disabled={loading}
      >
        {#if loading}
          <Spinner
            size="sm"
            strokeWidth={1.5}
            class="mr-1.5 align-middle"
          />
        {/if}
        {confirmText}
      </button>
    </DialogFooter>
  </DialogContent>
</Dialog>
