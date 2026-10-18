<script lang="ts">
	import type { Component } from 'svelte';

	// TabBar — 胶囊分类/分段 Tab 栏（对齐鸿蒙 TabsPillGroup 分段式视觉，全站统一）。
// 选中态语言：pills 筛选胶囊用 accent-soft 淡底 + accent 字（弱高亮，亮 accent 实底
// 收给主按钮，避免筛选态与主操作抢焦点——Gemini UI 审查 r07/r11-r19 多轮共识）；
// segmented 分段控制器保持 accent 实底浮起胶囊（系统级分段，区别于内容筛选）。
	// 支持两种形态（对齐鸿蒙 TabsPillGroup）：
	// - variant="pills"（默认，流式胶囊导航）：透明容器，横向滚动，选项带 gap-1.5 安全呼吸间距，杜绝贴边粘连。
	// - variant="segmented"（系统级分段控制器）：surface-subtle 浅色底轨容器 + ring-1 ring-border-default + p-1 内距，
	//   选项等宽网格（grid grid-flow-col auto-cols-fr gap-1），激活项浮起胶囊，消除粘连推挤感。
	// 底部边界线由上下文提供（PageHeader below 场景由 header 底线承担，
	// 正文场景由调用方外层 border-b 承担），避免与 header 形成双线。
	// 用 data-tab-bar 标记，全局 button:hover 规则据此排除（导航 Tab 不应触发实色按压填充），
	// 改用极轻背景 + 胶囊圆角的 subtle hover（见 utilities.css 的 .tab-item 规则）。

	export type TabItem = { id: string; label: string; count?: number; icon?: Component };

	let {
		items,
		value,
		onChange,
		variant = 'pills',
		class: className = ''
	}: {
		items: TabItem[];
		value: string;
		onChange: (id: string) => void;
		variant?: 'pills' | 'segmented';
		class?: string;
	} = $props();

	// 横向滚动渐隐提示（全站复审 C1）：后台 10 页的模块导航在 390px 下藏 328px，
	// 5 个入口默认不可见且原本无任何提示（no-scrollbar 连滚动条都隐掉了）。
	// 这里按真实溢出状态在两个方向分别加渐隐类，不改变单行布局与信息密度。
	let pillsEl = $state<HTMLDivElement | null>(null);
	let edges = $state({ left: false, right: false });
	const syncEdges = () => {
		const el = pillsEl;
		if (!el) return;
		const max = el.scrollWidth - el.clientWidth;
		edges = { left: el.scrollLeft > 2, right: max - el.scrollLeft > 2 };
	};
	$effect(() => {
		if (!pillsEl) return;
		syncEdges();
		if (typeof ResizeObserver === 'undefined') return;   // jsdom 无 RO
		const ro = new ResizeObserver(syncEdges);
		ro.observe(pillsEl);
		return () => ro.disconnect();
	});
	$effect(() => {
		void items.length;
		void value;
		queueMicrotask(syncEdges);
	});
</script>

{#if variant === 'segmented'}
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
				onclick={() => onChange(tab.id)}
				aria-pressed={isActive}
				data-active={isActive ? 'true' : undefined}
				class="tab-item flex items-center justify-center gap-1.5 rounded-full px-3 py-1.5 text-xs focus-ring transition-all duration-standard {isActive
					? 'font-medium bg-accent-default text-text-on-accent shadow-raised-sm'
					: 'font-normal text-text-secondary hover:text-text-primary'}"
			>
				{#if tab.icon}
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
		bind:this={pillsEl}
		onscroll={syncEdges}
		class="no-scrollbar flex items-center gap-1.5 overflow-x-auto {edges.left ? 'scroll-fade-left' : ''} {edges.right ? 'scroll-fade-right' : ''} {className}"
	>
		{#each items as tab}
			{@const isActive = value === tab.id}
			<button
				type="button"
				onclick={() => onChange(tab.id)}
				aria-pressed={isActive}
				class="tab-item flex shrink-0 items-center gap-1.5 rounded-full px-3.5 py-1.5 text-sm focus-ring transition-colors duration-standard {isActive
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
