<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	// 数据展示组件（data-display）。
	// NearbyImageCard — 图片优先内容卡片统一原语（Image-first Card，Round 2 改版）。
	//   - 纵排（默认）：图片 object-cover 铺满整卡，信息以「顶部/底部遮罩 + 覆盖层」呈现
	//     （不再有底部实底托盘；遮罩色走 --cst-surface-card-scrim-* token，三主题×明暗）。
	//   - 无图 fallback：主题化渐变底 + 低透明度主题水印（opacity-watermark=0.08）+ 主题文字色，
	//     不压白字（footer 通过 snippet 参数 onImage=false 自行切换文字色）。
	//   - 横排（horizontal）：图左 40% + 右侧实底信息面板（登记例外：紧凑横排卡保留托盘）。
	//   - hover 走 card-hover 抬升语言；图片微缩放由 .card-hover:hover 规则驱动；
	//     整卡按压缩放由 --cst-card-tap-scale 驱动；行内按钮经根上 auto-lift(z-20) 可点。
	// 替代 PostCard / FeaturedGroupCard / GroupCard / UserCard 内联重复实现。

	let {
		imageSrc,
		imageAlt = '',
		onClick,
		onImageError,
		placeholder,
		watermark,
		footer,
		children,
		class: className = '',
		horizontal = false
	}: {
		imageSrc: string;
		imageAlt?: string;
		onClick?: () => void;
		onImageError?: () => void;
		/** 无图时的占位内容（渲染在 accent 渐变层上；缺省为空白渐变层） */
		placeholder?: Snippet;
		/** 无图时的低透明度水印装饰（accen 色、aria-hidden，建议 64–96px） */
		watermark?: Snippet;
		/** footer 内容：onImage=true 表示有图（纵排需白字）；horizontal=false 时才走压图白字 */
		footer?: Snippet<[{ onImage: boolean; horizontal: boolean }]>;
		children?: Snippet;
		class?: string;
		/** 横排变体：图片占左侧 40%,footer tray 变右侧内容面板 */
		horizontal?: boolean;
	} = $props();

	// 图片/遮罩/占位层的覆盖盒：纵排=整卡,横排=左列。
	// w/h 必须显式拉满:abs 定位的替换元素(img)在 width/height:auto 时按固有尺寸渲染
	// (CSS2 §10.3.7),inset 全拉锚不开——小图按原尺寸露底、大图按固有比例错裁。
	const imageBox = $derived(
		horizontal ? 'absolute left-0 top-0 h-full w-2/5' : 'absolute left-0 top-0 h-full w-full'
	);

	// hover 与内容卡统一走 card-hover；键盘焦点由 focus-ring 承载；按压缩放按平台 token。
	let pressed = $state(false);
	const cardTransform = $derived(pressed ? 'scale(var(--cst-card-tap-scale, 0.94))' : '');

	// 图片加载态：加载完成前 accent 渐变兜底、onload 渐显；失败与「无图」走同一 fallback。
	let imageLoaded = $state(false);
	let imageFailed = $state(false);
	const hasImage = $derived(Boolean(imageSrc) && !imageFailed);
	$effect(() => {
		void imageSrc;
		imageLoaded = false;
		imageFailed = false;
	});

	// 行内子按钮的按压由按钮自身承载，不带动整卡缩放（否则像误触整卡）。
	function pressedFromEvent(e: PointerEvent): boolean {
		const btn = (e.target as HTMLElement).closest('button');
		return !btn || btn.hasAttribute('data-cover');
	}
</script>

<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
<article
	class={cn(
		// isolate：卡片自成堆叠上下文——内部按钮 z-20 保持覆盖按钮可点；
		// Round 2：去掉外框 border（连续画布，卡片边界只由图片/圆角/投影表达）。
		'card-hover isolate group relative h-full w-full overflow-hidden',
		onClick &&
			'[&_button:not([data-cover])]:relative [&_button:not([data-cover])]:z-20 [&_a]:relative [&_a]:z-20',
		className
	)}
	style={cardTransform ? `transform:${cardTransform};transition:transform var(--cst-motion-instant) var(--cst-ease-standard);` : ''}
	onpointerdown={(e) => {
		if (pressedFromEvent(e)) pressed = true;
	}}
	onpointerup={() => (pressed = false)}
	onpointerleave={() => (pressed = false)}
>
	{#if hasImage}
		{#if !imageLoaded}
			<div class="surface-accent-gradient pointer-events-none {imageBox} animate-pulse"></div>
		{/if}
		<img
			src={imageSrc}
			alt={imageAlt}
			loading="lazy"
			class="image-hover-frame__image {imageBox} object-cover will-change-transform transition-opacity duration-standard {imageLoaded ? 'opacity-100' : 'opacity-0'}"
			onload={() => (imageLoaded = true)}
			onerror={() => {
				imageFailed = true;
				onImageError?.();
			}}
		/>
		{#if !horizontal}
			<!-- 纵排：顶部/底部双遮罩承载覆盖信息（三主题 token） -->
			<!-- 顶部遮罩 128px（BLOCKER 修复，与 HomeFeedItem 统一）：96px 时两行作者区
			     （16–52px）落在 0.64→transparent 陡降段，最坏白底不足 4.5；128px 后中段 alpha≈0.63。 -->
			<div class="pointer-events-none absolute inset-x-0 top-0 h-32" style="background:var(--cst-surface-card-scrim-top);"></div>
			<div class="pointer-events-none absolute inset-x-0 bottom-0 h-[62%]" style="background:var(--cst-surface-card-scrim-bottom);"></div>
		{:else}
			<div class="pointer-events-none {imageBox}" style="background:var(--cst-surface-media-scrim-card);"></div>
		{/if}
	{:else}
		<!-- 无图 fallback（登记例外：不压白字）：主题渐变底 + 低透明主题水印 + 主题色文字 -->
		<div class="surface-accent-gradient pointer-events-none {imageBox}">
			{#if watermark}
				<div
					class="absolute inset-0 flex items-center justify-center text-accent-default opacity-watermark"
					aria-hidden="true"
				>
					{@render watermark()}
				</div>
			{/if}
			{#if placeholder}
				<div class="absolute inset-0 flex items-center justify-center">{@render placeholder()}</div>
			{/if}
		</div>
	{/if}
	{#if children}
		{@render children()}
	{/if}
	{#if footer}
		{#if horizontal}
			<div
				class="absolute inset-y-0 right-0 left-2/5 flex flex-col justify-center border-l border-border-card-menu bg-surface-card-menu"
			>
				{@render footer({ onImage: hasImage, horizontal })}
			</div>
		{:else}
			<div class="absolute inset-x-0 bottom-0">
				{@render footer({ onImage: hasImage, horizontal })}
			</div>
		{/if}
	{/if}
	{#if onClick}
		<!-- 卡片点击覆盖层（最后绘制，z-10）：真实 button 承载点击/键盘/aria，
		     行内操作按钮经根上 auto-lift(z-20) 保持可点；根不再持 button 语义 -->
		<button
			type="button"
			class="absolute inset-0 z-10 cursor-pointer focus-ring"
			data-cover
			aria-label={imageAlt || undefined}
			onclick={onClick}
		></button>
	{/if}
</article>
