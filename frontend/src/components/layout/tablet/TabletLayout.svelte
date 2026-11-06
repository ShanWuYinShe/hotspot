<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { Home, Plus, X, MessageCircle, CircleUser, Compass } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';
	import { composer, openComposer, closeComposer, toggleComposer } from '@/stores/composer.svelte';
	import { unread } from '@/stores/unread.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import CreateSheet from '@/components/overlays/CreateSheet.svelte';
	
	import NavTab from '@/components/layout/mobile/NavTab.svelte';
	import RoutedContentViewport from '@/components/layout/page/RoutedContentViewport.svelte';
	import AmbientBackdrop from '@/components/layout/ambient/AmbientBackdrop.svelte';
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
	<!-- RoutedContentViewport 是三端唯一的路由内容与滚动承载区。
	     地图/消息为全画布页：禁用滚动，canvas 固定视口高度。
	     宽平板 Dock 是流内浮动条（外层透明 padding + 居中玻璃轨道），不与内容重叠，
	     因此内容区不做 pb-* 预留——此前 pb-28 会把 h-full 全画布页真实内容盒压掉 112px。 -->
	<div class="relative isolate flex flex-1 min-h-0 flex-col overflow-hidden">
		<!-- 氛围画布固定在外壳内容区（不随路由滚动） -->
		<AmbientBackdrop />
		<RoutedContentViewport
			contentClassName="tablet-page-canvas {(pathname === ROUTES.map || (pathname === ROUTES.messages && isWide)) ? 'h-full' : (!isSubPage ? 'pb-36' : '')}"
			style={(pathname === ROUTES.map || pathname === ROUTES.messages) ? 'overflow:hidden;' : undefined}
		>
			{@render children()}
		</RoutedContentViewport>
	</div>

	<!-- 平板端统一悬浮孤岛导航栏（Floating Dock Island） -->
	{#if !isSubPage}
		<nav
			aria-label="主导航"
			class="fixed bottom-0 inset-x-0 z-[var(--cst-z-fixed)] pointer-events-none flex justify-center px-6 pb-[max(16px,env(safe-area-inset-bottom,16px))]"
		>
			<div
				class="pointer-events-auto flex items-center justify-between w-full max-w-lg rounded-[28px] border border-border-default/70 bg-surface-island/92 p-1.5 shadow-[0_12px_36px_rgba(0,0,0,0.35)] backdrop-blur-2xl ring-1 ring-white/10 dark:ring-white/5"
			>
				<!-- Slot 1: 首页 -->
				<NavTab
					label={NAV_ITEMS[0].label}
					iconSize={25}
					Icon={NAV_ITEMS[0].icon}
					isActive={isPrimaryRouteActive(pathname, NAV_ITEMS[0].path)}
					onClick={() => goto(NAV_ITEMS[0].path)}
				/>

				<!-- Slot 2: 附近 -->
				<NavTab
					label={NAV_ITEMS[1].label}
					iconSize={25}
					Icon={NAV_ITEMS[1].icon}
					isActive={isPrimaryRouteActive(pathname, NAV_ITEMS[1].path)}
					onClick={() => goto(NAV_ITEMS[1].path)}
				/>

				<!-- Slot 3: Centre FAB (发布) -->
				<div class="flex items-center justify-center shrink-0 px-1.5">
					<button
						type="button"
						class="hover-btn scale-btn flex items-center justify-center rounded-full focus-ring select-none size-12 shadow-[0_4px_16px_rgba(var(--cst-palette-accent-primary),0.35)] active:scale-95 transition-all duration-rapid {composer.composerOpen ? 'bg-text-secondary' : 'bg-accent-default'}"
						aria-label={composer.composerOpen ? '关闭' : '发布'}
						onclick={toggleComposer}
					>
						{#if composer.composerOpen}
							<X class="size-5 text-text-on-accent" strokeWidth={2.25} />
						{:else}
							<Plus class="size-5.5 text-text-on-accent" strokeWidth={2.25} />
						{/if}
					</button>
				</div>

				<!-- Slot 4: 消息 -->
				<NavTab
					label={NAV_ITEMS[2].label}
					iconSize={25}
					Icon={NAV_ITEMS[2].icon}
					isActive={isPrimaryRouteActive(pathname, NAV_ITEMS[2].path)}
					onClick={() => goto(NAV_ITEMS[2].path)}
					count={unread.chat}
				/>

				<!-- Slot 5: 我的 -->
				<NavTab
					label={NAV_ITEMS[3].label}
					iconSize={25}
					Icon={NAV_ITEMS[3].icon}
					isActive={isPrimaryRouteActive(pathname, NAV_ITEMS[3].path)}
					onClick={() => goto(NAV_ITEMS[3].path)}
				/>
			</div>
		</nav>
	{/if}

	{#if showCreate}
		<Dialog.Root open={showCreate} onOpenChange={(next) => (next ? openComposer() : closeComposer())}>
			<Dialog.Content side="bottom" class="overflow-hidden sm:max-w-2xl" style="height:var(--cst-sheet-height-xl);">
				<CreateSheet onClose={closeComposer} />
			</Dialog.Content>
		</Dialog.Root>
	{/if}
</div>
