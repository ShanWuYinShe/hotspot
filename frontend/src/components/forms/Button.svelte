<script lang="ts" module>
	import { cn } from '@/lib/utils';

	// 表单组件（forms）。
	// Button — 通用文本按钮，提供统一的变体与尺寸契约。
	// 对齐 Figma Make forms/Button 的 cva 变体结构，但用项目语义 token 实现
	// （--neu-accent-primary / --text-on-accent / --neu-border 等），不走 shadcn 灰阶。
	// 替代各页面内联手写的 <button class="neu-hover-btn ..."> 主/次文本按钮。
	//
	// ┌──────────────── 页面内容区 ────────────┐
	// │ 表单 / 卡片操作区                            │
	// │ ┌──────── Button ────────┐                  │
	// │ │ 文本 + 可选图标         │                  │
	// │ └────────────────────────┘                  │
	// └─────────────────────────────────────────────┘
	// 设备范围：桌面、平板和手机通用。

	export type ButtonVariant =
		| 'default'
		| 'destructive'
		| 'outline'
		| 'secondary'
		| 'ghost'
		| 'link';
	export type ButtonSize = 'default' | 'sm' | 'lg' | 'icon';

	const BASE =
		'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-all shrink-0 outline-none disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 focus-visible:outline-2 focus-visible:outline-[var(--neu-accent-primary)] focus-visible:outline-offset-2';

	const VARIANT_CLASS: Record<ButtonVariant, string> = {
		default:
			'bg-[var(--neu-accent-primary)] text-[var(--text-on-accent)] hover:brightness-110 active:brightness-95',
		destructive:
			'bg-[var(--status-danger)] text-white hover:brightness-110 active:brightness-95',
		outline:
			'border border-[var(--neu-border)] bg-[var(--neu-bg)] text-[var(--neu-text)] hover:bg-[var(--neu-accent-soft)] hover:border-[var(--neu-border-pressed)]',
		secondary:
			'bg-[var(--neu-accent-soft)] text-[var(--neu-accent-primary)] border border-[var(--neu-accent-soft-border)] hover:brightness-105 active:brightness-95',
		ghost:
			'text-[var(--neu-text)] hover:bg-[var(--neu-accent-soft)]',
		link:
			'text-[var(--neu-accent-primary)] underline-offset-4 hover:underline'
	};

	const SIZE_CLASS: Record<ButtonSize, string> = {
		default: 'h-9 px-4 py-2 [&_svg:not([class*=size-])]:size-4',
		sm: 'h-8 rounded-md gap-1.5 px-3 [&_svg:not([class*=size-])]:size-3.5',
		lg: 'h-10 rounded-md px-6 [&_svg:not([class*=size-])]:size-4',
		icon: 'size-9 [&_svg:not([class*=size-])]:size-4'
	};

	/** 解析 Button 的 class 字符串（供需要一致样式的 <a>/<button> 复用）。 */
	export function buttonVariants(
		variant: ButtonVariant = 'default',
		size: ButtonSize = 'default',
		className = ''
	): string {
		return cn(BASE, VARIANT_CLASS[variant], SIZE_CLASS[size], className);
	}
</script>

<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';

	let {
		variant = 'default',
		size = 'default',
		class: className = '',
		children,
		...rest
	}: {
		variant?: ButtonVariant;
		size?: ButtonSize;
		class?: string;
		children?: Snippet;
	} & HTMLButtonAttributes = $props();
</script>

<button data-slot="button" class={buttonVariants(variant, size, className)} {...rest}>
	{@render children?.()}
</button>
