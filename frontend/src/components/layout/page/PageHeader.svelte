<script lang="ts">
	import { untrack, type Snippet } from 'svelte';
	import { ArrowLeft } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import ContentColumn from './ContentColumn.svelte';
	import { platform } from '@/stores/platform.svelte';
	import { headerIsland, type IslandToken } from '@/stores/header-island.svelte';
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
		/** 桌面端滚动后交给顶栏灵动岛的右侧行动区（≤2 个控件）。提供后页头 trailing 入岛时隐藏，避免重复。 */
		islandTrailing?: Snippet;
		/** 桌面端滚动后是否把标题/上下文交给顶栏灵动岛（默认开启；面板内嵌页头可传 false）。 */
		island?: boolean;
		/** 岛上的副信息（缺省用 subtitle）。 */
		islandMeta?: string;
		below?: Snippet;
		sticky?: boolean;
		showBackOnDesktop?: boolean;
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

	// ── 页面锚点岛（Page Island，仅桌面端）────────────────────────────
	// 把会随滚动离开视口的标题/上下文固定到顶栏中央：吸顶页头用滚动阈值交接（标题淡出），
	// 非吸顶页头用 IntersectionObserver（标题滚出视口即入岛）。token 保证路由切换/多页头时不串台。
	const islandToken: IslandToken = Symbol('page-header-island');
	const islandEnabled = $derived(island && !floating && platform.platform === 'desktop');
	/** 吸顶页头：标题不会自己滚走，用滚动阈值触发交接；非吸顶页头用标题离场触发。 */
	const islandStickyTrigger = $derived(!integrated && sticky);
	/**
	 * 外层 sticky 容器探测（如 SplitPageLayout 桌面分支把 header 包在 sticky 容器里）：
	 * integrated 页头此时也不会滚出视口，IO 永不触发——必须改用滚动阈值交接。
	 */
	let stickyAncestor = $state(false);
	const islandHandoffTrigger = $derived(islandStickyTrigger || stickyAncestor);
	let islandShown = $state(false);
	/**
	 * 桌面端「岛接管后隐去内容区残留标题栏」：
	 * 默认吸顶页头（无 below）的标题/上下文/行动都已交给顶栏灵动岛，滚动 >120px 后
	 * 内容区不应再留一条空的玻璃标题带（用户反馈：既已有灵动岛，旧标题栏没有存在必要）。
	 * 有 below 的页头是「功能工具行」（搜索框吸顶、后台模块导航），保留整条不隐去，
	 * 否则滚走后会丢失操作入口；它们不属于「标题栏」。
	 */
	const islandBandHidden = $derived(islandEnabled && islandShown && islandStickyTrigger && !below);
	let islandAnchorEl = $state<HTMLElement | undefined>(undefined);

	$effect(() => {
		if (!islandEnabled) return;
		headerIsland.register(
			islandToken,
			untrack(() => ({
				title,
				meta: islandMeta ?? subtitle ?? '',
				trailing: islandTrailing,
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
			trailing: islandTrailing,
			onBack: showBack ? onBack : undefined,
			backLabel
		});
	});

	$effect(() => {
		if (!islandEnabled) return;
		headerIsland.setVisible(islandToken, islandShown);
	});

	// 非吸顶页头：标题离开视口（扣掉 52px 顶栏）即入岛；回到视口即退出。
	$effect(() => {
		if (!islandEnabled || islandHandoffTrigger) return;
		const el = islandAnchorEl;
		if (!el) return;
		const io = new IntersectionObserver(
			([entry]) => {
				islandShown = !entry.isIntersecting;
			},
			{ rootMargin: `-${Math.round(52 * Math.max(1, platform.uiScale))}px 0px 0px 0px`, threshold: 0 }
		);
		io.observe(el);
		return () => io.disconnect();
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
			// 吸顶页头的标题交接：滚过首屏后标题交给顶栏岛（120/72 滞回，避免抖动）；
			// 阈值随大屏 UI 缩放（顶栏高度/页头间距都是 rem）。
			if (islandEnabled && islandHandoffTrigger) {
				if (!islandShown && y > 120 * uiScale) islandShown = true;
				else if (islandShown && y < 72 * uiScale) islandShown = false;
			}
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

	// 凡传入 onBack 的页面（二级页/流程页/详情页），桌面/移动端统一保持常驻返回入口
	const showBack = $derived(Boolean(onBack));
	// header 统一预留黄金呼吸感：
	// - 移动端黄金比例：pt-2.5 pb-2（对齐首页顶栏紧凑通透语言，释放视口）；
	// - 桌面端舒展呼吸：纯标题留 pb-4.5，有 below 时留 pb-3.5。
	const headerPadding = $derived(
		integrated
			? 'pt-3 pb-1'
			: floating
				? 'pt-2 pb-2'
				: below
					? 'pt-2.5 sm:pt-4 pb-2 sm:pb-3.5'
					: 'pt-2.5 sm:pt-4.5 pb-2 sm:pb-4.5'
	);
</script>

<header
	use:headerScroll
	data-island-visible={islandEnabled && islandShown ? 'true' : undefined}
	data-island-band-hidden={islandBandHidden ? 'true' : undefined}
	aria-hidden={islandBandHidden ? 'true' : undefined}
	inert={islandBandHidden}
	class={cn(
		'shrink-0 transition-opacity duration-standard ease-standard',
		integrated
			? 'max-sm:sticky max-sm:top-0 max-sm:z-[var(--cst-z-sticky)] max-sm:bg-surface-page/85 max-sm:backdrop-blur-md max-sm:pb-1.5'
			: floating
				? 'sticky top-0 z-[var(--cst-z-sticky)] pointer-events-none'
				: [
						'page-header-scroll max-sm:bg-surface-page/85 max-sm:backdrop-blur-md max-sm:border-b-0',
						sticky && 'sticky top-0 z-[var(--cst-z-sticky)]',
						scrolled && 'page-header-scrolled',
						// 岛已接管：无 below 的吸顶页头整条隐去（保留占位、不产生版面跳动）
						islandBandHidden && 'pointer-events-none opacity-0'
					],
		className
	)}
>
	<ContentColumn {maxWidth} class={cn(headerPadding, contentClassName)}>
		{#if integrated}
			<!-- hub/列表页「内容内页头」：标题 + 副标题/计数 + 工具同一行；筛选紧随；桌面/平板无背景/边框/吸顶，移动端例外吸顶 -->
			<div data-page-intro bind:this={islandAnchorEl} class="flex flex-wrap items-center justify-between gap-x-3 gap-y-1.5">
				<div
					data-island-handoff={islandHandoffTrigger && islandShown ? 'true' : undefined}
					inert={islandHandoffTrigger && islandShown}
					class="flex min-w-0 flex-1 flex-wrap items-baseline gap-x-2.5 gap-y-1 transition-opacity duration-standard {islandHandoffTrigger && islandShown
						? 'opacity-0 pointer-events-none'
						: ''}"
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
					<h1 class="min-w-0 truncate text-lg font-semibold text-text-primary sm:text-xl">{title}</h1>
					{#if subtitle}
						<p class="flex min-w-0 items-center gap-1 text-xs text-text-secondary">
							{#if subtitleIcon}
								<span class="flex shrink-0 items-center">{@render subtitleIcon()}</span>
							{/if}
							<span class="truncate">{subtitle}</span>
						</p>
					{/if}
				</div>
				<!-- 入岛后若岛接管了行动区，页头 trailing 隐藏，避免同一动作出现两份 -->
				{#if trailing && !(islandTrailing && islandShown)}
					<div class="flex shrink-0 items-center gap-2">{@render trailing()}</div>
				{/if}
			</div>
			{#if below}
				<div class="mt-2.5">{@render below()}</div>
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
					<div
						inert={islandHandoffTrigger && islandShown}
						class="pointer-events-auto flex items-center rounded-full border border-border-default/60 bg-surface-primary/85 text-text-primary p-0.5 shadow-raised-sm backdrop-blur-md shrink-0 transition-opacity duration-standard {islandHandoffTrigger && islandShown
							? 'opacity-0 pointer-events-none'
							: ''}"
					>
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
				data-island-handoff={islandHandoffTrigger && islandShown ? 'true' : undefined}
				class="min-w-0 flex-1 transition-opacity duration-standard {islandHandoffTrigger && islandShown
					? 'opacity-0 pointer-events-none'
					: ''}"
			>
				<h1 class="text-text-primary font-semibold leading-none {TITLE_SIZE_CLASS[titleSize]}">{title}</h1>
				{#if subtitle}
					<p class="mt-1 flex min-w-0 items-center gap-1 text-text-secondary {SUBTITLE_SIZE_CLASS[subtitleSize]}">
						{#if subtitleIcon}
							<span class="flex shrink-0 items-center">{@render subtitleIcon()}</span>
						{/if}
						<span class="truncate">{subtitle}</span>
					</p>
				{/if}
			</div>
			<!-- 入岛后若岛接管了行动区，页头 trailing 隐藏，避免同一动作出现两份 -->
			{#if trailing && !(islandTrailing && islandShown)}
				<div class="shrink-0">{@render trailing()}</div>
			{/if}
		</div>
		{/if}
		{#if below && !floating && !integrated}
			{@render below()}
		{/if}
	</ContentColumn>
</header>
