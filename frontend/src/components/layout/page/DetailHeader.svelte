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
		scrollThreshold = 200,
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

	// 监听滚动：向上查找最近带 overflow-y 的滚动容器，回退 #main-content
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
		const apply = () => {
			raf = 0;
			scrolled = target.scrollTop > scrollThreshold;
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
		'sticky top-0 z-[var(--cst-z-sticky)] w-full pointer-events-none transition-all duration-standard select-none',
		className
	)}
>
	<ContentColumn {maxWidth} class="pointer-events-none px-3 sm:px-4 py-2 sm:py-2.5">
		<div class="flex items-center justify-between gap-2">
			<!-- ══ 左侧：返回胶囊 + 桌面实体面包屑 ══ -->
			<div
				data-detail-header-left
				class="pointer-events-auto flex items-center rounded-full border border-border-default/60 bg-surface-primary/80 p-0.5 shadow-raised-sm backdrop-blur-md transition-all duration-standard"
			>
				<ActionIconButton
					icon={ArrowLeft}
					label={backLabel}
					onClick={onBack}
					variant="ghost"
					size={36}
					class="shrink-0"
				/>

				{#if (category || entityName) && !scrolled}
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

			<!-- ══ 中间：滚动吸顶微名片（仅在下滑离开首屏后渐显展开） ══ -->
			{#if scrolled}
				<div
					data-detail-header-center
					class="pointer-events-auto flex min-w-0 max-w-[280px] sm:max-w-[360px] items-center rounded-full border border-border-default/60 bg-surface-primary/90 px-2.5 py-1 shadow-raised-sm backdrop-blur-md animate-in fade-in zoom-in-95 duration-snappy"
				>
					{#if author}
						<button
							type="button"
							class="hover-btn focus-ring flex min-w-0 items-center gap-2 rounded-full text-left"
							onclick={() => onAuthorClick?.()}
							aria-label={`查看 ${author.name} 的资料`}
						>
							<Avatar src={author.avatar} name={author.name} size={24} class="shrink-0" />
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
			{/if}

			<!-- ══ 右侧：操作区（分享 / 更多 / 互动等） ══ -->
			{#if trailing}
				<div
					data-detail-header-right
					class="pointer-events-auto flex items-center gap-1 rounded-full border border-border-default/60 bg-surface-primary/80 p-0.5 shadow-raised-sm backdrop-blur-md"
				>
					{@render trailing()}
				</div>
			{:else}
				<span class="size-9 shrink-0" aria-hidden="true"></span>
			{/if}
		</div>
	</ContentColumn>
</header>
