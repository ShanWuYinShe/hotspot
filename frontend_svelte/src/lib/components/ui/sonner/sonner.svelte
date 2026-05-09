<script lang="ts">
  import { Toaster as Sonner, type ToasterProps } from 'svelte-sonner';
  import { mode } from 'mode-watcher';

  type Props = ToasterProps;

  let { class: className, ...restProps }: Props & { class?: string } = $props();

  const theme = $derived(mode.current);
</script>

<Sonner
  {theme}
  class="toaster group"
  toastOptions={{
    classes: {
      toast: 'group toast',
      description: 'group-[.toast]:text-muted-foreground',
      actionButton: 'group-[.toast]:bg-primary group-[.toast]:text-primary-foreground',
      cancelButton: 'group-[.toast]:bg-muted group-[.toast]:text-muted-foreground'
    }
  }}
  {...restProps}
/>
