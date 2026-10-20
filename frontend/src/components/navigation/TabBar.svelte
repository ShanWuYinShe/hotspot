<script lang="ts">
	import type { Component } from 'svelte';
	import { scrollFade } from '@/lib/scroll-fade';

	// TabBar — 胶囊分类/分段 Tab 栏（对齐鸿蒙 TabsPillGroup 分段式视觉，全站统一）。
// 选中态语言：pills 筛选胶囊用 accent-soft 淡底 + accent 字（弱高亮，亮 accent 实底
// 收给主按钮，避免筛选态与主操作抢焦点——Gemini UI 审查 r07/r11-r19 多轮共识）；
// segmented 分段控制器保持 accent 实底浮起胶囊（系统级分段，区别于内容筛选）。
	// 支持两种形态（对齐鸿蒙 TabsPillGroup）：
	// - variant="pills"（默认，流式胶囊导航）：透明容器，横向滚动，选项带 gap-1.5 安全呼吸间距，杜绝贴边粘连。
	// - variant="segmented"（系统级分段控制器）：surface-subtle 浅色底轨容器 + ring-1 ring-border-default + p-1 内距，
	//   选项等宽网格（grid grid-flow-col auto-cols-fr gap-1），激活项浮起胶囊，消除粘连推挤感。
	// - variant="underline"（页级信息流切换，2026-10-20 新增）：**全宽、无轨道、无图标**，
	//   选项等分列，激活项 = 主文字色 + 底部 2px accent 滑动下划线（transition 位移，不重排）。
	//   用于首页「关注/发现」这类"页级两态切换"：浮动胶囊在窄屏上视觉过重、且与页头留白打架。
	// 底部边界线由上下文提供（PageHeader below 场景由 header 底线承担，
	// 正文场景由调用方外层 border-b 承担），避免与 header 形成双线。
	// 用 data-tab-bar 标记，全局 button:hover 规则据此排除（导航 Tab 不应触发实色按压填充），
	// 改用极轻背景 + 胶囊圆角的 subtle hover（见 utilities.css 的 .tab-item 规则）。

	export type TabItem = { id: string; label: string; count?: number; icon?: Component;
		/** 主题/色板选择器用：标签前的色点（值来自 THEME_OPTIONS.swatch）。 */
		swatch?: string };

	let {
		items,
		value,
		onChange,
		variant = 'pills',
		disabled = false,
		class: className = ''
	}: {
		items: TabItem[];
		value: string;
		onChange: (id: string) => void;
		variant?: 'pills' | 'segmented' | 'underline';
		/** 整体禁用（如「可见性探测中」：选择器可读但不可改）。 */
		disabled?: boolean;
		class?: string;
	} = $props();

	// 横向滚动渐隐提示（全站复审 C1）：机制与判定收敛在 lib/scroll-fade.ts，
	// 表格容器等其它横滚区共用同一 action（一个交互意图只允许一种实现）。
</script>

{#if variant === 'underline'}
	<!-- 页级信息流切换：全宽、无轨道、无图标；下划线随激活项滑动（transform，不触发布局） -->
	<div data-tab-bar class="relative flex w-full items-stretch {className}">
		{#each items as tab}
			{@const isActive = value === tab.id}
			<button
				type="button"
				disabled={disabled}
				onclick={() => onChange(tab.id)}
				aria-pressed={isActive}
				data-active={isActive ? 'true' : undefined}
				class="tab-item focus-ring flex flex-1 items-center justify-center py-2.5 text-md transition-colors duration-standard disabled:cursor-not-allowed disabled:opacity-dim {isActive
					? 'font-medium text-text-primary'
					: 'font-normal text-text-secondary hover:text-text-primary'}"
			>
				{tab.label}
			</button>
		{/each}
		<!-- 滑动下划线：宽度 = 1/n，位移 = 激活索引 × 100% -->
		<span
			aria-hidden="true"
			class="pointer-events-none absolute bottom-0 left-0 h-0.5 bg-accent-default transition-transform duration-standard ease-standard"
			style="width:{100 / Math.max(1, items.length)}%;transform:translateX({Math.max(0, items.findIndex((t) => t.id === value)) * 100}%)"
		></span>
	</div>
{:else if variant === 'segmented'}
	<div
		data-tab-bar
		class="grid grid-flow-col auto-cols-fr gap-1 rounded-full bg-surface-subtle p-1 ring-1 ring-border-default/70 {className}"
	>
		{#each items as tab}
			{@const isActive = value === tab.id}
			<!-- 语义与 pills 一致用 aria-pressed（内容筛选器，非 ARIA tabs 模式——
			     本组件无 tabpanel 关联与方向键导航，role=tab 是残缺实现） -->
			<button
				type="button"
				disabled={disabled}
				onclick={() => onChange(tab.id)}
				aria-pressed={isActive}
				data-active={isActive ? 'true' : undefined}
				class="tab-item hit-slop-y flex items-center justify-center gap-1.5 rounded-full px-3 py-2.5 text-xs focus-ring transition-all duration-standard disabled:cursor-not-allowed disabled:opacity-dim lg:py-1.5 {isActive
					? 'font-medium bg-accent-default text-text-on-accent shadow-raised-sm'
					: 'font-normal text-text-secondary hover:text-text-primary'}"
			>
				{#if tab.swatch}
					<!-- 主题色板点：让用户在选择前就看到各主题的主色（全站复审 s3 P2-5：
					     THEME_OPTIONS 提供了 swatch，此前页面未消费） -->
					<span
						class="size-3 shrink-0 rounded-full ring-1 ring-border-default"
						style="background:{tab.swatch};"
						aria-hidden="true"
					></span>
				{:else if tab.icon}
					<tab.icon class="size-3.5 shrink-0" strokeWidth={isActive ? 2 : 1.5} />
				{/if}
				<span class="truncate">{tab.label}</span>
				{#if tab.count !== undefined}
					<span class="text-xxs opacity-emphasis">{tab.count}</span>
				{/if}
			</button>
		{/each}
	</div>
{:else}
	<div
		data-tab-bar
		use:scrollFade
		class="no-scrollbar flex items-center gap-1.5 overflow-x-auto py-1 {className}"
	>
		{#each items as tab}
			{@const isActive = value === tab.id}
			<button
				type="button"
				disabled={disabled}
				onclick={() => onChange(tab.id)}
				aria-pressed={isActive}
				class="tab-item hit-slop-y flex shrink-0 items-center gap-1.5 rounded-full px-3.5 py-3 text-sm focus-ring transition-colors duration-standard disabled:cursor-not-allowed disabled:opacity-dim lg:py-1.5 {isActive
					? 'font-medium bg-accent-soft text-accent-default'
					: 'font-normal text-text-secondary hover:text-text-primary'}"
			>
				{#if tab.icon}
					<tab.icon class="size-3.5" strokeWidth={isActive ? 2 : 1.5} />
				{/if}
				{tab.label}
				{#if tab.count !== undefined}
					<span class="text-xxs opacity-emphasis">{tab.count}</span>
				{/if}
			</button>
		{/each}
	</div>
{/if}
