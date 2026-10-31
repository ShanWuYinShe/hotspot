<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';
	import { cn } from '@/lib/utils';

	// Button — 全站统一按钮原语（components/ui）。
	// variant：primary（accent 实底）/ secondary（面板描边）/ soft（中性软填，安静次级动作）/
	//          outline（透明描边）/ ghost（透明）/ danger（状态色实底）/
	//          surface-accent（accent 淡底 Tonal：
	//          列表次级动作，实体蓝让位全局主流程——对齐鸿蒙 surface-accent 同名变体）。
	// size：sm=32 / md=40 / lg=44 高度，字号与内边距对齐既有页面按钮。
	// loading：显示统一 spinner + aria-busy，并阻止重复提交。
	// 焦点环由 focus-ring 统一提供；hover/tap 由 hover-btn 承载。
	// 圆角由 shape 表达（rect=派 A 页面动作 rounded-md；pill=派 B 行内关系 chip rounded-full + text-xs），
	// **调用方禁止再用 className 覆盖 rounded-***（一致性检查 button-radius-override 拦截）。
	// 形状之外的宽度/布局类仍可经 className 覆盖（tailwind-merge）。

	type ButtonVariant = 'primary' | 'secondary' | 'soft' | 'outline' | 'ghost' | 'danger' | 'surface-accent';
	type ButtonSize = 'sm' | 'md' | 'lg';
	type ButtonShape = 'rect' | 'pill';

	// hover 反馈走组件级 token（全局 button hover 填充只兜底透明裸按钮）：
	// primary = accent 光晕抬档；secondary/outline = 描边加深；ghost = 淡底填充；
	// danger = 危险色光晕抬档（--shadow-danger-glow-*，随主题派生）。
	// 带 bg-* 实底的变体命中全局 hover 排除集，必须自带 hover 反馈（交互四态契约）。
	const VARIANTS: Record<ButtonVariant, string> = {
		primary: 'bg-accent-default text-text-on-accent shadow-raised-sm hover:shadow-raised-md',
		secondary: 'bg-surface-primary text-text-primary border border-border-default shadow-raised-sm hover:border-border-strong',
	// soft：安静次级动作（R13 统一口径）——中性软填底、无描边、hover 全量加深。
	// 用于页面/行内的次级动作（查看/发消息/私信/拒绝/新会话/展开），替代各页手写的
	// bg-surface-subtle/60 软填胶囊与描边胶囊两种形态；不承担主操作身份。
		soft: 'bg-surface-subtle/60 text-text-primary hover:bg-surface-subtle',
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

	// 形状档：rect=派 A（rounded-md）；pill=派 B 行内 chip（rounded-full + text-xs，字号由形状统一）。
	// 文字档只由 shape 决定，调用方不要再传 text-* 覆盖。
	const SHAPES: Record<ButtonShape, string> = {
		rect: 'rounded-md',
		pill: 'rounded-full text-xs'
	};

	const SIZES: Record<ButtonSize, string> = {
		sm: 'h-8 px-3 text-sm',
		md: 'h-10 px-4 text-sm',
		lg: 'h-11 px-5 text-sm'
	};

	let {
		children,
		variant = 'primary',
		size = 'md',
		shape = 'rect',
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
		/** rect=页面动作（派 A）/ pill=行内关系 chip（派 B，rounded-full + text-xs） */
		shape?: ButtonShape;
		loading?: boolean;
		/** true 时不带变体投影（shadow-raised-sm/-md）：用于列表内嵌的次级动作
	 *  按钮（如 LoadMoreButton），投影语义与卡片语言冲突的场景。
	 *  注意：shadow-raised-* 是自定义主题 token，不参与 tailwind-merge 冲突
	 *  消解——className 里写 shadow-none 消不掉它，必须由本 prop 在源头裁剪。 */
		flat?: boolean;
	} = $props();

	// 禁用/加载中的 primary：accent 实底叠加 opacity-faint 会渲染成低对比
	// 「蓝底暗字」（Gemini UI 审查 r08），改为中性弱化——subtle 底 + 次级字、
	// 去投影，再叠全局 opacity-faint；其余变体维持原 opacity-faint 语言。
	const off = $derived(disabled || loading);
	const variantCls = $derived(
		off && variant === 'primary'
			? VARIANTS.primary
					.replace('bg-accent-default', 'bg-surface-subtle')
					.replace('text-text-on-accent', 'text-text-secondary')
					.replace(/\bshadow-raised-\S+/g, '')
					.trim()
			: flat
				? VARIANTS[variant].replace(/\bshadow-raised-\S+/g, '').trim()
				: VARIANTS[variant]);
</script>

<button
	{type}
	disabled={disabled || loading}
	class={cn(
		'inline-flex shrink-0 cursor-pointer select-none items-center justify-center gap-2 whitespace-nowrap font-medium hover-btn focus-ring disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-faint',
		// flat 时在源头裁掉变体投影：shadow-raised-* 是自定义主题 token，不参与
		// tailwind-merge 的冲突消解——用 className 覆盖写 shadow-none 也消不掉
		// （生成 stylesheet 里 raised-* 排在后面会胜出）。禁用 primary 同理在源头替换。
		variantCls,
		SIZES[size],
		// 形状在后：pill 的 text-xs 必须能压过 size 的 text-sm
		SHAPES[shape],
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