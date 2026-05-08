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
	class="neu-small neu-pill-hover rounded-[18px] p-4 text-left outline-none transition-[box-shadow,transform] duration-200 hover:scale-[1.02] active:scale-[0.97] {className}"
	{...rest}
>
	<p style="font-size: 22px; font-weight: 700; color: var(--neu-text);">{value}</p>
	<p style="font-size: 12px; color: var(--neu-text-muted); margin-top: 2px;">{label}</p>
</button>
