<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { Home, Plus, X, MessageCircle, CircleUser, Compass } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';
	import { composer, openComposer, closeComposer, toggleComposer } from '@/stores/composer.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import CreateSheet from '@/components/overlays/CreateSheet.svelte';
	import StatusBar from '@/components/layout/StatusBar.svelte';
	import NavTab from '@/components/layout/mobile/NavTab.svelte';
	import RoutedContentViewport from '@/components/layout/page/RoutedContentViewport.svelte';
	import { ROUTES, isPrimaryRouteActive, isSubPagePath } from '@/lib/routes';

	// 平板布局（对齐 Figma Make TabletLayout 契约）：
	// - 紧凑平板（<960px）：贴边导航条（navGlass 背景 + 顶部边框 + elevation-nav）
	// - 宽平板（≥960px，isWideTablet）：居中 680px 浮动玻璃 Dock（radius 32 + 上抛阴影）
	// 主导航永不随宽设备全宽生长；选中 pill 使用导航语义 token。
	// Tab 对齐鸿蒙端 PhoneLayout 五 slot：首页 / 附近 / 发布FAB / 消息 / 我的。

	let { children }: { children: Snippet } = $props();

	const NAV_ITEMS: { path: string; icon: Component; label: string }[] = [
		{ path: ROUTES.home, icon: Home, label: '首页' },
		{ path: ROUTES.nearby, icon: Compass, label: '附近' },
		{ path: ROUTES.messages, icon: MessageCircle, label: '消息' },
		{ path: ROUTES.me, icon: CircleUser, label: '我的' }
	];

	const pathname = $derived(page.url.pathname);
	const isSubPage = $derived(isSubPagePath(pathname));
	const showCreate = $derived(composer.composerOpen);
	const isWide = $derived(platform.isWideTablet);
</script>

<div
	data-platform="tablet"
	class="flex flex-col h-dvh w-full overflow-hidden transition-colors duration-gentle font-sans bg-surface-page text-text-primary"
>
	<StatusBar />

	<!-- RoutedContentViewport 是三端唯一的路由内容与滚动承载区。
	     地图/消息为全画布页：禁用滚动，canvas 固定视口高度。
	     宽平板浮动 Dock 悬于内容之上，画布需预留底部净空。 -->
	<RoutedContentViewport
		contentClassName="tablet-page-canvas {pathname === ROUTES.map ? 'h-full' : ''} {isWide ? 'pb-28' : ''}"
		style={(pathname === ROUTES.map || pathname === ROUTES.messages) ? 'overflow:hidden;' : undefined}
	>
		{@render children()}
	</RoutedContentViewport>

	<!-- Compact tablet: edge-attached bar; wide tablet: centred floating Dock -->
	{#if !isSubPage}
		<nav
			aria-label="主导航"
			class="shrink-0 w-full flex justify-center"
			style={isWide
				? 'padding:10px 24px max(16px, env(safe-area-inset-bottom, 16px));background-color:transparent;'
				: 'padding:0;border-top:var(--cst-border-width-hairline) solid var(--cst-border-nav-glass);box-shadow:var(--cst-elevation-nav);'}
		>
			<div
				class="grid grid-cols-5 items-center w-full"
				style="width:100%;max-width:{isWide ? 'var(--cst-tablet-dock-max)' : 'none'};border-radius:{isWide ? 'var(--radius-sheet)' : 0};background:var(--cst-surface-nav-glass);backdrop-filter:blur(var(--cst-blur-lg)) saturate(1.8);-webkit-backdrop-filter:blur(var(--cst-blur-lg)) saturate(1.8);box-shadow:{isWide ? 'var(--cst-elevation-nav)' : 'none'};border:{isWide ? 'var(--cst-border-width-hairline) solid var(--cst-border-nav-glass)' : 'none'};padding:{isWide ? '8px 14px' : '8px max(32px, env(safe-area-inset-left, 32px)) max(14px, env(safe-area-inset-bottom, 14px))'};"
			>
				<!-- Slot 1: 首页 -->
				<div class="flex items-center justify-center w-full">
					<NavTab
						label={NAV_ITEMS[0].label}
						iconSize={24}
						Icon={NAV_ITEMS[0].icon}
						isActive={isPrimaryRouteActive(pathname, NAV_ITEMS[0].path)}
						onClick={() => goto(NAV_ITEMS[0].path)}
					/>
				</div>

				<!-- Slot 2: 附近 -->
				<div class="flex items-center justify-center w-full">
					<NavTab
						label={NAV_ITEMS[1].label}
						iconSize={24}
						Icon={NAV_ITEMS[1].icon}
						isActive={isPrimaryRouteActive(pathname, NAV_ITEMS[1].path)}
						onClick={() => goto(NAV_ITEMS[1].path)}
					/>
				</div>

				<!-- Slot 3: Centre FAB -->
				<div class="flex items-center justify-center w-full">
					<button
						type="button"
						class="hover-btn scale-btn flex items-center justify-center rounded-full focus-ring select-none size-12 {composer.composerOpen ? 'bg-text-secondary' : 'bg-accent-default'}"
						style="box-shadow:{composer.composerOpen ? 'none' : 'var(--cst-elevation-fab)'};"
						aria-label={composer.composerOpen ? '关闭' : '发布'}
						onclick={toggleComposer}
					>
						{#if composer.composerOpen}
							<X class="size-5 text-text-on-accent" strokeWidth={2} />
						{:else}
							<Plus class="size-5 text-text-on-accent" strokeWidth={2} />
						{/if}
					</button>
				</div>

				<!-- Slot 4: 消息 -->
				<div class="flex items-center justify-center w-full">
					<NavTab
						label={NAV_ITEMS[2].label}
						iconSize={24}
						Icon={NAV_ITEMS[2].icon}
						isActive={isPrimaryRouteActive(pathname, NAV_ITEMS[2].path)}
						onClick={() => goto(NAV_ITEMS[2].path)}
					/>
				</div>

				<!-- Slot 5: 我的 -->
				<div class="flex items-center justify-center w-full">
					<NavTab
						label={NAV_ITEMS[3].label}
						iconSize={24}
						Icon={NAV_ITEMS[3].icon}
						isActive={isPrimaryRouteActive(pathname, NAV_ITEMS[3].path)}
						onClick={() => goto(NAV_ITEMS[3].path)}
					/>
				</div>
			</div>
		</nav>
	{/if}

	{#if showCreate}
		<Dialog.Root open={showCreate} onOpenChange={(next) => (next ? openComposer() : closeComposer())}>
			<Dialog.Content side="bottom" class="overflow-hidden sm:max-w-2xl" style="height:90%;">
				<CreateSheet onClose={closeComposer} />
			</Dialog.Content>
		</Dialog.Root>
	{/if}
</div>
