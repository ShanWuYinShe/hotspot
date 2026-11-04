<script lang="ts">
	import type { Snippet } from 'svelte';
	import { islandMorph, type IslandRect } from '@/lib/island-morph';

	// IslandCapsule — 桌面端顶栏中央「灵动岛」的唯一形态实现（docs/design-system.md §22）。
	//
	// 为什么要有这一层：PageIsland（页面锚点岛）与 HomeFeedIsland（首页信息流岛）此前各自
	// 复制了一份「胶囊外壳 + 进度环 + hover 百分比」的标记，改形态要改两处、且两份漂移出
	// 「百分比与进度环同值双表达」这种重复 UI。外壳/呼吸态/进度环/交互动效现在只有这一份，
	// 两个业务岛只提供内容与行为。
	//
	// 交互契约（与 §22 一致）：
	//   - hover / focus-within：表面由半透明玻璃变为实底 + accent 细边 + 抬升档投影，
	//     明确「这颗胶囊可点」，但**不改变宽度**——旧实现 hover 横向舒展会让居中的胶囊
	//     左右同时外扩、内容整排位移，既抖又廉价；
	//   - 同一读数只出现一处：阅读进度只有右侧进度环表达，不再在 hover 里重复百分比；
	//   - 点击标题回页面锚点（语义由调用方 onTitleClick 决定）；
	//   - 进场/退场走 islandMorph：从调用方给出的锚点矩形收进/吐出顶栏（信息交接的连续感）；
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
		getMorphSource,
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
		/** 阅读进度 0–100（唯一滚动 owner 上报），纯展示。 */
		progress?: number;
		/** 进场/退场动画的来源矩形 getter（锚点最后可见位置）；无则退化为淡入。 */
		getMorphSource?: () => IslandRect | null;
		class?: string;
	} = $props();

	const safeProgress = $derived(
		Math.min(100, Math.max(0, Number.isFinite(progress) ? progress : 0))
	);
</script>

<div
	data-desktop-dynamic-island
	role="group"
	aria-label={ariaLabel}
	style="contain: layout paint;"
	transition:islandMorph={{ getSource: getMorphSource }}
	class="group/island pointer-events-auto relative flex h-9 min-w-0 max-w-md items-center gap-1.5 rounded-full border border-border-default/50 bg-surface-primary/90 px-2 text-text-primary shadow-raised-sm backdrop-blur-lg transition-[background-color,border-color,box-shadow,transform] duration-standard ease-standard hover:-translate-y-px hover:border-accent-border hover:bg-surface-primary hover:shadow-raised-md focus-within:border-accent-border focus-within:bg-surface-primary focus-within:shadow-raised-md {className}"
>
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
				<span class="truncate text-xxs leading-tight text-text-secondary">{meta}</span>
			{/if}
		</span>
	</button>

	{#if trailing}
		<div class="h-3.5 w-px shrink-0 bg-border-default/40" aria-hidden="true"></div>
		<div class="flex shrink-0 items-center gap-0.5">
			{@render trailing()}
		</div>
	{/if}

	<!-- 阅读进度：胶囊内底部 2px accent 细线（0–100%），纯展示、不参与交互。
	     同一读数只保留这一处表达——hover 里不再重复百分比；也不用圆环：
	     20px 小圆环在顶栏里更像「加载中」而不是「读到哪了」，细线不占右侧槽位、
	     不改变胶囊宽高。不画完整底轨：满宽灰线会被读成文字下划线，只留 accent 段
	     才是「读到哪了」的进度语义。 -->
	<span
		data-island-progress-bar
		data-island-progress={safeProgress}
		class="pointer-events-none absolute inset-x-2.5 bottom-[2px] h-px overflow-hidden rounded-full"
		aria-hidden="true"
	>
		<span
			class="absolute inset-y-0 left-0 rounded-full bg-accent-default transition-[width] duration-rapid ease-standard"
			style="width:{safeProgress}%"
		></span>
	</span>
</div>
