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

	// FAB 光晕：统一走 --elevation-fab（每主题/暗色已定义）
	const fabGlow = $derived(composer.composerOpen ? 'none' : 'var(--cst-elevation-fab)');
</script>

<div
	class="min-h-dvh w-full flex justify-center items-start font-sans transition-colors duration-gentle"
	style="background-color:var(--cst-surface-page);"
>
	<div
		class={cn('flex flex-col h-dvh w-full', 'transition-colors duration-gentle overflow-hidden relative')}
		style="background-color:var(--cst-surface-primary);color:var(--cst-text-primary);"
	>
		<StatusBar />

		<div class="flex-1 min-h-0 overflow-hidden">
			<RoutedContentViewport>{@render children()}</RoutedContentViewport>
		</div>

		{#if !isSubPage}
			<nav
				aria-label="主导航"
				class="shrink-0 w-full"
				style="background:var(--cst-surface-nav-glass);backdrop-filter:blur(var(--cst-blur-lg)) saturate(1.8);-webkit-backdrop-filter:blur(var(--cst-blur-lg)) saturate(1.8);border-top:var(--cst-border-width-hairline) solid var(--cst-border-nav-glass);box-shadow:var(--cst-elevation-nav);transition:background var(--cst-motion-slow) var(--cst-ease-standard), border-color var(--cst-motion-slow) var(--cst-ease-standard);"
			>
				<div
					class="flex items-center px-2"
					style="padding-top:var(--cst-bottom-nav-pad-top);padding-bottom:max(var(--cst-bottom-nav-pad-bottom), env(safe-area-inset-bottom, var(--cst-bottom-nav-pad-bottom)));"
				>
					<!-- 左侧：首页 + 附近（flex-1 与右侧对称，保持 FAB 居中） -->
					<div class="flex flex-1 justify-center gap-4">
						{#each NAV_ITEMS.slice(0, 2) as item}
							<NavTab
								label={item.label}
								isActive={isPrimaryRouteActive(pathname, item.path)}
								onClick={() => goto(item.path)}
								Icon={item.icon}
							/>
						{/each}
					</div>

					<!-- Centre FAB -->
					<div class="flex flex-col items-center w-17" >
						<button
							type="button"
							class="hover-btn scale-btn flex items-center justify-center focus-ring select-none rounded-full size-12.5"
							style="background:{composer.composerOpen ? 'var(--cst-text-secondary)' : 'var(--cst-accent-default)'};box-shadow:{fabGlow};transition:background var(--cst-motion-standard) var(--cst-ease-standard), box-shadow var(--cst-motion-standard) var(--cst-ease-standard), transform var(--cst-motion-snappy) var(--cst-ease-standard), filter var(--cst-motion-snappy) var(--cst-ease-standard);"
							onclick={toggleComposer}
							aria-label={composer.composerOpen ? '关闭' : '发布'}
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
								isActive={isPrimaryRouteActive(pathname, item.path)}
								onClick={() => goto(item.path)}
								Icon={item.icon}
							/>
						{/each}
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
