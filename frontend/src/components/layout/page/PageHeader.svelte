<script lang="ts">
	import type { Snippet } from 'svelte';
	import { ArrowLeft } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import ContentColumn from './ContentColumn.svelte';
	import { platform } from '@/stores/platform.svelte';
	import { cn } from '@/lib/utils';

	// 布局组件（layout/page）。
	// PageHeader — 与正文共用同一 ContentColumn gutter 的页面页头。
	// 对齐 Figma Make layout/page/PageHeader 契约：
	//   - 内容包在 ContentColumn 内，左/右边缘与页面正文严格对齐
	//   - 实底（glass=false）：.page-header-solid（外壳感知：移动/平板=画布色，
//     桌面白浮卡内=白色；sticky 时与滚动内容同底，不透出滚动文字）
	//   - 返回按钮默认仅非桌面显示；showBackOnDesktop 可显式开启
	//
	// ┌──────────────── 路由页面 ──────────────────────────┐
	// │ ┌──────── ContentColumn ────────┐                  │
	// │ │ ←  标题              trailing  │  (sticky 可选)   │
	// │ └───────────────────────────────┘                  │
	// └────────────────────────────────────────────────────┘
	// 滚动感知（Round 9）：页头在顶部透明（与画布融合、不再出现 78% 白底带），
	// 滚动 > 4px 后切换 glass（surface-page 78% + blur + 细底边/轻阴影），150ms 过渡；
	// reduced-motion 由全局规则瞬时切换。监听最近的滚动容器，passive + rAF 节流。
	// 设备范围：桌面、平板和手机通用。
	//
	// variant="floating"（2026-10-29，详情页沉浸式）：**永远不画整条背景/底边/阴影**，
	// 只保留左侧返回圆钮 + 右侧操作圆钮悬浮在画布上（标题仅 sr-only 保留语义）。
	// 依据 NN/g「Sticky headers」：持久页头只有在不抢内容、chrome 占比最小时才成立；
	// 整条 glass 带在图片详情页会把内容切出一条明显横杠（用户反馈的真实问题）。
	// 2026-10-29 补充（用户反馈「按钮压在图片上像内容重叠」）：floating 支持 sticky={false}
	// 配合 SplitPageLayout headerSticky={false}，让工具条留在文档流、随正文滚走——
	// 阅读/评论时零遮挡；返回走浏览器/系统返回与全局「回到顶部」。

	let {
		title,
		subtitle,
		subtitleIcon,
		onBack,
		backLabel = '返回',
		trailing,
		below,
		sticky = true,
		showBackOnDesktop = false,
		maxWidth,
		/** 标题字号档位（--text-* 刻度，替代内联 px：18→lg、20→xl、22/24→2xl、16→base）。 */
		titleSize = 'lg',
		/** 副标题字号档位（默认 11 = xs）。 */
		subtitleSize = 'xs',
		/** glass：半透明毛玻璃背景（对齐 Figma nearby 页 sticky header）。 */
		glass = false,
		/**
		 * floating：沉浸式页头（详情页专用）——无整条背景/底边/阴影，仅悬浮圆钮；
		 * 标题仅保留 sr-only h1 供读屏与路由播报，visual 上不出现标题横条。
		 */
		variant = 'default',
		class: className = '',
		contentClassName = ''
	}: {
		title: string;
		subtitle?: string;
		/** 副标题行内小图标（对齐 Figma nearby/groups 的 MapPin 副标题）。 */
		subtitleIcon?: Snippet;
		onBack?: () => void;
		backLabel?: string;
		trailing?: Snippet;
		below?: Snippet;
		sticky?: boolean;
		showBackOnDesktop?: boolean;
		maxWidth?: number | string;
		titleSize?: 'md' | 'base' | 'lg' | 'xl' | '2xl' | '3xl';
		subtitleSize?: 'xxs' | 'xs' | 'sm' | 'md';
		glass?: boolean;
		variant?: 'default' | 'floating';
		class?: string;
		/** 内部 ContentColumn 的 className（默认由 headerPadding 派生：有 below 时 pt-4 pb-3.5、
		    无 below 时 pt-4.5 pb-4.5；仅已登记的窄列特例传值覆盖） */
		contentClassName?: string;
	} = $props();

	const floating = $derived(variant === 'floating');

	// glass prop 保留兼容（历史调用）；统一改为「顶部透明 → 滚动后 glass」，不再直接消费。
	let scrolled = $state(false);

	// 滚动监听改为 Svelte action：action 在节点挂载时执行、组件重挂/平台重排时
	// 自动重建监听，避免「onMount + $state ref 绑定时序」竞态（主环境 8888 实测
	// 生产构建下 listener 未挂到 main 的问题）。向上找最近 overflow 滚动容器，
	// 回退 #main-content；passive + rAF 节流，destroy 清理。
	function headerScroll(node: HTMLElement) {
		let scroller: HTMLElement | null = node.parentElement;
		while (scroller) {
			const st = getComputedStyle(scroller);
			if (/(auto|scroll)/.test(st.overflowY)) break;
			scroller = scroller.parentElement;
		}
		if (!scroller) scroller = document.getElementById('main-content');
		if (!scroller) return;
		const target = scroller;
		let raf = 0;
		const apply = () => {
			raf = 0;
			scrolled = target.scrollTop > 4;
		};
		const onScroll = () => {
			if (raf) return;
			raf = requestAnimationFrame(apply);
		};
		target.addEventListener('scroll', onScroll, { passive: true });
		apply();
		return {
			destroy() {
				target.removeEventListener('scroll', onScroll);
				if (raf) cancelAnimationFrame(raf);
			}
		};
	}

	// 显式映射表：完整类名字面量可被 Tailwind 扫描器检测（动态拼接 text-{x} 不会）。
	// 3xl+semibold：顶级 tab 页大标题档（对齐 Figma Make 移动端 display 标题——28px 级
	// 半粗是页面的视觉锚点，与正文拉开层级；字重统一 semibold，不用 bold）。
	const TITLE_SIZE_CLASS: Record<'md' | 'base' | 'lg' | 'xl' | '2xl' | '3xl', string> = {
		md: 'text-md',
		base: 'text-base',
		lg: 'text-lg',
		xl: 'text-xl',
		'2xl': 'text-2xl',
		'3xl': 'text-3xl font-semibold'
	};
	const SUBTITLE_SIZE_CLASS: Record<'xxs' | 'xs' | 'sm' | 'md', string> = {
		xxs: 'text-xxs',
		xs: 'text-xs',
		sm: 'text-sm',
		md: 'text-md'
	};

	// floating 是详情/沉浸页：返回是唯一离开路径，桌面也必须显示
	const showBack = $derived(Boolean(onBack) && (floating || platform.platform !== 'desktop' || showBackOnDesktop));
	// header 统一预留黄金呼吸感：
	// - 纯标题时标题/副标题/操作组与下边框之间留 pb-4.5（18px），上下留白匀称舒展，杜绝标题与分割线紧贴粘连；
	// - below 存在时（搜索框/Tab 栏等），给 below 内容与下边框之间留 pb-3.5（14px）。
	const headerPadding = $derived(floating ? 'pt-3 pb-1' : below ? 'pt-4 pb-3.5' : 'pt-4.5 pb-4.5');
</script>

<header
	use:headerScroll
	class={cn(
		floating
			? ['shrink-0 pointer-events-none', sticky && 'sticky top-0 z-[var(--cst-z-sticky)]']
			: ['shrink-0 page-header-scroll', sticky && 'sticky top-0 z-[var(--cst-z-sticky)]', scrolled && 'page-header-scrolled'],
		className
	)}
>
	<ContentColumn {maxWidth} class={cn(headerPadding, contentClassName)}>
			<div class="flex items-center gap-3">
				{#if showBack}
					<!-- 42：返回是导航关键动作，视觉与命中区大于普通 header 图标钮（36），易点不误触 -->
					<div class={floating ? 'pointer-events-auto' : ''}>
						<ActionIconButton icon={ArrowLeft} label={backLabel} onClick={onBack} size={42} />
					</div>
				{/if}
			<!-- 标题列恒为弹性列（flex-1 min-w-0）：窄屏下标题区先让位收缩，
			     trailing 操作簇由 shrink-0 锁定不被压缩；溢出优先级 = 操作钮 > 装饰 chip > 副标题尾字。
			    （此前有 trailing 时标题列无 flex-1，trailing 又无 shrink-0，360px 级窄屏下
			     在线 chip 与副标题同时被压折行。） -->
			<div class="min-w-0 flex-1">
				{#if floating}
					<!-- 沉浸式：视觉上不出现标题横条，只保留读屏/SEO 语义 -->
					<h1 class="sr-only">{title}</h1>
				{:else}
					<h1 class="text-text-primary font-semibold leading-none {TITLE_SIZE_CLASS[titleSize]}">{title}</h1>
					{#if subtitle}
						<p class="mt-1 flex min-w-0 items-center gap-1 text-text-secondary {SUBTITLE_SIZE_CLASS[subtitleSize]}">
							{#if subtitleIcon}
								<span class="flex shrink-0 items-center">{@render subtitleIcon()}</span>
							{/if}
							<span class="truncate">{subtitle}</span>
						</p>
					{/if}
				{/if}
			</div>
			{#if trailing}
				<div class={floating ? 'shrink-0 pointer-events-auto' : 'shrink-0'}>{@render trailing()}</div>
			{/if}
		</div>
		{#if below && !floating}
			{@render below()}
		{/if}
	</ContentColumn>
</header>
