<script lang="ts">
	import { cn } from '$lib/utils';

	interface Props {
		checked?: boolean;
		onchange?: (checked: boolean) => void;
		disabled?: boolean;
		class?: string;
		id?: string;
		'aria-label'?: string;
	}

	let {
		checked = $bindable(false),
		onchange,
		disabled = false,
		class: className,
		id,
		'aria-label': ariaLabel = '切换开关'
	}: Props = $props();

	function toggle() {
		if (disabled) return;
		checked = !checked;
		onchange?.(checked);
	}
</script>

<button
	type="button"
	role="switch"
	aria-checked={checked}
	aria-label={ariaLabel}
	{disabled}
	{id}
	class={cn(
		'relative inline-flex h-[28px] w-[50px] shrink-0 cursor-pointer items-center rounded-full',
		'transition-all duration-200',
		'focus-visible:outline-none focus-visible:ring-[2px] focus-visible:ring-[var(--neu-glow)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--neu-bg)]',
		'disabled:cursor-not-allowed disabled:opacity-50',
		className
	)}
	style="background: {checked ? 'var(--neu-accent-primary)' : 'var(--neu-bg)'}; box-shadow: {checked ? '0 0 0 1px rgba(120,120,128,0.15), inset 0 1px 2px rgba(0,0,0,0.12)' : 'inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light)'};"
	onclick={toggle}
>
	<span
		class="pointer-events-none block h-[22px] w-[22px] rounded-full transition-transform duration-200"
		style="transform: translateX({checked ? '23px' : '2px'}); background: var(--neu-switch-thumb); box-shadow: {checked ? '0 1px 4px var(--neu-switch-shadow)' : '2px 2px 4px var(--neu-shadow-dark), -2px -2px 4px var(--neu-shadow-light)'};"
	></span>
</button>
