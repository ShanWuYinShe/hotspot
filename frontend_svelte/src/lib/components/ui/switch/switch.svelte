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
		'relative inline-flex h-7 w-12 shrink-0 cursor-pointer items-center rounded-full border-2 transition-all duration-200',
		'focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-primary/40 focus-visible:ring-offset-2',
		'disabled:cursor-not-allowed disabled:opacity-50',
		checked
			? 'border-primary bg-primary shadow-[2px_2px_0px_theme(colors.primary)]'
			: 'border-muted-foreground/30 bg-muted shadow-[2px_2px_0px_theme(colors.muted-foreground/0.2)]',
		className
	)}
	onclick={toggle}
>
	<span
		class={cn(
			'pointer-events-none block h-5 w-5 rounded-full border-2 shadow-sm transition-transform duration-200',
			checked
				? 'translate-x-5 border-primary-foreground bg-primary-foreground'
				: 'translate-x-0.5 border-muted-foreground/30 bg-background'
		)}
	></span>
</button>
