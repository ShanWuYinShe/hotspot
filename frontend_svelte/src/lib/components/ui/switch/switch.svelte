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
		'relative inline-flex h-7 w-12 shrink-0 cursor-pointer items-center rounded-full transition-all duration-200',
		'focus-visible:outline-none',
		'disabled:cursor-not-allowed disabled:opacity-50',
		className
	)}
	style="border: 2px solid {checked ? 'var(--neu-accent-primary)' : 'var(--neu-border)'}; background: {checked ? 'var(--neu-accent-primary)' : 'var(--neu-bg)'}; box-shadow: {checked ? '0 2px 8px var(--neu-glow)' : 'inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light)'};"
	onclick={toggle}
>
	<span
		class="pointer-events-none block h-5 w-5 rounded-full transition-transform duration-200"
		style="transform: translateX({checked ? '20px' : '2px'}); background: {checked ? 'var(--neu-active-text)' : 'var(--neu-bg)'}; box-shadow: {checked ? '0 1px 4px rgba(0,0,0,0.15)' : '2px 2px 4px var(--neu-shadow-dark), -2px -2px 4px var(--neu-shadow-light)'};"
	></span>
</button>
