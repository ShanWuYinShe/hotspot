<script lang="ts">
	import type { Snippet } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { MapPin, Compass, Plus, X, MessageCircle, CircleUser } from '@lucide/svelte';
	import { cn } from '@/lib/utils';
	import { themeStore } from '@/stores/theme.svelte';
	import { platformStore } from '@/stores/platform.svelte';
	import { composer, openComposer, closeComposer, toggleComposer } from '@/stores/composer.svelte';
	import CreateSheet from '@/components/layout/CreateSheet.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import StatusBar from '@/components/layout/StatusBar.svelte';
	import DesktopLayout from '@/components/layout/desktop/DesktopLayout.svelte';
	import TabletLayout from '@/components/layout/tablet/TabletLayout.svelte';
	import NavTab from '@/components/layout/NavTab.svelte';
	import MobileLandscape from '@/components/layout/MobileLandscape.svelte';
	import { ROUTES, isPrimaryRouteActive, isSubPagePath } from '@/lib/routes';

	// 迁移自旧版 MainLayout：桌面 → DesktopLayout，平板 → TabletLayout，
	// 移动竖屏 → 底部 4 Tab + 中央 FAB + 发布 Sheet；移动横屏 → 左侧 Rail。

	let { children }: { children: Snippet } = $props();

	const NAV_ITEMS: { path: string; icon: typeof MapPin; label: string }[] = [
		{ path: ROUTES.home, icon: MapPin, label: '附近' },
		{ path: ROUTES.nearbyMoments, icon: Compass, label: '发现' },
		{ path: ROUTES.messages, icon: MessageCircle, label: '消息' },
		{ path: ROUTES.me, icon: CircleUser, label: '我的' }
	];

	const pathname = $derived(page.url.pathname);
	const isMobileLandscape = $derived(platformStore.orientation === 'landscape');
	const isSubPage = $derived(isSubPagePath(pathname));
	const showCreate = $derived(composer.composerOpen);

	const fabGlow = $derived.by(() => {
		if (composer.composerOpen) return 'none';
		const isDark = themeStore.isDark;
		if (themeStore.theme === 'forest') {
			return isDark
				? '0 6px 22px rgba(82,184,120,0.38), 0 2px 6px rgba(0,0,0,0.20)'
				: '0 6px 18px rgba(61,122,85,0.32), 0 2px 4px rgba(0,0,0,0.10)';
		}
		return isDark
			? '0 6px 22px rgba(129,140,248,0.42), 0 2px 6px rgba(0,0,0,0.20)'
			: '0 6px 18px rgba(99,102,241,0.38), 0 2px 4px rgba(0,0,0,0.10)';
	});

</script>

{#if platformStore.platform === 'desktop'}
	<DesktopLayout>{@render children()}</DesktopLayout>
{:else if platformStore.platform === 'tablet'}
	<TabletLayout>{@render children()}</TabletLayout>
{:else if isMobileLandscape}
	<MobileLandscape>{@render children()}</MobileLandscape>
{:else}
	<div
		class="min-h-[100dvh] w-full flex justify-center items-start font-sans transition-colors duration-500"
		style="background-color:var(--page-bg);"
	>
		<div
			class={cn('flex flex-col h-[100dvh] w-full', 'transition-colors duration-500 overflow-hidden relative')}
			style="background-color:var(--neu-bg);color:var(--neu-text);"
		>
			<StatusBar />

			<main class="flex-1 overflow-y-auto overflow-x-hidden no-scrollbar scroll-smooth">
				{@render children()}
			</main>

			{#if !isSubPage}
				<nav
					aria-label="Main navigation"
					class="flex-shrink-0 w-full"
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
						<div class="flex flex-col items-center" style="width:68px;">
							<button
								type="button"
								class="neu-hover-btn neu-scale-btn flex items-center justify-center outline-none select-none rounded-full"
								style="width:50px;height:50px;background:{composer.composerOpen ? 'var(--neu-text-muted)' : 'var(--neu-accent-primary)'};box-shadow:{fabGlow};transition:background 0.22s ease, box-shadow 0.22s ease, transform 0.18s cubic-bezier(0.22,1,0.36,1), filter 0.18s ease;"
								onclick={toggleComposer}
								aria-label={composer.composerOpen ? '关闭' : '发布'}
							>
								{#if composer.composerOpen}
									<X style="width:20px;height:20px;color:white;" strokeWidth={2.2} />
								{:else}
									<Plus style="width:20px;height:20px;color:white;" strokeWidth={2.2} />
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
{/if}
