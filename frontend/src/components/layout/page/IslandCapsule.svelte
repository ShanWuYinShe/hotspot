<script lang="ts">
	import type { Snippet } from 'svelte';
	import { ChevronDown } from '@lucide/svelte';

	// IslandCapsule — 桌面端顶栏中央「灵动岛」的唯一形态实现（docs/design-system.md §22）。
	// 现升级为「默认常驻 + 交互中枢」模型：
	//   - 紧凑常驻态：严格 32px（h-8）极薄胶囊，居于 52px 顶栏正中央，无缝悬浮，不遮挡正文；
	//   - 紧凑两侧贴合：左侧图表/头像与右侧圆环紧密贴合胶囊两端圆弧，无多余白边；
	//   - 展开中枢态：点击展开为单向向下舒展的 400px 毛玻璃控制卡片，承载分类筛选、作者名片与操作；
	//   - 干净四角与全域收起：去除破坏层叠上下文的内嵌 fixed 遮罩，全局 pointerdown 侦测点击任意外部立即收回；
	//   - 阅读进度：最右侧集成 18px 极细圆环，随视口滚动严格同步。
	let {
		ariaLabel = '页面锚点与快捷操作',
		title,
		meta = '',
		titleAria,
		onTitleClick,
		leading,
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

	// 全局点击任意空白处收起灵动岛（解决 nested fixed 遮罩被父级 transform 限制并导致四个角发灰的缺陷）
	function handleWindowPointerDown(e: PointerEvent) {
		if (!isExpanded) return;
		const target = e.target as Node | null;
		if (islandEl && target && !islandEl.contains(target)) {
			isExpanded = false;
		}
	}
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
	class="group/island pointer-events-auto relative z-50 flex transition-[all] duration-standard ease-[cubic-bezier(0.16,1,0.3,1)] {isExpanded
		? 'w-[400px] min-w-[360px] max-w-[460px] flex-col items-stretch rounded-lg border border-accent-border/50 bg-surface-primary/95 px-3.5 pb-3 pt-1 shadow-modal backdrop-blur-2xl'
		: 'h-8 min-w-0 max-w-md items-center rounded-full border border-border-default/35 bg-surface-primary/80 pl-1 pr-1 text-text-primary shadow-raised-sm backdrop-blur-2xl hover:-translate-y-px hover:border-accent-border/60 hover:bg-surface-primary/95 hover:shadow-raised-md focus-within:border-accent-border/60 focus-within:bg-surface-primary/95 focus-within:shadow-raised-md'} {className}"
>
	<!-- 顶行紧凑条（常驻可见；展开时为头部） -->
	<div class="flex h-8 w-full min-w-0 items-center justify-between gap-1 select-none">
		<div class="flex h-full min-w-0 flex-1 items-center gap-1.5">
			{#if leading}
				<span class="flex shrink-0 items-center">{@render leading()}</span>
			{/if}

			<!-- 中间：标题 + 一行 meta -->
			<button
				type="button"
				class="hover-btn focus-ring flex min-w-0 flex-1 items-center rounded-full px-1 py-0.5 text-left active:scale-[0.98]"
				onclick={handleTitleClick}
				aria-label={titleAria ?? `回到「${title}」`}
			>
				<span class="flex min-w-0 flex-col justify-center pb-0.5">
					<span class="truncate text-xs font-semibold leading-tight text-text-primary">{title}</span>
					{#if meta}
						{#key meta}
							<span class="animate-fade-in truncate text-xxs leading-tight text-text-secondary">{meta}</span>
						{/key}
					{/if}
				</span>
			</button>
		</div>

		<div class="flex shrink-0 items-center gap-0.5">
			{#if expanded}
				<!-- 展开/收起引导箭头 -->
				<button
					type="button"
					class="flex size-5 shrink-0 items-center justify-center rounded-full text-text-secondary/70 hover:text-text-primary hover:bg-interactive-hover transition-transform duration-standard"
					onclick={toggleExpand}
					aria-label={isExpanded ? '收起控制面板' : '展开控制面板'}
				>
					<ChevronDown class="size-3 transition-transform duration-standard {isExpanded ? 'rotate-180 text-accent-default' : ''}" />
				</button>
			{/if}

			{#if trailing && !isExpanded}
				<div class="h-3.5 w-px shrink-0 bg-border-default/40 mx-0.5" aria-hidden="true"></div>
				<div class="flex shrink-0 items-center gap-0.5">
					{@render trailing()}
				</div>
			{/if}

			<!-- 阅读进度圆环（最右侧，紧贴右边缘） -->
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

	<!-- 展开态控制面板主体 -->
	{#if isExpanded && expanded}
		<div class="h-px w-full bg-border-default/20 my-1.5" aria-hidden="true"></div>
		<div class="w-full animate-fade-in pt-1 pb-0.5">
			{@render expanded()}
		</div>
	{/if}
</div>
