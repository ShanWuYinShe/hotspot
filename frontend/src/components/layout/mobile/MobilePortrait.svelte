<script lang="ts">
	import type { Snippet } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { Home, MapPin, Plus, X, MessageCircle, CircleUser } from '@lucide/svelte';
	import { cn } from '@/lib/utils';
	import { composer, openComposer, closeComposer, toggleComposer } from '@/stores/composer.svelte';
	import CreateSheet from '@/components/overlays/CreateSheet.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import StatusBar from '@/components/layout/StatusBar.svelte';
	import NavTab from '@/components/layout/mobile/NavTab.svelte';
	import RoutedContentViewport from '@/components/layout/page/RoutedContentViewport.svelte';
	import { ROUTES, isPrimaryRouteActive, isSubPagePath } from '@/lib/routes';

	// MobilePortrait — 移动竖屏壳：状态栏 + 内容区 + 底部 4 Tab 导航（中央 FAB + 发布 Sheet）。
	// 仅移动 portrait 分支使用（MainLayout 分派）。

	let { children }: { children: Snippet } = $props();

	const NAV_ITEMS: { path: string; icon: typeof MapPin; label: string }[] = [
		{ path: ROUTES.home, icon: Home, label: '首页' },
		{ path: ROUTES.nearby, icon: MapPin, label: '附近' },
		{ path: ROUTES.messages, icon: MessageCircle, label: '消息' },
		{ path: ROUTES.me, icon: CircleUser, label: '我的' }
	];

	const pathname = $derived(page.url.pathname);
	const isSubPage = $derived(isSubPagePath(pathname));

	// FAB 悬浮投影：统一走 --cst-elevation-fab（中性高雅，无彩色炫光）
	const fabShadow = $derived(composer.composerOpen ? 'none' : 'var(--cst-elevation-fab)');
</script>

<div
	class="bg-surface-page min-h-dvh w-full flex justify-center items-start font-sans transition-colors duration-gentle"
>
	<div
		class={cn('bg-surface-page text-text-primary flex flex-col h-dvh w-full', 'transition-colors duration-gentle overflow-hidden relative')}
	>
		<StatusBar />

		<div class="flex-1 min-h-0 overflow-hidden">
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
			<Dialog.Content side="bottom" class="overflow-hidden" style="height:92%;">
				<CreateSheet onClose={closeComposer} />
			</Dialog.Content>
		</Dialog.Root>
	</div>
</div>
