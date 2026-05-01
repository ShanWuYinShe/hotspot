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
	class="rounded-[18px] p-4 text-left outline-none transition-shadow {className}"
	style="background: var(--neu-bg); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
	{...rest}
>
	<p style="font-size: 22px; font-weight: 700; color: var(--neu-text);">{value}</p>
	<p style="font-size: 12px; color: var(--neu-text-muted); margin-top: 2px;">{label}</p>
</button>
