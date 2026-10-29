<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	// Badge —— 非压图徽章/标签唯一原语（2026-10-29 同类型一致性收口新增）。
	// 尺寸严格两档（frontend/AGENTS.md 徽标契约）：
	//   sm = 紧凑型 20px（h-5 px-2 text-xxs）/ md = 标准型 24px（h-6 px-2.5 text-xs）。
	// 压图徽章继续用 .media-badge（图片覆盖层专用，高度同为 20px）；两者不要互替。
	// tone 一律走语义 token（禁止页内拼 bg-* + text-* 组合）；颜色档只保留这里的枚举，
	// 需要新色先加 enum 再改 token，不得在调用方内联。数字/计数需求用 UnreadBadge（accent 实底）。
	let {
		size = 'sm',
		tone = 'neutral',
		class: className = '',
		/**
		 * 仅地图 marker 调色板等「动态语义色」场景的逃生口：值必须来自 design token 变量
		 * （如 style="background: var(--cst-map-pin-user); color: var(--cst-content-on-marker)"），
		 * 禁止写死 hex/rgba。常规场景一律用 tone，不得用 style 绕过主题。
		 * 已在 docs/ui-components.md 登记为 Badge 的登记例外用法。
		 */
		style = '',
		children
	}: {
		size?: 'sm' | 'md';
		tone?: 'neutral' | 'accent' | 'solid' | 'danger' | 'warning' | 'online' | 'outline';
		class?: string;
		style?: string;
		children?: Snippet;
	} = $props();

	const SIZES = {
		sm: 'h-5 px-2 text-xxs',
		md: 'h-6 px-2.5 text-xs'
	} as const;
	const TONES = {
		neutral: 'bg-surface-subtle text-text-secondary',
		accent: 'bg-accent-soft text-text-accent-on-soft',
		solid: 'bg-accent-default text-text-on-accent',
		danger: 'bg-danger-bg/50 text-text-status-danger-on-soft',
		warning: 'bg-status-warning-bg text-text-status-warning-on-soft',
		online: 'bg-status-online-bg text-text-status-online-on-soft',
		outline: 'border border-border-default bg-transparent text-text-secondary'
	} as const;
</script>

<span
	{style}
	class={cn(
		'inline-flex items-center justify-center gap-1 whitespace-nowrap rounded-full font-medium leading-none',
		SIZES[size],
		TONES[tone],
		className
	)}
>
	{@render children?.()}
</span>
