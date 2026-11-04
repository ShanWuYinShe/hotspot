<script lang="ts">
	import { ArrowLeft } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import type { AuthorInfo } from '@/stores/header-island.svelte';

	// PageIsland — 桌面端顶栏中央「页面锚点岛」的唯一渲染实现。
	// 职责：把会随滚动离开视口、但随时可能需要的信息（标题/作者身份/状态）与页面主行动
	//（≤2 个控件）以「紧凑胶囊」形态钉在 52px 顶栏中央，胶囊底部显示阅读进度。
	//
	// 交互契约（唯一权威：docs/design-system.md §22）：
	//   - 显隐由页面注册方决定（PageHeader / DetailHeader / 首页信息流），本组件只管形态与交互；
	//   - 点击标题：回到页面锚点（吸顶页头回页首、非吸顶页头回到标题位置）；
	//     详情页身份卡：点头像/名字进入作者主页（onTitleClick 由调用方给出语义）；
	//   - 左侧返回：只有注册了 onBack 的二级页/详情页出现；
	//   - hover / 键盘 focus-within：胶囊横向舒展（max-w 过渡），展开被截断的标题与 meta；
	//   - 底部 2px 阅读进度条（accent）：随主滚动容器进度增长，纯展示、不参与交互；
	//   - 键盘顺序：返回 → 标题/身份 → 行动区；全部控件有 aria-label 与 focus-ring。

	let {
		title = '',
		meta = '',
		author,
		onBack,
		backLabel = '返回',
		onTitleClick,
		progress = 0,
		trailing
	}: {
		/** 页面标题（作者身份模式下由 author.name 覆盖） */
		title: string;
		/** 第二行上下文：计数 / 位置 / 结果数 / 当前信息流等（缺省回退作者位置·时间） */
		meta?: string;
		/** 详情页身份（作者/群/用户）；存在时左侧显示头像 */
		author?: AuthorInfo;
		/** 左侧返回；未注册则不渲染返回钮 */
		onBack?: () => void;
		backLabel?: string;
		/** 点击标题/身份的语义：回锚点或进作者主页（由调用方决定） */
		onTitleClick?: () => void;
		/** 阅读进度 0–100，画在胶囊底部 */
		progress?: number;
		/** 右侧行动区（页面主行动，≤2 控件） */
		trailing?: Snippet;
	} = $props();

	const displayTitle = $derived(author?.name || title || '详情');
	const metaText = $derived(
		meta || (author ? [author.location, author.time].filter(Boolean).join(' · ') : '')
	);
	const titleAria = $derived(
		author ? `查看 ${author.name} 的资料` : `回到「${title || '页面标题'}」`
	);
	const safeProgress = $derived(Math.min(100, Math.max(0, Number.isFinite(progress) ? progress : 0)));
</script>

<div
	data-desktop-dynamic-island
	role="group"
	aria-label="页面锚点与快捷操作"
	style="contain: layout paint;"
	class="group/island pointer-events-auto relative flex h-9 min-w-0 max-w-md items-center gap-1 rounded-full border border-border-default/50 bg-surface-primary/90 px-1.5 text-text-primary shadow-raised-sm backdrop-blur-lg transition-[max-width,box-shadow] duration-standard ease-standard hover:max-w-lg hover:shadow-raised-md focus-within:max-w-lg focus-within:shadow-raised-md"
>
	{#if onBack}
		<ActionIconButton
			icon={ArrowLeft}
			label={backLabel}
			onClick={onBack}
			variant="ghost"
			tone="muted"
			size={32}
			class="shrink-0"
		/>
		<div class="h-3.5 w-px shrink-0 bg-border-default/40" aria-hidden="true"></div>
	{/if}

	<!-- 中间：身份微名片 / 页面标题 + 一行 meta（点击回到锚点） -->
	<div class="flex min-w-0 flex-1 items-center justify-center overflow-hidden">
		<button
			type="button"
			class="hover-btn focus-ring flex min-w-0 max-w-full items-center gap-2 rounded-full px-2 py-0.5 text-left active:scale-[0.98]"
			onclick={() => onTitleClick?.()}
			aria-label={titleAria}
		>
			{#if author}
				<Avatar src={author.avatar} name={author.name} size={20} class="shrink-0" />
			{/if}
			<div class="flex min-w-0 flex-col justify-center">
				<span class="truncate text-xs font-semibold leading-tight text-text-primary">
					{displayTitle}
				</span>
				{#if metaText}
					<span class="truncate text-xxs leading-tight text-text-secondary">{metaText}</span>
				{/if}
			</div>
		</button>
	</div>

	<!-- hover/focus 舒展：揭示精确阅读进度（与「点击标题回锚点」互补，不重复同一功能） -->
	<span
		class="pointer-events-none ml-auto hidden max-w-0 shrink-0 items-center justify-end overflow-hidden text-xxs font-medium tabular-nums text-accent-default opacity-0 transition-[max-width,opacity] duration-standard ease-standard group-hover/island:max-w-12 group-hover/island:opacity-100 group-focus-within/island:max-w-12 group-focus-within/island:opacity-100 sm:flex"
		aria-hidden="true"
	>
		{Math.round(safeProgress)}%
	</span>

	{#if trailing}
		<div class="h-3.5 w-px shrink-0 bg-border-default/40" aria-hidden="true"></div>
		<div class="flex shrink-0 items-center gap-0.5">
			{@render trailing()}
		</div>
	{/if}

	<!-- 阅读进度环：20px 迷你圆环（accent 圆弧 0–100%），纯展示、不参与交互 -->
	<span
		data-island-progress-ring
		data-island-progress={safeProgress}
		class="pointer-events-none flex size-5 shrink-0 items-center justify-center"
		aria-hidden="true"
	>
		<svg viewBox="0 0 20 20" class="size-5 -rotate-90" fill="none">
			<circle cx="10" cy="10" r="7.5" stroke-width="2" class="stroke-border-default" />
			<circle
				cx="10"
				cy="10"
				r="7.5"
				stroke-width="2"
				stroke-linecap="round"
				class="stroke-accent-default transition-[stroke-dashoffset] duration-rapid ease-standard"
				stroke-dasharray="47.12"
				stroke-dashoffset={47.12 * (1 - safeProgress / 100)}
			/>
		</svg>
	</span>
</div>
