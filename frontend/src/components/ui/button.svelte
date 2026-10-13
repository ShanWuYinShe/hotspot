<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';
	import { cn } from '@/lib/utils';

	// Button — 全站统一按钮原语（components/ui）。
	// variant：primary（accent 实底）/ secondary（面板描边）/ outline（透明描边）/
	//          ghost（透明）/ danger（状态色实底）/ surface-accent（accent 淡底 Tonal：
	//          列表次级动作，实体蓝让位全局主流程——对齐鸿蒙 surface-accent 同名变体）。
	// size：sm=32 / md=40 / lg=44 高度，字号与内边距对齐既有页面按钮。
	// loading：显示统一 spinner + aria-busy，并阻止重复提交。
	// 焦点环由 focus-ring 统一提供；hover/tap 由 hover-btn 承载。
	// 调用方 className 经 tailwind-merge 合并，可覆盖圆角/宽度等布局类。

	type ButtonVariant = 'primary' | 'secondary' | 'outline' | 'ghost' | 'danger' | 'surface-accent';
	type ButtonSize = 'sm' | 'md' | 'lg';

	// hover 反馈走组件级 token（全局 button hover 填充只兜底透明裸按钮）：
	// primary = accent 光晕抬档；secondary/outline = 描边加深；ghost = 淡底填充；
	// danger = 危险色光晕抬档（--shadow-danger-glow-*，随主题派生）。
	// 带 bg-* 实底的变体命中全局 hover 排除集，必须自带 hover 反馈（交互四态契约）。
	const VARIANTS: Record<ButtonVariant, string> = {
		primary: 'bg-accent-default text-text-on-accent shadow-raised-sm hover:shadow-raised-md',
		secondary: 'bg-surface-primary text-text-primary border border-border-default shadow-raised-sm hover:border-border-strong',
		outline: 'bg-transparent text-text-primary border border-border-default hover:border-border-strong',
		ghost: 'bg-transparent text-text-primary hover:bg-interactive-hover',
	// danger 实底文字走 text-text-on-accent（随明暗翻转：light 白 / dark 深），而非
	// text-text-on-status——后者是 online/warning 实底的文字位（vibrant light 下为深字，
	// 落在加深后的 danger 实底 #C81E4E 上对比度不足）。与各确认按钮
	//（enabled:bg-status-danger + text-text-on-accent）保持一致，对齐鸿蒙 danger 实底统一白字。
		danger: 'bg-status-danger text-text-on-accent shadow-raised-sm hover:shadow-raised-md',
	// surface-accent Tonal：accent 淡底 + accent 字；hover 走描边加深（同 secondary/
	// outline 档），无边投影（列表内嵌次级动作，quiet 语义）
		'surface-accent': 'bg-accent-bg text-accent-default border border-transparent hover:border-accent-border'
	};

	const SIZES: Record<ButtonSize, string> = {
		sm: 'h-8 px-3 text-sm rounded-md',
		md: 'h-10 px-4 text-sm rounded-md',
		lg: 'h-11 px-5 text-sm rounded-md'
	};

	let {
		children,
		variant = 'primary',
		size = 'md',
		loading = false,
		flat = false,
		class: className = '',
		type = 'button',
		disabled = false,
		...rest
	}: HTMLButtonAttributes & {
		children: Snippet;
		variant?: ButtonVariant;
		size?: ButtonSize;
		loading?: boolean;
		/** true 时不带变体投影（shadow-raised-sm/-md）：用于列表内嵌的次级动作
	 *  按钮（如 LoadMoreButton），投影语义与卡片语言冲突的场景。
	 *  注意：shadow-raised-* 是自定义主题 token，不参与 tailwind-merge 冲突
	 *  消解——className 里写 shadow-none 消不掉它，必须由本 prop 在源头裁剪。 */
		flat?: boolean;
	} = $props();
</script>

<button
	{type}
	disabled={disabled || loading}
	class={cn(
		'inline-flex shrink-0 cursor-pointer select-none items-center justify-center gap-2 whitespace-nowrap font-medium hover-btn focus-ring disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-faint',
		// flat 时在源头裁掉变体投影：shadow-raised-* 是自定义主题 token，不参与
		// tailwind-merge 的冲突消解——用 className 覆盖写 shadow-none 也消不掉
		// （生成 stylesheet 里 raised-* 排在后面会胜出）。
		flat ? VARIANTS[variant].replace(/\bshadow-raised-\S+/g, '').trim() : VARIANTS[variant],
		SIZES[size],
		loading && 'pointer-events-none opacity-soft',
		className
	)}
	aria-busy={loading || undefined}
	{...rest}
>
	{#if loading}
		<span class="spinner size-4" aria-hidden="true"></span>
	{/if}
	<span class="inline-flex items-center gap-1.5 {loading ? 'opacity-emphasis' : ''}">{@render children()}</span>
</button>
