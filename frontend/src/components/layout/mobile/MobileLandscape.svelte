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
		<RoutedContentViewport>
			{@render children()}
		</RoutedContentViewport>
	</div>

	<!-- 移动端底栏导航：永远位于视口底部，中央 FAB 发布按钮 -->
	{#if !isSubPage}
		<nav
			aria-label="主导航"
			class="shrink-0 w-full flex justify-center nav-glass nav-glass--bottom border-t border-border-default/40"
			style="box-shadow:var(--cst-elevation-nav);"
		>
			<div
				class="grid grid-cols-5 items-center w-full max-w-lg px-2"
				style="padding-top:var(--cst-bottom-nav-pad-top);padding-bottom:max(var(--cst-bottom-nav-pad-bottom), env(safe-area-inset-bottom, var(--cst-bottom-nav-pad-bottom)));"
			>
				<!-- Slot 1: 首页 -->
				<div class="flex items-center justify-center w-full">
					<NavTab
						label={NAV_ITEMS[0].label}
						isActive={isPrimaryRouteActive(pathname, NAV_ITEMS[0].path)}
						onClick={() => goto(NAV_ITEMS[0].path)}
						Icon={NAV_ITEMS[0].icon}
					/>
				</div>

				<!-- Slot 2: 附近 -->
				<div class="flex items-center justify-center w-full">
					<NavTab
						label={NAV_ITEMS[1].label}
						isActive={isPrimaryRouteActive(pathname, NAV_ITEMS[1].path)}
						onClick={() => goto(NAV_ITEMS[1].path)}
						Icon={NAV_ITEMS[1].icon}
					/>
				</div>

				<!-- Slot 3: Centre FAB -->
				<div class="flex items-center justify-center w-full">
					<button
						type="button"
						class="hover-btn scale-btn flex items-center justify-center rounded-full focus-ring select-none size-11 {composer.composerOpen ? '' : 'shadow-glow-md'}"
						style="background:{composer.composerOpen ? 'var(--cst-text-secondary)' : 'var(--cst-accent-default)'};"
						onclick={toggleComposer}
						aria-label={composer.composerOpen ? '关闭' : '发布'}
					>
						{#if composer.composerOpen}
							<X class="size-4.5 text-text-on-accent" strokeWidth={2} />
						{:else}
							<Plus class="size-4.5 text-text-on-accent" strokeWidth={2} />
						{/if}
					</button>
				</div>

				<!-- Slot 4: 消息 -->
				<div class="flex items-center justify-center w-full">
					<NavTab
						label={NAV_ITEMS[2].label}
						isActive={isPrimaryRouteActive(pathname, NAV_ITEMS[2].path)}
						onClick={() => goto(NAV_ITEMS[2].path)}
						Icon={NAV_ITEMS[2].icon}
						count={unread.chat}
					/>
				</div>

				<!-- Slot 5: 我的 -->
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
