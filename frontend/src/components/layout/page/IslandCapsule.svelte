<script lang="ts">
	import type { Snippet } from 'svelte';
	import { islandHandoff } from '@/lib/island-handoff';

	// IslandCapsule — 桌面端顶栏中央「灵动岛」的唯一形态实现（docs/design-system.md §22）。
	//
	// 为什么要有这一层：PageIsland（页面锚点岛）与 HomeFeedIsland（首页信息流岛）此前各自
	// 复制了一份外壳标记，改形态要改两处、且两份漂移出「百分比与进度环同值双表达」这种重复 UI。
	// 外壳/呼吸态/阅读进度/交互动效现在只有这一份，两个业务岛只提供内容与行为。
	//
	// 交互契约（与 §22 一致）：
	//   - hover / focus-within：表面由半透明玻璃变为实底 + accent 细边 + 抬升档投影，
	//     明确「这颗胶囊可点」，但**不改变宽度**——hover 横向舒展会让居中的胶囊左右同时外扩、
	//     内容整排位移，既抖又廉价；
	//   - 同一读数只出现一处：阅读进度只用最右侧圆环表达（accent 弧 + 极淡底轨，每帧跟手，
	//     不加 CSS 过渡），不再在 hover 里重复百分比，也不再画底部下划线；
	//   - 点击标题回页面锚点（语义由调用方 onTitleClick 决定）；
	//   - 进场/退场由滚动交接逐帧驱动（lib/island-handoff.ts）：滚动停在哪交接就停在哪、
	//     反向完全可逆；没有独立时长动画，路由切换时直接卸载，不会在新页面上残留动画。
	//   - 键盘：标题按钮有 aria-label 与 focus-ring，阻止冒泡的圆形按钮各自独立。
	let {
		ariaLabel = '页面锚点与快捷操作',
		title,
		meta = '',
		titleAria,
		onTitleClick,
		leading,
		trailing,
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
		/** 阅读进度 0–100（唯一滚动 owner 上报），渲染为最右侧圆环。 */
		progress?: number;
		class?: string;
	} = $props();

	const safeProgress = $derived(
		Math.min(100, Math.max(0, Number.isFinite(progress) ? progress : 0))
	);

	// 进度圆环几何：18px / 2px 描边；尺寸用 rem（1.125rem）随大屏根字号等比放大。
	const RING_SIZE = 18;
	const RING_STROKE = 2;
	const RING_RADIUS = (RING_SIZE - RING_STROKE) / 2 - 1; // 7
	const RING_CIRCUMFERENCE = 2 * Math.PI * RING_RADIUS;
	const ringOffset = $derived(RING_CIRCUMFERENCE * (1 - safeProgress / 100));
</script>

<div
	data-desktop-dynamic-island
	role="group"
	aria-label={ariaLabel}
	style="contain: layout paint;"
	use:islandHandoff
	class="group/island pointer-events-auto relative flex h-9 min-w-0 max-w-md items-center rounded-full border border-border-default/35 bg-surface-primary/75 backdrop-blur-2xl px-2.5 text-text-primary shadow-raised-sm transition-[background-color,border-color,box-shadow,transform] duration-standard ease-standard hover:-translate-y-px hover:border-accent-border/60 hover:bg-surface-primary/90 hover:shadow-raised-md focus-within:border-accent-border/60 focus-within:bg-surface-primary/90 focus-within:shadow-raised-md {className}"
>
	<!-- 内容层 -->
	<div class="flex h-full min-w-0 items-center gap-1.5">
		{#if leading}
			<span class="flex shrink-0 items-center">{@render leading()}</span>
		{/if}

		<!-- 中间：标题 + 一行 meta（点击回锚点；hover 由 hover-btn 给整块淡底，明确可点区域） -->
		<button
			type="button"
			class="hover-btn focus-ring flex min-w-0 items-center rounded-full px-1.5 py-0.5 text-left active:scale-[0.98]"
			onclick={() => onTitleClick?.()}
			aria-label={titleAria ?? title}
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

		{#if trailing}
			<div class="h-3.5 w-px shrink-0 bg-border-default/40" aria-hidden="true"></div>
			<div class="flex shrink-0 items-center gap-0.5">
				{@render trailing()}
			</div>
		{/if}

		<!-- 阅读进度圆环（唯一读数表达，固定在最右侧）：
		     accent 弧 + 极淡底轨，0% 时是一圈空环、100% 收成完整圆；
		     每帧由滚动 owner 直接写 dashoffset（不加 CSS 过渡），保证与滚动严格同频。 -->
		<span
			class="ml-1 flex shrink-0 items-center"
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