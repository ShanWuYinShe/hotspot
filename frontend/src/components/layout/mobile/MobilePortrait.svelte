<script lang="ts">
	import type { Snippet } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { ArrowLeft, Home, Plus, X, MessageCircle, CircleUser, Compass } from '@lucide/svelte';
	import { cn } from '@/lib/utils';
	import { composer, openComposer, closeComposer, toggleComposer } from '@/stores/composer.svelte';
	import { unread } from '@/stores/unread.svelte';
	import CreateSheet from '@/components/overlays/CreateSheet.svelte';
	import * as Dialog from '@/components/ui/dialog';
	
	import NavTab from '@/components/layout/mobile/NavTab.svelte';
	import RoutedContentViewport from '@/components/layout/page/RoutedContentViewport.svelte';
	import AmbientBackdrop from '@/components/layout/ambient/AmbientBackdrop.svelte';
	import { ROUTES, isPrimaryRouteActive, isSubPagePath } from '@/lib/routes';

	// MobilePortrait — 移动竖屏壳：状态栏 + 内容区 + 底部 4 Tab 导航（中央 FAB + 发布 Sheet）。
	// 仅移动 portrait 分支使用（MainLayout 分派）。
	// Tab 对齐鸿蒙端 PhoneLayout 五 slot：首页 / 附近 / 发布FAB / 消息 / 我的。
	// 「附近」为一级 Tab，落点为 /nearby 枢纽页（2026-10-19 IA 调整：人/瞬间/群组
	// 三段分区 + Hero 统计，各段「查看全部」进三个独立叶子页；枢纽用分区而非段位胶囊，
	// 避免与叶子页自带筛选胶囊叠成双 pill 行）。地理轴内容只归「附近」，首页为热门发现轴。

	let { children }: { children: Snippet } = $props();

	const NAV_ITEMS: { path: string; icon: typeof Home; label: string }[] = [
		{ path: ROUTES.home, icon: Home, label: '首页' },
		{ path: ROUTES.nearby, icon: Compass, label: '附近' },
		{ path: ROUTES.messages, icon: MessageCircle, label: '消息' },
		{ path: ROUTES.me, icon: CircleUser, label: '我的' }
	];

	const pathname = $derived(page.url.pathname);
	const isSubPage = $derived(isSubPagePath(pathname));

	// FAB 悬浮投影：统一走 --cst-elevation-fab（中性高雅，无彩色炫光）
	const fabShadow = $derived(composer.composerOpen ? 'none' : 'var(--cst-elevation-fab)');

	// 单手边缘轻扫返回手势（Swipe-to-Go-Back）——解救单手大屏握持痛点
	let touchStartX = 0;
	let touchStartY = 0;
	let swipeDistance = $state(0);
	let isEdgeSwiping = $state(false);

	function handleTouchStart(e: TouchEvent) {
		if (!isSubPage || e.touches.length !== 1) return;
		const touch = e.touches[0];
		if (!touch) return;
		if (touch.clientX <= 32) {
			touchStartX = touch.clientX;
			touchStartY = touch.clientY;
			isEdgeSwiping = true;
			swipeDistance = 0;
		}
	}

	function handleTouchMove(e: TouchEvent) {
		if (!isEdgeSwiping || e.touches.length !== 1) return;
		const touch = e.touches[0];
		if (!touch) return;
		const dx = touch.clientX - touchStartX;
		const dy = touch.clientY - touchStartY;
		// 垂直滚动倾向明显时取消边缘滑动手势
		if (Math.abs(dy) > Math.abs(dx) * 1.2 && dx < 20) {
			isEdgeSwiping = false;
			swipeDistance = 0;
			return;
		}
		if (dx > 0) {
			swipeDistance = Math.min(dx, 120);
		}
	}

	function handleTouchEnd() {
		if (isEdgeSwiping && swipeDistance >= 60) {
			history.back();
		}
		isEdgeSwiping = false;
		swipeDistance = 0;
	}
</script>

<svelte:window
	ontouchstart={handleTouchStart}
	ontouchmove={handleTouchMove}
	ontouchend={handleTouchEnd}
	ontouchcancel={handleTouchEnd}
/>

<div
	class="bg-surface-page min-h-dvh w-full flex justify-center items-start font-sans transition-colors duration-gentle"
>
	<div
		class={cn('bg-surface-page text-text-primary flex flex-col h-dvh w-full', 'transition-colors duration-gentle overflow-hidden relative')}
	>
		{#if isEdgeSwiping && swipeDistance > 15}
			<div
				class="pointer-events-none fixed left-0 top-1/2 -translate-y-1/2 z-[var(--cst-z-toast)] flex items-center gap-1.5 rounded-r-full bg-surface-primary/95 px-2.5 py-3 shadow-raised-lg border-y border-r border-border-default transition-all duration-rapid"
				style={`transform: translate3d(${Math.min(swipeDistance * 0.35, 20)}px, -50%, 0); opacity: ${Math.min(swipeDistance / 40, 1)};`}
			>
				<ArrowLeft class="size-4 text-accent-default" strokeWidth={2} />
				{#if swipeDistance >= 60}
					<span class="text-xs font-medium text-text-primary pr-1">松开返回</span>
				{/if}
			</div>
		{/if}

		<div class="relative isolate flex-1 min-h-0 overflow-hidden">
			<!-- 氛围画布固定在外壳内容区（不随路由滚动） -->
			<AmbientBackdrop />
			<RoutedContentViewport>{@render children()}</RoutedContentViewport>
		</div>

		{#if !isSubPage}
			<nav
				aria-label="主导航"
				class="shrink-0 w-full nav-glass nav-glass--bottom"
			>
				<div
					class="grid grid-cols-5 items-center w-full px-2"
					style="padding-top:var(--cst-bottom-nav-pad-top);padding-bottom:max(var(--cst-bottom-nav-pad-bottom), env(safe-area-inset-bottom, var(--cst-bottom-nav-pad-bottom)));"
				>
					<!-- Slot 1: 首页 (20%) -->
					<div class="flex items-center justify-center w-full">
						<NavTab
							label={NAV_ITEMS[0].label}
							isActive={isPrimaryRouteActive(pathname, NAV_ITEMS[0].path)}
							onClick={() => goto(NAV_ITEMS[0].path)}
							Icon={NAV_ITEMS[0].icon}
						/>
					</div>

					<!-- Slot 2: 附近 (20%) -->
					<div class="flex items-center justify-center w-full">
						<NavTab
							label={NAV_ITEMS[1].label}
							isActive={isPrimaryRouteActive(pathname, NAV_ITEMS[1].path)}
							onClick={() => goto(NAV_ITEMS[1].path)}
							Icon={NAV_ITEMS[1].icon}
						/>
					</div>

					<!-- Slot 3: Centre FAB (20%) -->
					<div class="flex items-center justify-center w-full">
						<button
							type="button"
							class="hover-btn scale-btn flex items-center justify-center focus-ring select-none rounded-full size-12"
							style="background:{composer.composerOpen ? 'var(--cst-text-secondary)' : 'var(--cst-accent-default)'};box-shadow:{fabShadow};"
							onclick={toggleComposer}
							aria-label={composer.composerOpen ? '关闭' : '发布'}
						>
							{#if composer.composerOpen}
								<X class="size-5 text-text-on-accent" strokeWidth={2} />
							{:else}
								<Plus class="size-5 text-text-on-accent" strokeWidth={2} />
							{/if}
						</button>
					</div>

					<!-- Slot 4: 消息 (20%) -->
					<div class="flex items-center justify-center w-full">
						<NavTab
							label={NAV_ITEMS[2].label}
							isActive={isPrimaryRouteActive(pathname, NAV_ITEMS[2].path)}
							onClick={() => goto(NAV_ITEMS[2].path)}
							Icon={NAV_ITEMS[2].icon}
							count={unread.chat}
						/>
					</div>

					<!-- Slot 5: 我的 (20%) -->
					<div class="flex items-center justify-center w-full">
						<NavTab
							label={NAV_ITEMS[3].label}
							isActive={isPrimaryRouteActive(pathname, NAV_ITEMS[3].path)}
							onClick={() => goto(NAV_ITEMS[3].path)}
							Icon={NAV_ITEMS[3].icon}
						/>
					</div>
				</div>
			</nav>
		{/if}

		<Dialog.Root open={composer.composerOpen} onOpenChange={(next) => (next ? openComposer() : closeComposer())}>
			<Dialog.Content side="bottom" class="overflow-hidden" style="height:var(--cst-sheet-height-2xl);">
				<CreateSheet onClose={closeComposer} />
			</Dialog.Content>
		</Dialog.Root>
	</div>
</div>
