<script lang="ts">
	import { ArrowLeft } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import Button from '@/components/ui/button.svelte';
	import IslandCapsule from './IslandCapsule.svelte';
	import type { AuthorInfo } from '@/stores/header-island.svelte';

	// PageIsland — 桌面端顶栏中央「页面锚点岛」的业务层（形态见 IslandCapsule）。
	// 现升级为「默认常驻 + 交互中枢」模型：
	//   - 默认常驻：页面只要注册即在顶栏中央稳定呈现，不再依赖滚动飞行动画；
	//   - 交互中枢：点击胶囊可单向向下展开控制面板，承载分类切换、排序、作者互动与回顶；
	//   - 紧凑尺寸：32px（h-8）极薄胶囊，彻底解放页面首屏内容。

	let {
		title = '',
		meta = '',
		author,
		onBack,
		backLabel = '返回',
		onTitleClick,
		progress = 0,
		icon,
		trailing,
		expanded
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
		/** 阅读进度 0–100，渲染为胶囊最右侧圆环 */
		progress?: number;
		/** 左侧语义图标（可选；P1-4 岛 leading 视觉锚点） */
		icon?: Snippet;
		/** 右侧行动区（页面主行动，≤2 控件） */
		trailing?: Snippet;
		/** 展开态控制面板内容（可选） */
		expanded?: Snippet;
	} = $props();

	const displayTitle = $derived(author?.name || title || '详情');
	const metaText = $derived(
		meta || (author ? [author.location, author.time].filter(Boolean).join(' · ') : '')
	);
	const titleAria = $derived(
		author ? `查看 ${author.name} 的资料` : `回到「${title || '页面标题'}」`
	);
</script>

{#snippet defaultAuthorPanel()}
	<div class="flex flex-col gap-3 py-1">
		<div class="flex items-center justify-between gap-3">
			<div class="flex min-w-0 items-center gap-2.5">
				<Avatar src={author?.avatar} name={author?.name ?? '作者'} size={34} class="shrink-0 ring-1 ring-border-default/40" />
				<div class="min-w-0 flex-1">
					<div class="flex items-center gap-1.5">
						<span class="truncate text-xs font-semibold text-text-primary">{author?.name}</span>
						<span class="rounded bg-accent-soft px-1.5 py-0.2 text-[10px] font-medium text-text-accent-on-soft">作者</span>
					</div>
					{#if author?.location || author?.time}
						<div class="truncate text-xxs font-mono text-text-secondary">
							{[author.location, author.time].filter(Boolean).join(' · ')}
						</div>
					{/if}
				</div>
			</div>
			<div class="flex shrink-0 items-center gap-1.5">
				<Button
					variant="primary"
					size="sm"
					shape="pill"
					onclick={() => onTitleClick?.()}
				>
					查看资料
				</Button>
			</div>
		</div>
		<div class="flex items-center justify-between border-t border-border-default/20 pt-2 text-xxs text-text-secondary">
			<span>已常驻于顶栏中枢</span>
			<button
				type="button"
				class="text-accent-default hover:underline"
				onclick={() => onTitleClick?.()}
			>
				平滑回顶 ↑
			</button>
		</div>
	</div>
{/snippet}

{#snippet defaultPagePanel()}
	<div class="flex flex-col gap-2.5 py-1">
		<div class="flex items-center justify-between text-xs text-text-secondary">
			<span class="font-medium text-text-primary">页面控制与导航</span>
			{#if metaText}
				<span class="font-mono text-xxs text-accent-default">{metaText}</span>
			{/if}
		</div>
		{#if trailing}
			<div class="flex items-center gap-2 pt-0.5">
				{@render trailing()}
			</div>
		{/if}
		<div class="flex items-center justify-between border-t border-border-default/20 pt-2 text-xxs text-text-secondary">
			<span>轻触回顶或点击任意处收起</span>
			<button
				type="button"
				class="text-accent-default hover:underline"
				onclick={() => onTitleClick?.()}
			>
				平滑回顶 ↑
			</button>
		</div>
	</div>
{/snippet}

<IslandCapsule
	ariaLabel="页面锚点与快捷操作"
	title={displayTitle}
	meta={metaText}
	titleAria={titleAria}
	onTitleClick={() => onTitleClick?.()}
	{progress}
	{trailing}
	expanded={expanded ?? (author ? defaultAuthorPanel : defaultPagePanel)}
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
		{:else if icon}
			<span class="flex size-5 shrink-0 items-center justify-center rounded-full bg-accent-soft text-text-accent-on-soft">
				{@render icon()}
			</span>
		{:else}
			<span class="ml-1 mr-0.5 size-1.5 shrink-0 rounded-full bg-accent-default/70" aria-hidden="true"></span>
		{/if}
	{/snippet}
</IslandCapsule>
