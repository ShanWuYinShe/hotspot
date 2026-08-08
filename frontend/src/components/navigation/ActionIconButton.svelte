<script lang="ts">
	import type { Component } from 'svelte';
	import { theme } from '@/stores/theme.svelte';

	// ActionIconButton — 圆形图标按钮（对齐 Figma Make ActionIconButton 契约）。
	// variant: surface（凸起）/ overlay（图片覆盖层 blur）/ ghost（透明）。
	// tone: muted / accent / inverse 决定图标颜色；hover/tap 由 scale-btn 承载。
	// Figma 无 active 态、无 strokeWidth 变化、无 focus ring（仅 outline-none）。

	let {
		icon: Icon,
		label,
		onClick,
		variant = 'surface',
		tone = 'muted',
		size = 36,
		disabled = false,
		type = 'button',
		class: className = ''
	}: {
		icon: Component;
		label: string;
		onClick?: () => void;
		variant?: 'surface' | 'overlay' | 'ghost';
		tone?: 'muted' | 'accent' | 'inverse';
		size?: 32 | 34 | 36 | 42;
		disabled?: boolean;
		type?: 'button' | 'submit' | 'reset';
		class?: string;
	} = $props();

	const iconColor = $derived(
		tone === 'inverse' ? 'var(--cst-content-inverse)' : tone === 'accent' ? 'var(--cst-accent-default)' : 'var(--cst-content-secondary)'
	);

	const style = $derived.by(() => {
		if (variant === 'overlay') {
			return `width:${size}px;height:${size}px;background:var(--cst-surface-overlay);backdrop-filter:blur(var(--cst-blur-md));border:1px solid var(--cst-border-inverse);`;
		} else if (variant === 'ghost') {
			return `width:${size}px;height:${size}px;background:transparent;border:1px solid transparent;`;
		}
		return `width:${size}px;height:${size}px;background:var(--cst-surface-raised);border:1px solid var(--cst-border-subtle);box-shadow:${theme.theme === 'forest' ? 'none' : 'var(--shadow-raised-sm)'};`;
	});
</script>

<button
	type={type}
	class="hover-btn scale-btn flex items-center justify-center rounded-full outline-none shrink-0 disabled:opacity-50 {className}"
	style={style}
	aria-label={label}
	title={label}
	onclick={() => onClick?.()}
	{disabled}
>
	<Icon size={Math.round(size * 0.42)} color={iconColor} strokeWidth={1.65} />
</button>
