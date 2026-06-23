<script lang="ts">
  import type { HTMLButtonAttributes } from 'svelte/elements';

  interface Props extends HTMLButtonAttributes {
    value: number | string;
    label: string;
  }

  let { value, label, class: className = '', ...rest }: Props = $props();
</script>

<button
  type="button"
  class="neu-small pill-hover rounded-[var(--radius-md)] p-4 text-left outline-none transition-[box-shadow,transform] duration-200 hover:scale-[1.02] active:scale-[0.97] {className}"
  {...rest}
>
  <p style="font-size: 22px; font-weight: 700; color: var(--text-primary);">{value}</p>
  <p style="font-size: 12px; color: var(--text-secondary); margin-top: 2px;">{label}</p>
</button>
