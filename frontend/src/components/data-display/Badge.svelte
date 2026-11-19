<script lang="ts">
	import type { Snippet } from 'svelte';
	import { extendTailwindMerge } from 'tailwind-merge';

	// Badge —— 非压图徽章/标签唯一原语（2026-10-29 同类型一致性收口新增）。
	// 尺寸四档（契约唯一目录 docs/ui-components.md「Badge」行已登记）：
	//   sm     = 紧凑型 20px（h-5 px-2 text-xxs）/ md = 标准型 24px（h-6 px-2.5 text-xs）；
	//   status = 状态/类型药丸 20px（h-5 px-2.5 text-xs，11px Medium、水平垂直居中）——
	//            举报状态与举报类型徽标专用（/admin/reports、/me/reports）；
	//   author = 评论作者微徽章 28×16（h-4 px-1 text-xxs，圆角 4）——评论昵称后的「作者」。
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
		size?: 'sm' | 'md' | 'status' | 'author';
		tone?: 'neutral' | 'accent' | 'solid' | 'danger' | 'warning' | 'online' | 'outline';
		class?: string;
		style?: string;
		children?: Snippet;
	} = $props();

	const SIZES = {
		sm: 'h-5 px-2 text-xxs',
		md: 'h-6 px-2.5 text-xs',
		// 状态/类型药丸：20px 高 + 11px Medium；水平垂直居中由根类的
		// inline-flex/items-center/justify-center/leading-none 统一承载。
		status: 'h-5 px-2.5 text-xs',
		// 作者微徽章：16px 高、10px Medium、px-1 使「作者」两字落成 28px 宽；
		// 4px 圆角属微徽章物理形状细件（圆角 4 档之外的唯一登记例外，
		// 见 docs/ui-components.md「Badge」行），不得扩散到其它徽章。
		author: 'h-4 px-1 rounded-[4px] text-xxs'
	} as const;
	// 本地合并器：tailwind-merge 内置字号刻度不认识项目自定义的 `text-xxs`（10px），
	// 会把它误判为文字颜色，与 tone 的文字色同批合并时被静默丢弃 → 徽章退化为继承字号
	// （父级 16px 时撑破 20/16px 高度）。这里把 xxs 注册进字号组，使 size 与 tone 共存；
	// 仅作用于 Badge，不改全局 cn 的合并行为。
	const mergeBadge = extendTailwindMerge({
		extend: { classGroups: { 'font-size': [{ text: ['xxs'] }] } }
	});

	const TONES = {
		neutral: 'bg-surface-subtle text-text-secondary',
		accent: 'bg-accent-soft text-text-accent-on-soft',
		solid: 'bg-accent-default text-text-on-accent',
		// danger 用整档 --cst-danger-bg(13%)：旧实现 /50 折半到 6.5%，与 warning/online 的整档
		// 不同构（用透明度造第 4 档，style skill 禁止），深浅主题下肉眼可见偏淡。
		danger: 'bg-danger-bg text-text-status-danger-on-soft',
		warning: 'bg-status-warning-bg text-text-status-warning-on-soft',
		online: 'bg-status-online-bg text-text-status-online-on-soft',
		outline: 'border border-border-default bg-transparent text-text-secondary'
	} as const;
</script>

<span
	{style}
	class={mergeBadge(
		'inline-flex items-center justify-center gap-1 whitespace-nowrap rounded-full font-medium',
		SIZES[size],
		// leading-none 必须排在字号类之后：tailwind-merge 认为 text-* 自带行高、会移除
		// 先出现的 leading-*（徽标契约要求两者同时声明，顺序写反会导致单行行高丢失）。
		'leading-none',
		TONES[tone],
		className
	)}
>
	{@render children?.()}
</span>
