<script lang="ts">
	import { ArrowLeft } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import IslandCapsule from './IslandCapsule.svelte';
	import type { IslandRect } from '@/lib/island-morph';
	import type { AuthorInfo } from '@/stores/header-island.svelte';

	// PageIsland — 桌面端顶栏中央「页面锚点岛」的业务层（形态见 IslandCapsule）。
	// 职责：把会随滚动离开视口、但随时可能需要的信息（标题/作者身份/状态）与页面主行动
	//（≤2 个控件）以「紧凑胶囊」形态钉在 52px 顶栏中央，胶囊右侧显示阅读进度。
	//
	// 交互契约（唯一权威：docs/design-system.md §22）：
	//   - 显隐由页面注册方决定（PageHeader / DetailHeader / 首页信息流），本组件只管形态与交互；
	//   - 点击标题：回到页面锚点（吸顶页头回页首、非吸顶页头回到标题位置）；
	//     详情页身份卡：点头像/名字进入作者主页（onTitleClick 由调用方给出语义）；
	//   - 左侧返回：只有注册了 onBack 的二级页/详情页出现；
	//   - hover / 键盘 focus-within：胶囊表面转实底 + accent 细边 + 抬升投影（不改变宽度）；
	//   - 右侧进度环：随主滚动容器进度增长，纯展示、不参与交互；
	//   - 键盘顺序：返回 → 标题/身份 → 行动区；全部控件有 aria-label 与 focus-ring；
	//   - 进场/退场：从页面锚点矩形收进/吐出顶栏（islandMorph），不再凭空闪现。

	let {
		title = '',
		meta = '',
		author,
		onBack,
		backLabel = '返回',
		onTitleClick,
		progress = 0,
		getMorphSource,
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
		/** 阅读进度 0–100，画在胶囊右侧 */
		progress?: number;
		/** 进场/退场动画的来源矩形 getter（页面锚点最后可见位置） */
		getMorphSource?: () => IslandRect | null;
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
</script>

<IslandCapsule
	ariaLabel="页面锚点与快捷操作"
	title={displayTitle}
	meta={metaText}
	titleAria={titleAria}
	onTitleClick={() => onTitleClick?.()}
	{progress}
	{getMorphSource}
	{trailing}
>
	{#snippet leading()}
		{#if onBack}
			<ActionIconButton
				icon={ArrowLeft}
				label={backLabel}
				onClick={onBack}
				variant="ghost"
				tone="muted"
				size={32}
			/>
			<div class="mx-1 h-3.5 w-px shrink-0 bg-border-default/40" aria-hidden="true"></div>
		{:else if author}
			<Avatar src={author.avatar} name={author.name} size={20} class="shrink-0" />
		{/if}
	{/snippet}
</IslandCapsule>
