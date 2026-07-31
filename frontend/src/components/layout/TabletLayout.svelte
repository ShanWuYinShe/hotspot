<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { MapPin, Compass, Plus, X, MessageCircle, CircleUser } from '@lucide/svelte';
	import { themeStore } from '@/stores/theme.svelte';
	import { composer, openComposer, closeComposer, toggleComposer } from '@/stores/composer.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import CreateSheet from '@/components/layout/CreateSheet.svelte';
	import StatusBar from '@/components/layout/StatusBar.svelte';
	import { ROUTES, isPrimaryRouteActive, isSubPagePath } from '@/lib/routes';

	// 平板布局：移动式 UX——全宽内容 + 悬浮玻璃药丸底部导航 + 发布底部 Sheet。
	// 横竖屏同布局，无居中/最大宽度约束。迁移自旧版 TabletLayout。

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

	const navShadow = $derived.by(() => {
		if (themeStore.isDark) return '0 -4px 24px rgba(0,0,0,0.40), 0 8px 32px rgba(0,0,0,0.28)';
		if (themeStore.theme === 'forest') return '0 -2px 16px rgba(61,122,85,0.08), 0 8px 24px rgba(61,122,85,0.10)';
		return '0 -2px 16px rgba(0,0,0,0.04), 0 8px 24px rgba(99,102,241,0.10)';
	});

</script>

<div
	class="flex flex-col h-[100dvh] w-full overflow-hidden transition-colors duration-500 font-sans"
	style="background-color:var(--neu-bg);color:var(--neu-text);"
>
	<StatusBar />

	<!-- Scrollable area: full width, individual elements keep their own proportions -->
	<main class="flex-1 overflow-y-auto overflow-x-hidden no-scrollbar w-full" style="background-color:var(--neu-bg);">
		<div class="tablet-page-canvas">
			{@render children()}
		</div>
	</main>

	<!-- Floating glass pill bottom nav -->
	{#if !isSubPage}
		<div
			class="flex-shrink-0 w-full flex justify-center"
			style="padding:8px 24px;padding-bottom:max(env(safe-area-inset-bottom, 16px), 16px);background-color:transparent;"
		>
			<div
				class="flex items-center"
				style="width:100%;max-width:520px;border-radius:40px;background:color-mix(in srgb, var(--neu-bg) 92%, transparent);backdrop-filter:blur(28px) saturate(1.8);-webkit-backdrop-filter:blur(28px) saturate(1.8);box-shadow:{navShadow};border:1px solid var(--neu-border);padding:8px 16px;gap:6px;"
			>
				{#each NAV_ITEMS.slice(0, 2) as item}
					<button
						type="button"
						class="neu-hover-btn relative flex flex-col items-center justify-center outline-none select-none"
						style="flex:1;min-width:0;padding-top:9px;padding-bottom:9px;gap:5px;"
						aria-label={item.label}
						aria-current={isPrimaryRouteActive(pathname, item.path) ? 'page' : undefined}
						onclick={() => void goto(item.path)}
					>
						{#if isPrimaryRouteActive(pathname, item.path)}
							<div style="position:absolute;inset:4px 6px;border-radius:20px;background:var(--neu-accent-primary);opacity:0.14;"></div>
						{/if}
						<div style="display:flex;align-items:center;justify-content:center;position:relative;z-index:1;">
							<item.icon
								style="width:24px;height:24px;color:{isPrimaryRouteActive(pathname, item.path) ? 'var(--neu-accent-primary)' : 'var(--neu-text-muted)'};transition:color 0.22s ease;"
								strokeWidth={isPrimaryRouteActive(pathname, item.path) ? 2.1 : 1.5}
							/>
						</div>
						<span
							style="font-size:11px;line-height:1;color:{isPrimaryRouteActive(pathname, item.path) ? 'var(--neu-accent-primary)' : 'var(--neu-text-muted)'};font-weight:{isPrimaryRouteActive(pathname, item.path) ? 600 : 400};transition:color 0.22s ease;letter-spacing:0.02em;position:relative;z-index:1;"
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
						style="width:48px;height:48px;background:{composer.composerOpen ? 'var(--neu-text-muted)' : 'var(--neu-accent-primary)'};box-shadow:{composer.composerOpen ? 'none' : '0 6px 20px var(--neu-glow)'};transition:background 0.22s, box-shadow 0.22s, transform 0.18s cubic-bezier(0.22,1,0.36,1), filter 0.18s ease;"
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
						class="neu-hover-btn relative flex flex-col items-center justify-center outline-none select-none"
						style="flex:1;min-width:0;padding-top:9px;padding-bottom:9px;gap:5px;"
						aria-label={item.label}
						aria-current={isPrimaryRouteActive(pathname, item.path) ? 'page' : undefined}
						onclick={() => void goto(item.path)}
					>
						{#if isPrimaryRouteActive(pathname, item.path)}
							<div style="position:absolute;inset:4px 6px;border-radius:20px;background:var(--neu-accent-primary);opacity:0.14;"></div>
						{/if}
						<div style="display:flex;align-items:center;justify-content:center;position:relative;z-index:1;">
							<item.icon
								style="width:24px;height:24px;color:{isPrimaryRouteActive(pathname, item.path) ? 'var(--neu-accent-primary)' : 'var(--neu-text-muted)'};transition:color 0.22s ease;"
								strokeWidth={isPrimaryRouteActive(pathname, item.path) ? 2.1 : 1.5}
							/>
						</div>
						<span
							style="font-size:11px;line-height:1;color:{isPrimaryRouteActive(pathname, item.path) ? 'var(--neu-accent-primary)' : 'var(--neu-text-muted)'};font-weight:{isPrimaryRouteActive(pathname, item.path) ? 600 : 400};transition:color 0.22s ease;letter-spacing:0.02em;position:relative;z-index:1;"
						>
							{item.label}
						</span>
					</button>
				{/each}
			</div>
		</div>
	{/if}

	{#if showCreate}
		<Dialog.Root open={showCreate} onOpenChange={(next) => (next ? openComposer() : closeComposer())}>
			<Dialog.Content side="bottom" class="overflow-hidden" style="height:90%;">
				<CreateSheet onClose={closeComposer} />
			</Dialog.Content>
		</Dialog.Root>
	{/if}
</div>
