<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';
	import { presetImageUrl, presetSrcSet } from '@/lib/images';
	import { cardPress } from '@/lib/motion';

	// ═══ MediaCard ═══
	// 全站通用的图片/视频/封面优先卡片设计系统原语（Image-first / Media Card）。
	// 特性：
	//   1. 纵排（默认）：图片 object-cover 铺满整卡，信息以渐变遮罩 + 覆盖层呈现；
	//   2. 横排（horizontal）：左侧 40% 图片，右侧实底信息面板；
	//   3. 零成本降级：无图时优雅渲染 accent 渐变底 + 低透明度品牌/分类水印；
	//   4. 交互层级：整卡点击走 data-cover 按钮，内部操作按钮自动抬升(z-20)可点；
	//   5. 性能引擎：启用 content-visibility 离屏计算跳过，纯 GPU Compositor 变换。

	let {
		imageSrc,
		imageAlt = '',
		priority = false,
		sizes = '(min-width: 1024px) 25vw, (min-width: 640px) 40vw, 50vw',
		onClick,
		onImageError,
		placeholder,
		watermark,
		footer,
		children,
		class: className = '',
		horizontal = false,
		virtualize = true
	}: {
		imageSrc: string;
		imageAlt?: string;
		/** 首屏图：eager + fetchpriority=high */
		priority?: boolean;
		/** 响应式 sizes */
		sizes?: string;
		onClick?: () => void;
		onImageError?: () => void;
		/** 无图时的占位内容 */
		placeholder?: Snippet;
		/** 无图时的低透明度水印装饰 */
		watermark?: Snippet;
		/** footer 内容：onImage=true 表示有图；horizontal=false 时才走压图白字 */
		footer?: Snippet<[{ onImage: boolean; horizontal: boolean }]>;
		children?: Snippet;
		class?: string;
		/** 横排变体：图片占左侧 40%，footer 变右侧内容面板 */
		horizontal?: boolean;
		/** 是否启用浏览器原生 content-visibility 离屏渲染跳过（默认 true，极大减少长列表渲染负担） */
		virtualize?: boolean;
	} = $props();

	const imageBox = $derived(
		horizontal ? 'absolute left-0 top-0 h-full w-2/5' : 'absolute left-0 top-0 h-full w-full'
	);

	let imageLoaded = $state(false);
	let imageFailed = $state(false);
	let imgEl = $state<HTMLImageElement | null>(null);
	const hasImage = $derived(Boolean(imageSrc) && !imageFailed);

	// 缓存命中时图片可能在 onload 绑定前就已 complete（profile 页二次进入实测
	// 图片 opacity 恒 0、只剩渐变底）。挂载后主动检查 complete/naturalWidth，
	// 与 onload 共同保证首帧可见。
	$effect(() => {
		void imageSrc;
		imageFailed = false;
		imageLoaded = Boolean(imgEl?.complete && imgEl.naturalWidth > 0);
	});
</script>

<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<article
	class={cn(
		'card-hover isolate group relative h-full w-full overflow-hidden',
		'[&_button:not([data-cover])]:pointer-events-auto [&_a]:pointer-events-auto',
		onClick &&
			'[&_button:not([data-cover])]:relative [&_button:not([data-cover])]:z-20 [&_a]:relative [&_a]:z-20',
		className
	)}
	style={virtualize ? 'content-visibility: auto; contain-intrinsic-size: auto 300px;' : undefined}
	use:cardPress
	onclick={(e) => {
		if (!onClick) return;
		const target = e.target as HTMLElement | null;
		if (target?.closest('button:not([data-cover]), a[href], input, textarea, select, [role="menuitem"]')) return;
		if (target?.closest('[data-cover]')) return;
		onClick();
	}}
>
	{#if hasImage}
		{#if !imageLoaded}
			<div class="surface-accent-gradient media-placeholder--pulse pointer-events-none {imageBox}" aria-hidden="true"></div>
		{/if}
		<img
			bind:this={imgEl}
			src={presetImageUrl(imageSrc, 'card', priority ? 900 : 800)}
			srcset={presetSrcSet(imageSrc, 'card')}
			{sizes}
			alt={imageAlt}
			loading={priority ? 'eager' : 'lazy'}
			fetchpriority={priority ? 'high' : 'auto'}
			decoding="async"
			class="pointer-events-none select-none image-hover-frame__image {imageBox} object-cover transition-opacity duration-standard {imageLoaded ? 'opacity-100' : 'opacity-0'}"
			onload={() => (imageLoaded = true)}
			onerror={() => {
				imageFailed = true;
				onImageError?.();
			}}
		/>
		{#if imageLoaded}
			{#if !horizontal}
				<!-- 顶部不再铺全宽遮罩：角标/徽章（media-badge）自带深色强底，顶部大遮罩
				     会压掉图片上半区。底部只保留承载标题/作者/操作的局部遮罩（36%），
				     压图白字对比度仍由主题 scrim-bottom token 保证。 -->
				<div class="pointer-events-none absolute inset-x-0 bottom-0 h-[36%]" style="background:var(--cst-surface-card-scrim-bottom);"></div>
			{:else}
				<div class="pointer-events-none {imageBox}" style="background:var(--cst-surface-media-scrim-card);"></div>
			{/if}
		{/if}
	{:else}
		<div class="surface-accent-gradient pointer-events-none {imageBox} flex flex-col items-center justify-center bg-surface-primary border border-border-default/60">
			{#if watermark}
				<div
					class="absolute inset-0 flex items-center justify-center text-text-secondary/30 opacity-watermark"
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
		<div class="pointer-events-none" data-card-children>
			{@render children()}
		</div>
	{/if}
	{#if footer}
		{#if horizontal}
			<div
				class="pointer-events-none absolute inset-y-0 right-0 left-2/5 flex flex-col justify-center border-l border-border-card-menu bg-surface-card-menu"
				data-card-footer
			>
				{@render footer({ onImage: hasImage, horizontal })}
			</div>
		{:else}
			<div class="pointer-events-none absolute inset-x-0 bottom-0" data-card-footer>
				{@render footer({ onImage: hasImage, horizontal })}
			</div>
		{/if}
	{/if}
	{#if onClick}
		<button
			type="button"
			class="absolute inset-0 z-10 cursor-pointer focus-ring"
			style="transform: none !important;"
			data-cover
			aria-label={imageAlt || undefined}
			onclick={onClick}
		></button>
	{/if}
</article>
