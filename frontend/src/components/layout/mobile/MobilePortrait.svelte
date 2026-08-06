<script lang="ts">
	import type { Snippet } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { MapPin, Compass, Plus, X, MessageCircle, CircleUser } from '@lucide/svelte';
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
		// 平板/移动端无桌面侧栏空间放多个 nearby 菜单：聚合为「发现」页
		{ path: ROUTES.discover, icon: Compass, label: '发现' },
		{ path: ROUTES.messages, icon: MessageCircle, label: '消息' },
		{ path: ROUTES.me, icon: CircleUser, label: '我的' }
	];

	const pathname = $derived(page.url.pathname);
	const isSubPage = $derived(isSubPagePath(pathname));

	// FAB 光晕：复用 --glow-accent（每主题/暗色已定义），叠加一层中性落影增强立体感。
	const fabGlow = $derived(
		composer.composerOpen ? 'none' : '0 6px 18px var(--glow-accent), 0 2px 6px rgba(0,0,0,0.14)'
	);
</script>

<div
	class="min-h-dvh w-full flex justify-center items-start font-sans transition-colors duration-500"
	style="background-color:var(--surface-page);"
>
	<div
		class={cn('flex flex-col h-dvh w-full', 'transition-colors duration-500 overflow-hidden relative')}
		style="background-color:var(--surface-primary);color:var(--text-primary);"
	>
		<StatusBar />

		<div class="flex-1 min-h-0 overflow-hidden">
			<RoutedContentViewport>{@render children()}</RoutedContentViewport>
		</div>

		{#if !isSubPage}
			<nav
				aria-label="Main navigation"
				class="shrink-0 w-full"
				style="background:var(--surface-nav-glass);backdrop-filter:blur(24px) saturate(1.8);-webkit-backdrop-filter:blur(24px) saturate(1.8);border-top:1px solid var(--border-nav-glass);box-shadow:var(--elevation-nav);transition:background 0.4s ease, border-color 0.4s ease;"
			>
				<div
					class="flex items-center justify-between px-2"
					style="padding-top:8px;padding-bottom:max(20px, env(safe-area-inset-bottom, 20px));"
				>
					{#each NAV_ITEMS.slice(0, 2) as item}
						<NavTab
							label={item.label}
							isActive={isPrimaryRouteActive(pathname, item.path)}
							onClick={() => goto(item.path)}
							Icon={item.icon}
						/>
					{/each}

					<!-- Centre FAB -->
					<div class="flex flex-col items-center w-17" >
						<button
							type="button"
							class="hover-btn scale-btn flex items-center justify-center outline-none select-none rounded-full"
							style="width:50px;height:50px;background:{composer.composerOpen ? 'var(--text-secondary)' : 'var(--accent-default)'};box-shadow:{fabGlow};transition:background 0.22s ease, box-shadow 0.22s ease, transform 0.18s cubic-bezier(0.22,1,0.36,1), filter 0.18s ease;"
							onclick={toggleComposer}
							aria-label={composer.composerOpen ? '关闭' : '发布'}
						>
							{#if composer.composerOpen}
								<X class="size-5"  style="color:var(--text-on-accent)" strokeWidth={2.2} />
							{:else}
								<Plus class="size-5"  style="color:var(--text-on-accent)" strokeWidth={2.2} />
							{/if}
						</button>
					</div>

					{#each NAV_ITEMS.slice(2, 4) as item}
						<NavTab
							label={item.label}
							isActive={isPrimaryRouteActive(pathname, item.path)}
							onClick={() => goto(item.path)}
							Icon={item.icon}
						/>
					{/each}
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
