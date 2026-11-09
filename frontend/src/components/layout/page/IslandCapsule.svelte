<script lang="ts">
	import type { Snippet } from 'svelte';
	import { slide, fade } from 'svelte/transition';
	import { cubicOut } from 'svelte/easing';
	import { ArrowUp, ChevronDown, X } from '@lucide/svelte';
	import { motionDuration } from '@/lib/motion';

	// IslandCapsule — 桌面端顶栏中央「灵动岛」的唯一形态实现（docs/design-system.md §22）。
	// 极致美学与艺术级交互规范：
	//   - 纯净常驻胶囊态：rounded-full 连续胶囊、严格 34px（h-8.5）黄金比例，居于 52px 顶栏正中央；
	//   - 整体响应统一交互：整颗胶囊为一个连贯的交互微岛，彻底消除内部碎片化 hover 灰框；
	//   - 展开中枢态：点击单向向下舒展为 440px 极致毛玻璃控制中枢，四角圆润优雅（rounded-2xl）；
	//   - 视界通透呼吸：统一标准化卡片头部、功能区与静音底栏，杜绝重叠、多余按钮与空间遮挡感；
	//   - 渐进进度指示：最右侧极细 16px 矢量圆环，与滚动平滑联动。

	let {
		ariaLabel = '页面锚点与快捷操作',
		title,
		meta = '',
		titleAria,
		onTitleClick,
		leading,
		center,
		trailing,
		expanded,
		progress = 0,
		class: className = ''
	}: {
		/** 岛的整体语义（role=group 的 aria-label）。 */
		ariaLabel?: string;
		/** 主标题（页面名 / 作者名 / 当前信息流）。 */
		title: string;
		/** 第二行上下文（计数 / 位置 / 状态）。 */
		meta?: string;
		/** 标题按钮的可访问名（含动作语义，如「回到「附近的人」」）。 */
		titleAria?: string;
		/** 点击标题的语义（回锚点 / 进作者主页），由调用方决定。 */
		onTitleClick?: () => void;
		/** 左侧区（返回按钮 / 作者头像 / 信息流图标），可选。 */
		leading?: Snippet;
		/** 中间自定义交互区（如搜索输入框等，优先于默认标题文本）。 */
		center?: Snippet;
		/** 右侧行动区（≤2 个控件），可选。 */
		trailing?: Snippet;
		/** 展开态控制中枢内容（分类切换/筛选器/作者互动等） */
		expanded?: Snippet;
		/** 阅读进度 0–100（唯一滚动 owner 上报），渲染为最右侧圆环。 */
		progress?: number;
		class?: string;
	} = $props();

	let islandEl = $state<HTMLElement | undefined>(undefined);
	let isExpanded = $state(false);

	const hasCenter = $derived(Boolean(center));

	const safeProgress = $derived(
		Math.min(100, Math.max(0, Number.isFinite(progress) ? progress : 0))
	);

	// 进度圆环几何：18px / 2px 描边；尺寸用 rem（1.125rem）随大屏根字号等比放大。
	const RING_SIZE = 18;
	const RING_STROKE = 2;
	const RING_RADIUS = (RING_SIZE - RING_STROKE) / 2 - 1; // 7
	const RING_CIRCUMFERENCE = 2 * Math.PI * RING_RADIUS;
	const ringOffset = $derived(RING_CIRCUMFERENCE * (1 - safeProgress / 100));

	function handleTitleClick(e: MouseEvent) {
		e.stopPropagation();
		onTitleClick?.();
		if (expanded) {
			isExpanded = !isExpanded;
		}
	}

	function toggleExpand(e: MouseEvent) {
		e.stopPropagation();
		if (expanded) {
			isExpanded = !isExpanded;
		}
	}

	function handleScrollToTop() {
		const scroller = document.getElementById('main-content');
		scroller?.scrollTo({ top: 0, behavior: 'smooth' });
	}

	// 全局点击任意空白处收起灵动岛
	function handleWindowPointerDown(e: PointerEvent) {
		if (!isExpanded) return;
		const target = e.target as Node | null;
		if (islandEl && target && !islandEl.contains(target)) {
			isExpanded = false;
		}
	}

	// 标题或路径上下文改变时自动收起
	$effect(() => {
		title;
		isExpanded = false;
	});

	const transitionDuration =
		typeof process !== 'undefined' && process.env?.NODE_ENV === 'test' ? 0 : motionDuration(220);
</script>

<svelte:window
	onpointerdown={handleWindowPointerDown}
	onkeydown={(e) => {
		if (e.key === 'Escape' && isExpanded) {
			isExpanded = false;
		}
	}}
/>

<div
	bind:this={islandEl}
	data-desktop-dynamic-island
	role="group"
	aria-label={ariaLabel}
	class="group/island pointer-events-auto relative z-50 flex flex-col transition-all duration-relaxed ease-[cubic-bezier(0.16,1,0.3,1)] {isExpanded
		? 'w-[440px] sm:w-[460px] max-w-[94vw] rounded-2xl border border-accent-border/40 bg-surface-primary/95 shadow-modal backdrop-blur-2xl dark:border-white/12 dark:bg-surface-primary/90'
		: `${hasCenter ? 'w-[360px] sm:w-[400px]' : 'min-w-[260px] sm:min-w-[290px] max-w-[380px]'} rounded-full border border-border-default/45 bg-surface-primary/75 text-text-primary shadow-xs backdrop-blur-xl hover:border-accent-border/50 hover:bg-surface-primary/95 hover:shadow-raised-sm hover:scale-[1.01] active:scale-[0.99] dark:border-white/10 dark:bg-surface-primary/65`} {className}"
>
	{#if !isExpanded}
		<!-- ══ 1. 极致精巧常驻胶囊态（高度严格 34px，rounded-full 整体交互） ══ -->
		<div class="flex h-8.5 w-full items-center justify-between pl-2 pr-2.5 select-none shrink-0">
			<!-- 左侧微徽章与标题区域 -->
			<div class="flex h-full min-w-0 flex-1 items-center gap-1.5">
				{#if leading}
					<span class="flex shrink-0 items-center transition-transform duration-snappy group-hover/island:scale-105">
						{@render leading()}
					</span>
				{/if}

				{#if center}
					<div class="flex h-full min-w-0 flex-1 items-center">
						{@render center()}
					</div>
				{:else}
					<!-- 标题与 Meta：两行优雅收放，居中呼吸 -->
					<button
						type="button"
						class="flex h-full min-w-0 flex-1 items-center text-left focus:outline-none"
						onclick={handleTitleClick}
						aria-label={titleAria ?? (expanded ? `展开「${title}」控制中枢` : `回到「${title}」`)}
					>
						<span class="flex min-w-0 flex-col justify-center py-0.5">
							<span class="truncate text-xs font-semibold leading-[1.15] text-text-primary tracking-tight">{title}</span>
							{#if meta}
								{#key meta}
									<span class="animate-fade-in truncate text-[11px] font-normal leading-[1.15] text-text-secondary/80 mt-0.5">{meta}</span>
								{/key}
							{/if}
						</span>
					</button>
				{/if}
			</div>

			<!-- 右侧行动与展开指示区 -->
			<div class="flex shrink-0 items-center gap-1.5 pl-1.5">
				{#if trailing}
					<div class="flex shrink-0 items-center gap-1">
						{@render trailing()}
					</div>
				{/if}

				{#if expanded}
					<button
						type="button"
						class="flex size-5.5 shrink-0 items-center justify-center rounded-full text-text-secondary/50 transition-all duration-snappy group-hover/island:text-text-primary hover:bg-surface-subtle"
						onclick={toggleExpand}
						aria-label="展开控制面板"
						title="展开控制面板"
					>
						<ChevronDown class="size-3.5 transition-transform duration-snappy group-hover/island:translate-y-0.5" />
					</button>
				{/if}

				<span
					class="flex shrink-0 items-center"
					data-island-progress-ring
					data-island-progress={safeProgress}
					aria-hidden="true"
				>
					<svg
						class="block -rotate-90"
						width="1.125rem"
						height="1.125rem"
						viewBox="0 0 {RING_SIZE} {RING_SIZE}"
					>
						<circle
							cx={RING_SIZE / 2}
							cy={RING_SIZE / 2}
							r={RING_RADIUS}
							fill="none"
							stroke="var(--color-border-default)"
							stroke-opacity="0.28"
							stroke-width={RING_STROKE}
						/>
						<circle
							cx={RING_SIZE / 2}
							cy={RING_SIZE / 2}
							r={RING_RADIUS}
							fill="none"
							stroke="var(--color-accent-default)"
							stroke-width={RING_STROKE}
							stroke-linecap="round"
							stroke-dasharray={RING_CIRCUMFERENCE}
							stroke-dashoffset={ringOffset}
						/>
					</svg>
				</span>
			</div>
		</div>
	{:else}
		<!-- ══ 2. 极致艺术品级展开态（独立 Command Palette 形态，呼吸感充沛） ══ -->
		<div
			transition:slide={{ duration: transitionDuration, easing: cubicOut }}
			class="w-full overflow-hidden"
		>
			<!-- 头部控制栏：宽敞、优雅、无缝对齐 -->
			<div class="flex h-11 w-full items-center justify-between border-b border-border-default/15 px-3.5 select-none shrink-0 bg-surface-subtle/25">
				<div class="flex min-w-0 flex-1 items-center gap-2">
					{#if leading}
						<span class="flex shrink-0 items-center">{@render leading()}</span>
					{/if}
					<div class="flex min-w-0 flex-col justify-center">
						<span class="truncate text-xs font-semibold text-text-primary tracking-tight">{title}</span>
						{#if meta}
							<span class="truncate text-[11px] text-text-secondary">{meta}</span>
						{/if}
					</div>
				</div>

				<!-- 极简毛玻璃微圆关闭钮 -->
				<button
					type="button"
					class="flex size-6 shrink-0 items-center justify-center rounded-full border border-border-default/20 bg-surface-subtle/60 text-text-secondary/70 transition-all hover:bg-interactive-hover hover:text-text-primary active:scale-90"
					onclick={toggleExpand}
					aria-label="收起控制面板"
					title="收起 (Esc)"
				>
					<X class="size-3.5" />
				</button>
			</div>

			<!-- 主体内容承载区：告别局促与遮挡 -->
			{#if expanded}
				<div class="w-full px-4 py-3.5 animate-fade-in">
					{@render expanded()}
				</div>
			{/if}

			<!-- 统一标准底部控制条：Esc 与回顶从容收纳 -->
			<div class="flex items-center justify-between border-t border-border-default/15 px-4 py-2 text-xxs text-text-secondary bg-surface-subtle/30 rounded-b-2xl">
				<div class="flex items-center gap-1.5 text-text-secondary/70">
					<kbd class="rounded border border-border-default/30 bg-surface-island px-1.5 py-0.5 font-mono text-[10px] shadow-xs">Esc</kbd>
					<span>收起</span>
				</div>
				<button
					type="button"
					class="group flex items-center gap-1 text-accent-default hover:underline transition-colors font-medium text-xxs"
					onclick={handleScrollToTop}
				>
					<span>平滑回顶</span>
					<ArrowUp class="size-3 transition-transform group-hover:-translate-y-0.5" />
				</button>
			</div>
		</div>
	{/if}
</div>
