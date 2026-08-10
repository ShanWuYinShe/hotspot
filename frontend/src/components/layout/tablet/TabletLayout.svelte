<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { Home, MapPin, Plus, X, MessageCircle, CircleUser } from '@lucide/svelte';
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

	let { children }: { children: Snippet } = $props();

	const NAV_ITEMS: { path: string; icon: Component; label: string }[] = [
		{ path: ROUTES.home, icon: Home, label: '首页' },
		{ path: ROUTES.nearby, icon: MapPin, label: '附近' },
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
	class="flex flex-col h-dvh w-full overflow-hidden transition-colors duration-gentle font-sans"
	style="background-color:var(--cst-surface-primary);color:var(--cst-text-primary);"
>
	<StatusBar />

	<!-- RoutedContentViewport 是三端唯一的路由内容与滚动承载区。
	     地图/消息为全画布页：禁用滚动，canvas 固定视口高度。 -->
	<RoutedContentViewport
		contentClassName="tablet-page-canvas {pathname === ROUTES.map ? 'h-full' : ''}"
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
				: 'padding:0;background-color:var(--cst-surface-nav-glass);border-top:var(--cst-border-width-hairline) solid var(--cst-border-nav-glass);box-shadow:var(--cst-elevation-nav);'}
		>
			<div
				class="flex items-center"
				style="width:100%;max-width:{isWide ? 'var(--cst-tablet-dock-max)' : 'none'};border-radius:{isWide ? 'var(--radius-sheet)' : 0};background:var(--cst-surface-nav-glass);backdrop-filter:blur(var(--cst-blur-lg)) saturate(1.8);-webkit-backdrop-filter:blur(var(--cst-blur-lg)) saturate(1.8);box-shadow:{isWide ? 'var(--cst-elevation-nav)' : 'none'};border:{isWide ? '1px solid var(--cst-border-nav-glass)' : 'none'};padding:{isWide ? '8px 14px' : '8px max(32px, env(safe-area-inset-left, 32px)) max(14px, env(safe-area-inset-bottom, 14px))'};gap:{isWide ? 6 : 14}px;"
			>
				<!-- 左侧：首页 + 附近（flex-1 与右侧对称，保持 FAB 居中） -->
				<div class="flex flex-1 justify-center gap-4">
					{#each NAV_ITEMS.slice(0, 2) as item}
						<NavTab
							label={item.label}
							iconSize={24}
							Icon={item.icon}
							isActive={isPrimaryRouteActive(pathname, item.path)}
							onClick={() => goto(item.path)}
						/>
					{/each}
				</div>

				<!-- Centre FAB -->
				<div class="flex items-center justify-center shrink-0 w-16" >
					<button
						type="button"
						class="hover-btn scale-btn flex items-center justify-center rounded-full focus-ring select-none size-12"
						style="background:{composer.composerOpen ? 'var(--cst-text-secondary)' : 'var(--cst-accent-default)'};box-shadow:{composer.composerOpen ? 'none' : 'var(--cst-elevation-fab)'};transition:background var(--cst-motion-standard) var(--cst-ease-standard), box-shadow var(--cst-motion-standard) var(--cst-ease-standard), transform var(--cst-motion-snappy) var(--cst-ease-standard), filter var(--cst-motion-snappy) var(--cst-ease-standard);"
						aria-label={composer.composerOpen ? '关闭' : '发布'}
						onclick={toggleComposer}
					>
						{#if composer.composerOpen}
							<X class="size-5 text-text-on-accent"   strokeWidth={2} />
						{:else}
							<Plus class="size-5 text-text-on-accent"   strokeWidth={2} />
						{/if}
					</button>
				</div>

				<!-- 右侧：消息 + 我的（flex-1 与左侧对称） -->
				<div class="flex flex-1 justify-center gap-4">
					{#each NAV_ITEMS.slice(2, 4) as item}
						<NavTab
							label={item.label}
							iconSize={24}
							Icon={item.icon}
							isActive={isPrimaryRouteActive(pathname, item.path)}
							onClick={() => goto(item.path)}
						/>
					{/each}
				</div>
			</div>
		</nav>
	{/if}

	{#if showCreate}
		<Dialog.Root open={showCreate} onOpenChange={(next) => (next ? openComposer() : closeComposer())}>
			<Dialog.Content side="bottom" class="overflow-hidden" style="height:90%;">
				<CreateSheet onClose={closeComposer} />
			</Dialog.Content>
		</Dialog.Root>
	{/if}
</div>
