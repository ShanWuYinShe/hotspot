<script lang="ts">
	import type { Snippet } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { MapPin, Plus, X, MessageCircle, CircleUser } from '@lucide/svelte';
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
		{ path: ROUTES.home, icon: MapPin, label: '附近' },
		{ path: ROUTES.messages, icon: MessageCircle, label: '消息' },
		{ path: ROUTES.me, icon: CircleUser, label: '我的' }
	];

	const pathname = $derived(page.url.pathname);
	const isSubPage = $derived(isSubPagePath(pathname));

	// FAB 光晕：统一走 --elevation-fab（每主题/暗色已定义）
	const fabGlow = $derived(composer.composerOpen ? 'none' : 'var(--cst-elevation-fab)');
</script>

<div
	class="min-h-dvh w-full flex justify-center items-start font-sans transition-colors duration-500"
	style="background-color:var(--cst-surface-page);"
>
	<div
		class={cn('flex flex-col h-dvh w-full', 'transition-colors duration-500 overflow-hidden relative')}
		style="background-color:var(--cst-surface-primary);color:var(--cst-text-primary);"
	>
		<StatusBar />

		<div class="flex-1 min-h-0 overflow-hidden">
			<RoutedContentViewport>{@render children()}</RoutedContentViewport>
		</div>

		{#if !isSubPage}
			<nav
				aria-label="Main navigation"
				class="shrink-0 w-full"
				style="background:var(--cst-surface-nav-glass);backdrop-filter:blur(var(--cst-blur-lg)) saturate(1.8);-webkit-backdrop-filter:blur(var(--cst-blur-lg)) saturate(1.8);border-top:1px solid var(--cst-border-nav-glass);box-shadow:var(--cst-elevation-nav);transition:background 0.4s ease, border-color 0.4s ease;"
			>
				<div
					class="flex items-center px-2"
					style="padding-top:8px;padding-bottom:max(20px, env(safe-area-inset-bottom, 20px));"
				>
					<!-- 左侧：附近（flex-1 与右侧对称，保持 FAB 居中） -->
					<div class="flex flex-1 justify-center">
						{#each NAV_ITEMS.slice(0, 1) as item}
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
							class="hover-btn scale-btn flex items-center justify-center outline-none select-none rounded-full"
							style="width:50px;height:50px;background:{composer.composerOpen ? 'var(--cst-text-secondary)' : 'var(--cst-accent-default)'};box-shadow:{fabGlow};transition:background 0.22s ease, box-shadow 0.22s ease, transform 0.18s var(--cst-ease-standard), filter 0.18s ease;"
							onclick={toggleComposer}
							aria-label={composer.composerOpen ? '关闭' : '发布'}
						>
							{#if composer.composerOpen}
								<X class="size-5"  style="color:var(--cst-text-on-accent)" strokeWidth={2.2} />
							{:else}
								<Plus class="size-5"  style="color:var(--cst-text-on-accent)" strokeWidth={2.2} />
							{/if}
						</button>
					</div>

					<!-- 右侧：消息 + 我的（flex-1 与左侧对称） -->
					<div class="flex flex-1 justify-center gap-4">
						{#each NAV_ITEMS.slice(1, 3) as item}
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
