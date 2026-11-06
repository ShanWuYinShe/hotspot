<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { Home, Plus, X, MessageCircle, CircleUser, Compass } from '@lucide/svelte';
	import { composer, openComposer, closeComposer, toggleComposer } from '@/stores/composer.svelte';
	import { unread } from '@/stores/unread.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import CreateSheet from '@/components/overlays/CreateSheet.svelte';
	import NavTab from '@/components/layout/mobile/NavTab.svelte';
	import RoutedContentViewport from '@/components/layout/page/RoutedContentViewport.svelte';
	import AmbientBackdrop from '@/components/layout/ambient/AmbientBackdrop.svelte';
	import { ROUTES, isPrimaryRouteActive, isSubPagePath } from '@/lib/routes';

	// 移动横屏：对齐移动端/平板端设计约束——「导航栏永远在底部」。
	// 5 slot 居中自适应，大屏横屏收敛在中央大拇指触控区（max-w-lg），不散落两侧，杜绝出现左侧侧边栏。

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
</script>

<div
	class="bg-surface-page text-text-primary flex flex-col h-dvh w-full overflow-hidden font-sans transition-colors duration-gentle"
>
	<div class="relative isolate flex flex-1 min-h-0 min-w-0 flex-col overflow-hidden">
		<!-- 氛围画布固定在外壳内容区（不随路由滚动） -->
		<AmbientBackdrop />
		<RoutedContentViewport contentClassName={!isSubPage ? 'pb-24' : ''}>
			{@render children()}
		</RoutedContentViewport>
	</div>

	<!-- 移动横屏悬浮孤岛导航栏 -->
	{#if !isSubPage}
		<nav
			aria-label="主导航"
			class="fixed bottom-0 inset-x-0 z-[var(--cst-z-fixed)] pointer-events-none flex justify-center px-4 pb-[max(10px,env(safe-area-inset-bottom,10px))]"
		>
			<div
				class="pointer-events-auto flex items-center justify-between w-full max-w-lg rounded-[28px] border border-border-default/70 bg-surface-island/92 p-1.5 shadow-[0_12px_36px_rgba(0,0,0,0.35)] backdrop-blur-2xl ring-1 ring-white/10 dark:ring-white/5"
			>
				<!-- Slot 1: 首页 -->
				<NavTab
					label={NAV_ITEMS[0].label}
					isActive={isPrimaryRouteActive(pathname, NAV_ITEMS[0].path)}
					onClick={() => goto(NAV_ITEMS[0].path)}
					Icon={NAV_ITEMS[0].icon}
					iconSize={24}
				/>

				<!-- Slot 2: 附近 -->
				<NavTab
					label={NAV_ITEMS[1].label}
					isActive={isPrimaryRouteActive(pathname, NAV_ITEMS[1].path)}
					onClick={() => goto(NAV_ITEMS[1].path)}
					Icon={NAV_ITEMS[1].icon}
					iconSize={24}
				/>

				<!-- Slot 3: Centre FAB (发布) -->
				<div class="flex items-center justify-center shrink-0 px-1">
					<button
						type="button"
						class="hover-btn scale-btn flex items-center justify-center rounded-full focus-ring select-none size-11 shadow-[0_4px_16px_rgba(var(--cst-palette-accent-primary),0.35)] active:scale-95 transition-all duration-rapid"
						style="background:{composer.composerOpen ? 'var(--cst-text-secondary)' : 'var(--cst-accent-default)'};"
						onclick={toggleComposer}
						aria-label={composer.composerOpen ? '关闭' : '发布'}
					>
						{#if composer.composerOpen}
							<X class="size-4.5 text-text-on-accent" strokeWidth={2.25} />
						{:else}
							<Plus class="size-5 text-text-on-accent" strokeWidth={2.25} />
						{/if}
					</button>
				</div>

				<!-- Slot 4: 消息 -->
				<NavTab
					label={NAV_ITEMS[2].label}
					isActive={isPrimaryRouteActive(pathname, NAV_ITEMS[2].path)}
					onClick={() => goto(NAV_ITEMS[2].path)}
					Icon={NAV_ITEMS[2].icon}
					iconSize={24}
					count={unread.chat}
				/>

				<!-- Slot 5: 我的 -->
				<NavTab
					label={NAV_ITEMS[3].label}
					isActive={isPrimaryRouteActive(pathname, NAV_ITEMS[3].path)}
					onClick={() => goto(NAV_ITEMS[3].path)}
					Icon={NAV_ITEMS[3].icon}
					iconSize={24}
				/>
			</div>
		</nav>
	{/if}

	<Dialog.Root open={showCreate} onOpenChange={(next) => (next ? openComposer() : closeComposer())}>
		<Dialog.Content
			side="bottom"
			class="overflow-hidden"
			style="height:var(--cst-sheet-height-2xl);"
		>
			<CreateSheet onClose={closeComposer} />
		</Dialog.Content>
	</Dialog.Root>
</div>
