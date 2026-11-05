<script lang="ts">
	import { untrack, type Snippet } from 'svelte';
	import { ArrowLeft } from '@lucide/svelte';
	import { smartBack } from '@/lib/smart-back';
	import ContentColumn from './ContentColumn.svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import { cn } from '@/lib/utils';
	import { platform } from '@/stores/platform.svelte';
	import { headerIsland, type IslandToken } from '@/stores/header-island.svelte';
	import { islandRectOf, type IslandRect } from '@/lib/island-morph';
	import { handoffRecede, scrollHandoffProgress, subscribeHandoff } from '@/lib/island-handoff';

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
		maxWidth,
		class: className = '',
		contentClassName = ''
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
		contentClassName?: string;
	} = $props();

	let scrolled = $state(false);
	let hidden = $state(false);
	/** 桌面端交接的锚点快照：左侧工具行（返回 + 面包屑）在吸顶容器里位置稳定，抓一次即可。 */
	let islandSource: IslandRect | null = null;
	/** 交接是否已基本落位（p>0.55）：决定页内工具行是否 aria-hidden / inert。 */
	let islandSettled = $state(false);
	// 详情页的桌面岛：滚动驱动交接（和 PageHeader 同一套模型），token 保证路由切换不串台。
	const islandToken: IslandToken = Symbol('detail-header-island');

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
			const uiScale = Math.max(1, platform.uiScale);

			scrolled = current > scrollThreshold * uiScale;

			// 标头吸顶逻辑：滚动离开首屏后保持优雅吸顶且常驻，不自动避让。
			hidden = false;

			lastScrollTop = Math.max(0, current);

			// 桌面端：标题/身份 → 顶栏岛的交接由滚动进度驱动（160px 起、280px 落位）。
			// 和 PageHeader 同一套 scroll-linked 模型：跟手、可逆、无独立时长动画。
			if (platform.platform !== 'desktop') return;
			if (!islandSource) {
				const captured = islandRectOf(node.querySelector('[data-detail-header-left]'));
				// 工具行锚点收紧到顶栏带内：交接沿顶栏横向滑入，不掠过正文。
				islandSource = captured
					? {
							...captured,
							y: Math.min(captured.y, Math.max(0, (52 * uiScale - captured.height) / 2))
						}
					: null;
			}
			headerIsland.setHandoff(
				islandToken,
				scrollHandoffProgress(current, scrollThreshold * uiScale, (scrollThreshold + 120) * uiScale),
				islandSource
			);
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

	// ── 桌面端：页面锚点岛（顶栏中央）────────────────────────────────
	// 详情页身份（作者/群/用户）在滚动离开首屏后交给顶栏岛；移动/平板保留页内吸顶岛。

	$effect(() => {
		if (platform.platform !== 'desktop') return;
		headerIsland.register(
			islandToken,
			untrack(() => ({
				title: author?.name || entityName || category || '详情',
				meta: '',
				author,
				onBack,
				backLabel,
				onAuthorClick,
				trailing
			}))
		);
		return () => headerIsland.release(islandToken);
	});

	$effect(() => {
		if (platform.platform !== 'desktop') return;
		headerIsland.update(islandToken, {
			title: author?.name || entityName || category || '详情',
			meta: '',
			author,
			onBack,
			backLabel,
			onAuthorClick,
			trailing
		});
	});

	// 把交接帧映射成少量响应式状态（几何由 lib/island-handoff.ts 的 action 逐帧写）。
	$effect(() => {
		if (platform.platform !== 'desktop') return;
		return subscribeHandoff((frame) => {
			const settled = frame.p > 0.55;
			if (settled !== islandSettled) islandSettled = settled;
		});
	});
</script>

<header
	use:headerScroll
	use:handoffRecede={{ enabled: platform.platform === 'desktop', mode: 'band' }}
	class={cn(
		// 桌面端入岛退场由 handoffRecede 逐帧写 opacity（与滚动同源、可逆）；
		// 这里不能挂 transition-all：过渡会让每帧写入再叠加一个时长，肉眼就是「慢半拍」。
		'sticky top-0 z-[var(--cst-z-sticky)] w-full select-none pointer-events-none bg-transparent py-2 sm:py-2.5',
		hidden ? '-translate-y-full opacity-0' : 'translate-y-0 opacity-100',
		className
	)}
>
	<ContentColumn {maxWidth} class={contentClassName}>
			<!-- 两态交叉淡出（不再用 {#if} 硬切）：初始工具行与收敛态胶囊挂在同一行高容器里，
			     滚动过阈值时工具行淡出、胶囊淡入，桌面端由顶栏灵动岛从工具行位置收进去——
			     三者同一条时间轴，才没有「工具行瞬间消失、胶囊凭空出现」的断层。 -->
			<div class="relative h-10">
				<!-- ══ 初始态：两侧角落悬浮透亮光斑胶囊（返回 + 桌面实体面包屑 / 操作区）══ -->
				<div
					data-detail-header-anchor
					aria-hidden={platform.platform === 'desktop'
						? islandSettled
							? 'true'
							: undefined
						: scrolled
							? 'true'
							: undefined}
					inert={platform.platform === 'desktop' ? islandSettled : scrolled}
					class="absolute inset-0 flex items-center justify-between gap-2 {platform.platform !== 'desktop'
						? 'transition-[opacity,transform] duration-rapid ease-standard'
						: ''} {platform.platform !== 'desktop' && scrolled
						? 'pointer-events-none -translate-y-0.5 opacity-0'
						: ''}"
				>
					<!-- 左侧：返回按键 + 桌面实体面包屑 -->
					<div
						data-detail-header-left
						class="pointer-events-auto flex items-center transition-[opacity,transform] duration-snappy rounded-full border border-border-default/60 bg-surface-primary/85 text-text-primary p-0.5 shadow-raised-sm backdrop-blur-md"
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
									<span>{category}</span>
									{#if entityName}
										<span class="opacity-subtle">/</span>
									{/if}
								{/if}
								{#if entityName}
									<span class="max-w-[12.5rem] truncate text-text-primary font-semibold">{entityName}</span>
								{/if}
							</div>
						{/if}
					</div>

					<!-- 右侧：操作区（分享 / 更多等） -->
					{#if trailing}
						<div
							data-detail-header-right
							class="pointer-events-auto flex items-center gap-1 transition-[opacity,transform] duration-snappy rounded-full border border-border-default/60 bg-surface-primary/85 text-text-primary p-0.5 shadow-raised-sm backdrop-blur-md"
						>
							{@render trailing()}
						</div>
					{:else}
						<span class="size-9 shrink-0" aria-hidden="true"></span>
					{/if}
				</div>

				<!-- ══ 收敛态：居中灵动胶囊（移动/平板在页面内吸顶；桌面端由顶栏岛接手，lg:hidden）══
				     仅在滚动过阈值后挂载（移动端入场走短促 fade+zoom；桌面端不渲染，避免与顶栏岛重复）。 -->
				{#if scrolled}
				<div
					data-detail-header-island
					class="pointer-events-auto absolute inset-0 mx-auto flex h-full items-center justify-between gap-2 sm:gap-3 rounded-full border border-border-default/40 bg-surface-primary/90 px-2.5 text-text-primary shadow-raised-md animate-in fade-in zoom-in-95 duration-snappy max-w-[92%] sm:max-w-xl lg:hidden"
					style="contain: layout paint;"
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
								aria-label="查看 {author.name} 的资料"
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
				{/if}
			</div>
		</ContentColumn>
</header>