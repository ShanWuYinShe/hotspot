<script lang="ts">
	import { untrack, type Snippet } from 'svelte';
	import { ArrowLeft } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import ContentColumn from './ContentColumn.svelte';
	import { platform } from '@/stores/platform.svelte';
	import { headerIsland, type IslandToken } from '@/stores/header-island.svelte';
	import { islandRectOf, islandTextRectOf, type IslandRect } from '@/lib/island-morph';
import {
	anchorHandoffProgress,
	handoffRecede,
	scrollHandoffProgress,
	subscribeHandoff
} from '@/lib/island-handoff';
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
	// variant="floating"（2026-10-29，详情页沉浸式）：**整行收进一块悬浮玻璃工具条**
	// （返回 + 标题 + 更多在同一面板内，圆角/细边/半透明底/轻阴影），不再让两枚按钮
	// 各自孤立地浮在内容上——用户反馈「只有两个按钮悬浮很奇怪，要整行区域的和谐感」。
	// 工具条吸顶，内容从其下方滚过并透出毛玻璃；没有通栏硬边，不形成割裂横条。
	// 依据 NN/g「Sticky headers」：页头只有在不抢内容、chrome 占比小时才成立。
	// 注：瞬间详情页已于同日改为「内容内极简工具行 + 画廊 hero」，本变体保留为通用沉浸页原语。
	//
	// variant="integrated"（2026-10-29，hub/列表页「内容内页头」）：桌面/平板无背景/边框/吸顶，
	// 移动端（<600px）例外吸顶（画布 85% + blur，保留筛选入口），
	// 标题 + 副标题（可含计数）+ 工具同一行，筛选 below 紧随其后；与正文同处一块画布，
	// 消除大块页头/重复分区标题带来的割裂感。用于附近人/群组/瞬间、关系、举报、通知。

	let {
		title,
		subtitle,
		subtitleIcon,
		onBack,
		backLabel = '返回',
		trailing,
		islandTrailing,
		island = true,
		islandMeta,
		islandIcon,
		islandExpanded,
		below,
		sticky = true,
		showBackOnDesktop = false,
		desktopVisible = false,
		maxWidth,
		/** 标题字号档位（--text-* 刻度，替代内联 px：18→lg、20→xl、22/24→2xl、16→base）。 */
		titleSize = 'lg',
		/** 副标题字号档位（默认 11 = xs）。 */
		subtitleSize = 'xs',
		/** glass：半透明毛玻璃背景（对齐 Figma nearby 页 sticky header）。 */
		glass = false,
		/**
		 * floating：沉浸式页头——整行玻璃工具条（返回/标题/更多同一面板，吸顶）。
		 * integrated：hub/列表页内容内页头——桌面/平板无背景/边框/吸顶（移动端例外吸顶），标题/副标题/工具一行，
		 * 筛选 below 紧随；与正文同一画布。
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
		/** 桌面端顶栏灵动岛的右侧行动区（≤2 个控件）；桌面端页头整体隐藏，移动/平板仍用页头 trailing。 */
		islandTrailing?: Snippet;
		/** 桌面端是否注册顶栏灵动岛（默认 true；floating 变体不注册，面板内嵌页头可传 false）。 */
		island?: boolean;
		/** 岛上的副信息（缺省用 subtitle）。 */
		islandMeta?: string;
		/** 桌面端顶栏灵动岛的左侧语义图标（可选；P1-4 视觉锚点）。 */
		islandIcon?: Snippet;
		/** 展开态控制中枢自定义内容（可选；未传且有 below 时自动把 below 映射进岛面板） */
		islandExpanded?: Snippet;
		below?: Snippet;
		sticky?: boolean;
		showBackOnDesktop?: boolean;
		desktopVisible?: boolean;
		maxWidth?: number | string;
		titleSize?: 'md' | 'base' | 'lg' | 'xl' | '2xl' | '3xl';
		subtitleSize?: 'xxs' | 'xs' | 'sm' | 'md';
		glass?: boolean;
		variant?: 'default' | 'floating' | 'integrated';
		class?: string;
		/** 内部 ContentColumn 的 className（默认由 headerPadding 派生：有 below 时 pt-4 pb-3.5、
		    无 below 时 pt-4.5 pb-4.5；仅已登记的窄列特例传值覆盖） */
		contentClassName?: string;
	} = $props();

	const floating = $derived(variant === 'floating');
	const integrated = $derived(variant === 'integrated');

	// ── 页面锚点岛（Page Island，仅桌面端）：注册即常驻 ─────────────────
	// 桌面端页头默认整体隐藏（desktopVisible=false），标题/上下文/行动交给顶栏常驻岛；
	// 岛不做滚动显隐/飞行动画；页头仍按滚动位置产出交接帧 p（lib/island-handoff.ts），
	// 仅用于页面侧 band 退场、inert/aria 与测试钩子；token 保证路由切换/多页头时不串台。

	const islandToken: IslandToken = Symbol('page-header-island');
	const islandEnabled = $derived(island && !floating && platform.platform === 'desktop');
	/** 吸顶页头：标题不会自己滚走，用滚动窗口驱动交接；integrated 页头则用锚点位置。 */
	const islandStickyTrigger = $derived(!integrated && sticky);
	/**
	 * 外层 sticky 容器探测（如 SplitPageLayout 桌面分支把 header 包在 sticky 容器里）：
	 * integrated 页头此时也不会滚出视口，必须改用滚动窗口交接。
	 */
	let stickyAncestor = $state(false);
	const islandHandoffTrigger = $derived(islandStickyTrigger || stickyAncestor);
	/** 岛是否已在交接中（p>0）；用于测试钩子与 inert。 */
	let islandActive = $state(false);
	/** 交接是否已基本落位（p>0.55）：整条页头此时才去掉指针/读屏可达性。 */
	let islandSettled = $state(false);
	/** 交接视觉锚点：标题文字那一块（块级 h1 的整列宽度不能当锚点）。 */
	let islandMorphEl = $state<HTMLElement | undefined>(undefined);
	let islandAnchorEl = $state<HTMLElement | undefined>(undefined);
	/** 吸顶页头：标题不动，锚点矩形抓一次快照即可。 */
	let stickySource: IslandRect | null = null;
	/** 非吸顶页头：缓存锚点的文档绝对位置（避免滚动每帧 getBoundingClientRect 引起布局抖动）。 */
	let anchorBox: { x: number; width: number; height: number; absTop: number } | null = null;

	const barBottom = (): number => Math.round(52 * Math.max(1, platform.uiScale));

	function measureAnchor(scroller: HTMLElement | null): void {
		const rect = islandTextRectOf(islandMorphEl) ?? islandRectOf(islandAnchorEl);
		if (!rect) return;
		const scrollTop = scroller?.scrollTop ?? 0;
		anchorBox = {
			x: rect.x,
			width: rect.width,
			height: rect.height,
			absTop: rect.y + scrollTop
		};
	}

	/** 非吸顶锚点当前帧的视口矩形：位置 = 绝对 top - 滚动量（纯算术，不读布局）。 */
	function anchorSource(scrollTop: number): IslandRect | null {
		if (!anchorBox) return null;
		return {
			x: anchorBox.x,
			y: anchorBox.absTop - scrollTop,
			width: anchorBox.width,
			height: anchorBox.height
		};
	}

	/**
	 * 整条页头隐去（吸顶页头专用）：标题/上下文/筛选都已交给岛，页头 chrome 不再占视线。
	 * 保留高度只改透明度 → 不产生 CLS；筛选状态由页面写进 islandMeta 带进岛。
	 */
	const islandBandHidden = $derived(islandEnabled && islandHandoffTrigger && islandSettled);
	const effectiveExpanded = $derived(islandExpanded ?? (below ? defaultHeaderExpanded : undefined));

	$effect(() => {
		if (!islandEnabled) return;
		headerIsland.register(
			islandToken,
			untrack(() => ({
				title,
				meta: islandMeta ?? subtitle ?? '',
				icon: islandIcon,
				trailing: islandTrailing,
				expanded: effectiveExpanded,
				onBack: showBack ? onBack : undefined,
				backLabel
			}))
		);
		return () => headerIsland.release(islandToken);
	});

	$effect(() => {
		if (!islandEnabled) return;
		headerIsland.update(islandToken, {
			title,
			meta: islandMeta ?? subtitle ?? '',
			icon: islandIcon,
			trailing: islandTrailing,
			expanded: effectiveExpanded,
			onBack: showBack ? onBack : undefined,
			backLabel
		});
	});

	// 把交接帧里的 p 映射成少量响应式状态（active/settled）；几何留给 action 逐帧写，避开 60fps 响应式更新。
	$effect(() => {
		if (!islandEnabled) return;
		return subscribeHandoff((frame) => {
			const active = frame.p > 0.001;
			const settled = frame.p > 0.55;
			if (active !== islandActive) islandActive = active;
			if (settled !== islandSettled) islandSettled = settled;
		});
	});

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
		// 检测外层 sticky 容器：integrated 页头被 sticky 包裹时标题恒在视口内，
		// IntersectionObserver 永不触发（/notify 的 SplitPageLayout 桌面分支即此场景）。
		let anc = node.parentElement;
		while (anc && anc !== document.body) {
			if (getComputedStyle(anc).position === 'sticky') {
				stickyAncestor = true;
				break;
			}
			anc = anc.parentElement;
		}
		let raf = 0;
		const apply = () => {
			raf = 0;
			const y = target.scrollTop;
			const uiScale = Math.max(1, platform.uiScale);
			scrolled = y > 4 * uiScale;
			if (!islandEnabled) return;
			if (islandHandoffTrigger) {
				if (!stickySource) {
					const captured = islandTextRectOf(islandMorphEl) ?? islandRectOf(islandAnchorEl);
					stickySource = captured;
				}
				const progress = scrollHandoffProgress(y, 120 * uiScale, 260 * uiScale);
				const isScrolled = progress >= 0.5;
				headerIsland.setHandoff(
					islandToken,
					isScrolled ? 1 : 0,
					stickySource
				);
				return;
			}
			// 非吸顶页头：标题随内容滚进顶栏，在标题接近顶栏（尚在视野上部清晰可见）时触发灵动岛流体转换动效。
			if (!anchorBox || y < 1) measureAnchor(target);
			const source = anchorSource(y);
			const isScrolled = source
				? source.y <= barBottom() + 38
				: y > 40 * uiScale;
			headerIsland.setHandoff(
				islandToken,
				isScrolled ? 1 : 0,
				source
			);
		};
		const onScroll = () => {
			if (raf) return;
			raf = requestAnimationFrame(apply);
		};
		target.addEventListener('scroll', onScroll, { passive: true });
		const onResize = () => {
			anchorBox = null;
			stickySource = null;
			apply();
		};
		window.addEventListener('resize', onResize);
		measureAnchor(target);
		apply();
		return {
			destroy() {
				target.removeEventListener('scroll', onScroll);
				window.removeEventListener('resize', onResize);
				if (raf) cancelAnimationFrame(raf);
			}
		};
	}

	// 显式映射表：完整类名字面量可被 Tailwind 扫描器检测（动态拼接 text-{x} 不会）。
	// 3xl+semibold：顶级 tab 页大标题档（对齐 Figma Make 移动端 display 标题——28px 级
	// 半粗是页面的视觉锚点，与正文拉开层级；字重统一 semibold，不用 bold）。
	const TITLE_SIZE_CLASS: Record<'md' | 'base' | 'lg' | 'xl' | '2xl' | '3xl', string> = {
		md: 'text-md',
		base: 'text-md',
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

	// 返回入口判定：移动/平板统一渲染；桌面端由 showBackOnDesktop 门控（或沉浸 floating 模式恒开启），避免顶层页多余返回
	const showBack = $derived(
		Boolean(onBack) && (platform.platform !== 'desktop' || showBackOnDesktop || floating)
	);
	// header 统一预留黄金呼吸感：
	// - 移动端黄金比例：pt-2.5 pb-2（对齐首页顶栏紧凑通透语言，释放视口）；
	// - 桌面端舒展呼吸：纯标题留 pb-4.5，有 below 时留 pb-3.5。
	const headerPadding = $derived(
		integrated
			? below
				? 'pt-3 pb-1 lg:pt-3.5 lg:pb-2.5'
				: 'pt-3 pb-1 lg:py-0'
			: floating
				? 'pt-2 pb-2 lg:py-0'
				: below
					? 'pt-2.5 sm:pt-4 pb-2 sm:pb-3.5 lg:pt-3.5 lg:pb-2.5'
					: 'pt-2.5 sm:pt-4.5 pb-2 sm:pb-4.5 lg:py-0'
	);
</script>

{#snippet defaultHeaderExpanded()}
	<div class="flex flex-col gap-2.5">
		{#if below}
			<div class="w-full">
				{@render below()}
			</div>
		{/if}
		{#if islandTrailing || trailing}
			<div class="flex items-center gap-2 pt-0.5">
				{#if islandTrailing}
					{@render islandTrailing()}
				{:else if trailing}
					{@render trailing()}
				{/if}
			</div>
		{/if}
	</div>
{/snippet}

<header
	use:headerScroll
	use:handoffRecede={{ enabled: islandEnabled && islandHandoffTrigger, mode: 'band' }}
	data-island-visible={islandEnabled && islandActive ? 'true' : undefined}
	data-island-band-hidden={islandBandHidden ? 'true' : undefined}
	aria-hidden={islandBandHidden ? 'true' : undefined}
	inert={islandBandHidden}
	class={cn(
		// 页头 band 退场过渡（桌面端页头隐藏时无可见效果；移动/平板不启用岛）
		'shrink-0',
		islandHandoffTrigger && 'transition-opacity duration-snappy ease-standard',
		integrated
			? 'max-sm:sticky max-sm:top-0 max-sm:z-[var(--cst-z-sticky)] max-sm:bg-surface-page/85 max-sm:backdrop-blur-md max-sm:pb-1.5'
			: floating
				? 'sticky top-0 z-[var(--cst-z-sticky)] pointer-events-none'
				: [
						'page-header-scroll max-sm:bg-surface-page/85 max-sm:backdrop-blur-md max-sm:border-b-0',
						sticky && 'sticky top-0 z-[var(--cst-z-sticky)]',
						scrolled && 'page-header-scrolled',
						// 页头 band 已由 handoffRecede 逐帧淡出（可逆），这里只切断指针事件。
						islandBandHidden && 'pointer-events-none'
					],
		!desktopVisible && 'lg:hidden',
		className
	)}
>
	<ContentColumn {maxWidth} class={cn(headerPadding, contentClassName)}>
		{#if integrated}
			<!-- hub/列表页「内容内页头」：标题 + 副标题/计数 + 工具同一行；筛选紧随；桌面/平板无背景/边框/吸顶，移动端例外吸顶 -->
			<!-- 桌面端整条页头隐藏（`lg:hidden`），标题/筛选由常驻灵动岛承担；移动/平板照常渲染 -->
			<div data-page-intro bind:this={islandAnchorEl} class="flex flex-wrap items-center justify-between gap-x-3 gap-y-1.5">
				<div
					data-island-handoff={islandActive ? 'true' : undefined}
					class="flex min-w-0 flex-1 flex-wrap items-baseline gap-x-2.5 gap-y-1"
				>
					{#if showBack}
						<div class="pointer-events-auto flex items-center rounded-full border border-border-default/60 bg-surface-primary/85 text-text-primary p-0.5 shadow-raised-sm backdrop-blur-md shrink-0 self-center -ml-0.5">
							<ActionIconButton
								variant="ghost"
								tone="current"
								icon={ArrowLeft}
								label={backLabel}
								onClick={onBack}
								size={36}
								class="shrink-0"
							/>
						</div>
					{/if}
					<h1 bind:this={islandMorphEl} class="min-w-0 truncate text-lg font-semibold text-text-primary sm:text-xl">{title}</h1>
					{#if subtitle}
						<p class="inline-flex min-w-0 items-center gap-1.5 rounded-full border border-border-default/30 bg-surface-island/60 px-2.5 py-0.5 text-xs text-text-secondary backdrop-blur-xl shadow-xs">
							{#if subtitleIcon}
								<span class="flex shrink-0 items-center text-text-secondary">{@render subtitleIcon()}</span>
							{/if}
							<span class="truncate">{subtitle}</span>
						</p>
					{/if}
				</div>
				<!-- 桌面端页头隐藏，行动区由岛 trailing 承担；移动/平板仍渲染本 trailing（岛不出现） -->
				{#if trailing}
					<!-- gap-3：ActionIconButton 自带 hit-slop 横向 -6px，8px 间距会让相邻热区重叠 -->
					<div class="flex shrink-0 items-center gap-3">
						{@render trailing()}
					</div>
				{/if}
			</div>
			{#if below}
				<div data-island-handoff={islandActive ? 'true' : undefined} class="mt-2.5">
					{@render below()}
				</div>
			{/if}
		{:else if floating}
			<!-- 整行悬浮工具条：返回 / 标题 / 更多收进同一块玻璃面板，而不是两枚孤立圆钮 -->
			<div
				data-floating-toolbar
				class="pointer-events-auto flex items-center gap-2 rounded-md border border-border-default/60 bg-surface-primary/85 px-1.5 py-1 shadow-raised-sm backdrop-blur-md"
			>
				{#if showBack}
					<div class="pointer-events-auto flex items-center rounded-full border border-border-default/60 bg-surface-primary/85 text-text-primary p-0.5 shadow-raised-sm backdrop-blur-md shrink-0">
						<ActionIconButton
							icon={ArrowLeft}
							label={backLabel}
							onClick={onBack}
							variant="ghost"
							tone="current"
							size={36}
							class="shrink-0"
						/>
					</div>
				{:else}
					<span class="size-[2.375rem] shrink-0" aria-hidden="true"></span>
				{/if}
				<h1 class="min-w-0 flex-1 truncate text-center text-md font-semibold text-text-primary">{title}</h1>
				{#if trailing}
					<div class="shrink-0">{@render trailing()}</div>
				{:else}
					<span class="size-[2.375rem] shrink-0" aria-hidden="true"></span>
				{/if}
			</div>
		{:else}
			<div class="flex items-center gap-3">
				{#if showBack}
					<!-- 全局统一返回按钮：38px 现代微毛玻璃圆钮（p-0.5 + 36px 幽灵钮），触控区经 hit-slop 达 44px；
					     入岛后返回动作移到岛上（leading），页头返回淡出，避免同一动作出现两份 -->
					<div class="pointer-events-auto flex items-center rounded-full border border-border-default/60 bg-surface-primary/85 text-text-primary p-0.5 shadow-raised-sm backdrop-blur-md shrink-0">
						<ActionIconButton
							icon={ArrowLeft}
							label={backLabel}
							onClick={onBack}
							variant="ghost"
							tone="current"
							size={36}
							class="shrink-0"
						/>
					</div>
				{/if}
			<!-- 标题列恒为弹性列（flex-1 min-w-0）：窄屏下标题区先让位收缩，
			     trailing 操作簇由 shrink-0 锁定不被压缩；溢出优先级 = 操作钮 > 装饰 chip > 副标题尾字。
			    （此前有 trailing 时标题列无 flex-1，trailing 又无 shrink-0，360px 级窄屏下
			     在线 chip 与副标题同时被压折行。） -->
			<div
				bind:this={islandAnchorEl}
				data-island-handoff={islandActive ? 'true' : undefined}
				class="min-w-0 flex-1"
			>
				<h1 bind:this={islandMorphEl} class="text-text-primary font-semibold leading-none {TITLE_SIZE_CLASS[titleSize]}">{title}</h1>
				{#if subtitle}
					<div class="mt-1.5">
						<p class="inline-flex min-w-0 items-center gap-1.5 rounded-full border border-border-default/30 bg-surface-island/60 px-2.5 py-0.5 text-xs text-text-secondary backdrop-blur-xl shadow-xs {SUBTITLE_SIZE_CLASS[subtitleSize]}">
							{#if subtitleIcon}
								<span class="flex shrink-0 items-center text-text-secondary">{@render subtitleIcon()}</span>
							{/if}
							<span class="truncate">{subtitle}</span>
						</p>
					</div>
				{/if}
			</div>
			<!-- 桌面端页头隐藏，行动区由岛 trailing 承担；移动/平板仍渲染本 trailing（岛不出现） -->
			{#if trailing}
				<div class="shrink-0">
					{@render trailing()}
				</div>
			{/if}
		</div>
		{/if}
		{#if below && !floating && !integrated}
			<div class="mt-2.5">
				{@render below()}
			</div>
		{/if}
	</ContentColumn>
</header>

{#if !desktopVisible}
	<!-- 桌面端页头整条隐藏（lg:hidden），但 h1 不能随视觉消失：保留唯一可访问一级标题
	     （移动端 display:none 不重复，桌面端 1px 视觉隐藏仍在无障碍树内；axe page-has-heading-one）。 -->
	<h1 class="sr-only hidden lg:block">{title}</h1>
{/if}