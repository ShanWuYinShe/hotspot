<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { MapPin, Compass, Plus, X, MessageCircle, CircleUser } from '@lucide/svelte';
	import { platformStore } from '@/stores/platform.svelte';
	import { composer, openComposer, closeComposer, toggleComposer } from '@/stores/composer.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import CreateSheet from '@/components/layout/CreateSheet.svelte';
	import StatusBar from '@/components/layout/StatusBar.svelte';
	import { ROUTES, isPrimaryRouteActive, isSubPagePath } from '@/lib/routes';

	// 平板布局（对齐 Figma Make TabletLayout 契约）：
	// - 紧凑平板（<960px）：贴边导航条（navGlass 背景 + 顶部边框 + elevation-nav）
	// - 宽平板（≥960px，isWideTablet）：居中 680px 浮动玻璃 Dock（radius 32 + 上抛阴影）
	// 主导航永不随宽设备全宽生长；选中 pill 使用导航语义 token。

	let { children }: { children: Snippet } = $props();

	const NAV_ITEMS: { path: string; icon: Component; label: string }[] = [
		{ path: ROUTES.home, icon: MapPin, label: '附近' },
		{ path: ROUTES.nearbyMoments, icon: Compass, label: '发现' },
		{ path: ROUTES.messages, icon: MessageCircle, label: '消息' },
		{ path: ROUTES.me, icon: CircleUser, label: '我的' }
	];

	const pathname = $derived(page.url.pathname);
	const isSubPage = $derived(isSubPagePath(pathname));
	const showCreate = $derived(composer.composerOpen);
	const isWide = $derived(platformStore.isWideTablet);
</script>

<div
	data-platform="tablet"
	class="flex flex-col h-[100dvh] w-full overflow-hidden transition-colors duration-500 font-sans"
	style="background-color:var(--neu-bg);color:var(--neu-text);"
>
	<StatusBar />

	<!-- Scrollable area: full width, individual elements keep their own proportions -->
	<!-- 地图探索为全画布页：不滚动，canvas 固定视口高度 -->
	<main class="flex-1 {[ROUTES.map, ROUTES.messages].includes(pathname) ? 'overflow-hidden' : 'overflow-y-auto overflow-x-hidden no-scrollbar'} w-full" style="background-color:var(--neu-bg);">
		<div class="tablet-page-canvas {pathname === ROUTES.map ? 'h-full' : ''}">
			{@render children()}
		</div>
	</main>

	<!-- Compact tablet: edge-attached bar; wide tablet: centred floating Dock -->
	{#if !isSubPage}
		<nav
			role="navigation"
			aria-label="主导航"
			class="flex-shrink-0 w-full flex justify-center"
			style={isWide
				? 'padding:10px 24px max(16px, env(safe-area-inset-bottom, 16px));background-color:transparent;'
				: 'padding:0;background-color:var(--surface-nav-glass);border-top:1px solid var(--border-nav-glass);box-shadow:var(--elevation-nav);'}
		>
			<div
				class="flex items-center"
				style="width:100%;max-width:{isWide ? 680 : 'none'};border-radius:{isWide ? 32 : 0}px;background:var(--surface-nav-glass);backdrop-filter:blur(28px) saturate(1.8);-webkit-backdrop-filter:blur(28px) saturate(1.8);box-shadow:{isWide ? 'var(--elevation-nav)' : 'none'};border:{isWide ? '1px solid var(--border-nav-glass)' : 'none'};padding:{isWide ? '8px 14px' : '8px max(32px, env(safe-area-inset-left, 32px)) max(14px, env(safe-area-inset-bottom, 14px))'};gap:{isWide ? 6 : 14}px;"
			>
				{#each NAV_ITEMS.slice(0, 2) as item}
					<button
						type="button"
						class="neu-hover-btn neu-nav-tab relative flex flex-col items-center justify-center outline-none select-none rounded-[var(--shape-panel)]"
						style="flex:1;min-width:0;padding-top:9px;padding-bottom:9px;gap:5px;"
						aria-label={item.label}
						aria-current={isPrimaryRouteActive(pathname, item.path) ? 'page' : undefined}
						onclick={() => void goto(item.path)}
					>
						{#if isPrimaryRouteActive(pathname, item.path)}
							<div style="position:absolute;inset:4px 6px;border-radius:var(--shape-panel);background:var(--navigation-active-indicator);opacity:0.14;"></div>
						{/if}
						<div style="display:flex;align-items:center;justify-content:center;position:relative;z-index:1;">
							<item.icon
								style="width:24px;height:24px;color:{isPrimaryRouteActive(pathname, item.path) ? 'var(--navigation-content-active)' : 'var(--navigation-content)'};transition:color 0.22s ease;"
								strokeWidth={isPrimaryRouteActive(pathname, item.path) ? 2.1 : 1.5}
							/>
						</div>
						<span
							style="font-size:11px;line-height:1;color:{isPrimaryRouteActive(pathname, item.path) ? 'var(--navigation-content-active)' : 'var(--navigation-content)'};font-weight:{isPrimaryRouteActive(pathname, item.path) ? 600 : 400};transition:color 0.22s ease;letter-spacing:0.02em;position:relative;z-index:1;"
						>
							{item.label}
						</span>
					</button>
				{/each}

				<!-- Centre FAB -->
				<div class="flex items-center justify-center flex-shrink-0" style="width:64px;">
					<button
						type="button"
						class="neu-hover-btn neu-scale-btn flex items-center justify-center rounded-full outline-none select-none"
						style="width:48px;height:48px;background:{composer.composerOpen ? 'var(--neu-text-muted)' : 'var(--neu-accent-primary)'};box-shadow:{composer.composerOpen ? 'none' : 'var(--elevation-fab)'};transition:background 0.22s, box-shadow 0.22s, transform 0.18s cubic-bezier(0.22,1,0.36,1), filter 0.18s ease;"
						aria-label={composer.composerOpen ? '关闭' : '发布'}
						onclick={toggleComposer}
					>
						{#if composer.composerOpen}
							<X style="width:19px;height:19px;color:white;" strokeWidth={2.2} />
						{:else}
							<Plus style="width:19px;height:19px;color:white;" strokeWidth={2.2} />
						{/if}
					</button>
				</div>

				{#each NAV_ITEMS.slice(2, 4) as item}
					<button
						type="button"
						class="neu-hover-btn neu-nav-tab relative flex flex-col items-center justify-center outline-none select-none rounded-[var(--shape-panel)]"
						style="flex:1;min-width:0;padding-top:9px;padding-bottom:9px;gap:5px;"
						aria-label={item.label}
						aria-current={isPrimaryRouteActive(pathname, item.path) ? 'page' : undefined}
						onclick={() => void goto(item.path)}
					>
						{#if isPrimaryRouteActive(pathname, item.path)}
							<div style="position:absolute;inset:4px 6px;border-radius:var(--shape-panel);background:var(--navigation-active-indicator);opacity:0.14;"></div>
						{/if}
						<div style="display:flex;align-items:center;justify-content:center;position:relative;z-index:1;">
							<item.icon
								style="width:24px;height:24px;color:{isPrimaryRouteActive(pathname, item.path) ? 'var(--navigation-content-active)' : 'var(--navigation-content)'};transition:color 0.22s ease;"
								strokeWidth={isPrimaryRouteActive(pathname, item.path) ? 2.1 : 1.5}
							/>
						</div>
						<span
							style="font-size:11px;line-height:1;color:{isPrimaryRouteActive(pathname, item.path) ? 'var(--navigation-content-active)' : 'var(--navigation-content)'};font-weight:{isPrimaryRouteActive(pathname, item.path) ? 600 : 400};transition:color 0.22s ease;letter-spacing:0.02em;position:relative;z-index:1;"
						>
							{item.label}
						</span>
					</button>
				{/each}
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
