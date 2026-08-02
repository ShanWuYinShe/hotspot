<script lang="ts">
	import type { Component } from 'svelte';
	import { themeStore } from '@/stores/theme.svelte';

	// ActionIconButton — 每个页面页头与紧凑工具栏共用的圆形图标按钮。
	// 迁移自旧版 ActionIconButton：motion 悬停/按下动画由
	// CSS 类（neu-hover-btn + neu-scale-btn）承载，主题差异由主题类选择器处理。

	let {
		icon: Icon,
		label,
		onClick,
		variant = 'surface',
		tone = 'muted',
		size = 36,
		disabled = false,
		active = false,
		type = 'button',
		className = ''
	}: {
		icon: Component;
		label: string;
		onClick?: () => void;
		variant?: 'surface' | 'overlay' | 'ghost';
		tone?: 'muted' | 'accent' | 'inverse';
		size?: 32 | 34 | 36 | 42;
		disabled?: boolean;
		active?: boolean;
		type?: 'button' | 'submit' | 'reset';
		className?: string;
	} = $props();

	let hovered = $state(false);

	const highlighted = $derived(!disabled && (active || (variant === 'ghost' && hovered)));
	const iconColor = $derived(
		tone === 'inverse' ? 'var(--content-inverse)' : tone === 'accent' || highlighted ? 'var(--neu-accent-primary)' : 'var(--content-secondary)'
	);

	const style = $derived.by(() => {
		if (variant === 'overlay') {
			return `width:${size}px;height:${size}px;background:var(--surface-overlay);backdrop-filter:blur(10px);border:1px solid var(--border-inverse);`;
		} else if (variant === 'ghost') {
			return `width:${size}px;height:${size}px;background:${highlighted ? (active ? (hovered ? 'var(--neu-active-hover)' : 'var(--neu-accent-soft)') : 'transparent') : 'transparent'};border:${active ? `1px solid ${hovered ? 'var(--neu-accent-primary)' : 'var(--neu-accent-soft-border)'}` : '1px solid transparent'};`;
		}
		return `width:${size}px;height:${size}px;background:${active ? (hovered ? 'var(--neu-active-hover)' : 'var(--neu-accent-soft)') : 'var(--surface-raised)'};border:${active ? `1px solid ${hovered ? 'var(--neu-accent-primary)' : 'var(--neu-accent-soft-border)'}` : '1px solid var(--border-subtle)'};box-shadow:${active || themeStore.theme === 'forest' ? 'none' : '4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)'};`;
	});
</script>

<button
	type={type}
	class="neu-hover-btn neu-scale-btn flex items-center justify-center rounded-full outline-none flex-shrink-0 focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--neu-bg)] disabled:opacity-45 {className}"
	style={style}
	aria-label={label}
	title={label}
	onclick={() => onClick?.()}
	onmouseenter={() => (hovered = true)}
	onmouseleave={() => (hovered = false)}
	{disabled}
>
	<Icon size={Math.round(size * 0.42)} color={iconColor} strokeWidth={highlighted ? 2 : 1.65} />
</button>
