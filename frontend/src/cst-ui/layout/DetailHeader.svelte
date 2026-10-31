<script lang="ts">
	import type { Snippet } from 'svelte';
	import { ArrowLeft } from '@lucide/svelte';
	import { smartBack } from '@/lib/smart-back';
	import ContentColumn from './ContentColumn.svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import { cn } from '@/lib/utils';

	export interface AuthorInfo {
		id?: string;
		name: string;
		avatar?: string | null;
		location?: string | null;
		time?: string | null;
	}

	let {
		onBack = () => smartBack(),
		backLabel = '返回',
		category = '',
		entityName = '',
		author,
		trailing,
		onAuthorClick,
		scrollThreshold = 160,
		maxWidth = '58rem',
		class: className = ''
	}: {
		onBack?: () => void;
		backLabel?: string;
		category?: string;
		entityName?: string;
		author?: AuthorInfo;
		trailing?: Snippet;
		onAuthorClick?: () => void;
		scrollThreshold?: number;
		maxWidth?: number | string;
		class?: string;
	} = $props();

	let scrolled = $state(false);
	let hidden = $state(false);

	// 监听滚动与方向（向上微划滑入 / 向下深划避让 / 顶部常驻）
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
		let raf = 0;
		let lastScrollTop = 0;

		const apply = () => {
			raf = 0;
			const current = target.scrollTop;
			const delta = current - lastScrollTop;

			scrolled = current > scrollThreshold;

			// 标头吸顶逻辑：
			// 移除过度自动避让，滚动离开首屏后保持优雅吸顶且常驻，
			// 不在用户向下浏览长文或评论时莫名消失，彻底消除「完全看不到标头」的问题。
			hidden = false;

			lastScrollTop = Math.max(0, current);
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
</script>

<header
	use:headerScroll
	class={cn(
		'sticky top-0 z-[var(--cst-z-sticky)] w-full select-none transition-all duration-snappy ease-standard pointer-events-none bg-transparent py-2 sm:py-2.5',
		hidden ? '-translate-y-full opacity-0' : 'translate-y-0 opacity-100',
		className
	)}
>
	<ContentColumn {maxWidth} class="px-3 sm:px-4">
		{#if scrolled}
			<!-- ══ 方案 A：居中灵动胶囊岛（Unified Dynamic Island Capsule） ══
			     滚动离开首屏后，以精致的居中胶囊岛浮现：将返回、作者微名片与操作按钮
			     收敛为一颗有机整体，半透毛玻璃融入内容，绝不对图面造成生硬横断。 -->
			<div
				data-detail-header-island
				class="pointer-events-auto mx-auto flex items-center justify-between gap-2 sm:gap-3 h-10 px-2.5 sm:px-3 rounded-full bg-surface-primary/90 backdrop-blur-xl border border-border-default/40 shadow-raised-md text-text-primary transition-all max-w-[92%] sm:max-w-xl animate-in fade-in zoom-in-95 duration-snappy"
				style="contain: layout paint; transform: translateZ(0);"
			>
				<!-- 左侧：返回按钮 -->
				<ActionIconButton
					icon={ArrowLeft}
					label={backLabel}
					onClick={onBack}
					variant="ghost"
					tone="muted"
					size={32}
					class="shrink-0"
				/>

				<div class="h-4 w-px bg-border-default/40 shrink-0" aria-hidden="true"></div>

				<!-- 中间：作者微名片 / 详情标题 -->
				<div
					data-detail-header-center
					class="flex min-w-0 flex-1 items-center justify-center overflow-hidden"
				>
					{#if author}
						<button
							type="button"
							class="hover-btn focus-ring flex min-w-0 max-w-full items-center gap-2 rounded-full px-2 py-0.5 text-left"
							onclick={() => onAuthorClick?.()}
							aria-label={`查看 ${author.name} 的资料`}
						>
							<Avatar src={author.avatar} name={author.name} size={22} class="shrink-0" />
							<div class="flex min-w-0 flex-col justify-center">
								<span class="truncate text-xs font-semibold text-text-primary leading-tight">
									{author.name}
								</span>
								{#if author.location || author.time}
									<span class="truncate text-xxs text-text-secondary leading-tight">
										{[author.location, author.time].filter(Boolean).join(' · ')}
									</span>
								{/if}
							</div>
						</button>
					{:else}
						<span class="truncate text-xs font-semibold text-text-primary px-1">
							{entityName || category || '详情'}
						</span>
					{/if}
				</div>

				{#if trailing}
					<div class="h-4 w-px bg-border-default/40 shrink-0" aria-hidden="true"></div>
					<!-- 右侧：操作区（分享 / 更多 / 互动等） -->
					<div data-detail-header-right class="flex items-center gap-0.5 sm:gap-1 shrink-0">
						{@render trailing()}
					</div>
				{/if}
			</div>
		{:else}
			<!-- ══ 顶部初始态：两侧角落悬浮透亮光斑胶囊 ══ -->
			<div class="flex items-center justify-between gap-2 h-9 sm:h-10">
				<!-- 左侧：返回按键 + 桌面实体面包屑 -->
				<div
					data-detail-header-left
					class="pointer-events-auto flex items-center transition-all duration-snappy rounded-full border border-content-inverse/20 bg-surface-media-scrim-strong text-content-inverse p-0.5 shadow-raised-sm backdrop-blur-md sm:border-border-default/60 sm:bg-surface-primary/80 sm:text-text-primary"
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

					{#if (category || entityName)}
						<div
							data-breadcrumb
							class="hidden sm:flex items-center gap-1.5 pl-0.5 pr-3 text-xs font-medium text-text-secondary animate-in fade-in duration-snappy"
						>
							{#if category}
								<span class="opacity-soft">{category}</span>
								{#if entityName}
									<span class="opacity-subtle">/</span>
								{/if}
							{/if}
							{#if entityName}
								<span class="max-w-[200px] truncate text-text-primary font-semibold">{entityName}</span>
							{/if}
						</div>
					{/if}
				</div>

				<!-- 右侧：操作区（分享 / 更多等） -->
				{#if trailing}
					<div
						data-detail-header-right
						class="pointer-events-auto flex items-center gap-1 transition-all duration-snappy rounded-full border border-content-inverse/20 bg-surface-media-scrim-strong text-content-inverse p-0.5 shadow-raised-sm backdrop-blur-md sm:border-border-default/60 sm:bg-surface-primary/80 sm:text-text-primary"
					>
						{@render trailing()}
					</div>
				{:else}
					<span class="size-9 shrink-0" aria-hidden="true"></span>
				{/if}
			</div>
		{/if}
	</ContentColumn>
</header>
